package repository

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	appmongo "github.com/oz-fatma/kontrata/backend/internal/mongo"
)

const promptVersionCollection = "prompt_surumleri"

const (
	PromptKindReader  = "OKUYUCU"
	PromptKindAuditor = "DENETCI"
)

// PromptVersion organizasyona ait bir Okuyucu veya Denetçi prompt sürümüdür.
type PromptVersion struct {
	ID        bson.ObjectID `bson:"_id,omitempty"`
	OrgID     bson.ObjectID `bson:"organizasyonId"`
	Kind      string        `bson:"tip"`
	Content   string        `bson:"icerik"`
	Version   int32         `bson:"surum"`
	CreatedBy bson.ObjectID `bson:"olusturanKullaniciId"`
	CreatedAt time.Time     `bson:"olusturmaTarihi"`
	Active    bool          `bson:"aktif"`
}

// PromptVersionRepository prompt_surumleri koleksiyonuna erişir.
type PromptVersionRepository struct {
	col *mongo.Collection
}

func NewPromptVersionRepository(client *appmongo.Client) *PromptVersionRepository {
	if client == nil {
		return &PromptVersionRepository{}
	}
	return &PromptVersionRepository{col: client.Collection(appmongo.DatabaseName(), promptVersionCollection)}
}

func (r *PromptVersionRepository) ready() bool {
	return r != nil && r.col != nil
}

func (r *PromptVersionRepository) EnsureIndexes(ctx context.Context) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	_, err := r.col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "organizasyonId", Value: 1}, {Key: "tip", Value: 1}, {Key: "surum", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "organizasyonId", Value: 1}, {Key: "tip", Value: 1}},
			Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.M{"aktif": true}),
		},
	})
	if err != nil {
		return ErrStore
	}
	return nil
}

func (r *PromptVersionRepository) Insert(ctx context.Context, doc *PromptVersion) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	if doc.ID.IsZero() {
		doc.ID = bson.NewObjectID()
	}
	res, err := r.col.InsertOne(ctx, doc)
	if err != nil {
		return ErrStore
	}
	if oid, ok := res.InsertedID.(bson.ObjectID); ok {
		doc.ID = oid
	}
	return nil
}

func (r *PromptVersionRepository) GetByID(ctx context.Context, id bson.ObjectID) (*PromptVersion, error) {
	if !r.ready() {
		return nil, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	var doc PromptVersion
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, ErrStore
	}
	return &doc, nil
}

func (r *PromptVersionRepository) ListByKind(ctx context.Context, orgID bson.ObjectID, kind string) ([]PromptVersion, error) {
	if !r.ready() {
		return nil, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	cur, err := r.col.Find(ctx, bson.M{"organizasyonId": orgID, "tip": kind}, options.Find().SetSort(bson.D{{Key: "surum", Value: -1}}))
	if err != nil {
		return nil, ErrStore
	}
	defer func() { _ = cur.Close(ctx) }()
	var out []PromptVersion
	if err := cur.All(ctx, &out); err != nil {
		return nil, ErrStore
	}
	if out == nil {
		out = []PromptVersion{}
	}
	return out, nil
}

func (r *PromptVersionRepository) GetActive(ctx context.Context, orgID bson.ObjectID, kind string) (*PromptVersion, error) {
	if !r.ready() {
		return nil, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	var doc PromptVersion
	err := r.col.FindOne(ctx, bson.M{"organizasyonId": orgID, "tip": kind, "aktif": true}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, ErrStore
	}
	return &doc, nil
}

func (r *PromptVersionRepository) MaxVersion(ctx context.Context, orgID bson.ObjectID, kind string) (int32, error) {
	if !r.ready() {
		return 0, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	var doc PromptVersion
	err := r.col.FindOne(ctx, bson.M{"organizasyonId": orgID, "tip": kind}, options.FindOne().SetSort(bson.D{{Key: "surum", Value: -1}})).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return 0, nil
	}
	if err != nil {
		return 0, ErrStore
	}
	return doc.Version, nil
}

func (r *PromptVersionRepository) DeactivateKind(ctx context.Context, orgID bson.ObjectID, kind string) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	_, err := r.col.UpdateMany(ctx, bson.M{"organizasyonId": orgID, "tip": kind, "aktif": true}, bson.M{"$set": bson.M{"aktif": false}})
	if err != nil {
		return ErrStore
	}
	return nil
}

func (r *PromptVersionRepository) Activate(ctx context.Context, id bson.ObjectID) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	res, err := r.col.UpdateByID(ctx, id, bson.M{"$set": bson.M{"aktif": true}})
	if err != nil {
		return ErrStore
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PromptVersionRepository) DeleteByOrg(ctx context.Context, orgID bson.ObjectID) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	_, err := r.col.DeleteMany(ctx, bson.M{"organizasyonId": orgID})
	if err != nil {
		return ErrStore
	}
	return nil
}

func (r *PromptVersionRepository) CountByOrg(ctx context.Context, orgID bson.ObjectID) (int64, error) {
	if !r.ready() {
		return 0, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	n, err := r.col.CountDocuments(ctx, bson.M{"organizasyonId": orgID})
	if err != nil {
		return 0, ErrStore
	}
	return n, nil
}
