package minecraft

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseClassStringsResolvesStringConstants(t *testing.T) {
	data := testClassWithStrings(t, []string{"first", "second"}, true)

	got, err := ParseClassStrings(data)
	if err != nil {
		t.Fatalf("ParseClassStrings() error = %v", err)
	}
	want := []string{"first", "second"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseClassStrings() = %#v, want %#v", got, want)
	}
}

func TestParseClassStringsDecodesModifiedUTF8NullAndSupplementaryRunes(t *testing.T) {
	data := testClassWithRawString(t, []byte{
		'a',
		0xC0, 0x80,
		0xED, 0xA0, 0xBD,
		0xED, 0xB8, 0x80,
	})

	got, err := ParseClassStrings(data)
	if err != nil {
		t.Fatalf("ParseClassStrings() error = %v", err)
	}
	want := []string{"a\x00😀"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseClassStrings() = %#v, want %#v", got, want)
	}
}

func TestParseClassStringsRejectsInvalidModifiedUTF8(t *testing.T) {
	tests := []struct {
		name string
		raw  []byte
	}{
		{name: "bare null", raw: []byte{'a', 0x00}},
		{name: "invalid continuation", raw: []byte{0xC2, 'a'}},
		{name: "truncated three byte", raw: []byte{0xE0, 0x80}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ParseClassStrings(testClassWithRawString(t, tt.raw)); err == nil {
				t.Fatal("ParseClassStrings() error = nil, want error")
			}
		})
	}
}

func TestExtractClientMinecraftVersion(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want string
	}{
		{name: "beta", in: []string{"Minecraft Minecraft Beta 1.7.3"}, want: "b1.7.3"},
		{name: "alpha", in: []string{"Minecraft Minecraft Alpha v1.2.6"}, want: "a1.2.6"},
		{name: "plain", in: []string{"Minecraft Minecraft 1.2.6"}, want: "1.2.6"},
		{name: "rc rejected", in: []string{"Minecraft Minecraft RC2"}, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtractClientMinecraftVersion(tt.in); got != tt.want {
				t.Fatalf("ExtractClientMinecraftVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractServerMinecraftVersion(t *testing.T) {
	got := ExtractServerMinecraftVersion([]string{
		"Server startup",
		"noise",
		"Beta 1.7.3",
		"Can't keep up! Did the system time change, or is the server overloaded?",
		"ignored 123",
	})
	if got != "Beta 1.7.3" {
		t.Fatalf("ExtractServerMinecraftVersion() = %q, want %q", got, "Beta 1.7.3")
	}

	if got := ExtractServerMinecraftVersion([]string{"version 1.2.3"}); got != "" {
		t.Fatalf("ExtractServerMinecraftVersion() without marker = %q, want empty", got)
	}

	if got := ExtractServerMinecraftVersion([]string{
		"Beta 1.7.3",
		"noise",
		"noise",
		"noise",
		"noise",
		"noise",
		"noise",
		"noise",
		"noise",
		"Can't keep up! Did the system time change, or is the server overloaded?",
	}); got != "" {
		t.Fatalf("ExtractServerMinecraftVersion() distant version = %q, want empty", got)
	}

	if got := ExtractServerMinecraftVersion([]string{
		"Java 8",
		"Loading 100%",
		"Beta 1.7.3",
		"Can't keep up! Did the system time change, or is the server overloaded?",
	}); got != "Beta 1.7.3" {
		t.Fatalf("ExtractServerMinecraftVersion() = %q, want Beta 1.7.3", got)
	}
}

func TestDetectMinecraftVersionFromZipPrefersOfficialVersionJSON(t *testing.T) {
	jar := testJar(t, map[string][]byte{
		"version.json": []byte(`{
			"id": "1.21.1",
			"name": "1.21.1",
			"world_version": 3955,
			"series_id": "main",
			"protocol_version": 767,
			"stable": true
		}`),
		clientMinecraftClass: testClassWithStrings(t, []string{"Minecraft Minecraft Beta 1.7.3"}, false),
	})

	got, ok := DetectMinecraftVersionFromZip(testZipReader(t, jar))
	if !ok || got != "1.21.1" {
		t.Fatalf("DetectMinecraftVersionFromZip() = %q, %v; want 1.21.1, true", got, ok)
	}
}

func TestDetectMinecraftVersionFromZipIgnoresOfficialVersionJSONWithoutID(t *testing.T) {
	jar := testJar(t, map[string][]byte{
		"version.json":       []byte(`{"name":"1.20.1"}`),
		clientMinecraftClass: testClassWithStrings(t, []string{"Minecraft Minecraft Beta 1.7.3"}, false),
	})

	got, ok := DetectMinecraftVersionFromZip(testZipReader(t, jar))
	if !ok || got != "b1.7.3" {
		t.Fatalf("DetectMinecraftVersionFromZip() = %q, %v; want b1.7.3, true", got, ok)
	}
}

func TestDetectMinecraftVersionFromZipUsesClientThenServer(t *testing.T) {
	jar := testJar(t, map[string][]byte{
		clientMinecraftClass: testClassWithStrings(t, []string{"Minecraft Minecraft Beta 1.7.3"}, false),
		serverMinecraftClass: testClassWithStrings(t, []string{
			"Alpha 1.0.17",
			"Can't keep up! Did the system time change, or is the server overloaded?",
		}, false),
	})

	got, ok := DetectMinecraftVersionFromZip(testZipReader(t, jar))
	if !ok || got != "b1.7.3" {
		t.Fatalf("DetectMinecraftVersionFromZip() = %q, %v; want b1.7.3, true", got, ok)
	}
}

func TestCheckManifestFallsBackToJarVersion(t *testing.T) {
	dir := t.TempDir()
	versionDir := filepath.Join(dir, "b1.7.3")
	if err := os.Mkdir(versionDir, 0o755); err != nil {
		t.Fatal(err)
	}
	jsonPath := filepath.Join(versionDir, "b1.7.3.json")
	if err := os.WriteFile(jsonPath, []byte(`{"id":"b1.7.3"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	jar := testJar(t, map[string][]byte{
		clientMinecraftClass: testClassWithStrings(t, []string{"Minecraft Minecraft Beta 1.7.3"}, false),
	})
	if err := os.WriteFile(filepath.Join(versionDir, "b1.7.3.jar"), jar, 0o644); err != nil {
		t.Fatal(err)
	}

	got, ok := CheckManifest(jsonPath)
	if !ok {
		t.Fatal("CheckManifest() ok = false, want true")
	}
	if got.ID != "b1.7.3" || got.Name != "b1.7.3" || got.MinecraftVersion != "b1.7.3" || got.ModLoader != "vanilla" {
		t.Fatalf("CheckManifest() = %#v", got)
	}
}

func TestCheckManifestPrefersJarVersionOverBareManifestID(t *testing.T) {
	dir := t.TempDir()
	versionDir := filepath.Join(dir, "custom-instance")
	if err := os.Mkdir(versionDir, 0o755); err != nil {
		t.Fatal(err)
	}
	jsonPath := filepath.Join(versionDir, "custom-instance.json")
	if err := os.WriteFile(jsonPath, []byte(`{"id":"custom-instance"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	jar := testJar(t, map[string][]byte{
		clientMinecraftClass: testClassWithStrings(t, []string{"Minecraft Minecraft Alpha v1.2.6"}, false),
	})
	if err := os.WriteFile(filepath.Join(versionDir, "custom-instance.jar"), jar, 0o644); err != nil {
		t.Fatal(err)
	}

	got, ok := CheckManifest(jsonPath)
	if !ok {
		t.Fatal("CheckManifest() ok = false, want true")
	}
	if got.ID != "custom-instance" || got.MinecraftVersion != "a1.2.6" || got.ModLoader != "vanilla" {
		t.Fatalf("CheckManifest() = %#v", got)
	}
}

func TestCheckManifestUsesPatchesForLoaderInstances(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "fabric-1.21.1.json")
	if err := os.WriteFile(jsonPath, []byte(`{
		"name": "Fabric 1.21.1",
		"id": "fabric-1.21.1",
		"patches": [
			{"id": "game", "version": "1.21.1"},
			{"id": "fabric", "version": "0.16.0"}
		]
	}`), 0o644); err != nil {
		t.Fatal(err)
	}

	got, ok := CheckManifest(jsonPath)
	if !ok {
		t.Fatal("CheckManifest() ok = false, want true")
	}
	if got.ID != "fabric-1.21.1" || got.Name != "Fabric 1.21.1" || got.MinecraftVersion != "1.21.1" || got.ModLoader != "fabric" {
		t.Fatalf("CheckManifest() = %#v", got)
	}
}

func testClassWithStrings(t *testing.T, strings []string, includeLong bool) []byte {
	t.Helper()

	type entry struct {
		tag  byte
		data []byte
	}
	entries := make([]entry, 0, len(strings)*2)
	if includeLong {
		entries = append(entries, entry{tag: 5, data: make([]byte, 8)})
	}
	poolIndex := 1
	if includeLong {
		poolIndex += 2
	}
	for _, s := range strings {
		var utf bytes.Buffer
		if err := binary.Write(&utf, binary.BigEndian, uint16(len(s))); err != nil {
			t.Fatal(err)
		}
		utf.WriteString(s)
		entries = append(entries, entry{tag: 1, data: utf.Bytes()})

		ref := poolIndex
		poolIndex++
		var stringRef bytes.Buffer
		if err := binary.Write(&stringRef, binary.BigEndian, uint16(ref)); err != nil {
			t.Fatal(err)
		}
		entries = append(entries, entry{tag: 8, data: stringRef.Bytes()})
		poolIndex++
	}

	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.BigEndian, uint32(0xCAFEBABE)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(&buf, binary.BigEndian, uint16(0)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(&buf, binary.BigEndian, uint16(52)); err != nil {
		t.Fatal(err)
	}

	count := uint16(1)
	for _, entry := range entries {
		count++
		if entry.tag == 5 || entry.tag == 6 {
			count++
		}
	}
	if err := binary.Write(&buf, binary.BigEndian, count); err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if err := buf.WriteByte(entry.tag); err != nil {
			t.Fatal(err)
		}
		if _, err := buf.Write(entry.data); err != nil {
			t.Fatal(err)
		}
	}
	return buf.Bytes()
}

func testClassWithRawString(t *testing.T, raw []byte) []byte {
	t.Helper()

	var utf bytes.Buffer
	if err := binary.Write(&utf, binary.BigEndian, uint16(len(raw))); err != nil {
		t.Fatal(err)
	}
	if _, err := utf.Write(raw); err != nil {
		t.Fatal(err)
	}

	var ref bytes.Buffer
	if err := binary.Write(&ref, binary.BigEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.BigEndian, uint32(0xCAFEBABE)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(&buf, binary.BigEndian, uint16(0)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(&buf, binary.BigEndian, uint16(52)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(&buf, binary.BigEndian, uint16(3)); err != nil {
		t.Fatal(err)
	}
	if err := buf.WriteByte(1); err != nil {
		t.Fatal(err)
	}
	if _, err := buf.Write(utf.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := buf.WriteByte(8); err != nil {
		t.Fatal(err)
	}
	if _, err := buf.Write(ref.Bytes()); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
