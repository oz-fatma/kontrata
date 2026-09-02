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
	OlayHesapSilindi              = "HESAP_SILINDI"
	OlayTumOturumlarKapatildi     = "TUM_OTURUMLAR_KAPATILDI"
	OlayUyeDavet                   = "UYE_DAVET"
	OlayUyeCikar                   = "UYE_CIKAR"
	OlayRolDegistir                = "ROL_DEGISTIR"
	OlayOrganizasyonSilindi        = "ORGANIZASYON_SILINDI"
)

// KullaniciSilinmis denetim kaydında anonimleştirilmiş kullanıcı kimliğidir.
const KullaniciSilinmis = "silinmis"

// DenetimKaydi kimlik ve yetki işlemlerinin izidir. Şifre, token ve e-posta yazılmaz.
type DenetimKaydi struct {
	ID             bson.ObjectID `bson:"_id,omitempty"`
	KullaniciID    any           `bson:"kullaniciId,omitempty"`
	Olay           string        `bson:"olay"`
	IPAdresi       string        `bson:"ipAdresi"`
	KullaniciAjani string        `bson:"kullaniciAjani"`
	Zaman          time.Time     `bson:"zaman"`
	Detay          string        `bson:"detay,omitempty"`
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

// AnonymizeByUser kullaniciId'yi "silinmis" yapar; IP ve kullanıcı ajanını boşaltır. Kayıt silinmez.
func (r *DenetimRepository) AnonymizeByUser(ctx context.Context, kullaniciID bson.ObjectID) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	_, err := r.col.UpdateMany(ctx, bson.M{"kullaniciId": kullaniciID}, bson.M{"$set": bson.M{
		"kullaniciId":    KullaniciSilinmis,
		"ipAdresi":       "",
		"kullaniciAjani": "",
	}})
	if err != nil {
		return ErrStore
	}
	return nil
}

func (r *DenetimRepository) ListByUser(ctx context.Context, kullaniciID bson.ObjectID) ([]DenetimKaydi, error) {
	if !r.ready() {
		return nil, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	cur, err := r.col.Find(ctx, bson.M{"kullaniciId": kullaniciID},
		options.Find().SetSort(bson.D{{Key: "zaman", Value: -1}}))
	if err != nil {
		return nil, ErrStore
	}
	defer func() {
		if err := cur.Close(ctx); err != nil {
			log.Printf("imleç kapatılamadı: %v", err)
		}
	}()
	var out []DenetimKaydi
	if err := cur.All(ctx, &out); err != nil {
		return nil, ErrStore
	}
	if out == nil {
		out = []DenetimKaydi{}
	}
	return out, nil
}

func (r *DenetimRepository) CountByUser(ctx context.Context, kullaniciID bson.ObjectID) (int64, error) {
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

func (r *DenetimRepository) CountSilinmis(ctx context.Context) (int64, error) {
	if !r.ready() {
		return 0, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	n, err := r.col.CountDocuments(ctx, bson.M{"kullaniciId": KullaniciSilinmis})
	if err != nil {
		return 0, ErrStore
	}
	return n, nil
}
