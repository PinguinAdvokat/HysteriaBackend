package auth

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"

	"HysteriaBackend/internal/config"
	"HysteriaBackend/internal/storage"
)

type Request struct {
	Addr string `json:"addr" validate:"required"`
	Auth string `json:"auth" validate:"required"`
	Tx   int    `json:"tx" validate:"omitempty,min=0"`
}

type Response struct {
	OK bool   `json:"ok"`
	ID string `json:"id,omitempty"`
}

type AccessChecker interface {
	CheckAccess(clientID string, connections int) (bool, storage.Client, error)
}

func New(cfg *config.Config, checker AccessChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slog.Debug(r.RemoteAddr)
		var req Request

		err := render.DecodeJSON(r.Body, &req)
		if errors.Is(err, io.EOF) {
			// Такую ошибку встретим, если получили запрос с пустым телом.
			// Обработаем её отдельно
			slog.Error("request body is empty")
			http.Error(w, "request body is empty", http.StatusBadRequest)
			return
		}
		if err != nil {
			slog.Error("failed to decode request: " + err.Error())
			http.Error(w, "failed to decode request", http.StatusBadRequest)
			return
		}

		slog.Debug("Decoded auth request", slog.Any("request", req))
		if err := validator.New().Struct(req); err != nil {
			validateErr := err.(validator.ValidationErrors)
			slog.Error("invalid request", slog.Any("errors", validateErr.Error()))
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		//check expired, max connections and existence
		allowed, client, err := checker.CheckAccess(req.Auth, 0)
		if errors.Is(err, storage.ErrClientNotFound) {
			slog.Warn("client not found", slog.String("clientID", req.Auth))
			render.JSON(w, r, Response{OK: false, ID: client.Username})
			return
		}
		if err != nil {
			slog.Error("failed to check access: " + err.Error())
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		if !allowed {
			slog.Debug("access denied", slog.String("clientID", req.Auth))
			render.JSON(w, r, Response{OK: false, ID: client.Username})
			return
		} else {
			ip := r.RemoteAddr
			if i := strings.IndexByte(r.RemoteAddr, ':'); i >= 0 {
				ip = r.RemoteAddr[:i]
			}
			if getConnectionsOfClient(cfg, ip, client.Username) >= client.MaxConns {
				slog.Debug("access denied due to max connections", slog.String("clientID", req.Auth))
				render.JSON(w, r, Response{OK: false, ID: client.Username})
				return
			}
			slog.Debug("access granted", slog.String("clientID", req.Auth))
			render.JSON(w, r, Response{OK: true, ID: client.Username})
			return
		}
	}
}

func getConnectionsOfClient(cfg *config.Config, address string, username string) int {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+address+":"+cfg.Hysteria.TrafficStatsPort+"/online", nil)
	if err != nil {
		slog.Error("failed to create request: " + err.Error())
		return 0
	}
	req.Header.Set("Authorization", cfg.Hysteria.Secret)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("failed to perform request: " + err.Error())
		return 0
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Error("unexpected status code: " + resp.Status)
		return 0
	}

	var stats map[string]int
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		slog.Error("failed to decode response body: " + err.Error())
		return 0
	}

	slog.Debug("fetched connections", slog.Any("stats", stats))

	// direct match
	if v, ok := stats[username]; ok {
		return v
	}

	// try case-insensitive / trimmed match
	uname := strings.TrimSpace(strings.ToLower(username))
	for k, v := range stats {
		if strings.ToLower(strings.TrimSpace(k)) == uname {
			return v
		}
	}

	return 0
}
