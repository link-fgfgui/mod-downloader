package appcore

import (
	"archive/zip"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/link-fgfgui/mod-downloader-core/global"
	structs "github.com/link-fgfgui/mod-downloader-core/structs/minecraft"
)

func TestReconcileConnectorVersionRequiresEnabledConnectorAndHostLoader(t *testing.T) {
	tests := []struct {
		name      string
		version   structs.VersionInfo
		available bool
		loader    string
	}{
		{
			name: "enabled neoforge connector",
			version: structs.VersionInfo{
				ModLoader: "NeoForge",
				Mods:      []structs.ModInfo{{ID: "Connector", Enabled: true}},
			},
			available: true,
			loader:    "neoforge",
		},
		{
			name: "enabled forge connector",
			version: structs.VersionInfo{
				ModLoader: "forge",
				Mods:      []structs.ModInfo{{ID: "connector", Enabled: true}},
			},
			available: true,
			loader:    "forge",
		},
		{
			name: "disabled connector",
			version: structs.VersionInfo{
				ModLoader:        "neoforge",
				ConnectorVirtual: true,
				Mods:             []structs.ModInfo{{ID: "connector", Enabled: false}},
			},
			loader: "neoforge",
		},
		{
			name: "fabric is not a host",
			version: structs.VersionInfo{
				ModLoader: "fabric",
				Mods:      []structs.ModInfo{{ID: "connector", Enabled: true}},
			},
			loader: "fabric",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := reconcileConnectorVersion(test.version)
			if got.ConnectorAvailable != test.available || got.ModLoader != test.loader {
				t.Fatalf("reconciled version = %#v", got)
			}
			if !test.available && got.ConnectorVirtual {
				t.Fatalf("unavailable connector retained virtual state: %#v", got)
			}
		})
	}
}

func TestToggleConnectorLoaderPreservesPhysicalInstance(t *testing.T) {
	dir := t.TempDir()
	base := reconcileConnectorVersion(structs.VersionInfo{
		ID:               "instance-id",
		Name:             "Instance Name",
		MinecraftVersion: "1.21.1",
		ModLoader:        "neoforge",
		Mods:             []structs.ModInfo{{ID: "connector", Enabled: true}},
	})
	global.SetMinecraftDir(dir)
	global.SetVersionsForDir(dir, []structs.VersionInfo{base})
	global.SetSelectedVersion(base)
	t.Cleanup(func() {
		global.SetMinecraftDir("")
		global.InvalidateVersions()
	})

	var events []Event
	svc := New(Options{OnEvent: func(event Event) { events = append(events, event) }})
	virtual, err := svc.ToggleConnectorLoader()
	if err != nil {
		t.Fatal(err)
	}
	if virtual.ModLoader != "fabric" || !virtual.ConnectorVirtual || virtual.ActualModLoader != "neoforge" {
		t.Fatalf("virtual version = %#v", virtual)
	}
	if virtual.ID != base.ID || virtual.Name != base.Name || virtual.MinecraftVersion != base.MinecraftVersion || !reflect.DeepEqual(virtual.Mods, base.Mods) {
		t.Fatalf("toggle changed physical identity: base=%#v virtual=%#v", base, virtual)
	}

	real, err := svc.ToggleConnectorLoader()
	if err != nil {
		t.Fatal(err)
	}
	if real.ModLoader != "neoforge" || real.ConnectorVirtual {
		t.Fatalf("real version = %#v", real)
	}
	if len(events) != 2 || events[0].Kind != EventSelectedVersionChanged || events[1].Kind != EventSelectedVersionChanged {
		t.Fatalf("events = %#v, want two selected-version events", events)
	}
	if selected := global.GetSelectedVersion(); selected.ModLoader != "neoforge" || selected.ConnectorVirtual {
		t.Fatalf("selected version = %#v", selected)
	}
}

func TestToggleConnectorLoaderRejectsDisabledConnector(t *testing.T) {
	dir := t.TempDir()
	version := reconcileConnectorVersion(structs.VersionInfo{
		ID:        "instance-id",
		ModLoader: "forge",
		Mods:      []structs.ModInfo{{ID: "connector", Enabled: false}},
	})
	global.SetMinecraftDir(dir)
	global.SetVersionsForDir(dir, []structs.VersionInfo{version})
	global.SetSelectedVersion(version)
	t.Cleanup(func() {
		global.SetMinecraftDir("")
		global.InvalidateVersions()
	})

	if _, err := New(Options{}).ToggleConnectorLoader(); err == nil {
		t.Fatal("ToggleConnectorLoader() returned nil error for disabled connector")
	}
}

func TestSelectVersionDetectsConnectorAndResetsVirtualView(t *testing.T) {
	root := t.TempDir()
	modsDir := filepath.Join(root, "versions", "test", "mods")
	if err := os.MkdirAll(modsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	connectorPath := filepath.Join(modsDir, "connector.jar")
	writeNeoForgeConnectorJar(t, connectorPath)

	version := structs.VersionInfo{
		ID:               "test",
		Name:             "Test",
		MinecraftVersion: "1.21.1",
		ModLoader:        "neoforge",
	}
	global.SetMinecraftDir(root)
	global.SetVersionsForDir(root, []structs.VersionInfo{version})
	t.Cleanup(func() {
		global.SetMinecraftDir("")
		global.InvalidateVersions()
		global.ClearLocalMods()
		global.ClearJarMetadata()
	})

	svc := New(Options{})
	t.Cleanup(svc.stopModsWatcher)
	t.Cleanup(svc.stopServiceBackground)
	selected, err := svc.SelectVersion("test")
	if err != nil {
		t.Fatal(err)
	}
	if !selected.ConnectorAvailable || selected.ModLoader != "neoforge" || selected.ActualModLoader != "neoforge" {
		t.Fatalf("selected connector version = %#v", selected)
	}
	if len(selected.Mods) != 1 || !reflect.DeepEqual(selected.Mods[0].Loaders, []string{"neoforge"}) {
		t.Fatalf("selected mods = %#v", selected.Mods)
	}

	if _, err := svc.ToggleConnectorLoader(); err != nil {
		t.Fatal(err)
	}
	selected, err = svc.SelectVersion("test")
	if err != nil {
		t.Fatal(err)
	}
	if selected.ModLoader != "neoforge" || selected.ConnectorVirtual || !selected.ConnectorAvailable {
		t.Fatalf("reselected version = %#v, want real Connector view", selected)
	}

	if err := os.Rename(connectorPath, connectorPath+".disabled"); err != nil {
		t.Fatal(err)
	}
	selected = svc.RefreshSelectedVersionMods()
	if selected.ConnectorAvailable || selected.ConnectorVirtual || selected.ModLoader != "neoforge" {
		t.Fatalf("disabled connector version = %#v", selected)
	}
}

func writeNeoForgeConnectorJar(t *testing.T, path string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create("META-INF/neoforge.mods.toml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("license=\"MIT\"\n[[mods]]\nmodId=\"connector\"\ndisplayName=\"Connector\"\nversion=\"1.0.0\"\n")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}
