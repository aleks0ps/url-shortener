package middleware

import (
	"compress/gzip"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	myhttp "github.com/aleks0ps/url-shortener/internal/app/http"

	m "github.com/go-chi/chi/middleware"
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
				typeCode := myhttp.GetContentTypeCode(r.Header.Get("Content-Type"))
				switch typeCode {
				case myhttp.PlainText, myhttp.HTML:
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
