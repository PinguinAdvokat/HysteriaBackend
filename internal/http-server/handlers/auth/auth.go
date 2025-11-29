package auth

import (
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"

	"HysteriaBackend/internal/storage"
)

type Request struct {
	Addr string `json:"addr" validate:"required"`
	Auth string `json:"auth" validate:"required"`
	Tx   int    `json:"tx" validate:"required"`
}

type Response struct {
	OK bool   `json:"ok"`
	ID string `json:"id,omitempty"`
}

type AccessChecker interface {
	CheckAccess(clientID string) (bool, storage.Client, error)
}

func New(checker AccessChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		allowed, client, err := checker.CheckAccess(req.Auth)
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
			slog.Debug("access granted", slog.String("clientID", req.Auth))
			render.JSON(w, r, Response{OK: true, ID: client.Username})
			return
		}
	}
}
