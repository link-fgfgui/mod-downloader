package minecraft

import (
	"path/filepath"
	"testing"
)

func TestExpandPathWithEnvSupportsWindowsPercentSyntax(t *testing.T) {
	root := t.TempDir()
	t.Setenv("MOD_DOWNLOADER_TEST_APPDATA", root)

	got := ExpandPathWithEnv(`%MOD_DOWNLOADER_TEST_APPDATA%\.minecraft`)
	want := filepath.Join(root, ".minecraft")
	if got != want {
		t.Fatalf("ExpandPathWithEnv() = %q, want %q", got, want)
	}
}

func TestExpandPathWithEnvSupportsDollarSyntaxWithWindowsSeparators(t *testing.T) {
	root := t.TempDir()
	t.Setenv("MOD_DOWNLOADER_TEST_APPDATA", root)

	got := ExpandPathWithEnv(`$MOD_DOWNLOADER_TEST_APPDATA\.minecraft`)
	want := filepath.Join(root, ".minecraft")
	if got != want {
		t.Fatalf("ExpandPathWithEnv() = %q, want %q", got, want)
	}
}

func TestExpandPathWithEnvSupportsBracedDollarSyntax(t *testing.T) {
	root := t.TempDir()
	t.Setenv("MOD_DOWNLOADER_TEST_APPDATA", root)

	got := ExpandPathWithEnv(`${MOD_DOWNLOADER_TEST_APPDATA}/.minecraft`)
	want := filepath.Join(root, ".minecraft")
	if got != want {
		t.Fatalf("ExpandPathWithEnv() = %q, want %q", got, want)
	}
}

func TestSimplifyPathWithEnvExpandsBeforeSimplifying(t *testing.T) {
	root := t.TempDir()
	t.Setenv("MOD_DOWNLOADER_TEST_APPDATA", root)

	got := SimplifyPathWithEnv(`%MOD_DOWNLOADER_TEST_APPDATA%\.minecraft`)
	want := filepath.Join("$MOD_DOWNLOADER_TEST_APPDATA", ".minecraft")
	if got != want {
		t.Fatalf("SimplifyPathWithEnv() = %q, want %q", got, want)
	}
}

func TestExpandPathWithEnvPreservesUnknownVariables(t *testing.T) {
	tests := []string{
		`$MOD_DOWNLOADER_UNKNOWN\.minecraft`,
		`${MOD_DOWNLOADER_UNKNOWN}\.minecraft`,
		`%MOD_DOWNLOADER_UNKNOWN%\.minecraft`,
	}
	for _, test := range tests {
		got := ExpandPathWithEnv(test)
		if got == ".minecraft" || got == filepath.Clean(".minecraft") {
			t.Fatalf("ExpandPathWithEnv(%q) dropped the unknown variable: %q", test, got)
		}
	}
}
