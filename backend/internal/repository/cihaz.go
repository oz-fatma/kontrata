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

const cihazCollection = "cihazlar"

// Cihaz kayıtlı istemci belgesidir. Parmak izi yalnızca hash olarak tutulur.
type Cihaz struct {
	ID             bson.ObjectID `bson:"_id,omitempty"`
	KullaniciID    bson.ObjectID `bson:"kullaniciId"`
	CihazParmakIzi string        `bson:"cihazParmakIzi"`
	Ad             string        `bson:"ad"`
	Guvenilir      bool          `bson:"guvenilir"`
	IlkGorulme     time.Time     `bson:"ilkGorulme"`
	SonGorulme     time.Time     `bson:"sonGorulme"`
	IPAdresi       string        `bson:"ipAdresi"`
	KullaniciAjani string        `bson:"kullaniciAjani"`
}

// CihazRepository cihazlar koleksiyonuna erişir.
type CihazRepository struct {
	col *mongo.Collection
}

func NewCihazRepository(client *appmongo.Client) *CihazRepository {
	if client == nil {
		return &CihazRepository{}
	}
	return &CihazRepository{col: client.Collection(appmongo.DatabaseName(), cihazCollection)}
}

func (r *CihazRepository) ready() bool {
	return r != nil && r.col != nil
}

func (r *CihazRepository) EnsureIndexes(ctx context.Context) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	_, err := r.col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "kullaniciId", Value: 1},
				{Key: "cihazParmakIzi", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
		{Keys: bson.D{{Key: "kullaniciId", Value: 1}}},
	})
	if err != nil {
		return ErrStore
	}
	return nil
}

func (r *CihazRepository) GetByID(ctx context.Context, id bson.ObjectID) (*Cihaz, error) {
	if !r.ready() {
		return nil, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	var doc Cihaz
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, ErrStore
	}
	return &doc, nil
}

func (r *CihazRepository) GetByUserAndFingerprint(ctx context.Context, kullaniciID bson.ObjectID, hash string) (*Cihaz, error) {
	if !r.ready() {
		return nil, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	var doc Cihaz
	err := r.col.FindOne(ctx, bson.M{
		"kullaniciId":    kullaniciID,
		"cihazParmakIzi": hash,
	}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, ErrStore
	}
	return &doc, nil
}

func (r *CihazRepository) Create(ctx context.Context, doc *Cihaz) error {
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

func (r *CihazRepository) Touch(ctx context.Context, id bson.ObjectID, now time.Time, ip, ua string) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	res, err := r.col.UpdateByID(ctx, id, bson.M{"$set": bson.M{
		"sonGorulme":     now,
		"ipAdresi":       ip,
		"kullaniciAjani": ua,
	}})
	if err != nil {
		return ErrStore
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *CihazRepository) ListByUser(ctx context.Context, kullaniciID bson.ObjectID) ([]Cihaz, error) {
	if !r.ready() {
		return nil, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	cur, err := r.col.Find(ctx, bson.M{"kullaniciId": kullaniciID},
		options.Find().SetSort(bson.D{{Key: "sonGorulme", Value: -1}}))
	if err != nil {
		return nil, ErrStore
	}
	defer func() {
		if err := cur.Close(ctx); err != nil {
			log.Printf("imleç kapatılamadı: %v", err)
		}
	}()
	var out []Cihaz
	if err := cur.All(ctx, &out); err != nil {
		return nil, ErrStore
	}
	if out == nil {
		out = []Cihaz{}
	}
	return out, nil
}

func (r *CihazRepository) Rename(ctx context.Context, id bson.ObjectID, ad string) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	res, err := r.col.UpdateByID(ctx, id, bson.M{"$set": bson.M{"ad": ad}})
	if err != nil {
		return ErrStore
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *CihazRepository) SetTrusted(ctx context.Context, id bson.ObjectID) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	res, err := r.col.UpdateByID(ctx, id, bson.M{"$set": bson.M{"guvenilir": true}})
	if err != nil {
		return ErrStore
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *CihazRepository) Delete(ctx context.Context, id bson.ObjectID) error {
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

func (r *CihazRepository) DeleteByUser(ctx context.Context, kullaniciID bson.ObjectID) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	_, err := r.col.DeleteMany(ctx, bson.M{"kullaniciId": kullaniciID})
	if err != nil {
		return ErrStore
	}
	return nil
}

func (r *CihazRepository) CountByUser(ctx context.Context, kullaniciID bson.ObjectID) (int64, error) {
	if !r.ready() {
		return 0, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	n, err := r.col.CountDocuments(ctx, bson.M{"kullaniciId": kullaniciID})
	if err != nil {
		return 0, ErrStore
	}
	return n, nil
}
