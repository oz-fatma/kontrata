package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	appmongo "github.com/oz-fatma/kontrata/backend/internal/mongo"
)

const kullaniciCollection = "kullanicilar"

const (
	DurumAktif     = "AKTIF"
	DurumBeklemede = "BEKLEMEDE"
	DurumAskida    = "ASKIDA"

	HesapBireysel = "BIREYSEL"
	HesapKurumsal = "KURUMSAL"

	RolSahip         = "SAHIP"
	RolYonetici      = "YONETICI"
	RolGoruntuleyici = "GORUNTULEYICI"
)

// ErrDuplicate benzersiz indeks çakışmasıdır; GraphQL'e yansımaz.
var ErrDuplicate = errors.New("kayıt zaten var")

// Kullanici kullanicilar koleksiyonu belgesidir.
type Kullanici struct {
	ID               bson.ObjectID `bson:"_id,omitempty"`
	Eposta           string        `bson:"eposta"`
	SifreHash        string        `bson:"sifreHash"`
	EpostaDogrulandi bool          `bson:"epostaDogrulandi"`
	Durum            string        `bson:"durum"`
	HesapTipi        string        `bson:"hesapTipi"`
	OrganizasyonID   bson.ObjectID `bson:"organizasyonId,omitempty"`
	Rol              string        `bson:"rol"`
	OlusturmaTarihi  time.Time     `bson:"olusturmaTarihi"`
	GuncellemeTarihi time.Time     `bson:"guncellemeTarihi"`
}

// KullaniciRepository kullanicilar koleksiyonuna erişir.
type KullaniciRepository struct {
	col *mongo.Collection
}

func NewKullaniciRepository(client *appmongo.Client) *KullaniciRepository {
	if client == nil {
		return &KullaniciRepository{}
	}
	return &KullaniciRepository{col: client.Collection(appmongo.DatabaseName(), kullaniciCollection)}
}

func (r *KullaniciRepository) ready() bool {
	return r != nil && r.col != nil
}

// EnsureIndexes eposta için benzersiz indeks oluşturur.
func (r *KullaniciRepository) EnsureIndexes(ctx context.Context) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	_, err := r.col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "eposta", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{Keys: bson.D{{Key: "organizasyonId", Value: 1}}},
	})
	if err != nil {
		return ErrStore
	}
	return nil
}

func (r *KullaniciRepository) Create(ctx context.Context, doc *Kullanici) error {
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

func (r *KullaniciRepository) GetByEposta(ctx context.Context, eposta string) (*Kullanici, error) {
	if !r.ready() {
		return nil, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	var doc Kullanici
	err := r.col.FindOne(ctx, bson.M{"eposta": strings.ToLower(strings.TrimSpace(eposta))}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, ErrStore
	}
	return &doc, nil
}

func (r *KullaniciRepository) GetByID(ctx context.Context, id bson.ObjectID) (*Kullanici, error) {
	if !r.ready() {
		return nil, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	var doc Kullanici
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, ErrStore
	}
	return &doc, nil
}

// MarkEmailVerified e-postayı doğrulanmış işaretler; BEKLEMEDE ise AKTIF yapar.
func (r *KullaniciRepository) MarkEmailVerified(ctx context.Context, id bson.ObjectID) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	now := time.Now().UTC()
	res, err := r.col.UpdateByID(ctx, id, bson.A{
		bson.M{"$set": bson.M{
			"epostaDogrulandi": true,
			"guncellemeTarihi": now,
			"durum": bson.M{"$cond": bson.A{
				bson.M{"$eq": bson.A{"$durum", DurumBeklemede}},
				DurumAktif,
				"$durum",
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

// UpdatePassword şifre özetini değiştirir. Düz şifre kabul etmez.
func (r *KullaniciRepository) UpdatePassword(ctx context.Context, id bson.ObjectID, sifreHash string) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	res, err := r.col.UpdateByID(ctx, id, bson.M{
		"$set": bson.M{
			"sifreHash":        sifreHash,
			"guncellemeTarihi": time.Now().UTC(),
		},
	})
	if err != nil {
		return ErrStore
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *KullaniciRepository) Delete(ctx context.Context, id bson.ObjectID) error {
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

// BackfillHesapAlanlari eski belgelere bireysel / sahip yazar.
func (r *KullaniciRepository) BackfillHesapAlanlari(ctx context.Context) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	_, err := r.col.UpdateMany(ctx, bson.M{
		"$or": bson.A{
			bson.M{"hesapTipi": bson.M{"$exists": false}},
			bson.M{"hesapTipi": ""},
		},
	}, bson.M{"$set": bson.M{"hesapTipi": HesapBireysel}})
	if err != nil {
		return ErrStore
	}
	_, err = r.col.UpdateMany(ctx, bson.M{
		"$or": bson.A{
			bson.M{"rol": bson.M{"$exists": false}},
			bson.M{"rol": ""},
		},
	}, bson.M{"$set": bson.M{"rol": RolSahip}})
	if err != nil {
		return ErrStore
	}
	return nil
}

func (r *KullaniciRepository) SetOrganizasyon(ctx context.Context, id, orgID bson.ObjectID, hesapTipi, rol string) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	res, err := r.col.UpdateByID(ctx, id, bson.M{"$set": bson.M{
		"organizasyonId":   orgID,
		"hesapTipi":        hesapTipi,
		"rol":              rol,
		"guncellemeTarihi": time.Now().UTC(),
	}})
	if err != nil {
		return ErrStore
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *KullaniciRepository) SetRol(ctx context.Context, id bson.ObjectID, rol string) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	res, err := r.col.UpdateByID(ctx, id, bson.M{"$set": bson.M{
		"rol":              rol,
		"guncellemeTarihi": time.Now().UTC(),
	}})
	if err != nil {
		return ErrStore
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *KullaniciRepository) ListByOrg(ctx context.Context, orgID bson.ObjectID) ([]Kullanici, error) {
	if !r.ready() {
		return nil, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	cur, err := r.col.Find(ctx, bson.M{"organizasyonId": orgID},
		options.Find().SetSort(bson.D{{Key: "olusturmaTarihi", Value: 1}}))
	if err != nil {
		return nil, ErrStore
	}
	defer func() { _ = cur.Close(ctx) }()
	var out []Kullanici
	if err := cur.All(ctx, &out); err != nil {
		return nil, ErrStore
	}
	if out == nil {
		out = []Kullanici{}
	}
	return out, nil
}

func (r *KullaniciRepository) CountByOrg(ctx context.Context, orgID bson.ObjectID) (int64, error) {
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

func (r *KullaniciRepository) ListKurumsal(ctx context.Context) ([]Kullanici, error) {
	if !r.ready() {
		return nil, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	cur, err := r.col.Find(ctx, bson.M{
		"organizasyonId": bson.M{"$exists": true, "$ne": bson.NilObjectID},
	})
	if err != nil {
		return nil, ErrStore
	}
	defer func() { _ = cur.Close(ctx) }()
	var out []Kullanici
	if err := cur.All(ctx, &out); err != nil {
		return nil, ErrStore
	}
	if out == nil {
		out = []Kullanici{}
	}
	return out, nil
}

// DetachOrg üyeyi bireysel sahip yapar (organizasyon bağı kesilir).
func (r *KullaniciRepository) DetachOrg(ctx context.Context, id bson.ObjectID) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	res, err := r.col.UpdateByID(ctx, id, bson.M{
		"$set": bson.M{
			"hesapTipi":        HesapBireysel,
			"rol":              RolSahip,
			"guncellemeTarihi": time.Now().UTC(),
		},
		"$unset": bson.M{"organizasyonId": ""},
	})
	if err != nil {
		return ErrStore
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *KullaniciRepository) DetachOrgByOrg(ctx context.Context, orgID bson.ObjectID) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	_, err := r.col.UpdateMany(ctx, bson.M{"organizasyonId": orgID}, bson.M{
		"$set": bson.M{
			"hesapTipi":        HesapBireysel,
			"rol":              RolSahip,
			"guncellemeTarihi": time.Now().UTC(),
		},
		"$unset": bson.M{"organizasyonId": ""},
	})
	if err != nil {
		return ErrStore
	}
	return nil
}
