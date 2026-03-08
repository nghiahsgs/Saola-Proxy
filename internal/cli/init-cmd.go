package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nguyennghia/saola-proxy/internal/config"
	"github.com/spf13/cobra"
)

// defaultConfigYAML is the annotated default config written by `saola init`.
const defaultConfigYAML = `# Saola Proxy Configuration
# https://github.com/nguyennghia/saola-proxy
version: 1

# Log level: debug | info | warn | error
log_level: info

# Write a session audit file after each saola wrap run.
audit_enabled: true

patterns:
  # Names of built-in patterns to disable.
  # Available: aws-access-key, github-token, stripe-key, generic-api-key,
  #            private-key, jwt, connection-string, email, ssn, credit-card,
  #            phone-us, ipv4-address, env-variable
  disabled: []

  # User-defined patterns added on top of the built-ins.
  # custom:
  #   - name: my-internal-token
  #     category: secret
  #     regex: 'INTERNAL-[A-Z0-9]{32}'
  #     description: Internal service token
  custom: []

# Values that are never redacted even when matched by a pattern.
whitelist:
  - 127.0.0.1
  - 0.0.0.0
  - localhost
  - example.com
  - test@example.com
`

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a default Saola Proxy config file",
	Long: `init writes a default config to ~/.saola/config.yaml (or
$XDG_CONFIG_HOME/saola/config.yaml) with comments explaining each option.

If the file already exists the command prints a warning and exits without
overwriting it.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := config.ConfigDir()
		if err != nil {
			return fmt.Errorf("config directory: %w", err)
		}
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("creating config directory: %w", err)
		}

		path := filepath.Join(dir, "config.yaml")
		if _, err := os.Stat(path); err == nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: config already exists at %s — skipping\n", path)
			return nil
		}

		if err := os.WriteFile(path, []byte(defaultConfigYAML), 0600); err != nil {
			return fmt.Errorf("writing config: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Config written to %s\n", path)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
