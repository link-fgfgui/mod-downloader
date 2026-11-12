// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

package httpserver

import (
	"bytes"
	"encoding/json"
	"go/parser"
	"go/token"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	appstructs "github.com/link-fgfgui/mod-downloader/core/structs"
)

func TestHealthEndpoint(t *testing.T) {
	s := New("")
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	s.handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("health status = %d, want %d", rec.Code, http.StatusOK)
	}
	if body := rec.Body.String(); !strings.Contains(body, `"ok"`) || !strings.Contains(body, `"apiVersion":1`) {
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
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestRootRejectsMismatchedResultCount(t *testing.T) {
	s := New("", Options{Accept: func(_ []appstructs.ExtensionProjectPayload) []appstructs.ModDownloadResult {
		return nil
	}})
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`[{"p":"mr","slug":"sodium","file":""}]`))
	req.Header.Set("Content-Type", "application/json")
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

// TestStartUsesProvidedListener proves the single-instance guard socket is
// reused instead of binding Addr a second time.
func TestStartUsesProvidedListener(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("bind listener: %v", err)
	}
	// Addr is deliberately invalid: Start must rely solely on the listener.
	s := New("127.0.0.1:1", Options{Listener: ln})
	if err := s.Start(); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	if got, want := s.Addr(), ln.Addr().String(); got != want {
		t.Fatalf("Addr() = %q, want listener address %q", got, want)
	}

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://" + s.Addr() + "/health")
	if err != nil {
		t.Fatalf("health request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("health status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	s.Stop()
}

func TestRootAcceptsEmptyArray(t *testing.T) {
	s := New("", Options{Accept: func(_ []appstructs.ExtensionProjectPayload) []appstructs.ModDownloadResult {
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

func TestProjectsEndpointAcceptsCanonicalPathAndSetsVersion(t *testing.T) {
	var called bool
	s := New("", Options{Accept: func(payloads []appstructs.ExtensionProjectPayload) []appstructs.ModDownloadResult {
		called = true
		return make([]appstructs.ModDownloadResult, len(payloads))
	}})
	req := httptest.NewRequest(http.MethodPost, ProjectsPath, strings.NewReader(`[{"p":"mr","slug":"sodium"}]`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "chrome-extension://abcdefghijklmnop")
	rec := httptest.NewRecorder()
	s.handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if !called {
		t.Fatal("accept handler was not called")
	}
	if got := rec.Header().Get("X-Mod-Downloader-API-Version"); got != "1" {
		t.Fatalf("api version header = %q, want 1", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "chrome-extension://abcdefghijklmnop" {
		t.Fatalf("allow-origin header = %q", got)
	}
}

func TestProjectsEndpointRejectsForeignOrigin(t *testing.T) {
	s := New("", Options{Accept: func(payloads []appstructs.ExtensionProjectPayload) []appstructs.ModDownloadResult {
		return make([]appstructs.ModDownloadResult, len(payloads))
	}})
	req := httptest.NewRequest(http.MethodPost, ProjectsPath, strings.NewReader("[]"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()
	s.handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestProjectsEndpointAllowsExtensionPreflight(t *testing.T) {
	s := New("")
	req := httptest.NewRequest(http.MethodOptions, ProjectsPath, nil)
	req.Header.Set("Origin", "moz-extension://abcdefghijklmnop")
	rec := httptest.NewRecorder()
	s.handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(got, "POST") {
		t.Fatalf("allow-methods header = %q", got)
	}
}

func TestProjectsEndpointRequiresJSONContentType(t *testing.T) {
	s := New("", Options{Accept: func(payloads []appstructs.ExtensionProjectPayload) []appstructs.ModDownloadResult {
		return make([]appstructs.ModDownloadResult, len(payloads))
	}})
	req := httptest.NewRequest(http.MethodPost, ProjectsPath, strings.NewReader("[]"))
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()
	s.handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnsupportedMediaType)
	}
}

func TestProjectsEndpointRejectsTooManyPayloads(t *testing.T) {
	called := false
	s := New("", Options{Accept: func(payloads []appstructs.ExtensionProjectPayload) []appstructs.ModDownloadResult {
		called = true
		return make([]appstructs.ModDownloadResult, len(payloads))
	}})
	payloads := make([]appstructs.ExtensionProjectPayload, maxExtensionPayloads+1)
	for i := range payloads {
		payloads[i] = appstructs.ExtensionProjectPayload{PlatformCode: "mr", Slug: "sodium"}
	}
	body, err := json.Marshal(payloads)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, ProjectsPath, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}
	if called {
		t.Fatal("accept handler called for oversized payload list")
	}
}

func TestProjectsEndpointRejectsOversizedBody(t *testing.T) {
	s := New("", Options{Accept: func(payloads []appstructs.ExtensionProjectPayload) []appstructs.ModDownloadResult {
		return make([]appstructs.ModDownloadResult, len(payloads))
	}})
	req := httptest.NewRequest(http.MethodPost, ProjectsPath, bytes.NewReader(make([]byte, maxBodyBytes+1)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestUnknownPathReturnsNotFound(t *testing.T) {
	s := New("")
	req := httptest.NewRequest(http.MethodPost, "/v2/extension/projects", strings.NewReader("[]"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

// TestPackageStaysTransportOnly guards the architecture boundary: the HTTP
// adapter must not reach into global, provider or storage state.
func TestPackageStaysTransportOnly(t *testing.T) {
	forbidden := map[string]bool{
		"github.com/link-fgfgui/mod-downloader/core/global":    true,
		"github.com/link-fgfgui/mod-downloader/core/providers": true,
		"github.com/link-fgfgui/mod-downloader/core/storage":   true,
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
