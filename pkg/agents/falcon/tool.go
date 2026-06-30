package falcon

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gollem-dev/gollem"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
	"github.com/secmon-lab/warren/pkg/utils/safe"
	"github.com/secmon-lab/warren/pkg/utils/toolset"
)

// internalTool implements gollem.ToolSet for CrowdStrike Falcon API operations.
type internalTool struct {
	tokenProvider *tokenProvider
	baseURL       string
	httpClient    *http.Client

	tools gollem.ToolSet
}

// newInternalTool creates a new internalTool for Falcon API calls.
func newInternalTool(tp *tokenProvider, baseURL string) *internalTool {
	t := &internalTool{
		tokenProvider: tp,
		baseURL:       baseURL,
		httpClient:    &http.Client{Timeout: 60 * time.Second},
	}

	// Build the type-safe tool set. Each tool's schema is inferred from its
	// typed input struct, replacing the hand-written ToolSpec literals and the
	// args[...] map extraction.
	t.tools = toolset.New(
		gollem.MustNewTool("falcon_search_incidents", descSearchIncidents, t.searchIncidents),
		gollem.MustNewTool("falcon_get_incidents", descGetIncidents, t.getIncidents),
		gollem.MustNewTool("falcon_search_alerts", descSearchAlerts, t.searchAlerts),
		gollem.MustNewTool("falcon_get_alerts", descGetAlerts, t.getAlerts),
		gollem.MustNewTool("falcon_search_behaviors", descSearchBehaviors, t.searchBehaviors),
		gollem.MustNewTool("falcon_get_behaviors", descGetBehaviors, t.getBehaviors),
		gollem.MustNewTool("falcon_get_crowdscores", descGetCrowdScores, t.getCrowdScores),
		gollem.MustNewTool("falcon_search_devices", descSearchDevices, t.searchDevices),
		gollem.MustNewTool("falcon_get_devices", descGetDevices, t.getDevices),
		gollem.MustNewTool("falcon_search_events", descSearchEvents, t.searchEvents),
	)

	return t
}

// Tool descriptions. Kept as constants so the typed-tool registration in
// newInternalTool stays readable and the wire-level descriptions are unchanged.
const (
	descSearchIncidents = "Search for incident IDs using FQL (Falcon Query Language) filters. Returns a list of incident IDs that can be used with falcon_get_incidents to retrieve full details."
	descGetIncidents    = "Get detailed information for specific incidents by their IDs. Returns full incident details including status, tactics, techniques, hosts, and users involved."
	descSearchAlerts    = "Search and retrieve alert details in one call using FQL filters with cursor-based pagination. Returns full alert objects including severity, tactic, technique, and device info."
	descGetAlerts       = "Get detailed alert information by composite IDs. Use this when you already have specific alert IDs."
	descSearchBehaviors = "Search for behavior IDs using FQL filters. Returns behavior IDs that can be used with falcon_get_behaviors for full details."
	descGetBehaviors    = "Get detailed behavior information by IDs. Returns behavior details including tactic, technique, severity, pattern, and associated device info."
	descGetCrowdScores  = "Get CrowdScore values for the environment. CrowdScore is an overall threat level indicator."
	descSearchDevices   = "Search for device (host) IDs using FQL filters. Returns a list of device IDs that can be used with falcon_get_devices to retrieve full host details including OS, IP addresses, sensor version, and containment status."
	descGetDevices      = "Get detailed device (host) information by device IDs. Returns full host details including hostname, OS, IP addresses, sensor version, tags, and containment status."
	descSearchEvents    = "Search EDR telemetry events using CrowdStrike Query Language (CQL). This uses the Next-Gen SIEM Search API to query raw event data (process executions, network connections, file writes, DNS requests, etc.). The search runs asynchronously and this tool automatically polls until results are ready."
)

// Typed inputs for each tool. The schema is inferred from these struct tags by
// gollem.NewTool. Pagination counts use float64 (not int64) so the inferred JSON
// schema stays "number" — matching the wire-level type the LLM was given before
// this migration.
type searchIncidentsInput struct {
	Filter string  `json:"filter" description:"FQL filter expression (e.g., \"status:'30'\", \"tags:'critical'\", \"start:>'2025-01-01'\")"`
	Sort   string  `json:"sort" description:"Sort expression (e.g., \"start.desc\", \"end.asc\")"`
	Limit  float64 `json:"limit" description:"Maximum number of IDs to return (default: 100, max: 500)"`
	Offset float64 `json:"offset" description:"Pagination offset"`
}

type getIncidentsInput struct {
	IDs string `json:"ids" required:"true" description:"Comma-separated incident IDs (e.g., \"inc:abc123:def456,inc:abc123:ghi789\")"`
}

type searchAlertsInput struct {
	Filter string  `json:"filter" description:"FQL filter expression (e.g., \"status:'new'\", \"severity:>50\", \"tactics:'Lateral Movement'\")"`
	Sort   string  `json:"sort" description:"Sort property (e.g., \"timestamp|desc\", \"severity|asc\")"`
	Limit  float64 `json:"limit" description:"Maximum number of alerts to return (default: 100, max: 1000)"`
	After  string  `json:"after" description:"Cursor pagination token from a previous response for fetching next page"`
}

type getAlertsInput struct {
	CompositeIDs string `json:"composite_ids" required:"true" description:"Comma-separated composite alert IDs"`
}

type searchBehaviorsInput struct {
	Filter string  `json:"filter" description:"FQL filter expression"`
	Limit  float64 `json:"limit" description:"Maximum number of IDs to return (default: 100, max: 500)"`
	Offset float64 `json:"offset" description:"Pagination offset"`
}

type getBehaviorsInput struct {
	IDs string `json:"ids" required:"true" description:"Comma-separated behavior IDs"`
}

type getCrowdScoresInput struct {
	Filter string `json:"filter" description:"FQL filter expression (e.g., \"timestamp:>'2025-01-01'\")"`
}

type searchDevicesInput struct {
	Filter string  `json:"filter" description:"FQL filter expression (e.g., \"hostname:'*web*'\", \"platform_name:'Windows'\", \"last_seen:>='2025-01-01'\", \"external_ip:'10.0.0.*'\")"`
	Sort   string  `json:"sort" description:"Sort expression (e.g., \"hostname.asc\", \"last_seen.desc\")"`
	Limit  float64 `json:"limit" description:"Maximum number of IDs to return (default: 100, max: 5000)"`
	Offset string  `json:"offset" description:"Pagination offset token from a previous response"`
}

type getDevicesInput struct {
	IDs string `json:"ids" required:"true" description:"Comma-separated device IDs (up to 5000, e.g., \"abc123def456,ghi789jkl012\")"`
}

type searchEventsInput struct {
	QueryString string `json:"query_string" required:"true" description:"CQL query string (e.g., \"aid=abc123\", \"#event_simpleName=ProcessRollup2 AND FileName=cmd.exe\", \"ComputerName=workstation1 | tail(100)\")"`
	Repository  string `json:"repository" description:"Repository to search. Values: \"search-all\" (all data, default), \"investigate_view\" (Falcon EDR), \"third-party\" (third-party data), \"falcon_for_it_view\" (IT Automation), \"forensics_view\" (Forensics)"`
	Start       string `json:"start" description:"Start time for the search (e.g., \"1d\" for last 1 day, \"24h\" for last 24 hours, \"2025-01-01T00:00:00Z\" for absolute time). Default: \"1d\""`
	End         string `json:"end" description:"End time for the search (e.g., \"now\", \"2025-01-02T00:00:00Z\"). Default: \"now\""`
}

func (t *internalTool) Specs(ctx context.Context) ([]gollem.ToolSpec, error) {
	return t.tools.Specs(ctx)
}

func (t *internalTool) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	return t.tools.Run(ctx, name, args)
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
		msg.Trace(ctx, "🔄 Received 401, refreshing token and retrying...")
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
	log := logging.From(ctx)
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
		log.Warn("Falcon API request failed",
			"status", resp.StatusCode,
			"path", path,
		)
		return nil, goerr.Wrap(&apiError{statusCode: resp.StatusCode, body: string(respBody)},
			"Falcon API request failed",
			goerr.V("status", resp.StatusCode),
			goerr.V("path", path),
		)
	}

	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, goerr.Wrap(err, "failed to parse response JSON", goerr.V("body", string(respBody)))
	}

	return result, nil
}

// searchIncidents searches for incident IDs using FQL filters.
func (t *internalTool) searchIncidents(ctx context.Context, in searchIncidentsInput) (map[string]any, error) {
	filter := in.Filter
	msg.Trace(ctx, "🔍 Searching incidents (filter: `%s`)", filter)

	path := "/incidents/queries/incidents/v1"
	params := queryParams(
		[2]string{"filter", filter},
		[2]string{"sort", in.Sort},
		[2]string{"limit", numParam(in.Limit)},
		[2]string{"offset", numParam(in.Offset)},
	)
	if params != "" {
		path += "?" + params
	}
	result, err := t.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		msg.Warn(ctx, "⚠️ *[Falcon]* Incident search failed (filter: `%s`): %v", filter, err)
		return nil, err
	}
	msg.Trace(ctx, "✅ Incident search completed")
	return result, nil
}

// getIncidents retrieves incident details by IDs.
func (t *internalTool) getIncidents(ctx context.Context, in getIncidentsInput) (map[string]any, error) {
	ids := in.IDs
	if ids == "" {
		return nil, goerr.New("ids is required")
	}

	msg.Trace(ctx, "📋 Retrieving incident details (ids: `%s`)", ids)
	body := map[string]any{
		"ids": splitAndTrim(ids),
	}
	result, err := t.doRequest(ctx, http.MethodPost, "/incidents/entities/incidents/GET/v1", body)
	if err != nil {
		msg.Warn(ctx, "⚠️ *[Falcon]* Failed to retrieve incidents (ids: `%s`): %v", ids, err)
		return nil, err
	}
	msg.Trace(ctx, "✅ Retrieved incident details")
	return result, nil
}

// searchAlerts searches and retrieves alert details using FQL filters.
func (t *internalTool) searchAlerts(ctx context.Context, in searchAlertsInput) (map[string]any, error) {
	filter := in.Filter
	msg.Trace(ctx, "🔍 Searching alerts (filter: `%s`)", filter)

	body := make(map[string]any)
	if filter != "" {
		body["filter"] = filter
	}
	if in.Sort != "" {
		body["sort"] = in.Sort
	}
	if in.Limit > 0 {
		body["limit"] = int(in.Limit)
	}
	if in.After != "" {
		body["after"] = in.After
	}
	result, err := t.doRequest(ctx, http.MethodPost, "/alerts/combined/alerts/v1", body)
	if err != nil {
		msg.Warn(ctx, "⚠️ *[Falcon]* Alert search failed (filter: `%s`): %v", filter, err)
		return nil, err
	}
	msg.Trace(ctx, "✅ Alert search completed")
	return result, nil
}

// getAlerts retrieves alert details by composite IDs.
func (t *internalTool) getAlerts(ctx context.Context, in getAlertsInput) (map[string]any, error) {
	compositeIDs := in.CompositeIDs
	if compositeIDs == "" {
		return nil, goerr.New("composite_ids is required")
	}

	msg.Trace(ctx, "📋 Retrieving alert details (ids: `%s`)", compositeIDs)
	body := map[string]any{
		"composite_ids": splitAndTrim(compositeIDs),
	}
	result, err := t.doRequest(ctx, http.MethodPost, "/alerts/entities/alerts/v2", body)
	if err != nil {
		msg.Warn(ctx, "⚠️ *[Falcon]* Failed to retrieve alerts (ids: `%s`): %v", compositeIDs, err)
		return nil, err
	}
	msg.Trace(ctx, "✅ Retrieved alert details")
	return result, nil
}

// searchBehaviors searches for behavior IDs using FQL filters.
func (t *internalTool) searchBehaviors(ctx context.Context, in searchBehaviorsInput) (map[string]any, error) {
	filter := in.Filter
	msg.Trace(ctx, "🔍 Searching behaviors (filter: `%s`)", filter)

	path := "/incidents/queries/behaviors/v1"
	params := queryParams(
		[2]string{"filter", filter},
		[2]string{"limit", numParam(in.Limit)},
		[2]string{"offset", numParam(in.Offset)},
	)
	if params != "" {
		path += "?" + params
	}
	result, err := t.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		msg.Warn(ctx, "⚠️ *[Falcon]* Behavior search failed (filter: `%s`): %v", filter, err)
		return nil, err
	}
	msg.Trace(ctx, "✅ Behavior search completed")
	return result, nil
}

// getBehaviors retrieves behavior details by IDs.
func (t *internalTool) getBehaviors(ctx context.Context, in getBehaviorsInput) (map[string]any, error) {
	ids := in.IDs
	if ids == "" {
		return nil, goerr.New("ids is required")
	}

	msg.Trace(ctx, "📋 Retrieving behavior details (ids: `%s`)", ids)
	body := map[string]any{
		"ids": splitAndTrim(ids),
	}
	result, err := t.doRequest(ctx, http.MethodPost, "/incidents/entities/behaviors/GET/v1", body)
	if err != nil {
		msg.Warn(ctx, "⚠️ *[Falcon]* Failed to retrieve behaviors (ids: `%s`): %v", ids, err)
		return nil, err
	}
	msg.Trace(ctx, "✅ Retrieved behavior details")
	return result, nil
}

// searchDevices searches for device (host) IDs using FQL filters.
func (t *internalTool) searchDevices(ctx context.Context, in searchDevicesInput) (map[string]any, error) {
	filter := in.Filter
	msg.Trace(ctx, "🔍 Searching devices (filter: `%s`)", filter)

	path := "/devices/queries/devices-scroll/v1"
	// Note: device offset is a scroll token (string), not a numeric offset.
	params := queryParams(
		[2]string{"filter", filter},
		[2]string{"sort", in.Sort},
		[2]string{"limit", numParam(in.Limit)},
		[2]string{"offset", in.Offset},
	)
	if params != "" {
		path += "?" + params
	}
	result, err := t.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		msg.Warn(ctx, "⚠️ *[Falcon]* Device search failed (filter: `%s`): %v", filter, err)
		return nil, err
	}
	msg.Trace(ctx, "✅ Device search completed")
	return result, nil
}

// getDevices retrieves device (host) details by IDs.
func (t *internalTool) getDevices(ctx context.Context, in getDevicesInput) (map[string]any, error) {
	ids := in.IDs
	if ids == "" {
		return nil, goerr.New("ids is required")
	}

	msg.Trace(ctx, "📋 Retrieving device details (ids: `%s`)", ids)
	body := map[string]any{
		"ids": splitAndTrim(ids),
	}
	result, err := t.doRequest(ctx, http.MethodPost, "/devices/entities/devices/v2", body)
	if err != nil {
		msg.Warn(ctx, "⚠️ *[Falcon]* Failed to retrieve devices (ids: `%s`): %v", ids, err)
		return nil, err
	}
	msg.Trace(ctx, "✅ Retrieved device details")
	return result, nil
}

// getCrowdScores retrieves CrowdScore values.
func (t *internalTool) getCrowdScores(ctx context.Context, in getCrowdScoresInput) (map[string]any, error) {
	filter := in.Filter
	msg.Trace(ctx, "📊 Retrieving CrowdScores (filter: `%s`)", filter)

	path := "/incidents/combined/crowdscores/v1"
	params := queryParams([2]string{"filter", filter})
	if params != "" {
		path += "?" + params
	}
	result, err := t.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		msg.Warn(ctx, "⚠️ *[Falcon]* CrowdScore retrieval failed (filter: `%s`): %v", filter, err)
		return nil, err
	}
	msg.Trace(ctx, "✅ CrowdScores retrieved")
	return result, nil
}

// searchEvents runs a CQL query via the Next-Gen SIEM Search API.
// It creates a query job and polls until the job completes, returning all events.
func (t *internalTool) searchEvents(ctx context.Context, in searchEventsInput) (map[string]any, error) {
	log := logging.From(ctx)

	queryString := in.QueryString
	if queryString == "" {
		return nil, goerr.New("query_string is required")
	}

	repository := "search-all"
	if in.Repository != "" {
		repository = in.Repository
	}

	body := map[string]any{
		"queryString": queryString,
	}
	if in.Start != "" {
		body["start"] = in.Start
	} else {
		body["start"] = "1d"
	}
	if in.End != "" {
		body["end"] = in.End
	} else {
		body["end"] = "now"
	}

	// Step 1: Create query job
	msg.Trace(ctx, "🔍 Searching events (query: `%s`, repo: `%s`)", queryString, repository)
	jobPath := fmt.Sprintf("/humio/api/v1/repositories/%s/queryjobs", repository)
	jobResp, err := t.doRequest(ctx, http.MethodPost, jobPath, body)
	if err != nil {
		msg.Warn(ctx, "⚠️ *[Falcon]* Failed to create event search job (query: `%s`): %v", queryString, err)
		return nil, goerr.Wrap(err, "failed to create event search query job",
			goerr.V("repository", repository),
			goerr.V("query", queryString),
		)
	}

	jobID, ok := jobResp["id"].(string)
	if !ok || jobID == "" {
		msg.Warn(ctx, "⚠️ *[Falcon]* No job ID returned from event search (query: `%s`)", queryString)
		return nil, goerr.New("no job ID returned from query job creation",
			goerr.V("response", jobResp),
		)
	}

	log.Debug("Event search query job created", "job_id", jobID, "repository", repository)
	msg.Trace(ctx, "⏳ Event search job created (job_id: `%s`), polling for results...", jobID)

	// Step 2: Poll for results until done
	resultPath := fmt.Sprintf("/humio/api/v1/repositories/%s/queryjobs/%s", repository, jobID)
	const (
		maxPolls     = 60
		pollInterval = 2 * time.Second
	)

	var allEvents []any

	for i := range maxPolls {
		if i > 0 {
			select {
			case <-ctx.Done():
				return nil, goerr.Wrap(ctx.Err(), "context canceled while polling event search")
			case <-time.After(pollInterval):
			}
		}

		pollResp, err := t.doRequest(ctx, http.MethodGet, resultPath, nil)
		if err != nil {
			msg.Warn(ctx, "⚠️ *[Falcon]* Failed to poll event search results (job: `%s`, attempt %d): %v", jobID, i+1, err)
			return nil, goerr.Wrap(err, "failed to poll event search results",
				goerr.V("job_id", jobID),
				goerr.V("poll_attempt", i+1),
			)
		}

		// Collect events from this poll
		if events, ok := pollResp["events"].([]any); ok {
			allEvents = append(allEvents, events...)
		}

		// Check if done
		if done, ok := pollResp["done"].(bool); ok && done {
			log.Debug("Event search completed",
				"job_id", jobID,
				"total_events", len(allEvents),
				"polls", i+1,
			)
			msg.Trace(ctx, "✅ Event search completed: %d events retrieved", len(allEvents))

			result := map[string]any{
				"done":       true,
				"events":     allEvents,
				"repository": repository,
			}

			// Include metadata if available
			if meta, ok := pollResp["metadataResult"]; ok {
				result["metadata"] = meta
			}

			return result, nil
		}

		log.Debug("Event search still running, polling...",
			"job_id", jobID,
			"poll_attempt", i+1,
			"events_so_far", len(allEvents),
		)
	}

	// Return partial results if max polls reached
	log.Warn("Event search reached max poll limit, returning partial results",
		"job_id", jobID,
		"total_events", len(allEvents),
	)
	msg.Trace(ctx, "⚠️ Event search reached poll limit, returning %d partial results", len(allEvents))

	return map[string]any{
		"done":       false,
		"events":     allEvents,
		"repository": repository,
		"warning":    "Search did not complete within the polling limit. Partial results returned.",
	}, nil
}

// queryParams constructs a URL query string from string key/value pairs,
// skipping empty values. Callers pre-format non-string values (e.g. numeric
// limits) before passing them in.
func queryParams(pairs ...[2]string) string {
	params := url.Values{}
	for _, kv := range pairs {
		if kv[1] != "" {
			params.Set(kv[0], kv[1])
		}
	}
	return params.Encode()
}

// numParam formats a numeric pagination value for a query string. Zero is
// treated as "absent" (returns ""), matching the previous behavior where a
// missing limit/offset argument was simply omitted from the request.
func numParam(v float64) string {
	if v <= 0 {
		return ""
	}
	return fmt.Sprintf("%d", int(v))
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
