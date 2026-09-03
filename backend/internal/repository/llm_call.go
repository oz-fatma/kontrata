package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	appmongo "github.com/oz-fatma/kontrata/backend/internal/mongo"
)

const llmCallCollection = "llm_cagrilari"

const llmCallTTLSeconds int32 = 90 * 24 * 60 * 60

// LLMCall bir model isteğinin izidir. Prompt ve çıktı yazılmaz.
type LLMCall struct {
	ID            bson.ObjectID `bson:"_id,omitempty"`
	OrgID         bson.ObjectID `bson:"organizasyonId,omitempty"`
	ContractID    bson.ObjectID `bson:"sozlesmeId,omitempty"`
	Agent         string        `bson:"agent"`
	Endpoint      string        `bson:"ucAdi"`
	Start         time.Time     `bson:"baslangic"`
	End           time.Time     `bson:"bitis"`
	DurationMs    int64         `bson:"sureMs"`
	InChars       int           `bson:"girisKarakter"`
	OutChars      int           `bson:"cikisKarakter"`
	Success       bool          `bson:"basarili"`
	ErrorType     string        `bson:"hataTipi"`
	Attempt       int32         `bson:"denemeNo"`
	PromptVersion *int32        `bson:"promptSurumu,omitempty"`
}

// LLMCallRepository llm_cagrilari koleksiyonuna erişir.
type LLMCallRepository struct {
	col *mongo.Collection
}

func NewLLMCallRepository(client *appmongo.Client) *LLMCallRepository {
	if client == nil {
		return &LLMCallRepository{}
	}
	return &LLMCallRepository{col: client.Collection(appmongo.DatabaseName(), llmCallCollection)}
}

func (r *LLMCallRepository) ready() bool {
	return r != nil && r.col != nil
}

func (r *LLMCallRepository) EnsureIndexes(ctx context.Context) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	_, err := r.col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "baslangic", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(llmCallTTLSeconds),
		},
		{
			Keys: bson.D{{Key: "organizasyonId", Value: 1}, {Key: "baslangic", Value: -1}},
		},
	})
	if err != nil {
		return ErrStore
	}
	return nil
}

func (r *LLMCallRepository) Insert(ctx context.Context, doc *LLMCall) error {
	if !r.ready() {
		return ErrUnavailable
	}
	if doc == nil {
		return nil
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	if doc.ID.IsZero() {
		doc.ID = bson.NewObjectID()
	}
	if _, err := r.col.InsertOne(ctx, doc); err != nil {
		return ErrStore
	}
	return nil
}

func (r *LLMCallRepository) ListSince(ctx context.Context, orgID bson.ObjectID, since time.Time) ([]LLMCall, error) {
	if !r.ready() {
		return nil, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	cur, err := r.col.Find(ctx, bson.M{
		"organizasyonId": orgID,
		"baslangic":      bson.M{"$gte": since},
	}, options.Find().SetSort(bson.D{{Key: "baslangic", Value: -1}}))
	if err != nil {
		return nil, ErrStore
	}
	defer func() { _ = cur.Close(ctx) }()
	var out []LLMCall
	if err := cur.All(ctx, &out); err != nil {
		return nil, ErrStore
	}
	if out == nil {
		out = []LLMCall{}
	}
	return out, nil
}

func (r *LLMCallRepository) ListRecent(ctx context.Context, orgID bson.ObjectID, limit int64) ([]LLMCall, error) {
	if !r.ready() {
		return nil, ErrUnavailable
	}
	if limit <= 0 {
		limit = 20
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	cur, err := r.col.Find(ctx, bson.M{"organizasyonId": orgID},
		options.Find().SetSort(bson.D{{Key: "baslangic", Value: -1}}).SetLimit(limit))
	if err != nil {
		return nil, ErrStore
	}
	defer func() { _ = cur.Close(ctx) }()
	var out []LLMCall
	if err := cur.All(ctx, &out); err != nil {
		return nil, ErrStore
	}
	if out == nil {
		out = []LLMCall{}
	}
	return out, nil
}
