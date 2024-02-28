package middleware

import (
	"compress/gzip"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	h "github.com/aleks0ps/url-shortener/internal/app/handler"
	m "github.com/go-chi/chi/middleware"
	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
)

type responseData struct {
	status int
	size   int
}

type loggingResponseWriter struct {
	http.ResponseWriter // встраиваем оригинальный http.ResponseWriter
	responseData        *responseData
}

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	// записываем ответ, используя оригинальный http.ResponseWriter
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size // захватываем размер
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	// записываем код статуса, используя оригинальный http.ResponseWriter
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode // захватываем код статуса
}

type DummyIO int

func (e DummyIO) Write(p []byte) (int, error) {
	e += DummyIO(len(p))
	return len(p), nil
}

func DisableDefaultLogger() func(next http.Handler) http.Handler {
	var dummy DummyIO
	dummyLogFormatter := m.DefaultLogFormatter{Logger: log.New(dummy, "", log.LstdFlags), NoColor: true}
	dummyLogger := m.RequestLogger(&dummyLogFormatter)
	return dummyLogger
}

type Claims struct {
	ID string `json:"id"`
	jwt.RegisteredClaims
}

const IDLength = 10

var jwtKey = []byte("my_secret_key")

func genUniqID(length uint64) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	uniqID := make([]byte, length)
	for i := range uniqID {
		uniqID[i] = charset[r.Intn(len(charset))]
	}
	return string(uniqID)
}

func newToken(expirationTime time.Time) (string, error) {
	ID := genUniqID(IDLength)
	claims := &Claims{
		// generate uniq ID
		ID: ID,
		RegisteredClaims: jwt.RegisteredClaims{
			// In JWT, the expiry time is expressed as unix milliseconds
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func checkToken(tokenStr string) (bool, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (any, error) {
		return jwtKey, nil
	})
	if err != nil {
		return false, err
	}
	// expired
	if !token.Valid {
		return false, nil
	}
	return true, nil
}

func refreshToken(expirationTime time.Time, tokenStr string) (string, bool, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (any, error) {
		return jwtKey, nil
	})
	if err != nil {
		return "", false, err
	}
	claims.ExpiresAt = jwt.NewNumericDate(expirationTime)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		return "", false, err
	}
	return tokenString, true, nil
}

func CookieChecker() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fnCheckCookie := func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("token")
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}
			_, err = checkToken(cookie.Value)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return

			}
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fnCheckCookie)
	}
}

func CookieIssuer() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fnCookies := func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("token")
			if err != nil {
				// Add Cookie if does not exists
				if err == http.ErrNoCookie {
					expirationTime := time.Now().Add(5 * time.Minute)
					tokenString, err := newToken(expirationTime)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					http.SetCookie(w, &http.Cookie{
						Name:    "token",
						Value:   tokenString,
						Expires: expirationTime,
					})
					next.ServeHTTP(w, r)
					return
				}
			}
			valid, err := checkToken(cookie.Value)
			if err != nil {
				expirationTime := time.Now().Add(5 * time.Minute)
				tokenString, err := newToken(expirationTime)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				http.SetCookie(w, &http.Cookie{
					Name:    "token",
					Value:   tokenString,
					Expires: expirationTime,
				})

			}
			if !valid {
				expirationTime := time.Now().Add(5 * time.Minute)
				tokenString, _, err := refreshToken(expirationTime, cookie.Value)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				http.SetCookie(w, &http.Cookie{
					Name:    "token",
					Value:   tokenString,
					Expires: expirationTime,
				})
			}
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fnCookies)
	}
}

func Logger(s *zap.SugaredLogger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fnLog := func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			responseData := &responseData{
				status: 0,
				size:   0,
			}
			lw := loggingResponseWriter{
				ResponseWriter: w,
				responseData:   responseData,
			}

			next.ServeHTTP(&lw, r)

			duration := time.Since(start)

			s.Infoln(
				"uri", r.RequestURI,
				"method", r.Method,
				"status", responseData.status, // получаем перехваченный код статуса ответа
				"duration", duration,
				"size", responseData.size, // получаем перехваченный размер ответа
			)
		}

		return http.HandlerFunc(fnLog)
	}
}

type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w gzipWriter) Write(b []byte) (int, error) {
	// w.Writer будет отвечать за gzip-сжатие, поэтому пишем в него
	return w.Writer.Write(b)
}

func Gziper() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fnDec := func(w http.ResponseWriter, r *http.Request) {
			if !strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
				next.ServeHTTP(w, r)
				return
			}
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			var bReader io.ReadCloser = gz
			defer gz.Close()
			dR, err := http.NewRequestWithContext(r.Context(), r.Method, r.URL.String(), bReader)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			// Set the same content type
			dR.Header.Set("Content-Type", r.Header.Get("Content-Type"))
			if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
				typeCode := h.GetContentTypeCode(r.Header.Get("Content-Type"))
				switch typeCode {
				case h.PlainText, h.HTML:
					eW, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					// Pass decoded request
					// Return encoded respose
					next.ServeHTTP(gzipWriter{ResponseWriter: w, Writer: eW}, dR)
				default:
					next.ServeHTTP(w, dR)
				}
			} else {
				next.ServeHTTP(w, dR)
			}
		}
		return http.HandlerFunc(fnDec)
	}
}
