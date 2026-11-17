package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"feria.local/catalogo/internal/catalogo"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	databaseURL := urlBase()
	if databaseURL == "" {
		slog.Error("falta DATABASE_URL o PGUSER, PGPASSWORD y PGDATABASE")
		os.Exit(1)
	}
	port := getenv("PORT", "8091")
	version := getenv("VERSION", "1.0.0")

	ctx := context.Background()
	store, err := catalogo.Conectar(ctx, databaseURL)
	if err != nil {
		slog.Error("base de datos", "err", err)
		os.Exit(1)
	}
	defer store.Close()
	if err := store.Migrar(ctx); err != nil {
		slog.Error("migracion", "err", err)
		os.Exit(1)
	}
	if os.Getenv("SEED") == "true" {
		if err := store.Sembrar(ctx); err != nil {
			slog.Error("semilla", "err", err)
			os.Exit(1)
		}
	}

	srv := &http.Server{
		Addr:              "0.0.0.0:" + port,
		Handler:           catalogo.NuevoServicio(store, version),
		ReadHeaderTimeout: 5 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		slog.Info("escuchando", "addr", srv.Addr, "servicio", "catalogo")
		errCh <- srv.ListenAndServe()
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-sig:
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			slog.Error("servidor", "err", err)
			os.Exit(1)
		}
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdown)
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func urlBase() string {
	if v := os.Getenv("DATABASE_URL"); v != "" {
		return v
	}
	user := os.Getenv("PGUSER")
	pass := os.Getenv("PGPASSWORD")
	db := os.Getenv("PGDATABASE")
	if user == "" || pass == "" || db == "" {
		return ""
	}
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		url.QueryEscape(user),
		url.QueryEscape(pass),
		getenv("PGHOST", "127.0.0.1"),
		getenv("PGPORT", "5432"),
		url.QueryEscape(db),
	)
}
