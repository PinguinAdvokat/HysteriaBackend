package postgres

import (
	"database/sql"
	"log/slog"
	"time"

	_ "github.com/lib/pq"

	"HysteriaBackend/internal/config"
	"HysteriaBackend/internal/storage"
)

type PostgresStorage struct {
	db *sql.DB
}

func (p *PostgresStorage) GetClientByClientID(clientID string) (storage.Client, error) {
	slog.Debug("Fetching client with clientID: %s", clientID, nil)
	var client storage.Client
	query := "SELECT id, chat_id, username, subID, client_id, expire, max_conns FROM users WHERE client_id = $1"
	row := p.db.QueryRow(query, clientID)
	err := row.Scan(&client.ID, &client.ChatID, &client.Username, &client.SubID, &client.ClientID, &client.Expire, &client.MaxConns)
	if err == sql.ErrNoRows {
		return client, storage.ErrClientNotFound
	}
	return client, err
}

func (p *PostgresStorage) CheckAccess(clientID string, connections int) (bool, storage.Client, error) {
	client, err := p.GetClientByClientID(clientID)
	if err != nil {
		return false, client, err
	}
	slog.Debug("Client found", slog.Any("client", client))
	now := int(time.Now().Unix())
	slog.Debug("Current time", slog.Int("now", now), nil)

	if client.MaxConns > 0 && connections >= client.MaxConns {
		slog.Debug("Client exceeded max connections", slog.String("clientID", client.Username))
		return false, client, nil
	}
	if client.Expire < now {
		slog.Debug("Client subscription expired", slog.String("clientID", client.Username))
		return false, client, nil
	} else {
		slog.Debug("Client subscription valid", slog.String("clientID", client.Username))
		return true, client, nil
	}
}

func (p *PostgresStorage) Close() error {
	return p.db.Close()
}

func MustLoad(cfg *config.Config) (*PostgresStorage, error) {
	connStr := "host=" + cfg.Database.Host +
		" port=" + cfg.Database.Port +
		" user=" + cfg.Database.User +
		" password=" + cfg.Database.Password +
		" dbname=" + cfg.Database.Dbname +
		" sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &PostgresStorage{db: db}, nil
}
