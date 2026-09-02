package service

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/oz-fatma/kontrata/backend/internal/auth"
	"github.com/oz-fatma/kontrata/backend/internal/repository"
)

type yetkiOp int

const (
	opSozlesmeOku yetkiOp = iota
	opSozlesmeYaz
	opSozlesmeSil
	opUyeGor
	opUyeYonet
	opOrgSil
)

type actor struct {
	user repository.Kullanici
}

func (a actor) rol() string {
	if a.user.Rol == "" {
		return repository.RolSahip
	}
	return a.user.Rol
}

func (a actor) orgID() bson.ObjectID {
	return a.user.OrganizasyonID
}

func (a actor) hasOrg() bool {
	return !a.user.OrganizasyonID.IsZero()
}

func (a actor) can(op yetkiOp) bool {
	switch a.rol() {
	case repository.RolSahip:
		return true
	case repository.RolYonetici:
		return op == opSozlesmeOku || op == opSozlesmeYaz || op == opUyeGor
	case repository.RolGoruntuleyici:
		return op == opSozlesmeOku
	default:
		return false
	}
}

func (a actor) sozlesmeFilter() bson.M {
	if a.hasOrg() {
		return bson.M{"organizasyonId": a.orgID()}
	}
	return bson.M{
		"kullaniciId": a.user.ID,
		"$or": bson.A{
			bson.M{"organizasyonId": bson.M{"$exists": false}},
			bson.M{"organizasyonId": bson.NilObjectID},
		},
	}
}

func (a actor) ownsSozlesme(doc *repository.Sozlesme) bool {
	if doc == nil {
		return false
	}
	if a.hasOrg() {
		return doc.OrganizasyonID == a.orgID()
	}
	return doc.KullaniciID == a.user.ID && doc.OrganizasyonID.IsZero()
}

func loadActor(ctx context.Context, users *repository.KullaniciRepository) (actor, error) {
	id, ok := auth.IdentityFrom(ctx)
	if !ok {
		return actor{}, auth.ErrUnauthorized
	}
	if users == nil {
		return actor{}, repository.ErrUnavailable
	}
	user, err := users.GetByID(ctx, id.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return actor{}, auth.ErrUnauthorized
		}
		return actor{}, err
	}
	if user.Rol == "" {
		user.Rol = repository.RolSahip
	}
	if user.HesapTipi == "" {
		user.HesapTipi = repository.HesapBireysel
	}
	return actor{user: *user}, nil
}

func (s *AuthService) actor(ctx context.Context) (actor, error) {
	return loadActor(ctx, s.users)
}

func (s *SozlesmeService) actor(ctx context.Context) (actor, error) {
	return loadActor(ctx, s.users)
}
