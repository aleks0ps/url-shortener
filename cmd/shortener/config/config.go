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

var Options Config = Config{
	ListenAddr: defaultAddress,
	BaseURL:    defaultBaseURL,
}

func ParseOptions() {
	if err := env.Parse(&Options); err != nil {
		fmt.Println("failed:", err)
	}
	flag.StringVar(&Options.ListenAddr, "a", Options.ListenAddr, "Listen address:port")
	flag.StringVar(&Options.BaseURL, "b", Options.BaseURL, "Base URL for shortened url")
	flag.Parse()
}
