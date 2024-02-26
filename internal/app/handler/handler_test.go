package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"

	"github.com/aleks0ps/url-shortener/internal/app/storage"

	"github.com/stretchr/testify/assert"
)

func TestShortenURL(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()
	contentType := "text/plain"
	storagePath := "/tmp/short-url-db.json"
	databaseDSN := ""
	testCases := []struct {
		method       string
		body         string
		expectedCode int
		expectedBody string
	}{
		{method: http.MethodPost, body: "https://ya.ru", expectedCode: http.StatusCreated, expectedBody: ""},
		{method: http.MethodPost, body: "", expectedCode: http.StatusCreated, expectedBody: ""},
	}
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()
	sugar := logger.Sugar()
	var storageURLs Storager
	db, err := storage.PGNewURLStorage(ctx, databaseDSN, sugar)
	if err != nil {
		storageURLs = db
	} else {
		storageURLs = storage.NewURLStorage(storagePath, sugar)
	}
	rt := Runtime{
		BaseURL:       "http://localhost:8080",
		ListenAddress: "",
		DBURL:         databaseDSN,
		URLs:          storageURLs,
	}
	handler := http.HandlerFunc(rt.ShortenURL)
	srv := httptest.NewServer(handler)
	defer srv.Close()
	for _, tc := range testCases {
		t.Run(tc.method, func(t *testing.T) {
			r := resty.New().R()
			r.Method = tc.method
			r.URL = srv.URL
			r.SetHeader("Content-Type", contentType)
			r.SetBody([]byte(tc.body))
			resp, err := r.Send()
			assert.NoError(t, err, "error making HTTP request")
			assert.Equal(t, tc.expectedCode, resp.StatusCode(), "Код ответа не совпадает с ожидаемым")
		})
	}
}

func TestGetOrigURL(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()
	contentType := "text/plain"
	storagePath := "/tmp/short-url-db.json"
	databaseDSN := ""
	urls := []struct {
		key     string
		origURL string
	}{
		{key: "qsBVYP", origURL: "https://ya.ru"},
		{key: "35D0WW", origURL: "https://google.com"},
	}
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()
	sugar := logger.Sugar()
	var storageURLs Storager
	db, err := storage.PGNewURLStorage(ctx, databaseDSN, sugar)
	if err != nil {
		storageURLs = db
	} else {
		storageURLs = storage.NewURLStorage(storagePath, sugar)
	}
	rt := Runtime{
		BaseURL:       "http://localhost:8080",
		ListenAddress: "",
		DBURL:         databaseDSN,
		URLs:          storageURLs,
	}
	for _, url := range urls {
		_, _, _ = rt.URLs.Store(ctx, url.key, url.origURL)
	}
	handler := http.HandlerFunc(rt.GetOrigURL)
	srv := httptest.NewServer(handler)
	defer srv.Close()
	testCases := []struct {
		method       string
		body         string
		expectedCode int
		expectedBody string
	}{
		{method: http.MethodGet, body: urls[0].key, expectedCode: http.StatusTemporaryRedirect, expectedBody: urls[0].origURL},
		{method: http.MethodGet, body: urls[1].key, expectedCode: http.StatusTemporaryRedirect, expectedBody: urls[1].origURL},
	}
	for _, tc := range testCases {
		t.Run(tc.method, func(t *testing.T) {
			r := resty.New().R()
			r.Method = tc.method
			r.URL = srv.URL + "/" + tc.body
			r.SetHeader("Content-Type", contentType)
			resp, err := r.Send()
			assert.NoError(t, err, "error making HTTP request")
			// return 200 instead of 30*
			assert.Equal(t, http.StatusOK, resp.StatusCode(), "Код ответа не совпадает с ожидаемым")
		})
	}
}
