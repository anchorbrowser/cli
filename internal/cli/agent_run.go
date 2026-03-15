package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/anchorbrowser/cli/internal/api/generated"
)

func newAgentRunCommand(app *App) *cobra.Command {
	var prompt, targetURL, sessionID, agent, provider, model, outputSchemaFlag string
	var maxSteps int
	var detectElements, humanIntervention, highlightElements, async bool
	var secrets []string

	cmd := &cobra.Command{
		Use:   "agent-run",
		Short: "Run an autonomous web task",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if strings.TrimSpace(prompt) == "" {
				return fmt.Errorf("--prompt is required")
			}
			resolved, err := app.resolveAPIKey()
			if err != nil {
				return err
			}

			payload := generated.PerformWebTaskRequest{Prompt: prompt}
			if cmd.Flags().Changed("url") {
				payload.Url = &targetURL
			}
			if cmd.Flags().Changed("agent") {
				payload.Agent = &agent
			}
			if cmd.Flags().Changed("provider") {
				payload.Provider = &provider
			}
			if cmd.Flags().Changed("model") {
				payload.Model = &model
			}
			if cmd.Flags().Changed("max-steps") {
				payload.MaxSteps = &maxSteps
			}
			if cmd.Flags().Changed("detect-elements") {
				payload.DetectElements = &detectElements
			}
			if cmd.Flags().Changed("human-intervention") {
				payload.HumanIntervention = &humanIntervention
			}
			if cmd.Flags().Changed("highlight-elements") {
				payload.HighlightElements = &highlightElements
			}
			if cmd.Flags().Changed("async") {
				payload.Async = &async
			}
			if cmd.Flags().Changed("secret") {
				parsed, err := parseKV(secrets)
				if err != nil {
					return err
				}
				payload.SecretValues = &parsed
			}
			if cmd.Flags().Changed("output-schema") {
				schemaMap, err := parseBodyAsMap(outputSchemaFlag)
				if err != nil {
					return fmt.Errorf("parse --output-schema: %w", err)
				}
				payload.OutputSchema = &schemaMap
			}

			client := app.newAPIClient()
			result, err := client.AgentRun(cmd.Context(), resolved.Value, sessionID, payload)
			return app.printDryRunOrValue(result, err)
		},
	}

	cmd.Flags().StringVar(&prompt, "prompt", "", "Task prompt")
	cmd.Flags().StringVar(&targetURL, "url", "", "URL to start from")
	cmd.Flags().StringVar(&sessionID, "session-id", "", "Existing session ID to run inside")
	cmd.Flags().StringVar(&agent, "agent", "", "Agent to use")
	cmd.Flags().StringVar(&provider, "provider", "", "Model provider")
	cmd.Flags().StringVar(&model, "model", "", "Model name")
	cmd.Flags().IntVar(&maxSteps, "max-steps", 0, "Maximum steps")
	cmd.Flags().BoolVar(&detectElements, "detect-elements", false, "Enable element detection")
	cmd.Flags().BoolVar(&humanIntervention, "human-intervention", false, "Allow human intervention")
	cmd.Flags().BoolVar(&highlightElements, "highlight-elements", false, "Highlight elements while running")
	cmd.Flags().BoolVar(&async, "async", false, "Run asynchronously")
	cmd.Flags().StringSliceVar(&secrets, "secret", nil, "Secret key=value pairs (repeatable)")
	cmd.Flags().StringVar(&outputSchemaFlag, "output-schema", "", "Output schema JSON/YAML file path, '-' for stdin, or inline JSON")

	statusCmd := &cobra.Command{
		Use:   "status <workflow-id>",
		Short: "Get status of async agent-run workflow",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			resolved, err := app.resolveAPIKey()
			if err != nil {
				return err
			}
			result, err := app.newAPIClient().AgentRunStatus(cmd.Context(), resolved.Value, args[0])
			return app.printDryRunOrValue(result, err)
		},
	}

	cmd.AddCommand(statusCmd)
	return cmd
}
