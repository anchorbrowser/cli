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

func (a *App) resolveSessionID(cmd *cobra.Command, args []string) (string, error) {
	flagSessionID, _ := cmd.Flags().GetString("session-id")
	flagSessionID = strings.TrimSpace(flagSessionID)

	var argSessionID string
	if len(args) > 0 {
		argSessionID = strings.TrimSpace(args[0])
	}

	if argSessionID != "" && flagSessionID != "" && argSessionID != flagSessionID {
		return "", fmt.Errorf("session ID specified twice with different values (arg=%q, --session-id=%q)", argSessionID, flagSessionID)
	}
	if argSessionID != "" {
		return argSessionID, nil
	}
	if flagSessionID != "" {
		return flagSessionID, nil
	}

	noCache, _ := cmd.Flags().GetBool("no-cache")
	if noCache {
		return "", fmt.Errorf("session ID required when --no-cache is set")
	}

	cfg, err := a.Config.Load()
	if err != nil {
		return "", err
	}
	cached := strings.TrimSpace(cfg.LastSessionID)
	if cached == "" {
		return "", fmt.Errorf("session ID required (pass <session-id>, --session-id, or create a session first)")
	}
	return cached, nil
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
