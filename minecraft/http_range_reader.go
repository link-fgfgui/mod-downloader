package minecraft

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type httpRangeReaderAt struct {
	client *http.Client
	url    string
	size   int64
}

func NewHTTPRangeReaderAt(url string) (*httpRangeReaderAt, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return nil, fmt.Errorf("empty url")
	}

	client := &http.Client{Timeout: 20 * time.Second}
	resolvedURL, err := resolveHTTPDownloadURL(client, url)
	if err != nil {
		resolvedURL = url
	}

	size, err := probeHTTPRangeSize(client, resolvedURL)
	if err != nil && resolvedURL != url {
		size, err = probeHTTPRangeSize(client, url)
		resolvedURL = url
	}
	if err != nil {
		return nil, err
	}

	return &httpRangeReaderAt{
		client: client,
		url:    resolvedURL,
		size:   size,
	}, nil
}

func resolveHTTPDownloadURL(client *http.Client, url string) (string, error) {
	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("resolve download url failed: status %d", resp.StatusCode)
	}
	return resp.Request.URL.String(), nil
}

func probeHTTPRangeSize(client *http.Client, url string) (int64, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Range", "bytes=0-0")

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent {
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

func (r *httpRangeReaderAt) Size() int64 {
	return r.size
}

func (r *httpRangeReaderAt) ReadAt(p []byte, off int64) (int, error) {
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

	req, err := http.NewRequest(http.MethodGet, r.url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", off, end))

	resp, err := r.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		if off > 0 {
			if _, err := io.CopyN(io.Discard, resp.Body, off); err != nil {
				return 0, err
			}
		}
		return readHTTPRangeBody(resp.Body, p, want)
	}
	if resp.StatusCode != http.StatusPartialContent {
		return 0, fmt.Errorf("range read failed: status %d", resp.StatusCode)
	}

	return readHTTPRangeBody(resp.Body, p, want)
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
