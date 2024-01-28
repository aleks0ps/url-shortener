package config

import (
	"flag"
	"fmt"

	"github.com/caarlos0/env/v6"
)

const (
	defaultAddress = "localhost:8080"
	defaultBaseURL = "http://localhost:8080"
)

type Config struct {
	ListenAddr string `env:"SERVER_ADDRESS"`
	BaseURL    string `env:"BASE_URL"`
}

func ParseOptions() *Config {
	opts := Config{
		ListenAddr: defaultAddress,
		BaseURL:    defaultBaseURL,
	}
	if err := env.Parse(&opts); err != nil {
		fmt.Println("failed:", err)
	}
	flag.StringVar(&opts.ListenAddr, "a", opts.ListenAddr, "Listen address:port")
	flag.StringVar(&opts.BaseURL, "b", opts.BaseURL, "Base URL for shortened url")
	flag.Parse()
	return &opts
}
