package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	mycookie "github.com/aleks0ps/url-shortener/internal/app/cookie"
	"github.com/aleks0ps/url-shortener/internal/app/storage"

	"github.com/jackc/pgx/v4"
	"go.uber.org/zap"
)

type ContentType int

// Service runtime context
type Runtime struct {
	BaseURL       string
	ListenAddress string
	DBURL         string
	URLs          storage.Storager
	Logger        *zap.SugaredLogger
}

const (
	None ContentType = iota
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
	return None
}

func GetContentTypeName(code ContentType) string {
	for _, t := range SupportedTypes {
		if code == t.Code {
			return t.Name
		}
	}
	return "none"
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

func newCookie(w *http.ResponseWriter) (map[string]string, error) {
	res := make(map[string]string)
	expirationTime := time.Now().Add(5 * time.Minute)
	tokenString, claims, err := mycookie.NewToken(expirationTime)
	if err != nil {
		return nil, err
	}
	http.SetCookie(*w, &http.Cookie{
		Name:    "token",
		Value:   tokenString,
		Expires: expirationTime,
	})
	http.SetCookie(*w, &http.Cookie{
		Name:    "id",
		Value:   claims.ID,
		Expires: expirationTime,
	})
	res["id"] = (*claims).ID
	res["token"] = tokenString
	return res, nil
}

func getCookie(r *http.Request, name string) (string, error) {
	cookie, err := r.Cookie(name)
	if err != nil {
		return "", err
	}
	return cookie.Value, nil
}

func ensureCookie(w *http.ResponseWriter, r *http.Request, name string) (string, error) {
	cookieValue, err := getCookie(r, name)
	if err != nil {
		// No cookie found
		// Generate new
		myCookies, err := newCookie(w)
		if err != nil {
			return "", err
		}
		cookieValue = myCookies[name]
	}
	// return cookie value
	return cookieValue, nil
}

func (rt *Runtime) newShortURL(key string) string {
	return rt.BaseURL + "/" + key
}

func writeResponse(w *http.ResponseWriter, t ContentType, status int, data []byte) {
	switch t {
	case None:
		(*w).WriteHeader(status)
		if data != nil {
			(*w).Write(data)
		}
	default:
		(*w).Header().Set("Content-Type", GetContentTypeName(t))
		(*w).Header().Set("Content-Length", strconv.Itoa(len(data)))
		(*w).WriteHeader(status)
		(*w).Write(data)
	}
}

func writeError(w *http.ResponseWriter, status int, err error) {
	http.Error(*w, err.Error(), status)
}

func (rt *Runtime) DeleteURLsJSON(w http.ResponseWriter, r *http.Request) {
	var buf bytes.Buffer
	var shortKeys []string
	var deletedURLs []*storage.URLRecord
	contentType := r.Header.Get("Content-Type")
	if GetContentTypeCode(contentType) != JSON {
		writeResponse(&w, None, http.StatusBadRequest, nil)
		return
	}
	userID, err := getCookie(r, "id")
	if err != nil {
		writeError(&w, http.StatusUnauthorized, err)
		return
	}
	// JSON
	_, err = buf.ReadFrom(r.Body)
	if err != nil {
		writeError(&w, http.StatusBadRequest, err)
		return
	}
	if err := json.Unmarshal(buf.Bytes(), &shortKeys); err != nil {
		writeError(&w, http.StatusBadRequest, err)
		return
	}
	if len(shortKeys) == 0 {
		writeResponse(&w, None, http.StatusBadRequest, nil)
		return
	}
	for _, key := range shortKeys {
		var rec storage.URLRecord
		rec.UserID = userID
		rec.ShortKey = key
		rec.DeletedFlag = true
		deletedURLs = append(deletedURLs, &rec)
	}
	// Delete selected URLs
	err = rt.URLs.Delete(r.Context(), deletedURLs)
	if err != nil {
		rt.Logger.Errorln(err)
	}
	writeResponse(&w, None, http.StatusAccepted, nil)
}

func (rt *Runtime) ListURLsJSON(w http.ResponseWriter, r *http.Request) {
	var recListJSON []ResponseJSONRecord
	// Check Cookie
	userID, err := getCookie(r, "id")
	if err != nil {
		writeError(&w, http.StatusUnauthorized, err)
		return
	}
	recList, err := rt.URLs.List(r.Context(), userID)
	if err != nil {
		writeError(&w, http.StatusInternalServerError, err)
		return
	}
	for _, rec := range recList {
		recJSON := ResponseJSONRecord{ShortURL: rt.newShortURL(rec.ShortKey), OriginalURL: rec.OriginalURL}
		recListJSON = append(recListJSON, recJSON)
	}
	res, err := json.Marshal(recListJSON)
	if err != nil {
		writeError(&w, http.StatusInternalServerError, err)
		return
	}
	writeResponse(&w, JSON, http.StatusOK, res)
}

func (rt *Runtime) ShortenURLJSONBatch(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	if GetContentTypeCode(contentType) != JSON {
		writeResponse(&w, None, http.StatusBadRequest, nil)
		return
	}
	// Issue cookie
	userID, err := ensureCookie(&w, r, "id")
	if err != nil {
		writeError(&w, http.StatusInternalServerError, err)
		return
	}
	// JSON
	var reqJSONBatch []RequestJSONBatchItem
	var resJSONBatch []ResponseJSONBatchItem
	var buf bytes.Buffer
	_, err = buf.ReadFrom(r.Body)
	if err != nil {
		writeError(&w, http.StatusBadRequest, err)
		return
	}
	if err := json.Unmarshal(buf.Bytes(), &reqJSONBatch); err != nil {
		writeError(&w, http.StatusBadRequest, err)
		return
	}
	if len(reqJSONBatch) == 0 {
		writeResponse(&w, None, http.StatusBadRequest, nil)
		return
	}
	URLs := make(map[string]*storage.URLRecord)
	for _, req := range reqJSONBatch {
		URLrec := storage.URLRecord{ShortKey: generateShortKey(), OriginalURL: req.OriginalURL, UserID: userID}
		URLs[req.CorrelationID] = &URLrec
	}
	origURLs, exist, _ := rt.URLs.StoreBatch(r.Context(), URLs)
	if exist {
		for id, URLrec := range origURLs {
			resConflict := ResponseJSONBatchItem{CorrelationID: id, ShortURL: rt.newShortURL(URLrec.ShortKey)}
			resJSONBatch = append(resJSONBatch, resConflict)
		}
		res, err := json.Marshal(resJSONBatch)
		if err != nil {
			writeError(&w, http.StatusInternalServerError, err)
			return
		}
		writeResponse(&w, JSON, http.StatusConflict, res)
		return
	}
	for id, URLrec := range URLs {
		res := ResponseJSONBatchItem{CorrelationID: id, ShortURL: rt.newShortURL(URLrec.ShortKey)}
		resJSONBatch = append(resJSONBatch, res)
	}
	res, err := json.Marshal(resJSONBatch)
	if err != nil {
		writeError(&w, http.StatusInternalServerError, err)
		return
	}
	writeResponse(&w, JSON, http.StatusCreated, res)
}

func (rt *Runtime) ShortenURLJSON(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	if GetContentTypeCode(contentType) != JSON {
		writeResponse(&w, None, http.StatusBadRequest, nil)
		return
	}
	// Issue cookie
	userID, err := ensureCookie(&w, r, "id")
	if err != nil {
		writeError(&w, http.StatusInternalServerError, err)
		return
	}
	// JSON
	var reqJSON RequestJSON
	var resJSON ResponseJSON
	var buf bytes.Buffer
	_, err = buf.ReadFrom(r.Body)
	if err != nil {
		writeError(&w, http.StatusBadRequest, err)
		return
	}
	if err := json.Unmarshal(buf.Bytes(), &reqJSON); err != nil {
		writeError(&w, http.StatusBadRequest, err)
		return
	}
	if len(reqJSON.URL) == 0 {
		writeResponse(&w, None, http.StatusBadRequest, nil)
		return
	}
	shortKey := generateShortKey()
	origRec, exist, _ := rt.URLs.Store(r.Context(), &storage.URLRecord{ShortKey: shortKey, OriginalURL: reqJSON.URL, UserID: userID})
	if exist {
		// return uniq short key
		resJSON.Result = rt.newShortURL(origRec.ShortKey)
		res, err := json.Marshal(resJSON)
		if err != nil {
			writeError(&w, http.StatusInternalServerError, err)
			return
		}
		writeResponse(&w, JSON, http.StatusConflict, res)
		return
	}
	resJSON.Result = rt.newShortURL(shortKey)
	res, err := json.Marshal(resJSON)
	if err != nil {
		writeError(&w, http.StatusInternalServerError, err)
		return
	}
	writeResponse(&w, JSON, http.StatusCreated, res)
}

// Send response to POST requests
func (rt *Runtime) ShortenURL(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	// Issue cookie
	userID, err := ensureCookie(&w, r, "id")
	if err != nil {
		writeError(&w, http.StatusInternalServerError, err)
		return
	}
	if GetContentTypeCode(contentType) == URLEncoded {
		r.ParseForm()
		origURL := strings.Join(r.PostForm["url"], "")
		if len(origURL) == 0 {
			writeResponse(&w, None, http.StatusBadRequest, nil)
			return
		}
		shortKey := generateShortKey()
		res, exist, _ := rt.URLs.Store(r.Context(), &storage.URLRecord{ShortKey: shortKey, OriginalURL: string(origURL), UserID: userID})
		if exist {
			shortenedURL := rt.newShortURL(res.ShortKey)
			writeResponse(&w, PlainText, http.StatusConflict, []byte(shortenedURL))
			return
		}
		shortenedURL := rt.newShortURL(shortKey)
		writeResponse(&w, PlainText, http.StatusCreated, []byte(shortenedURL))
	} else {
		origURL, err := io.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}
		shortKey := generateShortKey()
		res, exist, _ := rt.URLs.Store(r.Context(), &storage.URLRecord{ShortKey: shortKey, OriginalURL: string(origURL), UserID: userID})
		if exist {
			shortenedURL := rt.newShortURL(res.ShortKey)
			writeResponse(&w, PlainText, http.StatusConflict, []byte(shortenedURL))
			return
		}
		shortenedURL := rt.newShortURL(shortKey)
		writeResponse(&w, PlainText, http.StatusCreated, []byte(shortenedURL))
	}
}

func (rt *Runtime) GetOrigURL(w http.ResponseWriter, r *http.Request) {
	shortKey := r.URL.RequestURI()[1:]
	origRec, ok, _ := rt.URLs.Load(r.Context(), shortKey)
	if ok {
		if origRec.DeletedFlag {
			writeResponse(&w, None, http.StatusGone, nil)
			return
		}
		http.Redirect(w, r, origRec.OriginalURL, http.StatusTemporaryRedirect)
	} else {
		writeResponse(&w, None, http.StatusBadRequest, nil)
	}
}

func (rt *Runtime) DBIsAlive(w http.ResponseWriter, r *http.Request) {
	conn, err := pgx.Connect(r.Context(), rt.DBURL)
	if err != nil {
		writeResponse(&w, None, http.StatusInternalServerError, []byte(err.Error()))
		return
	}
	defer conn.Close(r.Context())
	writeResponse(&w, None, http.StatusOK, nil)
}
