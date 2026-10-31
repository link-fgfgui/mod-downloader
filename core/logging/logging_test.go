package logging

import (
	"bytes"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigureDisabledSuppressesAllOutputs(t *testing.T) {
	path := filepath.Join(t.TempDir(), DefaultFileName)
	var stderr bytes.Buffer
	if err := configure(Options{Disabled: true, ForceFile: true, FilePath: path}, &stderr, true); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(Close)
	Info("hidden message")
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("disabled log file stat error = %v, want not exist", err)
	}
}

func TestConfigureForceFileWritesStderrAndFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), DefaultFileName)
	var stderr bytes.Buffer
	if err := configure(Options{ForceFile: true, FilePath: path}, &stderr, true); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(Close)
	Info("forced message", "count", 2)
	if !strings.Contains(stderr.String(), "forced message") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "forced message") || !strings.Contains(string(data), "count=2") {
		t.Fatalf("log file = %q", data)
	}
}

func TestConfigureFallsBackToFileWhenStderrUnavailable(t *testing.T) {
	path := filepath.Join(t.TempDir(), DefaultFileName)
	var stderr bytes.Buffer
	if err := configure(Options{FilePath: path}, &stderr, false); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(Close)
	Warn("fallback message")
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "fallback message") {
		t.Fatalf("log file = %q", data)
	}
}

func TestConfigureUsesOnlyStderrByDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), DefaultFileName)
	var stderr bytes.Buffer
	if err := configure(Options{FilePath: path}, &stderr, true); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(Close)
	Info("stderr message")
	if !strings.Contains(stderr.String(), "stderr message") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("default log file stat error = %v, want not exist", err)
	}
}

func TestStderrAvailableRejectsClosedFile(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "stderr")
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if stderrAvailable(file) {
		t.Fatal("closed stderr reported available")
	}
}

func TestDebugSuppressedByDefault(t *testing.T) {
	var stderr bytes.Buffer
	if err := configure(Options{}, &stderr, true); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(Close)
	Debug("hidden debug")
	if strings.Contains(stderr.String(), "hidden debug") {
		t.Fatalf("debug leaked at default level: %q", stderr.String())
	}
	Info("visible info")
	if !strings.Contains(stderr.String(), "visible info") {
		t.Fatalf("info missing: %q", stderr.String())
	}
}

func TestConfigureDebugLevel(t *testing.T) {
	var stderr bytes.Buffer
	if err := configure(Options{Level: slog.LevelDebug}, &stderr, true); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(Close)
	Debug("visible debug")
	if !strings.Contains(stderr.String(), "visible debug") {
		t.Fatalf("debug missing: %q", stderr.String())
	}
}

func TestConfigureWarnLevelSuppressesInfo(t *testing.T) {
	var stderr bytes.Buffer
	if err := configure(Options{Level: slog.LevelWarn}, &stderr, true); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(Close)
	Info("hidden info")
	Warn("visible warn")
	out := stderr.String()
	if strings.Contains(out, "hidden info") {
		t.Fatalf("info leaked at warn level: %q", out)
	}
	if !strings.Contains(out, "visible warn") {
		t.Fatalf("warn missing: %q", out)
	}
}

func TestNoStderrWritesFileOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), DefaultFileName)
	var stderr bytes.Buffer
	if err := configure(Options{NoStderr: true, FilePath: path}, &stderr, true); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(Close)
	Info("file only message")
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "file only message") {
		t.Fatalf("log file = %q", data)
	}
}

func TestDefaultPathForCacheDir(t *testing.T) {
	base := filepath.Join(t.TempDir(), "cache")
	want := filepath.Join(base, "mod-downloader", "logs", DefaultFileName)
	if got := defaultPathForCacheDir(base); got != want {
		t.Fatalf("defaultPathForCacheDir() = %q, want %q", got, want)
	}
}

func TestRotatingFileKeepsConfiguredBackups(t *testing.T) {
	path := filepath.Join(t.TempDir(), "logs", DefaultFileName)
	writer, err := newRotatingFile(path, 100, 2)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		payload := bytes.Repeat([]byte{byte('a' + i)}, 80)
		if _, err := writer.Write(payload); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	current, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(current) != 80 || current[0] != 'e' {
		t.Fatalf("current log = %d bytes starting with %q, want latest write", len(current), current[0])
	}
	first, err := os.ReadFile(backupPath(path, 1))
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 80 || first[0] != 'd' {
		t.Fatalf("first backup starts with %q, want previous write", first[0])
	}
	second, err := os.ReadFile(backupPath(path, 2))
	if err != nil {
		t.Fatal(err)
	}
	if len(second) != 80 || second[0] != 'c' {
		t.Fatalf("second backup starts with %q, want older write", second[0])
	}
	if _, err := os.Stat(backupPath(path, 3)); !os.IsNotExist(err) {
		t.Fatalf("backup 3 stat error = %v, want not exist", err)
	}
}

func TestRotatingFileRejectsWriteAfterClose(t *testing.T) {
	path := filepath.Join(t.TempDir(), DefaultFileName)
	writer, err := newRotatingFile(path, 1024, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte("late")); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("Write after Close error = %v, want os.ErrClosed", err)
	}
}

func TestConfigureRotatesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), DefaultFileName)
	var stderr bytes.Buffer
	if err := configure(Options{
		NoStderr:     true,
		FilePath:     path,
		MaxSizeBytes: 512,
		MaxBackups:   1,
	}, &stderr, true); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(Close)
	for i := 0; i < 50; i++ {
		Info("rotation probe", "index", i)
	}
	Close()
	if _, err := os.Stat(backupPath(path, 1)); err != nil {
		t.Fatalf("first backup: %v", err)
	}
	if _, err := os.Stat(backupPath(path, 2)); !os.IsNotExist(err) {
		t.Fatalf("backup 2 stat error = %v, want not exist", err)
	}
}
