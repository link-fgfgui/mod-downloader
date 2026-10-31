package httpserver

import (
	"bytes"
	"encoding/json"
	"go/parser"
	"go/token"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	appstructs "github.com/link-fgfgui/mod-downloader-core/structs"
)

func TestHealthEndpoint(t *testing.T) {
	s := New("")
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	s.handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("health status = %d, want %d", rec.Code, http.StatusOK)
	}
	if body := rec.Body.String(); !strings.Contains(body, `"ok"`) {
		t.Fatalf("health body = %q", body)
	}
}

func TestRootRejectsGet(t *testing.T) {
	s := New("")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	s.handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET / status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestRootRejectsInvalidJSON(t *testing.T) {
	s := New("")
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{not json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid json status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestRootForwardsDecodedPayloads(t *testing.T) {
	var received []appstructs.ExtensionProjectPayload
	s := New("", Options{Accept: func(payloads []appstructs.ExtensionProjectPayload) []appstructs.ModDownloadResult {
		received = payloads
		results := make([]appstructs.ModDownloadResult, len(payloads))
		for i := range results {
			results[i] = appstructs.ModDownloadResult{Queued: true}
		}
		return results
	}})

	explicitID := "Pvdbn7mC"
	body := `[{"p":"mr","id":"Pvdbn7mC","slug":"ignored-slug","file":"ver-123"},` +
		`{"p":"cf","id":null,"slug":"jei","file":"ver-456"}]`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if len(received) != 2 {
		t.Fatalf("received %d payloads, want 2", len(received))
	}
	if received[0].PlatformCode != "mr" || received[0].ID == nil || *received[0].ID != explicitID || received[0].VersionID != "ver-123" {
		t.Fatalf("first payload = %#v", received[0])
	}
	if received[1].PlatformCode != "cf" || received[1].ID != nil || received[1].Slug != "jei" || received[1].VersionID != "ver-456" {
		t.Fatalf("second payload = %#v", received[1])
	}

	var results []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &results); err != nil {
		t.Fatalf("decode results failed: %v", err)
	}
	if len(results) != 2 || results[0]["queued"] != true || results[1]["queued"] != true {
		t.Fatalf("results = %#v", results)
	}
}

func TestRootRejectsWhenReceiverUnavailable(t *testing.T) {
	s := New("")
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("[]"))
	rec := httptest.NewRecorder()
	s.handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestRootRejectsMismatchedResultCount(t *testing.T) {
	s := New("", Options{Accept: func(payloads []appstructs.ExtensionProjectPayload) []appstructs.ModDownloadResult {
		return nil
	}})
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`[{"p":"mr","slug":"sodium","file":""}]`))
	rec := httptest.NewRecorder()
	s.handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestStartStopLifecycle(t *testing.T) {
	s := New("127.0.0.1:0")
	if err := s.Start(); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	addr := s.Addr()
	if addr == "" {
		t.Fatal("Addr() empty after Start")
	}

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://" + addr + "/health")
	if err != nil {
		t.Fatalf("health request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("health status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	if err := s.Start(); err != nil {
		t.Fatalf("second start failed: %v", err)
	}

	s.Stop()
	s.Stop()
}

func TestRootAcceptsEmptyArray(t *testing.T) {
	s := New("", Options{Accept: func(payloads []appstructs.ExtensionProjectPayload) []appstructs.ModDownloadResult {
		return []appstructs.ModDownloadResult{}
	}})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte("[]")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body, _ := io.ReadAll(rec.Body)
	if strings.TrimSpace(string(body)) != "[]" {
		t.Fatalf("empty array body = %q", string(body))
	}
}

// TestPackageStaysTransportOnly guards the architecture boundary: the HTTP
// adapter must not reach into global, provider or storage state.
func TestPackageStaysTransportOnly(t *testing.T) {
	forbidden := map[string]bool{
		"github.com/link-fgfgui/mod-downloader-core/global":    true,
		"github.com/link-fgfgui/mod-downloader-core/providers": true,
		"github.com/link-fgfgui/mod-downloader-core/storage":   true,
	}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	checked := 0
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(fset, file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		checked++
		for _, imp := range parsed.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			if forbidden[path] {
				t.Errorf("%s imports forbidden package %s", file, path)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no non-test source files were checked")
	}
}
