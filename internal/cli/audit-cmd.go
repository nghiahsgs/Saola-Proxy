package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/nguyennghia/saola-proxy/internal/audit"
	"github.com/spf13/cobra"
)

var auditSessionID string

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Show sanitization audit logs",
	Long: `audit displays a table of recent Saola Proxy sessions.

Use --session <ID> to print the full JSON record for a specific session.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if auditSessionID != "" {
			return showSession(cmd, auditSessionID)
		}
		return listSessions(cmd)
	},
}

func listSessions(cmd *cobra.Command) error {
	sessions, err := audit.ListSessions(10)
	if err != nil {
		return fmt.Errorf("reading audit log: %w", err)
	}

	if len(sessions) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No audit sessions found.")
		return nil
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "DATE\tCOMMAND\tDURATION\tDETECTIONS")
	for _, s := range sessions {
		date := s.StartTime.Format(time.DateTime)
		dur := fmt.Sprintf("%dms", s.DurationMS)
		detTotal := 0
		for _, n := range s.Detections {
			detTotal += n
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\n", date, s.Command, dur, detTotal)
	}
	return w.Flush()
}

func showSession(cmd *cobra.Command, id string) error {
	sessions, err := audit.ListSessions(0)
	if err != nil {
		return fmt.Errorf("reading audit log: %w", err)
	}

	for _, s := range sessions {
		if s.ID == id {
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(s)
		}
	}

	fmt.Fprintf(os.Stderr, "session %q not found\n", id)
	return nil
}

func init() {
	auditCmd.Flags().StringVar(&auditSessionID, "session", "", "Show detailed JSON for a specific session ID")
	rootCmd.AddCommand(auditCmd)
}
