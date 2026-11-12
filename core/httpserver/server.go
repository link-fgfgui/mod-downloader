// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

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

	"github.com/link-fgfgui/mod-downloader/core/logging"
	appstructs "github.com/link-fgfgui/mod-downloader/core/structs"
)

// DefaultAddr is the address the browser extension expects (see its manifest
// host_permissions and REMOTE_URL constant).
const DefaultAddr = "127.0.0.1:18801"

// APIVersion is exposed by /health and frozen as part of the extension
// protocol. Additive fields may be introduced within a version; incompatible
// request/response changes require a new path.
const APIVersion = 1

// ProjectsPath is the canonical project-submission endpoint.
const ProjectsPath = "/v1/extension/projects"

const (
	legacyProjectsPath   = "/"
	maxBodyBytes         = 4 << 20 // 4 MiB
	maxExtensionPayloads = 256
)

// AcceptFunc handles the extension workflow. It receives the decoded payloads
// and returns one result per payload in the same order.
type AcceptFunc func([]appstructs.ExtensionProjectPayload) []appstructs.ModDownloadResult

// Options configures the extension HTTP server.
type Options struct {
	Accept AcceptFunc
	// Listener optionally supplies an already-bound socket. The single-instance
	// guard binds the extension port before the window exists, so the HTTP
	// server must serve on that listener instead of binding Addr again.
	Listener net.Listener
}

// Server is the HTTP server accepting extension scan results.
type Server struct {
	addr    string
	options Options

	mu         sync.Mutex
	server     *http.Server
	listenAddr string
}

// New creates a Server bound to addr. Scan payloads are forwarded to
// Options.Accept; the HTTP layer never resolves metadata itself.
func New(addr string, options ...Options) *Server {
	if strings.TrimSpace(addr) == "" {
		addr = DefaultAddr
	}
	cfg := Options{}
	if len(options) > 0 {
		cfg = options[0]
	}
	return &Server{addr: addr, options: cfg}
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

	ln := s.options.Listener
	if ln == nil {
		var err error
		ln, err = net.Listen("tcp", s.addr)
		if err != nil {
			return fmt.Errorf("listen %s: %w", s.addr, err)
		}
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
	mux.HandleFunc(ProjectsPath, s.handleProjects)
	// Legacy endpoint used by extension builds before v1. It is handled by the
	// same authenticated handler and will be removed after the extension ships
	// the versioned path.
	mux.HandleFunc(legacyProjectsPath, s.handleProjects)
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

// handleProjects accepts POST requests carrying the extension's scan payload
// and forwards the decoded payload list to the application service.
func (s *Server) handleProjects(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != ProjectsPath && r.URL.Path != legacyProjectsPath {
		http.NotFound(w, r)
		return
	}
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if !allowedExtensionOrigin(origin) {
		writeError(w, http.StatusForbidden, "origin not allowed")
		return
	}
	setExtensionCORS(w, origin)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type"))); !strings.HasPrefix(contentType, "application/json") {
		writeError(w, http.StatusUnsupportedMediaType, "content-type must be application/json")
		return
	}

	body := http.MaxBytesReader(w, r.Body, maxBodyBytes)
	raw, err := io.ReadAll(body)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
			return
		}
		writeError(w, http.StatusBadRequest, "read body failed: "+err.Error())
		return
	}

	var payloads []appstructs.ExtensionProjectPayload
	if err := json.Unmarshal(raw, &payloads); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}
	if len(payloads) > maxExtensionPayloads {
		writeError(w, http.StatusRequestEntityTooLarge, "too many extension payloads")
		return
	}

	if s.options.Accept == nil {
		writeError(w, http.StatusServiceUnavailable, "extension receiver unavailable")
		return
	}

	results := s.options.Accept(payloads)
	if len(results) != len(payloads) {
		writeError(w, http.StatusInternalServerError, "extension receiver returned a mismatched result count")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Mod-Downloader-API-Version", fmt.Sprintf("%d", APIVersion))
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(results)
}

func allowedExtensionOrigin(origin string) bool {
	if origin == "" {
		return true
	}
	origin = strings.ToLower(origin)
	return strings.HasPrefix(origin, "chrome-extension://") ||
		strings.HasPrefix(origin, "moz-extension://")
}

func setExtensionCORS(w http.ResponseWriter, origin string) {
	w.Header().Add("Vary", "Origin")
	if origin == "" {
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

// handleHealth is a lightweight liveness probe for the extension / user.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprintf(w, `{"status":"ok","apiVersion":%d}`, APIVersion)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
