package api

//go:generate go run github.com/99designs/gqlgen generate

import "github.com/mi-raf/comment-project/internal/service"

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	ps *service.PostService
}

func NewResolver(ps *service.PostService) *Resolver {
	return &Resolver{ps: ps}
}
