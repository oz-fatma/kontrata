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

const (
	collectionName = "sozlesmeler"
	opTimeout      = 5 * time.Second
)

var (
	ErrNotFound    = errors.New("sözleşme bulunamadı")
	ErrUnavailable = errors.New("veritabanı kullanılamıyor")
	ErrInvalidID   = errors.New("geçersiz kimlik")
	ErrStore       = errors.New("veri kaydı başarısız")
)

// Sozlesme MongoDB belgesidir.
type Sozlesme struct {
	ID               bson.ObjectID     `bson:"_id,omitempty"`
	KullaniciID      bson.ObjectID     `bson:"kullaniciId,omitempty"`
	OrganizasyonID   bson.ObjectID     `bson:"organizasyonId,omitempty"`
	OlusturmaTarihi  time.Time         `bson:"olusturmaTarihi"`
	GuncellemeTarihi time.Time         `bson:"guncellemeTarihi"`
	Durum            string            `bson:"durum"`
	DosyaAdi         *string           `bson:"dosyaAdi,omitempty"`
	Meta             *SozlesmeMeta     `bson:"meta,omitempty"`
	Donem            *Donem            `bson:"donem,omitempty"`
	OdaKontenjanlari []OdaKontenjani   `bson:"odaKontenjanlari"`
	Fiyatlar         []Fiyat           `bson:"fiyatlar"`
	Release          *ReleaseKurali    `bson:"release,omitempty"`
	StopSale         []StopSaleAraligi `bson:"stopSale"`
	CocukPolitikasi  []CocukPolitikasi `bson:"cocukPolitikasi,omitempty"`
	IptalKosullari   []IptalKosulu     `bson:"iptalKosullari,omitempty"`
	NoShow           *NoShow           `bson:"noShow,omitempty"`
	Overbooking      *Overbooking      `bson:"overbooking,omitempty"`
	Odeme            *Odeme            `bson:"odeme,omitempty"`
	CikarimMeta      []CikarimMeta     `bson:"cikarimMeta,omitempty"`
}

type SozlesmeMeta struct {
	OtelAdi        *string `bson:"otelAdi,omitempty"`
	AcenteAdi      *string `bson:"acenteAdi,omitempty"`
	SozlesmeTipi   *string `bson:"sozlesmeTipi,omitempty"`
	Sezon          *string `bson:"sezon,omitempty"`
	ParaBirimi     *string `bson:"paraBirimi,omitempty"`
	KurEsasi       *string `bson:"kurEsasi,omitempty"`
	YetkiliMahkeme *string `bson:"yetkiliMahkeme,omitempty"`
	ImzaTarihi     *string `bson:"imzaTarihi,omitempty"`
}

type AltDonem struct {
	Ad        string `bson:"ad"`
	Baslangic string `bson:"baslangic"`
	Bitis     string `bson:"bitis"`
}

type Donem struct {
	Baslangic   *string    `bson:"baslangic,omitempty"`
	Bitis       *string    `bson:"bitis,omitempty"`
	AltDonemler []AltDonem `bson:"altDonemler,omitempty"`
}

type OdaKontenjani struct {
	OdaTipi  string  `bson:"odaTipi"`
	Adet     int32   `bson:"adet"`
	Aciklama *string `bson:"aciklama,omitempty"`
}

type Fiyat struct {
	OdaTipi    string  `bson:"odaTipi"`
	Pansiyon   *string `bson:"pansiyon,omitempty"`
	Tutar      float64 `bson:"tutar"`
	Birim      string  `bson:"birim"`
	AltDonemAd *string `bson:"altDonemAd,omitempty"`
}

type ReleaseKurali struct {
	Gun         int32   `bson:"gun"`
	Kapsam      *string `bson:"kapsam,omitempty"`
	KaynakIfade *string `bson:"kaynakIfade,omitempty"`
}

type StopSaleAraligi struct {
	Baslangic       *string `bson:"baslangic,omitempty"`
	Bitis           *string `bson:"bitis,omitempty"`
	Kapsam          *string `bson:"kapsam,omitempty"`
	BildirimYontemi *string `bson:"bildirimYontemi,omitempty"`
	KaynakIfade     *string `bson:"kaynakIfade,omitempty"`
}

type CocukPolitikasi struct {
	YasMin       *float64 `bson:"yasMin,omitempty"`
	YasMax       *float64 `bson:"yasMax,omitempty"`
	IndirimYuzde *float64 `bson:"indirimYuzde,omitempty"`
	Ucretsiz     *bool    `bson:"ucretsiz,omitempty"`
	Kosul        *string  `bson:"kosul,omitempty"`
}

type IptalKosulu struct {
	Kapsam           *string `bson:"kapsam,omitempty"`
	Gun              *int32  `bson:"gun,omitempty"`
	TazminatAciklama *string `bson:"tazminatAciklama,omitempty"`
}

type NoShow struct {
	SorumluTaraf     *string `bson:"sorumluTaraf,omitempty"`
	TazminatAciklama *string `bson:"tazminatAciklama,omitempty"`
}

type Overbooking struct {
	SorumluTaraf *string `bson:"sorumluTaraf,omitempty"`
	Aciklama     *string `bson:"aciklama,omitempty"`
}

type Odeme struct {
	FaturaSonrasiGun *int32  `bson:"faturaSonrasiGun,omitempty"`
	AvansVar         *bool   `bson:"avansVar,omitempty"`
	AvansAciklama    *string `bson:"avansAciklama,omitempty"`
}

type CikarimMeta struct {
	AlanYolu    string   `bson:"alanYolu"`
	Guven       *float64 `bson:"guven,omitempty"`
	KaynakSayfa *int32   `bson:"kaynakSayfa,omitempty"`
	KaynakMadde *string  `bson:"kaynakMadde,omitempty"`
}

// SozlesmeRepository sozlesmeler koleksiyonuna erişir.
type SozlesmeRepository struct {
	col *mongo.Collection
}

func NewSozlesmeRepository(client *appmongo.Client) *SozlesmeRepository {
	if client == nil {
		return &SozlesmeRepository{}
	}
	return &SozlesmeRepository{col: client.Collection(appmongo.DatabaseName(), collectionName)}
}

func (r *SozlesmeRepository) ready() bool {
	return r != nil && r.col != nil
}

func withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, opTimeout)
}

// EnsureIndexes olusturmaTarihi ve durum indekslerini idempotent oluşturur.
func (r *SozlesmeRepository) EnsureIndexes(ctx context.Context) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	_, err := r.col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "olusturmaTarihi", Value: -1}}},
		{Keys: bson.D{{Key: "durum", Value: 1}}},
		{Keys: bson.D{{Key: "kullaniciId", Value: 1}}},
		{Keys: bson.D{{Key: "organizasyonId", Value: 1}}},
	})
	if err != nil {
		return ErrStore
	}
	return nil
}

func (r *SozlesmeRepository) Create(ctx context.Context, doc *Sozlesme) error {
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

func (r *SozlesmeRepository) GetByID(ctx context.Context, id string) (*Sozlesme, error) {
	if !r.ready() {
		return nil, ErrUnavailable
	}
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrInvalidID
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	var doc Sozlesme
	err = r.col.FindOne(ctx, bson.M{"_id": oid}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, ErrStore
	}
	return &doc, nil
}

func (r *SozlesmeRepository) List(ctx context.Context, filter bson.M, limit, offset int64) ([]Sozlesme, error) {
	if !r.ready() {
		return nil, ErrUnavailable
	}
	if filter == nil {
		filter = bson.M{}
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	opts := options.Find().
		SetSort(bson.D{{Key: "olusturmaTarihi", Value: -1}}).
		SetSkip(offset).
		SetLimit(limit)
	cur, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, ErrStore
	}
	defer func() {
		if err := cur.Close(ctx); err != nil {
			log.Printf("imleç kapatılamadı: %v", err)
		}
	}()
	var out []Sozlesme
	if err := cur.All(ctx, &out); err != nil {
		return nil, ErrStore
	}
	if out == nil {
		out = []Sozlesme{}
	}
	return out, nil
}

func (r *SozlesmeRepository) Update(ctx context.Context, doc *Sozlesme) error {
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

func (r *SozlesmeRepository) Delete(ctx context.Context, id string) error {
	if !r.ready() {
		return ErrUnavailable
	}
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return ErrInvalidID
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	res, err := r.col.DeleteOne(ctx, bson.M{"_id": oid})
	if err != nil {
		return ErrStore
	}
	if res.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *SozlesmeRepository) DeleteByUser(ctx context.Context, kullaniciID bson.ObjectID) error {
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

func (r *SozlesmeRepository) ListByUser(ctx context.Context, kullaniciID bson.ObjectID) ([]Sozlesme, error) {
	if !r.ready() {
		return nil, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	cur, err := r.col.Find(ctx, bson.M{"kullaniciId": kullaniciID},
		options.Find().SetSort(bson.D{{Key: "olusturmaTarihi", Value: -1}}))
	if err != nil {
		return nil, ErrStore
	}
	defer func() {
		if err := cur.Close(ctx); err != nil {
			log.Printf("imleç kapatılamadı: %v", err)
		}
	}()
	var out []Sozlesme
	if err := cur.All(ctx, &out); err != nil {
		return nil, ErrStore
	}
	if out == nil {
		out = []Sozlesme{}
	}
	return out, nil
}

func (r *SozlesmeRepository) CountByUser(ctx context.Context, kullaniciID bson.ObjectID) (int64, error) {
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

func (r *SozlesmeRepository) DeleteByOrg(ctx context.Context, orgID bson.ObjectID) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	_, err := r.col.DeleteMany(ctx, bson.M{"organizasyonId": orgID})
	if err != nil {
		return ErrStore
	}
	return nil
}

func (r *SozlesmeRepository) ListByOrg(ctx context.Context, orgID bson.ObjectID) ([]Sozlesme, error) {
	if !r.ready() {
		return nil, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	cur, err := r.col.Find(ctx, bson.M{"organizasyonId": orgID},
		options.Find().SetSort(bson.D{{Key: "olusturmaTarihi", Value: -1}}))
	if err != nil {
		return nil, ErrStore
	}
	defer func() {
		if err := cur.Close(ctx); err != nil {
			log.Printf("imleç kapatılamadı: %v", err)
		}
	}()
	var out []Sozlesme
	if err := cur.All(ctx, &out); err != nil {
		return nil, ErrStore
	}
	if out == nil {
		out = []Sozlesme{}
	}
	return out, nil
}

func (r *SozlesmeRepository) CountByOrg(ctx context.Context, orgID bson.ObjectID) (int64, error) {
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

// BackfillOrganizasyon kullanıcının kurumsal sözleşmelerine organizasyonId yazar.
func (r *SozlesmeRepository) BackfillOrganizasyon(ctx context.Context, kullaniciID, orgID bson.ObjectID) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	_, err := r.col.UpdateMany(ctx, bson.M{
		"kullaniciId": kullaniciID,
		"$or": bson.A{
			bson.M{"organizasyonId": bson.M{"$exists": false}},
			bson.M{"organizasyonId": bson.NilObjectID},
		},
	}, bson.M{"$set": bson.M{"organizasyonId": orgID}})
	if err != nil {
		return ErrStore
	}
	return nil
}
