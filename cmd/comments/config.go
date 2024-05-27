package main

import (
	"github.com/caarlos0/env/v11"
	_ "github.com/joho/godotenv/autoload"
)

type config struct {
	Listen             string `env:"LISTEN" envDefault:":9000"`
	LogLevel           string `env:"LOG_LEVEL" envDefault:"debug"`
	LogFmt             string `env:"LOG_FMT" envDefault:"console"`
	InMemory           bool   `env:"IN_MEMORY" envDefault:"false"`
	MigrationDirectory string `env:"MIGRATION_DIR" envDefault:"file://init/migrations"`
	DbAddr             string `env:"DB_HOST"`
}

func initConfig() (*config, error) {
	cfg := &config{}

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
