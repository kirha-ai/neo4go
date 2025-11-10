package neo4go

import (
	"context"
	"sync"
	"time"
)

type mockStorage struct {
	mu                sync.RWMutex
	InitFunc          func(ctx context.Context) error
	GetAppliedFunc    func(ctx context.Context) ([]MigrationRecord, error)
	RecordFunc        func(ctx context.Context, migration Migration) error
	RemoveFunc        func(ctx context.Context, version int) error
	GetVersionFunc    func(ctx context.Context) (int, error)
	CloseFunc         func() error
	appliedMigrations map[int]MigrationRecord
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		appliedMigrations: make(map[int]MigrationRecord),
	}
}

func (m *mockStorage) Init(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.InitFunc != nil {
		return m.InitFunc(ctx)
	}
	return nil
}

func (m *mockStorage) GetAppliedMigrations(ctx context.Context) ([]MigrationRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.GetAppliedFunc != nil {
		return m.GetAppliedFunc(ctx)
	}

	var records []MigrationRecord
	for _, record := range m.appliedMigrations {
		records = append(records, record)
	}
	return records, nil
}

func (m *mockStorage) RecordMigration(ctx context.Context, migration Migration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.RecordFunc != nil {
		return m.RecordFunc(ctx, migration)
	}

	m.appliedMigrations[migration.Version] = MigrationRecord{
		Version:   migration.Version,
		Name:      migration.Name,
		AppliedAt: time.Now(),
		Checksum:  migration.Checksum,
	}
	return nil
}

func (m *mockStorage) RemoveMigration(ctx context.Context, version int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.RemoveFunc != nil {
		return m.RemoveFunc(ctx, version)
	}

	delete(m.appliedMigrations, version)
	return nil
}

func (m *mockStorage) GetCurrentVersion(ctx context.Context) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.GetVersionFunc != nil {
		return m.GetVersionFunc(ctx)
	}

	maxVersion := 0
	for version := range m.appliedMigrations {
		if version > maxVersion {
			maxVersion = version
		}
	}
	return maxVersion, nil
}

func (m *mockStorage) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

type mockLogger struct {
	mu       sync.RWMutex
	DebugLog []string
	InfoLog  []string
	WarnLog  []string
	ErrorLog []string
}

func newMockLogger() *mockLogger {
	return &mockLogger{
		DebugLog: make([]string, 0),
		InfoLog:  make([]string, 0),
		WarnLog:  make([]string, 0),
		ErrorLog: make([]string, 0),
	}
}

func (m *mockLogger) Debug(msg string, args ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.DebugLog = append(m.DebugLog, msg)
}

func (m *mockLogger) Info(msg string, args ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.InfoLog = append(m.InfoLog, msg)
}

func (m *mockLogger) Warn(msg string, args ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.WarnLog = append(m.WarnLog, msg)
}

func (m *mockLogger) Error(msg string, args ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ErrorLog = append(m.ErrorLog, msg)
}

type mockDriver struct{}

func (m *mockDriver) Close(ctx context.Context) error {
	return nil
}
