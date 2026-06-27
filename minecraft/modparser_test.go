package minecraft

import (
	"archive/zip"
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	structs "mod-downloader/structs/minecraft"
)

func TestParseFabricNestedJarsRecursively(t *testing.T) {
	grandchild := testJar(t, map[string][]byte{
		"fabric.mod.json": []byte(`{
			"schemaVersion": 1,
			"id": "grand-child",
			"version": "1.0.0"
		}`),
	})
	child := testJar(t, map[string][]byte{
		"fabric.mod.json": []byte(`{
			"schemaVersion": 1,
			"id": "child-fabric",
			"version": "1.0.0",
			"jars": [{"file": "META-INF/jars/grandchild.jar"}]
		}`),
		"META-INF/jars/grandchild.jar": grandchild,
	})
	top := testJar(t, map[string][]byte{
		"fabric.mod.json": []byte(`{
			"schemaVersion": 1,
			"id": "top-fabric",
			"version": "1.0.0",
			"provides": ["virtual-provided-id"],
			"jars": [{"file": "META-INF/jars/child.jar"}]
		}`),
		"META-INF/jars/child.jar": child,
	})

	got := modIDs(ParseModZipReader(testZipReader(t, top), "top.jar", "fabric"))
	want := []string{"top-fabric", "child-fabric", "grand-child"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseModZipReader() ids = %#v, want %#v", got, want)
	}
}

func TestParseForgeJarJarRecursivelyAndIgnoresDependencyModIDs(t *testing.T) {
	child := testJar(t, map[string][]byte{
		"META-INF/mods.toml": []byte(`
license="MIT"

[[mods]]
modId="${mod_id}"
displayName="Template Placeholder"

[[mods]]
modId="childforge"
displayName="Child Forge"

[[dependencies.childforge]]
modId="not_current_mod"
mandatory=true
`),
	})
	top := testJar(t, map[string][]byte{
		"META-INF/mods.toml": []byte(`
license="MIT"

[[mods]]
modId="topforge"
displayName="Top Forge"

[[mods]]
modId="jei"
displayName="JEI"

[[dependencies.topforge]]
modId="minecraft"
mandatory=true
`),
		"META-INF/jarjar/metadata.json": []byte(`{
			"jars": [{"path": "META-INF/jarjar/child.jar"}]
		}`),
		"META-INF/jarjar/child.jar": child,
	})

	got := modIDs(ParseModZipReader(testZipReader(t, top), "top.jar", "forge"))
	want := []string{"topforge", "jei", "childforge"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseModZipReader() ids = %#v, want %#v", got, want)
	}
}

func TestParseModZipReaderUsesSpecifiedLoaderOnly(t *testing.T) {
	jar := testJar(t, map[string][]byte{
		"META-INF/mods.toml": []byte(`
license="MIT"

[[mods]]
modId="forgeonly"
`),
	})

	got := modIDs(ParseModZipReader(testZipReader(t, jar), "forge-only.jar", "neoforge"))
	if len(got) != 0 {
		t.Fatalf("ParseModZipReader() ids = %#v, want none", got)
	}
}

func TestParseNeoForgeJarVersionFromManifest(t *testing.T) {
	jar := testJar(t, map[string][]byte{
		"META-INF/neoforge.mods.toml": []byte(`
license="MIT"

[[mods]]
modId="jade"
displayName="Jade"
version="${file.jarVersion}"
description="Minecraft mod shows what you are looking at."
`),
		"META-INF/MANIFEST.MF": []byte("Manifest-Version: 1.0\r\nImplementation-Title: Jade\r\nImplementation-Version: 15.10.5+neoforge\r\n\r\n"),
	})

	got := ParseModZipReader(testZipReader(t, jar), "Jade-1.21.1-NeoForge-15.10.5.jar", "neoforge")
	if len(got) != 1 {
		t.Fatalf("ParseModZipReader() returned %d mods, want 1: %#v", len(got), got)
	}
	if got[0].Version != "15.10.5+neoforge" {
		t.Fatalf("version = %q, want %q", got[0].Version, "15.10.5+neoforge")
	}
}

func TestParseModsTomlDropsUnresolvedPlaceholders(t *testing.T) {
	jar := testJar(t, map[string][]byte{
		"META-INF/mods.toml": []byte(`
license="MIT"

[[mods]]
modId="example"
displayName="${missing.name}"
version="${missing.version}"
`),
	})

	got := ParseModZipReader(testZipReader(t, jar), "example.jar", "forge")
	if len(got) != 1 {
		t.Fatalf("ParseModZipReader() returned %d mods, want 1: %#v", len(got), got)
	}
	if got[0].Name != "" || got[0].Version != "" {
		t.Fatalf("metadata placeholders were not dropped: %#v", got[0])
	}
}

func TestCreateHardLinkOrCopyDoesNotOverwriteExistingTarget(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.jar")
	dst := filepath.Join(dir, "dst.jar")
	if err := os.WriteFile(src, []byte("source"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}
	if err := os.WriteFile(dst, []byte("target"), 0o644); err != nil {
		t.Fatalf("write dst: %v", err)
	}

	err := CreateHardLinkOrCopy(src, dst)
	if !errors.Is(err, os.ErrExist) {
		t.Fatalf("CreateHardLinkOrCopy() error = %v, want os.ErrExist", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if string(got) != "target" {
		t.Fatalf("dst content = %q, want target", got)
	}
}

func modIDs(mods []structs.ModInfo) []string {
	ids := make([]string, 0, len(mods))
	for _, mod := range mods {
		ids = append(ids, mod.ID)
	}
	return ids
}

func testJar(t *testing.T, files map[string][]byte) []byte {
	t.Helper()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, data := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create zip entry %s: %v", name, err)
		}
		if _, err := w.Write(data); err != nil {
			t.Fatalf("write zip entry %s: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buf.Bytes()
}

func testZipReader(t *testing.T, data []byte) *zip.Reader {
	t.Helper()

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("open test zip: %v", err)
	}
	return zr
}
