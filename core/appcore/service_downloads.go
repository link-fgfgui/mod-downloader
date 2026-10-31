package appcore

import (
	"context"
	"errors"

	"github.com/link-fgfgui/mod-downloader-core/downloader"
	"github.com/link-fgfgui/mod-downloader-core/storage"
	appstructs "github.com/link-fgfgui/mod-downloader-core/structs"
)

func (s *Service) QueueModDownload(req appstructs.ModDownloadRequest) appstructs.ModDownloadResult {
	req = s.applyRuntimeDownloadRequest(req)
	return downloader.QueueModDownloadWithUserAgent(s.ctx, req, s.effectiveCurseforgeAPIKey(), s.UserAgent(), s.downloadEvents())
}

func (s *Service) InstallModAndWait(ctx context.Context, req appstructs.ModDownloadRequest) InstallWaitResult {
	req = s.applyRuntimeDownloadRequest(req)
	result := downloader.QueueModDownloadWithUserAgent(ctx, req, s.effectiveCurseforgeAPIKey(), s.UserAgent(), s.downloadEvents())
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
			downloader.CancelDownloadGroup(context.Background(), result.QueueID, s.downloadEvents())
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

func (s *Service) GetDownloadQueueState() appstructs.DownloadQueueState {
	return downloader.GetDownloadQueueState()
}

func (s *Service) GetUsageStats() storage.UsageStats {
	return storage.GetUsageStats()
}

func (s *Service) CancelDownload(id string) bool {
	return downloader.CancelDownload(s.ctx, id, s.downloadEvents())
}

func (s *Service) RetryDownload(id string) bool {
	return downloader.RetryDownload(s.ctx, id, s.downloadEvents())
}

func (s *Service) RemoveDownload(id string) bool {
	return downloader.RemoveDownload(id, s.downloadEvents())
}

func (s *Service) DismissOptionalDependencyReminder(id string) bool {
	return downloader.DismissOptionalDependencyReminder(id, s.downloadEvents())
}

func (s *Service) ClearOptionalDependencyReminders() bool {
	return downloader.ClearOptionalDependencyReminders(s.downloadEvents())
}

func (s *Service) InstallOptionalDependencies(id string) []appstructs.ModDownloadResult {
	return downloader.InstallOptionalDependenciesWithUserAgent(s.ctx, id, s.effectiveCurseforgeAPIKey(), s.UserAgent(), s.downloadEvents())
}

func (s *Service) AnalyzeBatchIncompatibleConflicts(req appstructs.BatchDownloadRequest) appstructs.BatchIncompatibleAnalysis {
	req = s.applyRuntimeBatchDownloadRequest(req)
	return downloader.AnalyzeBatchIncompatibleConflicts(req)
}

func (s *Service) GetDownloadStates(req appstructs.DownloadStatesRequest) []appstructs.ModDownloadButtonState {
	req = s.applyRuntimeDownloadStatesRequest(req)
	return downloader.GetDownloadStates(req, func() {
		s.emit(EventDownloadStatesUpdated, nil)
	})
}
