package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/mod/semver"

	"github.com/spf13/cobra"
)

const anchorIcon = `
           ███
          █████
         ███████
           ███
    ████████████████
      ████████████
        ████████
          ████

_______             ______                   ________
___    |_______________  /_______________    ___  __ )_______________      _____________________
__  /| |_  __ \  ___/_  __ \  __ \_  ___/    __  __  |_  ___/  __ \_ | /| / /_  ___/  _ \_  ___/
_  ___ |  / / / /__ _  / / / /_/ /  /        _  /_/ /_  /   / /_/ /_ |/ |/ /_(__  )/  __/  /
/_/  |_/_/ /_/\___/ /_/ /_/\____//_/         /_____/ /_/    \____/____/|__/ /____/ \___//_/
`

type latestReleaseResponse struct {
	TagName string `json:"tag_name"`
}

func newVersionCommand(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return app.printVersionInfo(cmd.Context())
		},
	}
}

func (a *App) printVersionInfo(ctx context.Context) error {
	displayVersion := strings.TrimSpace(a.Version)
	if displayVersion == "" {
		displayVersion = "dev"
	}
	currentSemver := normalizeSemver(a.Version)

	if _, err := fmt.Fprint(a.Stdout, anchorIcon); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(a.Stdout, "ANCHORBROWSER VERSION %s\n", displayVersion); err != nil {
		return err
	}

	latest, err := fetchLatestReleaseTag(ctx)
	if err != nil {
		_, _ = fmt.Fprintln(a.Stdout, "Update check: unavailable")
		return nil
	}
	if semver.Compare(normalizeSemver(latest), currentSemver) > 0 {
		_, err = fmt.Fprintf(a.Stdout, "Update available: %s (current %s)\n", latest, displayVersion)
		return err
	}
	_, err = fmt.Fprintf(a.Stdout, "You are up to date (%s)\n", displayVersion)
	return err
}

func fetchLatestReleaseTag(ctx context.Context) (string, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, "https://api.github.com/repos/anchorbrowser/cli/releases/latest", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "anchorbrowser-cli")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("release lookup failed: status %d", resp.StatusCode)
	}

	var parsed latestReleaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", err
	}
	if strings.TrimSpace(parsed.TagName) == "" {
		return "", fmt.Errorf("empty latest release tag")
	}
	return strings.TrimSpace(parsed.TagName), nil
}

func normalizeSemver(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "v0.0.0"
	}
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	if !semver.IsValid(v) {
		return "v0.0.0"
	}
	return v
}
