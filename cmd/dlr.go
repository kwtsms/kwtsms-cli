// dlr.go - fetch per-number delivery reports for a sent message.
// Endpoint: POST /API/dlr/
// Related files: internal/api/client.go, internal/output/output.go
// 2026-11-16 01:34 +03: new command, task int-cli-review.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kwtsms/kwtsms-cli/internal/api"
	"github.com/kwtsms/kwtsms-cli/internal/output"
	"github.com/kwtsms/kwtsms-cli/internal/sanitize"
)

var dlrCmd = &cobra.Command{
	Use:   "dlr <msg-id>",
	Short: "Show per-number delivery reports for a message",
	Long: `Fetch delivery reports for a message by its msg-id (from the send output).
Available for international numbers only: Kuwait numbers do not return
delivery reports. Wait at least 5 minutes after sending, and do not poll in
bulk without a delay or your IP may be blocked.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		msgID, err := sanitize.SanitizeMsgID(args[0])
		if err != nil {
			return fmt.Errorf("invalid msg-id: %w", err)
		}
		resp, err := api.GetDLR(Username, Password, msgID)
		if err != nil {
			return err
		}
		if jsonFlag {
			return output.PrintJSON(resp)
		}
		output.PrintDLR(resp)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(dlrCmd)
}
