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

const auditCollection = "denetim_kayitlari"

const (
	EventRegister               = "KAYIT"
	EventEmailVerified          = "EPOSTA_DOGRULANDI"
	EventVerificationResent     = "DOGRULAMA_TEKRAR_GONDERILDI"
	EventPasswordResetRequested = "SIFRE_SIFIRLAMA_ISTENDI"
	EventPasswordReset          = "SIFRE_SIFIRLANDI"
	EventLoginSuccess           = "GIRIS_BASARILI"
	EventLoginFailure           = "GIRIS_BASARISIZ"
	EventMFASuccess             = "MFA_BASARILI"
	EventMFAFailure             = "MFA_BASARISIZ"
	EventSessionRefreshed       = "OTURUM_YENILENDI"
	EventLogout                 = "CIKIS"
	EventAccountLocked          = "HESAP_KILITLENDI"
	EventAccountDeleted         = "HESAP_SILINDI"
	EventAllSessionsRevoked     = "TUM_OTURUMLAR_KAPATILDI"
	EventMemberInvited          = "UYE_DAVET"
	EventMemberRemoved          = "UYE_CIKAR"
	EventRoleChanged            = "ROL_DEGISTIR"
	EventOrganizationDeleted    = "ORGANIZASYON_SILINDI"
	EventContractApproved       = "SOZLESME_ONAYLANDI"
	EventContractFieldUpdated   = "SOZLESME_ALAN_GUNCELLENDI"
	EventPromptUpdated          = "PROMPT_GUNCELLENDI"
	EventPromptReverted         = "PROMPT_SURUME_DONULDU"
	EventSettingsUpdated        = "AYARLAR_GUNCELLENDI"
)

// UserDeleted denetim kaydında anonimleştirilmiş kullanıcı kimliğidir.
const UserDeleted = "silinmis"

// AuditRecord kimlik ve yetki işlemlerinin izidir. Şifre, token ve e-posta yazılmaz.
type AuditRecord struct {
	ID         bson.ObjectID `bson:"_id,omitempty"`
	UserID     any           `bson:"kullaniciId,omitempty"`
	Event      string        `bson:"olay"`
	IPAddress  string        `bson:"ipAdresi"`
	UserAgent  string        `bson:"kullaniciAjani"`
	OccurredAt time.Time     `bson:"zaman"`
	Detail     string        `bson:"detay,omitempty"`
}

// AuditRepository denetim_kayitlari koleksiyonuna erişir.
type AuditRepository struct {
	col *mongo.Collection
}

func NewAuditRepository(client *appmongo.Client) *AuditRepository {
	if client == nil {
		return &AuditRepository{}
	}
	return &AuditRepository{col: client.Collection(appmongo.DatabaseName(), auditCollection)}
}

func (r *AuditRepository) ready() bool {
	return r != nil && r.col != nil
}

// EnsureIndexes zaman ve kullanıcı indekslerini oluşturur.
func (r *AuditRepository) EnsureIndexes(ctx context.Context) error {
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

func (r *AuditRepository) Insert(ctx context.Context, doc *AuditRecord) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	if doc.ID.IsZero() {
		doc.ID = bson.NewObjectID()
	}
	if doc.OccurredAt.IsZero() {
		doc.OccurredAt = time.Now().UTC()
	}
	_, err := r.col.InsertOne(ctx, doc)
	if err != nil {
		return ErrStore
	}
	return nil
}

// Latest verilen olayın en yeni kaydını döner; olay boşsa tüm olaylar.
func (r *AuditRepository) Latest(ctx context.Context, event string) (*AuditRecord, error) {
	if !r.ready() {
		return nil, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	filter := bson.M{}
	if event != "" {
		filter["olay"] = event
	}
	opts := options.FindOne().SetSort(bson.D{{Key: "zaman", Value: -1}})
	var doc AuditRecord
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
func (r *AuditRepository) AnonymizeByUser(ctx context.Context, userID bson.ObjectID) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	_, err := r.col.UpdateMany(ctx, bson.M{"kullaniciId": userID}, bson.M{"$set": bson.M{
		"kullaniciId":    UserDeleted,
		"ipAdresi":       "",
		"kullaniciAjani": "",
	}})
	if err != nil {
		return ErrStore
	}
	return nil
}

func (r *AuditRepository) ListByUser(ctx context.Context, userID bson.ObjectID) ([]AuditRecord, error) {
	if !r.ready() {
		return nil, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	cur, err := r.col.Find(ctx, bson.M{"kullaniciId": userID},
		options.Find().SetSort(bson.D{{Key: "zaman", Value: -1}}))
	if err != nil {
		return nil, ErrStore
	}
	defer func() {
		if err := cur.Close(ctx); err != nil {
			log.Printf("imleç kapatılamadı: %v", err)
		}
	}()
	var out []AuditRecord
	if err := cur.All(ctx, &out); err != nil {
		return nil, ErrStore
	}
	if out == nil {
		out = []AuditRecord{}
	}
	return out, nil
}

func (r *AuditRepository) CountByUser(ctx context.Context, userID bson.ObjectID) (int64, error) {
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

func (r *AuditRepository) CountDeleted(ctx context.Context) (int64, error) {
	if !r.ready() {
		return 0, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	n, err := r.col.CountDocuments(ctx, bson.M{"kullaniciId": UserDeleted})
	if err != nil {
		return 0, ErrStore
	}
	return n, nil
}

func (r *AuditRepository) ListDeleted(ctx context.Context) ([]AuditRecord, error) {
	if !r.ready() {
		return nil, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	cur, err := r.col.Find(ctx, bson.M{"kullaniciId": UserDeleted},
		options.Find().SetSort(bson.D{{Key: "zaman", Value: -1}}))
	if err != nil {
		return nil, ErrStore
	}
	defer func() { _ = cur.Close(ctx) }()
	var out []AuditRecord
	if err := cur.All(ctx, &out); err != nil {
		return nil, ErrStore
	}
	if out == nil {
		out = []AuditRecord{}
	}
	return out, nil
}
