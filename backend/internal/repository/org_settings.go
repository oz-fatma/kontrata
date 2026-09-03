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

const orgSettingsCollection = "ayarlar"

const (
	DefaultAuditorRiskThreshold = 0.75
	DefaultMaxToken             = 600
)

// OrgSettings organizasyonun çalışma zamanı LLM ayarlarıdır.
type OrgSettings struct {
	ID            bson.ObjectID `bson:"_id,omitempty"`
	OrgID         bson.ObjectID `bson:"organizasyonId"`
	RiskThreshold float64       `bson:"denetciRiskEsigi"`
	MaxToken      int32         `bson:"maxToken"`
	UpdatedAt     time.Time     `bson:"guncellemeTarihi"`
	UpdatedBy     bson.ObjectID `bson:"guncelleyenKullaniciId,omitempty"`
}

// DefaultOrgSettings kod varsayılanlarını döner. İlk okumada Settings belgeyi yazar.
func DefaultOrgSettings(orgID bson.ObjectID) OrgSettings {
	return OrgSettings{
		OrgID:         orgID,
		RiskThreshold: DefaultAuditorRiskThreshold,
		MaxToken:      DefaultMaxToken,
		UpdatedAt:     time.Now().UTC(),
	}
}

// OrgSettingsRepository ayarlar koleksiyonuna erişir.
type OrgSettingsRepository struct {
	col *mongo.Collection
}

func NewOrgSettingsRepository(client *appmongo.Client) *OrgSettingsRepository {
	if client == nil {
		return &OrgSettingsRepository{}
	}
	return &OrgSettingsRepository{col: client.Collection(appmongo.DatabaseName(), orgSettingsCollection)}
}

func (r *OrgSettingsRepository) ready() bool {
	return r != nil && r.col != nil
}

func (r *OrgSettingsRepository) EnsureIndexes(ctx context.Context) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	_, err := r.col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "organizasyonId", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	})
	if err != nil {
		return ErrStore
	}
	return nil
}

func (r *OrgSettingsRepository) GetByOrg(ctx context.Context, orgID bson.ObjectID) (*OrgSettings, error) {
	if !r.ready() {
		return nil, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	var doc OrgSettings
	err := r.col.FindOne(ctx, bson.M{"organizasyonId": orgID}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, ErrStore
	}
	return &doc, nil
}

func (r *OrgSettingsRepository) Upsert(ctx context.Context, doc *OrgSettings) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	opts := options.UpdateOne().SetUpsert(true)
	set := bson.M{
		"organizasyonId":         doc.OrgID,
		"denetciRiskEsigi":       doc.RiskThreshold,
		"maxToken":               doc.MaxToken,
		"guncellemeTarihi":       doc.UpdatedAt,
		"guncelleyenKullaniciId": doc.UpdatedBy,
	}
	res, err := r.col.UpdateOne(ctx, bson.M{"organizasyonId": doc.OrgID}, bson.M{"$set": set}, opts)
	if err != nil {
		return ErrStore
	}
	if res.UpsertedID != nil {
		if oid, ok := res.UpsertedID.(bson.ObjectID); ok {
			doc.ID = oid
		}
	}
	return nil
}

func (r *OrgSettingsRepository) DeleteByOrg(ctx context.Context, orgID bson.ObjectID) error {
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
