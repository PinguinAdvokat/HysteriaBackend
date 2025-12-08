package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/metrics"

	"HysteriaBackend/internal/config"
	"HysteriaBackend/internal/http-server/handlers/auth"
	"HysteriaBackend/internal/storage/postgres"
)

func parseLogLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func main() {
	// Parse command-line flags
	var configPath string
	var loglevel string
	flag.StringVar(&configPath, "config", "config.yaml", "path to config file")
	flag.StringVar(&loglevel, "loglevel", "info", "log level")
	flag.Parse()

	// Load configuration
	cfg := config.MustLoad(configPath)

	// Set up structured logging
	level := parseLogLevel(loglevel)
	handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	logger := slog.New(handler)
	slog.SetDefault(logger)
	slog.Info(fmt.Sprintf("Log level set to %s", loglevel))
	slog.Info(fmt.Sprintf("Config: %#v", cfg))

	// Initialize storage and perform a test query
	storage, err := postgres.MustLoad(cfg)
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to connect to database: %v", err))
		return
	}
	slog.Info(fmt.Sprintf("Connected to database: %s", cfg.Database.Dbname))

	// Set up HTTP server with metrics
	router := chi.NewRouter()

	router.Use(metrics.Collector(metrics.CollectorOpts{
		Host:  false,
		Proto: true,
		Skip: func(r *http.Request) bool {
			return r.Method != "OPTIONS"
		},
	}))
	router.Handle("/metrics", metrics.Handler())
	transport := metrics.Transport(metrics.TransportOpts{
		Host: true,
	})
	http.DefaultClient.Transport = transport(http.DefaultTransport)

	router.Post("/auth", auth.New(cfg, storage))

	slog.Info(fmt.Sprintf("Starting HTTP server on %s", cfg.HTTPServer.Address))
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	srv := &http.Server{
		Addr:         cfg.HTTPServer.Address,
		Handler:      router,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			slog.Error("failed to start server")
		}
	}()

	<-done
	slog.Info("shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("failed to shut down server gracefully", slog.Any("error", err))
	}

	if storage.Close() != nil {
		slog.Error("failed to close storage")
	}

	slog.Info("server stopped")
}
