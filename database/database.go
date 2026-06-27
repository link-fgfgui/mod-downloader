package database

import (
	"crypto/rand"
	"encoding/gob"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"mod-downloader/logging"
	"mod-downloader/models"
	structs "mod-downloader/structs/minecraft"

	"github.com/klauspost/compress/zstd"
)

const (
	databaseFileName = "mods.gob.zst"
	cacheVersion     = 2
)

var (
	db     *cacheDB
	dbPath string
)

var errDatabaseNotOpen = errors.New("database is not open")

type cacheDB struct {
	mu    sync.RWMutex
	path  string
	state cacheState
}

type cacheState struct {
	Version                int
	JarMetadataVersion     string
	ModPlatforms           map[platformKey]models.ModProject
	PlatformAssociations   map[string]PlatformAssociation
	PlatformVersions       map[versionKey]models.ModVersion
	PlatformVersionScopes  map[versionScopeKey]storedVersionScope
	PinnedMods             map[pinnedModKey]PinnedMod
	JarMetadata            map[string][]structs.ModInfo
	PlatformVersionKeyByID map[string]versionKey
}

type platformKey struct {
	Platform  string
	ProjectID string
}

type versionKey struct {
	Platform  string
	ProjectID string
	VersionID string
}

type versionScopeKey struct {
	Platform         string
	ProjectID        string
	MinecraftVersion string
	ModLoader        string
}

type pinnedModKey struct {
	Platform         string
	ModID            string
	MinecraftVersion string
	ModLoader        string
}

func Open() error {
	dir, err := os.Getwd()
	if err != nil {
		logging.Error("resolve cache working directory failed", "error", err)
		return fmt.Errorf("get working dir: %w", err)
	}

	targetPath := filepath.Join(dir, databaseFileName)
	if db != nil && dbPath == targetPath {
		logging.Debug("cache already open", "path", targetPath)
		return nil
	}

	if db != nil {
		logging.Info("cache path changed, closing previous store", "previousPath", dbPath, "nextPath", targetPath)
		Close()
	}

	logging.Info("opening cache", "path", targetPath)
	state, err := loadCacheState(targetPath)
	if err != nil {
		logging.Error("open cache failed", "path", targetPath, "error", err)
		return fmt.Errorf("open cache: %w", err)
	}
	state.normalize()
	nextDB := &cacheDB{path: targetPath, state: state}
	nextDB.migrateLocked()

	db = nextDB
	dbPath = targetPath
	logging.Info("cache opened", "path", targetPath)
	return nil
}

func Close() {
	if db != nil {
		logging.Info("closing cache", "path", dbPath)
		if err := db.save(); err != nil {
			logging.Warn("save cache failed", "path", dbPath, "error", err)
		}
		db = nil
	}
	dbPath = ""
}

func readyDB() (*cacheDB, error) {
	if db == nil {
		logging.Warn("cache access attempted before open")
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

func loadCacheState(path string) (cacheState, error) {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return newCacheState(), nil
		}
		return cacheState{}, err
	}
	defer f.Close()

	zr, err := zstd.NewReader(f)
	if err != nil {
		return cacheState{}, err
	}
	defer zr.Close()

	var state cacheState
	if err := gob.NewDecoder(zr).Decode(&state); err != nil {
		return cacheState{}, err
	}
	if state.Version == 0 {
		state.Version = cacheVersion
	}
	if state.Version < cacheVersion {
		logging.Info("cache version outdated, discarding", "fileVersion", state.Version, "currentVersion", cacheVersion)
		return newCacheState(), nil
	}
	return state, nil
}

func newCacheState() cacheState {
	state := cacheState{Version: cacheVersion}
	state.normalize()
	return state
}

func (s *cacheState) normalize() {
	if s.Version == 0 {
		s.Version = cacheVersion
	}
	if s.ModPlatforms == nil {
		s.ModPlatforms = make(map[platformKey]models.ModProject)
	}
	if s.PlatformAssociations == nil {
		s.PlatformAssociations = make(map[string]PlatformAssociation)
	}
	if s.PlatformVersions == nil {
		s.PlatformVersions = make(map[versionKey]models.ModVersion)
	}
	if s.PlatformVersionScopes == nil {
		s.PlatformVersionScopes = make(map[versionScopeKey]storedVersionScope)
	}
	if s.PinnedMods == nil {
		s.PinnedMods = make(map[pinnedModKey]PinnedMod)
	}
	if s.JarMetadata == nil {
		s.JarMetadata = make(map[string][]structs.ModInfo)
	}
	if s.PlatformVersionKeyByID == nil {
		s.PlatformVersionKeyByID = make(map[string]versionKey)
		for key, version := range s.PlatformVersions {
			if version.ID != "" {
				s.PlatformVersionKeyByID[version.ID] = key
			}
		}
	}
}

func (d *cacheDB) migrateLocked() {
	if d.state.JarMetadataVersion == jarMetadataVersion {
		return
	}
	d.state.JarMetadata = make(map[string][]structs.ModInfo)
	d.state.JarMetadataVersion = jarMetadataVersion
}

func (d *cacheDB) view(fn func(*cacheState) error) error {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return fn(&d.state)
}

func (d *cacheDB) update(fn func(*cacheState) error) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return fn(&d.state)
}

func (d *cacheDB) save() error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if err := os.MkdirAll(filepath.Dir(d.path), 0755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(d.path), filepath.Base(d.path)+".*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	zw, err := zstd.NewWriter(tmp)
	if err != nil {
		_ = tmp.Close()
		return err
	}
	encodeErr := gob.NewEncoder(zw).Encode(d.state)
	closeZErr := zw.Close()
	syncErr := tmp.Sync()
	closeErr := tmp.Close()
	if encodeErr != nil {
		return encodeErr
	}
	if closeZErr != nil {
		return closeZErr
	}
	if syncErr != nil {
		return syncErr
	}
	if closeErr != nil {
		return closeErr
	}
	if err := os.Rename(tmpPath, d.path); err != nil {
		return err
	}
	cleanup = false
	return nil
}

func copyStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}

func copyDependencies(deps []models.ModDependency) []models.ModDependency {
	if len(deps) == 0 {
		return nil
	}
	out := make([]models.ModDependency, len(deps))
	copy(out, deps)
	return out
}

func copyModInfos(mods []structs.ModInfo) []structs.ModInfo {
	if len(mods) == 0 {
		return nil
	}
	out := make([]structs.ModInfo, len(mods))
	copy(out, mods)
	return out
}

func copyVersion(v models.ModVersion) models.ModVersion {
	v.GameVersions = copyStringSlice(v.GameVersions)
	v.Loaders = copyStringSlice(v.Loaders)
	v.Dependencies = copyDependencies(v.Dependencies)
	return v
}
