package repository

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	appmongo "github.com/oz-fatma/kontrata/backend/internal/mongo"
)

const organizationCollection = "organizasyonlar"

const (
	OrgStatusActive    = "AKTIF"
	OrgStatusSuspended = "ASKIDA"
)

// Organization kurumsal hesabın ev sahibi belgesidir.
type Organization struct {
	ID          bson.ObjectID `bson:"_id,omitempty"`
	Name        string        `bson:"ad"`
	TaxID       string        `bson:"vergiNo,omitempty"`
	OwnerUserID bson.ObjectID `bson:"sahipKullaniciId"`
	Status      string        `bson:"durum"`
	CreatedAt   time.Time     `bson:"olusturmaTarihi"`
}

// OrganizationRepository organizasyonlar koleksiyonuna erişir.
type OrganizationRepository struct {
	col *mongo.Collection
}

func NewOrganizationRepository(client *appmongo.Client) *OrganizationRepository {
	if client == nil {
		return &OrganizationRepository{}
	}
	return &OrganizationRepository{col: client.Collection(appmongo.DatabaseName(), organizationCollection)}
}

func (r *OrganizationRepository) ready() bool {
	return r != nil && r.col != nil
}

func (r *OrganizationRepository) EnsureIndexes(ctx context.Context) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	_, err := r.col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "sahipKullaniciId", Value: 1}}},
		{Keys: bson.D{{Key: "durum", Value: 1}}},
	})
	if err != nil {
		return ErrStore
	}
	return nil
}

func (r *OrganizationRepository) Create(ctx context.Context, doc *Organization) error {
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

func (r *OrganizationRepository) GetByID(ctx context.Context, id bson.ObjectID) (*Organization, error) {
	if !r.ready() {
		return nil, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	var doc Organization
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, ErrStore
	}
	return &doc, nil
}

func (r *OrganizationRepository) SetOwner(ctx context.Context, id, owner bson.ObjectID) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	res, err := r.col.UpdateByID(ctx, id, bson.M{"$set": bson.M{"sahipKullaniciId": owner}})
	if err != nil {
		return ErrStore
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *OrganizationRepository) Delete(ctx context.Context, id bson.ObjectID) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	res, err := r.col.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return ErrStore
	}
	if res.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}
