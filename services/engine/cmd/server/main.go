package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"credora/internal/assessment"
	apihttp "credora/internal/http"
	"credora/internal/policy"
	"credora/internal/repository"
)

func main() {
	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "8080"
	}

	logger := slog.Default()

	// Open database connection (shared by both repositories)
	db, err := openDatabase()
	if err != nil {
		logger.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Create repositories
	assessmentRepo, policyRepo := createRepositories(db)

	// Create policy registry and register default policies
	registry := policy.NewRegistry()
	if err := policy.RegisterDefaults(registry); err != nil {
		logger.Error("failed to register default policies", "error", err)
		os.Exit(1)
	}

	// Seed policy metadata to database
	if err := policy.SeedPolicies(context.Background(), registry, policyRepo, logger); err != nil {
		logger.Error("failed to seed policies", "error", err)
		os.Exit(1)
	}

	// Application service
	svc := assessment.NewService(assessmentRepo, registry, logger)

	// HTTP handler
	handler := apihttp.NewHandler(svc, policyRepo, logger)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	handler.RegisterRoutes(mux)
	apihttp.RegisterDocsRoutes(mux)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	logger.Info("credora engine listening", "addr", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func openDatabase() (*sql.DB, error) {
	driver := os.Getenv("DATABASE_DRIVER")
	dsn := os.Getenv("DATABASE_URL")

	switch driver {
	case "postgres", "postgresql":
		if dsn == "" {
			dsn = "postgres://credora:credora@localhost:5432/credora"
		}
		db, err := repository.ConnectPostgres(dsn)
		if err != nil {
			return nil, fmt.Errorf("postgres connection failed: %w", err)
		}
		slog.Default().Info("connected to PostgreSQL")
		return db, nil

	case "sqlite", "":
		if dsn == "" {
			dsn = "./data/credora.db"
		}
		if err := os.MkdirAll(filepath.Dir(dsn), 0755); err != nil {
			return nil, fmt.Errorf("create data directory: %w", err)
		}
		db, err := repository.ConnectSQLite(dsn)
		if err != nil {
			return nil, fmt.Errorf("sqlite connection failed: %w", err)
		}
		slog.Default().Info("connected to SQLite", "path", dsn)
		return db, nil

	default:
		return nil, fmt.Errorf("unsupported DATABASE_DRIVER: %s", driver)
	}
}

func createRepositories(db *sql.DB) (repository.AssessmentRepository, repository.PolicyRepository) {
	driver := os.Getenv("DATABASE_DRIVER")

	switch driver {
	case "postgres", "postgresql":
		return repository.NewPostgresRepository(db), repository.NewPostgresPolicyRepository(db)
	default:
		return repository.NewSQLiteRepository(db), repository.NewSQLitePolicyRepository(db)
	}
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
