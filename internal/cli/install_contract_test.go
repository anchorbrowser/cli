package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNpmPostinstallBootstrapsProxy(t *testing.T) {
	root := repoRootFromTestFile(t)
	data, err := os.ReadFile(filepath.Join(root, "npm", "lib", "install.js"))
	if err != nil {
		t.Fatalf("read npm install script: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "proxy', '--help'") && !strings.Contains(text, "\"proxy\", \"--help\"") {
		t.Fatalf("expected npm postinstall script to invoke proxy help bootstrap")
	}
}

func TestGoreleaserBrewInstallBootstrapsProxy(t *testing.T) {
	root := repoRootFromTestFile(t)
	data, err := os.ReadFile(filepath.Join(root, ".goreleaser.yaml"))
	if err != nil {
		t.Fatalf("read goreleaser config: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, `system "#{bin}/anchorbrowser", "proxy", "--help"`) {
		t.Fatalf("expected goreleaser brew install stanza to run proxy help bootstrap")
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
