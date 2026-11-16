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
		// 2026-11-16 16:20 +03: numbers that fail local cleaning go to the ER bucket; only the rest go to the API.
		// Before, one bad number failed the whole command (job 49, CLI 1.1.0 QA).
		phones, invalid := sanitize.SplitPhones(validateTo)
		if len(phones)+len(invalid) == 0 {
			return fmt.Errorf("no phone numbers provided")
		}
		if len(phones) > maxValidateNumbers {
			return fmt.Errorf("too many numbers: %d (max %d per call)", len(phones), maxValidateNumbers)
		}
		resp := &api.ValidateResponse{Result: "OK"}
		resp.Mobile.OK, resp.Mobile.NR = []string{}, []string{} // --json prints [] not null when no API call is made
		if len(phones) > 0 {
			var err error
			resp, err = api.ValidateNumbers(Username, Password, strings.Join(phones, ","))
			if err != nil {
				return err
			}
		}
		resp.Mobile.ER = append(resp.Mobile.ER, invalid...)
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
