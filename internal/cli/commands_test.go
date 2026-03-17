package cli

import "testing"

func TestSessionClickValidation(t *testing.T) {
	cmd, err := NewRootCommand("test")
	if err != nil {
		t.Fatalf("NewRootCommand: %v", err)
	}
	cmd.SetArgs([]string{"--api-key", "dummy", "anchor", "session", "click", "--session-id", "sess-1"})
	if err := cmd.Execute(); err == nil {
		t.Fatalf("expected validation error for missing selector/x/y")
	}
}

func TestSessionClickMutuallyExclusiveValidation(t *testing.T) {
	cmd, err := NewRootCommand("test")
	if err != nil {
		t.Fatalf("NewRootCommand: %v", err)
	}
	cmd.SetArgs([]string{"--api-key", "dummy", "anchor", "session", "click", "--session-id", "sess-1", "--selector", "#id", "--x", "1", "--y", "2"})
	if err := cmd.Execute(); err == nil {
		t.Fatalf("expected validation error for selector and x/y together")
	}
}

func TestTaskRunRequiresInput(t *testing.T) {
	cmd, err := NewRootCommand("test")
	if err != nil {
		t.Fatalf("NewRootCommand: %v", err)
	}
	cmd.SetArgs([]string{"--api-key", "dummy", "anchor", "task", "run", "task-1"})
	if err := cmd.Execute(); err == nil {
		t.Fatalf("expected error when no task input is provided")
	}
}

func TestRunAgentIsUnderAnchorSession(t *testing.T) {
	cmd, err := NewRootCommand("test")
	if err != nil {
		t.Fatalf("NewRootCommand: %v", err)
	}

	cmd.SetArgs([]string{"anchor", "session", "run-agent", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected anchor session run-agent command to exist: %v", err)
	}

	cmd, err = NewRootCommand("test")
	if err != nil {
		t.Fatalf("NewRootCommand: %v", err)
	}
	cmd.SetArgs([]string{"session", "--help"})
	if err := cmd.Execute(); err == nil {
		t.Fatalf("expected top-level session command to be unavailable")
	}
}

func TestSessionCreateInteractiveRejectsPayloadFlags(t *testing.T) {
	cmd, err := NewRootCommand("test")
	if err != nil {
		t.Fatalf("NewRootCommand: %v", err)
	}
	cmd.SetArgs([]string{"--api-key", "dummy", "anchor", "session", "create", "--interactive", "--body", "{}"})
	if err := cmd.Execute(); err == nil {
		t.Fatalf("expected validation error for --interactive with --body")
	}
}

func TestSessionCreateInteractiveRejectsDryRun(t *testing.T) {
	cmd, err := NewRootCommand("test")
	if err != nil {
		t.Fatalf("NewRootCommand: %v", err)
	}
	cmd.SetArgs([]string{"--api-key", "dummy", "--dry-run", "anchor", "session", "create", "--interactive"})
	if err := cmd.Execute(); err == nil {
		t.Fatalf("expected validation error for --interactive with --dry-run")
	}
}
