package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/anchorbrowser/cli/internal/config"
)

var reservedCommands = map[string]struct{}{
	"auth":         {},
	"update":       {},
	"version":      {},
	"backend":      {},
	"anchor":       {},
	"help":         {},
	"__completion": {},
}

var legacyTopLevelCommands = map[string]struct{}{
	"session":  {},
	"identity": {},
	"task":     {},
}

var (
	newRootCommandFn   = NewRootCommand
	newAppFn           = NewApp
	runParityCommandFn = runParityCommand
	renderRootHelpFn   = renderRootHelp
)

func Execute(version string, args []string) error {
	return executeWithIO(version, args, os.Stdin, os.Stdout, os.Stderr)
}

func executeWithIO(version string, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	first := firstNonFlagToken(args)
	if len(args) == 0 {
		return runRootHelp(version, stdin, stdout, stderr)
	}
	if first == "" && hasAny(args, "-h", "--help") && !hasAny(args, "--version", "-v", "-V") {
		return runRootHelp(version, stdin, stdout, stderr)
	}
	if first == "help" {
		return runHelpCommand(version, args, stdin, stdout, stderr)
	}
	if first == "" || isReservedCommand(first) {
		rootCmd, err := newRootCommandFn(version)
		if err != nil {
			return err
		}
		rootCmd.SetIn(stdin)
		rootCmd.SetOut(stdout)
		rootCmd.SetErr(stderr)
		rootCmd.SetArgs(args)
		return rootCmd.Execute()
	}
	if _, moved := legacyTopLevelCommands[first]; moved && !hasHelpOrVersionFlag(args) {
		return fmt.Errorf("`%s` moved under the anchor namespace. use `anchorbrowser anchor %s ...`", first, first)
	}

	app, err := newAppFn(version)
	if err != nil {
		return err
	}
	app.Stdin = stdin
	app.Stdout = stdout
	app.Stderr = stderr

	parsed, err := parseParityArgs(args)
	if err != nil {
		return err
	}
	applyParityGlobals(app, parsed.Global)
	return runParityCommandFn(app, parsed)
}

func isReservedCommand(token string) bool {
	_, ok := reservedCommands[strings.TrimSpace(token)]
	return ok
}

func firstNonFlagToken(args []string) string {
	skipNext := false
	for _, arg := range args {
		if skipNext {
			skipNext = false
			continue
		}
		if strings.TrimSpace(arg) == "" {
			continue
		}
		if strings.HasPrefix(arg, "--") {
			name := arg
			if idx := strings.Index(name, "="); idx >= 0 {
				name = name[:idx]
			}
			switch name {
			case "--api-key", "--key", "--base-url", "--timeout", "--output":
				if !strings.Contains(arg, "=") {
					skipNext = true
				}
			}
			continue
		}
		if strings.HasPrefix(arg, "-") {
			continue
		}
		return arg
	}
	return ""
}

func applyParityGlobals(app *App, global parityGlobalArgs) {
	if strings.TrimSpace(global.APIKey) != "" {
		app.Global.APIKey = strings.TrimSpace(global.APIKey)
	}
	if strings.TrimSpace(global.KeyName) != "" {
		app.Global.KeyName = strings.TrimSpace(global.KeyName)
	}
	if strings.TrimSpace(global.BaseURL) != "" {
		app.Global.BaseURL = strings.TrimSpace(global.BaseURL)
	}
	if global.TimeoutSet {
		app.Global.Timeout = global.Timeout
	}
	if global.OutputSet {
		app.Global.Output = global.Output
	}
	if global.CompactSet {
		app.Global.Compact = global.Compact
	}
	if global.DryRunSet {
		app.Global.DryRun = global.DryRun
	}
	if global.VerboseSet {
		app.Global.Verbose = global.Verbose
	}
	if strings.TrimSpace(app.Global.BaseURL) == "" {
		app.Global.BaseURL = "https://api.anchorbrowser.io"
	}
	if app.Config == nil {
		manager, _ := config.NewManager(config.DefaultAppName)
		app.Config = manager
	}
}

func runRootHelp(version string, stdin io.Reader, stdout, stderr io.Writer) error {
	app, err := newAppFn(version)
	if err != nil {
		return err
	}
	app.Stdin = stdin
	app.Stdout = stdout
	app.Stderr = stderr
	return renderRootHelpFn(app)
}

func runHelpCommand(version string, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	withoutHelp, _ := removeFirstPositionalToken(args)
	target := firstNonFlagToken(withoutHelp)
	if target == "" {
		return runRootHelp(version, stdin, stdout, stderr)
	}
	if isReservedCommand(target) {
		rootCmd, err := newRootCommandFn(version)
		if err != nil {
			return err
		}
		rootCmd.SetIn(stdin)
		rootCmd.SetOut(stdout)
		rootCmd.SetErr(stderr)
		rootCmd.SetArgs(args)
		return rootCmd.Execute()
	}

	if !hasHelpOrVersionFlag(withoutHelp) {
		withoutHelp = append(withoutHelp, "--help")
	}

	app, err := newAppFn(version)
	if err != nil {
		return err
	}
	app.Stdin = stdin
	app.Stdout = stdout
	app.Stderr = stderr

	parsed, err := parseParityArgs(withoutHelp)
	if err != nil {
		return err
	}
	applyParityGlobals(app, parsed.Global)
	return runParityCommandFn(app, parsed)
}

func removeFirstPositionalToken(args []string) ([]string, string) {
	out := make([]string, 0, len(args))
	skipNext := false
	removed := ""
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if skipNext {
			skipNext = false
			out = append(out, arg)
			continue
		}
		if strings.TrimSpace(arg) == "" {
			out = append(out, arg)
			continue
		}
		if strings.HasPrefix(arg, "--") {
			name := arg
			if idx := strings.Index(name, "="); idx >= 0 {
				name = name[:idx]
			}
			switch name {
			case "--api-key", "--key", "--base-url", "--timeout", "--output":
				if !strings.Contains(arg, "=") {
					skipNext = true
				}
			}
			out = append(out, arg)
			continue
		}
		if strings.HasPrefix(arg, "-") {
			out = append(out, arg)
			continue
		}
		if removed == "" {
			removed = arg
			continue
		}
		out = append(out, arg)
	}
	return out, removed
}

func hasAny(args []string, values ...string) bool {
	for _, arg := range args {
		for _, v := range values {
			if arg == v {
				return true
			}
		}
	}
	return false
}
