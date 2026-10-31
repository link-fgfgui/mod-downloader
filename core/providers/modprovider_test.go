package providers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	modrinth "codeberg.org/jmansfield/go-modrinth/modrinth"
	"github.com/link-fgfgui/mod-downloader-core/global"
	"github.com/link-fgfgui/mod-downloader-core/models"
	"github.com/link-fgfgui/mod-downloader-core/storage"
	curseforge "github.com/sjet47/go-curseforge"
	cfAPI "github.com/sjet47/go-curseforge/api"
	cfSchema "github.com/sjet47/go-curseforge/schema"
	cfEnum "github.com/sjet47/go-curseforge/schema/enum"
)

type providerRoundTripFunc func(*http.Request) (*http.Response, error)

func (fn providerRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestCurseForgeDownloadURL(t *testing.T) {
	tests := []struct {
		name string
		id   string
		file string
		want string
	}{
		{name: "standard", id: "12345678", file: "example.jar", want: "https://edge.forgecdn.net/files/1234/5678/example.jar"},
		{name: "escaped filename", id: "12345678", file: "example mod.jar", want: "https://edge.forgecdn.net/files/1234/5678/example%20mod.jar"},
		{name: "invalid id", id: "12ab5678", file: "example.jar"},
		{name: "short id", id: "1234", file: "example.jar"},
		{name: "missing filename", id: "12345678"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CurseForgeDownloadURL(tt.id, tt.file); got != tt.want {
				t.Fatalf("CurseForgeDownloadURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEffectiveDownloadURLPreservesProviderURL(t *testing.T) {
	version := models.ModVersion{Platform: "CurseForge", DownloadURL: " https://cdn.example.test/mod.jar "}
	if got := EffectiveDownloadURL(version); got != "https://cdn.example.test/mod.jar" {
		t.Fatalf("EffectiveDownloadURL() = %q, want trimmed provider URL", got)
	}
	if got := EffectiveDownloadURL(models.ModVersion{Platform: "Modrinth", ID: "12345678", FileName: "mod.jar"}); got != "" {
		t.Fatalf("EffectiveDownloadURL() for Modrinth = %q, want empty", got)
	}
}

func TestResolveMetadataByLocalFilesFallsBackToCurseForgeFingerprint(t *testing.T) {
	if err := storage.OpenAt(filepath.Join(t.TempDir(), storage.CacheFileName)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(storage.Close)
	const fingerprint = uint32(3456789012)
	transport := providerRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var body string
		switch req.URL.Path {
		case "/v1/fingerprints/432":
			var input struct {
				Fingerprints []uint32 `json:"fingerprints"`
			}
			if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
				t.Fatal(err)
			}
			if len(input.Fingerprints) != 1 || input.Fingerprints[0] != fingerprint {
				t.Fatalf("fingerprints request = %#v", input.Fingerprints)
			}
			body = `{"data":{"isCacheBuilt":true,"exactMatches":[{"id":123,"file":{"id":456,"gameId":432,"modId":123,"displayName":"CF Version","fileName":"cf-mod.jar","downloadUrl":"https://edge.forgecdn.net/files/cf-mod.jar","gameVersions":["1.21.1","Fabric"],"fileFingerprint":3456789012}}]}}`
		case "/v1/mods":
			body = `{"data":[{"id":123,"gameId":432,"name":"CF Mod","slug":"cf-mod","summary":"From CurseForge","classId":6,"logo":{"thumbnailUrl":"https://example.com/cf.png"},"categories":[{"name":"Technology","slug":"technology"}]}]}`
		default:
			t.Fatalf("unexpected CurseForge request path %q", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil
	})
	previous := global.GetCurseForgeClient()
	previousModrinth := global.GetModrinthClient()
	global.SetCurseForgeClient(&curseforge.Client{API: cfAPI.New(transport)})
	global.SetModrinthClient(nil)
	t.Cleanup(func() {
		global.SetCurseForgeClient(previous)
		global.SetModrinthClient(previousModrinth)
	})

	projects, versions := ResolveMetadataByLocalFiles(map[string]uint32{"abc123": fingerprint})
	project, ok := projects["abc123"]
	if !ok || project.Platform != "CurseForge" || project.ProjectID != "123" || project.Title != "CF Mod" {
		t.Fatalf("resolved project = %#v, %v", project, ok)
	}
	version, ok := versions["abc123"]
	if !ok || version.ID != "456" || version.SHA1 != "abc123" || version.FileName != "cf-mod.jar" {
		t.Fatalf("resolved version = %#v, %v", version, ok)
	}
	if cached, ok := storage.GetVersionBySHA1("curseforge", "abc123"); !ok || cached.ID != "456" {
		t.Fatalf("cached version = %#v, %v", cached, ok)
	}
}

// resetProviderBackgroundForTest isolates background-task reference counts and
// acceptance between tests, including repeated -count runs.
func resetProviderBackgroundForTest(t *testing.T) {
	t.Helper()
	reset := func() {
		providerBackground.lifecycle.Lock()
		providerBackground.mu.Lock()
		providerBackground.starts = 0
		providerBackground.accepting = false
		providerBackground.mu.Unlock()
		providerBackground.wg.Wait()
		providerBackground.mu.Lock()
		providerBackground.accepting = true
		providerBackground.mu.Unlock()
		providerBackground.lifecycle.Unlock()
	}
	reset()
	t.Cleanup(reset)
}

func TestStopBackgroundTasksWaitsAndRejectsNewWork(t *testing.T) {
	resetProviderBackgroundForTest(t)
	StartBackgroundTasks()
	started := make(chan struct{})
	release := make(chan struct{})
	finished := make(chan struct{})
	startBackgroundTask(func() {
		close(started)
		<-release
	})
	<-started
	go func() {
		StopBackgroundTasks()
		close(finished)
	}()
	select {
	case <-finished:
		t.Fatal("background wait returned before task completion")
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("background wait did not return")
	}
	rejected := make(chan struct{}, 1)
	startBackgroundTask(func() { rejected <- struct{}{} })
	select {
	case <-rejected:
		t.Fatal("background work accepted after stop")
	case <-time.After(20 * time.Millisecond):
	}
}

func TestDrainBackgroundTasksWaitsAndKeepsAccepting(t *testing.T) {
	resetProviderBackgroundForTest(t)
	StartBackgroundTasks()
	started := make(chan struct{})
	release := make(chan struct{})
	finished := make(chan struct{})
	startBackgroundTask(func() {
		close(started)
		<-release
	})
	<-started
	go func() {
		DrainBackgroundTasks()
		close(finished)
	}()
	select {
	case <-finished:
		t.Fatal("drain returned before task completion")
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("drain did not return")
	}
	ran := make(chan struct{}, 1)
	startBackgroundTask(func() { ran <- struct{}{} })
	select {
	case <-ran:
	case <-time.After(time.Second):
		t.Fatal("background work rejected after drain")
	}
}

func TestResolveMetadataByLocalFilesCachesProviderMiss(t *testing.T) {
	if err := storage.OpenAt(filepath.Join(t.TempDir(), storage.CacheFileName)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(storage.Close)
	const fingerprint = uint32(123456789)
	requests := 0
	transport := providerRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"data":{"isCacheBuilt":true,"exactMatches":[]}}`)),
			Request:    req,
		}, nil
	})
	previousCurseForge := global.GetCurseForgeClient()
	previousModrinth := global.GetModrinthClient()
	global.SetCurseForgeClient(&curseforge.Client{API: cfAPI.New(transport)})
	global.SetModrinthClient(nil)
	t.Cleanup(func() {
		global.SetCurseForgeClient(previousCurseForge)
		global.SetModrinthClient(previousModrinth)
	})

	files := map[string]uint32{"abc123": fingerprint}
	ResolveMetadataByLocalFiles(files)
	ResolveMetadataByLocalFiles(files)
	if requests != 1 {
		t.Fatalf("fingerprint requests = %d, want 1", requests)
	}
	if _, ok := storage.GetLocalMetadataMissCheckedAt("curseforge", "123456789"); !ok {
		t.Fatal("CurseForge miss was not cached")
	}
	if err := storage.SetLocalMetadataMisses("curseforge", []string{"123456789"}, time.Now().Add(-localMetadataMissTTL-time.Minute).Unix()); err != nil {
		t.Fatal(err)
	}
	ResolveMetadataByLocalFiles(files)
	if requests != 2 {
		t.Fatalf("fingerprint requests after TTL = %d, want 2", requests)
	}
}

func TestResolveMetadataByLocalFilesDoesNotCacheProviderError(t *testing.T) {
	if err := storage.OpenAt(filepath.Join(t.TempDir(), storage.CacheFileName)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(storage.Close)
	const fingerprint = uint32(987654321)
	requests := 0
	transport := providerRoundTripFunc(func(*http.Request) (*http.Response, error) {
		requests++
		return nil, errors.New("temporary provider failure")
	})
	previousCurseForge := global.GetCurseForgeClient()
	previousModrinth := global.GetModrinthClient()
	global.SetCurseForgeClient(&curseforge.Client{API: cfAPI.New(transport)})
	global.SetModrinthClient(nil)
	t.Cleanup(func() {
		global.SetCurseForgeClient(previousCurseForge)
		global.SetModrinthClient(previousModrinth)
	})

	files := map[string]uint32{"abc123": fingerprint}
	ResolveMetadataByLocalFiles(files)
	ResolveMetadataByLocalFiles(files)
	if requests != 2 {
		t.Fatalf("fingerprint requests = %d, want retry after error", requests)
	}
	if _, ok := storage.GetLocalMetadataMissCheckedAt("curseforge", "987654321"); ok {
		t.Fatal("provider error was cached as a miss")
	}
}

func TestCurseForgeLogoURL(t *testing.T) {
	tests := []struct {
		name string
		logo cfSchema.ModAsset
		want string
	}{
		{
			name: "uses thumbnail url first",
			logo: cfSchema.ModAsset{
				ThumbnailUrl: " https://example.com/thumb.png ",
				URL:          "https://example.com/original.webp",
			},
			want: "https://example.com/thumb.png",
		},
		{
			name: "falls back to url",
			logo: cfSchema.ModAsset{
				URL: " https://media.forgecdn.net/avatars/1025/127/638548475358792693.webp ",
			},
			want: "https://media.forgecdn.net/avatars/1025/127/638548475358792693.webp",
		},
		{
			name: "returns empty when both are empty",
			logo: cfSchema.ModAsset{},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := curseForgeLogoURL(tt.logo)
			if got != tt.want {
				t.Fatalf("curseForgeLogoURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCurseForgeModToModProjectCategories(t *testing.T) {
	project := curseForgeProvider{}.modToModProject(cfSchema.Mod{
		Name: "Example",
		Categories: []cfSchema.Category{
			{Slug: "technology", Name: "Technology"},
			{Name: "Magic"},
			{Slug: "technology"},
		},
	})
	want := []string{"technology", "magic"}
	assertStrings(t, project.Categories, want)
}

func TestModrinthSearchHitToModProjectCategories(t *testing.T) {
	title := "Example"
	projectID := "abc123"
	hit := &modrinth.SearchResult{
		Title:      &title,
		ProjectID:  &projectID,
		Categories: []string{"library", "Magic", "library"},
	}
	project := modrinthProvider{}.searchHitToModProject(hit)
	assertStrings(t, project.Categories, []string{"library", "magic"})
}

func TestModrinthProjectToModProjectCategories(t *testing.T) {
	title := "Example"
	projectID := "abc123"
	project := modrinthProvider{}.projectToModProject(&modrinth.Project{
		Title:                &title,
		ID:                   &projectID,
		Categories:           []string{"library", "technology"},
		AdditionalCategories: []string{"Magic", "library"},
	})
	assertStrings(t, project.Categories, []string{"library", "technology", "magic"})
}

func TestProviderVersionConvertersPreserveFileSize(t *testing.T) {
	curseForgeVersion := curseForgeProvider{}.fileToModVersion(cfSchema.File{
		FileName:   "curseforge.jar",
		FileLength: 4096,
	})
	if curseForgeVersion.FileSize != 4096 {
		t.Fatalf("CurseForge file size = %d, want 4096", curseForgeVersion.FileSize)
	}

	fileName := "modrinth.jar"
	fileSize := uint32(8192)
	modrinthVersion := models.ModVersion{}
	modrinthProvider{}.setModVersionFileFields(&modrinthVersion, &modrinth.File{
		Filename: &fileName,
		Size:     &fileSize,
	})
	if modrinthVersion.FileSize != 8192 {
		t.Fatalf("Modrinth file size = %d, want 8192", modrinthVersion.FileSize)
	}

	modrinthProvider{}.setModVersionFileFields(&modrinthVersion, &modrinth.File{})
	if modrinthVersion.FileSize != 8192 {
		t.Fatalf("missing Modrinth size replaced known value: %d", modrinthVersion.FileSize)
	}
}

func TestProviderVersionConvertersPreserveReleaseType(t *testing.T) {
	curseForgeVersion := curseForgeProvider{}.fileToModVersion(cfSchema.File{
		ReleaseType: cfEnum.FileReleaseTypeBeta,
	})
	if curseForgeVersion.ReleaseType != "beta" {
		t.Fatalf("CurseForge release type = %q, want beta", curseForgeVersion.ReleaseType)
	}

	versionType := "alpha"
	modrinthVersion := modrinthProvider{}.versionToModVersion(&modrinth.Version{
		VersionType: &versionType,
	})
	if modrinthVersion.ReleaseType != "alpha" {
		t.Fatalf("Modrinth release type = %q, want alpha", modrinthVersion.ReleaseType)
	}
}

func assertStrings(t *testing.T, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("strings = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("strings[%d] = %q, want %q (all: %#v)", i, got[i], want[i], got)
		}
	}
}
