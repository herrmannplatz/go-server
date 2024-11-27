package config

import (
	"github.com/caarlos0/env"
)

type Config struct {
	DatabaseUrl string `env:"DATABASE_URL,required"`
	PORT        string `env:"PORT,required"`
	JWTSecret   string `env:"JWT_SECRET,required"`
}

func GetConfig() (Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
