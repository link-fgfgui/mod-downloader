package filetransfer

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/link-fgfgui/mod-downloader-core/networking"
)

type StdlibBackend struct {
	client *http.Client
}

func NewStdlibBackend(client *http.Client) *StdlibBackend {
	if client == nil {
		client = networking.NewClientWithUserAgent(0, "mod-downloader/dev")
	}
	return &StdlibBackend{client: client}
}

func (*StdlibBackend) Kind() BackendKind { return BackendStdlib }

func (b *StdlibBackend) Download(ctx context.Context, request Request, tracker *ProgressTracker) (Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := normalizeRequest(request)
	if err != nil {
		return Result{}, err
	}
	if err := prepareDestination(req); err != nil {
		return Result{}, err
	}
	info, err := probeRemote(ctx, b.client, req)
	if err != nil {
		info = remoteInfo{size: -1}
	}
	tracker.Reset(info.size)

	if info.rangeSupported && info.size > 0 {
		if info.size <= req.MemoryLimit {
			err = b.downloadRangesToMemory(ctx, req, info.size, tracker)
		} else {
			err = b.downloadRangesToTemp(ctx, req, info.size, tracker)
		}
	} else {
		err = b.downloadDirect(ctx, req, info.size, tracker)
	}
	if err != nil {
		return Result{}, err
	}
	progress := tracker.Snapshot()
	return Result{
		Path:           req.Destination,
		BytesComplete:  progress.BytesComplete,
		TotalBytes:     progress.TotalBytes,
		RangeSupported: info.rangeSupported,
	}, nil
}

type byteRange struct {
	index int
	start int64
	end   int64
}

const adaptiveSegmentMultiplier int64 = 4

func rangesForSegments(size, segmentCount int64) []byteRange {
	if size <= 0 || segmentCount <= 0 {
		return nil
	}
	if segmentCount > size {
		segmentCount = size
	}
	chunkSize := size / segmentCount
	remainder := size % segmentCount
	count := int(segmentCount)
	ranges := make([]byteRange, 0, count)
	start := int64(0)
	for index := 0; index < count; index++ {
		partSize := chunkSize
		if int64(index) < remainder {
			partSize++
		}
		end := start + partSize - 1
		ranges = append(ranges, byteRange{index: index, start: start, end: end})
		start = end + 1
	}
	return ranges
}

func segmentCountForRequest(req Request) int64 {
	count := int64(req.Concurrency)
	if count <= 0 {
		count = DefaultConcurrency
	}
	if req.AdaptiveConcurrency && count <= (int64(^uint64(0)>>1))/adaptiveSegmentMultiplier {
		count *= adaptiveSegmentMultiplier
	}
	return count
}

func (b *StdlibBackend) downloadRangesToMemory(ctx context.Context, req Request, size int64, tracker *ProgressTracker) error {
	ranges := rangesForSegments(size, segmentCountForRequest(req))
	chunks := make([][]byte, len(ranges))
	err := runRequestRanges(ctx, ranges, req, tracker, func(ctx context.Context, part byteRange) error {
		var buffer bytes.Buffer
		if err := b.fetchRange(ctx, req, part.start, part.end, &trackingWriter{writer: &buffer, tracker: tracker}); err != nil {
			return err
		}
		chunks[part.index] = buffer.Bytes()
		return nil
	})
	if err != nil {
		return err
	}
	return writeAtomic(req.Destination, req.OverwriteExisting, func(file *os.File) error {
		for _, chunk := range chunks {
			if _, err := file.Write(chunk); err != nil {
				return err
			}
		}
		return nil
	})
}

func (b *StdlibBackend) downloadRangesToTemp(ctx context.Context, req Request, size int64, tracker *ProgressTracker) error {
	segmentCount := segmentCountForRequest(req)
	ranges := rangesForSegments(size, segmentCount)
	tempDir := rangedTempDir(req, size, segmentCount)
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		return err
	}
	err := runRequestRanges(ctx, ranges, req, tracker, func(ctx context.Context, part byteRange) error {
		path := filepath.Join(tempDir, fmt.Sprintf("%06d.part", part.index))
		expected := part.end - part.start + 1
		existing := existingPartSize(path, expected)
		if existing == expected {
			tracker.AddComplete(existing)
			return nil
		}
		file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, defaultFileMode)
		if err != nil {
			return err
		}
		if err := file.Truncate(existing); err != nil {
			_ = file.Close()
			return err
		}
		if _, err := file.Seek(existing, io.SeekStart); err != nil {
			_ = file.Close()
			return err
		}
		tracker.AddComplete(existing)
		err = b.fetchRange(ctx, req, part.start+existing, part.end, &trackingWriter{writer: file, tracker: tracker})
		closeErr := file.Close()
		if err != nil {
			return err
		}
		return closeErr
	})
	if err != nil {
		return err
	}
	if err := writeAtomic(req.Destination, req.OverwriteExisting, func(file *os.File) error {
		for _, part := range ranges {
			path := filepath.Join(tempDir, fmt.Sprintf("%06d.part", part.index))
			chunk, err := os.Open(path)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(file, chunk)
			closeErr := chunk.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
		}
		return nil
	}); err != nil {
		return err
	}
	return os.RemoveAll(tempDir)
}

func runRequestRanges(ctx context.Context, ranges []byteRange, req Request, tracker *ProgressTracker, download func(context.Context, byteRange) error) error {
	if !req.AdaptiveConcurrency || req.TargetBytesPerSecond <= 0 {
		return runRanges(ctx, ranges, req.Concurrency, download)
	}
	return runAdaptiveRanges(ctx, ranges, req.Concurrency, req.TargetBytesPerSecond, tracker, download)
}

var adaptiveRampInterval = time.Second

func runAdaptiveRanges(parent context.Context, ranges []byteRange, initialConcurrency int, targetBytesPerSecond int64, tracker *ProgressTracker, download func(context.Context, byteRange) error) error {
	if len(ranges) == 0 {
		return nil
	}
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	if initialConcurrency < 1 {
		initialConcurrency = 1
	}
	if initialConcurrency > len(ranges) {
		initialConcurrency = len(ranges)
	}

	jobs := make(chan byteRange)
	errCh := make(chan error, 1)
	done := make(chan struct{})
	var completed atomic.Int64
	var doneOnce sync.Once
	var workers sync.WaitGroup
	startWorker := func() {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for part := range jobs {
				if err := download(ctx, part); err != nil {
					select {
					case errCh <- err:
					default:
					}
					cancel()
					return
				}
				if completed.Add(1) == int64(len(ranges)) {
					doneOnce.Do(func() { close(done) })
				}
			}
		}()
	}
	for range initialConcurrency {
		startWorker()
	}
	go func() {
		defer close(jobs)
		for _, part := range ranges {
			select {
			case jobs <- part:
			case <-ctx.Done():
				return
			}
		}
	}()

	workerCount := initialConcurrency
	ticker := time.NewTicker(adaptiveRampInterval)
	defer ticker.Stop()
	lastProgress := tracker.Snapshot().BytesComplete
	lastSample := time.Now()
	for {
		select {
		case err := <-errCh:
			workers.Wait()
			return err
		case <-parent.Done():
			workers.Wait()
			return parent.Err()
		case <-done:
			workers.Wait()
			return nil
		case now := <-ticker.C:
			progress := tracker.Snapshot().BytesComplete
			elapsed := now.Sub(lastSample).Seconds()
			if elapsed > 0 && float64(progress-lastProgress)/elapsed < float64(targetBytesPerSecond) && workerCount < len(ranges) {
				startWorker()
				workerCount++
			}
			lastProgress = progress
			lastSample = now
		}
	}
}

func (b *StdlibBackend) downloadDirect(ctx context.Context, req Request, size int64, tracker *ProgressTracker) error {
	return writeAtomic(req.Destination, req.OverwriteExisting, func(file *os.File) error {
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, req.URL, nil)
		if err != nil {
			return err
		}
		applyHeaders(httpReq, req.Headers)
		resp, err := b.client.Do(httpReq)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			return fmt.Errorf("download failed: %s", resp.Status)
		}
		if size <= 0 && resp.ContentLength >= 0 {
			tracker.Reset(resp.ContentLength)
		}
		_, err = io.Copy(file, &trackingReader{reader: resp.Body, tracker: tracker})
		return err
	})
}

func (b *StdlibBackend) fetchRange(ctx context.Context, req Request, start, end int64, writer io.Writer) error {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, req.URL, nil)
	if err != nil {
		return err
	}
	applyHeaders(httpReq, req.Headers)
	httpReq.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))
	resp, err := b.client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("range download failed: %s", resp.Status)
	}
	written, err := io.Copy(writer, io.LimitReader(resp.Body, end-start+2))
	if err != nil {
		return err
	}
	if written != end-start+1 {
		return fmt.Errorf("range download size mismatch: got %d, want %d", written, end-start+1)
	}
	return nil
}

func runRanges(parent context.Context, ranges []byteRange, concurrency int, download func(context.Context, byteRange) error) error {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	jobs := make(chan byteRange)
	errCh := make(chan error, 1)
	var wg sync.WaitGroup
	if concurrency > len(ranges) {
		concurrency = len(ranges)
	}
	for range concurrency {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for part := range jobs {
				if err := download(ctx, part); err != nil {
					select {
					case errCh <- err:
					default:
					}
					cancel()
					return
				}
			}
		}()
	}
	go func() {
		defer close(jobs)
		for _, part := range ranges {
			select {
			case jobs <- part:
			case <-ctx.Done():
				return
			}
		}
	}()
	wg.Wait()
	select {
	case err := <-errCh:
		return err
	default:
		return parent.Err()
	}
}

func prepareDestination(req Request) error {
	if !req.OverwriteExisting {
		if _, err := os.Stat(req.Destination); err == nil {
			return fmt.Errorf("download destination already exists: %s", req.Destination)
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	return os.MkdirAll(filepath.Dir(req.Destination), 0o755)
}

func writeAtomic(destination string, overwrite bool, write func(*os.File) error) error {
	partPath := destination + ".part"
	file, err := os.OpenFile(partPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, defaultFileMode)
	if err != nil {
		return err
	}
	if err := write(file); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if overwrite {
		_ = os.Remove(destination)
	}
	return os.Rename(partPath, destination)
}

func rangedTempDir(req Request, size, segmentCount int64) string {
	root := req.TempDir
	if root == "" {
		root = os.TempDir()
	}
	layoutIdentity := fmt.Sprintf("v2|%s|%s|%d|%d", req.URL, req.Destination, size, segmentCount)
	sum := md5.Sum([]byte(layoutIdentity))
	return filepath.Join(root, "mod-downloader", hex.EncodeToString(sum[:]))
}

func existingPartSize(path string, expected int64) int64 {
	info, err := os.Stat(path)
	if err != nil || info.Size() < 0 || info.Size() > expected {
		return 0
	}
	return info.Size()
}

type trackingWriter struct {
	writer  io.Writer
	tracker *ProgressTracker
}

func (w *trackingWriter) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)
	w.tracker.AddComplete(int64(n))
	return n, err
}

type trackingReader struct {
	reader  io.Reader
	tracker *ProgressTracker
}

func (r *trackingReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.tracker.AddComplete(int64(n))
	return n, err
}
