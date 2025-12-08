package storage

import "errors"

var (
	ErrClientNotFound = errors.New("client not found")
)

type Client struct {
	ID       int
	ChatID   int
	Username string
	SubID    string
	ClientID string
	Expire   int
	MaxConns int
}
