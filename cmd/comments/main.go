package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mi-raf/comment-project/graph"
	"github.com/mi-raf/comment-project/internal/api"
	"github.com/mi-raf/comment-project/internal/database"
	"github.com/mi-raf/comment-project/internal/models"
	"github.com/mi-raf/comment-project/internal/service"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/xlab/closer"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {

	defer closer.Close()

	closer.Bind(func() {
		log.Info().Msg("shutdown")
	})

	cfg, err := initConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("Can't init config")
	}

	if err := initLogger(cfg); err != nil {
		log.Fatal().Err(err).Msg("Can't init logger")
	}

	mf, err := migrateData(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Can't init migration")
	}
	closer.Bind(mf)

	ctx, cancelCtx := context.WithCancel(context.Background())
	closer.Bind(cancelCtx)

	a, cleanup, err := initApp(ctx, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Can't init app")
	}
	closer.Bind(cleanup)
	closer.Bind(a.Close)
	if err := a.Start(); err != nil {
		log.Fatal().Err(err).Msg("Can't start app")
	}

}

func initLogger(c *config) error {
	log.Debug().Msg("init logger")
	logLvl, err := zerolog.ParseLevel(strings.ToLower(c.LogLevel))
	if err != nil {
		return err
	}
	zerolog.SetGlobalLevel(logLvl)
	switch c.LogFmt {
	case "console":
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	case "json":
	default:
		return fmt.Errorf("unknown output format %s", c.LogFmt)

	}
	return nil
}

func migrateData(cfg *config) (func(), error) {
	log.Debug().Msg("start migrating data")
	m, err := migrate.New(
		cfg.MigrationDirectory,
		cfg.DbAddr)
	if err != nil {
		return nil, err
	}

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		log.Error().Err(err).Msg("can not migrate data")
		return func() {
			if err, _ := m.Close(); err != nil {
				log.Error().Err(err).Msg("can not graceful stop migration")
			}
		}, err
	}

	v, _, err := m.Version()
	if err != nil {
		log.Error().Err(err).Msg("can not get migration version")
		return func() {
			if err, _ := m.Close(); err != nil {
				log.Error().Err(err).Msg("can not graceful stop migration")
			}
		}, err

	}
	log.Info().Uint("version", v).Msg("migration succesful")

	return func() {
		if err, _ := m.Close(); err != nil {
			log.Error().Err(err).Msg("can not graceful stop migration")
		}
	}, nil
}

func initApiConfig(cfg *config, res *api.Resolver) *api.Config {
	return &api.Config{Listen: cfg.Listen, GraphCfg: graph.Config{Resolvers: res}}
}

func initPostRepositoryConfig(cfg *config) *database.PostConfig {
	return &database.PostConfig{InMemory: cfg.InMemory, DbAddr: cfg.DbAddr}
}

func initCommentRepositoryConfig(cfg *config, ch chan models.CommentDTO) *database.CommentConfig {
	return &database.CommentConfig{InMemory: cfg.InMemory, DbAddr: cfg.DbAddr, CommentChan: ch}
}

func initServiceConfig(ch chan models.CommentDTO) *service.Config {
	return &service.Config{CommentChan: ch}
}
