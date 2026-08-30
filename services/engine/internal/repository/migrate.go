package repository

import (
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strings"
)

//go:embed migrations/sqlite/*.sql
var sqliteMigrations embed.FS

//go:embed migrations/postgres/*.sql
var postgresMigrations embed.FS

// migrate applies pending migrations to the given database.
// driver must be "sqlite" or "postgres".
// It tracks applied migrations in a schema_migrations table.
// It is idempotent: safe to call on every startup.
func migrate(db *sql.DB, driver string) error {
	if err := ensureMigrationsTable(db, driver); err != nil {
		return err
	}

	applied, err := getAppliedMigrations(db)
	if err != nil {
		return err
	}

	migrations, err := loadMigrations(driver)
	if err != nil {
		return err
	}

	for _, m := range migrations {
		if applied[m.name] {
			continue
		}

		if _, err := db.Exec(m.sql); err != nil {
			return fmt.Errorf("apply migration %s: %w", m.name, err)
		}

		if err := recordMigration(db, driver, m.name); err != nil {
			return fmt.Errorf("record migration %s: %w", m.name, err)
		}
	}

	return nil
}

type migration struct {
	name string
	sql  string
}

func ensureMigrationsTable(db *sql.DB, driver string) error {
	switch driver {
	case "sqlite":
		_, err := db.Exec(`
			CREATE TABLE IF NOT EXISTS schema_migrations (
				name TEXT PRIMARY KEY,
				applied_at TEXT NOT NULL DEFAULT (datetime('now'))
			)`)
		return err
	case "postgres":
		_, err := db.Exec(`
			CREATE TABLE IF NOT EXISTS schema_migrations (
				name TEXT PRIMARY KEY,
				applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
			)`)
		return err
	default:
		return fmt.Errorf("unsupported driver: %s", driver)
	}
}

func getAppliedMigrations(db *sql.DB) (map[string]bool, error) {
	applied := make(map[string]bool)
	rows, err := db.Query(`SELECT name FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("query migrations: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		applied[name] = true
	}
	return applied, rows.Err()
}

func recordMigration(db *sql.DB, driver, name string) error {
	switch driver {
	case "sqlite":
		_, err := db.Exec(`INSERT INTO schema_migrations (name) VALUES (?)`, name)
		return err
	case "postgres":
		_, err := db.Exec(`INSERT INTO schema_migrations (name) VALUES ($1)`, name)
		return err
	default:
		return fmt.Errorf("unsupported driver: %s", driver)
	}
}

func loadMigrations(driver string) ([]migration, error) {
	var fs embed.FS
	var dir string

	switch driver {
	case "sqlite":
		fs = sqliteMigrations
		dir = "migrations/sqlite"
	case "postgres":
		fs = postgresMigrations
		dir = "migrations/postgres"
	default:
		return nil, fmt.Errorf("unsupported driver: %s", driver)
	}

	entries, err := fs.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read migrations dir: %w", err)
	}

	var migrations []migration
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		content, err := fs.ReadFile(dir + "/" + entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}

		migrations = append(migrations, migration{
			name: strings.TrimSuffix(entry.Name(), ".sql"),
			sql:  string(content),
		})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].name < migrations[j].name
	})

	return migrations, nil
}
