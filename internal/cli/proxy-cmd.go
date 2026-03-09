package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"syscall"

	"github.com/nguyennghia/saola-proxy/internal/audit"
	"github.com/nguyennghia/saola-proxy/internal/config"
	"github.com/nguyennghia/saola-proxy/internal/proxy"
	"github.com/nguyennghia/saola-proxy/internal/sanitizer"
	"github.com/nguyennghia/saola-proxy/internal/scanner"
	"github.com/spf13/cobra"
)

var proxyPort int

var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Start HTTPS proxy server for PII sanitization",
	Long: `Starts a local HTTPS proxy that intercepts API calls to ai services and sanitizes PII.

The proxy performs MITM on HTTPS connections to api.anthropic.com, sanitizing
PII in request bodies before forwarding and rehydrating placeholders in
responses before returning to the client.

All other traffic is passed through transparently.

Usage:
  saola proxy [--port 8080]

Then run your AI tool with the proxy:
  HTTPS_PROXY=http://localhost:8080 claude`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load config.
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

		for _, name := range cfg.Patterns.Disabled {
			reg.Disable(name)
		}

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

		table := sanitizer.NewMappingTable()
		san := sanitizer.NewSanitizer(sc, table)
		reh := sanitizer.NewRehydrator(table)

		// Create audit session for proxy lifetime.
		session := audit.NewSession("proxy")

		san.OnDetection = func(patternName string) {
			session.RecordDetection(patternName)
		}
		reh.OnRehydration = func() {
			session.RecordRehydration()
		}

		// Load CA certificate for MITM.
		if !proxy.CAExists() {
			return fmt.Errorf("CA certificate not found. Run 'saola setup-ca' first")
		}
		ca, err := proxy.LoadCA()
		if err != nil {
			return fmt.Errorf("load CA: %w", err)
		}

		addr := fmt.Sprintf(":%d", proxyPort)
		ps := proxy.NewProxyServer(addr, san, reh, ca, reg, table, session)

		fmt.Fprintf(os.Stdout, "Saola proxy listening on :%d\n", proxyPort)
		fmt.Fprintf(os.Stdout, "Dashboard: http://localhost:%d\n", proxyPort)
		fmt.Fprintf(os.Stdout, "Usage:     HTTPS_PROXY=http://localhost:%d claude\n", proxyPort)

		// Graceful shutdown on SIGINT/SIGTERM.
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		errCh := make(chan error, 1)
		go func() {
			errCh <- ps.Start()
		}()

		select {
		case <-ctx.Done():
			fmt.Fprintln(os.Stdout, "\nsaola proxy: shutting down")
		case err := <-errCh:
			if err != nil {
				return fmt.Errorf("proxy: %w", err)
			}
		}

		session.End()
		if cfg.AuditEnabled {
			if werr := audit.WriteAudit(session); werr != nil {
				_, _ = fmt.Fprintf(os.Stderr, "saola: audit write failed: %v\n", werr)
			}
		}

		return nil
	},
}

func init() {
	proxyCmd.Flags().IntVar(&proxyPort, "port", 8080, "Port to listen on")
	rootCmd.AddCommand(proxyCmd)
}
