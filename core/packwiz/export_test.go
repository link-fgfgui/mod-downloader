package packwiz

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"strings"
	"testing"
)

func TestBuildFilesGeneratesPackwizMetadata(t *testing.T) {
	files, err := BuildFiles(Pack{
		Name:             "Performance Pack",
		MinecraftVersion: "1.21.1",
		Mods: []Mod{
			{
				Name:        "Sodium",
				FileName:    "sodium.jar",
				DownloadURL: "https://cdn.modrinth.com/data/sodium.jar",
				SHA1:        "ABC123",
				Platform:    "modrinth",
				ProjectID:   "AANobbMI",
				VersionID:   "ver1",
			},
			{
				Name:        "Just Enough Items",
				FileName:    "jei.jar",
				DownloadURL: "https://edge.forgecdn.net/files/1/2/jei.jar",
				SHA1:        "def456",
				Platform:    "curseforge",
				ProjectID:   "238222",
				VersionID:   "1234567",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	byPath := filesByPath(files)
	for _, path := range []string{"pack.toml", "index.toml", "mods/just-enough-items.pw.toml", "mods/sodium.pw.toml"} {
		if _, ok := byPath[path]; !ok {
			t.Fatalf("missing %s in %#v", path, byPath)
		}
	}

	pack := string(byPath["pack.toml"])
	if !strings.Contains(pack, `pack-format = "packwiz:1.1.0"`) ||
		!strings.Contains(pack, `minecraft = "1.21.1"`) ||
		!strings.Contains(pack, `hash = "`+sha256Hex(byPath["index.toml"])+`"`) {
		t.Fatalf("unexpected pack.toml:\n%s", pack)
	}

	index := string(byPath["index.toml"])
	if !strings.Contains(index, `file = "mods/sodium.pw.toml"`) ||
		!strings.Contains(index, `hash = "`+sha256Hex(byPath["mods/sodium.pw.toml"])+`"`) ||
		!strings.Contains(index, `metafile = true`) {
		t.Fatalf("unexpected index.toml:\n%s", index)
	}

	sodium := string(byPath["mods/sodium.pw.toml"])
	if !strings.Contains(sodium, `hash = "abc123"`) ||
		!strings.Contains(sodium, `[update.modrinth]`) ||
		!strings.Contains(sodium, `mod-id = "AANobbMI"`) ||
		!strings.Contains(sodium, `version = "ver1"`) {
		t.Fatalf("unexpected sodium metafile:\n%s", sodium)
	}

	jei := string(byPath["mods/just-enough-items.pw.toml"])
	if !strings.Contains(jei, `[update.curseforge]`) ||
		!strings.Contains(jei, `project-id = 238222`) ||
		!strings.Contains(jei, `file-id = 1234567`) {
		t.Fatalf("unexpected curseforge metafile:\n%s", jei)
	}
}

func TestBuildFilesSanitizesAndDeduplicatesMetafileNames(t *testing.T) {
	files, err := BuildFiles(Pack{
		Name:             "Pack",
		MinecraftVersion: "1.20.1",
		Mods: []Mod{
			{Name: "My Mod!", FileName: "a.jar", DownloadURL: "https://example.com/a.jar", SHA1: "a"},
			{Name: "My Mod?", FileName: "b.jar", DownloadURL: "https://example.com/b.jar", SHA1: "b"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	byPath := filesByPath(files)
	if _, ok := byPath["mods/my-mod.pw.toml"]; !ok {
		t.Fatalf("first sanitized path missing: %#v", byPath)
	}
	if _, ok := byPath["mods/my-mod-2.pw.toml"]; !ok {
		t.Fatalf("deduplicated path missing: %#v", byPath)
	}
}

func TestWriteZipContainsOnlyPackwizFiles(t *testing.T) {
	var buf bytes.Buffer
	err := WriteZip(&buf, Pack{
		Name:             "Pack",
		MinecraftVersion: "1.20.1",
		Mods: []Mod{{
			Name:        "Mod",
			FileName:    "mod.jar",
			DownloadURL: "https://example.com/mod.jar",
			SHA1:        "abc",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	paths := make(map[string]bool)
	for _, f := range zr.File {
		paths[f.Name] = true
		if strings.HasSuffix(f.Name, ".jar") {
			t.Fatalf("zip unexpectedly contains jar: %s", f.Name)
		}
		if _, err := readZipFile(f); err != nil {
			t.Fatalf("read %s: %v", f.Name, err)
		}
	}
	for _, path := range []string{"pack.toml", "index.toml", "mods/mod.pw.toml"} {
		if !paths[path] {
			t.Fatalf("missing %s from zip paths %#v", path, paths)
		}
	}
}

func TestBuildFilesValidatesRequiredFields(t *testing.T) {
	_, err := BuildFiles(Pack{Name: "Pack", MinecraftVersion: "1.20.1"})
	if err == nil || !strings.Contains(err.Error(), "at least one mod") {
		t.Fatalf("empty mod error = %v", err)
	}
	_, err = BuildFiles(Pack{
		Name:             "Pack",
		MinecraftVersion: "1.20.1",
		Mods:             []Mod{{Name: "Broken", FileName: "broken.jar", SHA1: "abc"}},
	})
	if err == nil || !strings.Contains(err.Error(), "download URL") {
		t.Fatalf("missing URL error = %v", err)
	}
}

func filesByPath(files []File) map[string][]byte {
	out := make(map[string][]byte, len(files))
	for _, file := range files {
		out[file.Path] = file.Data
	}
	return out
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func readZipFile(file *zip.File) ([]byte, error) {
	rc, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}
