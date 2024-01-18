package handler

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aleks0ps/url-service/internal/app/storage"
)

type ContentType int

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

// set this from config
var baseURL string

func checkContentType(name string) ContentType {
	for _, t := range supportedTypes {
		if name == t.name {
			return t.code
		}
	}
	return Unsupported
}

func GenerateShortKey() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const keyLength = 6

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	shortKey := make([]byte, keyLength)
	for i := range shortKey {
		shortKey[i] = charset[r.Intn(len(charset))]
	}
	return string(shortKey)
}

func emptyURL(url string) bool {
	return len(strings.TrimSpace(url)) == 0
}

func SetBaseURL(url string) {
	baseURL = url
}

// Send response to POST requests
func ShortenURL(w http.ResponseWriter, r *http.Request) {
	if emptyURL(baseURL) {
		panic("baseURL is no set")
	}
	if r.Method == http.MethodPost {
		contentType := r.Header.Get("Content-Type")
		if checkContentType(contentType) == URLEncoded {
			r.ParseForm()
			origURL := strings.Join(r.PostForm["url"], "")
			// XXX
			if len(origURL) == 0 {
				w.WriteHeader(http.StatusBadRequest)
			}
			shortKey := GenerateShortKey()
			storage.StoreURL(shortKey, string(origURL))
			shortenedURL := fmt.Sprintf("%s/%s", baseURL, shortKey)
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
			shortKey := GenerateShortKey()
			storage.StoreURL(shortKey, string(origURL))
			shortenedURL := fmt.Sprintf("%s/%s", baseURL, shortKey)
			// Return url
			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("Content-Length", strconv.Itoa(len(shortenedURL)))
			// 201
			w.WriteHeader(http.StatusCreated)
			//
			fmt.Fprint(w, shortenedURL)
		}
	}
}

func GetOrigURL(w http.ResponseWriter, r *http.Request) {
	if emptyURL(baseURL) {
		panic("baseURL is no set")
	}
	if r.Method == http.MethodGet {
		// ignore
		if r.URL.RequestURI() == "/favicon.ico" {
		} else {
			shortKey := r.URL.RequestURI()[1:]
			origURL, ok := storage.GetOrigURL(shortKey)
			if ok {
				http.Redirect(w, r, origURL, http.StatusTemporaryRedirect)
			} else {
				w.WriteHeader(http.StatusBadRequest)
			}
		}
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
}
