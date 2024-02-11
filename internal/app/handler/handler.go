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

// Service runtime context
type Runtime struct {
	BaseURL       string
	ListenAddress string
	URLs          *storage.URLStorage
	DbURL         string
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

func (rt *Runtime) ShortenURLJSON(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	if GetContentTypeCode(contentType) == JSON {
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
		rt.URLs.StoreURL(shortKey, reqJSON.URL)
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
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
}

// Send response to POST requests
func (rt *Runtime) ShortenURL(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	if GetContentTypeCode(contentType) == URLEncoded {
		r.ParseForm()
		origURL := strings.Join(r.PostForm["url"], "")
		// XXX
		if len(origURL) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		shortKey := generateShortKey()
		rt.URLs.StoreURL(shortKey, string(origURL))
		shortenedURL := fmt.Sprintf("%s/%s", rt.BaseURL, shortKey)
		// Return url
		w.Header().Set("Content-Type", GetContentTypeName(PlainText))
		w.Header().Set("Content-Length", strconv.Itoa(len(shortenedURL)))
		// 201
		w.WriteHeader(http.StatusCreated)
		//
		fmt.Fprint(w, shortenedURL)
	} else {
		origURL, err := io.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}
		shortKey := generateShortKey()
		rt.URLs.StoreURL(shortKey, string(origURL))
		shortenedURL := fmt.Sprintf("%s/%s", rt.BaseURL, shortKey)
		// Return url
		w.Header().Set("Content-Type", GetContentTypeName(PlainText))
		w.Header().Set("Content-Length", strconv.Itoa(len(shortenedURL)))
		// 201
		w.WriteHeader(http.StatusCreated)
		//
		fmt.Fprint(w, shortenedURL)
	}
}

func (rt *Runtime) GetOrigURL(w http.ResponseWriter, r *http.Request) {
	shortKey := r.URL.RequestURI()[1:]
	origURL, ok := rt.URLs.GetURL(shortKey)
	if ok {
		http.Redirect(w, r, origURL, http.StatusTemporaryRedirect)
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (rt *Runtime) DbIsAlive(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, rt.DbURL)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	defer conn.Close(ctx)
	w.WriteHeader(http.StatusOK)
}
