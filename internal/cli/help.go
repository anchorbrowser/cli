package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

type helpCommand struct {
	Name        string
	Description string
}

func renderRootHelp(app *App) error {
	entries := []helpCommand{
		{Name: "anchor", Description: "Anchor API commands"},
		{Name: "auth", Description: "Manage API key authentication"},
		{Name: "proxy", Description: "Run agent-browser commands through Anchor proxy"},
		{Name: "update", Description: "Update the AnchorBrowser CLI"},
		{Name: "version", Description: "Print version information"},
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
	_, _ = fmt.Fprintln(app.Stdout, "  anchorbrowser proxy open https://example.com")
	_, _ = fmt.Fprintln(app.Stdout, "  anchorbrowser proxy snapshot -i")
	_, _ = fmt.Fprintln(app.Stdout, "  anchorbrowser proxy click @e1")
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

func rewriteParityHelpText(input string) string {
	output := input
	output = strings.ReplaceAll(output, "AGENT_BROWSER_", "ANCHORBROWSER_")
	output = strings.ReplaceAll(output, "~/.agent-browser", "~/.anchorbrowser")
	output = strings.ReplaceAll(output, "agent-browser.json", "anchorbrowser.json")
	output = strings.ReplaceAll(output, "agent-browser", "anchorbrowser")
	return output
}
