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
	contractsCollection = "sozlesmeler"
	opTimeout           = 5 * time.Second
)

var (
	ErrNotFound    = errors.New("sözleşme bulunamadı")
	ErrUnavailable = errors.New("veritabanı kullanılamıyor")
	ErrInvalidID   = errors.New("geçersiz kimlik")
	ErrStore       = errors.New("veri kaydı başarısız")
)

// Contract MongoDB belgesidir.
type Contract struct {
	ID                bson.ObjectID      `bson:"_id,omitempty"`
	UserID            bson.ObjectID      `bson:"kullaniciId,omitempty"`
	OrganizationID    bson.ObjectID      `bson:"organizasyonId,omitempty"`
	CreatedAt         time.Time          `bson:"olusturmaTarihi"`
	UpdatedAt         time.Time          `bson:"guncellemeTarihi"`
	Status            string             `bson:"durum"`
	FileName          *string            `bson:"dosyaAdi,omitempty"`
	Meta              *ContractMeta      `bson:"meta,omitempty"`
	Period            *Period            `bson:"donem,omitempty"`
	RoomAllotments    []RoomAllotment    `bson:"odaKontenjanlari"`
	Prices            []Price            `bson:"fiyatlar"`
	Release           *ReleaseRule       `bson:"release,omitempty"`
	StopSale          []StopSaleRange    `bson:"stopSale"`
	ChildPolicies     []ChildPolicy      `bson:"cocukPolitikasi,omitempty"`
	CancellationTerms []CancellationTerm `bson:"iptalKosullari,omitempty"`
	NoShow            *NoShow            `bson:"noShow,omitempty"`
	Overbooking       *Overbooking       `bson:"overbooking,omitempty"`
	Payment           *Payment           `bson:"odeme,omitempty"`
	ExtractionMeta    []ExtractionMeta   `bson:"cikarimMeta,omitempty"`
	StoredFileID      string             `bson:"saklananDosyaId,omitempty"`
	Repairs           []string           `bson:"duzeltmeler,omitempty"`
	SchemaErrors      []string           `bson:"semaHatalari,omitempty"`
	ProcessingSeconds *float64           `bson:"islemSuresi,omitempty"`
}

type ContractMeta struct {
	HotelName      *string `bson:"otelAdi,omitempty"`
	AgencyName     *string `bson:"acenteAdi,omitempty"`
	ContractType   *string `bson:"sozlesmeTipi,omitempty"`
	Season         *string `bson:"sezon,omitempty"`
	Currency       *string `bson:"paraBirimi,omitempty"`
	ExchangeBasis  *string `bson:"kurEsasi,omitempty"`
	CompetentCourt *string `bson:"yetkiliMahkeme,omitempty"`
	SignatureDate  *string `bson:"imzaTarihi,omitempty"`
}

type SubPeriod struct {
	Name  string `bson:"ad"`
	Start string `bson:"baslangic"`
	End   string `bson:"bitis"`
}

type Period struct {
	Start      *string     `bson:"baslangic,omitempty"`
	End        *string     `bson:"bitis,omitempty"`
	SubPeriods []SubPeriod `bson:"altDonemler,omitempty"`
}

type RoomAllotment struct {
	RoomType    string  `bson:"odaTipi"`
	Quantity    int32   `bson:"adet"`
	Description *string `bson:"aciklama,omitempty"`
}

type Price struct {
	RoomType      string  `bson:"odaTipi"`
	Board         *string `bson:"pansiyon,omitempty"`
	Amount        float64 `bson:"tutar"`
	Unit          string  `bson:"birim"`
	SubPeriodName *string `bson:"altDonemAd,omitempty"`
}

type ReleaseRule struct {
	Days         int32   `bson:"gun"`
	Scope        *string `bson:"kapsam,omitempty"`
	SourcePhrase *string `bson:"kaynakIfade,omitempty"`
}

type StopSaleRange struct {
	Start              *string `bson:"baslangic,omitempty"`
	End                *string `bson:"bitis,omitempty"`
	Scope              *string `bson:"kapsam,omitempty"`
	NotificationMethod *string `bson:"bildirimYontemi,omitempty"`
	SourcePhrase       *string `bson:"kaynakIfade,omitempty"`
}

type ChildPolicy struct {
	AgeMin          *float64 `bson:"yasMin,omitempty"`
	AgeMax          *float64 `bson:"yasMax,omitempty"`
	DiscountPercent *float64 `bson:"indirimYuzde,omitempty"`
	Free            *bool    `bson:"ucretsiz,omitempty"`
	Condition       *string  `bson:"kosul,omitempty"`
}

type CancellationTerm struct {
	Scope            *string `bson:"kapsam,omitempty"`
	Days             *int32  `bson:"gun,omitempty"`
	CompensationNote *string `bson:"tazminatAciklama,omitempty"`
}

type NoShow struct {
	ResponsibleParty *string `bson:"sorumluTaraf,omitempty"`
	CompensationNote *string `bson:"tazminatAciklama,omitempty"`
}

type Overbooking struct {
	ResponsibleParty *string `bson:"sorumluTaraf,omitempty"`
	Description      *string `bson:"aciklama,omitempty"`
}

type Payment struct {
	DaysAfterInvoice *int32  `bson:"faturaSonrasiGun,omitempty"`
	HasAdvance       *bool   `bson:"avansVar,omitempty"`
	AdvanceNote      *string `bson:"avansAciklama,omitempty"`
}

type ExtractionMeta struct {
	FieldPath    string   `bson:"alanYolu"`
	Confidence   *float64 `bson:"guven,omitempty"`
	SourcePage   *int32   `bson:"kaynakSayfa,omitempty"`
	SourceClause *string  `bson:"kaynakMadde,omitempty"`
}

// ContractRepository sozlesmeler koleksiyonuna erişir.
type ContractRepository struct {
	col *mongo.Collection
}

func NewContractRepository(client *appmongo.Client) *ContractRepository {
	if client == nil {
		return &ContractRepository{}
	}
	return &ContractRepository{col: client.Collection(appmongo.DatabaseName(), contractsCollection)}
}

func (r *ContractRepository) ready() bool {
	return r != nil && r.col != nil
}

func withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, opTimeout)
}

// EnsureIndexes olusturmaTarihi ve durum indekslerini idempotent oluşturur.
func (r *ContractRepository) EnsureIndexes(ctx context.Context) error {
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

func (r *ContractRepository) Create(ctx context.Context, doc *Contract) error {
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

func (r *ContractRepository) GetByID(ctx context.Context, id string) (*Contract, error) {
	if !r.ready() {
		return nil, ErrUnavailable
	}
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrInvalidID
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	var doc Contract
	err = r.col.FindOne(ctx, bson.M{"_id": oid}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, ErrStore
	}
	return &doc, nil
}

func (r *ContractRepository) List(ctx context.Context, filter bson.M, limit, offset int64) ([]Contract, error) {
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
	var out []Contract
	if err := cur.All(ctx, &out); err != nil {
		return nil, ErrStore
	}
	if out == nil {
		out = []Contract{}
	}
	return out, nil
}

func (r *ContractRepository) Update(ctx context.Context, doc *Contract) error {
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

func (r *ContractRepository) Delete(ctx context.Context, id string) error {
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

func (r *ContractRepository) DeleteByUser(ctx context.Context, userID bson.ObjectID) error {
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

func (r *ContractRepository) ListByUser(ctx context.Context, userID bson.ObjectID) ([]Contract, error) {
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
	var out []Contract
	if err := cur.All(ctx, &out); err != nil {
		return nil, ErrStore
	}
	if out == nil {
		out = []Contract{}
	}
	return out, nil
}

func (r *ContractRepository) CountByUser(ctx context.Context, userID bson.ObjectID) (int64, error) {
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

func (r *ContractRepository) DeleteByOrg(ctx context.Context, orgID bson.ObjectID) error {
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

func (r *ContractRepository) ListByOrg(ctx context.Context, orgID bson.ObjectID) ([]Contract, error) {
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
	var out []Contract
	if err := cur.All(ctx, &out); err != nil {
		return nil, ErrStore
	}
	if out == nil {
		out = []Contract{}
	}
	return out, nil
}

func (r *ContractRepository) CountByOrg(ctx context.Context, orgID bson.ObjectID) (int64, error) {
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

// BackfillOrganization kullanıcının kurumsal sözleşmelerine organizasyonId yazar.
func (r *ContractRepository) BackfillOrganization(ctx context.Context, userID, orgID bson.ObjectID) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	_, err := r.col.UpdateMany(ctx, bson.M{
		"kullaniciId": userID,
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
