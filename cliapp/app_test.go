package cliapp

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"mod-downloader/appcore"
	structs "mod-downloader/structs/minecraft"
)

func TestConfigCommandJSON(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldwd); err != nil {
			t.Fatal(err)
		}
	})
	if err := os.WriteFile("mod-downloader.toml", []byte(`
[keys]
curseforge_api_key = "abcd1234wxyz"

[preferences]
theme = "system"
minecraft_dir = "/tmp/minecraft"
`), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := New(&stdout, &stderr)
	if err := app.RunContext(context.Background(), []string{"mod-downloader-cli", "config", "--json"}); err != nil {
		t.Fatalf("config command failed: %v\nstderr: %s", err, stderr.String())
	}

	var got appcore.SettingsView
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode json: %v\n%s", err, stdout.String())
	}
	if got.Theme != "system" {
		t.Fatalf("theme = %q, want system", got.Theme)
	}
	if !got.HasCurseforgeKey || got.CurseforgeKeyMask != "abcd****wxyz" {
		t.Fatalf("curseforge key view = %#v", got)
	}
}

func TestVersionsCommandJSONWithMinecraftDirOverride(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldwd); err != nil {
			t.Fatal(err)
		}
	})

	root := t.TempDir()
	writeFabricManifest(t, root, "fabric-loader-1.21.1", "1.21.1")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := New(&stdout, &stderr)
	if err := app.RunContext(context.Background(), []string{"mod-downloader-cli", "--minecraft-dir", root, "versions", "--json"}); err != nil {
		t.Fatalf("versions command failed: %v\nstderr: %s", err, stderr.String())
	}

	var versions []structs.VersionInfo
	if err := json.Unmarshal(stdout.Bytes(), &versions); err != nil {
		t.Fatalf("decode json: %v\n%s", err, stdout.String())
	}
	if len(versions) != 1 {
		t.Fatalf("versions len = %d, want 1: %+v", len(versions), versions)
	}
	if versions[0].ID != "fabric-loader-1.21.1" || versions[0].MinecraftVersion != "1.21.1" || versions[0].ModLoader != "fabric" {
		t.Fatalf("version = %#v", versions[0])
	}
}

func writeFabricManifest(t *testing.T, gameDir, versionID, mcVersion string) {
	t.Helper()
	versionDir := filepath.Join(gameDir, "versions", versionID)
	if err := os.MkdirAll(versionDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{
		"name": "` + versionID + `",
		"id": "` + versionID + `",
		"patches": [
			{"id": "game", "version": "` + mcVersion + `"},
			{"id": "fabric", "version": "0.16.0"}
		]
	}`
	if err := os.WriteFile(filepath.Join(versionDir, versionID+".json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
}
