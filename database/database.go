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

	"github.com/klauspost/compress/zstd"
)

const (
	databaseFileName = "mods.gob.zst"
	cacheVersion     = 3
)

var (
	db     *cacheDB
	dbPath string
)

var errDatabaseNotOpen = errors.New("database is not open")

type cacheDB struct {
	mu      sync.RWMutex
	path    string
	state   cacheState
	strings stringPool
}

type cacheState struct {
	Version                int
	ModPlatforms           map[platformKey]models.ModProject
	PlatformAssociations   map[string]PlatformAssociation
	PlatformVersions       map[versionKey]models.ModVersion
	PlatformVersionScopes  map[versionScopeKey]storedVersionScope
	PinnedMods             map[pinnedModKey]PinnedMod
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

type stringPool struct {
	values map[string]string
}

func newStringPool() stringPool {
	return stringPool{values: make(map[string]string)}
}

func (p *stringPool) Intern(s string) string {
	if s == "" {
		return ""
	}
	if v, ok := p.values[s]; ok {
		return v
	}
	p.values[s] = s
	return s
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
	pool := newStringPool()
	state.intern(&pool)
	nextDB := &cacheDB{path: targetPath, state: state, strings: pool}

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
	if s.PlatformVersionKeyByID == nil {
		s.PlatformVersionKeyByID = make(map[string]versionKey)
		for key, version := range s.PlatformVersions {
			if version.ID != "" {
				s.PlatformVersionKeyByID[version.ID] = key
			}
		}
	}
}

func (s *cacheState) intern(pool *stringPool) {
	if pool == nil {
		return
	}
	modPlatforms := make(map[platformKey]models.ModProject, len(s.ModPlatforms))
	for key, project := range s.ModPlatforms {
		key = internPlatformKey(pool, key)
		project = internModProject(pool, project)
		modPlatforms[key] = project
	}
	s.ModPlatforms = modPlatforms

	associations := make(map[string]PlatformAssociation, len(s.PlatformAssociations))
	for id, association := range s.PlatformAssociations {
		association = internPlatformAssociation(pool, association)
		associations[id] = association
	}
	s.PlatformAssociations = associations

	versions := make(map[versionKey]models.ModVersion, len(s.PlatformVersions))
	for key, version := range s.PlatformVersions {
		key = internVersionKey(pool, key)
		version = internModVersion(pool, version)
		versions[key] = version
	}
	s.PlatformVersions = versions

	scopes := make(map[versionScopeKey]storedVersionScope, len(s.PlatformVersionScopes))
	for key, scope := range s.PlatformVersionScopes {
		key = internVersionScopeKey(pool, key)
		scope = internStoredVersionScope(pool, scope)
		scopes[key] = scope
	}
	s.PlatformVersionScopes = scopes

	pinnedMods := make(map[pinnedModKey]PinnedMod, len(s.PinnedMods))
	for key, pin := range s.PinnedMods {
		key = internPinnedModKey(pool, key)
		pin = internPinnedMod(pool, pin)
		pinnedMods[key] = pin
	}
	s.PinnedMods = pinnedMods

	versionKeyByID := make(map[string]versionKey, len(s.PlatformVersionKeyByID))
	for id, key := range s.PlatformVersionKeyByID {
		versionKeyByID[id] = internVersionKey(pool, key)
	}
	s.PlatformVersionKeyByID = versionKeyByID
}

func internPlatformKey(pool *stringPool, key platformKey) platformKey {
	key.Platform = pool.Intern(key.Platform)
	key.ProjectID = pool.Intern(key.ProjectID)
	return key
}

func internVersionKey(pool *stringPool, key versionKey) versionKey {
	key.Platform = pool.Intern(key.Platform)
	key.ProjectID = pool.Intern(key.ProjectID)
	return key
}

func internVersionScopeKey(pool *stringPool, key versionScopeKey) versionScopeKey {
	key.Platform = pool.Intern(key.Platform)
	key.ProjectID = pool.Intern(key.ProjectID)
	key.MinecraftVersion = pool.Intern(key.MinecraftVersion)
	key.ModLoader = pool.Intern(key.ModLoader)
	return key
}

func internPinnedModKey(pool *stringPool, key pinnedModKey) pinnedModKey {
	key.Platform = pool.Intern(key.Platform)
	key.ModID = pool.Intern(key.ModID)
	key.MinecraftVersion = pool.Intern(key.MinecraftVersion)
	key.ModLoader = pool.Intern(key.ModLoader)
	return key
}

func internStringSlice(pool *stringPool, values []string) []string {
	for i, value := range values {
		values[i] = pool.Intern(value)
	}
	return values
}

func internModProject(pool *stringPool, project models.ModProject) models.ModProject {
	project.Platform = pool.Intern(project.Platform)
	project.ProjectID = pool.Intern(project.ProjectID)
	project.Slug = pool.Intern(project.Slug)
	project.Icon = pool.Intern(project.Icon)
	return project
}

func internPlatformAssociation(pool *stringPool, association PlatformAssociation) PlatformAssociation {
	association.CurseForgeProjectID = pool.Intern(association.CurseForgeProjectID)
	association.ModrinthProjectID = pool.Intern(association.ModrinthProjectID)
	return association
}

func internModVersion(pool *stringPool, version models.ModVersion) models.ModVersion {
	version.Platform = pool.Intern(version.Platform)
	version.ProjectID = pool.Intern(version.ProjectID)
	version.GameVersions = internStringSlice(pool, version.GameVersions)
	version.Loaders = internStringSlice(pool, version.Loaders)
	version.Dependencies = internDependencies(pool, version.Dependencies)
	version.ModIDs = internStringSlice(pool, version.ModIDs)
	return version
}

func internDependencies(pool *stringPool, deps []models.ModDependency) []models.ModDependency {
	for i := range deps {
		deps[i].DependencyProjectID = pool.Intern(deps[i].DependencyProjectID)
		deps[i].DependencyType = pool.Intern(deps[i].DependencyType)
	}
	return deps
}

func internStoredVersionScope(pool *stringPool, scope storedVersionScope) storedVersionScope {
	scope.Platform = pool.Intern(scope.Platform)
	scope.ProjectID = pool.Intern(scope.ProjectID)
	scope.MinecraftVersion = pool.Intern(scope.MinecraftVersion)
	scope.ModLoader = pool.Intern(scope.ModLoader)
	return scope
}

func internPinnedMod(pool *stringPool, pin PinnedMod) PinnedMod {
	pin.Platform = pool.Intern(pin.Platform)
	pin.ModID = pool.Intern(pin.ModID)
	pin.MinecraftVersion = pool.Intern(pin.MinecraftVersion)
	pin.ModLoader = pool.Intern(pin.ModLoader)
	return pin
}

func (d *cacheDB) view(fn func(*cacheState) error) error {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return fn(&d.state)
}

func (d *cacheDB) update(fn func(*cacheState, *stringPool) error) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return fn(&d.state, &d.strings)
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

func copyVersion(v models.ModVersion) models.ModVersion {
	v.GameVersions = copyStringSlice(v.GameVersions)
	v.Loaders = copyStringSlice(v.Loaders)
	v.Dependencies = copyDependencies(v.Dependencies)
	v.ModIDs = copyStringSlice(v.ModIDs)
	return v
}
