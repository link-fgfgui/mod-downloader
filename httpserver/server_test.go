package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestPlatformFromCode(t *testing.T) {
	cases := []struct {
		code string
		plat string
		ok   bool
	}{
		{"mr", "modrinth", true},
		{"MR", "modrinth", true},
		{" cf ", "curseforge", true},
		{"CF", "curseforge", true},
		{"xx", "", false},
		{"", "", false},
	}
	for _, c := range cases {
		plat, ok := platformFromCode(c.code)
		if plat != c.plat || ok != c.ok {
			t.Errorf("platformFromCode(%q) = (%q,%v), want (%q,%v)", c.code, plat, ok, c.plat, c.ok)
		}
	}
}

func TestHealthEndpoint(t *testing.T) {
	s := New(context.Background(), "")
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
	s := New(context.Background(), "")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	s.handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET / status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestRootRejectsInvalidJSON(t *testing.T) {
	s := New(context.Background(), "")
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{not json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid json status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestRootQueuesValidPayloadsAndSkipsInvalid(t *testing.T) {
	// No selected Minecraft instance is configured, so valid payloads are
	// skipped with "no selected version". This still exercises the full
	// parse -> request construction -> downloader path.
	s := New(context.Background(), "")

	// 0: mr with explicit project id (preferred over slug)
	// 1: cf with null id -> falls back to slug
	// 2: invalid platform code
	// 3: mr with null id and empty slug
	// 4: mr with empty-string id -> falls back to slug
	body := `[{"p":"mr","id":"Pvdbn7mC","slug":"ignored-slug","file":"ver-123"},` +
		`{"p":"cf","id":null,"slug":"jei","file":"ver-456"},` +
		`{"p":"xx","id":null,"slug":"bad","file":"ver-789"},` +
		`{"p":"mr","id":null,"slug":"","file":"ver-000"},` +
		`{"p":"mr","id":"","slug":"sodium","file":"ver-abc"}]`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var results []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &results); err != nil {
		t.Fatalf("decode results failed: %v; body=%s", err, rec.Body.String())
	}
	if len(results) != 5 {
		t.Fatalf("results count = %d, want 5", len(results))
	}

	// 0 and 1: valid, skipped because no instance selected.
	for i := 0; i < 2; i++ {
		skipped, _ := results[i]["skipped"].(bool)
		if !skipped {
			t.Errorf("result[%d] skipped = false, want true (no instance)", i)
		}
	}
	// 2: invalid platform.
	if skipped, _ := results[2]["skipped"].(bool); !skipped {
		t.Errorf("result[2] skipped = false, want true (invalid platform)")
	}
	if reason, _ := results[2]["reason"].(string); !strings.Contains(reason, "invalid platform") {
		t.Errorf("result[2] reason = %q", reason)
	}
	// 3: null id + empty slug.
	if skipped, _ := results[3]["skipped"].(bool); !skipped {
		t.Errorf("result[3] skipped = false, want true (empty id and slug)")
	}
	if reason, _ := results[3]["reason"].(string); !strings.Contains(reason, "empty project id and slug") {
		t.Errorf("result[3] reason = %q", reason)
	}
	// 4: empty-string id falls back to slug, valid -> skipped (no instance).
	if skipped, _ := results[4]["skipped"].(bool); !skipped {
		t.Errorf("result[4] skipped = false, want true (no instance)")
	}
}

func TestStartStopLifecycle(t *testing.T) {
	s := New(context.Background(), "127.0.0.1:0")
	if err := s.Start(); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	addr := s.Addr()
	if addr == "" {
		t.Fatal("Addr() empty after Start")
	}

	// Verify the server is actually serving by hitting /health.
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://" + addr + "/health")
	if err != nil {
		t.Fatalf("health request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("health status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Double Start is a no-op.
	if err := s.Start(); err != nil {
		t.Fatalf("second start failed: %v", err)
	}

	s.Stop()
	// Stop is idempotent.
	s.Stop()
}

// TestRootAcceptsEmptyArray ensures an empty scan result is handled gracefully.
func TestRootAcceptsEmptyArray(t *testing.T) {
	s := New(context.Background(), "")
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
