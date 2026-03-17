package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNpmPostinstallRunsBackendInstall(t *testing.T) {
	root := repoRootFromTestFile(t)
	data, err := os.ReadFile(filepath.Join(root, "npm", "lib", "install.js"))
	if err != nil {
		t.Fatalf("read npm install script: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "backend', 'install'") && !strings.Contains(text, "\"backend\", \"install\"") {
		t.Fatalf("expected npm postinstall script to invoke backend install")
	}
}

func TestGoreleaserBrewInstallRunsBackendInstall(t *testing.T) {
	root := repoRootFromTestFile(t)
	data, err := os.ReadFile(filepath.Join(root, ".goreleaser.yaml"))
	if err != nil {
		t.Fatalf("read goreleaser config: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, `system "#{bin}/anchorbrowser", "backend", "install"`) {
		t.Fatalf("expected goreleaser brew install stanza to run backend install")
	}
}

func repoRootFromTestFile(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("resolve test file path")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	return root
}
