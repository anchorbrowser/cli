package cli

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestValidateInteractiveSessionCreateCompatibility(t *testing.T) {
	cmd := &cobra.Command{Use: "create"}
	cmd.Flags().String("body", "", "")
	cmd.Flags().String("initial-url", "", "")
	cmd.Flags().StringSlice("tag", nil, "")
	cmd.Flags().Bool("recording", true, "")
	cmd.Flags().Bool("proxy-active", false, "")
	cmd.Flags().String("proxy-type", "", "")
	cmd.Flags().String("proxy-country-code", "", "")
	cmd.Flags().String("proxy-region", "", "")
	cmd.Flags().String("proxy-city", "", "")
	cmd.Flags().Int("max-duration", 0, "")
	cmd.Flags().Int("idle-timeout", 0, "")
	cmd.Flags().Bool("headless", false, "")
	cmd.Flags().Int("viewport-width", 0, "")
	cmd.Flags().Int("viewport-height", 0, "")
	cmd.Flags().String("profile-name", "", "")
	cmd.Flags().Bool("profile-persist", false, "")
	cmd.Flags().StringSlice("identity-id", nil, "")
	cmd.Flags().StringSlice("integration-id", nil, "")

	if err := cmd.Flags().Set("body", "{}"); err != nil {
		t.Fatalf("set body: %v", err)
	}
	if err := validateInteractiveSessionCreateCompatibility(cmd); err == nil {
		t.Fatalf("expected incompatibility error")
	}
}

func TestApplyRecommendedAntiBotPayload(t *testing.T) {
	payload := map[string]any{}
	applyRecommendedAntiBotPayload(payload)

	session := payload["session"].(map[string]any)
	proxy := session["proxy"].(map[string]any)
	if proxy["active"] != true || proxy["type"] != "anchor_proxy" {
		t.Fatalf("unexpected proxy payload: %#v", proxy)
	}

	browser := payload["browser"].(map[string]any)
	extraStealth := browser["extra_stealth"].(map[string]any)
	captcha := browser["captcha_solver"].(map[string]any)
	if extraStealth["active"] != true || captcha["active"] != true {
		t.Fatalf("unexpected browser payload: %#v", browser)
	}
}

func TestRequiresManualIdentityFlow(t *testing.T) {
	if requiresManualIdentityFlow([]string{"username_password", "authenticator", "custom"}) {
		t.Fatalf("expected supported flow not to require manual mode")
	}
	if !requiresManualIdentityFlow([]string{"profile"}) {
		t.Fatalf("profile flow should require manual mode")
	}
	if !requiresManualIdentityFlow([]string{"gmail_mfa"}) {
		t.Fatalf("unsupported method should require manual mode")
	}
}

func TestPromptCredentialsForFlow(t *testing.T) {
	in := strings.NewReader("user@example.com\npass123\nJBSWY3DPEHPK3PXP\n123456\notp_backup\n")
	reader := bufio.NewReader(in)
	out := &bytes.Buffer{}

	creds, name, err := promptCredentialsForFlow(reader, out, interactiveAuthFlow{
		Methods:      []string{"username_password", "authenticator", "custom"},
		CustomFields: []string{"backup_code"},
	})
	if err != nil {
		t.Fatalf("promptCredentialsForFlow: %v", err)
	}
	if name != "user@example.com" {
		t.Fatalf("unexpected identity name: %s", name)
	}
	if len(creds) != 3 {
		t.Fatalf("expected 3 credentials, got %d", len(creds))
	}
}

func TestFindRecentIdentityByNameFromResponse(t *testing.T) {
	now := time.Now().UTC()
	result := map[string]any{
		"identities": []any{
			map[string]any{
				"id":         "11111111-1111-4111-8111-111111111111",
				"name":       "cli-identity",
				"created_at": now.Add(-10 * time.Minute).Format(time.RFC3339),
			},
			map[string]any{
				"id":         "22222222-2222-4222-8222-222222222222",
				"name":       "cli-identity older",
				"created_at": now.Add(-2 * time.Hour).Format(time.RFC3339),
			},
		},
	}
	rows := extractIdentityRows(result)
	if len(rows) != 2 {
		t.Fatalf("expected 2 identity rows")
	}
	// Validate helper logic with time window and contains matching.
	bestID := ""
	bestTime := time.Time{}
	for _, row := range rows {
		name := strings.ToLower(strings.TrimSpace(firstString(row["name"])))
		if !strings.Contains(name, "cli-identity") {
			continue
		}
		createdAt, err := time.Parse(time.RFC3339, firstString(row["created_at"]))
		if err != nil || now.Sub(createdAt) > 45*time.Minute {
			continue
		}
		if bestID == "" || createdAt.After(bestTime) {
			bestID = firstString(row["id"])
			bestTime = createdAt
		}
	}
	if bestID != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("unexpected best identity id: %s", bestID)
	}
}
