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

func newDBStorage(ctx context.Context, opts *config.Config, logger *zap.SugaredLogger) *storage.PGURLStorage {
	if len(opts.DatabaseDSN) == 0 {
		logger.Warnln("DATABASE_DSN is empty")
		return nil
	}
	db, err := storage.PGNewURLStorage(ctx, opts.DatabaseDSN, logger)
	if err != nil {
		logger.Errorln(err)
		return nil
	}
	return db
}

func newMemStorage(ctx context.Context, opts *config.Config, logger *zap.SugaredLogger) *storage.URLStorage {
	mem := storage.NewURLStorage(opts.StoragePath, logger)
	mem.LoadFromFile(ctx)
	return mem
}

func Run() {
	var storage handler.Storager
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
	db := newDBStorage(ctx, opts, sugar)
	if db != nil {
		storage = db
	} else {
		storage = newMemStorage(ctx, opts, sugar)
	}
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
