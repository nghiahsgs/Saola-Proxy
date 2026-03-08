// Package cli provides the command-line interface for Saola Proxy.
// Saola Proxy is a transparent CLI wrapper that intercepts stdout/stderr
// from any command, detects PII/secrets using pattern matching, sanitizes
// them before sending to AI tools, and rehydrates the AI response.
package cli

import (
	"github.com/spf13/cobra"
)

// globalConfigPath is set via --config on the root command and consumed by wrap.
var globalConfigPath string

var rootCmd = &cobra.Command{
	Use:   "saola",
	Short: "Saola Proxy - Transparent PII-sanitizing CLI wrapper for AI tools",
	Long: `Saola Proxy intercepts command output, detects and redacts PII/secrets
before forwarding to AI assistants, then rehydrates AI responses with
the original values. Protect your secrets without changing your workflow.

Usage examples:
  saola --config ~/.saola/config.yaml wrap -- claude
  saola wrap -- cat logs/app.log | claude
  saola wrap -- kubectl logs my-pod | llm
  saola version`,
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&globalConfigPath, "config", "", "Path to config file (overrides default discovery)")
}
