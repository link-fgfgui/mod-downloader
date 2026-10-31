// Package httpserver exposes a thin HTTP transport for scan results produced by
// the mod-downloader browser extension.
//
// The extension (mod-downloader-chrome-plugin) POSTs a JSON array of payloads
// shaped as {"p":"mr"|"cf","id":null,"slug":"...","file":"..."} to
// http://127.0.0.1:18801. This package only enforces the request contract
// (method, body limit, JSON decoding and response encoding); metadata
// resolution, filtering and auto-pinning belong to the appcore application
// service passed through Options.Accept.
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

	"github.com/link-fgfgui/mod-downloader-core/logging"
	appstructs "github.com/link-fgfgui/mod-downloader-core/structs"
)

// DefaultAddr is the address the browser extension expects (see its manifest
// host_permissions and REMOTE_URL constant).
const DefaultAddr = "127.0.0.1:18801"

// maxBodyBytes limits the request body size to guard against abusive payloads.
const maxBodyBytes = 4 << 20 // 4 MiB

// AcceptFunc handles the extension workflow. It receives the decoded payloads
// and returns one result per payload in the same order.
type AcceptFunc func([]appstructs.ExtensionProjectPayload) []appstructs.ModDownloadResult

type Options struct {
	Accept AcceptFunc
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
// forwards the decoded payload list to the application service.
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

	var payloads []appstructs.ExtensionProjectPayload
	if err := json.Unmarshal(raw, &payloads); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json: "+err.Error())
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

func writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
