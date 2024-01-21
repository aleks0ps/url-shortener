package handler

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aleks0ps/url-shortener/internal/app/storage"
)

type ContentType int

// Service runtime context
type Runtime struct {
	BaseURL       string
	ListenAddress string
	URLs          *storage.URLStorage
}

const (
	Unsupported ContentType = iota
	PlainText
	URLEncoded
)

type ContentTypes struct {
	name string
	code ContentType
}

var supportedTypes = []ContentTypes{
	{
		name: "text/plain",
		code: PlainText,
	},
	{
		name: "application/x-www-form-urlencoded",
		code: URLEncoded,
	},
}

func checkContentType(name string) ContentType {
	for _, t := range supportedTypes {
		if name == t.name {
			return t.code
		}
	}
	return Unsupported
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

// Send response to POST requests
func (rt *Runtime) ShortenURL(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	if checkContentType(contentType) == URLEncoded {
		r.ParseForm()
		origURL := strings.Join(r.PostForm["url"], "")
		// XXX
		if len(origURL) == 0 {
			w.WriteHeader(http.StatusBadRequest)
		}
		shortKey := generateShortKey()
		rt.URLs.StoreURL(shortKey, string(origURL))
		shortenedURL := fmt.Sprintf("%s/%s", rt.BaseURL, shortKey)
		// Return url
		w.Header().Set("Content-Type", "text/plain")
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
		w.Header().Set("Content-Type", "text/plain")
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
