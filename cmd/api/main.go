// @title			Transactions Service
// @version			1.0
// @description		transactions service. Manage cardholder accounts and record financial operations.
// @host			localhost:8080
// @BasePath		/

package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nischalpatel/transactions-api/config"
	"github.com/nischalpatel/transactions-api/database"
	_ "github.com/nischalpatel/transactions-api/docs"
	"github.com/nischalpatel/transactions-api/internal/audit"
	"github.com/nischalpatel/transactions-api/internal/handler"
	handlerAccount "github.com/nischalpatel/transactions-api/internal/handler/account"
	handlerTransaction "github.com/nischalpatel/transactions-api/internal/handler/transaction"
	"github.com/nischalpatel/transactions-api/internal/repository"
	memaccount "github.com/nischalpatel/transactions-api/internal/repository/memory/account"
	memtransaction "github.com/nischalpatel/transactions-api/internal/repository/memory/transaction"
	pgaccount "github.com/nischalpatel/transactions-api/internal/repository/postgres/account"
	pgtransaction "github.com/nischalpatel/transactions-api/internal/repository/postgres/transaction"
	svcaccount "github.com/nischalpatel/transactions-api/internal/service/account"
	svctransaction "github.com/nischalpatel/transactions-api/internal/service/transaction"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	var (
		accRepo repository.AccountRepository
		txRepo  repository.TransactionRepository
		auditor audit.Logger
		db      *sql.DB // non-nil only in postgres mode; closed during shutdown
	)

	switch cfg.DBDriver {
	case config.DBDriverPostgres:
		db, err = database.NewPostgresConnector(database.PostgresConfig{
			Host:     cfg.Postgres.Host,
			Port:     cfg.Postgres.Port,
			User:     cfg.Postgres.User,
			Password: cfg.Postgres.Password,
			DBName:   cfg.Postgres.DBName,
			SSLMode:  cfg.Postgres.SSLMode,
		}).Connect()
		if err != nil {
			log.Fatalf("postgres connect: %v", err)
		}
		accRepo = pgaccount.NewAccountStore(db)
		txRepo = pgtransaction.NewTransactionStore(db)
		auditor = audit.NewPostgresLogger(db)
		log.Println("storage: PostgreSQL")

	default:
		accRepo = memaccount.NewAccountStore()
		txRepo = memtransaction.NewTransactionStore()
		auditor = audit.NoopLogger{}
		log.Println("storage: in-memory")
	}

	accSvc := svcaccount.NewAccountService(accRepo, auditor)
	txSvc := svctransaction.NewTransactionService(txRepo, accRepo, auditor)

	accHandler := handlerAccount.NewAccountHandler(accSvc)
	txHandler := handlerTransaction.NewTransactionHandler(txSvc)

	// ── API server ────────────────────────────────────────────────────────────
	apiSrv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      handler.NewRouter(accHandler, txHandler),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("API listening on :%s", cfg.Port)
		if err := apiSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("API server error: %v", err)
		}
	}()

	// ── Swagger UI server (separate port, disable in prod by setting SWAGGER_PORT="") ──
	var swaggerSrv *http.Server
	if cfg.SwaggerPort != "" {
		swaggerSrv = &http.Server{
			Addr:         fmt.Sprintf(":%s", cfg.SwaggerPort),
			Handler:      handler.NewSwaggerHandler(),
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  120 * time.Second,
		}
		go func() {
			log.Printf("Swagger UI listening on :%s  →  http://localhost:%s/swagger/index.html", cfg.SwaggerPort, cfg.SwaggerPort)
			if err := swaggerSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Printf("swagger server error: %v", err)
			}
		}()
	}

	// ── Graceful shutdown ─────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. Stop accepting new HTTP requests and drain in-flight ones.
	if err := apiSrv.Shutdown(ctx); err != nil {
		log.Fatalf("API forced shutdown: %v", err)
	}
	if swaggerSrv != nil {
		if err := swaggerSrv.Shutdown(ctx); err != nil {
			log.Printf("swagger forced shutdown: %v", err)
		}
	}

	// 2. Close the DB pool now that no handler can issue new queries.
	if db != nil {
		log.Println("closing database connection...")
		if err := db.Close(); err != nil {
			log.Printf("db close error: %v", err)
		}
		log.Println("database connection closed")
	}

	log.Println("server stopped")
}
