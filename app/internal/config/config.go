package config

import (
	"errors"
	"os"
)

type Config struct {
	Port      string
	DBDSN     string
	RedisAddr string
	RateRPS   int
	RateBurst int
}

func Load() (Config, error) {
	cfg := Config{
		Port:      pick(os.Getenv("PORT"), "8095"),
		DBDSN:     os.Getenv("DB_DSN"),
		RedisAddr: pick(os.Getenv("REDIS_ADDR"), "localhost:6379"),
	}
	if cfg.DBDSN == "" {
		return cfg, errors.New("DB_DSN required")
	}

	cfg.RateRPS = 10
	cfg.RateBurst = 20
	return cfg, nil
}

func pick(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
