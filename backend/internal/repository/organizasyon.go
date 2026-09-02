package repository

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	appmongo "github.com/oz-fatma/kontrata/backend/internal/mongo"
)

const organizasyonCollection = "organizasyonlar"

const (
	OrgDurumAktif  = "AKTIF"
	OrgDurumAskida = "ASKIDA"
)

// Organizasyon kurumsal hesabın ev sahibi belgesidir.
type Organizasyon struct {
	ID               bson.ObjectID `bson:"_id,omitempty"`
	Ad               string        `bson:"ad"`
	VergiNo          string        `bson:"vergiNo,omitempty"`
	SahipKullaniciID bson.ObjectID `bson:"sahipKullaniciId"`
	Durum            string        `bson:"durum"`
	OlusturmaTarihi  time.Time     `bson:"olusturmaTarihi"`
}

// OrganizasyonRepository organizasyonlar koleksiyonuna erişir.
type OrganizasyonRepository struct {
	col *mongo.Collection
}

func NewOrganizasyonRepository(client *appmongo.Client) *OrganizasyonRepository {
	if client == nil {
		return &OrganizasyonRepository{}
	}
	return &OrganizasyonRepository{col: client.Collection(appmongo.DatabaseName(), organizasyonCollection)}
}

func (r *OrganizasyonRepository) ready() bool {
	return r != nil && r.col != nil
}

func (r *OrganizasyonRepository) EnsureIndexes(ctx context.Context) error {
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

func (r *OrganizasyonRepository) Create(ctx context.Context, doc *Organizasyon) error {
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

func (r *OrganizasyonRepository) GetByID(ctx context.Context, id bson.ObjectID) (*Organizasyon, error) {
	if !r.ready() {
		return nil, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	var doc Organizasyon
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, ErrStore
	}
	return &doc, nil
}

func (r *OrganizasyonRepository) SetSahip(ctx context.Context, id, sahip bson.ObjectID) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	res, err := r.col.UpdateByID(ctx, id, bson.M{"$set": bson.M{"sahipKullaniciId": sahip}})
	if err != nil {
		return ErrStore
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *OrganizasyonRepository) Delete(ctx context.Context, id bson.ObjectID) error {
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
