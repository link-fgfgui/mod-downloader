package minecraft

import (
	"archive/zip"
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/link-fgfgui/mod-downloader-core/global"
	structs "github.com/link-fgfgui/mod-downloader-core/structs/minecraft"
)

func TestFileHashesUsesRawSHA1AndWhitespaceNormalizedFingerprint(t *testing.T) {
	raw := []byte("abc \t\ndef\r")
	path := filepath.Join(t.TempDir(), "mod.jar")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	gotSHA1, gotFingerprint := FileHashes(path)
	wantSHA1 := sha1.Sum(raw)
	if gotSHA1 != hex.EncodeToString(wantSHA1[:]) {
		t.Fatalf("SHA1 = %q, want %q", gotSHA1, hex.EncodeToString(wantSHA1[:]))
	}

	normalizedPath := filepath.Join(t.TempDir(), "normalized.jar")
	if err := os.WriteFile(normalizedPath, []byte("abcdef"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, normalizedFingerprint := FileHashes(normalizedPath)
	if gotFingerprint == 0 || gotFingerprint != normalizedFingerprint {
		t.Fatalf("fingerprints = %d/%d, want same non-zero value", gotFingerprint, normalizedFingerprint)
	}
}

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

	mods := ParseModZipReader(testZipReader(t, top), "top.jar", "fabric")
	got := modIDs(mods)
	want := []string{"top-fabric"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseModZipReader() ids = %#v, want %#v", got, want)
	}

	gotJij := jijModIDs(mods[0].JijMods)
	wantJij := []string{"child-fabric", "grand-child"}
	if !reflect.DeepEqual(gotJij, wantJij) {
		t.Fatalf("ParseModZipReader() JijMods ids = %#v, want %#v", gotJij, wantJij)
	}
}

func TestParseSkipsJijWhenDetectionDisabled(t *testing.T) {
	child := testJar(t, map[string][]byte{
		"fabric.mod.json": []byte(`{
			"schemaVersion": 1,
			"id": "nested-child",
			"version": "1.0.0"
		}`),
	})
	top := testJar(t, map[string][]byte{
		"fabric.mod.json": []byte(`{
			"schemaVersion": 1,
			"id": "top-fabric",
			"version": "1.0.0",
			"jars": [{"file": "META-INF/jars/child.jar"}]
		}`),
		"META-INF/jars/child.jar": child,
	})

	SetJijDetectionEnabled(false)
	t.Cleanup(func() { SetJijDetectionEnabled(true) })

	mods := ParseModZipReader(testZipReader(t, top), "top.jar", "fabric")
	if got := modIDs(mods); !reflect.DeepEqual(got, []string{"top-fabric"}) {
		t.Fatalf("ids = %#v, want [top-fabric]", got)
	}
	if len(mods[0].JijMods) != 0 {
		t.Fatalf("JijMods = %#v, want none when detection disabled", mods[0].JijMods)
	}
}

func TestParseJijCacheScopeFollowsToggle(t *testing.T) {
	child := testJar(t, map[string][]byte{
		"fabric.mod.json": []byte(`{
			"schemaVersion": 1,
			"id": "cache-child",
			"version": "1.0.0"
		}`),
	})
	top := testJar(t, map[string][]byte{
		"fabric.mod.json": []byte(`{
			"schemaVersion": 1,
			"id": "cache-top",
			"version": "1.0.0",
			"jars": [{"file": "META-INF/jars/child.jar"}]
		}`),
		"META-INF/jars/child.jar": child,
	})
	path := filepath.Join(t.TempDir(), "cache-top.jar")
	if err := os.WriteFile(path, top, 0o644); err != nil {
		t.Fatalf("write jar: %v", err)
	}

	SetJijDetectionEnabled(true)
	t.Cleanup(func() { SetJijDetectionEnabled(true) })

	on := ParseLocalModJarWithSHA1(path, "jij-cache-scope-toggle", "fabric")
	if len(on) != 1 || len(on[0].JijMods) == 0 {
		t.Fatalf("JijMods with detection enabled = %#v, want nested entries", on)
	}

	SetJijDetectionEnabled(false)
	off := ParseLocalModJarWithSHA1(path, "jij-cache-scope-toggle", "fabric")
	if len(off) != 1 {
		t.Fatalf("mods with detection disabled = %#v, want one mod", off)
	}
	if len(off[0].JijMods) != 0 {
		t.Fatalf("JijMods after toggle = %#v, want the cached JIJ result to be bypassed", off[0].JijMods)
	}
}

func TestParseFabricRequiredDependencies(t *testing.T) {
	jar := testJar(t, map[string][]byte{
		"fabric.mod.json": []byte(`{
			"schemaVersion": 1,
			"id": "top-fabric",
			"version": "1.0.0",
			"depends": {
				"fabricloader": ">=0.15.0",
				"minecraft": ">=1.21",
				"java": ">=21",
				"fabric-api": "*",
				"cloth-config": ["*"],
				"${placeholder}": "*"
			},
			"recommends": {
				"optional-helper": "*"
			}
		}`),
	})

	mods := ParseModZipReader(testZipReader(t, jar), "top.jar", "fabric")
	if len(mods) != 1 {
		t.Fatalf("ParseModZipReader() returned %d mods, want 1: %#v", len(mods), mods)
	}
	got := dependencyIDs(mods[0].Dependencies)
	want := []string{"fabric-api", "cloth-config"}
	sort.Strings(got)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("dependencies = %#v, want %#v", got, want)
	}
}

func TestParseFabricAcceptsRawControlCharsInStrings(t *testing.T) {
	// Fabric Loader accepts fabric.mod.json files whose strings contain literal
	// newlines/tabs (Gson does not validate control characters). ETF 7.2.6 ships
	// such a "description", so a strict encoding/json parse drops the whole mod.
	fabricModJSON := "{\n" +
		"  \"schemaVersion\": 1,\n" +
		"  \"id\": \"entity_texture_features\",\n" +
		"  \"version\": \"7.2.6\",\n" +
		"  \"name\": \"Entity Texture Features\",\n" +
		"  \"description\": \"Line one\nLine two\tindented \\\"quoted\\\" C:\\\\temp\",\n" +
		"  \"depends\": {\"fabricloader\": \"*\", \"minecraft\": \"*\", \"architectury\": \">=1.0\"}\n" +
		"}"
	jar := testJar(t, map[string][]byte{
		"fabric.mod.json": []byte(fabricModJSON),
	})

	mods := ParseModZipReader(testZipReader(t, jar), "raw-control.jar", "fabric")
	if len(mods) != 1 {
		t.Fatalf("ParseModZipReader() returned %d mods, want 1", len(mods))
	}
	mod := mods[0]
	if mod.ID != "entity_texture_features" || mod.Name != "Entity Texture Features" || mod.Version != "7.2.6" {
		t.Fatalf("mod = %#v, want parsed fabric metadata", mod)
	}
	wantDescription := "Line one\nLine two\tindented \"quoted\" C:\\temp"
	if mod.Description != wantDescription {
		t.Fatalf("description = %q, want %q", mod.Description, wantDescription)
	}
	if len(mod.Dependencies) != 1 || mod.Dependencies[0].ModID != "architectury" {
		t.Fatalf("dependencies = %#v, want architectury only", mod.Dependencies)
	}
}

func TestUnmarshalModMetadataJSONKeepsEscapedSequences(t *testing.T) {
	var meta struct {
		Description string `json:"description"`
		ID          string `json:"id"`
	}
	// Raw 0x01 must be escaped by the retry path, while the existing \n and \"
	// escapes must survive untouched.
	data := []byte("{\"id\":\"example\",\"description\":\"a\x01b\\n\\\"c\\\"\"}")
	if err := unmarshalModMetadataJSON(data, &meta); err != nil {
		t.Fatalf("unmarshalModMetadataJSON() error = %v, want nil", err)
	}
	if meta.ID != "example" || meta.Description != "a\x01b\n\"c\"" {
		t.Fatalf("meta = %#v, want original escapes preserved", meta)
	}

	if err := unmarshalModMetadataJSON([]byte(`{"id":}`), &meta); err == nil {
		t.Fatal("unmarshalModMetadataJSON() error = nil, want syntax error")
	}
}

func TestNormalizeModMetadataJSONPreservesStandardEscapes(t *testing.T) {
	data := []byte(`{"id":"x","d":"a\n\t\u0000\\\"\/b"}`)
	if got := normalizeModMetadataJSON(data); !bytes.Equal(got, data) {
		t.Fatalf("normalizeModMetadataJSON() = %q, want input unchanged", got)
	}
}

func TestUnmarshalModMetadataJSONGsonBoundaryParity(t *testing.T) {
	// Expected outcomes below were verified against the Gson JsonReader vendored
	// by fabric-loader 0.20.1 (non-lenient), google/gson 2.8.9, and 2.15.0.
	// Untagged fields match the JSON keys case-insensitively, like the parser.
	type meta struct {
		ID          string
		Description string
		Depends     map[string]json.RawMessage
	}
	bom := []byte{0xEF, 0xBB, 0xBF}
	withBOM := func(data []byte) []byte {
		out := make([]byte, 0, len(bom)+len(data))
		out = append(out, bom...)
		return append(out, data...)
	}
	rawControl := func(c byte) []byte {
		return []byte("{\"id\":\"x\",\"description\":\"a" + string(c) + "b\"}")
	}

	cases := []struct {
		name string
		data []byte
		ok   bool
		id   string
		desc string
	}{
		{name: "valid", data: []byte("{\"id\":\"x\"}"), ok: true, id: "x"},
		{name: "raw lf", data: rawControl('\n'), ok: true, id: "x", desc: "a\nb"},
		{name: "raw cr", data: rawControl('\r'), ok: true, id: "x", desc: "a\rb"},
		{name: "raw tab", data: rawControl('\t'), ok: true, id: "x", desc: "a\tb"},
		{name: "raw nul", data: rawControl(0x00), ok: true, id: "x", desc: "a\x00b"},
		{name: "raw 0x01", data: rawControl(0x01), ok: true, id: "x", desc: "a\x01b"},
		{name: "raw 0x1f", data: rawControl(0x1f), ok: true, id: "x", desc: "a\x1fb"},
		{name: "raw del", data: rawControl(0x7f), ok: true, id: "x", desc: "a\x7fb"},
		{name: "raw lf in key", data: []byte("{\"i\nd\":\"x\"}"), ok: true},
		{name: "invalid utf8", data: []byte("{\"id\":\"x\",\"description\":\"a\x80b\"}"), ok: true, id: "x", desc: "a\ufffdb"},
		{name: "leading bom", data: withBOM([]byte("{\"id\":\"x\"}")), ok: true, id: "x"},
		{name: "leading bom and raw lf", data: withBOM(rawControl('\n')), ok: true, id: "x", desc: "a\nb"},
		{name: "escaped single quote", data: []byte("{\"id\":\"x\",\"description\":\"a\\'b\"}"), ok: true, id: "x", desc: "a'b"},
		{name: "backslash lf", data: []byte("{\"id\":\"x\",\"description\":\"a\\\nb\"}"), ok: true, id: "x", desc: "a\nb"},
		{name: "backslash lf in key", data: []byte("{\"i\\\nd\":\"x\"}"), ok: true},
		{name: "escaped slash", data: []byte("{\"id\":\"x\",\"description\":\"a\\/b\"}"), ok: true, id: "x", desc: "a/b"},
		{name: "escaped unicode", data: []byte("{\"id\":\"x\",\"description\":\"a\\u0000b\"}"), ok: true, id: "x", desc: "a\x00b"},
		{name: "lone surrogate", data: []byte("{\"id\":\"x\",\"description\":\"a\\ud800b\"}"), ok: true, id: "x", desc: "a\ufffdb"},
		{name: "duplicate keys", data: []byte("{\"id\":\"x\",\"id\":\"y\"}"), ok: true, id: "y"},
		{name: "huge number in depends", data: []byte("{\"id\":\"x\",\"depends\":{\"a\":1e400}}"), ok: true, id: "x"},
		{name: "trailing document", data: []byte("{\"id\":\"x\"}{\"id\":\"y\"}"), ok: true, id: "x"},
		{name: "trailing garbage", data: []byte("{\"id\":\"x\"}x"), ok: true, id: "x"},
		{name: "trailing nul", data: append([]byte("{\"id\":\"x\"}"), 0x00), ok: true, id: "x"},

		{name: "trailing comma object", data: []byte("{\"id\":\"x\",}")},
		{name: "trailing comma array", data: []byte("{\"id\":\"x\",\"a\":[1,]}")},
		{name: "line comment", data: []byte("{\"id\":\"x\" // c\n}")},
		{name: "block comment", data: []byte("{\"id\":\"x\" /* c */}")},
		{name: "single quotes", data: []byte("{'id':'x'}")},
		{name: "unquoted key", data: []byte("{id:\"x\"}")},
		{name: "unquoted value", data: []byte("{\"id\":x}")},
		{name: "equals separator", data: []byte("{\"id\"=\"x\"}")},
		{name: "semicolon separator", data: []byte("{\"id\":\"x\";\"a\":1}")},
		{name: "nan", data: []byte("{\"id\":\"x\",\"a\":NaN}")},
		{name: "leading zero", data: []byte("{\"id\":\"x\",\"a\":01}")},
		{name: "plus number", data: []byte("{\"id\":\"x\",\"a\":+1}")},
		{name: "leading dot", data: []byte("{\"id\":\"x\",\"a\":.5}")},
		{name: "trailing dot", data: []byte("{\"id\":\"x\",\"a\":5.}")},
		{name: "bad escape", data: []byte("{\"id\":\"x\",\"d\":\"a\\qb\"}")},
		{name: "hex escape", data: []byte("{\"id\":\"x\",\"d\":\"a\\x41b\"}")},
		{name: "backslash cr", data: []byte("{\"id\":\"x\",\"d\":\"a\\\rb\"}")},
		{name: "vertical tab escape", data: []byte("{\"id\":\"x\",\"d\":\"a\\vb\"}")},
		{name: "zero escape", data: []byte("{\"id\":\"x\",\"d\":\"a\\0b\"}")},
		{name: "unterminated", data: []byte("{\"id\":\"x\",\"d\":\"abc")},
		{name: "short unicode", data: []byte("{\"id\":\"x\",\"d\":\"\\u12\"}")},
		{name: "bom after whitespace", data: append([]byte(" "), withBOM([]byte("{\"id\":\"x\"}"))...)},
		{name: "bad literal", data: []byte("{\"id\":\"x\",\"a\":tru}")},
		{name: "empty", data: []byte{}},
		{name: "top level array", data: []byte("[1]")},
		{name: "null root", data: []byte("null")},
		{name: "leading garbage", data: []byte("x{\"id\":\"x\"}")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var m meta
			err := unmarshalModMetadataJSON(tc.data, &m)
			if !tc.ok {
				if err == nil {
					t.Fatalf("unmarshalModMetadataJSON() accepted %q, want rejection like Gson", tc.data)
				}
				return
			}
			if err != nil {
				t.Fatalf("unmarshalModMetadataJSON() error = %v, want acceptance like Gson", err)
			}
			if m.ID != tc.id {
				t.Fatalf("id = %q, want %q", m.ID, tc.id)
			}
			if m.Description != tc.desc {
				t.Fatalf("description = %q, want %q", m.Description, tc.desc)
			}
		})
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

	mods := ParseModZipReader(testZipReader(t, top), "top.jar", "forge")
	got := modIDs(mods)
	want := []string{"topforge", "jei"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseModZipReader() ids = %#v, want %#v", got, want)
	}

	gotJij := jijModIDs(mods[0].JijMods)
	wantJij := []string{"childforge"}
	if !reflect.DeepEqual(gotJij, wantJij) {
		t.Fatalf("ParseModZipReader() JijMods ids = %#v, want %#v", gotJij, wantJij)
	}
}

func TestParseForgeAndNeoForgeRequiredDependencies(t *testing.T) {
	for _, tc := range []struct {
		name       string
		path       string
		loader     string
		tomlSource string
		want       []string
	}{
		{
			name:   "forge mandatory",
			path:   "META-INF/mods.toml",
			loader: "forge",
			tomlSource: `
license="MIT"

[[mods]]
modId="topforge"

[[dependencies.topforge]]
modId="minecraft"
mandatory=true

[[dependencies.topforge]]
modId="forge"
mandatory=true

[[dependencies.topforge]]
modId="architectury"
mandatory=true

[[dependencies.topforge]]
modId="optional-helper"
mandatory=false
`,
			want: []string{"architectury"},
		},
		{
			name:   "neoforge typed",
			path:   "META-INF/neoforge.mods.toml",
			loader: "neoforge",
			tomlSource: `
license="MIT"

[[mods]]
modId="topneo"

[[dependencies.topneo]]
modId="neoforge"
type="required"

[[dependencies.topneo]]
modId="kotlinforforge"
type="required"

[[dependencies.topneo]]
modId="optional-helper"
type="optional"
`,
			want: []string{"kotlinforforge"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			jar := testJar(t, map[string][]byte{
				tc.path: []byte(tc.tomlSource),
			})

			mods := ParseModZipReader(testZipReader(t, jar), "top.jar", tc.loader)
			if len(mods) != 1 {
				t.Fatalf("ParseModZipReader() returned %d mods, want 1: %#v", len(mods), mods)
			}
			got := dependencyIDs(mods[0].Dependencies)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("dependencies = %#v, want %#v", got, tc.want)
			}
		})
	}
}

// TestForgeJijModsAreAttachedToTopLevelMods verifies that top-level [[mods]]
// entries stay in the returned ModInfo list while nested jar / JIJ entries are
// attached to each top-level entry as JijMods.
func TestForgeJijModsAreAttachedToTopLevelMods(t *testing.T) {
	// child JAR: declare two mods, only reachable as a JIJ from the host.
	child := testJar(t, map[string][]byte{
		"META-INF/mods.toml": []byte(`
license="MIT"

[[mods]]
modId="childmod"
displayName="Child Mod"

[[mods]]
modId="childmod-extra"
displayName="Child Extra"
`),
	})

	// top JAR: declares two mods directly AND embeds the child as a JIJ.
	// The scenario mirrors a mod like tmrv that declares both "tmrv" and "jei"
	// in its own mods.toml, while also bundling a nested jar.
	top := testJar(t, map[string][]byte{
		"META-INF/mods.toml": []byte(`
license="MIT"

[[mods]]
modId="topmod"
displayName="Top Mod"

[[mods]]
modId="extramod"
displayName="Extra Mod declared in same mods.toml"
`),
		"META-INF/jarjar/metadata.json": []byte(`{
			"jars": [{"path": "META-INF/jarjar/child.jar"}]
		}`),
		"META-INF/jarjar/child.jar": child,
	})

	mods := ParseModZipReader(testZipReader(t, top), "top.jar", "forge")

	wantStrong := []string{"topmod", "extramod"}
	wantJij := []string{"childmod", "childmod-extra"}

	strongIDs := modIDs(mods)
	if !reflect.DeepEqual(strongIDs, wantStrong) {
		t.Errorf("strong modIDs = %#v, want %#v", strongIDs, wantStrong)
	}
	for _, mod := range mods {
		jijIDs := jijModIDs(mod.JijMods)
		if !reflect.DeepEqual(jijIDs, wantJij) {
			t.Errorf("%s JijMods = %#v, want %#v", mod.ID, jijIDs, wantJij)
		}
	}

	// PrimaryModIDs must return only the strong set.
	primary := PrimaryModIDs(mods)
	if !reflect.DeepEqual(primary, wantStrong) {
		t.Errorf("PrimaryModIDs() = %#v, want %#v", primary, wantStrong)
	}

	// PrimaryModIDs must NOT contain any JIJ mod ID.
	primarySet := make(map[string]bool, len(primary))
	for _, id := range primary {
		primarySet[id] = true
	}
	for _, id := range wantJij {
		if primarySet[id] {
			t.Errorf("PrimaryModIDs() unexpectedly contains JIJ modID %q", id)
		}
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

func TestParseLocalModZipReaderMergesLoadersForSameModID(t *testing.T) {
	jar := testJar(t, map[string][]byte{
		"fabric.mod.json": []byte(`{
			"schemaVersion": 1,
			"id": "example",
			"name": "Fabric Name",
			"version": "fabric-version"
		}`),
		"META-INF/neoforge.mods.toml": []byte(`
license="MIT"

[[mods]]
modId="example"
displayName="NeoForge Name"
version="neoforge-version"
`),
	})

	got := ParseLocalModZipReader(testZipReader(t, jar), "multi-loader.jar", "neoforge")
	if len(got) != 1 {
		t.Fatalf("ParseLocalModZipReader() returned %d mods, want 1: %#v", len(got), got)
	}
	if got[0].Name != "NeoForge Name" || got[0].Version != "neoforge-version" {
		t.Fatalf("preferred metadata = %#v, want NeoForge values", got[0])
	}
	if want := []string{"neoforge", "fabric"}; !reflect.DeepEqual(got[0].Loaders, want) {
		t.Fatalf("loaders = %#v, want %#v", got[0].Loaders, want)
	}
}

func TestJarMetadataCacheSeparatesRequestedAndLocalParserScopes(t *testing.T) {
	global.ClearJarMetadata()
	t.Cleanup(global.ClearJarMetadata)

	jar := testJar(t, map[string][]byte{
		"fabric.mod.json":    []byte(`{"schemaVersion":1,"id":"fabricmod","version":"1.0.0"}`),
		"META-INF/mods.toml": []byte("license=\"MIT\"\n[[mods]]\nmodId=\"forgemod\"\n"),
	})
	path := filepath.Join(t.TempDir(), "scoped.jar")
	if err := os.WriteFile(path, jar, 0o644); err != nil {
		t.Fatal(err)
	}
	sha := FileSHA1(path)

	if got := modIDs(ParseModJarWithSHA1(path, sha, "forge")); !reflect.DeepEqual(got, []string{"forgemod"}) {
		t.Fatalf("forge parse ids = %#v, want forge only", got)
	}
	local := ParseLocalModJarWithSHA1(path, sha, "forge")
	got := modIDs(local)
	want := []string{"forgemod", "fabricmod"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("local parse ids = %#v, want %#v", got, want)
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

func jijModIDs(mods []structs.JijModInfo) []string {
	ids := make([]string, 0, len(mods))
	for _, mod := range mods {
		ids = append(ids, mod.ID)
	}
	return ids
}

func dependencyIDs(deps []structs.LocalModDependency) []string {
	ids := make([]string, 0, len(deps))
	for _, dep := range deps {
		ids = append(ids, dep.ModID)
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
