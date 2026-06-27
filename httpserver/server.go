// Package httpserver exposes an HTTP endpoint that receives mod scan results
// from the mod-downloader browser extension and queues them for download.
//
// The extension (mod-downloader-chrome-plugin) POSTs a JSON array of payloads
// shaped as {"p":"mr"|"cf","id":null,"slug":"...","file":"..."} to
// http://127.0.0.1:18801. Each payload is converted into a ModDownloadRequest
// and handed off to the downloader package.
package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"mod-downloader/downloader"
	"mod-downloader/logging"
	"mod-downloader/models"
	appstructs "mod-downloader/structs"
)

// DefaultAddr is the address the browser extension expects (see its manifest
// host_permissions and REMOTE_URL constant).
const DefaultAddr = "127.0.0.1:18801"

// maxBodyBytes limits the request body size to guard against abusive payloads.
const maxBodyBytes = 4 << 20 // 4 MiB

// remotePayload mirrors the Chrome extension's RemotePayload struct.
type remotePayload struct {
	P    string  `json:"p"`    // platform code: "mr" (Modrinth) | "cf" (CurseForge)
	ID   *string `json:"id"`   // project ID; nil falls back to slug
	Slug string  `json:"slug"` // project slug (fallback when id is null)
	File string  `json:"file"` // platform version ID (ModVersion.ID)
}

// Server is the HTTP server accepting extension scan results.
type Server struct {
	addr string
	ctx  context.Context

	mu         sync.Mutex
	server     *http.Server
	listenAddr string
}

// New creates a Server bound to addr that forwards queued downloads using ctx.
// ctx should be the Wails app context so download events reach the frontend.
func New(ctx context.Context, addr string) *Server {
	if strings.TrimSpace(addr) == "" {
		addr = DefaultAddr
	}
	return &Server{addr: addr, ctx: ctx}
}

// Start begins listening on the configured address. It is safe to call once;
// subsequent calls are no-ops.
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.server != nil {
		return nil
	}

	mux := s.handler()

	server := &http.Server{
		Addr:              s.addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
	}

	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", s.addr, err)
	}
	s.listenAddr = ln.Addr().String()

	s.server = server
	logging.Info("http server starting", "addr", s.listenAddr)
	go func() {
		if err := server.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logging.Error("http server stopped", "error", err)
		}
	}()
	return nil
}

// Addr returns the actual address the server is listening on, or "" before
// Start has been called.
func (s *Server) Addr() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.listenAddr
}

// handler builds the routing mux used by both the live server and tests.
func (s *Server) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleRoot)
	mux.HandleFunc("/health", s.handleHealth)
	return mux
}

// Stop shuts the server down gracefully.
func (s *Server) Stop() {
	s.mu.Lock()
	server := s.server
	s.server = nil
	s.mu.Unlock()
	if server == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logging.Warn("http server shutdown failed", "error", err)
	}
}

// handleRoot accepts POST requests carrying the extension's scan payload and
// queues each detected mod for download.
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body := http.MaxBytesReader(w, r.Body, maxBodyBytes)
	raw, err := io.ReadAll(body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "read body failed: "+err.Error())
		return
	}

	var payloads []remotePayload
	if err := json.Unmarshal(raw, &payloads); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}

	results := make([]appstructs.ModDownloadResult, 0, len(payloads))
	for i, p := range payloads {
		platform, ok := platformFromCode(p.P)
		if !ok {
			results = append(results, appstructs.ModDownloadResult{
				Skipped: true,
				Reason:  fmt.Sprintf("payload[%d]: invalid platform %q", i, p.P),
			})
			continue
		}

		// Prefer the explicit project id; fall back to slug when id is null.
		projectRef := ""
		var projectSlug string
		if p.ID != nil {
			if id := strings.TrimSpace(*p.ID); id != "" {
				projectRef = models.ProjectKey(platform, id)
			}
		}
		if projectRef == "" {
			slug := strings.TrimSpace(p.Slug)
			if slug == "" {
				results = append(results, appstructs.ModDownloadResult{
					Skipped: true,
					Reason:  fmt.Sprintf("payload[%d]: empty project id and slug", i),
				})
				continue
			}
			projectRef = models.ProjectKey(platform, slug)
			projectSlug = slug
		}

		versionID := strings.TrimSpace(p.File)
		req := appstructs.ModDownloadRequest{
			ProjectID: projectRef,
			Result: models.ModProject{
				ID:       projectRef,
				Platform: platform,
				Slug:     projectSlug,
			},
			VersionID: versionID,
		}
		result := downloader.QueueModDownload(s.ctx, req)
		logging.Info("http server queued mod",
			"platform", platform, "projectId", projectRef, "versionId", versionID,
			"queued", result.Queued, "skipped", result.Skipped, "reason", result.Reason)
		results = append(results, result)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(results)
}

// handleHealth is a lightweight liveness probe for the extension / user.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func platformFromCode(code string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(code)) {
	case "mr":
		return "modrinth", true
	case "cf":
		return "curseforge", true
	}
	return "", false
}

func writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
