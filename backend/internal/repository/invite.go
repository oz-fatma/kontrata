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

const inviteCollection = "davetler"

// Invite organizasyona e-posta ile gönderilen üyelik davetidir. Token yalnızca hash olarak durur.
type Invite struct {
	ID             bson.ObjectID `bson:"_id,omitempty"`
	OrganizationID bson.ObjectID `bson:"organizasyonId"`
	Email          string        `bson:"eposta"`
	Role           string        `bson:"rol"`
	TokenHash      string        `bson:"tokenHash"`
	ExpiresAt      time.Time     `bson:"sonKullanma"`
	Used           bool          `bson:"kullanildi"`
	InvitedBy      bson.ObjectID `bson:"davetEdenKullaniciId"`
}

// InviteRepository davetler koleksiyonuna erişir.
type InviteRepository struct {
	col *mongo.Collection
}

func NewInviteRepository(client *appmongo.Client) *InviteRepository {
	if client == nil {
		return &InviteRepository{}
	}
	return &InviteRepository{col: client.Collection(appmongo.DatabaseName(), inviteCollection)}
}

func (r *InviteRepository) ready() bool {
	return r != nil && r.col != nil
}

func (r *InviteRepository) EnsureIndexes(ctx context.Context) error {
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

func (r *InviteRepository) Create(ctx context.Context, doc *Invite) error {
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

func (r *InviteRepository) GetByHash(ctx context.Context, hash string, now time.Time) (*Invite, error) {
	if !r.ready() {
		return nil, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	var doc Invite
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

func (r *InviteRepository) Consume(ctx context.Context, hash string, now time.Time) (*Invite, error) {
	if !r.ready() {
		return nil, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	var doc Invite
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

func (r *InviteRepository) InvalidateUnused(ctx context.Context, orgID bson.ObjectID, email string) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	_, err := r.col.UpdateMany(ctx, bson.M{
		"organizasyonId": orgID,
		"eposta":         email,
		"kullanildi":     false,
	}, bson.M{"$set": bson.M{"kullanildi": true}})
	if err != nil {
		return ErrStore
	}
	return nil
}

func (r *InviteRepository) DeleteByOrg(ctx context.Context, orgID bson.ObjectID) error {
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

func (r *InviteRepository) CountByOrg(ctx context.Context, orgID bson.ObjectID) (int64, error) {
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
