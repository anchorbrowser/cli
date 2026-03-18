package cli

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
)

type fakeHelpBackendManager struct {
	path string
	err  error
}

func (f fakeHelpBackendManager) EnsureInstalled(_ context.Context) (string, error) {
	return f.path, f.err
}

func TestRenderRootHelpIncludesProxyAndAnchorCommands(t *testing.T) {
	app, err := NewApp("test")
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	var out bytes.Buffer
	app.Stdout = &out
	app.Stderr = &bytes.Buffer{}
	app.Stdin = strings.NewReader("")

	if err := renderRootHelp(app); err != nil {
		t.Fatalf("renderRootHelp: %v", err)
	}
	text := out.String()
	for _, mustContain := range []string{
		"Available Commands:",
		"anchor",
		"auth",
		"proxy",
		"update",
		"version",
	} {
		if !strings.Contains(text, mustContain) {
			t.Fatalf("expected root help to contain %q\n%s", mustContain, text)
		}
	}
	for _, mustNotContain := range []string{"backend"} {
		if strings.Contains(text, mustNotContain) {
			t.Fatalf("expected root help to hide %q\n%s", mustNotContain, text)
		}
	}
}

func TestHelpCommandRejectsDirectParityTopic(t *testing.T) {
	var out, errOut bytes.Buffer
	err := executeWithIO("test", []string{"help", "click"}, strings.NewReader(""), &out, &errOut)
	if err == nil {
		t.Fatalf("expected help click to be rejected")
	}
	if !strings.Contains(err.Error(), "proxy") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHelpCommandRoutesReservedToCobra(t *testing.T) {
	var out bytes.Buffer
	if err := executeWithIO("test", []string{"help", "anchor"}, strings.NewReader(""), &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executeWithIO(help anchor): %v", err)
	}
	if !strings.Contains(out.String(), "Anchor API commands") {
		t.Fatalf("expected reserved help to route to cobra, got: %s", out.String())
	}
}

func TestNoArgsUseCustomRootHelp(t *testing.T) {
	origRender := renderRootHelpFn
	origNewApp := newAppFn
	defer func() {
		renderRootHelpFn = origRender
		newAppFn = origNewApp
	}()

	called := false
	renderRootHelpFn = func(_ *App) error {
		called = true
		return nil
	}
	newAppFn = func(version string) (*App, error) {
		return &App{
			Version: version,
			Global:  &GlobalOptions{BaseURL: "https://api.anchorbrowser.io", Output: "json"},
			Stdin:   strings.NewReader(""),
			Stdout:  &bytes.Buffer{},
			Stderr:  &bytes.Buffer{},
		}, nil
	}

	if err := executeWithIO("test", []string{}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("executeWithIO(no args): %v", err)
	}
	if !called {
		t.Fatalf("expected custom root help renderer to be called")
	}
}

func TestParityHelpRewritesBrandingAndSkipsCDP(t *testing.T) {
	origMgr := newParityBackendManagerFn
	origCombined := runParityCombinedCommandFn
	origStreaming := runParityStreamingCommandFn
	defer func() {
		newParityBackendManagerFn = origMgr
		runParityCombinedCommandFn = origCombined
		runParityStreamingCommandFn = origStreaming
	}()

	newParityBackendManagerFn = func() (parityBackendInstaller, error) {
		return fakeHelpBackendManager{path: "/tmp/anchorbrowser-backend"}, nil
	}
	runParityStreamingCommandFn = func(_ context.Context, _ string, _ io.Reader, _ io.Writer, _ io.Writer, _ ...string) error {
		t.Fatalf("streaming path should not be called for --help")
		return nil
	}
	capturedArgs := []string{}
	runParityCombinedCommandFn = func(_ context.Context, _ string, _ io.Reader, args ...string) ([]byte, error) {
		capturedArgs = append([]string(nil), args...)
		return []byte("agent-browser click help\nAGENT_BROWSER_SESSION=default\n~/.agent-browser"), nil
	}

	app, err := NewApp("test")
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	var out bytes.Buffer
	app.Stdout = &out
	app.Stderr = &bytes.Buffer{}
	app.Stdin = strings.NewReader("")

	parsed, err := parseParityArgs([]string{"click", "--help"})
	if err != nil {
		t.Fatalf("parseParityArgs: %v", err)
	}
	if err := runParityCommand(app, parsed); err != nil {
		t.Fatalf("runParityCommand: %v", err)
	}
	for _, arg := range capturedArgs {
		if arg == "--cdp" || strings.HasPrefix(arg, "--cdp=") {
			t.Fatalf("did not expect cdp injection for help command: %v", capturedArgs)
		}
	}
	rendered := out.String()
	if strings.Contains(rendered, "agent-browser") || strings.Contains(rendered, "AGENT_BROWSER_") || strings.Contains(rendered, "~/.agent-browser") {
		t.Fatalf("expected rewritten output, got: %s", rendered)
	}
	if !strings.Contains(rendered, "anchorbrowser") || !strings.Contains(rendered, "ANCHORBROWSER_") || !strings.Contains(rendered, "~/.anchorbrowser") {
		t.Fatalf("expected white-label replacements, got: %s", rendered)
	}
}
