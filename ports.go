package neo4go

import "context"

type Migrator interface {
	Up(ctx context.Context) error
	Down(ctx context.Context) error
	UpTo(ctx context.Context, version int) error
	DownTo(ctx context.Context, version int) error
	Status(ctx context.Context) ([]MigrationStatus, error)
	Version(ctx context.Context) (int, error)
	Close() error
}

type Storage interface {
	Init(ctx context.Context) error
	GetAppliedMigrations(ctx context.Context) ([]MigrationRecord, error)
	RecordMigration(ctx context.Context, migration Migration) error
	RemoveMigration(ctx context.Context, version int) error
	GetCurrentVersion(ctx context.Context) (int, error)
	Close() error
}

type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}
