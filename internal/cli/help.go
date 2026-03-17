package cli

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"sort"
	"strings"

	"github.com/anchorbrowser/cli/internal/backend"
	"github.com/anchorbrowser/cli/internal/config"
)

type backendInstaller interface {
	EnsureInstalled(ctx context.Context) (string, error)
}

var (
	newHelpBackendManagerFn = func() (backendInstaller, error) {
		return backend.NewManager(config.DefaultAppName)
	}
	runHelpBackendCommandFn = runHelpBackendCommand
)

type helpCommand struct {
	Name        string
	Description string
}

func renderRootHelp(app *App) error {
	raw, err := fetchParityRootHelp(app)
	if err != nil {
		return err
	}
	rewritten := rewriteParityHelpText(raw)
	commands := collectRootCommands(rewritten)

	anchorCommands := []helpCommand{
		{Name: "anchor", Description: "Anchor API commands"},
		{Name: "auth", Description: "Manage API key authentication"},
		{Name: "backend", Description: "Manage embedded parity backend"},
		{Name: "update", Description: "Update the AnchorBrowser CLI"},
		{Name: "version", Description: "Print version information"},
	}
	for _, c := range anchorCommands {
		commands[c.Name] = c.Description
	}

	delete(commands, "help")
	for legacy := range legacyTopLevelCommands {
		delete(commands, legacy)
	}

	entries := make([]helpCommand, 0, len(commands))
	for name, desc := range commands {
		name = strings.TrimSpace(name)
		desc = strings.TrimSpace(desc)
		if name == "" {
			continue
		}
		if desc == "" {
			desc = "Parity browser command"
		}
		entries = append(entries, helpCommand{Name: name, Description: desc})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})

	_, _ = fmt.Fprintln(app.Stdout, "AnchorBrowser CLI")
	_, _ = fmt.Fprintln(app.Stdout)
	_, _ = fmt.Fprintln(app.Stdout, "Usage:")
	_, _ = fmt.Fprintln(app.Stdout, "  anchorbrowser [flags]")
	_, _ = fmt.Fprintln(app.Stdout, "  anchorbrowser [command]")
	_, _ = fmt.Fprintln(app.Stdout)
	_, _ = fmt.Fprintln(app.Stdout, "Examples:")
	_, _ = fmt.Fprintln(app.Stdout, "  anchorbrowser open https://example.com")
	_, _ = fmt.Fprintln(app.Stdout, "  anchorbrowser snapshot -i")
	_, _ = fmt.Fprintln(app.Stdout, "  anchorbrowser click @e1")
	_, _ = fmt.Fprintln(app.Stdout, "  anchorbrowser anchor session create --interactive")
	_, _ = fmt.Fprintln(app.Stdout)
	_, _ = fmt.Fprintln(app.Stdout, "Available Commands:")
	printCommandTable(app.Stdout, entries)
	_, _ = fmt.Fprintln(app.Stdout)
	_, _ = fmt.Fprintln(app.Stdout, "Flags:")
	_, _ = fmt.Fprintln(app.Stdout, "      --api-key string     API key value (highest precedence)")
	_, _ = fmt.Fprintln(app.Stdout, "      --base-url string    API base URL (default \"https://api.anchorbrowser.io\")")
	_, _ = fmt.Fprintln(app.Stdout, "      --compact            Compact output (json only)")
	_, _ = fmt.Fprintln(app.Stdout, "      --dry-run            Print request payloads without sending API calls")
	_, _ = fmt.Fprintln(app.Stdout, "  -h, --help               help for anchorbrowser")
	_, _ = fmt.Fprintln(app.Stdout, "      --key string         Named API key profile to use")
	_, _ = fmt.Fprintln(app.Stdout, "      --output string      Output format: json|yaml (default \"json\")")
	_, _ = fmt.Fprintln(app.Stdout, "      --timeout duration   HTTP request timeout (default 2m0s)")
	_, _ = fmt.Fprintln(app.Stdout, "      --verbose            Verbose request logging")
	_, _ = fmt.Fprintln(app.Stdout, "      --version            Print version information")
	_, _ = fmt.Fprintln(app.Stdout)
	_, _ = fmt.Fprintln(app.Stdout, "Use \"anchorbrowser [command] --help\" for more information about a command.")
	return nil
}

func printCommandTable(w io.Writer, entries []helpCommand) {
	width := 0
	for _, entry := range entries {
		if len(entry.Name) > width {
			width = len(entry.Name)
		}
	}
	if width < 6 {
		width = 6
	}
	for _, entry := range entries {
		_, _ = fmt.Fprintf(w, "  %-*s  %s\n", width, entry.Name, entry.Description)
	}
}

func fetchParityRootHelp(app *App) (string, error) {
	manager, err := newHelpBackendManagerFn()
	if err != nil {
		return "", err
	}
	exe, err := manager.EnsureInstalled(context.Background())
	if err != nil {
		return "", err
	}
	out, err := runHelpBackendCommandFn(context.Background(), exe, app.Stdin, "--help")
	if err != nil {
		return "", err
	}
	return string(out), nil
}

var (
	reIndentedCommandLine = regexp.MustCompile(`^\s{2}(.+?)\s{2,}(.+)$`)
	reRelaxedCommandLine  = regexp.MustCompile(`^\s{2}(.+?)\s+([A-Z].+)$`)
	reSectionCommand      = regexp.MustCompile(`^([A-Za-z][A-Za-z ]+):\s+anchorbrowser\s+([a-z][a-z0-9-]*)\b(.*)$`)
	reCommandToken        = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
)

func collectRootCommands(helpText string) map[string]string {
	commands := map[string]string{}
	for _, line := range strings.Split(helpText, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "Install:") {
			break
		}
		if m := reIndentedCommandLine.FindStringSubmatch(line); len(m) == 3 {
			fields := strings.Fields(m[1])
			if len(fields) == 0 {
				continue
			}
			name := strings.TrimSpace(fields[0])
			if !reCommandToken.MatchString(name) {
				continue
			}
			if name == "anchorbrowser" {
				continue
			}
			if _, exists := commands[name]; !exists {
				commands[name] = strings.TrimSpace(m[2])
			}
			continue
		}
		if m := reRelaxedCommandLine.FindStringSubmatch(line); len(m) == 3 {
			fields := strings.Fields(m[1])
			if len(fields) == 0 {
				continue
			}
			name := strings.TrimSpace(fields[0])
			if !reCommandToken.MatchString(name) || name == "anchorbrowser" {
				continue
			}
			if _, exists := commands[name]; !exists {
				commands[name] = strings.TrimSpace(m[2])
			}
			continue
		}
		if m := reSectionCommand.FindStringSubmatch(trimmed); len(m) == 4 {
			name := strings.TrimSpace(m[2])
			if name == "" {
				continue
			}
			if _, exists := commands[name]; !exists {
				commands[name] = strings.TrimSpace(m[1])
			}
		}
	}
	return commands
}

func rewriteParityHelpText(input string) string {
	output := input
	output = strings.ReplaceAll(output, "AGENT_BROWSER_", "ANCHORBROWSER_")
	output = strings.ReplaceAll(output, "~/.agent-browser", "~/.anchorbrowser")
	output = strings.ReplaceAll(output, "agent-browser.json", "anchorbrowser.json")
	output = strings.ReplaceAll(output, "agent-browser", "anchorbrowser")
	return output
}

func runHelpBackendCommand(ctx context.Context, exe string, stdin io.Reader, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, exe, args...)
	cmd.Stdin = stdin
	return cmd.CombinedOutput()
}
