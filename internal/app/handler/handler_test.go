package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-resty/resty/v2"

	"github.com/aleks0ps/url-shortener/internal/app/storage"

	"github.com/stretchr/testify/assert"
)

func TestShortenURL(t *testing.T) {
	contentType := "text/plain"
	storagePath := "/tmp/short-url-db.json"
	testCases := []struct {
		method       string
		body         string
		expectedCode int
		expectedBody string
	}{
		{method: http.MethodPost, body: "https://ya.ru", expectedCode: http.StatusCreated, expectedBody: ""},
		{method: http.MethodPost, body: "", expectedCode: http.StatusCreated, expectedBody: ""},
	}
	rt := Runtime{
		BaseURL:       "http://localhost:8080",
		ListenAddress: "",
		URLs:          storage.NewURLStorage(storagePath),
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
	contentType := "text/plain"
	storagePath := "/tmp/short-url-db.json"
	urls := []struct {
		key     string
		origURL string
	}{
		{key: "qsBVYP", origURL: "https://ya.ru"},
		{key: "35D0WW", origURL: "https://google.com"},
	}
	rt := Runtime{
		BaseURL:       "http://localhost:8080",
		ListenAddress: "",
		URLs:          storage.NewURLStorage(storagePath),
	}
	for _, url := range urls {
		rt.URLs.StoreURL(url.key, url.origURL)
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
