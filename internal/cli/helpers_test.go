package cli

import "testing"

func TestParseBodyAsMapInlineJSON(t *testing.T) {
	body, err := parseBodyAsMap(`{"session":{"initial_url":"https://example.com"}}`)
	if err != nil {
		t.Fatalf("parseBodyAsMap: %v", err)
	}
	session, ok := body["session"].(map[string]any)
	if !ok {
		t.Fatalf("session object missing: %#v", body)
	}
	if session["initial_url"] != "https://example.com" {
		t.Fatalf("unexpected initial_url: %#v", session["initial_url"])
	}
}

func TestRedactSensitive(t *testing.T) {
	in := map[string]any{
		"credentials": []any{
			map[string]any{"type": "username_password", "username": "u", "password": "p"},
			map[string]any{"type": "authenticator", "secret": "s", "otp": "123456"},
		},
	}
	out := redactSensitive(in, false)
	root := out.(map[string]any)
	creds := root["credentials"].([]any)
	first := creds[0].(map[string]any)
	if first["password"] != "***REDACTED***" {
		t.Fatalf("password not redacted: %#v", first)
	}
	second := creds[1].(map[string]any)
	if second["secret"] != "***REDACTED***" || second["otp"] != "***REDACTED***" {
		t.Fatalf("secret/otp not redacted: %#v", second)
	}
}

func TestMergeMap(t *testing.T) {
	base := map[string]any{"a": 1, "b": 2}
	merged := mergeMap(base, map[string]any{"b": 3, "c": 4})
	if merged["a"] != 1 || merged["b"] != 3 || merged["c"] != 4 {
		t.Fatalf("unexpected merged map: %#v", merged)
	}
}
