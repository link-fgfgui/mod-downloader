package logging

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const DefaultFileName = "mod-downloader.log"

type Options struct {
	Disabled bool
	// ForceFile mirrors the config's force_file setting: write to the file and
	// also mirror to stderr when it is usable.
	ForceFile bool
	FilePath  string
	// Level is the minimum level to emit. The zero value is slog.LevelInfo.
	Level slog.Level
	// NoStderr treats stderr as unavailable even when it is a terminal. The TUI
	// uses this so log lines never corrupt the full-screen program; output goes
	// to FilePath only.
	NoStderr bool
	// MaxSizeBytes is the active file size that triggers rotation. Zero uses
	// DefaultMaxSizeBytes.
	MaxSizeBytes int64
	// MaxBackups is the number of historical files to keep. Zero uses
	// DefaultMaxBackups; a negative value disables backups.
	MaxBackups int
}

var (
	loggerMu sync.RWMutex
	logger   = newLogger(os.Stderr, slog.LevelInfo)
	logFile  io.Closer
)

// DefaultPath returns the per-user log file used when no explicit path is
// configured. If the platform cache directory is unavailable it falls back to
// the system temp directory, never the working directory. Logs are
// intentionally not written to the working directory or a mods directory
// unless the user configured [logging].file.
func DefaultPath() string {
	dir, err := os.UserCacheDir()
	if err != nil || strings.TrimSpace(dir) == "" {
		dir = os.TempDir()
	}
	return defaultPathForCacheDir(dir)
}

func defaultPathForCacheDir(cacheDir string) string {
	return filepath.Join(cacheDir, "mod-downloader", "logs", DefaultFileName)
}

func init() {
	slog.SetDefault(logger)
}

func Configure(options Options) error {
	usable := !options.NoStderr && stderrAvailable(os.Stderr)
	return configure(options, os.Stderr, usable)
}

func configure(options Options, stderr io.Writer, stderrUsable bool) error {
	var output io.Writer = io.Discard
	var nextFile io.Closer
	configuredPath := ""
	if !options.Disabled {
		fileOnly := options.NoStderr
		if options.ForceFile || fileOnly || !stderrUsable {
			path := strings.TrimSpace(options.FilePath)
			if path == "" {
				path = DefaultPath()
			}
			file, err := newRotatingFile(path, effectiveMaxSize(options.MaxSizeBytes), effectiveMaxBackups(options.MaxBackups))
			if err != nil {
				return err
			}
			nextFile = file
			output = file
			configuredPath = path
			if !fileOnly && stderrUsable && stderr != nil {
				output = io.MultiWriter(stderr, file)
			}
		} else if stderr != nil {
			output = stderr
		}
	}

	nextLogger := newLogger(output, options.Level)
	loggerMu.Lock()
	previousFile := logFile
	logger = nextLogger
	logFile = nextFile
	slog.SetDefault(nextLogger)
	if previousFile != nil {
		_ = previousFile.Close()
	}
	// Emit exactly one record describing the final sink. It goes through the
	// same level filter as every other record, so a quiet CLI stays quiet.
	// Keep the lock until the record is written so a concurrent Close cannot
	// close the new sink before this line lands.
	nextLogger.Info("logging configured",
		"path", configuredPath,
		"level", options.Level.String(),
		"maxSizeBytes", effectiveMaxSize(options.MaxSizeBytes),
		"maxBackups", effectiveMaxBackups(options.MaxBackups),
	)
	loggerMu.Unlock()
	return nil
}

func effectiveMaxSize(size int64) int64 {
	if size <= 0 {
		return DefaultMaxSizeBytes
	}
	return size
}

func effectiveMaxBackups(backups int) int {
	if backups == 0 {
		return DefaultMaxBackups
	}
	if backups < 0 {
		return 0
	}
	return backups
}

func Close() {
	loggerMu.Lock()
	previousFile := logFile
	logger = newLogger(io.Discard, slog.LevelInfo)
	logFile = nil
	slog.SetDefault(logger)
	if previousFile != nil {
		_ = previousFile.Close()
	}
	loggerMu.Unlock()
}

func newLogger(output io.Writer, level slog.Level) *slog.Logger {
	return slog.New(slog.NewTextHandler(output, &slog.HandlerOptions{Level: level}))
}

func stderrAvailable(stderr *os.File) bool {
	if stderr == nil {
		return false
	}
	_, err := stderr.Stat()
	return err == nil
}

func Debug(msg string, args ...any) {
	loggerMu.RLock()
	defer loggerMu.RUnlock()
	logger.Debug(msg, args...)
}

func Info(msg string, args ...any) {
	loggerMu.RLock()
	defer loggerMu.RUnlock()
	logger.Info(msg, args...)
}

func Warn(msg string, args ...any) {
	loggerMu.RLock()
	defer loggerMu.RUnlock()
	logger.Warn(msg, args...)
}

func Error(msg string, args ...any) {
	loggerMu.RLock()
	defer loggerMu.RUnlock()
	logger.Error(msg, args...)
}
