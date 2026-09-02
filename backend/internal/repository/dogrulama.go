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

const dogrulamaTokenCollection = "dogrulama_tokenlari"

const (
	AmacEpostaDogrulama = "EPOSTA_DOGRULAMA"
	AmacSifreSifirlama  = "SIFRE_SIFIRLAMA"
)

// DogrulamaTokeni hash'lenmiş doğrulama kodu belgesidir. Düz metin yazılmaz.
type DogrulamaTokeni struct {
	ID          bson.ObjectID `bson:"_id,omitempty"`
	KullaniciID bson.ObjectID `bson:"kullaniciId"`
	Token       string        `bson:"token"`
	Amac        string        `bson:"amac"`
	SonKullanma time.Time     `bson:"sonKullanma"`
	Kullanildi  bool          `bson:"kullanildi"`
}

// DogrulamaTokenRepository dogrulama_tokenlari koleksiyonuna erişir.
type DogrulamaTokenRepository struct {
	col *mongo.Collection
}

func NewDogrulamaTokenRepository(client *appmongo.Client) *DogrulamaTokenRepository {
	if client == nil {
		return &DogrulamaTokenRepository{}
	}
	return &DogrulamaTokenRepository{col: client.Collection(appmongo.DatabaseName(), dogrulamaTokenCollection)}
}

func (r *DogrulamaTokenRepository) ready() bool {
	return r != nil && r.col != nil
}

// EnsureIndexes token hash benzersizliği ve sonKullanma TTL indekslerini oluşturur.
func (r *DogrulamaTokenRepository) EnsureIndexes(ctx context.Context) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	_, err := r.col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "token", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "sonKullanma", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(0),
		},
		{
			Keys: bson.D{{Key: "kullaniciId", Value: 1}, {Key: "amac", Value: 1}},
		},
	})
	if err != nil {
		return ErrStore
	}
	return nil
}

func (r *DogrulamaTokenRepository) Create(ctx context.Context, doc *DogrulamaTokeni) error {
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

func (r *DogrulamaTokenRepository) GetByHash(ctx context.Context, hash string) (*DogrulamaTokeni, error) {
	if !r.ready() {
		return nil, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	var doc DogrulamaTokeni
	err := r.col.FindOne(ctx, bson.M{"token": hash}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, ErrStore
	}
	return &doc, nil
}

func (r *DogrulamaTokenRepository) Update(ctx context.Context, doc *DogrulamaTokeni) error {
	if !r.ready() {
		return ErrUnavailable
	}
	if doc.ID.IsZero() {
		return ErrInvalidID
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	res, err := r.col.ReplaceOne(ctx, bson.M{"_id": doc.ID}, doc)
	if err != nil {
		return ErrStore
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

// InvalidateUnused aynı kullanıcı ve amaçtaki kullanılmamış kodları geçersiz kılar.
func (r *DogrulamaTokenRepository) InvalidateUnused(ctx context.Context, kullaniciID bson.ObjectID, amac string) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	_, err := r.col.UpdateMany(ctx, bson.M{
		"kullaniciId": kullaniciID,
		"amac":        amac,
		"kullanildi":  false,
	}, bson.M{"$set": bson.M{"kullanildi": true}})
	if err != nil {
		return ErrStore
	}
	return nil
}

// Consume geçerli ve kullanılmamış kodu atomik olarak kullanılmış işaretler.
func (r *DogrulamaTokenRepository) Consume(ctx context.Context, hash, amac string, now time.Time) (*DogrulamaTokeni, error) {
	if !r.ready() {
		return nil, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	opts := options.FindOneAndUpdate().SetReturnDocument(options.Before)
	var doc DogrulamaTokeni
	err := r.col.FindOneAndUpdate(ctx, bson.M{
		"token":      hash,
		"amac":       amac,
		"kullanildi": false,
		"sonKullanma": bson.M{
			"$gt": now,
		},
	}, bson.M{"$set": bson.M{"kullanildi": true}}, opts).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, ErrStore
	}
	return &doc, nil
}
