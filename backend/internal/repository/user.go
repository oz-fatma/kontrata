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

const userCollection = "kullanicilar"

const (
	StatusActive    = "AKTIF"
	StatusPending   = "BEKLEMEDE"
	StatusSuspended = "ASKIDA"

	AccountIndividual = "BIREYSEL"
	AccountCorporate  = "KURUMSAL"

	RoleOwner  = "SAHIP"
	RoleAdmin  = "YONETICI"
	RoleViewer = "GORUNTULEYICI"
)

// ErrDuplicate benzersiz indeks çakışmasıdır; GraphQL'e yansımaz.
var ErrDuplicate = errors.New("kayıt zaten var")

// User kullanicilar koleksiyonu belgesidir.
type User struct {
	ID             bson.ObjectID `bson:"_id,omitempty"`
	Email          string        `bson:"eposta"`
	PasswordHash   string        `bson:"sifreHash"`
	EmailVerified  bool          `bson:"epostaDogrulandi"`
	Status         string        `bson:"durum"`
	AccountType    string        `bson:"hesapTipi"`
	OrganizationID bson.ObjectID `bson:"organizasyonId,omitempty"`
	Role           string        `bson:"rol"`
	CreatedAt      time.Time     `bson:"olusturmaTarihi"`
	UpdatedAt      time.Time     `bson:"guncellemeTarihi"`
}

// UserRepository kullanicilar koleksiyonuna erişir.
type UserRepository struct {
	col *mongo.Collection
}

func NewUserRepository(client *appmongo.Client) *UserRepository {
	if client == nil {
		return &UserRepository{}
	}
	return &UserRepository{col: client.Collection(appmongo.DatabaseName(), userCollection)}
}

func (r *UserRepository) ready() bool {
	return r != nil && r.col != nil
}

// EnsureIndexes eposta için benzersiz indeks oluşturur.
func (r *UserRepository) EnsureIndexes(ctx context.Context) error {
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

func (r *UserRepository) Create(ctx context.Context, doc *User) error {
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

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	if !r.ready() {
		return nil, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	var doc User
	err := r.col.FindOne(ctx, bson.M{"eposta": strings.ToLower(strings.TrimSpace(email))}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, ErrStore
	}
	return &doc, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id bson.ObjectID) (*User, error) {
	if !r.ready() {
		return nil, ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	var doc User
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
func (r *UserRepository) MarkEmailVerified(ctx context.Context, id bson.ObjectID) error {
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
				bson.M{"$eq": bson.A{"$durum", StatusPending}},
				StatusActive,
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
func (r *UserRepository) UpdatePassword(ctx context.Context, id bson.ObjectID, passwordHash string) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	res, err := r.col.UpdateByID(ctx, id, bson.M{
		"$set": bson.M{
			"sifreHash":        passwordHash,
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

func (r *UserRepository) Delete(ctx context.Context, id bson.ObjectID) error {
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

// BackfillAccountFields eski belgelere bireysel / sahip yazar.
func (r *UserRepository) BackfillAccountFields(ctx context.Context) error {
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
	}, bson.M{"$set": bson.M{"hesapTipi": AccountIndividual}})
	if err != nil {
		return ErrStore
	}
	_, err = r.col.UpdateMany(ctx, bson.M{
		"$or": bson.A{
			bson.M{"rol": bson.M{"$exists": false}},
			bson.M{"rol": ""},
		},
	}, bson.M{"$set": bson.M{"rol": RoleOwner}})
	if err != nil {
		return ErrStore
	}
	return nil
}

func (r *UserRepository) SetOrganization(ctx context.Context, id, orgID bson.ObjectID, accountType, role string) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	res, err := r.col.UpdateByID(ctx, id, bson.M{"$set": bson.M{
		"organizasyonId":   orgID,
		"hesapTipi":        accountType,
		"rol":              role,
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

func (r *UserRepository) SetRole(ctx context.Context, id bson.ObjectID, role string) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	res, err := r.col.UpdateByID(ctx, id, bson.M{"$set": bson.M{
		"rol":              role,
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

func (r *UserRepository) ListByOrg(ctx context.Context, orgID bson.ObjectID) ([]User, error) {
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
	var out []User
	if err := cur.All(ctx, &out); err != nil {
		return nil, ErrStore
	}
	if out == nil {
		out = []User{}
	}
	return out, nil
}

func (r *UserRepository) CountByOrg(ctx context.Context, orgID bson.ObjectID) (int64, error) {
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

func (r *UserRepository) ListCorporate(ctx context.Context) ([]User, error) {
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
	var out []User
	if err := cur.All(ctx, &out); err != nil {
		return nil, ErrStore
	}
	if out == nil {
		out = []User{}
	}
	return out, nil
}

// DetachOrg üyeyi bireysel sahip yapar (organizasyon bağı kesilir).
func (r *UserRepository) DetachOrg(ctx context.Context, id bson.ObjectID) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	res, err := r.col.UpdateByID(ctx, id, bson.M{
		"$set": bson.M{
			"hesapTipi":        AccountIndividual,
			"rol":              RoleOwner,
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

func (r *UserRepository) DetachOrgByOrg(ctx context.Context, orgID bson.ObjectID) error {
	if !r.ready() {
		return ErrUnavailable
	}
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	_, err := r.col.UpdateMany(ctx, bson.M{"organizasyonId": orgID}, bson.M{
		"$set": bson.M{
			"hesapTipi":        AccountIndividual,
			"rol":              RoleOwner,
			"guncellemeTarihi": time.Now().UTC(),
		},
		"$unset": bson.M{"organizasyonId": ""},
	})
	if err != nil {
		return ErrStore
	}
	return nil
}
