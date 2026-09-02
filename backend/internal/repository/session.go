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

const sessionCollection = "oturumlar"

const (
	RevokeLogout          = "cikis"
	RevokeRefresh         = "yenileme"
	RevokePasswordReset   = "sifre_sifirlama"
	RevokeDeviceRemoved   = "cihaz_kaldirildi"
	RevokePreDeviceRecord = "cihaz_kaydi_oncesi"
	RevokeAllSessions     = "tum_oturumlar_kapatildi"
)

// Session yenileme jetonu hash'i ile saklanan oturum belgesidir.
type Session struct {
	ID               bson.ObjectID `bson:"_id,omitempty"`
	UserID           bson.ObjectID `bson:"kullaniciId"`
	RefreshTokenHash string        `bson:"yenilemeTokenHash"`
	CreatedAt        time.Time     `bson:"olusturmaTarihi"`
	ExpiresAt        time.Time     `bson:"sonKullanma"`
	Revoked          bool          `bson:"iptalEdildi"`
	RevokeReason     string        `bson:"iptalNedeni,omitempty"`
	IPAddress        string        `bson:"ipAdresi"`
	UserAgent        string        `bson:"kullaniciAjani"`
	DeviceID         bson.ObjectID `bson:"cihazId"`
}

// SessionRepository oturumlar koleksiyonuna erişir.
type SessionRepository struct {
	col *mongo.Collection
}

func NewSessionRepository(client *appmongo.Client) *SessionRepository {
	if client == nil {
		return &SessionRepository{}
	}
	return &SessionRepository{col: client.Collection(appmongo.DatabaseName(), sessionCollection)}
}

func (r *SessionRepository) ready() bool {
	return r != nil && r.col != nil
}

func (r *SessionRepository) EnsureIndexes(ctx context.Context) error {
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
		{Keys: bson.D{{Key: "cihazId", Value: 1}}},
	})
	if err != nil {
		return ErrStore
	}
	return nil
}

func (r *SessionRepository) Create(ctx context.Context, doc *Session) error {
	if !r.ready() {
		return ErrUnavailable
	}
	if doc == nil || doc.DeviceID.IsZero() {
		return ErrInvalidID
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

func (r *SessionRepository) GetByID(ctx context.Context, id bson.ObjectID) (*Session, error) {
	if !r.ready() {
		return nil, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	var doc Session
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, ErrStore
	}
	return &doc, nil
}

func (r *SessionRepository) GetByRefreshHash(ctx context.Context, hash string, now time.Time) (*Session, error) {
	if !r.ready() {
		return nil, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	var doc Session
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

func (r *SessionRepository) ListActiveByUser(ctx context.Context, userID bson.ObjectID, now time.Time) ([]Session, error) {
	if !r.ready() {
		return nil, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	cur, err := r.col.Find(ctx, bson.M{
		"kullaniciId": userID,
		"iptalEdildi": false,
		"sonKullanma": bson.M{"$gt": now},
	}, options.Find().SetSort(bson.D{{Key: "olusturmaTarihi", Value: -1}}))
	if err != nil {
		return nil, ErrStore
	}
	defer func() {
		if err := cur.Close(ctx); err != nil {
			log.Printf("imleç kapatılamadı: %v", err)
		}
	}()
	var out []Session
	if err := cur.All(ctx, &out); err != nil {
		return nil, ErrStore
	}
	if out == nil {
		out = []Session{}
	}
	return out, nil
}

func (r *SessionRepository) Revoke(ctx context.Context, id bson.ObjectID, reason string) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	res, err := r.col.UpdateByID(ctx, id, bson.M{
		"$set": bson.M{"iptalEdildi": true, "iptalNedeni": reason},
	})
	if err != nil {
		return ErrStore
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *SessionRepository) RevokeAllForUser(ctx context.Context, userID bson.ObjectID, reason string) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	_, err := r.col.UpdateMany(ctx, bson.M{
		"kullaniciId": userID,
		"iptalEdildi": false,
	}, bson.M{"$set": bson.M{"iptalEdildi": true, "iptalNedeni": reason}})
	if err != nil {
		return ErrStore
	}
	return nil
}

func (r *SessionRepository) RevokeByDevice(ctx context.Context, deviceID bson.ObjectID, reason string) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	_, err := r.col.UpdateMany(ctx, bson.M{
		"cihazId":     deviceID,
		"iptalEdildi": false,
	}, bson.M{"$set": bson.M{"iptalEdildi": true, "iptalNedeni": reason}})
	if err != nil {
		return ErrStore
	}
	return nil
}

// RevokeMissingDevice cihaz kaydı öncesi açılmış oturumları bir kerelik iptal eder.
func (r *SessionRepository) RevokeMissingDevice(ctx context.Context) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	_, err := r.col.UpdateMany(ctx, bson.M{
		"iptalEdildi": false,
		"$or": bson.A{
			bson.M{"cihazId": bson.M{"$exists": false}},
			bson.M{"cihazId": nil},
			bson.M{"cihazId": bson.NilObjectID},
		},
	}, bson.M{"$set": bson.M{
		"iptalEdildi": true,
		"iptalNedeni": RevokePreDeviceRecord,
	}})
	if err != nil {
		return ErrStore
	}
	return nil
}

// RevokeAllExcept kullanıcının mevcut oturumu dışındaki aktif oturumları iptal eder.
func (r *SessionRepository) RevokeAllExcept(ctx context.Context, userID, except bson.ObjectID, reason string) (int64, error) {
	if !r.ready() {
		return 0, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	res, err := r.col.UpdateMany(ctx, bson.M{
		"kullaniciId": userID,
		"iptalEdildi": false,
		"_id":         bson.M{"$ne": except},
	}, bson.M{"$set": bson.M{"iptalEdildi": true, "iptalNedeni": reason}})
	if err != nil {
		return 0, ErrStore
	}
	return res.ModifiedCount, nil
}

func (r *SessionRepository) ListByUser(ctx context.Context, userID bson.ObjectID) ([]Session, error) {
	if !r.ready() {
		return nil, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	cur, err := r.col.Find(ctx, bson.M{"kullaniciId": userID},
		options.Find().SetSort(bson.D{{Key: "olusturmaTarihi", Value: -1}}))
	if err != nil {
		return nil, ErrStore
	}
	defer func() {
		if err := cur.Close(ctx); err != nil {
			log.Printf("imleç kapatılamadı: %v", err)
		}
	}()
	var out []Session
	if err := cur.All(ctx, &out); err != nil {
		return nil, ErrStore
	}
	if out == nil {
		out = []Session{}
	}
	return out, nil
}

func (r *SessionRepository) DeleteByUser(ctx context.Context, userID bson.ObjectID) error {
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

func (r *SessionRepository) CountByUser(ctx context.Context, userID bson.ObjectID) (int64, error) {
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
