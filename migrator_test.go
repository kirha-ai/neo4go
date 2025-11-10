package neo4go

import (
	"context"
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"
	"time"
)

func TestMigratorUp(t *testing.T) {
	tests := []struct {
		name              string
		migrations        []Migration
		appliedVersions   []int
		expectError       bool
		storageInitError  error
		storageRecordErr  error
	}{
		{
			name: "all migrations already applied",
			migrations: []Migration{
				{Version: 1, Name: "initial", UpSQL: "CREATE CONSTRAINT c1;", DownSQL: "DROP CONSTRAINT c1;", Checksum: "abc"},
			},
			appliedVersions: []int{1},
			expectError:     false,
		},
		{
			name: "storage init error",
			migrations: []Migration{
				{Version: 1, Name: "initial", UpSQL: "CREATE CONSTRAINT c1;", DownSQL: "DROP CONSTRAINT c1;", Checksum: "abc"},
			},
			appliedVersions:  []int{},
			expectError:      true,
			storageInitError: errors.New("init failed"),
		},
		{
			name: "storage record error",
			migrations: []Migration{
				{Version: 1, Name: "initial", UpSQL: "CREATE CONSTRAINT c1;", DownSQL: "DROP CONSTRAINT c1;", Checksum: "abc"},
			},
			appliedVersions:  []int{},
			expectError:      true,
			storageRecordErr: errors.New("record failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			storage := newMockStorage()
			logger := newMockLogger()

			if tt.storageInitError != nil {
				storage.InitFunc = func(ctx context.Context) error {
					return tt.storageInitError
				}
			}

			if tt.storageRecordErr != nil {
				storage.RecordFunc = func(ctx context.Context, migration Migration) error {
					return tt.storageRecordErr
				}
			}

			for _, version := range tt.appliedVersions {
				storage.RecordMigration(ctx, Migration{
					Version:  version,
					Name:     "test",
					Checksum: "test",
				})
			}

			storage.GetAppliedFunc = func(ctx context.Context) ([]MigrationRecord, error) {
				var records []MigrationRecord
				for _, v := range tt.appliedVersions {
					records = append(records, MigrationRecord{
						Version:   v,
						Name:      "test",
						AppliedAt: time.Now(),
						Checksum:  "test",
					})
				}
				return records, nil
			}

			m := &migrator{
				driver:     nil,
				storage:    storage,
				migrations: tt.migrations,
				database:   "neo4j",
				logger:     logger,
			}

			err := m.Up(ctx)

			if tt.expectError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestMigratorDown(t *testing.T) {
	tests := []struct {
		name            string
		migrations      []Migration
		currentVersion  int
		expectError     bool
		expectedVersion int
		storageInitErr  error
	}{
		{
			name: "rollback last migration",
			migrations: []Migration{
				{Version: 1, Name: "initial", UpSQL: "CREATE CONSTRAINT c1;", DownSQL: "DROP CONSTRAINT c1;", Checksum: "abc"},
				{Version: 2, Name: "indexes", UpSQL: "CREATE INDEX i1;", DownSQL: "DROP INDEX i1;", Checksum: "def"},
			},
			currentVersion:  2,
			expectError:     false,
			expectedVersion: 1,
		},
		{
			name: "no migrations to rollback",
			migrations: []Migration{
				{Version: 1, Name: "initial", UpSQL: "CREATE CONSTRAINT c1;", DownSQL: "DROP CONSTRAINT c1;", Checksum: "abc"},
			},
			currentVersion:  0,
			expectError:     false,
			expectedVersion: 0,
		},
		{
			name: "migration not found",
			migrations: []Migration{
				{Version: 1, Name: "initial", UpSQL: "CREATE CONSTRAINT c1;", DownSQL: "DROP CONSTRAINT c1;", Checksum: "abc"},
			},
			currentVersion:  3,
			expectError:     true,
			expectedVersion: 3,
		},
		{
			name: "storage init error",
			migrations: []Migration{
				{Version: 1, Name: "initial", UpSQL: "CREATE CONSTRAINT c1;", DownSQL: "DROP CONSTRAINT c1;", Checksum: "abc"},
			},
			currentVersion:  1,
			expectError:     true,
			expectedVersion: 1,
			storageInitErr:  errors.New("init failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			storage := newMockStorage()
			logger := newMockLogger()

			if tt.storageInitErr != nil {
				storage.InitFunc = func(ctx context.Context) error {
					return tt.storageInitErr
				}
			}

			storage.GetVersionFunc = func(ctx context.Context) (int, error) {
				return tt.currentVersion, nil
			}

			m := &migrator{
				driver:     nil,
				storage:    storage,
				migrations: tt.migrations,
				database:   "neo4j",
				logger:     logger,
			}

			err := m.Down(ctx)

			if tt.expectError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestMigratorUpTo(t *testing.T) {
	tests := []struct {
		name            string
		migrations      []Migration
		appliedVersions []int
		targetVersion   int
		expectedApplied []int
		expectError     bool
	}{
		{
			name: "migrate to specific version",
			migrations: []Migration{
				{Version: 1, Name: "initial", UpSQL: "CREATE CONSTRAINT c1;", DownSQL: "DROP CONSTRAINT c1;", Checksum: "abc"},
				{Version: 2, Name: "indexes", UpSQL: "CREATE INDEX i1;", DownSQL: "DROP INDEX i1;", Checksum: "def"},
				{Version: 3, Name: "more", UpSQL: "CREATE INDEX i2;", DownSQL: "DROP INDEX i2;", Checksum: "ghi"},
			},
			appliedVersions: []int{},
			targetVersion:   2,
			expectedApplied: []int{1, 2},
			expectError:     false,
		},
		{
			name: "skip already applied migrations",
			migrations: []Migration{
				{Version: 1, Name: "initial", UpSQL: "CREATE CONSTRAINT c1;", DownSQL: "DROP CONSTRAINT c1;", Checksum: "abc"},
				{Version: 2, Name: "indexes", UpSQL: "CREATE INDEX i1;", DownSQL: "DROP INDEX i1;", Checksum: "def"},
				{Version: 3, Name: "more", UpSQL: "CREATE INDEX i2;", DownSQL: "DROP INDEX i2;", Checksum: "ghi"},
			},
			appliedVersions: []int{1},
			targetVersion:   2,
			expectedApplied: []int{1, 2},
			expectError:     false,
		},
		{
			name: "invalid version",
			migrations: []Migration{
				{Version: 1, Name: "initial", UpSQL: "CREATE CONSTRAINT c1;", DownSQL: "DROP CONSTRAINT c1;", Checksum: "abc"},
			},
			appliedVersions: []int{},
			targetVersion:   -1,
			expectedApplied: []int{},
			expectError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			storage := newMockStorage()
			logger := newMockLogger()

			for _, version := range tt.appliedVersions {
				storage.RecordMigration(ctx, Migration{
					Version:  version,
					Name:     "test",
					Checksum: "test",
				})
			}

			storage.GetAppliedFunc = func(ctx context.Context) ([]MigrationRecord, error) {
				var records []MigrationRecord
				for _, v := range tt.appliedVersions {
					records = append(records, MigrationRecord{
						Version:   v,
						Name:      "test",
						AppliedAt: time.Now(),
						Checksum:  "test",
					})
				}
				return records, nil
			}

			m := &migrator{
				driver:     nil,
				storage:    storage,
				migrations: tt.migrations,
				database:   "neo4j",
				logger:     logger,
			}

			err := m.UpTo(ctx, tt.targetVersion)

			if tt.expectError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestMigratorDownTo(t *testing.T) {
	tests := []struct {
		name            string
		migrations      []Migration
		appliedVersions []int
		targetVersion   int
		expectedRemoved []int
		expectError     bool
	}{
		{
			name: "rollback to specific version",
			migrations: []Migration{
				{Version: 1, Name: "initial", UpSQL: "CREATE CONSTRAINT c1;", DownSQL: "DROP CONSTRAINT c1;", Checksum: "abc"},
				{Version: 2, Name: "indexes", UpSQL: "CREATE INDEX i1;", DownSQL: "DROP INDEX i1;", Checksum: "def"},
				{Version: 3, Name: "more", UpSQL: "CREATE INDEX i2;", DownSQL: "DROP INDEX i2;", Checksum: "ghi"},
			},
			appliedVersions: []int{1, 2, 3},
			targetVersion:   1,
			expectedRemoved: []int{2, 3},
			expectError:     false,
		},
		{
			name: "no rollback needed",
			migrations: []Migration{
				{Version: 1, Name: "initial", UpSQL: "CREATE CONSTRAINT c1;", DownSQL: "DROP CONSTRAINT c1;", Checksum: "abc"},
				{Version: 2, Name: "indexes", UpSQL: "CREATE INDEX i1;", DownSQL: "DROP INDEX i1;", Checksum: "def"},
			},
			appliedVersions: []int{1, 2},
			targetVersion:   2,
			expectedRemoved: []int{},
			expectError:     false,
		},
		{
			name: "invalid version",
			migrations: []Migration{
				{Version: 1, Name: "initial", UpSQL: "CREATE CONSTRAINT c1;", DownSQL: "DROP CONSTRAINT c1;", Checksum: "abc"},
			},
			appliedVersions: []int{1},
			targetVersion:   -1,
			expectedRemoved: []int{},
			expectError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			storage := newMockStorage()
			logger := newMockLogger()

			storage.GetAppliedFunc = func(ctx context.Context) ([]MigrationRecord, error) {
				var records []MigrationRecord
				for _, v := range tt.appliedVersions {
					records = append(records, MigrationRecord{
						Version:   v,
						Name:      "test",
						AppliedAt: time.Now(),
						Checksum:  "test",
					})
				}
				return records, nil
			}

			m := &migrator{
				driver:     nil,
				storage:    storage,
				migrations: tt.migrations,
				database:   "neo4j",
				logger:     logger,
			}

			err := m.DownTo(ctx, tt.targetVersion)

			if tt.expectError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestMigratorStatus(t *testing.T) {
	tests := []struct {
		name            string
		migrations      []Migration
		appliedVersions []int
		expectedCount   int
		expectError     bool
	}{
		{
			name: "mixed applied and pending migrations",
			migrations: []Migration{
				{Version: 1, Name: "initial", UpSQL: "CREATE CONSTRAINT c1;", DownSQL: "DROP CONSTRAINT c1;", Checksum: "abc"},
				{Version: 2, Name: "indexes", UpSQL: "CREATE INDEX i1;", DownSQL: "DROP INDEX i1;", Checksum: "def"},
				{Version: 3, Name: "more", UpSQL: "CREATE INDEX i2;", DownSQL: "DROP INDEX i2;", Checksum: "ghi"},
			},
			appliedVersions: []int{1, 2},
			expectedCount:   3,
			expectError:     false,
		},
		{
			name: "all migrations pending",
			migrations: []Migration{
				{Version: 1, Name: "initial", UpSQL: "CREATE CONSTRAINT c1;", DownSQL: "DROP CONSTRAINT c1;", Checksum: "abc"},
				{Version: 2, Name: "indexes", UpSQL: "CREATE INDEX i1;", DownSQL: "DROP INDEX i1;", Checksum: "def"},
			},
			appliedVersions: []int{},
			expectedCount:   2,
			expectError:     false,
		},
		{
			name: "all migrations applied",
			migrations: []Migration{
				{Version: 1, Name: "initial", UpSQL: "CREATE CONSTRAINT c1;", DownSQL: "DROP CONSTRAINT c1;", Checksum: "abc"},
				{Version: 2, Name: "indexes", UpSQL: "CREATE INDEX i1;", DownSQL: "DROP INDEX i1;", Checksum: "def"},
			},
			appliedVersions: []int{1, 2},
			expectedCount:   2,
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			storage := newMockStorage()
			logger := newMockLogger()

			storage.GetAppliedFunc = func(ctx context.Context) ([]MigrationRecord, error) {
				var records []MigrationRecord
				for _, v := range tt.appliedVersions {
					records = append(records, MigrationRecord{
						Version:   v,
						Name:      "test",
						AppliedAt: time.Now(),
						Checksum:  "test",
					})
				}
				return records, nil
			}

			m := &migrator{
				driver:     nil,
				storage:    storage,
				migrations: tt.migrations,
				database:   "neo4j",
				logger:     logger,
			}

			statuses, err := m.Status(ctx)

			if tt.expectError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(statuses) != tt.expectedCount {
				t.Errorf("expected %d statuses, got %d", tt.expectedCount, len(statuses))
			}

			appliedMap := make(map[int]bool)
			for _, v := range tt.appliedVersions {
				appliedMap[v] = true
			}

			for _, status := range statuses {
				expectedApplied := appliedMap[status.Version]
				if status.Applied != expectedApplied {
					t.Errorf("version %d: expected applied=%v, got %v", status.Version, expectedApplied, status.Applied)
				}
			}
		})
	}
}

func TestMigratorVersion(t *testing.T) {
	tests := []struct {
		name            string
		currentVersion  int
		expectError     bool
		storageInitErr  error
		storageVersionErr error
	}{
		{
			name:           "get current version",
			currentVersion: 5,
			expectError:    false,
		},
		{
			name:           "no migrations applied",
			currentVersion: 0,
			expectError:    false,
		},
		{
			name:           "storage init error",
			currentVersion: 0,
			expectError:    true,
			storageInitErr: errors.New("init failed"),
		},
		{
			name:              "storage version error",
			currentVersion:    0,
			expectError:       true,
			storageVersionErr: errors.New("version failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			storage := newMockStorage()
			logger := newMockLogger()

			if tt.storageInitErr != nil {
				storage.InitFunc = func(ctx context.Context) error {
					return tt.storageInitErr
				}
			}

			if tt.storageVersionErr != nil {
				storage.GetVersionFunc = func(ctx context.Context) (int, error) {
					return 0, tt.storageVersionErr
				}
			} else {
				storage.GetVersionFunc = func(ctx context.Context) (int, error) {
					return tt.currentVersion, nil
				}
			}

			m := &migrator{
				driver:     nil,
				storage:    storage,
				migrations: []Migration{},
				database:   "neo4j",
				logger:     logger,
			}

			version, err := m.Version(ctx)

			if tt.expectError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if version != tt.currentVersion {
				t.Errorf("expected version %d, got %d", tt.currentVersion, version)
			}
		})
	}
}

func TestNewMigrator(t *testing.T) {
	tests := []struct {
		name        string
		filesystem  fs.FS
		dir         string
		expectError bool
	}{
		{
			name: "valid migrations",
			filesystem: fstest.MapFS{
				"001_initial.cypher": &fstest.MapFile{
					Data: []byte("-- +neo4go Up\nCREATE CONSTRAINT c1;\n\n-- +neo4go Down\nDROP CONSTRAINT c1;"),
				},
			},
			dir:         ".",
			expectError: false,
		},
		{
			name:        "no migrations",
			filesystem:  fstest.MapFS{},
			dir:         ".",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := newMockStorage()
			logger := newMockLogger()

			m, err := newMigrator(nil, storage, tt.filesystem, tt.dir, "neo4j", logger)

			if tt.expectError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if m == nil {
				t.Fatal("expected migrator, got nil")
			}
		})
	}
}
