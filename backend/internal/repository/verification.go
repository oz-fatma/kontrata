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

const verificationTokenCollection = "dogrulama_tokenlari"

const (
	PurposeEmailVerification = "EPOSTA_DOGRULAMA"
	PurposePasswordReset     = "SIFRE_SIFIRLAMA"
	PurposeAccountDelete     = "HESAP_SILME"
)

// VerificationToken hash'lenmiş doğrulama kodu belgesidir. Düz metin yazılmaz.
type VerificationToken struct {
	ID        bson.ObjectID `bson:"_id,omitempty"`
	UserID    bson.ObjectID `bson:"kullaniciId"`
	Token     string        `bson:"token"`
	Purpose   string        `bson:"amac"`
	ExpiresAt time.Time     `bson:"sonKullanma"`
	Used      bool          `bson:"kullanildi"`
}

// VerificationTokenRepository dogrulama_tokenlari koleksiyonuna erişir.
type VerificationTokenRepository struct {
	col *mongo.Collection
}

func NewVerificationTokenRepository(client *appmongo.Client) *VerificationTokenRepository {
	if client == nil {
		return &VerificationTokenRepository{}
	}
	return &VerificationTokenRepository{col: client.Collection(appmongo.DatabaseName(), verificationTokenCollection)}
}

func (r *VerificationTokenRepository) ready() bool {
	return r != nil && r.col != nil
}

// EnsureIndexes token hash benzersizliği ve sonKullanma TTL indekslerini oluşturur.
func (r *VerificationTokenRepository) EnsureIndexes(ctx context.Context) error {
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

func (r *VerificationTokenRepository) Create(ctx context.Context, doc *VerificationToken) error {
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

func (r *VerificationTokenRepository) GetByHash(ctx context.Context, hash string) (*VerificationToken, error) {
	if !r.ready() {
		return nil, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	var doc VerificationToken
	err := r.col.FindOne(ctx, bson.M{"token": hash}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, ErrStore
	}
	return &doc, nil
}

func (r *VerificationTokenRepository) Update(ctx context.Context, doc *VerificationToken) error {
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
func (r *VerificationTokenRepository) InvalidateUnused(ctx context.Context, userID bson.ObjectID, purpose string) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	_, err := r.col.UpdateMany(ctx, bson.M{
		"kullaniciId": userID,
		"amac":        purpose,
		"kullanildi":  false,
	}, bson.M{"$set": bson.M{"kullanildi": true}})
	if err != nil {
		return ErrStore
	}
	return nil
}

// Consume geçerli ve kullanılmamış kodu atomik olarak kullanılmış işaretler.
func (r *VerificationTokenRepository) Consume(ctx context.Context, hash, purpose string, now time.Time) (*VerificationToken, error) {
	if !r.ready() {
		return nil, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	opts := options.FindOneAndUpdate().SetReturnDocument(options.Before)
	var doc VerificationToken
	err := r.col.FindOneAndUpdate(ctx, bson.M{
		"token":      hash,
		"amac":       purpose,
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

func (r *VerificationTokenRepository) DeleteByUser(ctx context.Context, userID bson.ObjectID) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	_, err := r.col.DeleteMany(ctx, bson.M{"kullaniciId": userID})
	if err != nil {
		return ErrStore
	}
	return nil
}

func (r *VerificationTokenRepository) CountByUser(ctx context.Context, userID bson.ObjectID) (int64, error) {
	if !r.ready() {
		return 0, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	n, err := r.col.CountDocuments(ctx, bson.M{"kullaniciId": userID})
	if err != nil {
		return 0, ErrStore
	}
	return n, nil
}
