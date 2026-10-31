package storage

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type userStore struct {
	db   *sql.DB
	path string
}

const sqliteDriverName = "sqlite"

const favoriteRefMaxDepth = 32

var errFavoriteListRefCycle = errors.New("favorite list reference would create a cycle")

var ErrFavoriteListNameConflict = errors.New("favorite list name already exists in target scope")

func openUserStore(path string) (*userStore, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("empty user database path")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open(sqliteDriverName, path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	store := &userStore{db: db, path: path}
	if err := store.ensureSchema(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *userStore) close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *userStore) ensureSchema() error {
	statements := []string{
		`PRAGMA foreign_keys = ON`,
		`PRAGMA busy_timeout = 5000`,
		`CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS pinned_mods (
			id TEXT PRIMARY KEY,
			platform TEXT NOT NULL,
			mod_id TEXT NOT NULL,
			version_id TEXT NOT NULL,
			minecraft_version TEXT NOT NULL,
			mod_loader TEXT NOT NULL,
			UNIQUE(platform, mod_id, minecraft_version, mod_loader)
		)`,
		`CREATE TABLE IF NOT EXISTS favorite_lists (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			sort_order INTEGER NOT NULL,
			system INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS favorite_mods (
			id TEXT PRIMARY KEY,
			list_id TEXT NOT NULL REFERENCES favorite_lists(id) ON DELETE CASCADE,
			platform TEXT NOT NULL,
			mod_id TEXT NOT NULL,
			version_id TEXT NOT NULL DEFAULT '',
			minecraft_version TEXT NOT NULL DEFAULT '',
			mod_loader TEXT NOT NULL DEFAULT '',
			title TEXT NOT NULL DEFAULT '',
			slug TEXT NOT NULL DEFAULT '',
			icon_url TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			categories_json TEXT NOT NULL DEFAULT '[]',
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			UNIQUE(list_id, platform, mod_id, minecraft_version, mod_loader)
		)`,
		`CREATE TABLE IF NOT EXISTS usage_stats (
			key TEXT PRIMARY KEY,
			value INTEGER NOT NULL DEFAULT 0 CHECK(value >= 0),
			updated_at INTEGER NOT NULL
		)`,
		`INSERT OR IGNORE INTO schema_migrations(version, applied_at) VALUES (1, unixepoch())`,
	}
	for _, statement := range statements {
		if _, err := s.db.Exec(statement); err != nil {
			return err
		}
	}
	if err := s.ensureSchemaV2(); err != nil {
		return err
	}
	if err := s.ensureSchemaV3(); err != nil {
		return err
	}
	return nil
}

func (s *userStore) ensureSchemaV2() error {
	columns := map[string]string{
		"group_id":          "TEXT NOT NULL DEFAULT ''",
		"minecraft_version": "TEXT NOT NULL DEFAULT ''",
		"mod_loader":        "TEXT NOT NULL DEFAULT ''",
		"icon_kind":         "TEXT NOT NULL DEFAULT ''",
		"icon_value":        "TEXT NOT NULL DEFAULT ''",
		"icon_url":          "TEXT NOT NULL DEFAULT ''",
		"pinned":            "INTEGER NOT NULL DEFAULT 0",
	}
	for name, definition := range columns {
		ok, err := s.columnExists("favorite_lists", name)
		if err != nil {
			return err
		}
		if ok {
			continue
		}
		if _, err := s.db.Exec(`ALTER TABLE favorite_lists ADD COLUMN ` + name + ` ` + definition); err != nil {
			return err
		}
	}

	statements := []string{
		`CREATE TABLE IF NOT EXISTS favorite_groups (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			sort_order INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS favorite_list_refs (
			id TEXT PRIMARY KEY,
			parent_list_id TEXT NOT NULL REFERENCES favorite_lists(id) ON DELETE CASCADE,
			child_list_id TEXT NOT NULL REFERENCES favorite_lists(id) ON DELETE CASCADE,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			sort_order INTEGER NOT NULL,
			UNIQUE(parent_list_id, child_list_id),
			CHECK(parent_list_id <> child_list_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_favorite_lists_pinned_order ON favorite_lists(pinned, sort_order, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_favorite_lists_group_order ON favorite_lists(group_id, sort_order, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_favorite_groups_order ON favorite_groups(sort_order, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_favorite_list_refs_parent_order ON favorite_list_refs(parent_list_id, sort_order)`,
		`CREATE INDEX IF NOT EXISTS idx_favorite_list_refs_child ON favorite_list_refs(child_list_id)`,
		`INSERT OR IGNORE INTO schema_migrations(version, applied_at) VALUES (2, unixepoch())`,
	}
	for _, statement := range statements {
		if _, err := s.db.Exec(statement); err != nil {
			return err
		}
	}
	return nil
}

func (s *userStore) ensureSchemaV3() error {
	var applied int
	err := s.db.QueryRow(`SELECT 1 FROM schema_migrations WHERE version = 3`).Scan(&applied)
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	statements := []string{
		`DROP TABLE IF EXISTS favorite_list_refs`,
		`DROP TABLE IF EXISTS favorite_mods`,
		`DROP TABLE IF EXISTS favorite_lists`,
		`DROP TABLE IF EXISTS favorite_groups`,
		`CREATE TABLE favorite_lists (
			id TEXT PRIMARY KEY,
			group_id TEXT NOT NULL DEFAULT '',
			name TEXT NOT NULL COLLATE NOCASE,
			minecraft_version TEXT NOT NULL DEFAULT '',
			mod_loader TEXT NOT NULL DEFAULT '',
			icon_kind TEXT NOT NULL DEFAULT '',
			icon_value TEXT NOT NULL DEFAULT '',
			icon_url TEXT NOT NULL DEFAULT '',
			pinned INTEGER NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			sort_order INTEGER NOT NULL,
			system INTEGER NOT NULL DEFAULT 0,
			UNIQUE(name, minecraft_version, mod_loader)
		)`,
		`CREATE TABLE favorite_mods (
			id TEXT PRIMARY KEY,
			list_id TEXT NOT NULL REFERENCES favorite_lists(id) ON DELETE CASCADE,
			platform TEXT NOT NULL,
			mod_id TEXT NOT NULL,
			version_id TEXT NOT NULL DEFAULT '',
			minecraft_version TEXT NOT NULL DEFAULT '',
			mod_loader TEXT NOT NULL DEFAULT '',
			title TEXT NOT NULL DEFAULT '',
			slug TEXT NOT NULL DEFAULT '',
			icon_url TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			categories_json TEXT NOT NULL DEFAULT '[]',
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			UNIQUE(list_id, platform, mod_id, minecraft_version, mod_loader)
		)`,
		`CREATE TABLE favorite_groups (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			sort_order INTEGER NOT NULL
		)`,
		`CREATE TABLE favorite_list_refs (
			id TEXT PRIMARY KEY,
			parent_list_id TEXT NOT NULL REFERENCES favorite_lists(id) ON DELETE CASCADE,
			child_list_id TEXT NOT NULL REFERENCES favorite_lists(id) ON DELETE CASCADE,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			sort_order INTEGER NOT NULL,
			UNIQUE(parent_list_id, child_list_id),
			CHECK(parent_list_id <> child_list_id)
		)`,
		`CREATE TRIGGER favorite_mod_scope_insert
			BEFORE INSERT ON favorite_mods
			WHEN NOT EXISTS (
				SELECT 1 FROM favorite_lists fl
				WHERE fl.id = NEW.list_id
				  AND fl.minecraft_version = NEW.minecraft_version
				  AND fl.mod_loader = NEW.mod_loader
			)
			BEGIN
				SELECT RAISE(ABORT, 'favorite mod scope must match favorite list scope');
			END`,
		`CREATE TRIGGER favorite_mod_scope_update
			BEFORE UPDATE OF list_id, minecraft_version, mod_loader ON favorite_mods
			WHEN NOT EXISTS (
				SELECT 1 FROM favorite_lists fl
				WHERE fl.id = NEW.list_id
				  AND fl.minecraft_version = NEW.minecraft_version
				  AND fl.mod_loader = NEW.mod_loader
			)
			BEGIN
				SELECT RAISE(ABORT, 'favorite mod scope must match favorite list scope');
			END`,
		`CREATE INDEX idx_favorite_lists_pinned_order ON favorite_lists(pinned, sort_order, created_at)`,
		`CREATE INDEX idx_favorite_lists_group_order ON favorite_lists(group_id, sort_order, created_at)`,
		`CREATE INDEX idx_favorite_groups_order ON favorite_groups(sort_order, created_at)`,
		`CREATE INDEX idx_favorite_list_refs_parent_order ON favorite_list_refs(parent_list_id, sort_order)`,
		`CREATE INDEX idx_favorite_list_refs_child ON favorite_list_refs(child_list_id)`,
		`INSERT INTO schema_migrations(version, applied_at) VALUES (3, unixepoch())`,
	}
	for _, statement := range statements {
		if _, err := tx.Exec(statement); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

func (s *userStore) columnExists(table, column string) (bool, error) {
	rows, err := s.db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, typ string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			return false, err
		}
		if strings.EqualFold(name, column) {
			return true, nil
		}
	}
	return false, rows.Err()
}

type favoriteListScanner interface {
	Scan(dest ...any) error
}

func (s *userStore) reorderRows(table string, ids []string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	now := time.Now().Unix()
	for i, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, err := tx.Exec(`UPDATE `+table+` SET sort_order = ?, updated_at = ? WHERE id = ?`, i+1, now, id); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

type favoriteGroupScanner interface {
	Scan(dest ...any) error
}

type favoriteListRefScanner interface {
	Scan(dest ...any) error
}

type favoriteModScanner interface {
	Scan(dest ...any) error
}
