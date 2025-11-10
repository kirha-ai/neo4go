package neo4go

import "time"

type Migration struct {
	Version  int
	Name     string
	UpSQL    string
	DownSQL  string
	Checksum string
}

type MigrationStatus struct {
	Version   int
	Name      string
	Applied   bool
	AppliedAt *time.Time
	Checksum  string
}

type MigrationRecord struct {
	Version   int
	Name      string
	AppliedAt time.Time
	Checksum  string
}
