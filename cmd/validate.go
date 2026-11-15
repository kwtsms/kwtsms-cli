// validate.go - check numbers for format and route before sending.
// Endpoint: POST /API/validate/
// Related files: internal/api/client.go, internal/output/output.go
// 2026-11-16 01:34 +03: new command, task int-cli-review.
package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kwtsms/kwtsms-cli/internal/api"
	"github.com/kwtsms/kwtsms-cli/internal/output"
	"github.com/kwtsms/kwtsms-cli/internal/sanitize"
)

// maxValidateNumbers is the API limit for the mobile parameter of /API/validate/.
const maxValidateNumbers = 2500

var validateTo string

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Check numbers for format and route before sending",
	Long: `Check phone numbers against the kwtSMS API without sending anything.

Numbers land in three buckets:
  OK  valid and routable
  ER  format error
  NR  no route (country not activated on your account)

Up to 2500 numbers per call. Example: kwtsms-cli validate -t 965XXXXXXXX,965YYYYYYYY`,
	RunE: func(cmd *cobra.Command, args []string) error {
		phones, err := sanitize.SanitizePhones(validateTo)
		if err != nil {
			return fmt.Errorf("invalid phone number(s): %w", err)
		}
		if len(phones) > maxValidateNumbers {
			return fmt.Errorf("too many numbers: %d (max %d per call)", len(phones), maxValidateNumbers)
		}
		resp, err := api.ValidateNumbers(Username, Password, strings.Join(phones, ","))
		if err != nil {
			return err
		}
		if jsonFlag {
			return output.PrintJSON(resp)
		}
		output.PrintValidate(resp)
		return nil
	},
}

func init() {
	validateCmd.Flags().StringVarP(&validateTo, "to", "t", "", "phone number(s) to validate, comma-separated (required)")
	_ = validateCmd.MarkFlagRequired("to")
	rootCmd.AddCommand(validateCmd)
}
