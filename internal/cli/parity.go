package cli

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/anchorbrowser/cli/internal/backend"
	"github.com/anchorbrowser/cli/internal/config"
)

type parityGlobalArgs struct {
	APIKey     string
	KeyName    string
	BaseURL    string
	Timeout    time.Duration
	TimeoutSet bool
	Output     string
	OutputSet  bool
	Compact    bool
	CompactSet bool
	DryRun     bool
	DryRunSet  bool
	Verbose    bool
	VerboseSet bool
}

type paritySessionArgs struct {
	SessionID  string
	NewSession bool
	NoCache    bool
}

type parityParsedArgs struct {
	Global       parityGlobalArgs
	Session      paritySessionArgs
	BackendArgs  []string
	CommandToken string
}

type parityBackendInstaller interface {
	EnsureInstalled(ctx context.Context) (string, error)
}

var (
	newParityBackendManagerFn = func() (parityBackendInstaller, error) {
		return backend.NewManager(config.DefaultAppName)
	}
	runParityStreamingCommandFn = runParityStreamingCommand
	runParityCombinedCommandFn  = runParityCombinedCommand
)

func parseParityArgs(args []string) (*parityParsedArgs, error) {
	parsed := &parityParsedArgs{
		BackendArgs: make([]string, 0, len(args)),
	}

	readBool := func(raw string) (bool, error) {
		v, err := strconv.ParseBool(strings.TrimSpace(raw))
		if err != nil {
			return false, fmt.Errorf("invalid boolean value %q", raw)
		}
		return v, nil
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			parsed.BackendArgs = append(parsed.BackendArgs, args[i:]...)
			break
		}

		switch {
		case arg == "--api-key":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--api-key requires a value")
			}
			i++
			parsed.Global.APIKey = args[i]
		case strings.HasPrefix(arg, "--api-key="):
			parsed.Global.APIKey = strings.TrimPrefix(arg, "--api-key=")
		case arg == "--key":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--key requires a value")
			}
			i++
			parsed.Global.KeyName = args[i]
		case strings.HasPrefix(arg, "--key="):
			parsed.Global.KeyName = strings.TrimPrefix(arg, "--key=")
		case arg == "--base-url":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--base-url requires a value")
			}
			i++
			parsed.Global.BaseURL = args[i]
		case strings.HasPrefix(arg, "--base-url="):
			parsed.Global.BaseURL = strings.TrimPrefix(arg, "--base-url=")
		case arg == "--timeout":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--timeout requires a value")
			}
			i++
			d, err := time.ParseDuration(args[i])
			if err != nil {
				return nil, fmt.Errorf("parse --timeout: %w", err)
			}
			parsed.Global.Timeout = d
			parsed.Global.TimeoutSet = true
		case strings.HasPrefix(arg, "--timeout="):
			d, err := time.ParseDuration(strings.TrimPrefix(arg, "--timeout="))
			if err != nil {
				return nil, fmt.Errorf("parse --timeout: %w", err)
			}
			parsed.Global.Timeout = d
			parsed.Global.TimeoutSet = true
		case arg == "--output":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--output requires a value")
			}
			i++
			parsed.Global.Output = args[i]
			parsed.Global.OutputSet = true
		case strings.HasPrefix(arg, "--output="):
			parsed.Global.Output = strings.TrimPrefix(arg, "--output=")
			parsed.Global.OutputSet = true
		case arg == "--compact":
			parsed.Global.Compact = true
			parsed.Global.CompactSet = true
		case strings.HasPrefix(arg, "--compact="):
			v, err := readBool(strings.TrimPrefix(arg, "--compact="))
			if err != nil {
				return nil, fmt.Errorf("parse --compact: %w", err)
			}
			parsed.Global.Compact = v
			parsed.Global.CompactSet = true
		case arg == "--dry-run":
			parsed.Global.DryRun = true
			parsed.Global.DryRunSet = true
		case strings.HasPrefix(arg, "--dry-run="):
			v, err := readBool(strings.TrimPrefix(arg, "--dry-run="))
			if err != nil {
				return nil, fmt.Errorf("parse --dry-run: %w", err)
			}
			parsed.Global.DryRun = v
			parsed.Global.DryRunSet = true
		case arg == "--verbose":
			parsed.Global.Verbose = true
			parsed.Global.VerboseSet = true
		case strings.HasPrefix(arg, "--verbose="):
			v, err := readBool(strings.TrimPrefix(arg, "--verbose="))
			if err != nil {
				return nil, fmt.Errorf("parse --verbose: %w", err)
			}
			parsed.Global.Verbose = v
			parsed.Global.VerboseSet = true

		case arg == "--session-id":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--session-id requires a value")
			}
			i++
			parsed.Session.SessionID = args[i]
		case strings.HasPrefix(arg, "--session-id="):
			parsed.Session.SessionID = strings.TrimPrefix(arg, "--session-id=")
		case arg == "--new-session":
			parsed.Session.NewSession = true
		case strings.HasPrefix(arg, "--new-session="):
			v, err := readBool(strings.TrimPrefix(arg, "--new-session="))
			if err != nil {
				return nil, fmt.Errorf("parse --new-session: %w", err)
			}
			parsed.Session.NewSession = v
		case arg == "--no-cache":
			parsed.Session.NoCache = true
		case strings.HasPrefix(arg, "--no-cache="):
			v, err := readBool(strings.TrimPrefix(arg, "--no-cache="))
			if err != nil {
				return nil, fmt.Errorf("parse --no-cache: %w", err)
			}
			parsed.Session.NoCache = v
		default:
			parsed.BackendArgs = append(parsed.BackendArgs, arg)
		}
	}

	if len(parsed.BackendArgs) == 0 {
		return nil, fmt.Errorf("no command provided")
	}
	parsed.CommandToken = firstNonFlagToken(parsed.BackendArgs)
	if parsed.CommandToken == "" {
		parsed.CommandToken = parsed.BackendArgs[0]
	}
	return parsed, nil
}

type paritySessionTarget struct {
	ID      string
	CDPURL  string
	Created bool
}

func runParityCommand(app *App, parsed *parityParsedArgs) error {
	manager, err := newParityBackendManagerFn()
	if err != nil {
		return err
	}

	if parsed.Global.DryRun {
		_, _ = fmt.Fprintf(app.Stdout, "DRY RUN: backend command %q\n", strings.Join(parsed.BackendArgs, " "))
		return nil
	}

	commandArgs := append([]string(nil), parsed.BackendArgs...)
	var sessionTarget *paritySessionTarget
	var apiKey string
	injectCDP := shouldInjectCDP(parsed.CommandToken) && !hasHelpOrVersionFlag(parsed.BackendArgs)
	if injectCDP {
		if hasCDPFlag(commandArgs) {
			return fmt.Errorf("`--cdp` is managed by anchorbrowser. use `--session-id` to target a specific Anchor session")
		}
		resolved, err := app.resolveAPIKey()
		if err != nil {
			return err
		}
		apiKey = resolved.Value
		target, err := app.resolveParitySession(context.Background(), apiKey, parsed.Session)
		if err != nil {
			return err
		}
		sessionTarget = target
		commandArgs = append([]string{"--cdp", target.CDPURL}, commandArgs...)
	}

	exe, err := manager.EnsureInstalled(context.Background())
	if err != nil {
		return err
	}

	if hasHelpOrVersionFlag(commandArgs) {
		out, err := runParityCombinedCommandFn(context.Background(), exe, app.Stdin, commandArgs...)
		if len(out) > 0 {
			_, _ = fmt.Fprint(app.Stdout, rewriteParityHelpText(string(out)))
		}
		if err != nil {
			return fmt.Errorf("agent-browser command failed: %w", err)
		}
		return nil
	}

	runErr := runParityStreamingCommandFn(context.Background(), exe, app.Stdin, app.Stdout, app.Stderr, commandArgs...)

	if isCloseCommand(parsed.CommandToken) && sessionTarget != nil && apiKey != "" {
		closeErr := app.endParitySession(context.Background(), apiKey, sessionTarget.ID)
		if closeErr != nil && runErr == nil {
			return closeErr
		}
	}
	if runErr != nil {
		return fmt.Errorf("agent-browser command failed: %w", runErr)
	}
	return nil
}

func shouldInjectCDP(commandToken string) bool {
	token := strings.TrimSpace(strings.ToLower(commandToken))
	switch token {
	case "", "help", "install", "--help", "-h", "--version", "-v", "-V", "version":
		return false
	default:
		return true
	}
}

func isCloseCommand(commandToken string) bool {
	token := strings.TrimSpace(strings.ToLower(commandToken))
	return token == "close" || token == "quit" || token == "exit"
}

func hasCDPFlag(args []string) bool {
	for i := range args {
		if args[i] == "--cdp" || strings.HasPrefix(args[i], "--cdp=") {
			return true
		}
	}
	return false
}

func hasHelpOrVersionFlag(args []string) bool {
	for i := range args {
		switch args[i] {
		case "-h", "--help", "-v", "--version", "-V":
			return true
		}
	}
	return false
}

func runParityCombinedCommand(ctx context.Context, exe string, stdin io.Reader, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, exe, args...)
	cmd.Stdin = stdin
	return cmd.CombinedOutput()
}

func runParityStreamingCommand(ctx context.Context, exe string, stdin io.Reader, stdout, stderr io.Writer, args ...string) error {
	cmd := exec.CommandContext(ctx, exe, args...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

func (a *App) resolveParitySession(ctx context.Context, apiKey string, opts paritySessionArgs) (*paritySessionTarget, error) {
	if strings.TrimSpace(opts.SessionID) != "" {
		target, err := a.fetchParitySession(ctx, apiKey, strings.TrimSpace(opts.SessionID))
		if err != nil {
			return nil, err
		}
		a.printSessionTarget(target.ID, "flag")
		return target, nil
	}

	if !opts.NewSession && !opts.NoCache {
		cfg, err := a.Config.Load()
		if err != nil {
			return nil, err
		}
		cached := strings.TrimSpace(cfg.LastSessionID)
		if cached != "" {
			target, err := a.fetchParitySession(ctx, apiKey, cached)
			if err == nil {
				a.printSessionTarget(target.ID, "cached")
				return target, nil
			}
			_, _ = fmt.Fprintf(a.Stderr, "Cached session %s unavailable, creating a new session.\n", cached)
		}
	}

	target, err := a.createParitySession(ctx, apiKey)
	if err != nil {
		return nil, err
	}
	if !opts.NoCache {
		if err := a.cacheSessionID(target.ID); err != nil {
			return nil, err
		}
	}
	a.printSessionTarget(target.ID, "created")
	return target, nil
}

func (a *App) fetchParitySession(ctx context.Context, apiKey, sessionID string) (*paritySessionTarget, error) {
	result, err := a.newAPIClient().SessionGet(ctx, apiKey, sessionID)
	if err != nil {
		return nil, err
	}
	cdpURL, err := a.resolveParityCDPURL(ctx, apiKey, sessionID, extractSessionCDPURLFromResponse(result))
	if err != nil {
		return nil, err
	}
	return &paritySessionTarget{
		ID:     sessionID,
		CDPURL: cdpURL,
	}, nil
}

func (a *App) createParitySession(ctx context.Context, apiKey string) (*paritySessionTarget, error) {
	payload := map[string]any{
		"session": map[string]any{
			"proxy": map[string]any{
				"active": true,
				"type":   "anchor_proxy",
			},
		},
		"browser": map[string]any{
			"extra_stealth":  map[string]any{"active": true},
			"captcha_solver": map[string]any{"active": true},
		},
	}
	result, err := a.newAPIClient().SessionCreate(ctx, apiKey, payload)
	if err != nil {
		return nil, err
	}
	id := extractSessionIDFromResponse(result)
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("session create response missing id")
	}
	cdpURL, err := a.resolveParityCDPURL(ctx, apiKey, id, extractSessionCDPURLFromResponse(result))
	if err != nil {
		return nil, err
	}
	return &paritySessionTarget{
		ID:      id,
		CDPURL:  cdpURL,
		Created: true,
	}, nil
}

func (a *App) resolveParityCDPURL(ctx context.Context, apiKey, sessionID, fallback string) (string, error) {
	const maxAttempts = 10
	const backoff = 300 * time.Millisecond

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		pages, err := a.newAPIClient().SessionPages(ctx, apiKey, sessionID)
		if err == nil {
			if pageID := extractSessionPrimaryPageIDFromPagesResponse(pages); pageID != "" {
				return buildSessionCDPURLFromPage(sessionID, pageID), nil
			}
			lastErr = fmt.Errorf("session %s pages response missing page id", sessionID)
		} else {
			lastErr = err
		}

		if attempt == maxAttempts-1 {
			break
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(backoff):
		}
	}

	fallback = strings.TrimSpace(fallback)
	if fallback != "" {
		return fallback, nil
	}
	if lastErr != nil {
		return "", fmt.Errorf("resolve cdp url for session %s: %w", sessionID, lastErr)
	}
	return "", fmt.Errorf("session %s does not expose cdp_url", sessionID)
}

func (a *App) endParitySession(ctx context.Context, apiKey, sessionID string) error {
	if strings.TrimSpace(sessionID) == "" {
		return nil
	}
	if _, err := a.newAPIClient().SessionEnd(ctx, apiKey, sessionID); err != nil {
		return err
	}
	if err := a.clearSessionIDCache(); err != nil {
		return err
	}
	return nil
}
