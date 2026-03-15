package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

func parseBodyAsMap(bodyFlag string) (map[string]any, error) {
	bodyFlag = strings.TrimSpace(bodyFlag)
	if bodyFlag == "" {
		return map[string]any{}, nil
	}

	var raw []byte
	var err error
	switch {
	case bodyFlag == "-":
		raw, err = io.ReadAll(os.Stdin)
	case strings.HasPrefix(bodyFlag, "{") || strings.HasPrefix(bodyFlag, "["):
		raw = []byte(bodyFlag)
	default:
		raw, err = os.ReadFile(bodyFlag)
	}
	if err != nil {
		return nil, fmt.Errorf("read --body: %w", err)
	}

	if len(strings.TrimSpace(string(raw))) == 0 {
		return map[string]any{}, nil
	}

	var out map[string]any
	if err := json.Unmarshal(raw, &out); err == nil {
		return out, nil
	}
	if err := yaml.Unmarshal(raw, &out); err == nil {
		return out, nil
	}
	return nil, fmt.Errorf("--body must be valid json or yaml object")
}

func parseJSONObjectFlag(raw string) (map[string]any, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("invalid json object: %w", err)
	}
	return out, nil
}

func ensureMap(root map[string]any, key string) map[string]any {
	if root == nil {
		root = map[string]any{}
	}
	if current, ok := root[key].(map[string]any); ok {
		return current
	}
	current := map[string]any{}
	root[key] = current
	return current
}

func mergeMap(dst, src map[string]any) map[string]any {
	if dst == nil {
		dst = map[string]any{}
	}
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func parseKV(input []string) (map[string]string, error) {
	out := map[string]string{}
	for _, pair := range input {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
			return nil, fmt.Errorf("invalid key=value pair %q", pair)
		}
		out[strings.TrimSpace(parts[0])] = parts[1]
	}
	return out, nil
}

func writeBinary(outPath string, data []byte) error {
	if outPath == "" || outPath == "-" {
		_, err := os.Stdout.Write(data)
		if err != nil {
			return fmt.Errorf("write stdout: %w", err)
		}
		return nil
	}
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		return fmt.Errorf("write %q: %w", outPath, err)
	}
	return nil
}

func redactSensitive(v any, reveal bool) any {
	if reveal {
		return v
	}
	return redactValue(v)
}

func redactValue(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			if isSensitiveKey(k) {
				out[k] = "***REDACTED***"
				continue
			}
			out[k] = redactValue(val)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, val := range t {
			out[i] = redactValue(val)
		}
		return out
	default:
		return v
	}
}

func isSensitiveKey(key string) bool {
	k := strings.ToLower(strings.TrimSpace(key))
	switch k {
	case "password", "secret", "otp", "token", "api_key", "api-key", "access_token":
		return true
	default:
		return false
	}
}
