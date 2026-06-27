package minecraft

import (
	"os"
	"path/filepath"
	"testing"

	structs "mod-downloader/structs/minecraft"
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
	if err := os.MkdirAll(filepath.Join(instanceDir, ".minecraft"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !IsPrismInstancesDir(dir) {
		t.Fatalf("IsPrismInstancesDir() = false after adding instance with .minecraft/")
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
	// Without .minecraft/, returns the instance dir itself.
	if got := PrismInstanceGameDir(instanceDir); got != instanceDir {
		t.Fatalf("PrismInstanceGameDir() without .minecraft = %q, want %q", got, instanceDir)
	}
	// With .minecraft/, returns the subfolder.
	gameDir := filepath.Join(instanceDir, ".minecraft")
	if err := os.MkdirAll(gameDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := PrismInstanceGameDir(instanceDir); got != gameDir {
		t.Fatalf("PrismInstanceGameDir() with .minecraft = %q, want %q", got, gameDir)
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
	if err := os.MkdirAll(gameDir, 0o755); err != nil {
		t.Fatal(err)
	}

	got := VersionDirPath(instancesDir, structs.VersionInfo{ID: "MyFabric/fabric-loader-1.21.1"})
	want := filepath.Join(gameDir, "versions", "fabric-loader-1.21.1")
	if got != want {
		t.Fatalf("VersionDirPath() = %q, want %q", got, want)
	}
}

func TestVersionDirPathResolvesPrismInstanceWithoutDotMinecraft(t *testing.T) {
	instancesDir := t.TempDir()
	instanceDir := filepath.Join(instancesDir, "BareInstance")
	// Instance exists but has no .minecraft subfolder (uses instance root as game dir).
	writeMarkerFile(t, filepath.Join(instanceDir, "instance.cfg"))

	got := VersionDirPath(instancesDir, structs.VersionInfo{ID: "BareInstance/fabric-loader-1.21.1"})
	want := filepath.Join(instanceDir, "versions", "fabric-loader-1.21.1")
	if got != want {
		t.Fatalf("VersionDirPath() = %q, want %q", got, want)
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
