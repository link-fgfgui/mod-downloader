package minecraft

import (
	"os"
	"path/filepath"
	"testing"

	structs "github.com/link-fgfgui/mod-downloader-core/structs/minecraft"
)

func writeMarkerFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestIsPrismInstancesDirRejectsEmptyAndNonInstanceDirs(t *testing.T) {
	if IsPrismInstancesDir("") {
		t.Fatalf("IsPrismInstancesDir(\"\") = true, want false")
	}

	dir := t.TempDir()
	if IsPrismInstancesDir(dir) {
		t.Fatalf("IsPrismInstancesDir() = true on empty dir")
	}

	// A non-instance subfolder should not trigger detection.
	if err := os.MkdirAll(filepath.Join(dir, "random"), 0o755); err != nil {
		t.Fatal(err)
	}
	if IsPrismInstancesDir(dir) {
		t.Fatalf("IsPrismInstancesDir() = true with non-instance subfolder")
	}

	// A marker file that is a directory should not trigger detection.
	if err := os.MkdirAll(filepath.Join(dir, "FakeInstance", "mmc-pack.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	if IsPrismInstancesDir(dir) {
		t.Fatalf("IsPrismInstancesDir() = true when marker is a directory")
	}
}

func TestIsPrismInstancesDirDetectsDotMinecraftSubfolder(t *testing.T) {
	dir := t.TempDir()
	instanceDir := filepath.Join(dir, "MyFabric")
	for _, gameDirName := range prismInstanceGameDirNames() {
		if err := os.RemoveAll(instanceDir); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(instanceDir, gameDirName), 0o755); err != nil {
			t.Fatal(err)
		}
		if !IsPrismInstancesDir(dir) {
			t.Fatalf("IsPrismInstancesDir() = false after adding instance with %s/", gameDirName)
		}
	}
}

func TestIsPrismInstancesDirDetectsMarkerFiles(t *testing.T) {
	for _, marker := range prismInstanceConfigFiles {
		t.Run(marker, func(t *testing.T) {
			dir := t.TempDir()
			writeMarkerFile(t, filepath.Join(dir, "MyFabric", marker))
			if !IsPrismInstancesDir(dir) {
				t.Fatalf("IsPrismInstancesDir() = false with marker %q", marker)
			}
		})
	}
}

func TestPrismInstanceGameDirPrefersDotMinecraft(t *testing.T) {
	instanceDir := t.TempDir()
	// Without a game-dir subfolder, returns the instance dir itself.
	if got := PrismInstanceGameDir(instanceDir); got != instanceDir {
		t.Fatalf("PrismInstanceGameDir() without game dir = %q, want %q", got, instanceDir)
	}
	// Prism commonly uses minecraft/.
	minecraftDir := filepath.Join(instanceDir, "minecraft")
	if err := os.MkdirAll(minecraftDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := PrismInstanceGameDir(instanceDir); got != minecraftDir {
		t.Fatalf("PrismInstanceGameDir() with minecraft = %q, want %q", got, minecraftDir)
	}
	// If both exist, prefer .minecraft for backward compatibility.
	dotMinecraftDir := filepath.Join(instanceDir, ".minecraft")
	if err := os.MkdirAll(dotMinecraftDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := PrismInstanceGameDir(instanceDir); got != dotMinecraftDir {
		t.Fatalf("PrismInstanceGameDir() with both game dirs = %q, want %q", got, dotMinecraftDir)
	}
}

func TestPrismVersionInfoFromPack(t *testing.T) {
	instanceDir := t.TempDir()
	writeMarkerFile(t, filepath.Join(instanceDir, "mmc-pack.json"))
	if err := os.WriteFile(filepath.Join(instanceDir, "mmc-pack.json"), []byte(`{
		"components": [
			{"uid": "org.lwjgl3", "version": "3.3.3"},
			{"uid": "net.minecraft", "version": "1.21.1"},
			{"uid": "net.neoforged", "version": "21.1.233"}
		],
		"formatVersion": 1
	}`), 0o644); err != nil {
		t.Fatal(err)
	}

	got, ok := prismVersionInfoFromPack(instanceDir)
	if !ok {
		t.Fatal("prismVersionInfoFromPack() ok = false, want true")
	}
	if got.ID != "neoforge-1.21.1" || got.MinecraftVersion != "1.21.1" || got.ModLoader != "neoforge" {
		t.Fatalf("prismVersionInfoFromPack() = %#v", got)
	}
}

func TestMakeAndSplitPrismVersionID(t *testing.T) {
	got := MakePrismVersionID("MyFabric", "fabric-loader-1.21.1")
	if got != "MyFabric/fabric-loader-1.21.1" {
		t.Fatalf("MakePrismVersionID() = %q", got)
	}

	instance, folder, ok := SplitPrismVersionID(got)
	if !ok || instance != "MyFabric" || folder != "fabric-loader-1.21.1" {
		t.Fatalf("SplitPrismVersionID() = %q, %q, %v", instance, folder, ok)
	}
}

func TestSplitPrismVersionIDRejectsSimpleIDs(t *testing.T) {
	cases := []string{"", "simple", "/leading", "trailing/"}
	for _, in := range cases {
		if _, _, ok := SplitPrismVersionID(in); ok {
			t.Fatalf("SplitPrismVersionID(%q) = ok, want false", in)
		}
	}
}

func TestVersionFolderNameHandlesCompositeAndSimple(t *testing.T) {
	cases := []struct {
		name    string
		version structs.VersionInfo
		want    string
	}{
		{name: "composite id", version: structs.VersionInfo{ID: "MyFabric/fabric-loader-1.21.1"}, want: "fabric-loader-1.21.1"},
		{name: "simple id", version: structs.VersionInfo{ID: "fabric-loader-1.21.1"}, want: "fabric-loader-1.21.1"},
		{name: "fallback to name", version: structs.VersionInfo{Name: "1.21.1"}, want: "1.21.1"},
		{name: "empty", version: structs.VersionInfo{}, want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := VersionFolderName(tc.version); got != tc.want {
				t.Fatalf("VersionFolderName() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestVersionDirPathResolvesPrismInstancesFolder(t *testing.T) {
	instancesDir := t.TempDir()
	instanceDir := filepath.Join(instancesDir, "MyFabric")
	gameDir := filepath.Join(instanceDir, ".minecraft")
	versionDir := filepath.Join(gameDir, "versions", "fabric-loader-1.21.1")
	if err := os.MkdirAll(versionDir, 0o755); err != nil {
		t.Fatal(err)
	}

	got := VersionDirPath(instancesDir, structs.VersionInfo{ID: "MyFabric/fabric-loader-1.21.1"})
	want := versionDir
	if got != want {
		t.Fatalf("VersionDirPath() = %q, want %q", got, want)
	}
}

func TestVersionDirPathResolvesPrismInstanceWithoutDotMinecraft(t *testing.T) {
	instancesDir := t.TempDir()
	instanceDir := filepath.Join(instancesDir, "BareInstance")
	// Instance exists but has no .minecraft subfolder (uses instance root as game dir).
	writeMarkerFile(t, filepath.Join(instanceDir, "instance.cfg"))
	versionDir := filepath.Join(instanceDir, "versions", "fabric-loader-1.21.1")
	if err := os.MkdirAll(versionDir, 0o755); err != nil {
		t.Fatal(err)
	}

	got := VersionDirPath(instancesDir, structs.VersionInfo{ID: "BareInstance/fabric-loader-1.21.1"})
	want := versionDir
	if got != want {
		t.Fatalf("VersionDirPath() = %q, want %q", got, want)
	}
}

func TestVersionDirPathResolvesPrismGameDirWithoutVersionsFolder(t *testing.T) {
	instancesDir := t.TempDir()
	instanceDir := filepath.Join(instancesDir, "MyFabric")
	gameDir := filepath.Join(instanceDir, "minecraft")
	if err := os.MkdirAll(gameDir, 0o755); err != nil {
		t.Fatal(err)
	}

	got := VersionDirPath(instancesDir, structs.VersionInfo{ID: "MyFabric/fabric-loader-1.21.1"})
	if got != gameDir {
		t.Fatalf("VersionDirPath() = %q, want %q", got, gameDir)
	}
}

func TestVersionDirPathResolvesStandardMinecraftDir(t *testing.T) {
	// User selected a regular .minecraft folder — no Prism markers, no composite ID.
	mcDir := t.TempDir()
	got := VersionDirPath(mcDir, structs.VersionInfo{ID: "1.21.1"})
	want := filepath.Join(mcDir, "versions", "1.21.1")
	if got != want {
		t.Fatalf("VersionDirPath() = %q, want %q", got, want)
	}
}

func TestVersionDirPathReturnsEmptyOnMissingInputs(t *testing.T) {
	if got := VersionDirPath("", structs.VersionInfo{ID: "1.21.1"}); got != "" {
		t.Fatalf("VersionDirPath() with empty mcDir = %q, want empty", got)
	}
	if got := VersionDirPath(t.TempDir(), structs.VersionInfo{}); got != "" {
		t.Fatalf("VersionDirPath() with empty version = %q, want empty", got)
	}
}
