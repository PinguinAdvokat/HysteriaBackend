package postgres

import (
	"HysteriaBackend/internal/config"
	"database/sql"
	"log/slog"
	"os"

	_ "github.com/lib/pq"
)

type Client struct {
	id        int
	chat_id   int
	username  string
	subID     string
	client_id string
	expire    int
}

type PostgresStorage struct {
	db *sql.DB
}

func (p *PostgresStorage) GetClientByClientID(clientID string) (Client, error) {
	slog.Debug("Fetching client with clientID: %s", clientID, nil)
	var client Client
	query := "SELECT id, chat_id, username, subID, client_id, expire FROM users WHERE client_id = $1"
	row := p.db.QueryRow(query, clientID)
	err := row.Scan(&client.id, &client.chat_id, &client.username, &client.subID, &client.client_id, &client.expire)
	return client, err
}

func MustLoad(cfg *config.Config) *PostgresStorage {
	connStr := "host=" + cfg.Database.Host +
		" port=" + cfg.Database.Port +
		" user=" + cfg.Database.User +
		" password=" + cfg.Database.Password +
		" dbname=" + cfg.Database.Dbname +
		" sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		slog.Error("failed to connect to postgres: " + err.Error())
		os.Exit(1)
		return nil
	}
	if err := db.Ping(); err != nil {
		slog.Error("Cant connect to database: " + err.Error())
		os.Exit(1)
	}
	return &PostgresStorage{db: db}
}
