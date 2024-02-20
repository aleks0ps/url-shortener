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

func newStorage(ctx context.Context, opts *config.Config, logger *zap.SugaredLogger) handler.Storager {
	db := storage.PGNewURLStorage(ctx, opts.DatabaseDSN, logger)
	if db.IsReady() {
		return db
	}
	// store urls in memory if db is not available
	mem := storage.NewURLStorage(opts.StoragePath, logger)
	mem.LoadFromFile(ctx)
	return mem
}

func Run() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()
	sugar := logger.Sugar()
	opts := config.ParseOptions()
	storage := newStorage(ctx, opts, sugar)
	rt := handler.Runtime{
		BaseURL:       opts.BaseURL,
		ListenAddress: opts.ListenAddr,
		DBURL:         opts.DatabaseDSN,
		URLs:          storage,
	}
	r := chi.NewRouter()
	r.Use(mw.DisableDefaultLogger())
	r.Use(mw.Logger(sugar))
	r.Use(mw.Gziper())
	r.Get("/ping", rt.DBIsAlive)
	r.Get("/{id}", rt.GetOrigURL)
	r.Post("/api/shorten", rt.ShortenURLJSON)
	r.Post("/api/shorten/batch", rt.ShortenURLJSONBatch)
	r.Post("/", rt.ShortenURL)
	http.ListenAndServe(rt.ListenAddress, r)
}
