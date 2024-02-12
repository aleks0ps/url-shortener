package app

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/aleks0ps/url-shortener/cmd/shortener/config"
	"github.com/aleks0ps/url-shortener/internal/app/handler"
	mw "github.com/aleks0ps/url-shortener/internal/app/middleware"
	"github.com/aleks0ps/url-shortener/internal/app/storage"
)

func Run() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	opts := config.ParseOptions()
	rt := handler.Runtime{
		BaseURL:       opts.BaseURL,
		ListenAddress: opts.ListenAddr,
		DBURL:         opts.DatabaseDSN,
		URLs:          storage.NewURLStorage(opts.StoragePath),
		URLsDB:        storage.PGNewURLStorage(ctx, opts.DatabaseDSN),
	}
	rt.URLs.LoadFromFile()
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()
	sugar := logger.Sugar()
	r := chi.NewRouter()
	r.Use(mw.DisableDefaultLogger())
	r.Use(mw.Logger(sugar))
	r.Use(mw.Gziper())
	r.Get("/ping", rt.DBIsAlive)
	r.Get("/{id}", rt.GetOrigURL)
	r.Post("/api/shorten", rt.ShortenURLJSON)
	r.Post("/", rt.ShortenURL)
	http.ListenAndServe(rt.ListenAddress, r)
}
