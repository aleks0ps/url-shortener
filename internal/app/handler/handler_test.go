package handler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"

	"github.com/aleks0ps/url-shortener/internal/app/storage"

	"github.com/stretchr/testify/assert"
)

func TestShortenURL(t *testing.T) {
	contentType := "text/plain"
	storagePath := "/tmp/short-url-db.json"
	databaseDSN := os.Getenv("DATABASE_DSN")
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
	storageURLs := storage.NewURLStorage(storagePath, sugar)
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
	databaseDSN := os.Getenv("DATABASE_DSN")
	urls := []storage.URLRecord{
		{ShortKey: "qsBVYP", OriginalURL: "https://ya.ru", UserID: "User1"},
		{ShortKey: "35D0WW", OriginalURL: "https://google.com", UserID: "User2"},
	}
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()
	sugar := logger.Sugar()
	storageURLs := storage.NewURLStorage(storagePath, sugar)
	rt := Runtime{
		BaseURL:       "http://localhost:8080",
		ListenAddress: "",
		DBURL:         databaseDSN,
		URLs:          storageURLs,
	}
	for _, url := range urls {
		_, _, err := rt.URLs.Store(ctx, &url)
		if err != nil {
			fmt.Println("STORE ERROR!!")
			fmt.Println(err)
		}
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
		{method: http.MethodGet, body: urls[0].ShortKey, expectedCode: http.StatusTemporaryRedirect, expectedBody: urls[0].OriginalURL},
		{method: http.MethodGet, body: urls[1].ShortKey, expectedCode: http.StatusTemporaryRedirect, expectedBody: urls[1].OriginalURL},
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
