package storage

import "errors"

var (
	ErrUsernameTaken = errors.New("username taken")
	ErrUserNotFound  = errors.New("user not found")
)
