package handler

import (
	"github.com/aleks0ps/url-shortener/internal/app/storage"
	"go.uber.org/zap"
)

// Service runtime context
type Runtime struct {
	BaseURL       string
	ListenAddress string
	DBURL         string
	URLs          storage.Storager
	Logger        *zap.SugaredLogger
}

func (rt *Runtime) newShortURL(key string) string {
	return rt.BaseURL + "/" + key
}
