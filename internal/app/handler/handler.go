package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aleks0ps/url-shortener/internal/app/storage"
	"github.com/jackc/pgx/v4"
)

type ContentType int

type Storager interface {
	Load(ctx context.Context, key string) (URL string, ok bool, err error)
	Store(ctx context.Context, key string, URL string) (origKey string, exist bool, err error)
	StoreBatch(ctx context.Context, URLs map[string]*storage.URLRecord) (map[string]*storage.URLRecord, bool, error)
	List(ctx context.Context) ([]*storage.URLRecord, error)
}

// Service runtime context
type Runtime struct {
	BaseURL       string
	ListenAddress string
	DBURL         string
	URLs          Storager
}

const (
	Unsupported ContentType = iota
	PlainText
	URLEncoded
	JSON
	JS
	CSS
	HTML
	XML
)

type ContentTypes struct {
	Name string
	Code ContentType
}

type RequestJSON struct {
	URL string `json:"url"`
}

type ResponseJSON struct {
	Result string `json:"result"`
}

type RequestJSONBatchItem struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type ResponseJSONBatchItem struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

type ResponseJSONRecord struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

var SupportedTypes = []ContentTypes{
	{
		Name: "text/plain",
		Code: PlainText,
	},
	{
		Name: "application/x-www-form-urlencoded",
		Code: URLEncoded,
	},
	{
		Name: "application/json",
		Code: JSON,
	},
	{
		Name: "application/javascript",
		Code: JS,
	},
	{
		Name: "text/css",
		Code: CSS,
	},
	{
		Name: "text/html",
		Code: HTML,
	},
	{
		Name: "text/xml",
		Code: XML,
	},
}

func GetContentTypeCode(name string) ContentType {
	for _, t := range SupportedTypes {
		if name == t.Name {
			return t.Code
		}
	}
	return Unsupported
}

func GetContentTypeName(code ContentType) string {
	for _, t := range SupportedTypes {
		if code == t.Code {
			return t.Name
		}
	}
	return "unsupported"
}

func generateShortKey() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const keyLength = 6

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	shortKey := make([]byte, keyLength)
	for i := range shortKey {
		shortKey[i] = charset[r.Intn(len(charset))]
	}
	return string(shortKey)
}

func (rt *Runtime) ListURLsJSON(w http.ResponseWriter, r *http.Request) {
	var recListJSON []ResponseJSONRecord
	recList, err := rt.URLs.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// No records
	if len(recList) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	for _, rec := range recList {
		shortURL := fmt.Sprintf("%s/%s", rt.BaseURL, rec.ShortKey)
		recJSON := ResponseJSONRecord{ShortURL: shortURL, OriginalURL: rec.OriginalURL}
		recListJSON = append(recListJSON, recJSON)
	}
	res, err := json.Marshal(recListJSON)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", GetContentTypeName(JSON))
	w.Header().Set("Content-Length", strconv.Itoa(len(res)))
	w.WriteHeader(http.StatusOK)
	w.Write(res)
	return
}

func (rt *Runtime) ShortenURLJSONBatch(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	if GetContentTypeCode(contentType) != JSON {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// JSON
	var reqJSONBatch []RequestJSONBatchItem
	var resJSONBatch []ResponseJSONBatchItem
	var buf bytes.Buffer
	_, err := buf.ReadFrom(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := json.Unmarshal(buf.Bytes(), &reqJSONBatch); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(reqJSONBatch) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	URLs := make(map[string]*storage.URLRecord)
	for _, req := range reqJSONBatch {
		URLrec := storage.URLRecord{ShortKey: generateShortKey(), OriginalURL: req.OriginalURL}
		URLs[req.CorrelationID] = &URLrec
	}
	origURLs, exist, _ := rt.URLs.StoreBatch(r.Context(), URLs)
	if exist {
		for id, URLrec := range origURLs {
			shortURL := fmt.Sprintf("%s/%s", rt.BaseURL, URLrec.ShortKey)
			resConflict := ResponseJSONBatchItem{CorrelationID: id, ShortURL: shortURL}
			resJSONBatch = append(resJSONBatch, resConflict)
		}
		res, err := json.Marshal(resJSONBatch)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", GetContentTypeName(JSON))
		w.Header().Set("Content-Length", strconv.Itoa(len(res)))
		w.WriteHeader(http.StatusConflict)
		w.Write(res)
		return
	}
	for id, URLrec := range URLs {
		shortURL := fmt.Sprintf("%s/%s", rt.BaseURL, URLrec.ShortKey)
		res := ResponseJSONBatchItem{CorrelationID: id, ShortURL: shortURL}
		resJSONBatch = append(resJSONBatch, res)
	}
	res, err := json.Marshal(resJSONBatch)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", GetContentTypeName(JSON))
	w.Header().Set("Content-Length", strconv.Itoa(len(res)))
	// 201
	w.WriteHeader(http.StatusCreated)
	w.Write(res)
}

func (rt *Runtime) ShortenURLJSON(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	if GetContentTypeCode(contentType) != JSON {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// JSON
	var reqJSON RequestJSON
	var resJSON ResponseJSON
	var buf bytes.Buffer
	_, err := buf.ReadFrom(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := json.Unmarshal(buf.Bytes(), &reqJSON); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(reqJSON.URL) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	shortKey := generateShortKey()
	uniqKey, exist, _ := rt.URLs.Store(r.Context(), shortKey, reqJSON.URL)
	if exist {
		// return uniq short key
		shortenedURL := fmt.Sprintf("%s/%s", rt.BaseURL, uniqKey)
		resJSON.Result = shortenedURL
		res, err := json.Marshal(resJSON)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", GetContentTypeName(JSON))
		w.Header().Set("Content-Length", strconv.Itoa(len(res)))
		w.WriteHeader(http.StatusConflict)
		w.Write(res)
		return
	}
	shortenedURL := fmt.Sprintf("%s/%s", rt.BaseURL, shortKey)
	resJSON.Result = shortenedURL
	res, err := json.Marshal(resJSON)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", GetContentTypeName(JSON))
	w.Header().Set("Content-Length", strconv.Itoa(len(res)))
	// 201
	w.WriteHeader(http.StatusCreated)
	w.Write(res)
}

// Send response to POST requests
func (rt *Runtime) ShortenURL(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	if GetContentTypeCode(contentType) == URLEncoded {
		r.ParseForm()
		origURL := strings.Join(r.PostForm["url"], "")
		if len(origURL) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		shortKey := generateShortKey()
		uniqKey, exist, _ := rt.URLs.Store(r.Context(), shortKey, string(origURL))
		if exist {
			shortenedURL := fmt.Sprintf("%s/%s", rt.BaseURL, uniqKey)
			w.Header().Set("Content-Type", GetContentTypeName(PlainText))
			w.Header().Set("Content-Length", strconv.Itoa(len(shortenedURL)))
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(shortenedURL))
			return
		}
		shortenedURL := fmt.Sprintf("%s/%s", rt.BaseURL, shortKey)
		// Return url
		w.Header().Set("Content-Type", GetContentTypeName(PlainText))
		w.Header().Set("Content-Length", strconv.Itoa(len(shortenedURL)))
		// 201
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(shortenedURL))
	} else {
		origURL, err := io.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}
		shortKey := generateShortKey()
		uniqKey, exist, _ := rt.URLs.Store(r.Context(), shortKey, string(origURL))
		if exist {
			shortenedURL := fmt.Sprintf("%s/%s", rt.BaseURL, uniqKey)
			w.Header().Set("Content-Type", GetContentTypeName(PlainText))
			w.Header().Set("Content-Length", strconv.Itoa(len(shortenedURL)))
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(shortenedURL))
			return
		}
		shortenedURL := fmt.Sprintf("%s/%s", rt.BaseURL, shortKey)
		// Return url
		w.Header().Set("Content-Type", GetContentTypeName(PlainText))
		w.Header().Set("Content-Length", strconv.Itoa(len(shortenedURL)))
		// 201
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(shortenedURL))
	}
}

func (rt *Runtime) GetOrigURL(w http.ResponseWriter, r *http.Request) {
	var origURL string
	var ok bool
	shortKey := r.URL.RequestURI()[1:]
	origURL, ok, _ = rt.URLs.Load(r.Context(), shortKey)
	if ok {
		http.Redirect(w, r, origURL, http.StatusTemporaryRedirect)
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (rt *Runtime) DBIsAlive(w http.ResponseWriter, r *http.Request) {
	conn, err := pgx.Connect(r.Context(), rt.DBURL)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	defer conn.Close(r.Context())
	w.WriteHeader(http.StatusOK)
}
