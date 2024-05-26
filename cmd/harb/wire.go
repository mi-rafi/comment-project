package main

import (
	"context"

	"github.com/google/wire"
)

func initApp(ctx context.Context, cfg *config) (a *api.API, closer func(), err error) {
	wire.Build(
		initApiConfig,
	)
	return nil, nil, nil
}
