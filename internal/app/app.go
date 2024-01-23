package app

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/aleks0ps/url-shortener/cmd/shortener/config"
	"github.com/aleks0ps/url-shortener/internal/app/handler"
	"github.com/aleks0ps/url-shortener/internal/app/storage"
)

func Run() {
	opts := config.ParseOptions()
	rt := handler.Runtime{
		BaseURL:       opts.BaseURL,
		ListenAddress: opts.ListenAddr,
		URLs:          storage.NewURLStorage(),
	}
	r := chi.NewRouter()
	r.Get("/{id}", rt.GetOrigURL)
	r.Post("/", rt.ShortenURL)
	http.ListenAndServe(rt.ListenAddress, r)
}
