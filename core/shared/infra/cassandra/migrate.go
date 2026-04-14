package cassandra

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"go-socket/core/shared/pkg/stackErr"

	"github.com/gocql/gocql"
)

var migrateOnce sync.Once
var migrateSingleton MigrateTool
var migrateMutex = &sync.Mutex{}

type Migration struct {
	ID         string
	Name       string
	Statements []string
}

//go:generate mockgen -package=cassandra -destination=migrate_mock.go -source=migrate.go
type MigrateTool interface {
	Migrate(ctx context.Context, session *gocql.Session, migrationTable string, migrations []Migration) error
	MigrateFromSource(ctx context.Context, session *gocql.Session, migrationTable string, source string) error
}

type migrateTool struct{}

func NewMigrateTool() MigrateTool {
	migrateOnce.Do(func() {
		migrateSingleton = &migrateTool{}
	})
	return migrateSingleton
}

func (m *migrateTool) Migrate(ctx context.Context, session *gocql.Session, migrationTable string, migrations []Migration) error {
	migrateMutex.Lock()
	defer migrateMutex.Unlock()

	if session == nil {
		return stackErr.Error(fmt.Errorf("cassandra migrate requires a valid session"))
	}
	migrationTable = strings.TrimSpace(migrationTable)
	if migrationTable == "" {
		return stackErr.Error(fmt.Errorf("cassandra migrate requires migration table name"))
	}

	if err := ensureMigrationTable(ctx, session, migrationTable); err != nil {
		return stackErr.Error(err)
	}

	for _, migration := range migrations {
		applied, err := isMigrationApplied(ctx, session, migrationTable, migration.ID)
		if err != nil {
			return stackErr.Error(err)
		}
		if applied {
			continue
		}

		for _, statement := range migration.Statements {
			if err := session.Query(statement).WithContext(ctx).Exec(); err != nil {
				if isExistingColumnError(err) {
					continue
				}
				return stackErr.Error(fmt.Errorf("run cassandra migration %s failed: %v", migration.ID, err))
			}
		}

		if err := session.Query(
			fmt.Sprintf(`INSERT INTO %s (id, name, applied_at) VALUES (?, ?, ?)`, migrationTable),
			migration.ID,
			migration.Name,
			time.Now().UTC(),
		).WithContext(ctx).Exec(); err != nil {
			return stackErr.Error(fmt.Errorf("mark cassandra migration %s applied failed: %v", migration.ID, err))
		}
	}
	return nil
}

func (m *migrateTool) MigrateFromSource(ctx context.Context, session *gocql.Session, migrationTable string, source string) error {
	path, err := normalizeFileSource(source)
	if err != nil {
		return stackErr.Error(err)
	}

	files, err := listMigrationFiles(path)
	if err != nil {
		return stackErr.Error(err)
	}

	migrations := make([]Migration, 0, len(files))
	for _, file := range files {
		content, readErr := os.ReadFile(file.Path)
		if readErr != nil {
			return stackErr.Error(fmt.Errorf("read cassandra migration file failed: %v", readErr))
		}
		statements := splitCQLStatements(string(content))
		if len(statements) == 0 {
			return stackErr.Error(fmt.Errorf("cassandra migration file has no statements: %s", file.Path))
		}
		migrations = append(migrations, Migration{
			ID:         file.Version,
			Name:       filepath.Base(file.Path),
			Statements: statements,
		})
	}
	return m.Migrate(ctx, session, migrationTable, migrations)
}

type migrationFile struct {
	Path    string
	Version string
}

func normalizeFileSource(source string) (string, error) {
	path := strings.TrimPrefix(strings.TrimSpace(source), "file://")
	if path == "" {
		return "", stackErr.Error(fmt.Errorf("cassandra migration source path is empty"))
	}
	if !filepath.IsAbs(path) {
		cwd, err := os.Getwd()
		if err != nil {
			return "", stackErr.Error(fmt.Errorf("get cwd failed: %v", err))
		}
		path = filepath.Join(cwd, path)
	}
	return filepath.Clean(path), nil
}

func listMigrationFiles(dir string) ([]migrationFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, stackErr.Error(fmt.Errorf("read cassandra migration dir failed: %v", err))
	}

	files := make([]migrationFile, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".up.cql") {
			continue
		}
		version, parseErr := parseMigrationVersion(name)
		if parseErr != nil {
			return nil, stackErr.Error(parseErr)
		}
		files = append(files, migrationFile{
			Path:    filepath.Join(dir, name),
			Version: version,
		})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Version < files[j].Version
	})
	return files, nil
}

func parseMigrationVersion(name string) (string, error) {
	parts := strings.SplitN(name, "_", 2)
	if len(parts) < 2 {
		return "", stackErr.Error(fmt.Errorf("invalid cassandra migration filename: %s", name))
	}
	version := strings.TrimSpace(parts[0])
	if version == "" {
		return "", stackErr.Error(fmt.Errorf("empty cassandra migration version in %s", name))
	}
	return version, nil
}

func splitCQLStatements(input string) []string {
	lines := strings.Split(input, "\n")
	out := make([]string, 0)
	var current strings.Builder

	flush := func() {
		stmt := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(current.String()), ";"))
		if stmt != "" {
			out = append(out, stmt)
		}
		current.Reset()
	}

	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "--") {
			continue
		}
		current.WriteString(line)
		current.WriteString("\n")
		if strings.HasSuffix(trim, ";") {
			flush()
		}
	}

	if strings.TrimSpace(current.String()) != "" {
		flush()
	}
	return out
}

func ensureMigrationTable(ctx context.Context, session *gocql.Session, migrationTable string) error {
	statement := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id text PRIMARY KEY,
			name text,
			applied_at timestamp
		)
	`, migrationTable)
	if err := session.Query(statement).WithContext(ctx).Exec(); err != nil {
		return stackErr.Error(fmt.Errorf("ensure cassandra migration table failed: %v", err))
	}
	return nil
}

func isMigrationApplied(ctx context.Context, session *gocql.Session, migrationTable, migrationID string) (bool, error) {
	var id string
	err := session.Query(
		fmt.Sprintf(`SELECT id FROM %s WHERE id = ? LIMIT 1`, migrationTable),
		strings.TrimSpace(migrationID),
	).WithContext(ctx).Scan(&id)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, gocql.ErrNotFound) {
		return false, nil
	}
	return false, stackErr.Error(fmt.Errorf("check cassandra migration failed: %v", err))
}

func isExistingColumnError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "already exists")
}
