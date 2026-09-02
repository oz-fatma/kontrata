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

const mfaCollection = "mfa_kodlari"

// MFAKodu hash'lenmiş altı haneli giriş kodudur.
type MFAKodu struct {
	ID              bson.ObjectID `bson:"_id,omitempty"`
	KullaniciID     bson.ObjectID `bson:"kullaniciId"`
	KodHash         string        `bson:"kodHash"`
	SonKullanma     time.Time     `bson:"sonKullanma"`
	Kullanildi      bool          `bson:"kullanildi"`
	DenemeSayisi    int32         `bson:"denemeSayisi"`
	OlusturmaTarihi time.Time     `bson:"olusturmaTarihi"`
}

// MFAKoduRepository mfa_kodlari koleksiyonuna erişir.
type MFAKoduRepository struct {
	col *mongo.Collection
}

func NewMFAKoduRepository(client *appmongo.Client) *MFAKoduRepository {
	if client == nil {
		return &MFAKoduRepository{}
	}
	return &MFAKoduRepository{col: client.Collection(appmongo.DatabaseName(), mfaCollection)}
}

func (r *MFAKoduRepository) ready() bool {
	return r != nil && r.col != nil
}

func (r *MFAKoduRepository) EnsureIndexes(ctx context.Context) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	_, err := r.col.Indexes().CreateMany(ctx, []mongo.IndexModel{
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

func (r *MFAKoduRepository) Create(ctx context.Context, doc *MFAKodu) error {
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

func (r *MFAKoduRepository) InvalidateUnused(ctx context.Context, kullaniciID bson.ObjectID) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	_, err := r.col.UpdateMany(ctx, bson.M{
		"kullaniciId": kullaniciID,
		"kullanildi":  false,
	}, bson.M{"$set": bson.M{"kullanildi": true}})
	if err != nil {
		return ErrStore
	}
	return nil
}

// GetActive kullanılmamış, süresi dolmamış ve deneme sınırı aşılmamış kodu döner.
func (r *MFAKoduRepository) GetActive(ctx context.Context, kullaniciID bson.ObjectID, now time.Time) (*MFAKodu, error) {
	if !r.ready() {
		return nil, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	opts := options.FindOne().SetSort(bson.D{{Key: "olusturmaTarihi", Value: -1}})
	var doc MFAKodu
	err := r.col.FindOne(ctx, bson.M{
		"kullaniciId":  kullaniciID,
		"kullanildi":   false,
		"sonKullanma":  bson.M{"$gt": now},
		"denemeSayisi": bson.M{"$lt": 5},
	}, opts).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, ErrStore
	}
	return &doc, nil
}

func (r *MFAKoduRepository) GetByID(ctx context.Context, id bson.ObjectID) (*MFAKodu, error) {
	if !r.ready() {
		return nil, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	var doc MFAKodu
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, ErrStore
	}
	return &doc, nil
}

func (r *MFAKoduRepository) Update(ctx context.Context, doc *MFAKodu) error {
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

// RegisterFailure deneme sayısını artırır; sınıra ulaşınca kodu geçersiz kılar.
func (r *MFAKoduRepository) RegisterFailure(ctx context.Context, id bson.ObjectID) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	res, err := r.col.UpdateByID(ctx, id, bson.A{
		bson.M{"$set": bson.M{
			"denemeSayisi": bson.M{"$add": bson.A{"$denemeSayisi", 1}},
			"kullanildi": bson.M{"$cond": bson.A{
				bson.M{"$gte": bson.A{bson.M{"$add": bson.A{"$denemeSayisi", 1}}, int32(5)}},
				true,
				"$kullanildi",
			}},
		}},
	})
	if err != nil {
		return ErrStore
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *MFAKoduRepository) MarkUsed(ctx context.Context, id bson.ObjectID) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	res, err := r.col.UpdateByID(ctx, id, bson.M{"$set": bson.M{"kullanildi": true}})
	if err != nil {
		return ErrStore
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}
