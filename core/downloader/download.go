package downloader

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	pathpkg "path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/link-fgfgui/mod-downloader-core/downloader/filetransfer"
	"github.com/link-fgfgui/mod-downloader-core/global"
	"github.com/link-fgfgui/mod-downloader-core/logging"
	"github.com/link-fgfgui/mod-downloader-core/minecraft"
	"github.com/link-fgfgui/mod-downloader-core/modbridge"
	"github.com/link-fgfgui/mod-downloader-core/models"
	"github.com/link-fgfgui/mod-downloader-core/providers"
	appstructs "github.com/link-fgfgui/mod-downloader-core/structs"
)

type Events struct {
	OnQueueState        func(appstructs.DownloadQueueState)
	OnDownloadFailed    func(appstructs.DownloadFailedEvent)
	OnDownloadCompleted func()
}

type downloadJob struct {
	ID string
	// WaitID groups the main job with the dependencies queued by the same
	// request. It stays stable across internal requeues so a caller can wait on
	// one queue entry instead of the whole active queue.
	WaitID            string
	ProjectID         string
	Version           models.ModVersion
	Result            models.ModProject
	TargetDir         string
	InstanceID        string
	MinecraftVersion  string
	ModLoader         string
	CurseForgeAPIKey  string
	UserAgent         string
	AutoAttempt       int
	IncompatiblePaths []global.LocalModFilePath
	visited           map[string]bool
	dependenciesDone  bool
	blockedBy         []string
	cancel            context.CancelFunc
}

type Config struct {
	FileConcurrency         int
	ConcurrentDownloads     int
	AdaptiveFileConcurrency bool
	TargetDownloadRateMiB   float64
	VerifySHA1              bool
}

type retryableDownloadJob struct {
	Job    downloadJob
	Status string
	Reason string
}

type downloadProgress struct {
	BytesComplete  int64
	TotalBytes     int64
	BytesPerSecond int64
}

type optionalDependencyReminderState struct {
	Snapshot appstructs.OptionalDependencyReminder
	Requests []appstructs.ModDownloadRequest
}

const (
	maxRetryableDownloadHistory = 20
	maxDownloadStallRetries     = 2
	maxDownloadSHA1Retries      = 2
)

var (
	errDownloadStalled               = errors.New("download stalled: no byte progress")
	errDownloadRequeued              = errors.New("download requeued after dependency analysis")
	errDownloadQueueUnavailable      = errors.New("download queue is unavailable")
	downloadStallTimeout             = 30 * time.Second
	downloadProgressPollInterval     = time.Second
	refreshMatchingProjectVersions   = providers.RefreshMatchingProjectVersions
	installStatusPrecise             = modbridge.InstallStatusPrecise
	newFileTransferBackend           = func() filetransfer.Backend { return filetransfer.NewStdlibBackend(nil) }
	downloadFileConcurrency          atomic.Int64
	downloadAdaptiveFileConcurrency  atomic.Bool
	downloadTargetRateBytesPerSecond atomic.Int64
	downloadVerifySHA1               atomic.Bool
)

type downloadGroup struct {
	waitID   string
	members  map[string]struct{}
	done     chan struct{}
	status   string
	reason   string
	errors   []appstructs.DownloadFailedEvent
	finished bool
}

// DownloadWaitResult is the terminal snapshot of one queued install request.
type DownloadWaitResult struct {
	Status string
	Reason string
	Errors []appstructs.DownloadFailedEvent
}

var downloadQueue = struct {
	sync.Mutex
	nextID            int64
	nextReminderID    int64
	pending           []downloadJob
	retryable         []retryableDownloadJob
	optionalReminders []optionalDependencyReminderState
	cancelRequested   map[string]struct{}
	finalizedCancel   map[string]struct{}
	restartRequested  map[string]struct{}
	workers           int
	maxConcurrent     int
	current           map[string]downloadJob
	progress          map[string]downloadProgress
	closed            bool
	groups            map[string]*downloadGroup
	finishedGroups    []string
	workerWG          sync.WaitGroup
}{}

// Start opens the package-level queue for enqueueing. The owning service must
// call it after its runtime (providers, storage, logging) is ready. It is
// idempotent so a service can be started again after a clean Shutdown.
func Start() {
	downloadQueue.Lock()
	downloadQueue.closed = false
	downloadQueue.groups = nil
	downloadQueue.finishedGroups = nil
	downloadQueue.Unlock()
}

// Shutdown closes the queue: it rejects new jobs, cancels running jobs, drops
// pending jobs, marks every waiter terminal and waits for all queue workers to
// exit. Callers must not use provider or storage resources after it returns.
func Shutdown() {
	downloadQueue.Lock()
	downloadQueue.closed = true

	cancels := make([]context.CancelFunc, 0, len(downloadQueue.current))
	for id, job := range downloadQueue.current {
		if job.cancel != nil {
			cancels = append(cancels, job.cancel)
		}
		delete(downloadQueue.progress, id)
		finishDownloadLocked(job, "canceled", "shutdown")
	}
	for _, job := range downloadQueue.pending {
		delete(downloadQueue.progress, job.ID)
		finishDownloadLocked(job, "canceled", "shutdown")
	}
	downloadQueue.pending = nil
	downloadQueue.retryable = nil
	downloadQueue.optionalReminders = nil
	downloadQueue.cancelRequested = nil
	downloadQueue.finalizedCancel = nil
	downloadQueue.restartRequested = nil
	for _, group := range downloadQueue.groups {
		finishDownloadGroupLocked(group, "canceled", "shutdown")
	}
	downloadQueue.Unlock()

	for _, cancel := range cancels {
		cancel()
	}
	downloadQueue.workerWG.Wait()
}

// IsClosed reports whether the queue currently rejects new jobs.
func IsClosed() bool {
	downloadQueue.Lock()
	defer downloadQueue.Unlock()
	return downloadQueue.closed
}

// WaitForDownload blocks until the queue entry identified by queueID reaches a
// terminal state ("completed", "failed", "canceled" or "removed"). The ctx
// argument only bounds the wait; it does not cancel the download.
func WaitForDownload(ctx context.Context, queueID string) (DownloadWaitResult, error) {
	id := strings.TrimSpace(queueID)
	if id == "" {
		return DownloadWaitResult{}, errors.New("missing download queue id")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	downloadQueue.Lock()
	group := downloadQueue.groups[id]
	if group == nil {
		closed := downloadQueue.closed
		downloadQueue.Unlock()
		if closed {
			return DownloadWaitResult{}, errDownloadQueueUnavailable
		}
		return DownloadWaitResult{}, fmt.Errorf("unknown download queue id %q", id)
	}
	done := group.done
	if group.finished {
		result := downloadGroupResultLocked(group)
		downloadQueue.Unlock()
		return result, nil
	}
	if downloadQueue.closed {
		// Shutdown always finishes known groups; this only covers a waiter that
		// won the race against a group that is being registered.
		downloadQueue.Unlock()
		return DownloadWaitResult{Status: "canceled", Reason: "shutdown"}, nil
	}
	downloadQueue.Unlock()

	select {
	case <-ctx.Done():
		return DownloadWaitResult{}, ctx.Err()
	case <-done:
	}

	downloadQueue.Lock()
	result := downloadGroupResultLocked(group)
	downloadQueue.Unlock()
	return result, nil
}

func downloadGroupResultLocked(group *downloadGroup) DownloadWaitResult {
	return DownloadWaitResult{
		Status: group.status,
		Reason: group.reason,
		Errors: append([]appstructs.DownloadFailedEvent(nil), group.errors...),
	}
}

// CancelDownloadGroup cancels every active queue entry that belongs to the
// same install request (the main job and its required dependencies).
func CancelDownloadGroup(ctx context.Context, queueID string, events ...Events) bool {
	id := strings.TrimSpace(queueID)
	if id == "" {
		return false
	}
	eventSink := firstEvents(events)
	cancels := make([]context.CancelFunc, 0, 1)
	changed := false

	downloadQueue.Lock()
	group := downloadQueue.groups[id]
	if group == nil || group.finished {
		downloadQueue.Unlock()
		return false
	}
	for memberID := range group.members {
		for i, job := range downloadQueue.pending {
			if job.ID != memberID {
				continue
			}
			downloadQueue.pending = append(downloadQueue.pending[:i], downloadQueue.pending[i+1:]...)
			delete(downloadQueue.progress, job.ID)
			appendRetryableDownloadLocked(job, "canceled", "canceled")
			finishDownloadLocked(job, "canceled", "canceled")
			changed = true
			break
		}
		job, ok := downloadQueue.current[memberID]
		if !ok || job.cancel == nil {
			continue
		}
		appendRetryableDownloadLocked(job, "canceled", "canceled")
		delete(downloadQueue.progress, memberID)
		delete(downloadQueue.current, memberID)
		if downloadQueue.finalizedCancel == nil {
			downloadQueue.finalizedCancel = make(map[string]struct{})
		}
		downloadQueue.finalizedCancel[memberID] = struct{}{}
		cancels = append(cancels, job.cancel)
		finishDownloadLocked(job, "canceled", "canceled")
		changed = true
	}
	downloadQueue.Unlock()

	if !changed {
		return false
	}
	for _, cancel := range cancels {
		cancel()
	}
	emitDownloadQueueState(eventSink)
	return true
}

func ensureDownloadGroupLocked(waitID string) *downloadGroup {
	if downloadQueue.groups == nil {
		downloadQueue.groups = make(map[string]*downloadGroup)
	}
	if group, ok := downloadQueue.groups[waitID]; ok && !group.finished {
		return group
	}
	group := &downloadGroup{
		waitID:  waitID,
		members: make(map[string]struct{}),
		done:    make(chan struct{}),
	}
	downloadQueue.groups[waitID] = group
	return group
}

func registerDownloadJobLocked(job *downloadJob) {
	assignDownloadIDLocked(job)
	if strings.TrimSpace(job.WaitID) == "" {
		job.WaitID = job.ID
	}
	ensureDownloadGroupLocked(job.WaitID).members[job.ID] = struct{}{}
	registerDownloadProgressLocked(*job)
}

func finishDownloadLocked(job downloadJob, status, reason string) {
	group := downloadQueue.groups[job.WaitID]
	if group == nil {
		return
	}
	delete(group.members, job.ID)
	if job.ID != job.WaitID || group.finished {
		return
	}
	finishDownloadGroupLocked(group, status, reason)
}

func finishDownloadGroupLocked(group *downloadGroup, status, reason string) {
	if group.finished {
		return
	}
	group.finished = true
	group.status = strings.TrimSpace(status)
	group.reason = strings.TrimSpace(reason)
	close(group.done)
	downloadQueue.finishedGroups = append(downloadQueue.finishedGroups, group.waitID)
	for len(downloadQueue.finishedGroups) > maxRetryableDownloadHistory {
		delete(downloadQueue.groups, downloadQueue.finishedGroups[0])
		downloadQueue.finishedGroups = downloadQueue.finishedGroups[1:]
	}
}

func Configure(config Config) {
	fileConcurrency := config.FileConcurrency
	if fileConcurrency <= 0 {
		fileConcurrency = filetransfer.DefaultConcurrency
	}
	concurrentDownloads := config.ConcurrentDownloads
	if concurrentDownloads <= 0 {
		concurrentDownloads = 1
	}
	downloadFileConcurrency.Store(int64(fileConcurrency))
	downloadAdaptiveFileConcurrency.Store(config.AdaptiveFileConcurrency)
	targetRateBytesPerSecond := int64(config.TargetDownloadRateMiB * 1024 * 1024)
	if targetRateBytesPerSecond <= 0 {
		targetRateBytesPerSecond = int64(1024 * 1024)
	}
	downloadTargetRateBytesPerSecond.Store(targetRateBytesPerSecond)
	downloadVerifySHA1.Store(config.VerifySHA1)
	downloadQueue.Lock()
	downloadQueue.maxConcurrent = concurrentDownloads
	downloadQueue.Unlock()
}

// ConfigureSimpleMode applies the shared remote-metadata policy and clears
// optional-dependency reminders that are no longer valid in this mode.
func ConfigureSimpleMode(enabled bool, events ...Events) {
	modbridge.ConfigureSimpleMode(enabled)
	if enabled {
		ClearOptionalDependencyReminders(events...)
	}
}

func configuredAdaptiveFileConcurrency() (bool, int64) {
	return downloadAdaptiveFileConcurrency.Load(), downloadTargetRateBytesPerSecond.Load()
}

func configuredFileConcurrency() int {
	value := int(downloadFileConcurrency.Load())
	if value <= 0 {
		return filetransfer.DefaultConcurrency
	}
	return value
}

func configuredVerifySHA1() bool {
	return downloadVerifySHA1.Load()
}

func QueueModDownload(ctx context.Context, req appstructs.ModDownloadRequest, curseForgeAPIKey string, events ...Events) appstructs.ModDownloadResult {
	return QueueModDownloadWithUserAgent(ctx, req, curseForgeAPIKey, "", events...)
}

func QueueModDownloadWithUserAgent(ctx context.Context, req appstructs.ModDownloadRequest, curseForgeAPIKey string, userAgent string, events ...Events) appstructs.ModDownloadResult {
	return queueModDownload(ctx, req, curseForgeAPIKey, userAgent, make(map[string]bool), firstEvents(events))
}

func queueModDownload(ctx context.Context, req appstructs.ModDownloadRequest, curseForgeAPIKey string, userAgent string, visited map[string]bool, events Events) appstructs.ModDownloadResult {
	job, result, ok := buildDownloadJob(req, curseForgeAPIKey, userAgent, visited)
	if !ok {
		return result
	}
	queueID, queued := enqueueDownload(ctx, job, events)
	if !queued {
		result.Queued = false
		result.Skipped = false
		result.Reason = errDownloadQueueUnavailable.Error()
		return result
	}
	result.QueueID = queueID
	return result
}

func buildDownloadJob(req appstructs.ModDownloadRequest, curseForgeAPIKey string, userAgent string, visited map[string]bool) (downloadJob, appstructs.ModDownloadResult, bool) {
	req.MinecraftVersion = strings.TrimSpace(req.MinecraftVersion)
	req.ModLoader = strings.ToLower(strings.TrimSpace(req.ModLoader))
	req.ProjectID = strings.TrimSpace(req.ProjectID)
	if req.ProjectID == "" {
		req.ProjectID = providers.ProjectReferenceFromSearchResult(req.Result)
	}

	req, instanceID, targetDir, ok := modbridge.ResolveInstallTarget(req)
	if !ok {
		return downloadJob{}, appstructs.ModDownloadResult{Skipped: true, Reason: "no install target"}, false
	}
	if req.ProjectID == "" || req.MinecraftVersion == "" || req.ModLoader == "" {
		return downloadJob{}, appstructs.ModDownloadResult{Skipped: true, Reason: "invalid request"}, false
	}

	version, ok := modbridge.ResolveVersion(req)
	if !ok {
		return downloadJob{}, appstructs.ModDownloadResult{Skipped: true, Reason: "no matching version"}, false
	}
	version.DownloadURL = providers.EffectiveDownloadURL(version)
	if strings.TrimSpace(version.DownloadURL) == "" {
		return downloadJob{}, appstructs.ModDownloadResult{Skipped: true, Reason: "missing download url"}, false
	}
	if visited == nil {
		visited = make(map[string]bool)
	}
	if key := projectVersionJobKey(version); key != "" {
		if visited[key] {
			return downloadJob{}, appstructs.ModDownloadResult{Skipped: true, Reason: "already queued"}, false
		}
		visited[key] = true
	}

	return downloadJob{
			ProjectID:        req.ProjectID,
			Version:          version,
			Result:           req.Result,
			TargetDir:        targetDir,
			InstanceID:       instanceID,
			MinecraftVersion: req.MinecraftVersion,
			ModLoader:        req.ModLoader,
			CurseForgeAPIKey: curseForgeAPIKey,
			UserAgent:        normalizedUserAgent(userAgent),
			visited:          visited,
		}, appstructs.ModDownloadResult{
			Queued:    true,
			FileName:  version.FileName,
			VersionID: version.ID,
		}, true
}

func GetDownloadQueueState() appstructs.DownloadQueueState {
	return currentDownloadQueueState()
}

func CancelDownload(ctx context.Context, id string, events ...Events) bool {
	id = strings.TrimSpace(id)
	if id == "" {
		return false
	}
	eventSink := firstEvents(events)

	downloadQueue.Lock()
	for i, job := range downloadQueue.pending {
		if job.ID != id {
			continue
		}
		downloadQueue.pending = append(downloadQueue.pending[:i], downloadQueue.pending[i+1:]...)
		delete(downloadQueue.progress, job.ID)
		appendRetryableDownloadLocked(job, "canceled", "canceled")
		finishDownloadLocked(job, "canceled", "canceled")
		downloadQueue.Unlock()
		emitDownloadQueueState(eventSink)
		return true
	}

	var cancel context.CancelFunc
	if job, ok := downloadQueue.current[id]; ok {
		cancel = job.cancel
		if cancel != nil {
			appendRetryableDownloadLocked(job, "canceled", "canceled")
			delete(downloadQueue.progress, id)
			delete(downloadQueue.current, id)
			if downloadQueue.finalizedCancel == nil {
				downloadQueue.finalizedCancel = make(map[string]struct{})
			}
			downloadQueue.finalizedCancel[id] = struct{}{}
			finishDownloadLocked(job, "canceled", "canceled")
		}
	}
	downloadQueue.Unlock()

	if cancel == nil {
		return false
	}
	cancel()
	emitDownloadQueueState(eventSink)
	return true
}

func RetryDownload(ctx context.Context, id string, events ...Events) bool {
	id = strings.TrimSpace(id)
	if id == "" {
		return false
	}
	eventSink := firstEvents(events)

	downloadQueue.Lock()
	index := -1
	var job downloadJob
	for i, item := range downloadQueue.retryable {
		if item.Job.ID != id {
			continue
		}
		index = i
		job = item.Job
		break
	}
	if index >= 0 {
		downloadQueue.retryable = append(downloadQueue.retryable[:index], downloadQueue.retryable[index+1:]...)
		delete(downloadQueue.progress, id)
		downloadQueue.Unlock()

		job.ID = ""
		job.WaitID = ""
		job.cancel = nil
		job.AutoAttempt = 0
		enqueueDownload(ctx, job, eventSink)
		return true
	}

	var cancel context.CancelFunc
	if job, ok := downloadQueue.current[id]; ok {
		cancel = job.cancel
		if cancel != nil {
			if downloadQueue.restartRequested == nil {
				downloadQueue.restartRequested = make(map[string]struct{})
			}
			downloadQueue.restartRequested[id] = struct{}{}
		}
	}
	downloadQueue.Unlock()

	if cancel == nil {
		return false
	}
	cancel()
	emitDownloadQueueState(eventSink)
	return true
}

func RemoveDownload(id string, events ...Events) bool {
	id = strings.TrimSpace(id)
	if id == "" {
		return false
	}
	eventSink := firstEvents(events)

	downloadQueue.Lock()
	removed := false
	for i, item := range downloadQueue.retryable {
		if item.Job.ID != id || (item.Status != "canceled" && item.Status != "failed") {
			continue
		}
		downloadQueue.retryable = append(downloadQueue.retryable[:i], downloadQueue.retryable[i+1:]...)
		removed = true
		break
	}
	if !removed {
		for i, job := range downloadQueue.pending {
			if job.ID != id {
				continue
			}
			downloadQueue.pending = append(downloadQueue.pending[:i], downloadQueue.pending[i+1:]...)
			finishDownloadLocked(job, "removed", "removed")
			removed = true
			break
		}
	}
	if removed {
		delete(downloadQueue.progress, id)
	}
	downloadQueue.Unlock()

	if !removed {
		return false
	}
	emitDownloadQueueState(eventSink)
	return true
}

func DismissOptionalDependencyReminder(id string, events ...Events) bool {
	id = strings.TrimSpace(id)
	if id == "" {
		return false
	}
	eventSink := firstEvents(events)
	downloadQueue.Lock()
	index := -1
	for i, reminder := range downloadQueue.optionalReminders {
		if reminder.Snapshot.ID == id {
			index = i
			break
		}
	}
	if index < 0 {
		downloadQueue.Unlock()
		return false
	}
	downloadQueue.optionalReminders = append(downloadQueue.optionalReminders[:index], downloadQueue.optionalReminders[index+1:]...)
	downloadQueue.Unlock()
	emitDownloadQueueState(eventSink)
	return true
}

func ClearOptionalDependencyReminders(events ...Events) bool {
	eventSink := firstEvents(events)
	downloadQueue.Lock()
	if len(downloadQueue.optionalReminders) == 0 {
		downloadQueue.Unlock()
		return false
	}
	downloadQueue.optionalReminders = nil
	downloadQueue.Unlock()
	emitDownloadQueueState(eventSink)
	return true
}

func InstallOptionalDependencies(ctx context.Context, id string, curseForgeAPIKey string, events ...Events) []appstructs.ModDownloadResult {
	return InstallOptionalDependenciesWithUserAgent(ctx, id, curseForgeAPIKey, "", events...)
}

func InstallOptionalDependenciesWithUserAgent(ctx context.Context, id string, curseForgeAPIKey string, userAgent string, events ...Events) []appstructs.ModDownloadResult {
	if modbridge.SimpleModeEnabled() {
		return nil
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}
	eventSink := firstEvents(events)
	downloadQueue.Lock()
	index := -1
	var requests []appstructs.ModDownloadRequest
	for i, reminder := range downloadQueue.optionalReminders {
		if reminder.Snapshot.ID != id {
			continue
		}
		index = i
		requests = append(requests, reminder.Requests...)
		break
	}
	if index >= 0 {
		downloadQueue.optionalReminders = append(downloadQueue.optionalReminders[:index], downloadQueue.optionalReminders[index+1:]...)
	}
	downloadQueue.Unlock()
	if index < 0 {
		return nil
	}
	emitDownloadQueueState(eventSink)

	results := make([]appstructs.ModDownloadResult, 0, len(requests))
	visited := make(map[string]bool)
	for _, req := range requests {
		result := queueModDownload(ctx, req, curseForgeAPIKey, userAgent, visited, eventSink)
		results = append(results, result)
	}
	return results
}

func AnalyzeBatchIncompatibleConflicts(req appstructs.BatchDownloadRequest) appstructs.BatchIncompatibleAnalysis {
	return modbridge.AnalyzeBatchIncompatibleConflicts(req)
}

func GetDownloadStates(req appstructs.DownloadStatesRequest, onBackfillComplete func()) []appstructs.ModDownloadButtonState {
	states := modbridge.DownloadStates(req, onBackfillComplete)
	queued := queuedProjectKeys()
	for i := range states {
		if queued[states[i].Key] {
			states[i].Loading = true
		}
	}
	return states
}

// queuedProjectKeys returns the set of project reference keys for all jobs
// currently in the download queue (both pending and running).
func queuedProjectKeys() map[string]bool {
	downloadQueue.Lock()
	defer downloadQueue.Unlock()
	keys := make(map[string]bool, len(downloadQueue.pending)+len(downloadQueue.current))
	for _, job := range downloadQueue.current {
		if k := providers.ProjectReferenceFromSearchResult(job.Result); k != "" {
			keys[k] = true
		}
	}
	for _, job := range downloadQueue.pending {
		if k := providers.ProjectReferenceFromSearchResult(job.Result); k != "" {
			keys[k] = true
		}
	}
	return keys
}

func projectVersionJobKey(version models.ModVersion) string {
	platform := strings.ToLower(strings.TrimSpace(version.Platform))
	projectID := strings.TrimSpace(version.ProjectID)
	versionID := strings.TrimSpace(version.ID)
	if platform == "" && projectID == "" && versionID == "" {
		return ""
	}
	return platform + ":" + projectID + ":" + versionID
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func enqueueDownload(ctx context.Context, job downloadJob, events Events) (string, bool) {
	downloadQueue.Lock()
	if downloadQueue.closed {
		downloadQueue.Unlock()
		return "", false
	}
	if len(downloadQueue.current) == 0 && len(downloadQueue.pending) == 0 {
		downloadQueue.progress = nil
	}
	registerDownloadJobLocked(&job)
	downloadQueue.pending = append(downloadQueue.pending, job)
	workersToStart := reserveDownloadWorkersLocked()
	if workersToStart > 0 {
		downloadQueue.workerWG.Add(workersToStart)
	}
	queueID := job.ID
	downloadQueue.Unlock()

	emitDownloadQueueState(events)
	for range workersToStart {
		go func() {
			defer downloadQueue.workerWG.Done()
			runDownloadQueue(ctx, events)
		}()
	}
	return queueID, true
}

func reserveDownloadWorkersLocked() int {
	limit := downloadQueue.maxConcurrent
	if limit <= 0 {
		limit = 1
	}
	count := min(limit-downloadQueue.workers, len(downloadQueue.pending))
	if count <= 0 {
		return 0
	}
	downloadQueue.workers += count
	return count
}

func prependPendingDownloads(jobs []downloadJob, events Events) bool {
	if len(jobs) == 0 {
		return true
	}
	downloadQueue.Lock()
	if downloadQueue.closed {
		downloadQueue.Unlock()
		return false
	}
	for i := range jobs {
		jobs[i].cancel = nil
		registerDownloadJobLocked(&jobs[i])
	}
	downloadQueue.pending = append(append([]downloadJob(nil), jobs...), downloadQueue.pending...)
	downloadQueue.Unlock()
	emitDownloadQueueState(events)
	return true
}

func assignDownloadIDLocked(job *downloadJob) {
	if strings.TrimSpace(job.ID) != "" {
		return
	}
	downloadQueue.nextID++
	job.ID = fmt.Sprintf("download-%d", downloadQueue.nextID)
}

func registerDownloadProgressLocked(job downloadJob) {
	if downloadQueue.progress == nil {
		downloadQueue.progress = make(map[string]downloadProgress)
	}
	total := job.Version.FileSize
	if total < 0 {
		total = 0
	}
	downloadQueue.progress[job.ID] = downloadProgress{TotalBytes: total}
}

func removeDownloadProgress(id string) {
	downloadQueue.Lock()
	delete(downloadQueue.progress, id)
	downloadQueue.Unlock()
}

func removeCurrentDownload(id string) {
	downloadQueue.Lock()
	delete(downloadQueue.current, id)
	if len(downloadQueue.current) == 0 && len(downloadQueue.pending) == 0 {
		downloadQueue.progress = nil
	}
	downloadQueue.Unlock()
}

func completeDownloadProgress(id string) {
	downloadQueue.Lock()
	progress, ok := downloadQueue.progress[id]
	if ok {
		if progress.TotalBytes > 0 {
			progress.BytesComplete = progress.TotalBytes
		}
		progress.BytesPerSecond = 0
		downloadQueue.progress[id] = progress
	}
	downloadQueue.Unlock()
}

func updateDownloadProgress(id string, value filetransfer.Progress, bytesPerSecond int64, events Events) {
	downloadQueue.Lock()
	progress, ok := downloadQueue.progress[id]
	if !ok {
		downloadQueue.Unlock()
		return
	}
	if value.TotalBytes > 0 {
		progress.TotalBytes = value.TotalBytes
	}
	if value.BytesComplete >= 0 {
		progress.BytesComplete = value.BytesComplete
	}
	if progress.TotalBytes > 0 && progress.BytesComplete > progress.TotalBytes {
		progress.BytesComplete = progress.TotalBytes
	}
	if bytesPerSecond < 0 {
		bytesPerSecond = 0
	}
	progress.BytesPerSecond = bytesPerSecond
	downloadQueue.progress[id] = progress
	downloadQueue.Unlock()
	emitDownloadQueueState(events)
}

func runDownloadQueue(ctx context.Context, events Events) {
	parentCtx := ctx
	if parentCtx == nil {
		parentCtx = context.Background()
	}
	for {
		downloadQueue.Lock()
		if downloadQueue.closed {
			downloadQueue.workers--
			downloadQueue.Unlock()
			emitDownloadQueueState(events)
			return
		}
		job, ok := takeNextDownloadLocked()
		if !ok {
			downloadQueue.workers--
			downloadQueue.Unlock()
			emitDownloadQueueState(events)
			return
		}
		jobCtx, cancel := context.WithCancel(parentCtx)
		job.cancel = cancel
		if downloadQueue.current == nil {
			downloadQueue.current = make(map[string]downloadJob)
		}
		// Publish the running job before releasing the lock so Shutdown can
		// always see and cancel jobs that left the pending list.
		downloadQueue.current[job.ID] = job
		downloadQueue.Unlock()
		emitDownloadQueueState(events)

		installed, err := downloadModJob(jobCtx, job, events)
		if installed && events.OnDownloadCompleted != nil {
			events.OnDownloadCompleted()
		}
		if consumeFinalizedDownloadCancel(job.ID) {
			cancel()
			removeCurrentDownload(job.ID)
			emitDownloadQueueState(events)
			continue
		}
		if err != nil {
			if errors.Is(err, context.Canceled) {
				logging.Info("download mod canceled", "fileName", job.Version.FileName, "versionID", job.Version.ID, "targetDir", job.TargetDir)
				removeDownloadProgress(job.ID)
				if consumeDownloadRestartRequest(job.ID) {
					job.AutoAttempt = 0
					enqueueDownload(parentCtx, job, events)
				} else if consumeDownloadCancelRequest(job.ID) {
					appendRetryableDownload(job, "canceled", "canceled")
					finishDownload(job, "canceled", "canceled")
				} else {
					finishDownload(job, "canceled", "canceled")
				}
			} else if errors.Is(err, errDownloadRequeued) {
				removeDownloadProgress(job.ID)
				logging.Debug("download mod requeued after dependency analysis", "fileName", job.Version.FileName, "versionID", job.Version.ID, "targetDir", job.TargetDir)
			} else if errors.Is(err, errDownloadStalled) && retryStalledDownload(parentCtx, job, events) {
				logging.Warn("download mod stalled, retrying", "fileName", job.Version.FileName, "versionID", job.Version.ID, "targetDir", job.TargetDir, "attempt", job.AutoAttempt+1, "maxRetries", maxDownloadStallRetries)
			} else {
				removeDownloadProgress(job.ID)
				logging.Error("download mod failed", "fileName", job.Version.FileName, "versionID", job.Version.ID, "targetDir", job.TargetDir, "error", err)
				appendRetryableDownload(job, "failed", err.Error())
				downloadQueue.Lock()
				if group := downloadQueue.groups[job.WaitID]; group != nil {
					group.errors = append(group.errors, appstructs.DownloadFailedEvent{
						FileName:  job.Version.FileName,
						VersionID: job.Version.ID,
						Reason:    err.Error(),
					})
				}
				finishDownloadLocked(job, "failed", err.Error())
				downloadQueue.Unlock()
				emitDownloadFailed(events, job, err)
			}
		} else {
			completeDownloadProgress(job.ID)
			finishDownload(job, "completed", "")
			emitDownloadQueueState(events)
		}
		cancel()
		removeCurrentDownload(job.ID)
		emitDownloadQueueState(events)
	}
}

func finishDownload(job downloadJob, status, reason string) {
	downloadQueue.Lock()
	finishDownloadLocked(job, status, reason)
	downloadQueue.Unlock()
}

func takeNextDownloadLocked() (downloadJob, bool) {
	for index, job := range downloadQueue.pending {
		if downloadJobBlockedLocked(job) {
			continue
		}
		downloadQueue.pending = append(downloadQueue.pending[:index], downloadQueue.pending[index+1:]...)
		return job, true
	}
	return downloadJob{}, false
}

func downloadJobBlockedLocked(job downloadJob) bool {
	for _, key := range job.blockedBy {
		for _, current := range downloadQueue.current {
			if projectVersionJobKey(current.Version) == key {
				return true
			}
		}
		for _, pending := range downloadQueue.pending {
			if pending.ID != job.ID && projectVersionJobKey(pending.Version) == key {
				return true
			}
		}
	}
	return false
}

func downloadModJob(ctx context.Context, job downloadJob, events Events) (bool, error) {
	if err := os.MkdirAll(job.TargetDir, 0o755); err != nil {
		return false, err
	}
	if !job.dependenciesDone {
		dependencies, preparedJob, err := prepareDownloadDependencies(ctx, job, events)
		if err != nil {
			return false, err
		}
		if err := ctxErr(ctx); err != nil {
			return false, err
		}
		if len(dependencies) > 0 {
			preparedJob.blockedBy = make([]string, 0, len(dependencies))
			for _, dependency := range dependencies {
				if key := projectVersionJobKey(dependency.Version); key != "" {
					preparedJob.blockedBy = append(preparedJob.blockedBy, key)
				}
			}
			if !prependPendingDownloads(append(dependencies, preparedJob), events) {
				// Shutdown started while dependency analysis was running.
				return false, context.Canceled
			}
			return false, errDownloadRequeued
		}
		job = preparedJob
	}
	archiveIncompatibleModJars(job.IncompatiblePaths)
	if handled, installed := tryHardlinkInstall(job); handled {
		return installed, nil
	}
	// Try to get mod IDs from platform version (persisted or lazy-parsed via modbridge)
	modIDs := modbridge.VersionModIDs(job.Version, job.ModLoader)
	if len(modIDs) > 0 {
		existing := modbridge.LocalModPathsForModIDs(modIDs, job.InstanceID)
		if alreadyInstalled(existing, job.Version.SHA1) {
			return false, nil
		}
		return downloadModToTarget(ctx, job, existing, events)
	}

	return downloadModWithLocalParse(ctx, job, events)
}

// tryHardlinkInstall consults the SHA1→path hardlink index before downloading.
// On a hit with an existing source file it hardlinks (falling back to a no-overwrite
// copy) into the target location and fires a discard fetch in the background to
// preserve the mod author's download count.
// SHA1 is always available before download, so this works whether or not mod IDs
// are resolvable: with mod IDs it links straight to the final path; without them it
// links to a temp dir, parses locally, then moves to the final path.
func tryHardlinkInstall(job downloadJob) (bool, bool) {
	srcPath, ok := global.HardlinkIndexLookup(job.Version.SHA1)
	if !ok {
		return false, false
	}
	finalPath := filepath.Join(job.TargetDir, filepath.Base(srcPath))
	if job.Version.FileName != "" {
		finalPath = filepath.Join(job.TargetDir, job.Version.FileName)
	}

	modIDs := modbridge.VersionModIDs(job.Version, job.ModLoader)
	var existing []global.LocalModFilePath
	if len(modIDs) > 0 {
		existing = modbridge.LocalModPathsForModIDs(modIDs, job.InstanceID)
		if alreadyInstalled(existing, job.Version.SHA1) {
			return true, false
		}
	}

	if downloadTargetExists(finalPath) && !pathInLocalModPaths(finalPath, existing) {
		logging.Info("hardlink skipped because target file already exists", "path", finalPath, "versionID", job.Version.ID)
		return true, false
	}
	archiveSupersededModJars(existing)
	if downloadTargetExists(finalPath) {
		logging.Info("hardlink skipped because target file already exists after archiving", "path", finalPath, "versionID", job.Version.ID)
		return true, false
	}

	if len(modIDs) > 0 {
		if err := minecraft.CreateHardLinkOrCopy(srcPath, finalPath); err != nil {
			logging.Warn("hardlink creation failed, falling back to download", "src", srcPath, "dst", finalPath, "error", err)
			return false, false
		}
		logging.Info("mod installed via hardlink/copy", "src", srcPath, "dst", finalPath, "versionID", job.Version.ID)
		go DiscardFetchFromNetwork(job.Version.DownloadURL, job.CurseForgeAPIKey, job.UserAgent)
		upsertDownloadedMod(finalPath, job)
		return true, true
	}

	tempDir := filepath.Join(filepath.Dir(job.TargetDir), ".mod-downloader-tmp")
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		logging.Warn("hardlink temp dir creation failed", "error", err)
		return false, false
	}
	tempPath := filepath.Join(tempDir, filepath.Base(finalPath))
	if err := os.Remove(tempPath); err != nil && !os.IsNotExist(err) {
		logging.Warn("hardlink temp cleanup failed", "path", tempPath, "error", err)
		return false, false
	}
	if err := minecraft.CreateHardLinkOrCopy(srcPath, tempPath); err != nil {
		logging.Warn("hardlink to temp failed, falling back to download", "src", srcPath, "dst", tempPath, "error", err)
		return false, false
	}
	sha1 := minecraft.FileSHA1(tempPath)
	parsedMods := minecraft.ParseModJarWithSHA1(tempPath, sha1, job.ModLoader)
	if len(parsedMods) == 0 {
		_ = os.Remove(tempPath)
		logging.Warn("hardlink source jar has no parseable metadata, falling back to download", "src", srcPath)
		return false, false
	}
	// Extract primary (strong-reference) mod IDs only. Mirrors downloadModWithLocalParse.
	parsedModIDs := minecraft.PrimaryModIDs(parsedMods)
	if len(parsedModIDs) > 0 && job.Version.ID != "" {
		_ = modbridge.PersistVersionModIDs(job.Version.ID, parsedModIDs)
	}
	parsedExisting := modbridge.LocalModPathsForModIDs(parsedModIDs, job.InstanceID)
	if alreadyInstalled(parsedExisting, sha1) {
		_ = os.Remove(tempPath)
		return true, false
	}
	archiveSupersededModJars(parsedExisting)
	if downloadTargetExists(finalPath) {
		logging.Info("hardlink skipped because target file already exists", "path", finalPath, "versionID", job.Version.ID)
		_ = os.Remove(tempPath)
		return true, false
	}
	if err := os.Rename(tempPath, finalPath); err != nil {
		_ = os.Remove(tempPath)
		return false, false
	}
	logging.Info("mod installed via hardlink (local parse path)", "src", srcPath, "dst", finalPath, "versionID", job.Version.ID)
	go DiscardFetchFromNetwork(job.Version.DownloadURL, job.CurseForgeAPIKey, job.UserAgent)
	upsertDownloadedMod(finalPath, job)
	return true, true
}

// alreadyInstalled checks whether any existing file with the same modId already has the target SHA1.
func alreadyInstalled(existing []global.LocalModFilePath, latestSHA1 string) bool {
	latestSHA1 = strings.ToLower(strings.TrimSpace(latestSHA1))
	if latestSHA1 == "" {
		return false
	}
	for _, p := range existing {
		if strings.ToLower(strings.TrimSpace(p.FileSHA1)) == latestSHA1 {
			return true
		}
	}
	return false
}

func archiveSupersededModJars(paths []global.LocalModFilePath) {
	mcDir := global.GetMinecraftDir()
	for _, p := range paths {
		abs := p.Path
		if mcDir != "" && !filepath.IsAbs(abs) {
			abs = filepath.Join(mcDir, p.Path)
		}
		archivedPath := nextOldJarPath(abs)
		if err := os.Rename(abs, archivedPath); err != nil && !os.IsNotExist(err) {
			logging.Warn("archive superseded mod jar failed", "path", abs, "archivedPath", archivedPath, "error", err)
		} else {
			logging.Info("superseded mod jar archived", "path", abs, "archivedPath", archivedPath)
		}
		global.RemoveLocalModByPath(p.Path)
		global.HardlinkIndexRemove(abs)
	}
}

func archiveIncompatibleModJars(paths []global.LocalModFilePath) {
	if len(paths) == 0 {
		return
	}
	archiveSupersededModJars(paths)
}

func nextOldJarPath(path string) string {
	base := path + ".old"
	if _, err := os.Stat(base); os.IsNotExist(err) {
		return base
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s.old.%d", path, i)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}

func prepareDownloadDependencies(ctx context.Context, job downloadJob, events Events) ([]downloadJob, downloadJob, error) {
	if err := ctxErr(ctx); err != nil {
		return nil, job, err
	}
	if modbridge.SimpleModeEnabled() {
		job.dependenciesDone = true
		return nil, job, nil
	}
	req := downloadRequestFromJob(job)
	version := hydrateDownloadDependencies(req, job.Version)
	if err := ctxErr(ctx); err != nil {
		return nil, job, err
	}

	job.Version = version
	job.IncompatiblePaths = incompatiblePathsFromConflicts(modbridge.IncompatibleConflicts(req), version.SHA1)
	if err := ctxErr(ctx); err != nil {
		return nil, job, err
	}
	appendOptionalDependencyReminder(req, version, events)
	if err := ctxErr(ctx); err != nil {
		return nil, job, err
	}
	job.dependenciesDone = true
	dependencies, err := missingRequiredDependencyJobs(ctx, req, version, job.CurseForgeAPIKey, job.UserAgent, job.visited)
	if err != nil {
		return nil, job, err
	}
	// Required dependencies belong to the same install request, so waiting on
	// the main queue ID waits for the dependency work as well.
	for i := range dependencies {
		dependencies[i].WaitID = job.WaitID
	}
	return dependencies, job, nil
}

func downloadRequestFromJob(job downloadJob) appstructs.ModDownloadRequest {
	return appstructs.ModDownloadRequest{
		ProjectID:        job.ProjectID,
		Result:           job.Result,
		MinecraftVersion: job.MinecraftVersion,
		ModLoader:        job.ModLoader,
		TargetDir:        job.TargetDir,
		InstanceID:       job.InstanceID,
		VersionID:        job.Version.ID,
	}
}

func missingRequiredDependencyJobs(ctx context.Context, req appstructs.ModDownloadRequest, version models.ModVersion, curseForgeAPIKey string, userAgent string, visited map[string]bool) ([]downloadJob, error) {
	if modbridge.SimpleModeEnabled() {
		return nil, nil
	}
	dependencies := make([]downloadJob, 0)
	for _, dep := range version.Dependencies {
		if err := ctxErr(ctx); err != nil {
			return nil, err
		}
		if !isRequiredDependency(dep) {
			continue
		}

		depReq, ok := modbridge.DependencyDownloadRequest(version.Platform, dep, req)
		if !ok {
			logging.Warn("required dependency cannot be queued", "platform", version.Platform, "projectID", dep.DependencyProjectID, "versionID", dep.DependencyVersionID, "type", dep.DependencyType)
			continue
		}

		if installStatusPrecise(depReq) != modbridge.BtnStatusNew {
			if err := ctxErr(ctx); err != nil {
				return nil, err
			}
			continue
		}
		if err := ctxErr(ctx); err != nil {
			return nil, err
		}
		job, result, ok := buildDownloadJob(depReq, curseForgeAPIKey, userAgent, visited)
		if err := ctxErr(ctx); err != nil {
			return nil, err
		}
		if result.Skipped {
			logging.Warn("required dependency queue skipped", "platform", depReq.Result.Platform, "projectID", depReq.ProjectID, "versionID", dep.DependencyVersionID, "reason", result.Reason)
			continue
		}
		if ok {
			dependencies = append(dependencies, job)
		}
	}
	return dependencies, nil
}

func appendOptionalDependencyReminder(parent appstructs.ModDownloadRequest, version models.ModVersion, events Events) {
	if modbridge.SimpleModeEnabled() {
		return
	}
	candidates := make([]appstructs.OptionalDependencyCandidate, 0)
	requests := make([]appstructs.ModDownloadRequest, 0)
	for _, dep := range version.Dependencies {
		if !isOptionalDependency(dep) {
			continue
		}
		depReq, ok := modbridge.DependencyDownloadRequest(version.Platform, dep, parent)
		if !ok {
			logging.Warn("optional dependency cannot be resolved", "platform", version.Platform, "projectID", dep.DependencyProjectID, "versionID", dep.DependencyVersionID, "type", dep.DependencyType)
			continue
		}
		candidate := optionalDependencyCandidate(depReq)
		candidates = append(candidates, candidate)
		if !candidate.Disabled {
			requests = append(requests, depReq)
		}
	}
	if len(candidates) == 0 || len(requests) == 0 {
		return
	}

	mainKey := providers.ProjectReferenceFromSearchResult(parent.Result)
	if mainKey == "" {
		mainKey = strings.TrimSpace(parent.ProjectID)
	}
	snapshot := appstructs.OptionalDependencyReminder{
		MainProjectKey:   mainKey,
		MainTitle:        displayProjectTitle(parent.Result, mainKey),
		MainVersionID:    version.ID,
		MinecraftVersion: parent.MinecraftVersion,
		ModLoader:        parent.ModLoader,
		Dependencies:     candidates,
	}

	downloadQueue.Lock()
	for i, existing := range downloadQueue.optionalReminders {
		if existing.Snapshot.MainProjectKey == snapshot.MainProjectKey && existing.Snapshot.MainVersionID == snapshot.MainVersionID {
			snapshot.ID = existing.Snapshot.ID
			downloadQueue.optionalReminders[i] = optionalDependencyReminderState{Snapshot: snapshot, Requests: requests}
			downloadQueue.Unlock()
			emitDownloadQueueState(events)
			return
		}
	}
	downloadQueue.nextReminderID++
	snapshot.ID = fmt.Sprintf("optional-%d", downloadQueue.nextReminderID)
	downloadQueue.optionalReminders = append(downloadQueue.optionalReminders, optionalDependencyReminderState{Snapshot: snapshot, Requests: requests})
	downloadQueue.Unlock()
	emitDownloadQueueState(events)
}

func optionalDependencyCandidate(req appstructs.ModDownloadRequest) appstructs.OptionalDependencyCandidate {
	status := installStatusPrecise(req)
	disabled := status == modbridge.BtnStatusInstalled
	reason := ""
	if _, ok := modbridge.ResolveVersion(req); !ok {
		disabled = true
		reason = "no matching version"
	}
	return appstructs.OptionalDependencyCandidate{
		ProjectKey: req.ProjectID,
		Platform:   req.Result.Platform,
		ProjectID:  req.Result.ProjectID,
		Title:      displayProjectTitle(req.Result, req.ProjectID),
		VersionID:  req.VersionID,
		Status:     status,
		Disabled:   disabled,
		Reason:     reason,
	}
}

func incompatiblePathsFromConflicts(conflicts []appstructs.IncompatibleConflict, targetSHA1 string) []global.LocalModFilePath {
	if len(conflicts) == 0 {
		return nil
	}
	targetSHA1 = strings.ToLower(strings.TrimSpace(targetSHA1))
	seen := make(map[string]struct{})
	out := make([]global.LocalModFilePath, 0)
	for _, conflict := range conflicts {
		for _, local := range conflict.Paths {
			if local.Path == "" {
				continue
			}
			if targetSHA1 != "" && strings.EqualFold(strings.TrimSpace(local.SHA1), targetSHA1) {
				continue
			}
			if _, dup := seen[local.Path]; dup {
				continue
			}
			seen[local.Path] = struct{}{}
			out = append(out, global.LocalModFilePath{
				ID:       local.Path,
				FileSHA1: local.SHA1,
				Path:     local.Path,
			})
		}
	}
	return out
}

func displayProjectTitle(project models.ModProject, fallback string) string {
	if title := strings.TrimSpace(project.Title); title != "" {
		return title
	}
	if slug := strings.TrimSpace(project.Slug); slug != "" {
		return slug
	}
	if projectID := strings.TrimSpace(project.ProjectID); projectID != "" {
		return projectID
	}
	if fallback = strings.TrimSpace(fallback); fallback != "" {
		return fallback
	}
	return "Mod"
}

func hydrateDownloadDependencies(req appstructs.ModDownloadRequest, version models.ModVersion) models.ModVersion {
	if hasDownloadDependency(version.Dependencies) {
		return version
	}

	refreshed := refreshMatchingProjectVersions(req.Result, req.MinecraftVersion, req.ModLoader)
	if refreshedVersion, found := modbridge.FindVersionByID(refreshed, version.ID); found && hasDownloadDependency(refreshedVersion.Dependencies) {
		return refreshedVersion
	}
	return version
}

func ctxErr(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}

func hasDownloadDependency(deps []models.ModDependency) bool {
	for _, dep := range deps {
		if isRequiredDependency(dep) || isOptionalDependency(dep) || isIncompatibleDependency(dep) {
			return true
		}
	}
	return false
}

func isRequiredDependency(dep models.ModDependency) bool {
	return modbridge.IsDependencyType(dep, "required")
}

func isOptionalDependency(dep models.ModDependency) bool {
	return modbridge.IsDependencyType(dep, "optional")
}

func isIncompatibleDependency(dep models.ModDependency) bool {
	return modbridge.IsDependencyType(dep, "incompatible")
}

func downloadModToTarget(ctx context.Context, job downloadJob, existing []global.LocalModFilePath, events Events) (bool, error) {
	tempPath, err := downloadFileToTemp(ctx, job, events)
	if err != nil {
		return false, err
	}

	finalPath := filepath.Join(job.TargetDir, filepath.Base(tempPath))
	if downloadTargetExists(finalPath) && !pathInLocalModPaths(finalPath, existing) {
		logging.Info("download skipped because target file already exists", "path", finalPath, "versionID", job.Version.ID)
		_ = os.Remove(tempPath)
		return false, nil
	}
	archiveSupersededModJars(existing)
	if downloadTargetExists(finalPath) {
		logging.Info("download skipped because target file already exists after archiving superseded jars", "path", finalPath, "versionID", job.Version.ID)
		_ = os.Remove(tempPath)
		return false, nil
	}
	if err := os.Rename(tempPath, finalPath); err != nil {
		return false, err
	}
	upsertDownloadedMod(finalPath, job)
	return true, nil
}

func downloadModWithLocalParse(ctx context.Context, job downloadJob, events Events) (bool, error) {
	tempPath, err := downloadFileToTemp(ctx, job, events)
	if err != nil {
		return false, err
	}

	sha1 := minecraft.FileSHA1(tempPath)
	mods := minecraft.ParseModJarWithSHA1(tempPath, sha1, job.ModLoader)
	if len(mods) == 0 {
		_ = os.Remove(tempPath)
		return false, fmt.Errorf("downloaded jar has no parseable mod metadata: %s", filepath.Base(tempPath))
	}

	// Extract primary (strong-reference) mod IDs only. JIJ / nested-jar entries
	// are excluded: they must not participate in version persistence, conflict
	// detection, or replacement-archive decisions.
	modIDs := minecraft.PrimaryModIDs(mods)
	if len(modIDs) > 0 && job.Version.ID != "" {
		_ = modbridge.PersistVersionModIDs(job.Version.ID, modIDs)
	}

	existing := modbridge.LocalModPathsForModIDs(modIDs, job.InstanceID)
	if alreadyInstalled(existing, sha1) {
		_ = os.Remove(tempPath)
		return false, nil
	}
	archiveSupersededModJars(existing)

	finalPath := filepath.Join(job.TargetDir, filepath.Base(tempPath))
	if downloadTargetExists(finalPath) {
		logging.Info("download skipped because target file already exists", "path", finalPath, "versionID", job.Version.ID)
		_ = os.Remove(tempPath)
		return false, nil
	}
	if err := os.Rename(tempPath, finalPath); err != nil {
		return false, err
	}
	upsertDownloadedMod(finalPath, job)
	return true, nil
}

func downloadTargetExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	if _, err := os.Stat(path); err == nil {
		return true
	} else if err != nil && !os.IsNotExist(err) {
		logging.Warn("download target stat failed", "path", path, "error", err)
	}
	return false
}

// isCurseForgeCDNURL checks if the URL is a CurseForge CDN download URL.
func isCurseForgeCDNURL(url string) bool {
	return strings.HasPrefix(url, "https://edge.forgecdn.net/files/")
}

func downloadHeaders(job downloadJob) map[string]string {
	headers := map[string]string{"User-Agent": normalizedUserAgent(job.UserAgent)}
	if job.CurseForgeAPIKey != "" && isCurseForgeCDNURL(job.Version.DownloadURL) {
		headers["x-api-key"] = job.CurseForgeAPIKey
	}
	return headers
}

func downloadFileToTemp(ctx context.Context, job downloadJob, events ...Events) (string, error) {
	eventSink := firstEvents(events)
	tempDir := filepath.Join(filepath.Dir(job.TargetDir), ".mod-downloader-tmp")
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		return "", err
	}
	tempPath := filepath.Join(tempDir, downloadFileName(job))
	adaptiveConcurrency, targetRateBytesPerSecond := configuredAdaptiveFileConcurrency()
	request := filetransfer.Request{
		URL:                  job.Version.DownloadURL,
		Destination:          tempPath,
		Headers:              downloadHeaders(job),
		TempDir:              tempDir,
		Concurrency:          configuredFileConcurrency(),
		AdaptiveConcurrency:  adaptiveConcurrency,
		TargetBytesPerSecond: targetRateBytesPerSecond,
		OverwriteExisting:    true,
	}
	verifySHA1 := configuredVerifySHA1() && strings.TrimSpace(job.Version.SHA1) != ""
	attempts := 1
	if verifySHA1 {
		attempts += maxDownloadSHA1Retries
	}
	for attempt := 1; attempt <= attempts; attempt++ {
		if _, err := doDownloadWithStallTimeout(ctx, newFileTransferBackend(), request, func(progress filetransfer.Progress, bytesPerSecond int64) {
			updateDownloadProgress(job.ID, progress, bytesPerSecond, eventSink)
		}); err != nil {
			return "", err
		}
		if !verifySHA1 {
			return tempPath, nil
		}
		expectedSHA1 := strings.ToLower(strings.TrimSpace(job.Version.SHA1))
		actualSHA1 := minecraft.FileSHA1(tempPath)
		if strings.EqualFold(actualSHA1, expectedSHA1) {
			return tempPath, nil
		}
		_ = os.Remove(tempPath)
		if attempt < attempts {
			logging.Warn("download SHA1 mismatch; retrying", "path", tempPath, "expectedSHA1", expectedSHA1, "actualSHA1", actualSHA1, "attempt", attempt, "maxAttempts", attempts)
			continue
		}
		return "", fmt.Errorf("downloaded file SHA1 mismatch: got %s, want %s", actualSHA1, expectedSHA1)
	}
	return "", fmt.Errorf("download SHA1 verification failed")
}

func downloadFileName(job downloadJob) string {
	if name := filepath.Base(strings.TrimSpace(job.Version.FileName)); name != "" && name != "." {
		return name
	}
	parsed, err := url.Parse(strings.TrimSpace(job.Version.DownloadURL))
	if err == nil {
		name := pathpkg.Base(parsed.Path)
		if unescaped, unescapeErr := url.PathUnescape(name); unescapeErr == nil {
			name = unescaped
		}
		name = filepath.Base(strings.TrimSpace(name))
		if name != "" && name != "." && name != string(filepath.Separator) {
			return name
		}
	}
	return "download.jar"
}

func normalizedUserAgent(userAgent string) string {
	userAgent = strings.TrimSpace(userAgent)
	if userAgent == "" {
		return "mod-downloader/dev"
	}
	return userAgent
}

func firstString(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func pathInLocalModPaths(path string, paths []global.LocalModFilePath) bool {
	path = filepath.Clean(path)
	mcDir := global.GetMinecraftDir()
	for _, p := range paths {
		existing := p.Path
		if mcDir != "" && !filepath.IsAbs(existing) {
			existing = filepath.Join(mcDir, existing)
		}
		if filepath.Clean(existing) == path {
			return true
		}
	}
	return false
}

func upsertDownloadedMod(path string, job downloadJob) {
	fileName := minecraft.StripJarSuffix(filepath.Base(path))
	sha1, fingerprint := minecraft.FileHashes(path)
	// Parse the downloaded JAR for mod metadata (pure local, no platform merging)
	r, err := zip.OpenReader(path)
	if err != nil {
		logging.Warn("parse downloaded jar failed", "path", path, "error", err)
		return
	}
	defer r.Close()
	mods := minecraft.ParseModZipReader(&r.Reader, filepath.Base(path), job.ModLoader)
	if len(mods) == 0 {
		logging.Warn("downloaded jar has no parseable mod metadata", "path", path)
		return
	}

	// Store local JAR metadata enriched with install-time platform display fields.
	if strings.TrimSpace(job.Result.ProjectID) != "" && strings.TrimSpace(job.Result.Title) != "" {
		if err := providers.StoreProject(job.Result); err != nil {
			logging.Warn("store downloaded mod project metadata failed", "platform", job.Result.Platform, "projectID", job.Result.ProjectID, "error", err)
		}
	}
	mods = modbridge.ApplyPlatformMetadataToModInfos(mods, job.Result, job.Version)
	global.SetJarMetadata(sha1, "loader:"+strings.ToLower(strings.TrimSpace(job.ModLoader)), mods)

	relPath := path
	if mcDir := global.GetMinecraftDir(); mcDir != "" {
		if rel, err := filepath.Rel(mcDir, path); err == nil {
			relPath = rel
		}
	}
	absPath := path
	if !filepath.IsAbs(absPath) {
		if mcDir := global.GetMinecraftDir(); mcDir != "" {
			absPath = filepath.Join(mcDir, path)
		}
	}
	global.HardlinkIndexAdd(sha1, absPath)
	for i := range mods {
		mods[i].FileName = fileName
		mods[i].Path = relPath
		mods[i].SHA1 = sha1
		mods[i].CfFingerprint = fingerprint
		mods[i].Enabled = true
		global.UpsertLocalMod(mods[i], job.InstanceID, job.MinecraftVersion, job.ModLoader)
	}
}

func currentDownloadQueueState() appstructs.DownloadQueueState {
	downloadQueue.Lock()
	defer downloadQueue.Unlock()
	running := len(downloadQueue.current)
	bytesComplete, totalBytes, bytesPerSecond := aggregateDownloadProgressLocked()
	items := make([]appstructs.DownloadQueueItem, 0, running+len(downloadQueue.pending)+len(downloadQueue.retryable))
	currentIDs := make([]string, 0, len(downloadQueue.current))
	for id := range downloadQueue.current {
		currentIDs = append(currentIDs, id)
	}
	sort.Strings(currentIDs)
	for _, id := range currentIDs {
		item := downloadQueueItemFromJob(downloadQueue.current[id], "running", true)
		item.Retryable = true
		applyDownloadProgressLocked(&item, id)
		items = append(items, item)
	}
	for _, job := range downloadQueue.pending {
		item := downloadQueueItemFromJob(job, "pending", true)
		applyDownloadProgressLocked(&item, job.ID)
		items = append(items, item)
	}
	for _, item := range downloadQueue.retryable {
		queueItem := downloadQueueItemFromJob(item.Job, item.Status, false)
		queueItem.Retryable = true
		queueItem.Reason = item.Reason
		items = append(items, queueItem)
	}
	reminders := make([]appstructs.OptionalDependencyReminder, 0, len(downloadQueue.optionalReminders))
	for _, reminder := range downloadQueue.optionalReminders {
		reminders = append(reminders, reminder.Snapshot)
	}
	return appstructs.DownloadQueueState{
		Active:            running > 0 || len(downloadQueue.pending) > 0 || len(downloadQueue.retryable) > 0,
		Pending:           len(downloadQueue.pending),
		Running:           running,
		MessageCount:      len(reminders),
		BytesComplete:     bytesComplete,
		TotalBytes:        totalBytes,
		BytesPerSecond:    bytesPerSecond,
		Items:             items,
		OptionalReminders: reminders,
	}
}

func applyDownloadProgressLocked(item *appstructs.DownloadQueueItem, id string) {
	progress, ok := downloadQueue.progress[id]
	if !ok {
		return
	}
	item.BytesComplete = nonNegativeDownloadBytes(progress.BytesComplete)
	item.TotalBytes = nonNegativeDownloadBytes(progress.TotalBytes)
	if item.TotalBytes > 0 && item.BytesComplete > item.TotalBytes {
		item.BytesComplete = item.TotalBytes
	}
}

func aggregateDownloadProgressLocked() (int64, int64, int64) {
	var complete, total, speed int64
	for _, progress := range downloadQueue.progress {
		if progress.TotalBytes > 0 {
			complete = addDownloadBytes(complete, minDownloadBytes(progress.BytesComplete, progress.TotalBytes))
			total = addDownloadBytes(total, progress.TotalBytes)
		}
	}
	for id := range downloadQueue.current {
		if progress, ok := downloadQueue.progress[id]; ok && progress.BytesPerSecond > 0 {
			speed = addDownloadBytes(speed, progress.BytesPerSecond)
		}
	}
	return complete, total, speed
}

func nonNegativeDownloadBytes(value int64) int64 {
	if value < 0 {
		return 0
	}
	return value
}

func minDownloadBytes(left, right int64) int64 {
	left = nonNegativeDownloadBytes(left)
	right = nonNegativeDownloadBytes(right)
	if left < right {
		return left
	}
	return right
}

func addDownloadBytes(left, right int64) int64 {
	left = nonNegativeDownloadBytes(left)
	right = nonNegativeDownloadBytes(right)
	const maxInt64 = int64(^uint64(0) >> 1)
	if left > maxInt64-right {
		return maxInt64
	}
	return left + right
}

func downloadQueueItemFromJob(job downloadJob, status string, cancelable bool) appstructs.DownloadQueueItem {
	title := strings.TrimSpace(job.Result.Title)
	if title == "" {
		title = firstNonEmpty(job.Version.Name, job.Version.FileName, job.Version.Version, job.Version.ID)
	}
	return appstructs.DownloadQueueItem{
		ID:               job.ID,
		ProjectID:        job.ProjectID,
		Status:           status,
		Title:            title,
		FileName:         job.Version.FileName,
		VersionID:        job.Version.ID,
		Platform:         job.Version.Platform,
		MinecraftVersion: job.MinecraftVersion,
		ModLoader:        job.ModLoader,
		Cancelable:       cancelable,
	}
}

func appendRetryableDownload(job downloadJob, status string, reason string) {
	downloadQueue.Lock()
	defer downloadQueue.Unlock()
	appendRetryableDownloadLocked(job, status, reason)
}

func appendRetryableDownloadLocked(job downloadJob, status string, reason string) {
	job.cancel = nil
	downloadQueue.retryable = append(downloadQueue.retryable, retryableDownloadJob{
		Job:    job,
		Status: strings.TrimSpace(status),
		Reason: strings.TrimSpace(reason),
	})
	if overflow := len(downloadQueue.retryable) - maxRetryableDownloadHistory; overflow > 0 {
		downloadQueue.retryable = append([]retryableDownloadJob(nil), downloadQueue.retryable[overflow:]...)
	}
}

func consumeDownloadCancelRequest(id string) bool {
	downloadQueue.Lock()
	defer downloadQueue.Unlock()
	if downloadQueue.cancelRequested == nil {
		return false
	}
	if _, ok := downloadQueue.cancelRequested[id]; !ok {
		return false
	}
	delete(downloadQueue.cancelRequested, id)
	return true
}

func consumeFinalizedDownloadCancel(id string) bool {
	downloadQueue.Lock()
	defer downloadQueue.Unlock()
	if downloadQueue.finalizedCancel == nil {
		return false
	}
	if _, ok := downloadQueue.finalizedCancel[id]; !ok {
		return false
	}
	delete(downloadQueue.finalizedCancel, id)
	return true
}

func consumeDownloadRestartRequest(id string) bool {
	downloadQueue.Lock()
	defer downloadQueue.Unlock()
	if downloadQueue.restartRequested == nil {
		return false
	}
	if _, ok := downloadQueue.restartRequested[id]; !ok {
		return false
	}
	delete(downloadQueue.restartRequested, id)
	return true
}

func retryStalledDownload(ctx context.Context, job downloadJob, events Events) bool {
	if job.AutoAttempt >= maxDownloadStallRetries {
		return false
	}
	removeDownloadProgress(job.ID)
	job.ID = ""
	job.cancel = nil
	job.AutoAttempt++
	_, queued := enqueueDownload(ctx, job, events)
	return queued
}

type transferOutcome struct {
	result filetransfer.Result
	err    error
}

func doDownloadWithStallTimeout(ctx context.Context, backend filetransfer.Backend, req filetransfer.Request, onProgress ...func(filetransfer.Progress, int64)) (filetransfer.Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	attemptCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	tracker := filetransfer.NewProgressTracker(nil)
	done := make(chan transferOutcome, 1)
	var progressCallback func(filetransfer.Progress, int64)
	if len(onProgress) > 0 {
		progressCallback = onProgress[0]
	}
	lastSampleBytes := tracker.Snapshot().BytesComplete
	lastSampleAt := time.Now()
	notifyProgress := func() {
		if progressCallback == nil {
			return
		}
		progress := tracker.Snapshot()
		elapsed := time.Since(lastSampleAt).Seconds()
		bytesPerSecond := int64(0)
		if elapsed > 0 && progress.BytesComplete > lastSampleBytes {
			bytesPerSecond = int64(float64(progress.BytesComplete-lastSampleBytes) / elapsed)
		}
		progressCallback(progress, bytesPerSecond)
		lastSampleBytes = progress.BytesComplete
		lastSampleAt = time.Now()
	}
	go func() {
		result, err := backend.Download(attemptCtx, req, tracker)
		done <- transferOutcome{result: result, err: err}
	}()
	if downloadStallTimeout <= 0 {
		outcome := <-done
		notifyProgress()
		return outcome.result, outcome.err
	}
	pollInterval := downloadProgressPollInterval
	if pollInterval <= 0 {
		pollInterval = time.Second
	}
	lastBytes := tracker.Snapshot().BytesComplete
	lastProgress := time.Now()
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		select {
		case outcome := <-done:
			notifyProgress()
			return outcome.result, outcome.err
		case <-ctx.Done():
			cancel()
			<-done
			notifyProgress()
			return filetransfer.Result{}, ctx.Err()
		case <-ticker.C:
			notifyProgress()
			currentBytes := tracker.Snapshot().BytesComplete
			if currentBytes > lastBytes {
				lastBytes = currentBytes
				lastProgress = time.Now()
				continue
			}
			if time.Since(lastProgress) >= downloadStallTimeout {
				select {
				case outcome := <-done:
					return outcome.result, outcome.err
				default:
				}
				cancel()
				<-done
				notifyProgress()
				return filetransfer.Result{}, errDownloadStalled
			}
		}
	}
}

func firstEvents(events []Events) Events {
	if len(events) == 0 {
		return Events{}
	}
	return events[0]
}

func emitDownloadQueueState(events Events) {
	if events.OnQueueState == nil {
		return
	}
	events.OnQueueState(currentDownloadQueueState())
}

func emitDownloadFailed(events Events, job downloadJob, err error) {
	if events.OnDownloadFailed == nil || err == nil {
		return
	}
	events.OnDownloadFailed(appstructs.DownloadFailedEvent{
		FileName:  job.Version.FileName,
		VersionID: job.Version.ID,
		Reason:    err.Error(),
	})
}
