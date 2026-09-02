package repository

import (
	"context"
	"errors"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	appmongo "github.com/oz-fatma/kontrata/backend/internal/mongo"
)

const oturumCollection = "oturumlar"

const (
	IptalCikis          = "cikis"
	IptalYenileme       = "yenileme"
	IptalSifreSifirlama = "sifre_sifirlama"
)

// Oturum yenileme jetonu hash'i ile saklanan oturum belgesidir.
type Oturum struct {
	ID                bson.ObjectID `bson:"_id,omitempty"`
	KullaniciID       bson.ObjectID `bson:"kullaniciId"`
	YenilemeTokenHash string        `bson:"yenilemeTokenHash"`
	OlusturmaTarihi   time.Time     `bson:"olusturmaTarihi"`
	SonKullanma       time.Time     `bson:"sonKullanma"`
	IptalEdildi       bool          `bson:"iptalEdildi"`
	IptalNedeni       string        `bson:"iptalNedeni,omitempty"`
	IPAdresi          string        `bson:"ipAdresi"`
	KullaniciAjani    string        `bson:"kullaniciAjani"`
}

// OturumRepository oturumlar koleksiyonuna erişir.
type OturumRepository struct {
	col *mongo.Collection
}

func NewOturumRepository(client *appmongo.Client) *OturumRepository {
	if client == nil {
		return &OturumRepository{}
	}
	return &OturumRepository{col: client.Collection(appmongo.DatabaseName(), oturumCollection)}
}

func (r *OturumRepository) ready() bool {
	return r != nil && r.col != nil
}

func (r *OturumRepository) EnsureIndexes(ctx context.Context) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	_, err := r.col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "yenilemeTokenHash", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "sonKullanma", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(0),
		},
		{Keys: bson.D{{Key: "kullaniciId", Value: 1}}},
	})
	if err != nil {
		return ErrStore
	}
	return nil
}

func (r *OturumRepository) Create(ctx context.Context, doc *Oturum) error {
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

func (r *OturumRepository) GetByID(ctx context.Context, id bson.ObjectID) (*Oturum, error) {
	if !r.ready() {
		return nil, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	var doc Oturum
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, ErrStore
	}
	return &doc, nil
}

func (r *OturumRepository) GetByRefreshHash(ctx context.Context, hash string, now time.Time) (*Oturum, error) {
	if !r.ready() {
		return nil, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	var doc Oturum
	err := r.col.FindOne(ctx, bson.M{
		"yenilemeTokenHash": hash,
		"iptalEdildi":       false,
		"sonKullanma":       bson.M{"$gt": now},
	}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, ErrStore
	}
	return &doc, nil
}

func (r *OturumRepository) ListActiveByUser(ctx context.Context, kullaniciID bson.ObjectID, now time.Time) ([]Oturum, error) {
	if !r.ready() {
		return nil, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	cur, err := r.col.Find(ctx, bson.M{
		"kullaniciId": kullaniciID,
		"iptalEdildi": false,
		"sonKullanma": bson.M{"$gt": now},
	}, options.Find().SetSort(bson.D{{Key: "olusturmaTarihi", Value: -1}}))
	if err != nil {
		return nil, ErrStore
	}
	defer func() {
		if err := cur.Close(ctx); err != nil {
			log.Printf("imleç kapatılamadı")
		}
	}()
	var out []Oturum
	if err := cur.All(ctx, &out); err != nil {
		return nil, ErrStore
	}
	if out == nil {
		out = []Oturum{}
	}
	return out, nil
}

func (r *OturumRepository) Revoke(ctx context.Context, id bson.ObjectID, neden string) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	res, err := r.col.UpdateByID(ctx, id, bson.M{
		"$set": bson.M{"iptalEdildi": true, "iptalNedeni": neden},
	})
	if err != nil {
		return ErrStore
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *OturumRepository) RevokeAllForUser(ctx context.Context, kullaniciID bson.ObjectID, neden string) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	_, err := r.col.UpdateMany(ctx, bson.M{
		"kullaniciId": kullaniciID,
		"iptalEdildi": false,
	}, bson.M{"$set": bson.M{"iptalEdildi": true, "iptalNedeni": neden}})
	if err != nil {
		return ErrStore
	}
	return nil
}
