package appcore

import (
	"os/exec"
	"strings"
	"testing"
)

func TestCoreAndCLIDoNotDependOnWailsRuntime(t *testing.T) {
	packages := []string{
		"mod-downloader/appcore",
		"mod-downloader/cliapp",
		"mod-downloader/cmd/mod-downloader-cli",
	}
	args := append([]string{"list", "-deps"}, packages...)
	out, err := exec.Command("go", args...).Output()
	if err != nil {
		t.Fatalf("go list deps failed: %v", err)
	}
	for _, pkg := range strings.Fields(string(out)) {
		if pkg == "github.com/wailsapp/wails/v2/pkg/runtime" {
			t.Fatalf("core/CLI dependency tree includes Wails runtime")
		}
	}
}
