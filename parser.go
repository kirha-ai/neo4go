package neo4go

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	upMarker   = "-- +neo4go Up"
	downMarker = "-- +neo4go Down"
)

var migrationFilePattern = regexp.MustCompile(`^(\d+)_(.+)\.cypher$`)

type parser struct {
	fs fs.FS
}

func newParser(filesystem fs.FS) *parser {
	return &parser{fs: filesystem}
}

func (p *parser) parseMigrations(dir string) ([]Migration, error) {
	entries, err := fs.ReadDir(p.fs, dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var migrations []Migration
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		matches := migrationFilePattern.FindStringSubmatch(entry.Name())
		if matches == nil {
			continue
		}

		version, err := strconv.Atoi(matches[1])
		if err != nil {
			continue
		}

		name := matches[2]
		filePath := filepath.Join(dir, entry.Name())

		migration, err := p.parseMigrationFile(filePath, version, name)
		if err != nil {
			return nil, fmt.Errorf("failed to parse migration %s: %w", entry.Name(), err)
		}

		migrations = append(migrations, migration)
	}

	if len(migrations) == 0 {
		return nil, ErrNoMigrations
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

func (p *parser) parseMigrationFile(filePath string, version int, name string) (Migration, error) {
	file, err := p.fs.Open(filePath)
	if err != nil {
		return Migration{}, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return Migration{}, fmt.Errorf("failed to read file: %w", err)
	}

	upSQL, downSQL, err := p.splitUpDown(string(content))
	if err != nil {
		return Migration{}, err
	}

	checksum := calculateChecksum(content)

	return Migration{
		Version:  version,
		Name:     name,
		UpSQL:    upSQL,
		DownSQL:  downSQL,
		Checksum: checksum,
	}, nil
}

func (p *parser) splitUpDown(content string) (string, string, error) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	var upSQL, downSQL strings.Builder
	var currentSection string

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, upMarker) {
			currentSection = "up"
			continue
		}

		if strings.HasPrefix(line, downMarker) {
			currentSection = "down"
			continue
		}

		switch currentSection {
		case "up":
			upSQL.WriteString(line)
			upSQL.WriteString("\n")
		case "down":
			downSQL.WriteString(line)
			downSQL.WriteString("\n")
		}
	}

	if err := scanner.Err(); err != nil {
		return "", "", fmt.Errorf("failed to scan file: %w", err)
	}

	upStr := strings.TrimSpace(upSQL.String())
	downStr := strings.TrimSpace(downSQL.String())

	if upStr == "" {
		return "", "", ErrNoUpStatement
	}

	if downStr == "" {
		return "", "", ErrNoDownStatement
	}

	return upStr, downStr, nil
}

func calculateChecksum(content []byte) string {
	hash := sha256.Sum256(content)
	return fmt.Sprintf("%x", hash)
}
