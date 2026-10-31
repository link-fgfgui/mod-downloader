package minecraft

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/link-fgfgui/mod-downloader-core/logging"
	"github.com/link-fgfgui/mod-downloader-core/networking"
)

type httpRangeReaderAt struct {
	client        *http.Client
	url           string
	size          int64
	deadline      time.Time
	rangeRequests atomic.Int64
	rangeBytes    atomic.Int64
}

func NewHTTPRangeReaderAt(url string) (*httpRangeReaderAt, error) {
	return newHTTPRangeReaderAt(url, 0)
}

func NewHTTPRangeReaderAtWithTimeout(url string, timeout time.Duration) (*httpRangeReaderAt, error) {
	return newHTTPRangeReaderAt(url, timeout)
}

func newHTTPRangeReaderAt(url string, timeout time.Duration) (*httpRangeReaderAt, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return nil, fmt.Errorf("empty url")
	}

	clientTimeout := 20 * time.Second
	var deadline time.Time
	if timeout > 0 {
		deadline = time.Now().Add(timeout)
		if timeout < clientTimeout {
			clientTimeout = timeout
		}
	}

	client := networking.NewClientWithUserAgent(clientTimeout, "mod-downloader/dev")
	resolvedURL, err := resolveHTTPDownloadURL(client, url, deadline)
	if err != nil {
		resolvedURL = url
	}

	size, err := probeHTTPRangeSize(client, resolvedURL, deadline)
	if err != nil && resolvedURL != url {
		size, err = probeHTTPRangeSize(client, url, deadline)
		resolvedURL = url
	}
	if err != nil {
		return nil, err
	}

	return &httpRangeReaderAt{
		client:   client,
		url:      resolvedURL,
		size:     size,
		deadline: deadline,
	}, nil
}

func resolveHTTPDownloadURL(client *http.Client, url string, deadline time.Time) (string, error) {
	started := time.Now()
	req, cancel, err := newDeadlineRequest(http.MethodHead, url, deadline)
	if err != nil {
		return "", err
	}
	defer cancel()

	resp, err := client.Do(req)
	if err != nil {
		logging.Debug("remote jar URL resolve failed", "url", url, "error", err, "elapsedMs", time.Since(started).Milliseconds())
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		logging.Debug("remote jar URL resolve returned non-success", "url", url, "status", resp.StatusCode, "elapsedMs", time.Since(started).Milliseconds())
		return "", fmt.Errorf("resolve download url failed: status %d", resp.StatusCode)
	}
	logging.Debug("remote jar URL resolved", "url", url, "resolvedURL", resp.Request.URL.String(), "status", resp.StatusCode, "contentLength", resp.ContentLength, "elapsedMs", time.Since(started).Milliseconds())
	return resp.Request.URL.String(), nil
}

func probeHTTPRangeSize(client *http.Client, url string, deadline time.Time) (int64, error) {
	started := time.Now()
	req, cancel, err := newDeadlineRequest(http.MethodGet, url, deadline)
	if err != nil {
		return 0, err
	}
	defer cancel()
	req.Header.Set("Range", "bytes=0-0")

	resp, err := client.Do(req)
	if err != nil {
		logging.Debug("remote jar range probe failed", "url", url, "error", err, "elapsedMs", time.Since(started).Milliseconds())
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent {
		logging.Debug("remote jar range probe unsupported", "url", url, "status", resp.StatusCode, "contentLength", resp.ContentLength, "elapsedMs", time.Since(started).Milliseconds())
		return 0, fmt.Errorf("range request unsupported: status %d", resp.StatusCode)
	}

	size, err := parseContentRangeSize(resp.Header.Get("Content-Range"))
	if err != nil {
		return 0, err
	}
	if size <= 0 {
		return 0, fmt.Errorf("invalid content size: %d", size)
	}

	_, _ = io.Copy(io.Discard, resp.Body)
	logging.Debug("remote jar range probe completed", "url", url, "status", resp.StatusCode, "contentLength", resp.ContentLength, "contentRange", resp.Header.Get("Content-Range"), "size", size, "elapsedMs", time.Since(started).Milliseconds())
	return size, nil
}

func parseContentRangeSize(contentRange string) (int64, error) {
	_, sizeText, ok := strings.Cut(strings.TrimSpace(contentRange), "/")
	if !ok {
		return 0, fmt.Errorf("invalid content-range: %s", contentRange)
	}
	sizeText = strings.TrimSpace(sizeText)
	if sizeText == "" || sizeText == "*" {
		return 0, fmt.Errorf("unknown content size")
	}
	return strconv.ParseInt(sizeText, 10, 64)
}

func newDeadlineRequest(method string, url string, deadline time.Time) (*http.Request, context.CancelFunc, error) {
	ctx := context.Background()
	cancel := func() {}
	if !deadline.IsZero() {
		if time.Now().After(deadline) {
			return nil, cancel, context.DeadlineExceeded
		}
		ctx, cancel = context.WithDeadline(ctx, deadline)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		cancel()
		return nil, func() {}, err
	}
	return req, cancel, nil
}

func (r *httpRangeReaderAt) Size() int64 {
	return r.size
}

// RangeStats reports the number of range reads and payload bytes fetched.
func (r *httpRangeReaderAt) RangeStats() (int64, int64) {
	return r.rangeRequests.Load(), r.rangeBytes.Load()
}

func (r *httpRangeReaderAt) ReadAt(p []byte, off int64) (int, error) {
	if !r.deadline.IsZero() && time.Now().After(r.deadline) {
		return 0, context.DeadlineExceeded
	}
	if off < 0 {
		return 0, fmt.Errorf("negative offset")
	}
	if len(p) == 0 {
		return 0, nil
	}
	if off >= r.size {
		return 0, io.EOF
	}

	want := len(p)
	end := off + int64(want) - 1
	if end >= r.size {
		end = r.size - 1
		want = int(end-off) + 1
	}

	req, cancel, err := newDeadlineRequest(http.MethodGet, r.url, r.deadline)
	if err != nil {
		return 0, err
	}
	defer cancel()
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", off, end))

	started := time.Now()
	resp, err := r.client.Do(req)
	if err != nil {
		logging.Debug("remote jar range read failed", "url", r.url, "host", remoteHost(r.url), "offset", off, "end", end, "error", err, "elapsedMs", time.Since(started).Milliseconds())
		return 0, err
	}
	defer resp.Body.Close()
	r.rangeRequests.Add(1)

	if resp.StatusCode == http.StatusOK {
		logging.Warn("range read fell back to full download (server returned 200 instead of 206)",
			"url", r.url, "status", resp.StatusCode, "offset", off, "end", end, "size", r.size)
		if off > 0 {
			if _, err := io.CopyN(io.Discard, resp.Body, off); err != nil {
				return 0, err
			}
		}
		n, readErr := readHTTPRangeBody(resp.Body, p, want)
		r.rangeBytes.Add(int64(n))
		logging.Debug("remote jar range read completed", "url", r.url, "host", remoteHost(r.url), "offset", off, "end", end, "status", resp.StatusCode, "contentLength", resp.ContentLength, "contentRange", resp.Header.Get("Content-Range"), "bytes", n, "elapsedMs", time.Since(started).Milliseconds(), "fallbackFullBody", true)
		return n, readErr
	}
	if resp.StatusCode != http.StatusPartialContent {
		logging.Warn("range read failed with unexpected status",
			"url", r.url, "status", resp.StatusCode, "offset", off, "end", end, "size", r.size)
		return 0, fmt.Errorf("range read failed: status %d", resp.StatusCode)
	}

	n, readErr := readHTTPRangeBody(resp.Body, p, want)
	r.rangeBytes.Add(int64(n))
	logging.Debug("remote jar range read completed", "url", r.url, "host", remoteHost(r.url), "offset", off, "end", end, "status", resp.StatusCode, "contentLength", resp.ContentLength, "contentRange", resp.Header.Get("Content-Range"), "bytes", n, "elapsedMs", time.Since(started).Milliseconds(), "fallbackFullBody", false)
	return n, readErr
}

func remoteHost(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return parsed.Host
}

func readHTTPRangeBody(body io.Reader, p []byte, want int) (int, error) {
	n, err := io.ReadFull(body, p[:want])
	if err == io.ErrUnexpectedEOF {
		err = io.EOF
	}
	if n < len(p) && err == nil {
		err = io.EOF
	}
	return n, err
}
