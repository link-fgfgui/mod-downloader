// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

package minecraft

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	structs "github.com/link-fgfgui/mod-downloader/core/structs/minecraft"
)

func writeMetadataJar(t *testing.T, dir, name string, entries map[string]string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for entryName, content := range entries {
		w, err := zw.Create(entryName)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func parseMetadataJar(t *testing.T, path string) []structs.ModInfo {
	t.Helper()
	r, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	mods := ParseLocalModZipReader(&r.Reader, filepath.Base(path), "")
	if len(mods) == 0 {
		t.Fatalf("no metadata parsed from %s", filepath.Base(path))
	}
	return mods
}

func TestParseFabricRichMetadata(t *testing.T) {
	dir := t.TempDir()
	entries := map[string]string{
		"fabric.mod.json": `{"schemaVersion":1,"id":"iris","name":"Iris","version":"1.8.14",
		"description":"shaders","authors":[{"name":"coderbot","contact":{"email":"a@b.c"}},{"name":"IMS212"}],
		"environment":"client","license":"LGPL-3.0-only",
		"depends":{"fabricloader":">=0.12.3","minecraft":["1.21.1"],"sodium":["0.8.x"]},
		"breaks":{"embeddium":"*","originsenhanced":"<=1.3.2"},
		"provides":["iris"]}`,
	}
	mods := parseMetadataJar(t, writeMetadataJar(t, dir, "iris.jar", entries))
	if len(mods) != 1 {
		t.Fatalf("mods = %d, want 1", len(mods))
	}
	mod := mods[0]
	if mod.ID != "iris" || mod.Name != "Iris" || mod.Environment != "client" || mod.License != "LGPL-3.0-only" {
		t.Fatalf("metadata = %#v", mod)
	}
	if len(mod.Authors) != 2 || mod.Authors[0] != "coderbot" || mod.Authors[1] != "IMS212" {
		t.Fatalf("authors = %#v", mod.Authors)
	}
	if mod.MinecraftVersion != "1.21.1" {
		t.Fatalf("minecraft version = %q, want 1.21.1", mod.MinecraftVersion)
	}
	if len(mod.Loaders) != 1 || mod.Loaders[0] != loaderFabric {
		t.Fatalf("loaders = %#v", mod.Loaders)
	}
	deps := map[string]string{}
	for _, dep := range mod.Dependencies {
		deps[dep.ModID] = dep.Version
	}
	if deps["sodium"] != "0.8.x" {
		t.Fatalf("depends = %#v (loader-provided ids must be filtered)", mod.Dependencies)
	}
	if len(mod.Breaks) != 2 || len(mod.Provides) != 1 || mod.Provides[0] != "iris" {
		t.Fatalf("breaks/provides = %#v / %#v", mod.Breaks, mod.Provides)
	}
}

func TestParseFabricMultiValueRange(t *testing.T) {
	dir := t.TempDir()
	entries := map[string]string{
		"fabric.mod.json": `{"schemaVersion":1,"id":"sodium","version":"0.8.13",
		"authors":["JellySquid"],"depends":{"minecraft":["1.21","1.21.1"]}}`,
	}
	mod := parseMetadataJar(t, writeMetadataJar(t, dir, "sodium.jar", entries))[0]
	if mod.MinecraftVersion != "[1.21,1.21.1]" {
		t.Fatalf("minecraft version = %q", mod.MinecraftVersion)
	}
	if allowed, known := VersionRangeAllows(mod.MinecraftVersion, "1.21.1"); !known || !allowed {
		t.Fatalf("range %q rejected 1.21.1 (allowed=%v known=%v)", mod.MinecraftVersion, allowed, known)
	}
}

func TestParseQuiltMetadata(t *testing.T) {
	dir := t.TempDir()
	quilt := `{
		"schema_version": 1,
		"quilt_loader": {
			"id": "qmod",
			"version": "1.0.0",
			"metadata": {"name": "Q Mod", "description": "d", "license": "MIT",
				"contributors": {"alice": "Author"}},
			"depends": [
				{"id": "quilt_loader", "versions": ">=0.17.0"},
				{"id": "minecraft", "versions": ">=1.20"},
				{"id": "optional-lib", "versions": "*", "optional": true}
			],
			"breaks": [{"id": "bad-lib", "versions": "*"}]
		}
	}`
	mod := parseMetadataJar(t, writeMetadataJar(t, dir, "qmod.jar", map[string]string{"quilt.mod.json": quilt}))[0]
	if mod.ID != "qmod" || mod.Name != "Q Mod" || mod.License != "MIT" {
		t.Fatalf("metadata = %#v", mod)
	}
	if mod.MinecraftVersion != ">=1.20" {
		t.Fatalf("minecraft version = %q", mod.MinecraftVersion)
	}
	if len(mod.Loaders) != 1 || mod.Loaders[0] != loaderQuilt {
		t.Fatalf("loaders = %#v", mod.Loaders)
	}
	if len(mod.Dependencies) != 0 {
		t.Fatalf("depends = %#v, want none (loader ids filtered)", mod.Dependencies)
	}
	if len(mod.Recommends) != 1 || mod.Recommends[0].ModID != "optional-lib" {
		t.Fatalf("recommends = %#v", mod.Recommends)
	}
	if len(mod.Breaks) != 1 || mod.Breaks[0].ModID != "bad-lib" {
		t.Fatalf("breaks = %#v", mod.Breaks)
	}
	if len(mod.Authors) != 1 || mod.Authors[0] != "alice (Author)" {
		t.Fatalf("authors = %#v", mod.Authors)
	}
}

func TestParseModsTomlRichMetadata(t *testing.T) {
	dir := t.TempDir()
	toml := `
modLoader="javafml"
loaderVersion="[47,)"
license="LGPL-3.0"

[[mods]]
modId="examplemod"
version="1.2.3"
displayName="Example Mod"
description="An example"
authors="Alice, Bob"

[[dependencies.examplemod]]
    modId="forge"
    mandatory=true
    versionRange="[47,)"
    side="BOTH"
[[dependencies.examplemod]]
    modId="minecraft"
    mandatory=true
    versionRange="[1.20,1.22)"
    side="BOTH"
[[dependencies.examplemod]]
    modId="jei"
    mandatory=false
    versionRange="[15,)"
    side="CLIENT"
[[dependencies.examplemod]]
    modId="optifine"
    mandatory=false
    type="incompatible"
    versionRange="*"
`
	mod := parseMetadataJar(t, writeMetadataJar(t, dir, "example.jar", map[string]string{"META-INF/mods.toml": toml}))[0]
	if mod.ID != "examplemod" || mod.Version != "1.2.3" || mod.License != "LGPL-3.0" {
		t.Fatalf("metadata = %#v", mod)
	}
	if len(mod.Authors) != 2 || mod.Authors[0] != "Alice" || mod.Authors[1] != "Bob" {
		t.Fatalf("authors = %#v", mod.Authors)
	}
	if mod.MinecraftVersion != "[1.20,1.22)" {
		t.Fatalf("minecraft version = %q", mod.MinecraftVersion)
	}
	if len(mod.Loaders) != 1 || mod.Loaders[0] != loaderForge {
		t.Fatalf("loaders = %#v", mod.Loaders)
	}
	if len(mod.Dependencies) != 0 {
		t.Fatalf("depends = %#v, want none (forge/minecraft filtered)", mod.Dependencies)
	}
	if len(mod.Recommends) != 1 || mod.Recommends[0].ModID != "jei" || mod.Recommends[0].Version != "[15,)" {
		t.Fatalf("recommends = %#v", mod.Recommends)
	}
	if len(mod.Breaks) != 1 || mod.Breaks[0].ModID != "optifine" {
		t.Fatalf("breaks = %#v", mod.Breaks)
	}
}

func TestParseMcmodInfoMetadata(t *testing.T) {
	dir := t.TempDir()
	info := `[{"modid":"legacy","name":"Legacy","version":"1.0","description":"d",
		"authorList":["Alice"],"license":"MIT","mcversion":"1.12.2",
		"dependencies":["forge"],"requiredMods":["other@[1,)"]}]`
	mod := parseMetadataJar(t, writeMetadataJar(t, dir, "legacy.jar", map[string]string{"mcmod.info": info}))[0]
	if mod.ID != "legacy" || mod.Version != "1.0" || mod.License != "MIT" {
		t.Fatalf("metadata = %#v", mod)
	}
	if mod.MinecraftVersion != "1.12.2" {
		t.Fatalf("minecraft version = %q", mod.MinecraftVersion)
	}
	if len(mod.Authors) != 1 || mod.Authors[0] != "Alice" {
		t.Fatalf("authors = %#v", mod.Authors)
	}
	if len(mod.Loaders) != 1 || mod.Loaders[0] != loaderForge {
		t.Fatalf("loaders = %#v", mod.Loaders)
	}
	if len(mod.Dependencies) != 1 || mod.Dependencies[0].ModID != "other" || mod.Dependencies[0].Version != "[1,)" {
		t.Fatalf("depends = %#v", mod.Dependencies)
	}
}

// TestFabricWinsOverQuiltForDualMetadata locks the parser precedence: the
// first parser with an id owns the fields, later ones only merge loaders.
func TestFabricWinsOverQuiltForDualMetadata(t *testing.T) {
	dir := t.TempDir()
	entries := map[string]string{
		"fabric.mod.json": `{"schemaVersion":1,"id":"iris","name":"Iris","version":"1.8.14"}`,
		"quilt.mod.json":  `{"quilt_loader":{"id":"iris","version":"9.9.9","metadata":{"name":"Quilt Iris"}}}`,
	}
	mods := parseMetadataJar(t, writeMetadataJar(t, dir, "iris.jar", entries))
	if len(mods) != 1 {
		t.Fatalf("mods = %d, want 1", len(mods))
	}
	if mods[0].Name != "Iris" || mods[0].Version != "1.8.14" {
		t.Fatalf("fabric metadata must win: %#v", mods[0])
	}
	if len(mods[0].Loaders) != 2 {
		t.Fatalf("loaders = %#v, want fabric+quilt", mods[0].Loaders)
	}
}
