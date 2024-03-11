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
	var store storage.Storager
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()
	sugar := logger.Sugar()
	opts := config.ParseOptions()
	// init storage
	db := storage.NewDBStorage(ctx, opts.DatabaseDSN, sugar)
	if db != nil {
		store = db
	} else {
		// user memory if database is not available
		store = storage.NewMemStorage(ctx, opts.StoragePath, sugar)
	}
	rt := handler.Runtime{
		BaseURL:       opts.BaseURL,
		ListenAddress: opts.ListenAddr,
		DBURL:         opts.DatabaseDSN,
		URLs:          store,
		Logger:        sugar,
	}
	r := chi.NewRouter()
	r.Use(mw.DisableDefaultLogger())
	r.Use(mw.Logger(sugar))
	r.Use(mw.Gziper())
	r.Get("/ping", rt.PingDB)
	r.Get("/{id}", rt.GetOrigURL)
	r.Route("/api/user/urls", func(r chi.Router) {
		r.Get("/", rt.ListURLsJSON)
		r.Delete("/", rt.DeleteURLsJSON)
	})
	r.Post("/api/shorten", rt.ShortenURLJSON)
	r.Post("/api/shorten/batch", rt.ShortenURLBatchJSON)
	r.Post("/", rt.ShortenURL)
	http.ListenAndServe(rt.ListenAddress, r)
}
