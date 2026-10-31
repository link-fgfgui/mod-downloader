package packwiz

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"path"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/BurntSushi/toml"
)

const (
	PackFormat = "packwiz:1.1.0"
	SideBoth   = "both"
)

type Pack struct {
	Name             string
	MinecraftVersion string
	ModLoader        string
	Mods             []Mod
}

type Mod struct {
	Name        string
	FileName    string
	DownloadURL string
	SHA1        string
	Platform    string
	ProjectID   string
	VersionID   string
}

type packTOML struct {
	Name       string       `toml:"name"`
	PackFormat string       `toml:"pack-format"`
	Index      packIndex    `toml:"index"`
	Versions   packVersions `toml:"versions"`
}

type packIndex struct {
	File       string `toml:"file"`
	HashFormat string `toml:"hash-format"`
	Hash       string `toml:"hash"`
}

type packVersions struct {
	Minecraft string `toml:"minecraft"`
}

type indexTOML struct {
	HashFormat string      `toml:"hash-format"`
	Files      []indexFile `toml:"files"`
}

type indexFile struct {
	File     string `toml:"file"`
	Hash     string `toml:"hash"`
	Metafile bool   `toml:"metafile"`
}

type modTOML struct {
	Name     string      `toml:"name"`
	FileName string      `toml:"filename"`
	Side     string      `toml:"side"`
	Download modDownload `toml:"download"`
	Update   *modUpdate  `toml:"update,omitempty"`
}

type modDownload struct {
	URL        string `toml:"url"`
	HashFormat string `toml:"hash-format"`
	Hash       string `toml:"hash"`
}

type modUpdate struct {
	Modrinth   *modrinthUpdate   `toml:"modrinth,omitempty"`
	CurseForge *curseForgeUpdate `toml:"curseforge,omitempty"`
}

type modrinthUpdate struct {
	ModID   string `toml:"mod-id"`
	Version string `toml:"version"`
}

type curseForgeUpdate struct {
	ProjectID int `toml:"project-id"`
	FileID    int `toml:"file-id"`
}

type File struct {
	Path string
	Data []byte
}

func WriteZip(w io.Writer, pack Pack) error {
	files, err := BuildFiles(pack)
	if err != nil {
		return err
	}

	zw := zip.NewWriter(w)
	for _, file := range files {
		fw, err := zw.Create(file.Path)
		if err != nil {
			_ = zw.Close()
			return err
		}
		if _, err := fw.Write(file.Data); err != nil {
			_ = zw.Close()
			return err
		}
	}
	return zw.Close()
}

func BuildFiles(pack Pack) ([]File, error) {
	pack.Name = strings.TrimSpace(pack.Name)
	pack.MinecraftVersion = strings.TrimSpace(pack.MinecraftVersion)
	if pack.Name == "" {
		return nil, fmt.Errorf("pack name is required")
	}
	if pack.MinecraftVersion == "" {
		return nil, fmt.Errorf("minecraft version is required")
	}
	if len(pack.Mods) == 0 {
		return nil, fmt.Errorf("at least one mod is required")
	}

	modFiles, indexFiles, err := buildModFiles(pack.Mods)
	if err != nil {
		return nil, err
	}
	indexData, err := encodeTOML(indexTOML{
		HashFormat: "sha256",
		Files:      indexFiles,
	})
	if err != nil {
		return nil, err
	}
	packData, err := encodeTOML(packTOML{
		Name:       pack.Name,
		PackFormat: PackFormat,
		Index: packIndex{
			File:       "index.toml",
			HashFormat: "sha256",
			Hash:       hexSHA256(indexData),
		},
		Versions: packVersions{Minecraft: pack.MinecraftVersion},
	})
	if err != nil {
		return nil, err
	}

	files := make([]File, 0, len(modFiles)+2)
	files = append(files, File{Path: "pack.toml", Data: packData})
	files = append(files, File{Path: "index.toml", Data: indexData})
	files = append(files, modFiles...)
	return files, nil
}

func buildModFiles(mods []Mod) ([]File, []indexFile, error) {
	used := make(map[string]int, len(mods))
	files := make([]File, 0, len(mods))
	indexFiles := make([]indexFile, 0, len(mods))
	for _, mod := range mods {
		mod = normalizeMod(mod)
		if err := validateMod(mod); err != nil {
			return nil, nil, err
		}
		filePath := uniqueMetaPath(mod, used)
		data, err := encodeModTOML(mod)
		if err != nil {
			return nil, nil, err
		}
		files = append(files, File{Path: filePath, Data: data})
		indexFiles = append(indexFiles, indexFile{
			File:     filePath,
			Hash:     hexSHA256(data),
			Metafile: true,
		})
	}
	sort.SliceStable(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	sort.SliceStable(indexFiles, func(i, j int) bool { return indexFiles[i].File < indexFiles[j].File })
	return files, indexFiles, nil
}

func normalizeMod(mod Mod) Mod {
	mod.Name = strings.TrimSpace(mod.Name)
	mod.FileName = strings.TrimSpace(mod.FileName)
	mod.DownloadURL = strings.TrimSpace(mod.DownloadURL)
	mod.SHA1 = strings.ToLower(strings.TrimSpace(mod.SHA1))
	mod.Platform = strings.ToLower(strings.TrimSpace(mod.Platform))
	mod.ProjectID = strings.TrimSpace(mod.ProjectID)
	mod.VersionID = strings.TrimSpace(mod.VersionID)
	return mod
}

func validateMod(mod Mod) error {
	name := firstNonEmpty(mod.Name, mod.ProjectID, mod.FileName)
	if name == "" {
		return fmt.Errorf("mod name is required")
	}
	if mod.FileName == "" {
		return fmt.Errorf("%s: filename is required", name)
	}
	if mod.DownloadURL == "" {
		return fmt.Errorf("%s: download URL is required", name)
	}
	if mod.SHA1 == "" {
		return fmt.Errorf("%s: sha1 is required", name)
	}
	return nil
}

func encodeModTOML(mod Mod) ([]byte, error) {
	out := modTOML{
		Name:     firstNonEmpty(mod.Name, mod.ProjectID, mod.FileName),
		FileName: mod.FileName,
		Side:     SideBoth,
		Download: modDownload{
			URL:        mod.DownloadURL,
			HashFormat: "sha1",
			Hash:       mod.SHA1,
		},
		Update: updateMetadata(mod),
	}
	return encodeTOML(out)
}

func updateMetadata(mod Mod) *modUpdate {
	switch mod.Platform {
	case "modrinth":
		if mod.ProjectID == "" || mod.VersionID == "" {
			return nil
		}
		return &modUpdate{Modrinth: &modrinthUpdate{ModID: mod.ProjectID, Version: mod.VersionID}}
	case "curseforge":
		projectID, projectErr := strconv.Atoi(mod.ProjectID)
		fileID, fileErr := strconv.Atoi(mod.VersionID)
		if projectErr != nil || fileErr != nil {
			return nil
		}
		return &modUpdate{CurseForge: &curseForgeUpdate{ProjectID: projectID, FileID: fileID}}
	default:
		return nil
	}
}

func uniqueMetaPath(mod Mod, used map[string]int) string {
	base := safePathSegment(firstNonEmpty(mod.Name, mod.ProjectID, mod.FileName, "mod"))
	count := used[base]
	used[base] = count + 1
	if count > 0 {
		base = fmt.Sprintf("%s-%d", base, count+1)
	}
	return path.Join("mods", base+".pw.toml")
}

func SafeArchiveName(name string) string {
	name = safePathSegment(name)
	if name == "" {
		return "packwiz-export.zip"
	}
	return name + ".zip"
}

func safePathSegment(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(value) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == '_' || r == '-' || r == '.':
			b.WriteRune(r)
			lastDash = false
		case unicode.IsSpace(r):
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-.")
}

func encodeTOML(v any) ([]byte, error) {
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func hexSHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
