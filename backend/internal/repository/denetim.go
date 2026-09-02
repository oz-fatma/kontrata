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

const denetimCollection = "denetim_kayitlari"

const (
	OlayKayit                     = "KAYIT"
	OlayEpostaDogrulandi          = "EPOSTA_DOGRULANDI"
	OlayDogrulamaTekrarGonderildi = "DOGRULAMA_TEKRAR_GONDERILDI"
	OlaySifreSifirlamaIstendi     = "SIFRE_SIFIRLAMA_ISTENDI"
	OlaySifreSifirlandi           = "SIFRE_SIFIRLANDI"
	OlayGirisBasarili             = "GIRIS_BASARILI"
	OlayGirisBasarisiz            = "GIRIS_BASARISIZ"
	OlayMFABasarili               = "MFA_BASARILI"
	OlayMFABasarisiz              = "MFA_BASARISIZ"
	OlayOturumYenilendi           = "OTURUM_YENILENDI"
	OlayCikis                     = "CIKIS"
	OlayHesapKilitlendi           = "HESAP_KILITLENDI"
)

// DenetimKaydi kimlik ve yetki işlemlerinin izidir. Şifre, token ve e-posta yazılmaz.
type DenetimKaydi struct {
	ID             bson.ObjectID  `bson:"_id,omitempty"`
	KullaniciID    *bson.ObjectID `bson:"kullaniciId,omitempty"`
	Olay           string         `bson:"olay"`
	IPAdresi       string         `bson:"ipAdresi"`
	KullaniciAjani string         `bson:"kullaniciAjani"`
	Zaman          time.Time      `bson:"zaman"`
	Detay          string         `bson:"detay,omitempty"`
}

// DenetimRepository denetim_kayitlari koleksiyonuna erişir.
type DenetimRepository struct {
	col *mongo.Collection
}

func NewDenetimRepository(client *appmongo.Client) *DenetimRepository {
	if client == nil {
		return &DenetimRepository{}
	}
	return &DenetimRepository{col: client.Collection(appmongo.DatabaseName(), denetimCollection)}
}

func (r *DenetimRepository) ready() bool {
	return r != nil && r.col != nil
}

// EnsureIndexes zaman ve kullanıcı indekslerini oluşturur.
func (r *DenetimRepository) EnsureIndexes(ctx context.Context) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	_, err := r.col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "zaman", Value: -1}}},
		{Keys: bson.D{{Key: "kullaniciId", Value: 1}}},
		{Keys: bson.D{{Key: "olay", Value: 1}}},
	})
	if err != nil {
		return ErrStore
	}
	return nil
}

func (r *DenetimRepository) Insert(ctx context.Context, doc *DenetimKaydi) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	if doc.ID.IsZero() {
		doc.ID = bson.NewObjectID()
	}
	if doc.Zaman.IsZero() {
		doc.Zaman = time.Now().UTC()
	}
	_, err := r.col.InsertOne(ctx, doc)
	if err != nil {
		return ErrStore
	}
	return nil
}

// Latest verilen olayın en yeni kaydını döner; olay boşsa tüm olaylar.
func (r *DenetimRepository) Latest(ctx context.Context, olay string) (*DenetimKaydi, error) {
	if !r.ready() {
		return nil, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	filter := bson.M{}
	if olay != "" {
		filter["olay"] = olay
	}
	opts := options.FindOne().SetSort(bson.D{{Key: "zaman", Value: -1}})
	var doc DenetimKaydi
	err := r.col.FindOne(ctx, filter, opts).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, ErrStore
	}
	return &doc, nil
}
