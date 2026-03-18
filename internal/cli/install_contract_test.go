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
	// Bootstrap must run in post_install (outside Homebrew's network sandbox)
	// rather than in the install block which blocks network access on macOS.
	if !strings.Contains(text, "def post_install") || !strings.Contains(text, `system "#{bin}/anchorbrowser", "proxy", "install"`) {
		t.Fatalf("expected goreleaser brew post_install to bootstrap proxy runtime via 'proxy install'")
	}
	if strings.Contains(text, `system "#{bin}/anchorbrowser", "proxy", "--help"`) {
		t.Fatalf("goreleaser brew install stanza must not run proxy bootstrap (network blocked in Homebrew sandbox)")
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
