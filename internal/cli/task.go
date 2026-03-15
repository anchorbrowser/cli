package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/anchorbrowser/cli/internal/api/generated"
)

func newTaskCommand(app *App) *cobra.Command {
	cmd := &cobra.Command{Use: "task", Short: "Run task executions"}
	cmd.AddCommand(newTaskRunCommand(app))
	cmd.AddCommand(newTaskStatusCommand(app))
	return cmd
}

func newTaskRunCommand(app *App) *cobra.Command {
	var inputPairs []string
	var inputFile string
	var identityID, sessionID string
	var cleanupSessions bool

	cmd := &cobra.Command{
		Use:   "run <task-id>",
		Short: "Run a task by task ID (v2)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			resolved, err := app.resolveAPIKey()
			if err != nil {
				return err
			}

			inputs := map[string]string{}
			if cmd.Flags().Changed("input-file") {
				raw, err := os.ReadFile(inputFile)
				if err != nil {
					return fmt.Errorf("read --input-file: %w", err)
				}
				if err := json.Unmarshal(raw, &inputs); err != nil {
					return fmt.Errorf("parse --input-file as json object: %w", err)
				}
			}
			if cmd.Flags().Changed("input") {
				parsed, err := parseKV(inputPairs)
				if err != nil {
					return err
				}
				for k, v := range parsed {
					inputs[k] = v
				}
			}
			if len(inputs) == 0 {
				return fmt.Errorf("at least one --input key=value or --input-file is required")
			}

			payload := generated.RunTaskRequest{InputParams: inputs}
			if cmd.Flags().Changed("identity-id") {
				payload.IdentityId = &identityID
			}
			if cmd.Flags().Changed("session-id") {
				payload.SessionId = &sessionID
			}
			if cmd.Flags().Changed("cleanup-sessions") {
				payload.CleanupSessions = &cleanupSessions
			}

			result, err := app.newAPIClient().TaskRun(cmd.Context(), resolved.Value, args[0], payload)
			return app.printDryRunOrValue(result, err)
		},
	}
	cmd.Flags().StringSliceVar(&inputPairs, "input", nil, "Input key=value pair (repeatable)")
	cmd.Flags().StringVar(&inputFile, "input-file", "", "JSON file with input parameters")
	cmd.Flags().StringVar(&identityID, "identity-id", "", "Identity ID")
	cmd.Flags().StringVar(&sessionID, "session-id", "", "Session ID")
	cmd.Flags().BoolVar(&cleanupSessions, "cleanup-sessions", false, "Clean up sessions after task execution")
	return cmd
}

func newTaskStatusCommand(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "status <run-id>",
		Short: "Get task run status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(args[0]) == "" {
				return fmt.Errorf("run-id cannot be empty")
			}
			resolved, err := app.resolveAPIKey()
			if err != nil {
				return err
			}
			result, err := app.newAPIClient().TaskStatus(cmd.Context(), resolved.Value, args[0])
			return app.printDryRunOrValue(result, err)
		},
	}
}
