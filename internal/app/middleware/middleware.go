package middleware

import (
	"log"
	"net/http"
	"time"

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
