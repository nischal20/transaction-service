package handler

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	handlerAccount "github.com/nischalpatel/transactions-api/internal/handler/account"
	handlerTransaction "github.com/nischalpatel/transactions-api/internal/handler/transaction"
	"github.com/nischalpatel/transactions-api/internal/utils"
	httpSwagger "github.com/swaggo/http-swagger"
)

const maxBodyBytes = 1 << 20 // 1 MB

// responseWriter wraps http.ResponseWriter to capture the status code for logging.
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// requestLogger assigns a unique request ID, sets the X-Request-ID response
// header, and logs method, path, status, and duration for every request.
func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := uuid.New().String()
		r = r.WithContext(utils.NewRequestID(r.Context(), id))
		w.Header().Set("X-Request-ID", id)

		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		log.Printf("[%s] %s %s %d %s", id, r.Method, r.URL.Path, rw.status, time.Since(start))
	})
}

func maxBodyLimit(next http.Handler) http.Handler {
	return http.MaxBytesHandler(next, maxBodyBytes)
}

// NewRouter wires all routes and returns the handler with logging and body limit.
func NewRouter(accounts *handlerAccount.AccountHandler, transactions *handlerTransaction.TransactionHandler) http.Handler {
	r := chi.NewRouter()

	r.Use(requestLogger)
	r.Use(maxBodyLimit)

	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		utils.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
	})
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		utils.WriteError(w, http.StatusNotFound, "not found")
	})

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r.Post("/accounts", accounts.CreateAccount)
	r.Get("/accounts/{accountId}", accounts.GetAccount)
	r.Post("/transactions", transactions.CreateTransaction)

	return r
}

// NewSwaggerHandler returns a handler that serves only the Swagger UI.
// Intended to run on a separate port so it can be disabled in production.
func NewSwaggerHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/swagger/", httpSwagger.WrapHandler)
	return mux
}
