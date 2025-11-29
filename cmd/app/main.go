package main

import (
	"HysteriaBackend/internal/config"
	"HysteriaBackend/internal/storage/postgres"
	"flag"
	"fmt"
	"log/slog"
	_ "net/http"
	"os"
	"strings"
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
	storage := postgres.MustLoad(cfg)
	slog.Info(fmt.Sprintf("Connected to database: %s", cfg.Database.Dbname))
	user, err := storage.GetClientByClientID("ewq321fds654fsd")
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to load users: %v", err))
	}
	slog.Info(fmt.Sprintf("Loaded %d users from database", user))
}
