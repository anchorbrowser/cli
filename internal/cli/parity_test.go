package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestParseParityArgs(t *testing.T) {
	parsed, err := parseParityArgs([]string{
		"--api-key", "sk-1",
		"--base-url", "https://example.com",
		"--timeout=45s",
		"--session-id", "sess-1",
		"--new-session",
		"open", "https://anchorbrowser.io", "--headed",
	})
	if err != nil {
		t.Fatalf("parseParityArgs: %v", err)
	}
	if parsed.Global.APIKey != "sk-1" {
		t.Fatalf("expected API key override, got %q", parsed.Global.APIKey)
	}
	if parsed.Global.BaseURL != "https://example.com" {
		t.Fatalf("expected base-url override, got %q", parsed.Global.BaseURL)
	}
	if !parsed.Global.TimeoutSet || parsed.Global.Timeout != 45*time.Second {
		t.Fatalf("expected timeout override, got set=%t timeout=%s", parsed.Global.TimeoutSet, parsed.Global.Timeout)
	}
	if parsed.Session.SessionID != "sess-1" {
		t.Fatalf("expected session id, got %q", parsed.Session.SessionID)
	}
	if !parsed.Session.NewSession {
		t.Fatalf("expected --new-session to be true")
	}
	if len(parsed.BackendArgs) != 3 {
		t.Fatalf("expected 3 backend args, got %d (%v)", len(parsed.BackendArgs), parsed.BackendArgs)
	}
}

func TestLegacyTopLevelCommandReturnsMigrationError(t *testing.T) {
	var out, errOut bytes.Buffer
	err := executeWithIO("test", []string{"session", "list"}, strings.NewReader(""), &out, &errOut)
	if err == nil {
		t.Fatalf("expected migration error")
	}
	if !strings.Contains(err.Error(), "moved under the anchor namespace") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestShouldInjectCDP(t *testing.T) {
	if shouldInjectCDP("install") {
		t.Fatalf("install should not require cdp injection")
	}
	if shouldInjectCDP("help") {
		t.Fatalf("help should not require cdp injection")
	}
	if !shouldInjectCDP("open") {
		t.Fatalf("open should require cdp injection")
	}
}

func TestHasHelpOrVersionFlag(t *testing.T) {
	if !hasHelpOrVersionFlag([]string{"snapshot", "--help"}) {
		t.Fatalf("expected --help to disable cdp injection")
	}
	if !hasHelpOrVersionFlag([]string{"open", "-V"}) {
		t.Fatalf("expected -V to disable cdp injection")
	}
	if hasHelpOrVersionFlag([]string{"open", "https://example.com"}) {
		t.Fatalf("unexpected help/version flag detection")
	}
}

func TestReservedCommandDispatchesToCobra(t *testing.T) {
	var out, errOut bytes.Buffer
	err := executeWithIO("test", []string{"anchor", "--help"}, strings.NewReader(""), &out, &errOut)
	if err != nil {
		t.Fatalf("expected reserved command to execute through cobra: %v", err)
	}
	if !strings.Contains(out.String(), "Anchor API commands") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}
