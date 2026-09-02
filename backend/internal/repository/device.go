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

const deviceCollection = "cihazlar"

// Device kayıtlı istemci belgesidir. Parmak izi yalnızca hash olarak tutulur.
type Device struct {
	ID          bson.ObjectID `bson:"_id,omitempty"`
	UserID      bson.ObjectID `bson:"kullaniciId"`
	Fingerprint string        `bson:"cihazParmakIzi"`
	Name        string        `bson:"ad"`
	Trusted     bool          `bson:"guvenilir"`
	FirstSeen   time.Time     `bson:"ilkGorulme"`
	LastSeen    time.Time     `bson:"sonGorulme"`
	IPAddress   string        `bson:"ipAdresi"`
	UserAgent   string        `bson:"kullaniciAjani"`
}

// DeviceRepository cihazlar koleksiyonuna erişir.
type DeviceRepository struct {
	col *mongo.Collection
}

func NewDeviceRepository(client *appmongo.Client) *DeviceRepository {
	if client == nil {
		return &DeviceRepository{}
	}
	return &DeviceRepository{col: client.Collection(appmongo.DatabaseName(), deviceCollection)}
}

func (r *DeviceRepository) ready() bool {
	return r != nil && r.col != nil
}

func (r *DeviceRepository) EnsureIndexes(ctx context.Context) error {
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

func (r *DeviceRepository) GetByID(ctx context.Context, id bson.ObjectID) (*Device, error) {
	if !r.ready() {
		return nil, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	var doc Device
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, ErrStore
	}
	return &doc, nil
}

func (r *DeviceRepository) GetByUserAndFingerprint(ctx context.Context, userID bson.ObjectID, hash string) (*Device, error) {
	if !r.ready() {
		return nil, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	var doc Device
	err := r.col.FindOne(ctx, bson.M{
		"kullaniciId":    userID,
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

func (r *DeviceRepository) Create(ctx context.Context, doc *Device) error {
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

func (r *DeviceRepository) Touch(ctx context.Context, id bson.ObjectID, now time.Time, ip, ua string) error {
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

func (r *DeviceRepository) ListByUser(ctx context.Context, userID bson.ObjectID) ([]Device, error) {
	if !r.ready() {
		return nil, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	cur, err := r.col.Find(ctx, bson.M{"kullaniciId": userID},
		options.Find().SetSort(bson.D{{Key: "sonGorulme", Value: -1}}))
	if err != nil {
		return nil, ErrStore
	}
	defer func() {
		if err := cur.Close(ctx); err != nil {
			log.Printf("imleç kapatılamadı: %v", err)
		}
	}()
	var out []Device
	if err := cur.All(ctx, &out); err != nil {
		return nil, ErrStore
	}
	if out == nil {
		out = []Device{}
	}
	return out, nil
}

func (r *DeviceRepository) Rename(ctx context.Context, id bson.ObjectID, name string) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	res, err := r.col.UpdateByID(ctx, id, bson.M{"$set": bson.M{"ad": name}})
	if err != nil {
		return ErrStore
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *DeviceRepository) SetTrusted(ctx context.Context, id bson.ObjectID) error {
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

func (r *DeviceRepository) Delete(ctx context.Context, id bson.ObjectID) error {
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

func (r *DeviceRepository) DeleteByUser(ctx context.Context, userID bson.ObjectID) error {
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

func (r *DeviceRepository) CountByUser(ctx context.Context, userID bson.ObjectID) (int64, error) {
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
