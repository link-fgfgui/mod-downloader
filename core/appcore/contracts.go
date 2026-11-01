// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

package appcore

import (
	"context"
	"log/slog"
	"sync"

	"github.com/link-fgfgui/mod-downloader/core/configs"
	"github.com/link-fgfgui/mod-downloader/core/models"
	"github.com/link-fgfgui/mod-downloader/core/storage"
	appstructs "github.com/link-fgfgui/mod-downloader/core/structs"
	appview "github.com/link-fgfgui/mod-downloader/core/structs/view"
)

// EventKind is the adapter-neutral event taxonomy emitted by Service.
type EventKind string

// Wire names are constants because Wails' binding generator evaluates
// RegisterEvent arguments at build time. EventKind.Name is the only reader.
const (
	EventNameSearchModsUpdated      = "search-mods-updated"
	EventNameExtensionModsAccepted  = "extension-mods-accepted"
	EventNameDownloadStatesUpdated  = "download-states-updated"
	EventNameDownloadQueueUpdated   = "download-queue-updated"
	EventNameDownloadFailed         = "download-failed"
	EventNameDownloadCompleted      = "download-completed"
	EventNameMinecraftDirChanged    = "minecraft-dir-changed"
	EventNameSelectedVersionChanged = "selected-version-changed"
	EventNameUsageStatsUpdated      = "usage-stats-updated"
)

// Event constants are the internal kinds; Name maps them to the kebab-case
// wire names in EventName*.
const (
	// EventSearchModsUpdated carries a streamed search page.
	EventSearchModsUpdated EventKind = "searchModsUpdated"
	// EventExtensionModsAccepted carries projects accepted from the browser
	// extension as a view.SearchModsUpdate payload.
	EventExtensionModsAccepted  EventKind = "extensionModsAccepted"
	EventDownloadStatesUpdated  EventKind = "downloadStatesUpdated"
	EventDownloadQueueUpdated   EventKind = "downloadQueueUpdated"
	EventDownloadFailed         EventKind = "downloadFailed"
	EventDownloadCompleted      EventKind = "downloadCompleted"
	EventMinecraftDirChanged    EventKind = "minecraftDirChanged"
	EventSelectedVersionChanged EventKind = "selectedVersionChanged"
	EventUsageStatsUpdated      EventKind = "usageStatsUpdated"
)

// Event is one typed core notification delivered to adapters.
type Event struct {
	Kind    EventKind
	Payload any
}

// Name returns the stable wire name for the event. Adapters use it as the
// frontend event name, so the kebab-case spelling is part of the contract.
func (k EventKind) Name() string {
	switch k {
	case EventSearchModsUpdated:
		return EventNameSearchModsUpdated
	case EventExtensionModsAccepted:
		return EventNameExtensionModsAccepted
	case EventDownloadStatesUpdated:
		return EventNameDownloadStatesUpdated
	case EventDownloadQueueUpdated:
		return EventNameDownloadQueueUpdated
	case EventDownloadFailed:
		return EventNameDownloadFailed
	case EventDownloadCompleted:
		return EventNameDownloadCompleted
	case EventMinecraftDirChanged:
		return EventNameMinecraftDirChanged
	case EventSelectedVersionChanged:
		return EventNameSelectedVersionChanged
	case EventUsageStatsUpdated:
		return EventNameUsageStatsUpdated
	default:
		return string(k)
	}
}

// StartupError identifies which startup stage failed so adapters can present a
// typed, diagnosable status instead of a chain of RPC rejections.
type StartupError struct {
	Stage string
	Err   error
}

func (e *StartupError) Error() string {
	return "startup failed at " + e.Stage + ": " + e.Err.Error()
}

func (e *StartupError) Unwrap() error {
	return e.Err
}

// Options configures New. Config may be nil, in which case the core loads it
// from disk or environment.
type Options struct {
	Config                *configs.Config
	ConfigOverrides       ConfigOverrides
	Runtime               RuntimeOptions
	Version               string
	LoadMinecraftReleases bool
	OnEvent               func(Event)
	// LogLevel is the minimum level emitted by the core logger. The zero value
	// is slog.LevelInfo.
	LogLevel slog.Level
	// LogToFileOnly keeps core logs out of stderr (the TUI owns the terminal).
	LogToFileOnly bool
}

// RuntimeOptions overrides paths and the initially selected instance.
type RuntimeOptions struct {
	WorkDir          string
	TargetModsDir    string
	MinecraftVersion string
	ModLoader        string
	CacheDir         string
	CachePath        string
	DefaultCacheDir  string
	NoConfigFile     bool
}

// ConfigOverrides lists adapter-supplied values that win over the config file.
// The Has* flags distinguish "override with empty" from "no override".
type ConfigOverrides struct {
	MinecraftDir        string
	CurseForgeAPIKey    string
	ModrinthAPIKey      string
	HasMinecraftDir     bool
	HasCurseForgeAPIKey bool
	HasModrinthAPIKey   bool
}

// FavoriteBulkAddRequest adds the same mods to several favorite lists.
type FavoriteBulkAddRequest struct {
	TargetListIDs []string              `json:"targetListIds"`
	Mods          []storage.FavoriteMod `json:"mods"`
}

// FavoriteListCopyRequest copies one favorite list into another.
type FavoriteListCopyRequest struct {
	SourceListID string `json:"sourceListId"`
	TargetListID string `json:"targetListId"`
}

// FavoriteBulkOperationResult summarizes a bulk favorite operation.
type FavoriteBulkOperationResult struct {
	Added   int      `json:"added"`
	Updated int      `json:"updated"`
	Skipped int      `json:"skipped"`
	Errors  []string `json:"errors,omitempty"`
}

// FavoriteCrossVersionCopyRequest copies a list into the scope described by
// MinecraftVersion and ModLoader.
type FavoriteCrossVersionCopyRequest struct {
	SourceListID     string `json:"sourceListId"`
	MinecraftVersion string `json:"minecraftVersion"`
	ModLoader        string `json:"modLoader"`
	IgnoreConflicts  bool   `json:"ignoreConflicts,omitempty"`
}

// FavoriteCrossVersionCopyPreview is the dry-run result of a cross-version
// copy, listing matched versions and conflicts before anything is written.
type FavoriteCrossVersionCopyPreview struct {
	SourceListID     string                             `json:"sourceListId"`
	TargetListName   string                             `json:"targetListName"`
	MinecraftVersion string                             `json:"minecraftVersion"`
	ModLoader        string                             `json:"modLoader"`
	NameConflict     bool                               `json:"nameConflict"`
	Matched          []FavoriteCrossVersionCopyMatch    `json:"matched"`
	Conflicts        []FavoriteCrossVersionCopyConflict `json:"conflicts"`
	Errors           []string                           `json:"errors,omitempty"`
}

// FavoriteCrossVersionCopyMatch pairs a source favorite with the target
// version that satisfies the new scope.
type FavoriteCrossVersionCopyMatch struct {
	Source  storage.FavoriteModEntry `json:"source"`
	Target  storage.FavoriteMod      `json:"target"`
	Project models.ModProject        `json:"project"`
	Version models.ModVersion        `json:"version"`
}

// FavoriteCrossVersionCopyConflict is a source favorite that has no compatible
// version in the target scope.
type FavoriteCrossVersionCopyConflict struct {
	Source storage.FavoriteModEntry `json:"source"`
	Reason string                   `json:"reason"`
}

// FavoriteCrossVersionCopyApplyResult reports the applied copy and the preview
// it was based on.
type FavoriteCrossVersionCopyApplyResult struct {
	Applied    bool                            `json:"applied"`
	TargetList storage.FavoriteList            `json:"targetList"`
	Preview    FavoriteCrossVersionCopyPreview `json:"preview"`
	Result     FavoriteBulkOperationResult     `json:"result"`
}

// InstallWaitResult is the terminal result of an install-and-wait call.
type InstallWaitResult struct {
	Result appstructs.ModDownloadResult     `json:"result"`
	State  appview.DownloadQueueState       `json:"state"`
	Errors []appstructs.DownloadFailedEvent `json:"errors,omitempty"`
}

// InstallToDirRequest installs one project directly into an arbitrary mods
// directory instead of the selected instance.
type InstallToDirRequest struct {
	ProjectID        string            `json:"projectId"`
	Project          models.ModProject `json:"project"`
	TargetDir        string            `json:"targetDir"`
	MinecraftVersion string            `json:"minecraftVersion"`
	ModLoader        string            `json:"modLoader"`
	VersionID        string            `json:"versionId,omitempty"`
}

// Service is the adapter-neutral workflow coordinator shared by Wails and CLI.
// The current implementation owns process-wide resources (providers, storage,
// logging, download queue), so a process must only run one active Service at a
// time. Startup rejects a second concurrent Service instead of corrupting the
// shared runtime.
type Service struct {
	ctx                context.Context
	config             *configs.Config
	options            Options
	modsWatchMu        sync.Mutex
	modsWatchRestartMu sync.Mutex
	modsWatchCancel    context.CancelFunc
	modsWatchDir       string
	modsWatchDone      chan struct{}
	lifecycleMu        sync.Mutex
	lifecycleState     serviceLifecycleState
	backgroundMu       sync.Mutex
	backgroundWG       sync.WaitGroup
	backgroundDone     bool
	saveErrMu          sync.Mutex
	lastSaveError      string
}

type serviceLifecycleState uint8

const (
	serviceStateIdle serviceLifecycleState = iota
	serviceStateStarting
	serviceStateStarted
)
