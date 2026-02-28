package intune

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/secmon-lab/warren/pkg/utils/safe"
	"github.com/urfave/cli/v3"
)

// Action implements the Intune device information retrieval tool
// using Microsoft Graph API with OAuth 2.0 Client Credentials Flow.
type Action struct {
	tenantID      string
	clientID      string
	clientSecret  string
	baseURL       string
	tokenEndpoint string

	mu          sync.Mutex
	accessToken string
	tokenExpiry time.Time
}

var _ interfaces.Tool = &Action{}

func (x *Action) Name() string {
	return "intune"
}

func (x *Action) Helper() *cli.Command {
	return nil
}

func (x *Action) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "intune-tenant-id",
			Usage:       "Azure AD tenant ID",
			Destination: &x.tenantID,
			Category:    "Tool",
			Sources:     cli.EnvVars("WARREN_INTUNE_TENANT_ID"),
		},
		&cli.StringFlag{
			Name:        "intune-client-id",
			Usage:       "Azure AD application (client) ID",
			Destination: &x.clientID,
			Category:    "Tool",
			Sources:     cli.EnvVars("WARREN_INTUNE_CLIENT_ID"),
		},
		&cli.StringFlag{
			Name:        "intune-client-secret",
			Usage:       "Azure AD client secret",
			Destination: &x.clientSecret,
			Category:    "Tool",
			Sources:     cli.EnvVars("WARREN_INTUNE_CLIENT_SECRET"),
		},
		&cli.StringFlag{
			Name:        "intune-base-url",
			Usage:       "Microsoft Graph API base URL",
			Destination: &x.baseURL,
			Category:    "Tool",
			Value:       "https://graph.microsoft.com/v1.0",
			Sources:     cli.EnvVars("WARREN_INTUNE_BASE_URL"),
		},
	}
}

func (x *Action) Configure(ctx context.Context) error {
	if x.tenantID == "" || x.clientID == "" || x.clientSecret == "" {
		return errutil.ErrActionUnavailable
	}
	x.tokenEndpoint = fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", x.tenantID)
	if x.baseURL == "" {
		x.baseURL = "https://graph.microsoft.com/v1.0"
	}
	return nil
}

func (x *Action) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("tenant_id", x.tenantID),
		slog.Int("client_id.len", len(x.clientID)),
		slog.Int("client_secret.len", len(x.clientSecret)),
		slog.String("base_url", x.baseURL),
	)
}

func (x *Action) Prompt(ctx context.Context) (string, error) {
	return "", nil
}

func (x *Action) Specs(ctx context.Context) ([]gollem.ToolSpec, error) {
	return []gollem.ToolSpec{
		{
			Name:        "intune_devices_by_user",
			Description: "Search Intune managed devices by user's email address or UPN (User Principal Name). Returns device details including compliance state, OS, encryption, and recent sign-in IP history.",
			Parameters: map[string]*gollem.Parameter{
				"user_principal_name": {
					Type:        gollem.TypeString,
					Description: "User's email address or UPN",
				},
			},
		},
		{
			Name:        "intune_devices_by_hostname",
			Description: "Search Intune managed device by device hostname. Returns device details including compliance state, OS, encryption, owner information, and recent sign-in IP history.",
			Parameters: map[string]*gollem.Parameter{
				"device_name": {
					Type:        gollem.TypeString,
					Description: "Device hostname to search",
				},
			},
		},
	}, nil
}

func (x *Action) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	switch name {
	case "intune_devices_by_user":
		upn, ok := args["user_principal_name"].(string)
		if !ok {
			return nil, goerr.New("user_principal_name parameter is required")
		}
		return x.searchDevicesByUser(ctx, upn)

	case "intune_devices_by_hostname":
		deviceName, ok := args["device_name"].(string)
		if !ok {
			return nil, goerr.New("device_name parameter is required")
		}
		return x.searchDevicesByHostname(ctx, deviceName)

	default:
		return nil, goerr.New("invalid function name", goerr.V("name", name))
	}
}

func (x *Action) searchDevicesByUser(ctx context.Context, upn string) (map[string]any, error) {
	filter := fmt.Sprintf("userPrincipalName eq '%s'", upn)
	devices, err := x.queryManagedDevices(ctx, filter)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to query managed devices by user",
			goerr.V("upn", upn))
	}

	signIns := x.fetchSignInLogs(ctx, fmt.Sprintf("userPrincipalName eq '%s'", upn))

	return map[string]any{
		"devices":       devices,
		"signInHistory": signIns,
		"totalDevices":  len(devices),
	}, nil
}

func (x *Action) searchDevicesByHostname(ctx context.Context, deviceName string) (map[string]any, error) {
	filter := fmt.Sprintf("deviceName eq '%s'", deviceName)
	devices, err := x.queryManagedDevices(ctx, filter)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to query managed devices by hostname",
			goerr.V("device_name", deviceName))
	}

	var signIns []any
	if len(devices) > 0 {
		if first, ok := devices[0].(map[string]any); ok {
			if upn, ok := first["userPrincipalName"].(string); ok && upn != "" {
				signIns = x.fetchSignInLogs(ctx, fmt.Sprintf("userPrincipalName eq '%s'", upn))
			}
		}
	}

	return map[string]any{
		"devices":       devices,
		"signInHistory": signIns,
		"totalDevices":  len(devices),
	}, nil
}

// queryManagedDevices queries the Microsoft Graph API for managed devices with the given filter.
// It handles 401 responses by clearing the token cache and retrying once.
func (x *Action) queryManagedDevices(ctx context.Context, filter string) ([]any, error) {
	params := url.Values{"$filter": {filter}}
	endpoint := fmt.Sprintf("%s/deviceManagement/managedDevices?%s", x.baseURL, params.Encode())

	body, err := x.callGraphAPI(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, goerr.Wrap(err, "failed to unmarshal managed devices response")
	}

	values, ok := result["value"].([]any)
	if !ok {
		return []any{}, nil
	}

	return values, nil
}

// fetchSignInLogs retrieves sign-in logs from Azure AD. This is optional;
// failures are logged but do not cause the overall request to fail.
func (x *Action) fetchSignInLogs(ctx context.Context, filter string) []any {
	params := url.Values{
		"$filter":  {filter},
		"$top":     {"50"},
		"$orderby": {"createdDateTime desc"},
		"$select":  {"ipAddress,createdDateTime,clientAppUsed,deviceDetail"},
	}
	endpoint := fmt.Sprintf("%s/auditLogs/signIns?%s", x.baseURL, params.Encode())

	body, err := x.callGraphAPI(ctx, endpoint)
	if err != nil {
		slog.WarnContext(ctx, "failed to fetch sign-in logs (optional)",
			slog.Any("error", err))
		return nil
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		slog.WarnContext(ctx, "failed to unmarshal sign-in logs",
			slog.Any("error", err))
		return nil
	}

	values, ok := result["value"].([]any)
	if !ok {
		return nil
	}

	return values
}

// callGraphAPI makes an authenticated GET request to the Graph API.
// On 401 responses, it clears the cached token and retries once.
func (x *Action) callGraphAPI(ctx context.Context, endpoint string) ([]byte, error) {
	body, statusCode, err := x.doGraphRequest(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	if statusCode == http.StatusUnauthorized {
		x.clearToken()
		body, statusCode, err = x.doGraphRequest(ctx, endpoint)
		if err != nil {
			return nil, err
		}
	}

	if statusCode != http.StatusOK {
		return nil, goerr.New("Graph API request failed",
			goerr.V("status_code", statusCode),
			goerr.V("body", string(body)),
			goerr.V("endpoint", endpoint))
	}

	return body, nil
}

func (x *Action) doGraphRequest(ctx context.Context, endpoint string) ([]byte, int, error) {
	token, err := x.getToken(ctx)
	if err != nil {
		return nil, 0, goerr.Wrap(err, "failed to get access token")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, 0, goerr.Wrap(err, "failed to create request")
	}
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, goerr.Wrap(err, "failed to send request")
	}
	defer safe.Close(ctx, resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, goerr.Wrap(err, "failed to read response body")
	}

	return body, resp.StatusCode, nil
}

// getToken returns a cached access token or fetches a new one if expired.
func (x *Action) getToken(ctx context.Context) (string, error) {
	x.mu.Lock()
	defer x.mu.Unlock()

	if x.accessToken != "" && time.Now().Before(x.tokenExpiry.Add(-5*time.Minute)) {
		return x.accessToken, nil
	}

	return x.fetchToken(ctx)
}

// fetchToken requests a new access token using the Client Credentials Flow.
// Must be called with x.mu held.
func (x *Action) fetchToken(ctx context.Context) (string, error) {
	data := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {x.clientID},
		"client_secret": {x.clientSecret},
		"scope":         {"https://graph.microsoft.com/.default"},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", x.tokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return "", goerr.Wrap(err, "failed to create token request")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", goerr.Wrap(err, "failed to send token request")
	}
	defer safe.Close(ctx, resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", goerr.Wrap(err, "failed to read token response")
	}

	if resp.StatusCode != http.StatusOK {
		slog.WarnContext(ctx, "token request failed",
			slog.Int("status_code", resp.StatusCode),
			slog.String("body", string(body)))
		return "", goerr.New("failed to obtain access token",
			goerr.V("status_code", resp.StatusCode),
			goerr.V("body", string(body)))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", goerr.Wrap(err, "failed to unmarshal token response")
	}

	x.accessToken = tokenResp.AccessToken
	x.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	return x.accessToken, nil
}

// clearToken clears the cached access token for retry on 401.
func (x *Action) clearToken() {
	x.mu.Lock()
	defer x.mu.Unlock()
	x.accessToken = ""
	x.tokenExpiry = time.Time{}
}
