package neo4go

import (
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"
)

func TestParserParseMigrations(t *testing.T) {
	tests := []struct {
		name        string
		files       map[string]string
		wantCount   int
		wantErr     error
		wantVersion int
		wantName    string
	}{
		{
			name: "valid single migration",
			files: map[string]string{
				"001_initial.cypher": `-- +neo4go Up
CREATE CONSTRAINT user_id IF NOT EXISTS FOR (u:User) REQUIRE u.id IS UNIQUE;

-- +neo4go Down
DROP CONSTRAINT user_id IF EXISTS;`,
			},
			wantCount:   1,
			wantErr:     nil,
			wantVersion: 1,
			wantName:    "initial",
		},
		{
			name: "multiple migrations ordered correctly",
			files: map[string]string{
				"001_initial.cypher": `-- +neo4go Up
CREATE CONSTRAINT user_id IF NOT EXISTS FOR (u:User) REQUIRE u.id IS UNIQUE;

-- +neo4go Down
DROP CONSTRAINT user_id IF EXISTS;`,
				"002_add_index.cypher": `-- +neo4go Up
CREATE INDEX user_email IF NOT EXISTS FOR (u:User) ON (u.email);

-- +neo4go Down
DROP INDEX user_email IF EXISTS;`,
			},
			wantCount: 2,
			wantErr:   nil,
		},
		{
			name:      "no migrations",
			files:     map[string]string{},
			wantCount: 0,
			wantErr:   ErrNoMigrations,
		},
		{
			name: "missing up statement",
			files: map[string]string{
				"001_initial.cypher": `-- +neo4go Down
DROP CONSTRAINT user_id IF EXISTS;`,
			},
			wantCount: 0,
			wantErr:   ErrNoUpStatement,
		},
		{
			name: "missing down statement",
			files: map[string]string{
				"001_initial.cypher": `-- +neo4go Up
CREATE CONSTRAINT user_id IF NOT EXISTS FOR (u:User) REQUIRE u.id IS UNIQUE;`,
			},
			wantCount: 0,
			wantErr:   ErrNoDownStatement,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filesystem := fstest.MapFS{}
			for name, content := range tt.files {
				filesystem[name] = &fstest.MapFile{
					Data: []byte(content),
					Mode: fs.FileMode(0644),
				}
			}

			p := newParser(filesystem)
			migrations, err := p.parseMigrations(".")

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(migrations) != tt.wantCount {
				t.Fatalf("expected %d migrations, got %d", tt.wantCount, len(migrations))
			}

			if tt.wantCount > 0 && tt.wantVersion > 0 {
				if migrations[0].Version != tt.wantVersion {
					t.Errorf("expected version %d, got %d", tt.wantVersion, migrations[0].Version)
				}

				if migrations[0].Name != tt.wantName {
					t.Errorf("expected name %s, got %s", tt.wantName, migrations[0].Name)
				}
			}
		})
	}
}

func TestParserSplitUpDown(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantUpSQL   string
		wantDownSQL string
		wantErr     error
	}{
		{
			name: "valid migration",
			content: `-- +neo4go Up
CREATE CONSTRAINT user_id IF NOT EXISTS FOR (u:User) REQUIRE u.id IS UNIQUE;

-- +neo4go Down
DROP CONSTRAINT user_id IF EXISTS;`,
			wantUpSQL:   "CREATE CONSTRAINT user_id IF NOT EXISTS FOR (u:User) REQUIRE u.id IS UNIQUE;",
			wantDownSQL: "DROP CONSTRAINT user_id IF EXISTS;",
			wantErr:     nil,
		},
		{
			name: "multiple statements",
			content: `-- +neo4go Up
CREATE CONSTRAINT user_id IF NOT EXISTS FOR (u:User) REQUIRE u.id IS UNIQUE;
CREATE INDEX user_email IF NOT EXISTS FOR (u:User) ON (u.email);

-- +neo4go Down
DROP CONSTRAINT user_id IF EXISTS;
DROP INDEX user_email IF EXISTS;`,
			wantUpSQL:   "CREATE CONSTRAINT user_id IF NOT EXISTS FOR (u:User) REQUIRE u.id IS UNIQUE;\nCREATE INDEX user_email IF NOT EXISTS FOR (u:User) ON (u.email);",
			wantDownSQL: "DROP CONSTRAINT user_id IF EXISTS;\nDROP INDEX user_email IF EXISTS;",
			wantErr:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &parser{}
			upSQL, downSQL, err := p.splitUpDown(tt.content)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if upSQL != tt.wantUpSQL {
				t.Errorf("expected up SQL:\n%s\ngot:\n%s", tt.wantUpSQL, upSQL)
			}

			if downSQL != tt.wantDownSQL {
				t.Errorf("expected down SQL:\n%s\ngot:\n%s", tt.wantDownSQL, downSQL)
			}
		})
	}
}

func TestParserMigrationOrdering(t *testing.T) {
	filesystem := fstest.MapFS{
		"003_third.cypher": &fstest.MapFile{
			Data: []byte("-- +neo4go Up\nCREATE INDEX i3;\n\n-- +neo4go Down\nDROP INDEX i3;"),
			Mode: fs.FileMode(0644),
		},
		"001_first.cypher": &fstest.MapFile{
			Data: []byte("-- +neo4go Up\nCREATE INDEX i1;\n\n-- +neo4go Down\nDROP INDEX i1;"),
			Mode: fs.FileMode(0644),
		},
		"002_second.cypher": &fstest.MapFile{
			Data: []byte("-- +neo4go Up\nCREATE INDEX i2;\n\n-- +neo4go Down\nDROP INDEX i2;"),
			Mode: fs.FileMode(0644),
		},
	}

	p := newParser(filesystem)
	migrations, err := p.parseMigrations(".")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(migrations) != 3 {
		t.Fatalf("expected 3 migrations, got %d", len(migrations))
	}

	for i, migration := range migrations {
		expectedVersion := i + 1
		if migration.Version != expectedVersion {
			t.Errorf("migration %d: expected version %d, got %d", i, expectedVersion, migration.Version)
		}
	}
}

func TestParserInvalidFilenames(t *testing.T) {
	filesystem := fstest.MapFS{
		"invalid.cypher": &fstest.MapFile{
			Data: []byte("-- +neo4go Up\nCREATE INDEX i1;\n\n-- +neo4go Down\nDROP INDEX i1;"),
			Mode: fs.FileMode(0644),
		},
		"001_valid.cypher": &fstest.MapFile{
			Data: []byte("-- +neo4go Up\nCREATE INDEX i1;\n\n-- +neo4go Down\nDROP INDEX i1;"),
			Mode: fs.FileMode(0644),
		},
		"notamigration.txt": &fstest.MapFile{
			Data: []byte("some text"),
			Mode: fs.FileMode(0644),
		},
	}

	p := newParser(filesystem)
	migrations, err := p.parseMigrations(".")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(migrations) != 1 {
		t.Errorf("expected 1 valid migration, got %d", len(migrations))
	}

	if migrations[0].Version != 1 {
		t.Errorf("expected version 1, got %d", migrations[0].Version)
	}
}
