package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/xlab/closer"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

var (
	storage bool
)

func init() {
	flag.BoolVar(&storage, "memory", false, "Save data in memory")
}

func main() {

	flag.Parse()
	if storage == true {
		log.Info().Msg("save data in memory")
	} else {
		log.Info().Msg("save data in postgresql")
	}

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
	closer.Bind(func() {
		if err := a.Close(); err != nil {
			log.Error().Err(err).Msg("Can't stop web application")
		}
	})
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

// func initHttpClientConfiguration(cfg *config) *swagger.Configuration {
// 	return &swagger.Configuration{

// 		BasePath:      cfg.CarApiBasePath,
// 		DefaultHeader: make(map[string]string),
// 	}
// }

// func initValidator() *validator.Validate {
// 	validate := validator.New(validator.WithRequiredStructEnabled())
// 	validate.RegisterValidation("c-year", internal.LessThanCurrYearValidator)
// 	return validate
// }

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

func initPostgresConnection(ctx context.Context, cfg *config) (*pgxpool.Pool, func(), error) {
	pg, err := pgxpool.New(ctx, cfg.DbAddr)
	if err != nil {
		return nil, nil, err
	}
	err = pg.Ping(ctx)
	if err != nil {
		return nil, nil, err
	}

	return pg, pg.Close, nil
}

func initApiConfig(cfg *config) *api.Config {
	return &api.Config{Addr: cfg.Listen}
}
