package cli

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
)

type updateStrategy struct {
	Name    string
	Steps   [][]string
	Display string
}

func newUpdateCommand(app *App) *cobra.Command {
	var checkOnly bool

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update the AnchorBrowser CLI",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			current := strings.TrimSpace(app.Version)
			if current == "" {
				current = "dev"
			}
			latest, err := fetchLatestReleaseTagFn(cmd.Context())
			if err != nil {
				return fmt.Errorf("could not check latest release: %w", err)
			}

			if semver.Compare(normalizeSemver(latest), normalizeSemver(current)) <= 0 {
				_, _ = fmt.Fprintf(app.Stdout, "You are already up to date (%s).\n", current)
				return nil
			}

			_, _ = fmt.Fprintf(app.Stdout, "Update available: %s (current %s)\n", latest, current)
			strategy, err := detectUpdateStrategy(cmd.Context())
			if err != nil {
				return err
			}

			if checkOnly {
				_, _ = fmt.Fprintf(app.Stdout, "Run this command to update: %s\n", strategy.Display)
				return nil
			}

			_, _ = fmt.Fprintf(app.Stdout, "Updating via %s...\n", strategy.Name)
			if err := runUpdateStrategy(cmd.Context(), app, strategy); err != nil {
				return err
			}

			_, _ = fmt.Fprintln(app.Stdout, "Update complete. Run `anchorbrowser --version` to verify.")
			return nil
		},
	}
	cmd.Flags().BoolVar(&checkOnly, "check", false, "Only check whether an update is available and print the command")
	return cmd
}

func detectUpdateStrategy(ctx context.Context) (*updateStrategy, error) {
	return detectUpdateStrategyWith(ctx, exec.LookPath, runCommandCapture)
}

func detectUpdateStrategyWith(
	ctx context.Context,
	lookPath func(file string) (string, error),
	runCapture func(ctx context.Context, name string, args ...string) (string, error),
) (*updateStrategy, error) {
	if _, err := lookPath("brew"); err == nil {
		if out, listErr := runCapture(ctx, "brew", "list", "--versions", "anchorbrowser/tap/anchorbrowser"); listErr == nil && strings.TrimSpace(out) != "" {
			return &updateStrategy{
				Name:    "Homebrew",
				Steps:   [][]string{{"brew", "update"}, {"brew", "upgrade", "anchorbrowser/tap/anchorbrowser"}},
				Display: "brew update && brew upgrade anchorbrowser/tap/anchorbrowser",
			}, nil
		}
		if out, listErr := runCapture(ctx, "brew", "list", "--versions", "anchorbrowser"); listErr == nil && strings.TrimSpace(out) != "" {
			return &updateStrategy{
				Name:    "Homebrew",
				Steps:   [][]string{{"brew", "update"}, {"brew", "upgrade", "anchorbrowser/tap/anchorbrowser"}},
				Display: "brew update && brew upgrade anchorbrowser/tap/anchorbrowser",
			}, nil
		}
	}

	if _, err := lookPath("npm"); err == nil {
		if _, npmErr := runCapture(ctx, "npm", "list", "-g", "--depth=0", "@anchor-browser/cli"); npmErr == nil {
			return &updateStrategy{
				Name:    "npm",
				Steps:   [][]string{{"npm", "i", "-g", "@anchor-browser/cli@latest"}},
				Display: "npm i -g @anchor-browser/cli@latest",
			}, nil
		}
	}

	return nil, fmt.Errorf("could not detect how this CLI was installed. update manually with Homebrew (`brew upgrade anchorbrowser/tap/anchorbrowser`) or npm (`npm i -g @anchor-browser/cli@latest`)")
}

func runUpdateStrategy(ctx context.Context, app *App, strategy *updateStrategy) error {
	for _, step := range strategy.Steps {
		if len(step) == 0 {
			continue
		}
		_, _ = fmt.Fprintf(app.Stdout, "Running: %s\n", strings.Join(step, " "))

		cmd := exec.CommandContext(ctx, step[0], step[1:]...)
		cmd.Stdin = app.Stdin
		cmd.Stdout = app.Stdout
		cmd.Stderr = app.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("update step failed (%s): %w", strings.Join(step, " "), err)
		}
	}
	return nil
}

func runCommandCapture(ctx context.Context, name string, args ...string) (string, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, name, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return strings.TrimSpace(out.String()), err
	}
	return strings.TrimSpace(out.String()), nil
}
