package app

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/aleks0ps/url-shortener/cmd/shortener/config"
	"github.com/aleks0ps/url-shortener/internal/app/handler"
	mw "github.com/aleks0ps/url-shortener/internal/app/middleware"
	"github.com/aleks0ps/url-shortener/internal/app/storage"
)

func Run() {
	opts := config.ParseOptions()
	rt := handler.Runtime{
		BaseURL:       opts.BaseURL,
		ListenAddress: opts.ListenAddr,
		URLs:          storage.NewURLStorage(),
	}
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()
	sugar := logger.Sugar()
	r := chi.NewRouter()
	r.Use(mw.DisableDefaultLogger())
	r.Use(mw.Logger(sugar))
	r.Get("/{id}", rt.GetOrigURL)
	r.Post("/", rt.ShortenURL)
	http.ListenAndServe(rt.ListenAddress, r)
}
