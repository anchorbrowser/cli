package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func (a *App) cacheSessionID(sessionID string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil
	}
	cfg, err := a.Config.Load()
	if err != nil {
		return err
	}
	cfg.LastSessionID = sessionID
	return a.Config.Save(cfg)
}

func (a *App) clearSessionIDCache() error {
	cfg, err := a.Config.Load()
	if err != nil {
		return err
	}
	if strings.TrimSpace(cfg.LastSessionID) == "" {
		return nil
	}
	cfg.LastSessionID = ""
	return a.Config.Save(cfg)
}

func (a *App) resolveSessionID(cmd *cobra.Command) (string, error) {
	flagSessionID, _ := cmd.Flags().GetString("session-id")
	flagSessionID = strings.TrimSpace(flagSessionID)
	if flagSessionID != "" {
		a.printSessionTarget(flagSessionID, "flag")
		return flagSessionID, nil
	}

	noCache, _ := cmd.Flags().GetBool("no-cache")
	if noCache {
		return "", fmt.Errorf("session ID required when --no-cache is set (pass --session-id)")
	}

	cfg, err := a.Config.Load()
	if err != nil {
		return "", err
	}
	cached := strings.TrimSpace(cfg.LastSessionID)
	if cached == "" {
		return "", fmt.Errorf("session ID required (pass --session-id, or create a session first)")
	}
	a.printSessionTarget(cached, "cached")
	return cached, nil
}

func (a *App) printSessionTarget(sessionID, source string) {
	if a == nil || a.Stderr == nil || strings.TrimSpace(sessionID) == "" {
		return
	}
	if source == "cached" {
		fmt.Fprintf(a.Stderr, "Using session: %s (cached latest)\n", sessionID)
		return
	}
	fmt.Fprintf(a.Stderr, "Using session: %s\n", sessionID)
}

func extractSessionIDFromResponse(v any) string {
	root, ok := v.(map[string]any)
	if !ok {
		return ""
	}
	data, ok := root["data"].(map[string]any)
	if !ok {
		return ""
	}
	id, ok := data["id"].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(id)
}

func extractSessionCDPURLFromResponse(v any) string {
	root, ok := v.(map[string]any)
	if !ok {
		return ""
	}
	data, ok := root["data"].(map[string]any)
	if !ok {
		return ""
	}
	cdpURL, ok := data["cdp_url"].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(cdpURL)
}
