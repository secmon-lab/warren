package falcon

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/safe"
)

// internalTool implements gollem.ToolSet for CrowdStrike Falcon API operations.
type internalTool struct {
	tokenProvider *tokenProvider
	baseURL       string
	httpClient    *http.Client
}

// newInternalTool creates a new internalTool for Falcon API calls.
func newInternalTool(tp *tokenProvider, baseURL string) *internalTool {
	return &internalTool{
		tokenProvider: tp,
		baseURL:       baseURL,
		httpClient:    &http.Client{Timeout: 60 * time.Second},
	}
}

func (t *internalTool) Specs(_ context.Context) ([]gollem.ToolSpec, error) {
	return []gollem.ToolSpec{
		{
			Name:        "falcon_search_incidents",
			Description: "Search for incident IDs using FQL (Falcon Query Language) filters. Returns a list of incident IDs that can be used with falcon_get_incidents to retrieve full details.",
			Parameters: map[string]*gollem.Parameter{
				"filter": {
					Type:        gollem.TypeString,
					Description: "FQL filter expression (e.g., \"status:'30'\", \"tags:'critical'\", \"start:>'2025-01-01'\")",
				},
				"sort": {
					Type:        gollem.TypeString,
					Description: "Sort expression (e.g., \"start.desc\", \"end.asc\")",
				},
				"limit": {
					Type:        gollem.TypeNumber,
					Description: "Maximum number of IDs to return (default: 100, max: 500)",
				},
				"offset": {
					Type:        gollem.TypeNumber,
					Description: "Pagination offset",
				},
			},
		},
		{
			Name:        "falcon_get_incidents",
			Description: "Get detailed information for specific incidents by their IDs. Returns full incident details including status, tactics, techniques, hosts, and users involved.",
			Parameters: map[string]*gollem.Parameter{
				"ids": {
					Type:        gollem.TypeString,
					Description: "Comma-separated incident IDs (e.g., \"inc:abc123:def456,inc:abc123:ghi789\")",
					Required:    true,
				},
			},
		},
		{
			Name:        "falcon_search_alerts",
			Description: "Search and retrieve alert details in one call using FQL filters with cursor-based pagination. Returns full alert objects including severity, tactic, technique, and device info.",
			Parameters: map[string]*gollem.Parameter{
				"filter": {
					Type:        gollem.TypeString,
					Description: "FQL filter expression (e.g., \"status:'new'\", \"severity:>50\", \"tactics:'Lateral Movement'\")",
				},
				"sort": {
					Type:        gollem.TypeString,
					Description: "Sort property (e.g., \"timestamp|desc\", \"severity|asc\")",
				},
				"limit": {
					Type:        gollem.TypeNumber,
					Description: "Maximum number of alerts to return (default: 100, max: 1000)",
				},
				"after": {
					Type:        gollem.TypeString,
					Description: "Cursor pagination token from a previous response for fetching next page",
				},
			},
		},
		{
			Name:        "falcon_get_alerts",
			Description: "Get detailed alert information by composite IDs. Use this when you already have specific alert IDs.",
			Parameters: map[string]*gollem.Parameter{
				"composite_ids": {
					Type:        gollem.TypeString,
					Description: "Comma-separated composite alert IDs",
					Required:    true,
				},
			},
		},
		{
			Name:        "falcon_search_behaviors",
			Description: "Search for behavior IDs using FQL filters. Returns behavior IDs that can be used with falcon_get_behaviors for full details.",
			Parameters: map[string]*gollem.Parameter{
				"filter": {
					Type:        gollem.TypeString,
					Description: "FQL filter expression",
				},
				"limit": {
					Type:        gollem.TypeNumber,
					Description: "Maximum number of IDs to return (default: 100, max: 500)",
				},
				"offset": {
					Type:        gollem.TypeNumber,
					Description: "Pagination offset",
				},
			},
		},
		{
			Name:        "falcon_get_behaviors",
			Description: "Get detailed behavior information by IDs. Returns behavior details including tactic, technique, severity, pattern, and associated device info.",
			Parameters: map[string]*gollem.Parameter{
				"ids": {
					Type:        gollem.TypeString,
					Description: "Comma-separated behavior IDs",
					Required:    true,
				},
			},
		},
		{
			Name:        "falcon_get_crowdscores",
			Description: "Get CrowdScore values for the environment. CrowdScore is an overall threat level indicator.",
			Parameters: map[string]*gollem.Parameter{
				"filter": {
					Type:        gollem.TypeString,
					Description: "FQL filter expression (e.g., \"timestamp:>'2025-01-01'\")",
				},
			},
		},
	}, nil
}

func (t *internalTool) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	switch name {
	case "falcon_search_incidents":
		return t.searchIncidents(ctx, args)
	case "falcon_get_incidents":
		return t.getIncidents(ctx, args)
	case "falcon_search_alerts":
		return t.searchAlerts(ctx, args)
	case "falcon_get_alerts":
		return t.getAlerts(ctx, args)
	case "falcon_search_behaviors":
		return t.searchBehaviors(ctx, args)
	case "falcon_get_behaviors":
		return t.getBehaviors(ctx, args)
	case "falcon_get_crowdscores":
		return t.getCrowdScores(ctx, args)
	default:
		return nil, goerr.New("unknown tool name", goerr.V("name", name))
	}
}

// doRequest performs an authenticated HTTP request to the CrowdStrike API.
// On 401 responses, it clears the token and retries once.
func (t *internalTool) doRequest(ctx context.Context, method, path string, body any) (map[string]any, error) {
	log := logging.From(ctx)
	log.Debug("Falcon API request", "method", method, "path", path)

	result, err := t.doRequestOnce(ctx, method, path, body)
	if err == nil {
		return result, nil
	}

	// Check if this is an authentication error and retry once
	var apiErr *apiError
	if errors.As(err, &apiErr) && apiErr.statusCode == http.StatusUnauthorized {
		log.Debug("Received 401, clearing token and retrying")
		t.tokenProvider.clearToken()
		return t.doRequestOnce(ctx, method, path, body)
	}

	return nil, err
}

// apiError represents an HTTP error from the CrowdStrike API.
type apiError struct {
	statusCode int
	body       string
}

func (e *apiError) Error() string {
	return fmt.Sprintf("Falcon API error: status=%d", e.statusCode)
}

// doRequestOnce performs a single authenticated HTTP request.
func (t *internalTool) doRequestOnce(ctx context.Context, method, path string, body any) (map[string]any, error) {
	token, err := t.tokenProvider.getToken(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get auth token")
	}

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to marshal request body")
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, t.baseURL+path, reqBody)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create request")
	}

	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to send request", goerr.V("path", path))
	}
	defer safe.Close(ctx, resp.Body)

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to read response body")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, goerr.Wrap(&apiError{statusCode: resp.StatusCode, body: string(respBody)},
			"Falcon API request failed",
			goerr.V("status", resp.StatusCode),
			goerr.V("path", path),
			goerr.V("body", string(respBody)),
		)
	}

	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, goerr.Wrap(err, "failed to parse response JSON", goerr.V("body", string(respBody)))
	}

	return result, nil
}

// searchIncidents searches for incident IDs using FQL filters.
func (t *internalTool) searchIncidents(ctx context.Context, args map[string]any) (map[string]any, error) {
	path := "/incidents/queries/incidents/v1"
	params := buildQueryParams(args, "filter", "sort", "limit", "offset")
	if params != "" {
		path += "?" + params
	}
	return t.doRequest(ctx, http.MethodGet, path, nil)
}

// getIncidents retrieves incident details by IDs.
func (t *internalTool) getIncidents(ctx context.Context, args map[string]any) (map[string]any, error) {
	ids, ok := args["ids"].(string)
	if !ok || ids == "" {
		return nil, goerr.New("ids is required")
	}

	body := map[string]any{
		"ids": splitAndTrim(ids),
	}
	return t.doRequest(ctx, http.MethodPost, "/incidents/entities/incidents/GET/v1", body)
}

// searchAlerts searches and retrieves alert details using FQL filters.
func (t *internalTool) searchAlerts(ctx context.Context, args map[string]any) (map[string]any, error) {
	body := make(map[string]any)
	if filter, ok := args["filter"].(string); ok && filter != "" {
		body["filter"] = filter
	}
	if sort, ok := args["sort"].(string); ok && sort != "" {
		body["sort"] = sort
	}
	if limit, ok := args["limit"].(float64); ok {
		body["limit"] = int(limit)
	}
	if after, ok := args["after"].(string); ok && after != "" {
		body["after"] = after
	}
	return t.doRequest(ctx, http.MethodPost, "/alerts/combined/alerts/v1", body)
}

// getAlerts retrieves alert details by composite IDs.
func (t *internalTool) getAlerts(ctx context.Context, args map[string]any) (map[string]any, error) {
	compositeIDs, ok := args["composite_ids"].(string)
	if !ok || compositeIDs == "" {
		return nil, goerr.New("composite_ids is required")
	}

	body := map[string]any{
		"composite_ids": splitAndTrim(compositeIDs),
	}
	return t.doRequest(ctx, http.MethodPost, "/alerts/entities/alerts/v2", body)
}

// searchBehaviors searches for behavior IDs using FQL filters.
func (t *internalTool) searchBehaviors(ctx context.Context, args map[string]any) (map[string]any, error) {
	path := "/incidents/queries/behaviors/v1"
	params := buildQueryParams(args, "filter", "limit", "offset")
	if params != "" {
		path += "?" + params
	}
	return t.doRequest(ctx, http.MethodGet, path, nil)
}

// getBehaviors retrieves behavior details by IDs.
func (t *internalTool) getBehaviors(ctx context.Context, args map[string]any) (map[string]any, error) {
	ids, ok := args["ids"].(string)
	if !ok || ids == "" {
		return nil, goerr.New("ids is required")
	}

	body := map[string]any{
		"ids": splitAndTrim(ids),
	}
	return t.doRequest(ctx, http.MethodPost, "/incidents/entities/behaviors/GET/v1", body)
}

// getCrowdScores retrieves CrowdScore values.
func (t *internalTool) getCrowdScores(ctx context.Context, args map[string]any) (map[string]any, error) {
	path := "/incidents/combined/crowdscores/v1"
	params := buildQueryParams(args, "filter")
	if params != "" {
		path += "?" + params
	}
	return t.doRequest(ctx, http.MethodGet, path, nil)
}

// buildQueryParams constructs URL query parameters from tool arguments.
func buildQueryParams(args map[string]any, keys ...string) string {
	var parts []string
	for _, key := range keys {
		if val, ok := args[key]; ok {
			switch v := val.(type) {
			case string:
				if v != "" {
					parts = append(parts, fmt.Sprintf("%s=%s", key, v))
				}
			case float64:
				parts = append(parts, fmt.Sprintf("%s=%d", key, int(v)))
			}
		}
	}
	return strings.Join(parts, "&")
}

// splitAndTrim splits a comma-separated string and trims whitespace from each element.
func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
