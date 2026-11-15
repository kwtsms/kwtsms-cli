// status.go - look up the queue status of a sent message.
// Endpoint: POST /API/status/
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

var statusCmd = &cobra.Command{
	Use:   "status <msg-id>",
	Short: "Show the queue status of a sent message",
	Long: `Look up the queue status of a message by its msg-id (from the send output).
Shows whether the message is pending, stuck with an error in the queue, or
sent to the gateway. This is not a delivery report: use "dlr" for that.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		msgID, err := sanitize.SanitizeMsgID(args[0])
		if err != nil {
			return fmt.Errorf("invalid msg-id: %w", err)
		}
		resp, err := api.GetStatus(Username, Password, msgID)
		if err != nil {
			return err
		}
		if jsonFlag {
			return output.PrintJSON(resp)
		}
		output.PrintStatus(resp)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
