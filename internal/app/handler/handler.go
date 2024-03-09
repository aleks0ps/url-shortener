package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/aleks0ps/url-shortener/internal/app/appjson"
	mycookie "github.com/aleks0ps/url-shortener/internal/app/cookie"
	myhttp "github.com/aleks0ps/url-shortener/internal/app/http"
	"github.com/aleks0ps/url-shortener/internal/app/storage"
	"github.com/aleks0ps/url-shortener/internal/app/util"

	"github.com/jackc/pgx/v4"
)

func (rt *Runtime) DeleteURLsJSON(w http.ResponseWriter, r *http.Request) {
	var buf bytes.Buffer
	var shortKeys []string
	var deletedURLs []*storage.URLRecord

	contentType := r.Header.Get("Content-Type")
	if myhttp.GetContentTypeCode(contentType) != myhttp.JSON {
		myhttp.WriteResponse(&w, myhttp.None, http.StatusBadRequest, nil)
		return
	}
	err := mycookie.ValidateCookie(r)
	if err != nil {
		myhttp.WriteError(&w, http.StatusUnauthorized, err)
		return
	}
	userID, _ := mycookie.GetCookie(r, "id")
	// JSON
	_, err = buf.ReadFrom(r.Body)
	if err != nil {
		myhttp.WriteError(&w, http.StatusBadRequest, err)
		return
	}
	if err := json.Unmarshal(buf.Bytes(), &shortKeys); err != nil {
		myhttp.WriteError(&w, http.StatusBadRequest, err)
		return
	}
	if len(shortKeys) == 0 {
		myhttp.WriteResponse(&w, myhttp.None, http.StatusBadRequest, nil)
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
	myhttp.WriteResponse(&w, myhttp.None, http.StatusAccepted, nil)
}

func (rt *Runtime) ListURLsJSON(w http.ResponseWriter, r *http.Request) {
	var recURLs []appjson.URLRecord
	// Check Cookie
	err := mycookie.ValidateCookie(r)
	if err != nil {
		myhttp.WriteError(&w, http.StatusUnauthorized, err)
		return
	}
	userID, _ := mycookie.GetCookie(r, "id")
	recList, err := rt.URLs.List(r.Context(), userID)
	if err != nil {
		myhttp.WriteError(&w, http.StatusInternalServerError, err)
		return
	}
	for _, rec := range recList {
		recJSON := appjson.URLRecord{ShortURL: rt.newShortURL(rec.ShortKey), OriginalURL: rec.OriginalURL}
		recURLs = append(recURLs, recJSON)
	}
	res, err := json.Marshal(recURLs)
	if err != nil {
		myhttp.WriteError(&w, http.StatusInternalServerError, err)
		return
	}
	myhttp.WriteResponse(&w, myhttp.JSON, http.StatusOK, res)
}

func (rt *Runtime) ShortenURLJSONBatch(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	if myhttp.GetContentTypeCode(contentType) != myhttp.JSON {
		myhttp.WriteResponse(&w, myhttp.None, http.StatusBadRequest, nil)
		return
	}
	// Issue cookie
	userID, err := mycookie.EnsureCookie(&w, r, "id")
	if err != nil {
		myhttp.WriteError(&w, http.StatusInternalServerError, err)
		return
	}
	// JSON
	var reqOriginalURLs []appjson.BatchOriginalURL
	var resShortURLs []appjson.BatchShortURL
	var buf bytes.Buffer
	_, err = buf.ReadFrom(r.Body)
	if err != nil {
		myhttp.WriteError(&w, http.StatusBadRequest, err)
		return
	}
	if err := json.Unmarshal(buf.Bytes(), &reqOriginalURLs); err != nil {
		myhttp.WriteError(&w, http.StatusBadRequest, err)
		return
	}
	if len(reqOriginalURLs) == 0 {
		myhttp.WriteResponse(&w, myhttp.None, http.StatusBadRequest, nil)
		return
	}
	URLs := make(map[string]*storage.URLRecord)
	for _, req := range reqOriginalURLs {
		URLrec := storage.URLRecord{ShortKey: util.GenerateShortKey(), OriginalURL: req.OriginalURL, UserID: userID}
		URLs[req.CorrelationID] = &URLrec
	}
	origURLs, exist, _ := rt.URLs.StoreBatch(r.Context(), URLs)
	if exist {
		for id, URLrec := range origURLs {
			resConflict := appjson.BatchShortURL{CorrelationID: id, ShortURL: rt.newShortURL(URLrec.ShortKey)}
			resShortURLs = append(resShortURLs, resConflict)
		}
		res, err := json.Marshal(resShortURLs)
		if err != nil {
			myhttp.WriteError(&w, http.StatusInternalServerError, err)
			return
		}
		myhttp.WriteResponse(&w, myhttp.JSON, http.StatusConflict, res)
		return
	}
	for id, URLrec := range URLs {
		res := appjson.BatchShortURL{CorrelationID: id, ShortURL: rt.newShortURL(URLrec.ShortKey)}
		resShortURLs = append(resShortURLs, res)
	}
	res, err := json.Marshal(resShortURLs)
	if err != nil {
		myhttp.WriteError(&w, http.StatusInternalServerError, err)
		return
	}
	myhttp.WriteResponse(&w, myhttp.JSON, http.StatusCreated, res)
}

func (rt *Runtime) ShortenURLJSON(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	if myhttp.GetContentTypeCode(contentType) != myhttp.JSON {
		myhttp.WriteResponse(&w, myhttp.None, http.StatusBadRequest, nil)
		return
	}
	// Issue cookie
	userID, err := mycookie.EnsureCookie(&w, r, "id")
	if err != nil {
		myhttp.WriteError(&w, http.StatusInternalServerError, err)
		return
	}
	// JSON
	var reqURL appjson.URL
	var resResult appjson.Result
	var buf bytes.Buffer
	_, err = buf.ReadFrom(r.Body)
	if err != nil {
		myhttp.WriteError(&w, http.StatusBadRequest, err)
		return
	}
	if err := json.Unmarshal(buf.Bytes(), &reqURL); err != nil {
		myhttp.WriteError(&w, http.StatusBadRequest, err)
		return
	}
	if len(reqURL.URL) == 0 {
		myhttp.WriteResponse(&w, myhttp.None, http.StatusBadRequest, nil)
		return
	}
	shortKey := util.GenerateShortKey()
	origRec, exist, _ := rt.URLs.Store(r.Context(), &storage.URLRecord{ShortKey: shortKey, OriginalURL: reqURL.URL, UserID: userID})
	if exist {
		// return uniq short key
		resResult.Result = rt.newShortURL(origRec.ShortKey)
		res, err := json.Marshal(resResult)
		if err != nil {
			myhttp.WriteError(&w, http.StatusInternalServerError, err)
			return
		}
		myhttp.WriteResponse(&w, myhttp.JSON, http.StatusConflict, res)
		return
	}
	resResult.Result = rt.newShortURL(shortKey)
	res, err := json.Marshal(resResult)
	if err != nil {
		myhttp.WriteError(&w, http.StatusInternalServerError, err)
		return
	}
	myhttp.WriteResponse(&w, myhttp.JSON, http.StatusCreated, res)
}

// Send response to POST requests
func (rt *Runtime) ShortenURL(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	// Issue cookie
	userID, err := mycookie.EnsureCookie(&w, r, "id")
	if err != nil {
		myhttp.WriteError(&w, http.StatusInternalServerError, err)
		return
	}
	if myhttp.GetContentTypeCode(contentType) == myhttp.URLEncoded {
		r.ParseForm()
		origURL := strings.Join(r.PostForm["url"], "")
		if len(origURL) == 0 {
			myhttp.WriteResponse(&w, myhttp.None, http.StatusBadRequest, nil)
			return
		}
		shortKey := util.GenerateShortKey()
		res, exist, _ := rt.URLs.Store(r.Context(), &storage.URLRecord{ShortKey: shortKey, OriginalURL: string(origURL), UserID: userID})
		if exist {
			shortenedURL := rt.newShortURL(res.ShortKey)
			myhttp.WriteResponse(&w, myhttp.PlainText, http.StatusConflict, []byte(shortenedURL))
			return
		}
		shortenedURL := rt.newShortURL(shortKey)
		myhttp.WriteResponse(&w, myhttp.PlainText, http.StatusCreated, []byte(shortenedURL))
	} else {
		origURL, err := io.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}
		shortKey := util.GenerateShortKey()
		res, exist, _ := rt.URLs.Store(r.Context(), &storage.URLRecord{ShortKey: shortKey, OriginalURL: string(origURL), UserID: userID})
		if exist {
			shortenedURL := rt.newShortURL(res.ShortKey)
			myhttp.WriteResponse(&w, myhttp.PlainText, http.StatusConflict, []byte(shortenedURL))
			return
		}
		shortenedURL := rt.newShortURL(shortKey)
		myhttp.WriteResponse(&w, myhttp.PlainText, http.StatusCreated, []byte(shortenedURL))
	}
}

func (rt *Runtime) GetOrigURL(w http.ResponseWriter, r *http.Request) {
	shortKey := r.URL.RequestURI()[1:]
	origRec, ok, _ := rt.URLs.Load(r.Context(), shortKey)
	if ok {
		if origRec.DeletedFlag {
			myhttp.WriteResponse(&w, myhttp.None, http.StatusGone, nil)
			return
		}
		http.Redirect(w, r, origRec.OriginalURL, http.StatusTemporaryRedirect)
	} else {
		myhttp.WriteResponse(&w, myhttp.None, http.StatusBadRequest, nil)
	}
}

func (rt *Runtime) DBIsAlive(w http.ResponseWriter, r *http.Request) {
	conn, err := pgx.Connect(r.Context(), rt.DBURL)
	if err != nil {
		myhttp.WriteResponse(&w, myhttp.None, http.StatusInternalServerError, []byte(err.Error()))
		return
	}
	defer conn.Close(r.Context())
	myhttp.WriteResponse(&w, myhttp.None, http.StatusOK, nil)
}
