package cli

import (
	"context"
	"errors"
	"testing"
)

func TestDetectUpdateStrategyWithPrefersBrewWhenInstalled(t *testing.T) {
	lookPath := func(file string) (string, error) {
		if file == "brew" {
			return "/opt/homebrew/bin/brew", nil
		}
		return "", errors.New("not found")
	}
	runCapture := func(_ context.Context, _ string, _ ...string) (string, error) {
		return "anchorbrowser 0.1.20", nil
	}

	strategy, err := detectUpdateStrategyWith(context.Background(), lookPath, runCapture)
	if err != nil {
		t.Fatalf("detectUpdateStrategyWith: %v", err)
	}
	if strategy.Name != "Homebrew" {
		t.Fatalf("expected Homebrew strategy, got %s", strategy.Name)
	}
}

func TestDetectUpdateStrategyWithFallsBackToNpm(t *testing.T) {
	lookPath := func(file string) (string, error) {
		switch file {
		case "brew":
			return "/opt/homebrew/bin/brew", nil
		case "npm":
			return "/usr/local/bin/npm", nil
		default:
			return "", errors.New("not found")
		}
	}
	runCapture := func(_ context.Context, name string, _ ...string) (string, error) {
		if name == "brew" {
			return "", errors.New("brew formula missing")
		}
		return "@anchor-browser/cli@0.1.20", nil
	}

	strategy, err := detectUpdateStrategyWith(context.Background(), lookPath, runCapture)
	if err != nil {
		t.Fatalf("detectUpdateStrategyWith: %v", err)
	}
	if strategy.Name != "npm" {
		t.Fatalf("expected npm strategy, got %s", strategy.Name)
	}
}

func TestDetectUpdateStrategyWithReturnsErrorWhenUnknown(t *testing.T) {
	lookPath := func(_ string) (string, error) {
		return "", errors.New("not found")
	}
	runCapture := func(_ context.Context, _ string, _ ...string) (string, error) {
		return "", errors.New("not found")
	}

	_, err := detectUpdateStrategyWith(context.Background(), lookPath, runCapture)
	if err == nil {
		t.Fatalf("expected error when update strategy is unknown")
	}
}
