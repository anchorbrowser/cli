package cli

import (
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"

	"github.com/anchorbrowser/cli/internal/config"
)

func newSessionCacheTestApp(t *testing.T) *App {
	t.Helper()
	return &App{
		Global: &GlobalOptions{},
		Config: config.NewManagerWithPath(filepath.Join(t.TempDir(), "config.yaml")),
	}
}

func newSessionIDTestCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("session-id", "", "")
	cmd.Flags().Bool("no-cache", false, "")
	return cmd
}

func TestResolveSessionIDUsesArgThenFlagThenCache(t *testing.T) {
	app := newSessionCacheTestApp(t)
	if err := app.cacheSessionID("cached-1"); err != nil {
		t.Fatalf("cacheSessionID: %v", err)
	}

	cmd := newSessionIDTestCommand()
	got, err := app.resolveSessionID(cmd, []string{"arg-1"})
	if err != nil || got != "arg-1" {
		t.Fatalf("arg session id: got=%q err=%v", got, err)
	}

	cmd = newSessionIDTestCommand()
	if err := cmd.Flags().Set("session-id", "flag-1"); err != nil {
		t.Fatalf("set flag: %v", err)
	}
	got, err = app.resolveSessionID(cmd, nil)
	if err != nil || got != "flag-1" {
		t.Fatalf("flag session id: got=%q err=%v", got, err)
	}

	cmd = newSessionIDTestCommand()
	got, err = app.resolveSessionID(cmd, nil)
	if err != nil || got != "cached-1" {
		t.Fatalf("cached session id: got=%q err=%v", got, err)
	}
}

func TestResolveSessionIDNoCacheRequiresExplicitID(t *testing.T) {
	app := newSessionCacheTestApp(t)
	if err := app.cacheSessionID("cached-1"); err != nil {
		t.Fatalf("cacheSessionID: %v", err)
	}

	cmd := newSessionIDTestCommand()
	if err := cmd.Flags().Set("no-cache", "true"); err != nil {
		t.Fatalf("set no-cache: %v", err)
	}
	if _, err := app.resolveSessionID(cmd, nil); err == nil {
		t.Fatalf("expected error when --no-cache is set without explicit session id")
	}
}
