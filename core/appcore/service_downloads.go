// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

package appcore

import (
	"context"
	"errors"

	"github.com/link-fgfgui/mod-downloader/core/downloader"
	"github.com/link-fgfgui/mod-downloader/core/modbridge"
	"github.com/link-fgfgui/mod-downloader/core/storage"
	appstructs "github.com/link-fgfgui/mod-downloader/core/structs"
	appview "github.com/link-fgfgui/mod-downloader/core/structs/view"
)

// QueueModDownload enqueues a mod download together with its required
// dependencies.
func (s *Service) QueueModDownload(req appstructs.ModDownloadRequest) appstructs.ModDownloadResult {
	req = s.applyRuntimeDownloadRequest(req)
	return downloader.QueueModDownloadWithUserAgent(s.ctx, req, s.effectiveCurseForgeAPIKey(), s.UserAgent(), s.downloadEvents())
}

// InstallModAndWait queues a download and blocks until that request's queue
// group reaches a terminal state or ctx is canceled.
func (s *Service) InstallModAndWait(ctx context.Context, req appstructs.ModDownloadRequest) InstallWaitResult {
	req = s.applyRuntimeDownloadRequest(req)
	result := downloader.QueueModDownloadWithUserAgent(ctx, req, s.effectiveCurseForgeAPIKey(), s.UserAgent(), s.downloadEvents())
	if !result.Queued {
		return InstallWaitResult{Result: result, State: downloader.GetDownloadQueueState()}
	}
	if ctx == nil {
		ctx = context.Background()
	}

	// Wait for exactly this request (main job plus its required dependencies)
	// instead of the whole queue, so concurrent installs finish independently.
	status, err := downloader.WaitForDownload(ctx, result.QueueID)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			// A ctx that only bounds the wait must not report an install that is
			// still running as skipped: cancel the request's queue entries first.
			downloader.CancelDownloadGroup(result.QueueID, s.downloadEvents())
			final, waitErr := downloader.WaitForDownload(context.Background(), result.QueueID)
			if waitErr != nil {
				canceled := result
				canceled.Queued = false
				canceled.Reason = err.Error()
				return InstallWaitResult{Result: canceled, State: downloader.GetDownloadQueueState()}
			}
			status = final
		} else {
			final := result
			final.Queued = false
			final.Reason = err.Error()
			return InstallWaitResult{Result: final, State: downloader.GetDownloadQueueState()}
		}
	}

	final := result
	if status.Status != "completed" {
		final.Queued = false
		final.Reason = status.Reason
		if final.Reason == "" {
			final.Reason = status.Status
		}
	}
	return InstallWaitResult{
		Result: final,
		State:  downloader.GetDownloadQueueState(),
		Errors: status.Errors,
	}
}

// InstallModToDirAndWait installs a resolved project into an explicit target
// directory and waits for completion.
func (s *Service) InstallModToDirAndWait(ctx context.Context, req InstallToDirRequest) InstallWaitResult {
	return s.InstallModAndWait(ctx, appstructs.ModDownloadRequest{
		ProjectID:        req.ProjectID,
		Result:           req.Project,
		MinecraftVersion: req.MinecraftVersion,
		ModLoader:        req.ModLoader,
		VersionID:        req.VersionID,
		TargetDir:        req.TargetDir,
	})
}

// GetDownloadQueueState returns the current queue snapshot.
func (s *Service) GetDownloadQueueState() appview.DownloadQueueState {
	return downloader.GetDownloadQueueState()
}

// GetUsageStats returns the persisted usage counters.
func (s *Service) GetUsageStats() storage.UsageStats {
	return storage.GetUsageStats()
}

// CancelDownload cancels one active queue entry.
func (s *Service) CancelDownload(id string) bool {
	return downloader.CancelDownload(id, s.downloadEvents())
}

// RetryDownload requeues a failed or canceled queue entry.
func (s *Service) RetryDownload(id string) bool {
	return downloader.RetryDownload(s.ctx, id, s.downloadEvents())
}

// RemoveDownload removes a finished queue entry from the list.
func (s *Service) RemoveDownload(id string) bool {
	return downloader.RemoveDownload(id, s.downloadEvents())
}

// DismissOptionalDependencyReminder hides the optional-dependency prompt for
// one completed download.
func (s *Service) DismissOptionalDependencyReminder(id string) bool {
	return downloader.DismissOptionalDependencyReminder(id, s.downloadEvents())
}

// ClearOptionalDependencyReminders hides all optional-dependency prompts.
func (s *Service) ClearOptionalDependencyReminders() bool {
	return downloader.ClearOptionalDependencyReminders(s.downloadEvents())
}

// InstallOptionalDependencies queues the optional dependencies offered for one
// completed download.
func (s *Service) InstallOptionalDependencies(id string) []appstructs.ModDownloadResult {
	return downloader.InstallOptionalDependenciesWithUserAgent(s.ctx, id, s.effectiveCurseForgeAPIKey(), s.UserAgent(), s.downloadEvents())
}

// AnalyzeBatchIncompatibleConflicts reports conflicts that a batch install
// would introduce into the selected instance.
func (s *Service) AnalyzeBatchIncompatibleConflicts(req appstructs.BatchDownloadRequest) appstructs.BatchIncompatibleAnalysis {
	req = s.applyRuntimeBatchDownloadRequest(req)
	return modbridge.AnalyzeBatchIncompatibleConflicts(req)
}

// GetDownloadStates returns download button states for the requested projects.
func (s *Service) GetDownloadStates(req appstructs.DownloadStatesRequest) []appview.ModDownloadButtonState {
	req = s.applyRuntimeDownloadStatesRequest(req)
	return downloader.GetDownloadStates(req, func() {
		s.emit(EventDownloadStatesUpdated, nil)
	})
}
