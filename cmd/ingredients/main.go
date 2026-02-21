package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/lib/pq"
	"github.com/mwhite7112/woodpantry-ingredients/internal/api"
	"github.com/mwhite7112/woodpantry-ingredients/internal/db"
	"github.com/mwhite7112/woodpantry-ingredients/internal/logging"
	"github.com/mwhite7112/woodpantry-ingredients/internal/service"
)

func main() {
	logging.Setup()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		slog.Error("DB_URL is required")
		os.Exit(1)
	}

	threshold := 0.8
	if t := os.Getenv("RESOLVE_THRESHOLD"); t != "" {
		v, err := strconv.ParseFloat(t, 64)
		if err != nil {
			slog.Error("invalid RESOLVE_THRESHOLD", "error", err)
			os.Exit(1)
		}
		threshold = v
	}

	sqlDB, err := sql.Open("postgres", dbURL)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer sqlDB.Close()

	if err := sqlDB.Ping(); err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	if err := runMigrations(sqlDB); err != nil {
		slog.Error("migrations failed", "error", err)
		os.Exit(1)
	}

	queries := db.New(sqlDB)
	svc := service.New(queries, sqlDB, threshold)
	handler := api.NewRouter(svc)

	addr := fmt.Sprintf(":%s", port)
	slog.Info("ingredients service listening", "addr", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}

func runMigrations(sqlDB *sql.DB) error {
	srcDriver, err := iofs.New(db.MigrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("create migration source: %w", err)
	}
	dbDriver, err := postgres.WithInstance(sqlDB, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("create migration driver: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", srcDriver, "postgres", dbDriver)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}
