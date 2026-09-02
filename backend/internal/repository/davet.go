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

const davetCollection = "davetler"

// Davet organizasyona e-posta ile gönderilen üyelik davetidir. Token yalnızca hash olarak durur.
type Davet struct {
	ID                 bson.ObjectID `bson:"_id,omitempty"`
	OrganizasyonID     bson.ObjectID `bson:"organizasyonId"`
	Eposta             string        `bson:"eposta"`
	Rol                string        `bson:"rol"`
	TokenHash          string        `bson:"tokenHash"`
	SonKullanma        time.Time     `bson:"sonKullanma"`
	Kullanildi         bool          `bson:"kullanildi"`
	DavetEdenKullanici bson.ObjectID `bson:"davetEdenKullaniciId"`
}

// DavetRepository davetler koleksiyonuna erişir.
type DavetRepository struct {
	col *mongo.Collection
}

func NewDavetRepository(client *appmongo.Client) *DavetRepository {
	if client == nil {
		return &DavetRepository{}
	}
	return &DavetRepository{col: client.Collection(appmongo.DatabaseName(), davetCollection)}
}

func (r *DavetRepository) ready() bool {
	return r != nil && r.col != nil
}

func (r *DavetRepository) EnsureIndexes(ctx context.Context) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	_, err := r.col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "tokenHash", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "sonKullanma", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(0),
		},
		{Keys: bson.D{{Key: "organizasyonId", Value: 1}, {Key: "eposta", Value: 1}}},
	})
	if err != nil {
		return ErrStore
	}
	return nil
}

func (r *DavetRepository) Create(ctx context.Context, doc *Davet) error {
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
		if mongo.IsDuplicateKeyError(err) {
			return ErrDuplicate
		}
		return ErrStore
	}
	if oid, ok := res.InsertedID.(bson.ObjectID); ok {
		doc.ID = oid
	}
	return nil
}

func (r *DavetRepository) GetByHash(ctx context.Context, hash string, now time.Time) (*Davet, error) {
	if !r.ready() {
		return nil, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	var doc Davet
	err := r.col.FindOne(ctx, bson.M{
		"tokenHash":   hash,
		"kullanildi":  false,
		"sonKullanma": bson.M{"$gt": now},
	}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, ErrStore
	}
	return &doc, nil
}

func (r *DavetRepository) Consume(ctx context.Context, hash string, now time.Time) (*Davet, error) {
	if !r.ready() {
		return nil, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	var doc Davet
	err := r.col.FindOneAndUpdate(ctx, bson.M{
		"tokenHash":   hash,
		"kullanildi":  false,
		"sonKullanma": bson.M{"$gt": now},
	}, bson.M{"$set": bson.M{"kullanildi": true}}, options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, ErrStore
	}
	return &doc, nil
}

func (r *DavetRepository) InvalidateUnused(ctx context.Context, orgID bson.ObjectID, eposta string) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	_, err := r.col.UpdateMany(ctx, bson.M{
		"organizasyonId": orgID,
		"eposta":         eposta,
		"kullanildi":     false,
	}, bson.M{"$set": bson.M{"kullanildi": true}})
	if err != nil {
		return ErrStore
	}
	return nil
}

func (r *DavetRepository) DeleteByOrg(ctx context.Context, orgID bson.ObjectID) error {
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
