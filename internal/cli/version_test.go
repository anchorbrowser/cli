package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestNormalizeSemver(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "0.1.3", want: "v0.1.3"},
		{in: "v0.1.3", want: "v0.1.3"},
		{in: "dev", want: "v0.0.0"},
		{in: "", want: "v0.0.0"},
	}
	for _, tt := range tests {
		got := normalizeSemver(tt.in)
		if got != tt.want {
			t.Fatalf("normalizeSemver(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestPrintVersionInfoShowsUpdateCommandHint(t *testing.T) {
	old := fetchLatestReleaseTagFn
	fetchLatestReleaseTagFn = func(context.Context) (string, error) {
		return "v9.9.9", nil
	}
	t.Cleanup(func() {
		fetchLatestReleaseTagFn = old
	})

	out := &bytes.Buffer{}
	app := &App{
		Version: "0.1.20",
		Stdout:  out,
	}

	if err := app.printVersionInfo(context.Background()); err != nil {
		t.Fatalf("printVersionInfo: %v", err)
	}
	text := out.String()
	if !strings.Contains(text, "Run `anchorbrowser update` to upgrade.") {
		t.Fatalf("expected update command hint in version output, got:\n%s", text)
	}
}
