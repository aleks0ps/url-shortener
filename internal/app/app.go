package app

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/aleks0ps/url-shortener/cmd/shortener/config"
	"github.com/aleks0ps/url-shortener/internal/app/handler"
)

func Run() {
	config.ParseOptions()
	handler.SetBaseURL(config.Options.BaseURL)
	r := chi.NewRouter()
	r.Get("/{id}", handler.GetOrigURL)
	r.Post("/", handler.ShortenURL)
	http.ListenAndServe(config.Options.ListenAddr, r)
}
