package service

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/oz-fatma/kontrata/backend/internal/auth"
	"github.com/oz-fatma/kontrata/backend/internal/repository"
)

type authzOp int

const (
	opContractRead authzOp = iota
	opContractWrite
	opContractApprove
	opContractDelete
	opMemberView
	opMemberManage
	opOrgDelete
)

type actor struct {
	user repository.User
}

func (a actor) role() string {
	if a.user.Role == "" {
		return repository.RoleOwner
	}
	return a.user.Role
}

func (a actor) orgID() bson.ObjectID {
	return a.user.OrganizationID
}

func (a actor) hasOrg() bool {
	return !a.user.OrganizationID.IsZero()
}

func (a actor) can(op authzOp) bool {
	switch a.role() {
	case repository.RoleOwner:
		return true
	case repository.RoleAdmin:
		return op == opContractRead || op == opContractWrite || op == opContractApprove || op == opMemberView
	case repository.RoleViewer:
		return op == opContractRead
	default:
		return false
	}
}

func (a actor) contractFilter() bson.M {
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

func (a actor) ownsContract(doc *repository.Contract) bool {
	if doc == nil {
		return false
	}
	if a.hasOrg() {
		return doc.OrganizationID == a.orgID()
	}
	return doc.UserID == a.user.ID && doc.OrganizationID.IsZero()
}

func loadActor(ctx context.Context, users *repository.UserRepository) (actor, error) {
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
	if user.Role == "" {
		user.Role = repository.RoleOwner
	}
	if user.AccountType == "" {
		user.AccountType = repository.AccountIndividual
	}
	return actor{user: *user}, nil
}

func (s *AuthService) actor(ctx context.Context) (actor, error) {
	return loadActor(ctx, s.users)
}

func (s *ContractService) actor(ctx context.Context) (actor, error) {
	return loadActor(ctx, s.users)
}
