// Package api provides HTTP client functions for the kwtSMS REST/JSON API.
// All requests use POST with Content-Type: application/json.
// Credentials are passed in the JSON body, never in headers or query params.
// API base URL: https://www.kwtsms.com/API/
// Related files: cmd/balance.go, cmd/send.go, cmd/senderid.go, cmd/coverage.go
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// baseURL is a var so tests can override it with an httptest.Server URL.
var baseURL = "https://www.kwtsms.com/API/"

const (
	requestTimeout = 30 * time.Second
	dialTimeout    = 10 * time.Second
	tlsTimeout     = 10 * time.Second
)

// UserAgent is set at build time via ldflags: -X github.com/kwtsms/kwtsms-cli/internal/api.UserAgent=...
// Defaults to "kwtsms-cli/dev" if not set.
var UserAgent = "kwtsms-cli/dev"

// httpClient is the package-level HTTP client with explicit dial and TLS timeouts.
// TLS certificate validation is always enabled (no InsecureSkipVerify).
var httpClient = &http.Client{
	Timeout: requestTimeout,
	Transport: &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   dialTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: tlsTimeout,
	},
}

// APIError holds the error fields returned by the kwtSMS API when result == "ERROR".
type APIError struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}

// BalanceResponse is returned by POST /API/balance/
type BalanceResponse struct {
	Result    string `json:"result"`
	Available int64  `json:"available"`
	Purchased int64  `json:"purchased"`
	APIError
}

// SenderIDResponse is returned by POST /API/senderid/
type SenderIDResponse struct {
	Result   string   `json:"result"`
	SenderID []string `json:"senderid"`
	APIError
}

// SendResponse is returned by POST /API/send/
type SendResponse struct {
	Result        string `json:"result"`
	MsgID         string `json:"msg-id"`
	Numbers       int64  `json:"numbers"`
	PointsCharged int64  `json:"points-charged"`
	BalanceAfter  int64  `json:"balance-after"`
	APIError
}

// networkError converts a raw Go HTTP/network error into a clean user-facing message.
// It never exposes internal URLs, IP addresses, or DNS server details.
func networkError(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()

	// Timeout (30s limit exceeded)
	if errors.Is(err, fmt.Errorf("context deadline exceeded")) ||
		strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "deadline exceeded") {
		return fmt.Errorf("connection timed out. Check your internet connection and try again")
	}

	// DNS / host not found
	var dnsErr *url.Error
	if errors.As(err, &dnsErr) {
		inner := dnsErr.Err.Error()
		if strings.Contains(inner, "no such host") ||
			strings.Contains(inner, "lookup") ||
			strings.Contains(inner, "DNS") {
			return fmt.Errorf("cannot reach kwtsms.com. Check your internet connection")
		}
		if strings.Contains(inner, "connection refused") {
			return fmt.Errorf("connection refused by server. Try again later")
		}
		if strings.Contains(inner, "network is unreachable") ||
			strings.Contains(inner, "no route to host") {
			return fmt.Errorf("no internet connection. Check your network and try again")
		}
		if strings.Contains(inner, "timeout") || strings.Contains(inner, "deadline") {
			return fmt.Errorf("connection timed out. Check your internet connection and try again")
		}
	}

	// Generic fallback - no internal details exposed
	return fmt.Errorf("network error. Check your internet connection and try again")
}

// post sends a JSON POST request to the given kwtSMS endpoint and returns the
// raw response body. It sets Content-Type and Accept headers to application/json.
// The request body is never logged to protect credentials and phone numbers.
func post(endpoint string, body interface{}) ([]byte, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", UserAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, networkError(err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response from %s: %w", endpoint, err)
	}

	if resp.StatusCode != http.StatusOK {
		// 2026-11-16 01:34 +03: the API returns HTTP 4xx with a JSON error body (ERR019, ERR020, ERR029 ...);
		// keep the body when it is a well-formed API error so callers show the ERR code (task int-cli-review).
		var probe struct {
			Result string `json:"result"`
			Code   string `json:"code"`
		}
		if json.Unmarshal(data, &probe) == nil && probe.Result == "ERROR" && probe.Code != "" {
			return data, nil
		}
		return nil, fmt.Errorf("unexpected HTTP status %d from %s", resp.StatusCode, endpoint)
	}

	return data, nil
}

// apiError returns a formatted error from a non-OK API response.
func apiError(code, description string) error {
	if description == "" {
		description = "unknown error"
	}
	return fmt.Errorf("[%s] %s", code, description)
}

// GetBalance calls POST /API/balance/ and returns the account balance.
func GetBalance(username, password string) (*BalanceResponse, error) {
	body := struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{username, password}

	data, err := post("balance/", body)
	if err != nil {
		return nil, err
	}

	var resp BalanceResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse balance response: %w", err)
	}
	if resp.Result == "ERROR" {
		return nil, apiError(resp.Code, resp.Description)
	}
	return &resp, nil
}

// GetSenderID calls POST /API/senderid/ and returns available sender IDs.
func GetSenderID(username, password string) (*SenderIDResponse, error) {
	body := struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{username, password}

	data, err := post("senderid/", body)
	if err != nil {
		return nil, err
	}

	var resp SenderIDResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse senderid response: %w", err)
	}
	if resp.Result == "ERROR" {
		return nil, apiError(resp.Code, resp.Description)
	}
	return &resp, nil
}

// GetCoverage calls POST /API/coverage/ and returns raw JSON (dynamic structure).
func GetCoverage(username, password string) (json.RawMessage, error) {
	body := struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{username, password}

	data, err := post("coverage/", body)
	if err != nil {
		return nil, err
	}

	// Check for API error by peeking at the result field
	var peek struct {
		Result      string `json:"result"`
		Code        string `json:"code"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(data, &peek); err != nil {
		return nil, fmt.Errorf("failed to parse coverage response: %w", err)
	}
	if peek.Result == "ERROR" {
		return nil, apiError(peek.Code, peek.Description)
	}
	return json.RawMessage(data), nil
}

// SendSMS calls POST /API/send/ to send an SMS message.
// numbers must be a comma-separated string of phone numbers (digits only, no +/00 prefix).
// Set test=true to queue without delivery (development mode).
func SendSMS(username, password, sender, numbers, message string, test bool) (*SendResponse, error) {
	testVal := "0"
	if test {
		testVal = "1"
	}

	body := struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Sender   string `json:"sender"`
		Mobile   string `json:"mobile"`
		Message  string `json:"message"`
		Test     string `json:"test"`
	}{username, password, sender, numbers, message, testVal}

	data, err := post("send/", body)
	if err != nil {
		return nil, err
	}

	var resp SendResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse send response: %w", err)
	}
	if resp.Result == "ERROR" {
		return nil, apiError(resp.Code, resp.Description)
	}
	return &resp, nil
}

// 2026-11-16 01:34 +03: ValidateResponse is returned by POST /API/validate/: three buckets OK, ER (format error), NR (no route) (task int-cli-review).
type ValidateResponse struct {
	Result string `json:"result"`
	Mobile struct {
		OK []string `json:"OK"`
		ER []string `json:"ER"`
		NR []string `json:"NR"`
	} `json:"mobile"`
	APIError
}

// 2026-11-16 01:34 +03: StatusResponse is returned by POST /API/status/ (task int-cli-review).
type StatusResponse struct {
	Result      string `json:"result"`
	Status      string `json:"status"`
	Description string `json:"description"`
	Code        string `json:"code,omitempty"`
}

// 2026-11-16 01:34 +03: DLRReport is one per-number delivery report entry (task int-cli-review).
type DLRReport struct {
	Number string `json:"Number"`
	Status string `json:"Status"`
}

// 2026-11-16 01:34 +03: DLRResponse is returned by POST /API/dlr/ (task int-cli-review).
type DLRResponse struct {
	Result string      `json:"result"`
	Report []DLRReport `json:"report"`
	APIError
}

// 2026-11-16 01:34 +03: ValidateNumbers calls POST /API/validate/ (max 2500 comma-separated numbers) (task int-cli-review).
func ValidateNumbers(username, password, numbers string) (*ValidateResponse, error) {
	body := struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Mobile   string `json:"mobile"`
	}{username, password, numbers}

	data, err := post("validate/", body)
	if err != nil {
		return nil, err
	}
	var resp ValidateResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse validate response: %w", err)
	}
	if resp.Result == "ERROR" {
		return nil, apiError(resp.Code, resp.Description)
	}
	// Always render empty buckets as [] in --json output.
	if resp.Mobile.OK == nil {
		resp.Mobile.OK = []string{}
	}
	if resp.Mobile.ER == nil {
		resp.Mobile.ER = []string{}
	}
	if resp.Mobile.NR == nil {
		resp.Mobile.NR = []string{}
	}
	return &resp, nil
}

// 2026-11-16 01:34 +03: GetStatus calls POST /API/status/ for the queue status of a sent message (task int-cli-review).
func GetStatus(username, password, msgID string) (*StatusResponse, error) {
	body := struct {
		Username string `json:"username"`
		Password string `json:"password"`
		MsgID    string `json:"msgid"`
	}{username, password, msgID}

	data, err := post("status/", body)
	if err != nil {
		return nil, err
	}
	var resp StatusResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse status response: %w", err)
	}
	if resp.Result == "ERROR" {
		return nil, apiError(resp.Code, resp.Description)
	}
	return &resp, nil
}

// 2026-11-16 01:34 +03: GetDLR calls POST /API/dlr/ for per-number delivery reports (task int-cli-review).
func GetDLR(username, password, msgID string) (*DLRResponse, error) {
	body := struct {
		Username string `json:"username"`
		Password string `json:"password"`
		MsgID    string `json:"msgid"`
	}{username, password, msgID}

	data, err := post("dlr/", body)
	if err != nil {
		return nil, err
	}
	var resp DLRResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse dlr response: %w", err)
	}
	if resp.Result == "ERROR" {
		return nil, apiError(resp.Code, resp.Description)
	}
	if resp.Report == nil {
		resp.Report = []DLRReport{}
	}
	return &resp, nil
}
