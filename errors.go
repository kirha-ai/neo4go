package neo4go

import "errors"

var (
	ErrNoMigrations            = errors.New("no migrations found")
	ErrInvalidVersion          = errors.New("invalid version number")
	ErrMigrationNotFound       = errors.New("migration not found")
	ErrNoUpStatement           = errors.New("migration missing up statement")
	ErrNoDownStatement         = errors.New("migration missing down statement")
	ErrInvalidConfig           = errors.New("invalid configuration")
	ErrDatabaseConnection      = errors.New("database connection error")
	ErrTransactionFailed       = errors.New("transaction failed")
	ErrCannotCreateInEmbedded  = errors.New("cannot create migration files when using embedded filesystem")
	ErrInvalidMigrationName    = errors.New("invalid migration name: only alphanumeric characters and underscores are allowed")
)
