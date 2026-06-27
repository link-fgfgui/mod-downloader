package minecraft

import (
	"archive/zip"
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"mod-downloader/global"
	"mod-downloader/logging"
	structs "mod-downloader/structs/minecraft"

	"github.com/BurntSushi/toml"
)

type ModMetadataParser interface {
	Name() string
	CanParse(r *zip.Reader) bool
	Parse(r *zip.Reader) (parsedModMetadata, error)
}

type parsedModMetadata struct {
	Mods       []structs.ModInfo
	NestedJars []string
	// TODO: Add Dependencies field for JAR-embedded dependency declarations.
	// Each parser (Fabric: depends in fabric.mod.json, Forge/NeoForge: [[dependencies]] in mods.toml)
	// should populate this. Not implemented yet — only platform-level dependencies are used.
}

// --- Fabric ---

type fabricModParser struct{}

func (fabricModParser) Name() string { return "fabric" }

func (fabricModParser) CanParse(r *zip.Reader) bool {
	return zipHasFile(r, "fabric.mod.json")
}

func (fabricModParser) Parse(r *zip.Reader) (parsedModMetadata, error) {
	data, err := readZipFile(r, "fabric.mod.json")
	if err != nil {
		return parsedModMetadata{}, err
	}

	var meta struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Version     string `json:"version"`
		Description string `json:"description"`
		Jars        []struct {
			File string `json:"file"`
		} `json:"jars"`
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		return parsedModMetadata{}, fmt.Errorf("parse fabric.mod.json: %w", err)
	}

	result := parsedModMetadata{
		NestedJars: make([]string, 0, len(meta.Jars)),
	}
	if id := declaredModID(meta.ID); id != "" {
		result.Mods = append(result.Mods, structs.ModInfo{
			ID:          id,
			Name:        meta.Name,
			Version:     meta.Version,
			Description: meta.Description,
		})
	}
	for _, jar := range meta.Jars {
		if path := nestedJarPath(jar.File); path != "" {
			result.NestedJars = append(result.NestedJars, path)
		}
	}
	return result, nil
}

// --- NeoForge ---

type neoForgeModParser struct{}

func (neoForgeModParser) Name() string { return "neoforge" }

func (neoForgeModParser) CanParse(r *zip.Reader) bool {
	return zipHasFile(r, "META-INF/neoforge.mods.toml")
}

func (neoForgeModParser) Parse(r *zip.Reader) (parsedModMetadata, error) {
	mods, err := parseModsToml(r, "META-INF/neoforge.mods.toml")
	if err != nil {
		return parsedModMetadata{}, err
	}
	return parsedModMetadata{Mods: mods}, nil
}

// --- Forge ---

type forgeModParser struct{}

func (forgeModParser) Name() string { return "forge" }

func (forgeModParser) CanParse(r *zip.Reader) bool {
	return zipHasFile(r, "META-INF/mods.toml")
}

func (forgeModParser) Parse(r *zip.Reader) (parsedModMetadata, error) {
	mods, err := parseModsToml(r, "META-INF/mods.toml")
	if err != nil {
		return parsedModMetadata{}, err
	}
	return parsedModMetadata{Mods: mods}, nil
}

// --- shared mods.toml parser ---

func parseModsToml(r *zip.Reader, path string) ([]structs.ModInfo, error) {
	data, err := readZipFile(r, path)
	if err != nil {
		return nil, err
	}
	properties := jarManifestProperties(r)

	var meta struct {
		Mods []struct {
			ModID       string `toml:"modId"`
			DisplayName string `toml:"displayName"`
			Version     string `toml:"version"`
			Description string `toml:"description"`
		} `toml:"mods"`
	}
	if err := toml.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	infos := make([]structs.ModInfo, 0, len(meta.Mods))
	for _, m := range meta.Mods {
		id := declaredModID(m.ModID)
		if id == "" {
			continue
		}
		infos = append(infos, structs.ModInfo{
			ID:          id,
			Name:        resolveMetadataValue(m.DisplayName, properties),
			Version:     resolveMetadataValue(m.Version, properties),
			Description: resolveMetadataValue(m.Description, properties),
		})
	}
	return infos, nil
}

func jarManifestProperties(r *zip.Reader) map[string]string {
	data, err := readZipFile(r, "META-INF/MANIFEST.MF")
	if err != nil {
		return nil
	}

	attrs := parseManifestMainAttributes(data)
	if len(attrs) == 0 {
		return nil
	}

	props := make(map[string]string, len(attrs)+1)
	if version := strings.TrimSpace(attrs["Implementation-Version"]); version != "" {
		props["file.jarVersion"] = version
	}
	for key, value := range attrs {
		if strings.TrimSpace(value) == "" {
			continue
		}
		props["file."+key] = strings.TrimSpace(value)
	}
	return props
}

func parseManifestMainAttributes(data []byte) map[string]string {
	attrs := make(map[string]string)
	var currentKey string
	for _, rawLine := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		line := strings.TrimRight(rawLine, "\r")
		if line == "" {
			break
		}
		if strings.HasPrefix(line, " ") && currentKey != "" {
			attrs[currentKey] += line[1:]
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			currentKey = ""
			continue
		}
		currentKey = strings.TrimSpace(key)
		if currentKey == "" {
			continue
		}
		attrs[currentKey] = strings.TrimSpace(value)
	}
	return attrs
}

func resolveMetadataValue(value string, properties map[string]string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	resolved := value
	for key, replacement := range properties {
		resolved = strings.ReplaceAll(resolved, "${"+key+"}", replacement)
	}
	if strings.Contains(resolved, "${") {
		return ""
	}
	return resolved
}

// --- registry ---

var modMetadataParsers = []ModMetadataParser{
	fabricModParser{},
	neoForgeModParser{},
	forgeModParser{},
}

const (
	loaderFabric       = "fabric"
	loaderNeoForge     = "neoforge"
	loaderForge        = "forge"
	jarJarMetadataPath = "META-INF/jarjar/metadata.json"
	maxNestedJarDepth  = 32
)

// --- scanning ---

func isModJar(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, ".jar") || strings.HasSuffix(lower, ".jar.disabled")
}

func StripJarSuffix(name string) string {
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, ".jar.disabled") {
		return name[:len(name)-len(".jar.disabled")]
	}
	if strings.HasSuffix(lower, ".jar") {
		return name[:len(name)-len(".jar")]
	}
	return name
}

func isJarEnabled(name string) bool {
	return !strings.HasSuffix(strings.ToLower(name), ".jar.disabled")
}

func FileSHA1(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		return ""
	}
	return hex.EncodeToString(h.Sum(nil))
}

func ScanVersionMods(versionDir string, instanceID string, minecraftVersion string, modLoader string, minecraftDir string) []structs.ModInfo {
	modsDir := filepath.Join(versionDir, "mods")
	entries, err := os.ReadDir(modsDir)
	if err != nil {
		return nil
	}

	var allMods []structs.ModInfo
	for _, entry := range entries {
		if entry.IsDir() || !isModJar(entry.Name()) {
			continue
		}

		jarPath := filepath.Join(modsDir, entry.Name())
		hash := FileSHA1(jarPath)
		global.HardlinkIndexAdd(hash, jarPath)
		mods := ParseModJarWithSHA1(jarPath, hash, modLoader)
		enabled := isJarEnabled(entry.Name())
		baseName := StripJarSuffix(entry.Name())

		relPath := entry.Name()
		if minecraftDir != "" {
			if rel, err := filepath.Rel(minecraftDir, jarPath); err == nil {
				relPath = rel
			}
		}

		if len(mods) == 0 {
			continue
		}

		for i := range mods {
			mods[i].FileName = baseName
			mods[i].Path = relPath
			mods[i].SHA1 = hash
			mods[i].Enabled = enabled
			global.UpsertLocalMod(mods[i], instanceID, minecraftVersion, modLoader)
		}
		allMods = append(allMods, mods...)
	}
	return allMods
}

func parseModJar(jarPath string, modLoader string) []structs.ModInfo {
	return ParseModJarWithSHA1(jarPath, FileSHA1(jarPath), modLoader)
}

func ParseModJarWithSHA1(jarPath string, sha1 string, modLoader string) []structs.ModInfo {
	if mods, ok := global.GetJarMetadata(sha1); ok {
		return mods
	}

	r, err := zip.OpenReader(jarPath)
	if err != nil {
		return nil
	}
	defer r.Close()

	mods := ParseModZipReader(&r.Reader, filepath.Base(jarPath), modLoader)
	if len(mods) > 0 {
		global.SetJarMetadata(sha1, mods)
	}
	return mods
}

func ParseModZipReader(r *zip.Reader, sourceName string, modLoader string) []structs.ModInfo {
	ctx := modJarParseContext{
		modLoader: normalizeModLoader(modLoader),
		visited:   make(map[[32]byte]struct{}),
	}
	return uniqueModsByID(ctx.parseZipReader(r, sourceName, 0))
}

type modJarParseContext struct {
	modLoader string
	visited   map[[32]byte]struct{}
}

func (ctx modJarParseContext) parseZipReader(r *zip.Reader, sourceName string, depth int) []structs.ModInfo {
	if depth > maxNestedJarDepth {
		logging.Warn("nested mod jar parse depth limit reached", "sourceName", sourceName, "depth", depth)
		return nil
	}

	metadata := ctx.parseDeclaredMetadata(r, sourceName)
	mods := make([]structs.ModInfo, 0, len(metadata.Mods))
	mods = append(mods, metadata.Mods...)

	for _, nestedPath := range uniqueNestedJarPaths(metadata.NestedJars) {
		mods = append(mods, ctx.parseNestedJar(r, nestedPath, sourceName, depth+1)...)
	}
	return mods
}

func (ctx modJarParseContext) parseDeclaredMetadata(r *zip.Reader, sourceName string) parsedModMetadata {
	var out parsedModMetadata
	for _, parser := range ctx.parsers() {
		if !parser.CanParse(r) {
			continue
		}
		metadata, err := parser.Parse(r)
		if err != nil {
			logging.Warn("parse mod metadata failed", "sourceName", sourceName, "parser", parser.Name(), "error", err)
			continue
		}
		out.Mods = append(out.Mods, metadata.Mods...)
		out.NestedJars = append(out.NestedJars, metadata.NestedJars...)
		if parser.Name() == loaderForge || parser.Name() == loaderNeoForge {
			paths, err := parseJarJarNestedPaths(r)
			if err != nil {
				logging.Warn("parse jarjar metadata failed", "sourceName", sourceName, "parser", parser.Name(), "error", err)
				continue
			}
			out.NestedJars = append(out.NestedJars, paths...)
		}
	}
	return out
}

func (ctx modJarParseContext) parsers() []ModMetadataParser {
	if ctx.modLoader == "" {
		return modMetadataParsers
	}
	for _, parser := range modMetadataParsers {
		if parser.Name() == ctx.modLoader {
			return []ModMetadataParser{parser}
		}
	}
	return modMetadataParsers
}

func (ctx modJarParseContext) parseNestedJar(r *zip.Reader, nestedPath string, sourceName string, depth int) []structs.ModInfo {
	data, err := readZipFile(r, nestedPath)
	if err != nil {
		logging.Warn("read nested mod jar failed", "sourceName", sourceName, "nestedPath", nestedPath, "error", err)
		return nil
	}

	sum := sha256.Sum256(data)
	if _, ok := ctx.visited[sum]; ok {
		return nil
	}
	ctx.visited[sum] = struct{}{}

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		logging.Warn("open nested mod jar failed", "sourceName", sourceName, "nestedPath", nestedPath, "error", err)
		return nil
	}
	return ctx.parseZipReader(zr, sourceName+" > "+nestedPath, depth)
}

func parseJarJarNestedPaths(r *zip.Reader) ([]string, error) {
	if !zipHasFile(r, jarJarMetadataPath) {
		return nil, nil
	}
	data, err := readZipFile(r, jarJarMetadataPath)
	if err != nil {
		return nil, err
	}

	var meta struct {
		Jars []struct {
			Path string `json:"path"`
		} `json:"jars"`
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("parse %s: %w", jarJarMetadataPath, err)
	}

	paths := make([]string, 0, len(meta.Jars))
	for _, jar := range meta.Jars {
		if path := nestedJarPath(jar.Path); path != "" {
			paths = append(paths, path)
		}
	}
	return paths, nil
}

func normalizeModLoader(modLoader string) string {
	switch strings.ToLower(strings.TrimSpace(modLoader)) {
	case loaderFabric:
		return loaderFabric
	case loaderForge:
		return loaderForge
	case loaderNeoForge:
		return loaderNeoForge
	default:
		return ""
	}
}

func declaredModID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" || strings.Contains(id, "${") {
		return ""
	}
	return id
}

func nestedJarPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "/")
	return path
}

func uniqueNestedJarPaths(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		path = nestedJarPath(path)
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		out = append(out, path)
	}
	return out
}

func uniqueModsByID(mods []structs.ModInfo) []structs.ModInfo {
	seen := make(map[string]struct{}, len(mods))
	out := make([]structs.ModInfo, 0, len(mods))
	for _, mod := range mods {
		mod.ID = declaredModID(mod.ID)
		if mod.ID == "" {
			continue
		}
		key := strings.ToLower(mod.ID)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, mod)
	}
	return out
}

// --- zip helpers ---

func zipHasFile(r *zip.Reader, name string) bool {
	for _, f := range r.File {
		if f.Name == name {
			return true
		}
	}
	return false
}

func readZipFile(r *zip.Reader, name string) ([]byte, error) {
	for _, f := range r.File {
		if f.Name == name {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("file not found in archive: %s", name)
}
