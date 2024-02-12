package config

import (
	"flag"
	"fmt"

	"github.com/caarlos0/env/v6"
)

const (
	defaultAddress     = "localhost:8080"
	defaultBaseURL     = "http://localhost:8080"
	defaultStoragePath = "/tmp/short-url-db.json"
	defaultDatabaseDSN = "postgres://url-shortener:url-shortener@localhost:5432/url-shortener"
)

type Config struct {
	ListenAddr  string `env:"SERVER_ADDRESS"`
	BaseURL     string `env:"BASE_URL"`
	StoragePath string `env:"FILE_STORAGE_PATH"`
	DatabaseDSN string `env:"DATABASE_DSN"`
}

func ParseOptions() *Config {
	opts := Config{
		ListenAddr:  defaultAddress,
		BaseURL:     defaultBaseURL,
		StoragePath: "",
		DatabaseDSN: "",
	}
	if err := env.Parse(&opts); err != nil {
		fmt.Println("failed:", err)
	}
	flag.StringVar(&opts.ListenAddr, "a", opts.ListenAddr, "Listen address:port")
	flag.StringVar(&opts.BaseURL, "b", opts.BaseURL, "Base URL for shortened url")
	flag.StringVar(&opts.StoragePath, "f", opts.StoragePath, "Path to URL storage on disk (default: /tmp/short-url-db.json)")
	flag.StringVar(&opts.DatabaseDSN, "d", opts.DatabaseDSN, "Postgres connection string")
	flag.Parse()
	return &opts
}
