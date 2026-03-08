package cli

import (
	"fmt"
	"os"
	"regexp"

	"github.com/nguyennghia/saola-proxy/internal/audit"
	"github.com/nguyennghia/saola-proxy/internal/config"
	"github.com/nguyennghia/saola-proxy/internal/sanitizer"
	"github.com/nguyennghia/saola-proxy/internal/scanner"
	"github.com/nguyennghia/saola-proxy/internal/wrapper"
	"github.com/spf13/cobra"
)

var wrapCmd = &cobra.Command{
	Use:   "wrap -- <command> [args...]",
	Short: "Wrap a command and sanitize its output",
	Long: `Wrap executes the given command and sanitizes PII/secrets from
its stdout and stderr before they reach AI tools.

Example:
  saola wrap -- cat /var/log/app.log
  saola wrap -- kubectl logs my-pod
  saola wrap -- claude`,
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Strip leading "--" separator if present.
		if len(args) > 0 && args[0] == "--" {
			args = args[1:]
		}
		if len(args) == 0 {
			return fmt.Errorf("no command specified; usage: saola wrap -- <command> [args...]")
		}

		command := args[0]
		cmdArgs := args[1:]

		// Load config (explicit path via --config flag, or default discovery).
		var cfg *config.Config
		var err error
		if globalConfigPath != "" {
			cfg, err = config.LoadFromPath(globalConfigPath)
		} else {
			cfg, err = config.Load()
		}
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		// Build scanner with all built-in patterns.
		reg := scanner.NewRegistry()
		scanner.RegisterBuiltins(reg)

		// Disable patterns listed in config.
		for _, name := range cfg.Patterns.Disabled {
			reg.Disable(name)
		}

		// Register custom patterns from config.
		for _, cp := range cfg.Patterns.Custom {
			re, err := regexp.Compile(cp.Regex)
			if err != nil {
				return fmt.Errorf("custom pattern %q: invalid regex: %w", cp.Name, err)
			}
			reg.Register(scanner.Pattern{
				Name:        cp.Name,
				Category:    cp.Category,
				Regex:       re,
				Description: cp.Description,
				Enabled:     true,
			})
		}

		sc := scanner.NewScanner(reg)
		sc.SetWhitelist(cfg.Whitelist)

		// Shared mapping table keeps sanitize/rehydrate in sync.
		table := sanitizer.NewMappingTable()
		san := sanitizer.NewSanitizer(sc, table)
		reh := sanitizer.NewRehydrator(table)

		// Create audit session.
		session := audit.NewSession(command)

		// Wire OnDetection callback into sanitizer.
		san.OnDetection = func(patternName string) {
			session.RecordDetection(patternName)
		}

		// Wire OnRehydration callback into rehydrator.
		reh.OnRehydration = func() {
			session.RecordRehydration()
		}

		w := wrapper.NewWrapper(command, cmdArgs, san, reh)
		exitCode, err := w.Run()

		// Finalise audit session.
		session.End()
		if cfg.AuditEnabled {
			if werr := audit.WriteAudit(session); werr != nil {
				_, _ = fmt.Fprintf(os.Stderr, "saola: audit write failed: %v\n", werr)
			}
		}

		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "saola: %v\n", err)
		}
		os.Exit(exitCode)
		return nil // unreachable; satisfies RunE signature
	},
}

func init() {
	rootCmd.AddCommand(wrapCmd)
}
