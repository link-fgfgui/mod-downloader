package database

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"mod-downloader/logging"

	_ "github.com/mattn/go-sqlite3"
)

var (
	db     *sql.DB
	dbPath string
)

var errDatabaseNotOpen = errors.New("database is not open")

func Open() error {
	dir, err := os.Getwd()
	if err != nil {
		logging.Error("resolve database working directory failed", "error", err)
		return fmt.Errorf("get working dir: %w", err)
	}

	targetPath := filepath.Join(dir, "mods.db")
	if db != nil && dbPath == targetPath {
		if err := db.Ping(); err == nil {
			logging.Debug("database already open", "path", targetPath)
			return nil
		}
		logging.Warn("database ping failed, reopening", "path", targetPath, "error", err)
		Close()
	}

	if db != nil {
		logging.Info("database path changed, closing previous connection", "previousPath", dbPath, "nextPath", targetPath)
		Close()
	}

	dsn := targetPath + "?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=ON"
	logging.Info("opening database", "path", targetPath)
	nextDB, err := sql.Open("sqlite3", dsn)
	if err != nil {
		logging.Error("open database failed", "path", targetPath, "error", err)
		return fmt.Errorf("open db: %w", err)
	}

	if err := nextDB.Ping(); err != nil {
		nextDB.Close()
		logging.Error("ping database failed", "path", targetPath, "error", err)
		return fmt.Errorf("ping db: %w", err)
	}

	db = nextDB
	dbPath = targetPath
	if err := migrate(); err != nil {
		Close()
		logging.Error("migrate database failed", "path", targetPath, "error", err)
		return fmt.Errorf("migrate db: %w", err)
	}

	logging.Info("database opened", "path", targetPath)
	return nil
}

func Close() {
	if db != nil {
		logging.Info("closing database", "path", dbPath)
		db.Close()
		db = nil
	}
	dbPath = ""
}

func readyDB() (*sql.DB, error) {
	if db == nil {
		logging.Warn("database access attempted before open")
		return nil, errDatabaseNotOpen
	}
	return db, nil
}

func NewID() string {
	var buf [16]byte
	_, _ = rand.Read(buf[:])
	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		buf[0:4], buf[4:6], buf[6:8], buf[8:10], buf[10:16])
}

func migrate() error {
	d, err := readyDB()
	if err != nil {
		return err
	}
	logging.Debug("database migration started")
	if _, err = d.Exec(schema); err != nil {
		logging.Error("database schema migration failed", "error", err)
		return err
	}
	if err := ensureColumn(d, "mod_platforms", "updated_at", "INTEGER DEFAULT 0"); err != nil {
		logging.Error("database column migration failed", "table", "mod_platforms", "column", "updated_at", "error", err)
		return err
	}
	if err := ensureColumn(d, "mod_platform_versions", "sha1", "TEXT"); err != nil {
		logging.Error("database column migration failed", "table", "mod_platform_versions", "column", "sha1", "error", err)
		return err
	}
	if err := ensureColumn(d, "mod_platform_versions", "published_at", "INTEGER DEFAULT 0"); err != nil {
		logging.Error("database column migration failed", "table", "mod_platform_versions", "column", "published_at", "error", err)
		return err
	}
	if err := normalizeNullVersionArrays(d); err != nil {
		logging.Error("database normalization failed", "error", err)
		return err
	}
	if err := resetJarMetadataCacheWhenNeeded(d); err != nil {
		logging.Error("database jar metadata cache reset failed", "error", err)
		return err
	}
	logging.Debug("database migration completed")
	return nil
}

func ensureColumn(d *sql.DB, table string, column string, definition string) error {
	rows, err := d.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, typ string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		if name == column {
			return rows.Err()
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	_, err = d.Exec(`ALTER TABLE ` + table + ` ADD COLUMN ` + column + ` ` + definition)
	if err == nil {
		logging.Info("database column added", "table", table, "column", column)
	}
	return err
}

func normalizeNullVersionArrays(d *sql.DB) error {
	if _, err := d.Exec(`
		UPDATE mod_platforms
		SET updated_at = 0
		WHERE EXISTS (
			SELECT 1 FROM mod_platform_versions
			WHERE mod_platform_versions.platform = mod_platforms.platform
				AND mod_platform_versions.project_id = mod_platforms.project_id
				AND (mod_platform_versions.loaders IS NULL OR mod_platform_versions.loaders = 'null')
		)
	`); err != nil {
		return err
	}
	_, err := d.Exec(`
		UPDATE mod_platform_versions
		SET loaders = '[]'
		WHERE loaders IS NULL OR loaders = 'null'
	`)
	return err
}

func resetJarMetadataCacheWhenNeeded(d *sql.DB) error {
	const currentVersion = "recursive-jar-mod-id-v4"
	if _, err := d.Exec(`
		CREATE TABLE IF NOT EXISTS cache_metadata (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)
	`); err != nil {
		return err
	}

	var version string
	err := d.QueryRow(`SELECT value FROM cache_metadata WHERE key = 'jar_metadata_version'`).Scan(&version)
	if err == nil && version == currentVersion {
		return nil
	}
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	if _, err := d.Exec(`DELETE FROM mod_jar_metadata`); err != nil {
		return err
	}
	_, err = d.Exec(`
		INSERT INTO cache_metadata (key, value)
		VALUES ('jar_metadata_version', ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, currentVersion)
	return err
}

const schema = `
DROP TABLE IF EXISTS mod_file_paths;
DROP TABLE IF EXISTS mod_files;

CREATE TABLE IF NOT EXISTS mod_platforms (
    platform    TEXT NOT NULL,
    project_id  TEXT NOT NULL,
    slug        TEXT,
    name        TEXT,
    description TEXT,
    mcmod_url   TEXT,
    updated_at  INTEGER DEFAULT 0,
    PRIMARY KEY(platform, project_id)
);
CREATE INDEX IF NOT EXISTS idx_mod_platforms_slug ON mod_platforms(platform, slug);

CREATE TABLE IF NOT EXISTS platform_associations (
    id                    TEXT PRIMARY KEY,
    curseforge_project_id TEXT,
    modrinth_project_id   TEXT
);

CREATE TABLE IF NOT EXISTS mod_platform_versions (
    id            TEXT PRIMARY KEY,
    platform      TEXT NOT NULL,
    project_id    TEXT NOT NULL,
    version_id    TEXT NOT NULL,
    name          TEXT,
    version       TEXT,
    file_name     TEXT,
    download_url  TEXT,
    sha1          TEXT,
    published_at  INTEGER DEFAULT 0,
    downloads     INTEGER DEFAULT 0,
    game_versions TEXT,
    loaders       TEXT,
    FOREIGN KEY(platform, project_id) REFERENCES mod_platforms(platform, project_id) ON DELETE CASCADE,
    UNIQUE(platform, project_id, version_id)
);
CREATE INDEX IF NOT EXISTS idx_mod_platform_versions_project ON mod_platform_versions(platform, project_id);

CREATE TABLE IF NOT EXISTS mod_platform_version_scopes (
    platform          TEXT NOT NULL,
    project_id        TEXT NOT NULL,
    minecraft_version TEXT NOT NULL,
    mod_loader        TEXT NOT NULL,
    updated_at        INTEGER DEFAULT 0,
    PRIMARY KEY(platform, project_id, minecraft_version, mod_loader),
    FOREIGN KEY(platform, project_id) REFERENCES mod_platforms(platform, project_id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_mod_platform_version_scopes_project ON mod_platform_version_scopes(platform, project_id);

CREATE TABLE IF NOT EXISTS mod_dependencies (
    id                    TEXT PRIMARY KEY,
    platform_version_id   TEXT NOT NULL REFERENCES mod_platform_versions(id) ON DELETE CASCADE,
    dependency_project_id TEXT NOT NULL,
    dependency_version_id TEXT,
    dependency_type       TEXT
);
CREATE INDEX IF NOT EXISTS idx_mod_dependencies_pvid ON mod_dependencies(platform_version_id);

CREATE TABLE IF NOT EXISTS pinned_mods (
    id                TEXT PRIMARY KEY,
    platform          TEXT NOT NULL,
    project_id        TEXT NOT NULL,
    version_id        TEXT NOT NULL,
    minecraft_version TEXT NOT NULL,
    mod_loader        TEXT NOT NULL,
    UNIQUE(platform, project_id, minecraft_version, mod_loader)
);

CREATE TABLE IF NOT EXISTS mod_jar_metadata (
    sha1        TEXT NOT NULL,
    mod_id      TEXT NOT NULL,
    name        TEXT,
    version     TEXT,
    description TEXT,
    PRIMARY KEY(sha1, mod_id)
);
`
