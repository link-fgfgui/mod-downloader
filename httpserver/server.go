// Package httpserver exposes an HTTP endpoint that receives mod scan results
// from the mod-downloader browser extension, resolves their metadata, and
// displays them on the frontend download page.
//
// The extension (mod-downloader-chrome-plugin) POSTs a JSON array of payloads
// shaped as {"p":"mr"|"cf","id":null,"slug":"...","file":"..."} to
// http://127.0.0.1:18801. Each payload is resolved to full project metadata,
// filtered by the current instance's version and mod loader, and emitted to
// the frontend via a Wails event. Payloads carrying an explicit version ID
// that matches the current filter are auto-pinned.
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

	"mod-downloader/database"
	"mod-downloader/global"
	"mod-downloader/logging"
	"mod-downloader/models"
	"mod-downloader/providers"
	appstructs "mod-downloader/structs"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// DefaultAddr is the address the browser extension expects (see its manifest
// host_permissions and REMOTE_URL constant).
const DefaultAddr = "127.0.0.1:18801"

// maxBodyBytes limits the request body size to guard against abusive payloads.
const maxBodyBytes = 4 << 20 // 4 MiB

const extensionModsAcceptedEvent = "extension-mods-accepted"

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

// acceptedPayload holds a parsed and validated payload item ready for
// concurrent metadata resolution.
type acceptedPayload struct {
	index     int
	platform  string
	idOrSlug  string
	versionID string
}

// acceptResult is the per-item outcome of metadata resolution.
type acceptResult struct {
	index   int
	project models.ModProject
	pinned  bool
	skipped bool
	reason  string
}

// handleRoot accepts POST requests carrying the extension's scan payload,
// resolves metadata for each item, filters by the current instance's version
// and mod loader, emits matching projects to the frontend download page, and
// auto-pins explicit versions that match the filter.
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

	selected := global.GetSelectedVersion()
	mcVersion := strings.TrimSpace(selected.MinecraftVersion)
	modLoader := strings.ToLower(strings.TrimSpace(selected.ModLoader))

	// Phase 1: validate payloads and collect items for resolution.
	results := make([]appstructs.ModDownloadResult, len(payloads))
	var accepted []acceptedPayload
	for i, p := range payloads {
		platform, ok := platformFromCode(p.P)
		if !ok {
			results[i] = appstructs.ModDownloadResult{
				Skipped: true,
				Reason:  fmt.Sprintf("payload[%d]: invalid platform %q", i, p.P),
			}
			continue
		}

		idOrSlug := ""
		if p.ID != nil {
			idOrSlug = strings.TrimSpace(*p.ID)
		}
		if idOrSlug == "" {
			idOrSlug = strings.TrimSpace(p.Slug)
		}
		if idOrSlug == "" {
			results[i] = appstructs.ModDownloadResult{
				Skipped: true,
				Reason:  fmt.Sprintf("payload[%d]: empty project id and slug", i),
			}
			continue
		}

		accepted = append(accepted, acceptedPayload{
			index:     i,
			platform:  platform,
			idOrSlug:  idOrSlug,
			versionID: strings.TrimSpace(p.File),
		})
	}

	// Phase 2: resolve metadata concurrently.
	resolved := make([]acceptResult, len(accepted))
	var wg sync.WaitGroup
	for j, ap := range accepted {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resolved[j] = s.resolveAcceptedPayload(ap, mcVersion, modLoader)
		}()
	}
	wg.Wait()

	// Phase 3: collect filtered projects, build HTTP response.
	var filteredProjects []models.ModProject
	seen := make(map[string]bool)
	for _, ar := range resolved {
		results[ar.index] = appstructs.ModDownloadResult{
			Skipped: ar.skipped,
			Reason:  ar.reason,
			Queued:  !ar.skipped,
		}
		if ar.skipped || ar.project.ID == "" {
			continue
		}
		if seen[ar.project.ID] {
			continue
		}
		seen[ar.project.ID] = true
		filteredProjects = append(filteredProjects, ar.project)
	}

	// Phase 4: emit to frontend.
	if len(filteredProjects) > 0 && s.ctx != nil {
		runtime.EventsEmit(s.ctx, extensionModsAcceptedEvent, appstructs.SearchModsUpdate{
			Results: filteredProjects,
			Loading: false,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(results)
}

// resolveAcceptedPayload fetches metadata for a single payload item, checks
// version filter match, and auto-pins if the payload carries a matching
// explicit version.
func (s *Server) resolveAcceptedPayload(ap acceptedPayload, mcVersion, modLoader string) acceptResult {
	project, ok := providers.LookupProjectByPlatform(ap.platform, ap.idOrSlug, mcVersion, modLoader)
	if !ok {
		logging.Warn("http server project lookup failed", "platform", ap.platform, "idOrSlug", ap.idOrSlug)
		return acceptResult{index: ap.index, skipped: true, reason: "project not found"}
	}

	versions := providers.ListMatchingProjectVersions(project, mcVersion, modLoader)
	if len(versions) == 0 {
		logging.Info("http server project has no matching versions", "platform", ap.platform, "idOrSlug", ap.idOrSlug, "mcVersion", mcVersion, "modLoader", modLoader)
		return acceptResult{index: ap.index, skipped: true, reason: "no matching version"}
	}

	ar := acceptResult{index: ap.index, project: project}

	if ap.versionID != "" && mcVersion != "" && modLoader != "" {
		for _, v := range versions {
			if v.ID == ap.versionID {
				modID := extractModID(project)
				if modID != "" {
					if err := database.UpsertPinnedMod(database.PinnedMod{
						Platform:         strings.ToLower(strings.TrimSpace(project.Platform)),
						ModID:            modID,
						VersionID:        ap.versionID,
						MinecraftVersion: mcVersion,
						ModLoader:        modLoader,
					}); err != nil {
						logging.Error("http server auto-pin failed", "platform", project.Platform, "modID", modID, "versionID", ap.versionID, "error", err)
					} else {
						ar.pinned = true
						logging.Info("http server auto-pinned version", "platform", project.Platform, "modID", modID, "versionID", ap.versionID)
					}
				}
				break
			}
		}
	}

	return ar
}

// extractModID returns the platform-specific project ID from a ModProject,
// stripping the "platform:" prefix from the composite ID.
func extractModID(project models.ModProject) string {
	if project.ProjectID != "" {
		return strings.ToLower(strings.TrimSpace(project.ProjectID))
	}
	id := project.ID
	if idx := strings.Index(id, ":"); idx >= 0 {
		return strings.ToLower(strings.TrimSpace(id[idx+1:]))
	}
	return strings.ToLower(strings.TrimSpace(id))
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
