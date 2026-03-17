package backend

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManagerStatusAndPath(t *testing.T) {
	dir := t.TempDir()
	manager := NewManagerWithBaseDir(dir)

	path, err := manager.BinaryPath()
	if err != nil {
		t.Fatalf("BinaryPath: %v", err)
	}
	if filepath.Dir(path) == "" {
		t.Fatalf("expected non-empty binary path")
	}

	status, err := manager.Status()
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status.Installed {
		t.Fatalf("expected backend to be reported as not installed")
	}
}

func TestManagerUninstall(t *testing.T) {
	dir := t.TempDir()
	manager := NewManagerWithBaseDir(dir)

	dummyPath := filepath.Join(dir, "agent-browser", PinnedVersion, BinaryFileName())
	if err := os.MkdirAll(filepath.Dir(dummyPath), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(dummyPath, []byte("x"), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := manager.Uninstall(); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "agent-browser")); !os.IsNotExist(err) {
		t.Fatalf("expected backend root to be removed")
	}
}
