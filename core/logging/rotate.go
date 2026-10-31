package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const (
	// DefaultMaxSizeBytes is the rotation threshold used when a caller does not
	// specify one: 6 MiB.
	DefaultMaxSizeBytes int64 = 6 * 1024 * 1024
	// DefaultMaxBackups keeps two historical files next to the active log. With
	// DefaultMaxSizeBytes the default on-disk ceiling is about 18 MiB.
	DefaultMaxBackups = 2
)

// rotatingFile is an append-only log file that renames itself when the next
// write would exceed maxSize. It owns the file handle, so callers must Close it.
type rotatingFile struct {
	mu         sync.Mutex
	path       string
	file       *os.File
	size       int64
	maxSize    int64
	maxBackups int
	closed     bool
}

func newRotatingFile(path string, maxSize int64, maxBackups int) (*rotatingFile, error) {
	if maxSize <= 0 {
		maxSize = DefaultMaxSizeBytes
	}
	if maxBackups < 0 {
		maxBackups = 0
	}
	writer := &rotatingFile{
		path:       path,
		maxSize:    maxSize,
		maxBackups: maxBackups,
	}
	if err := writer.open(); err != nil {
		return nil, err
	}
	return writer, nil
}

func (w *rotatingFile) open() error {
	if w.closed {
		return os.ErrClosed
	}
	if dir := filepath.Dir(w.path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create log directory: %w", err)
		}
	}
	file, err := os.OpenFile(w.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return err
	}
	w.file = file
	w.size = info.Size()
	return nil
}

func (w *rotatingFile) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return 0, os.ErrClosed
	}
	if w.file == nil {
		if err := w.open(); err != nil {
			return 0, err
		}
	}
	// Rotate before writing. Do not rotate an empty file when a single record is
	// larger than maxSize: it still has to be written somewhere.
	if w.size > 0 && w.size+int64(len(p)) > w.maxSize {
		if err := w.rotate(); err != nil {
			// Rotation housekeeping failed. If the file was reopened, still write
			// the record and report the rotation error alongside it.
			if w.file == nil {
				return 0, err
			}
			n, writeErr := w.file.Write(p)
			w.size += int64(n)
			if writeErr != nil {
				return n, writeErr
			}
			return n, err
		}
	}
	n, err := w.file.Write(p)
	w.size += int64(n)
	return n, err
}

func (w *rotatingFile) rotate() error {
	if w.file != nil {
		_ = w.file.Close()
		w.file = nil
	}
	if err := w.rotateFiles(); err != nil {
		// Keep logging to the original path if housekeeping failed. A failure to
		// rename an old backup must not silently drop new records.
		if openErr := w.open(); openErr != nil {
			return err
		}
		return err
	}
	w.size = 0
	return w.open()
}

func (w *rotatingFile) rotateFiles() error {
	if w.maxBackups == 0 {
		if err := os.Remove(w.path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	oldest := backupPath(w.path, w.maxBackups)
	if err := os.Remove(oldest); err != nil && !os.IsNotExist(err) {
		return err
	}
	for i := w.maxBackups - 1; i >= 1; i-- {
		from := backupPath(w.path, i)
		to := backupPath(w.path, i+1)
		if err := os.Rename(from, to); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	if err := os.Rename(w.path, backupPath(w.path, 1)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (w *rotatingFile) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return nil
	}
	w.closed = true
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}

func backupPath(path string, index int) string {
	return fmt.Sprintf("%s.%d", path, index)
}
