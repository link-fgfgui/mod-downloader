package database

import (
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"mod-downloader/logging"

	"github.com/tidwall/buntdb"
)

const databaseFileName = "mods.buntdb"

var (
	db     *buntdb.DB
	dbPath string
)

var errDatabaseNotOpen = errors.New("database is not open")

func Open() error {
	dir, err := os.Getwd()
	if err != nil {
		logging.Error("resolve database working directory failed", "error", err)
		return fmt.Errorf("get working dir: %w", err)
	}

	targetPath := filepath.Join(dir, databaseFileName)
	if db != nil && dbPath == targetPath {
		logging.Debug("database already open", "path", targetPath)
		return nil
	}

	if db != nil {
		logging.Info("database path changed, closing previous connection", "previousPath", dbPath, "nextPath", targetPath)
		Close()
	}

	logging.Info("opening database", "path", targetPath)
	nextDB, err := buntdb.Open(targetPath)
	if err != nil {
		logging.Error("open database failed", "path", targetPath, "error", err)
		return fmt.Errorf("open db: %w", err)
	}
	config := buntdb.Config{
		SyncPolicy:           buntdb.Never,
		AutoShrinkPercentage: 100,
		AutoShrinkMinSize:    32 * 1024 * 1024,
	}
	if err := nextDB.SetConfig(config); err != nil {
		_ = nextDB.Close()
		logging.Error("configure database failed", "path", targetPath, "error", err)
		return fmt.Errorf("configure db: %w", err)
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
		if err := db.Close(); err != nil {
			logging.Warn("close database failed", "path", dbPath, "error", err)
		}
		db = nil
	}
	dbPath = ""
}

func readyDB() (*buntdb.DB, error) {
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
	if err := d.Update(func(tx *buntdb.Tx) error {
		return resetJarMetadataCacheWhenNeeded(tx)
	}); err != nil {
		return err
	}
	logging.Debug("database migration completed")
	return nil
}
