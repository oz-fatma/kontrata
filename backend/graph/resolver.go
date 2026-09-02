package graph

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require
// here.

import "github.com/oz-fatma/kontrata/backend/internal/service"

// Resolver GraphQL kök çözümleyicisidir.
type Resolver struct {
	Service *service.SozlesmeService
	Auth    *service.AuthService
}
