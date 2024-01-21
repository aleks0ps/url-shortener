package app

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/aleks0ps/url-shortener/cmd/shortener/config"
	"github.com/aleks0ps/url-shortener/internal/app/handler"
	"github.com/aleks0ps/url-shortener/internal/app/storage"
)

func ignoreFavicon(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RequestURI() == "/favicon.ico" {
			http.Error(w, http.StatusText(404), 404)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func Run() {
	opts := config.ParseOptions()
	rt := handler.Runtime{
		BaseURL:       opts.BaseURL,
		ListenAddress: opts.ListenAddr,
		URLs:          storage.NewURLStorage(),
	}
	r := chi.NewRouter()
	r.Route("/{id}", func(r chi.Router) {
		r.Use(ignoreFavicon)
		r.Get("/", rt.GetOrigURL)
	})
	r.Post("/", rt.ShortenURL)
	http.ListenAndServe(rt.ListenAddress, r)
}
