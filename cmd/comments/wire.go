//go:build wireinject
// +build wireinject

package main

import (
	"context"

	"github.com/google/wire"
	"github.com/mi-raf/comment-project/internal/api"
	"github.com/mi-raf/comment-project/internal/database"
	"github.com/mi-raf/comment-project/internal/service"
)

func initApp(ctx context.Context, cfg *config) (a *api.API, closer func(), err error) {
	wire.Build(
		service.NewCommentChan,
		initPostRepositoryConfig,
		initCommentRepositoryConfig,
		initServiceConfig,
		database.NewPostRepositoryProvider,
		database.NewCommentRepositoryProvider,
		service.NewPostService,
		api.NewResolver,
		initApiConfig,
		api.NewApi,
	)
	return nil, nil, nil
}
