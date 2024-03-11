package handler

import (
	"io"
	"net/http"
	"strings"

	mycookie "github.com/aleks0ps/url-shortener/internal/app/cookie"
	myhttp "github.com/aleks0ps/url-shortener/internal/app/http"
	"github.com/aleks0ps/url-shortener/internal/app/storage"
	"github.com/aleks0ps/url-shortener/internal/app/util"

	"github.com/jackc/pgx/v4"
)

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

func (rt *Runtime) PingDB(w http.ResponseWriter, r *http.Request) {
	conn, err := pgx.Connect(r.Context(), rt.DBURL)
	if err != nil {
		myhttp.WriteResponse(&w, myhttp.None, http.StatusInternalServerError, []byte(err.Error()))
		return
	}
	defer conn.Close(r.Context())
	myhttp.WriteResponse(&w, myhttp.None, http.StatusOK, nil)
}
