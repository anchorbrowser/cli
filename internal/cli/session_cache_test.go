package cli

import (
	"path/filepath"
	"strings"
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

func TestResolveSessionIDUsesFlagThenCache(t *testing.T) {
	app := newSessionCacheTestApp(t)
	if err := app.cacheSessionID("cached-1"); err != nil {
		t.Fatalf("cacheSessionID: %v", err)
	}

	cmd := newSessionIDTestCommand()
	if err := cmd.Flags().Set("session-id", "flag-1"); err != nil {
		t.Fatalf("set flag: %v", err)
	}
	got, err := app.resolveSessionID(cmd)
	if err != nil || got != "flag-1" {
		t.Fatalf("flag session id: got=%q err=%v", got, err)
	}

	cmd = newSessionIDTestCommand()
	got, err = app.resolveSessionID(cmd)
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
	if _, err := app.resolveSessionID(cmd); err == nil {
		t.Fatalf("expected error when --no-cache is set without explicit session id")
	}
}

func TestExtractSessionPrimaryPageIDFromPagesResponse(t *testing.T) {
	resp := map[string]any{
		"data": map[string]any{
			"items": []any{
				map[string]any{"id": "  "},
				map[string]any{"id": "PAGE-123"},
			},
		},
	}
	if got := extractSessionPrimaryPageIDFromPagesResponse(resp); got != "PAGE-123" {
		t.Fatalf("expected PAGE-123, got %q", got)
	}
}

func TestBuildSessionCDPURLFromPage(t *testing.T) {
	got := buildSessionCDPURLFromPage("sess-1", "PAGE-123")
	wantPrefix := "wss://connect.anchorbrowser.io/devtools/page/PAGE-123?sessionId="
	if !strings.HasPrefix(got, wantPrefix) {
		t.Fatalf("expected prefix %q, got %q", wantPrefix, got)
	}
	if !strings.Contains(got, "sessionId=sess-1") {
		t.Fatalf("expected escaped session id in URL, got %q", got)
	}
	if gotEmpty := buildSessionCDPURLFromPage("", "PAGE-123"); gotEmpty != "" {
		t.Fatalf("expected empty URL for missing session id, got %q", gotEmpty)
	}
}
