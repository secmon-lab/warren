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
	"strconv"
	"strings"
	"time"

	"github.com/gollem-dev/gollem"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
	"github.com/secmon-lab/warren/pkg/utils/safe"
)

// internalTool implements gollem.ToolSet for CrowdStrike Falcon API operations.
type internalTool struct {
	tokenProvider *tokenProvider
	baseURL       string
	httpClient    *http.Client

	// storage holds event search snapshots for stable pagination across
	// separate tool calls. It is the warren-wide shared client (bucket
	// bound). When nil, event search degrades to returning the first page
	// directly without snapshotting. storagePrefix namespaces the objects.
	storage       interfaces.StorageClient
	storagePrefix string
}

// newInternalTool creates a new internalTool for Falcon API calls.
func newInternalTool(tp *tokenProvider, baseURL string, storage interfaces.StorageClient, storagePrefix string) *internalTool {
	return &internalTool{
		tokenProvider: tp,
		baseURL:       baseURL,
		httpClient:    &http.Client{Timeout: 60 * time.Second},
		storage:       storage,
		storagePrefix: storagePrefix,
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
		{
			Name:        "falcon_search_devices",
			Description: "Search for device (host) IDs using FQL filters. Returns a list of device IDs that can be used with falcon_get_devices to retrieve full host details including OS, IP addresses, sensor version, and containment status.",
			Parameters: map[string]*gollem.Parameter{
				"filter": {
					Type:        gollem.TypeString,
					Description: "FQL filter expression (e.g., \"hostname:'*web*'\", \"platform_name:'Windows'\", \"last_seen:>='2025-01-01'\", \"external_ip:'10.0.0.*'\")",
				},
				"sort": {
					Type:        gollem.TypeString,
					Description: "Sort expression (e.g., \"hostname.asc\", \"last_seen.desc\")",
				},
				"limit": {
					Type:        gollem.TypeNumber,
					Description: "Maximum number of IDs to return (default: 100, max: 5000)",
				},
				"offset": {
					Type:        gollem.TypeString,
					Description: "Pagination offset token from a previous response",
				},
			},
		},
		{
			Name:        "falcon_get_devices",
			Description: "Get detailed device (host) information by device IDs. Returns full host details including hostname, OS, IP addresses, sensor version, tags, and containment status.",
			Parameters: map[string]*gollem.Parameter{
				"ids": {
					Type:        gollem.TypeString,
					Description: "Comma-separated device IDs (up to 5000, e.g., \"abc123def456,ghi789jkl012\")",
					Required:    true,
				},
			},
		},
		{
			Name: "falcon_search_events",
			Description: "Search EDR telemetry events using CrowdStrike Query Language (CQL) via the Next-Gen SIEM Search API (process executions, network connections, file writes, DNS requests, etc.). " +
				"Results are returned in pages: at most 100 events are returned per call along with the total count. " +
				"NOTE: a filter query returns at most 200 events by default; to retrieve more (up to 20000), append `| tail(N)` to query_string. For the exact number of matching events, use an aggregation like `| count()` instead of paging through raw events. " +
				"To page through results, pass the returned result_set_id with an increased offset; this avoids re-running the query.",
			Parameters: map[string]*gollem.Parameter{
				"query_string": {
					Type:        gollem.TypeString,
					Description: "CQL query string (e.g., \"aid=abc123\", \"#event_simpleName=ProcessRollup2 AND FileName=cmd.exe\", \"ComputerName=workstation1 | tail(1000)\"). Required for a new search; omitted when paging with result_set_id.",
				},
				"repository": {
					Type:        gollem.TypeString,
					Description: "Repository to search. Values: \"search-all\" (all data, default), \"investigate_view\" (Falcon EDR), \"third-party\" (third-party data), \"falcon_for_it_view\" (IT Automation), \"forensics_view\" (Forensics)",
				},
				"start": {
					Type:        gollem.TypeString,
					Description: "Start time for the search (e.g., \"1d\" for last 1 day, \"24h\" for last 24 hours, \"2025-01-01T00:00:00Z\" for absolute time). Default: \"1d\"",
				},
				"end": {
					Type:        gollem.TypeString,
					Description: "End time for the search (e.g., \"now\", \"2025-01-02T00:00:00Z\"). Default: \"now\"",
				},
				"limit": {
					Type:        gollem.TypeNumber,
					Description: "Maximum number of events to return in this page (default: 100, max: 100).",
				},
				"offset": {
					Type:        gollem.TypeNumber,
					Description: "Zero-based index of the first event to return for pagination (default: 0).",
				},
				"result_set_id": {
					Type:        gollem.TypeString,
					Description: "ID of a previously created result set (returned by an earlier call). When set, returns another page from the stored snapshot without re-running the query. Use together with offset/limit.",
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
	case "falcon_search_devices":
		return t.searchDevices(ctx, args)
	case "falcon_get_devices":
		return t.getDevices(ctx, args)
	case "falcon_get_crowdscores":
		return t.getCrowdScores(ctx, args)
	case "falcon_search_events":
		return t.searchEvents(ctx, args)
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
func (t *internalTool) searchIncidents(ctx context.Context, args map[string]any) (map[string]any, error) {
	filter, _ := args["filter"].(string)
	msg.Trace(ctx, "🔍 Searching incidents (filter: `%s`)", filter)

	path := "/incidents/queries/incidents/v1"
	params := buildQueryParams(args, "filter", "sort", "limit", "offset")
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
func (t *internalTool) getIncidents(ctx context.Context, args map[string]any) (map[string]any, error) {
	ids, ok := args["ids"].(string)
	if !ok || ids == "" {
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
func (t *internalTool) searchAlerts(ctx context.Context, args map[string]any) (map[string]any, error) {
	filter, _ := args["filter"].(string)
	msg.Trace(ctx, "🔍 Searching alerts (filter: `%s`)", filter)

	body := make(map[string]any)
	if filter != "" {
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
	result, err := t.doRequest(ctx, http.MethodPost, "/alerts/combined/alerts/v1", body)
	if err != nil {
		msg.Warn(ctx, "⚠️ *[Falcon]* Alert search failed (filter: `%s`): %v", filter, err)
		return nil, err
	}
	msg.Trace(ctx, "✅ Alert search completed")
	return result, nil
}

// getAlerts retrieves alert details by composite IDs.
func (t *internalTool) getAlerts(ctx context.Context, args map[string]any) (map[string]any, error) {
	compositeIDs, ok := args["composite_ids"].(string)
	if !ok || compositeIDs == "" {
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
func (t *internalTool) searchBehaviors(ctx context.Context, args map[string]any) (map[string]any, error) {
	filter, _ := args["filter"].(string)
	msg.Trace(ctx, "🔍 Searching behaviors (filter: `%s`)", filter)

	path := "/incidents/queries/behaviors/v1"
	params := buildQueryParams(args, "filter", "limit", "offset")
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
func (t *internalTool) getBehaviors(ctx context.Context, args map[string]any) (map[string]any, error) {
	ids, ok := args["ids"].(string)
	if !ok || ids == "" {
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
func (t *internalTool) searchDevices(ctx context.Context, args map[string]any) (map[string]any, error) {
	filter, _ := args["filter"].(string)
	msg.Trace(ctx, "🔍 Searching devices (filter: `%s`)", filter)

	path := "/devices/queries/devices-scroll/v1"
	params := buildQueryParams(args, "filter", "sort", "limit", "offset")
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
func (t *internalTool) getDevices(ctx context.Context, args map[string]any) (map[string]any, error) {
	ids, ok := args["ids"].(string)
	if !ok || ids == "" {
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
func (t *internalTool) getCrowdScores(ctx context.Context, args map[string]any) (map[string]any, error) {
	filter, _ := args["filter"].(string)
	msg.Trace(ctx, "📊 Retrieving CrowdScores (filter: `%s`)", filter)

	path := "/incidents/combined/crowdscores/v1"
	params := buildQueryParams(args, "filter")
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

const (
	defaultEventLimit = 100
	maxEventLimit     = 100
	maxEventPolls     = 60
	eventPollInterval = 2 * time.Second
)

// searchEvents runs a CQL query via the Next-Gen SIEM Search API and returns
// events one page at a time (at most maxEventLimit per call) together with the
// total result-set size.
//
// On a new search it polls the query job until completion and, when storage is
// configured, snapshots the full result set so later pages can be served via
// result_set_id without re-running the query. The query job poll returns the
// cumulative result set on every poll, so only the final (done) response is
// authoritative — earlier polls are not accumulated (doing so would duplicate
// events).
func (t *internalTool) searchEvents(ctx context.Context, args map[string]any) (map[string]any, error) {
	limit := parseLimit(args)
	offset := parseOffset(args)

	// Pagination over an existing snapshot: serve from storage, no query run.
	if resultSetID, ok := args["result_set_id"].(string); ok && resultSetID != "" {
		return t.paginateEventSnapshot(ctx, resultSetID, offset, limit)
	}

	queryString, ok := args["query_string"].(string)
	if !ok || queryString == "" {
		return nil, goerr.New("query_string is required", goerr.T(errutil.TagValidation))
	}

	repository := "search-all"
	if repo, ok := args["repository"].(string); ok && repo != "" {
		repository = repo
	}

	jobID, events, metadata, done, err := t.runEventQuery(ctx, queryString, repository, args)
	if err != nil {
		return nil, err
	}

	// Snapshot for later pagination. Best-effort: a storage failure must not
	// fail the search, but it does mean only the first page is reachable.
	resultSetID := ""
	if t.storage != nil {
		if err := t.writeEventSnapshot(ctx, jobID, events); err != nil {
			errutil.Handle(ctx, goerr.Wrap(err, "failed to snapshot falcon events for pagination", goerr.V("job_id", jobID)))
			msg.Warn(ctx, "⚠️ *[Falcon]* Could not store result set for pagination; only the first page is available")
		} else {
			resultSetID = jobID
		}
	}

	page, returned, hasMore := paginate(events, offset, limit)
	result := buildEventResult(resultSetID, page, len(events), offset, limit, returned, hasMore, done)
	result["repository"] = repository
	if metadata != nil {
		result["metadata"] = metadata
	}
	// Surface the true match count when the API reports it (it may exceed the
	// number of events actually returned, e.g. when the query has no tail()).
	if matched, ok := eventCountFromMetadata(metadata); ok && matched != len(events) {
		result["total_matched"] = matched
	}
	if !done {
		result["warning"] = "Search did not complete within the polling limit. Partial results returned."
	}
	return result, nil
}

// runEventQuery creates a Next-Gen SIEM query job and polls until completion.
// It returns the job ID, the final (cumulative) events, the metaData object,
// and whether the search completed within the poll limit. Intermediate polls
// overwrite (not append) the events, since each poll returns the full result
// set computed so far.
func (t *internalTool) runEventQuery(ctx context.Context, queryString, repository string, args map[string]any) (jobID string, events []any, metadata any, done bool, err error) {
	log := logging.From(ctx)

	body := map[string]any{"queryString": queryString}
	if start, ok := args["start"].(string); ok && start != "" {
		body["start"] = start
	} else {
		body["start"] = "1d"
	}
	if end, ok := args["end"].(string); ok && end != "" {
		body["end"] = end
	} else {
		body["end"] = "now"
	}

	msg.Trace(ctx, "🔍 Searching events (query: `%s`, repo: `%s`)", queryString, repository)
	jobPath := fmt.Sprintf("/humio/api/v1/repositories/%s/queryjobs", repository)
	jobResp, err := t.doRequest(ctx, http.MethodPost, jobPath, body)
	if err != nil {
		msg.Warn(ctx, "⚠️ *[Falcon]* Failed to create event search job (query: `%s`): %v", queryString, err)
		return "", nil, nil, false, goerr.Wrap(err, "failed to create event search query job",
			goerr.T(errutil.TagExternal),
			goerr.V("repository", repository),
			goerr.V("query", queryString),
		)
	}

	jobID, ok := jobResp["id"].(string)
	if !ok || jobID == "" {
		msg.Warn(ctx, "⚠️ *[Falcon]* No job ID returned from event search (query: `%s`)", queryString)
		return "", nil, nil, false, goerr.New("no job ID returned from query job creation",
			goerr.T(errutil.TagExternal),
			goerr.V("response", jobResp),
		)
	}

	log.Debug("Event search query job created", "job_id", jobID, "repository", repository)
	msg.Trace(ctx, "⏳ Event search job created (job_id: `%s`), polling for results...", jobID)

	resultPath := fmt.Sprintf("/humio/api/v1/repositories/%s/queryjobs/%s", repository, jobID)

	var lastEvents []any
	var lastMeta any

	for i := range maxEventPolls {
		if i > 0 {
			select {
			case <-ctx.Done():
				return "", nil, nil, false, goerr.Wrap(ctx.Err(), "context canceled while polling event search", goerr.T(errutil.TagTimeout))
			case <-time.After(eventPollInterval):
			}
		}

		pollResp, err := t.doRequest(ctx, http.MethodGet, resultPath, nil)
		if err != nil {
			msg.Warn(ctx, "⚠️ *[Falcon]* Failed to poll event search results (job: `%s`, attempt %d): %v", jobID, i+1, err)
			return "", nil, nil, false, goerr.Wrap(err, "failed to poll event search results",
				goerr.T(errutil.TagExternal),
				goerr.V("job_id", jobID),
				goerr.V("poll_attempt", i+1),
			)
		}

		// Each poll returns the cumulative result set, so overwrite rather
		// than append to avoid duplicating events across polls.
		if evs, ok := pollResp["events"].([]any); ok {
			lastEvents = evs
		}
		if meta, ok := pollResp["metaData"]; ok {
			lastMeta = meta
		}

		if isDone, _ := pollResp["done"].(bool); isDone {
			log.Debug("Event search completed", "job_id", jobID, "total_events", len(lastEvents), "polls", i+1)
			msg.Trace(ctx, "✅ Event search completed: %d events retrieved", len(lastEvents))
			return jobID, lastEvents, lastMeta, true, nil
		}

		log.Debug("Event search still running, polling...", "job_id", jobID, "poll_attempt", i+1, "events_so_far", len(lastEvents))
	}

	log.Warn("Event search reached max poll limit, returning partial results", "job_id", jobID, "total_events", len(lastEvents))
	msg.Trace(ctx, "⚠️ Event search reached poll limit, returning %d partial results", len(lastEvents))
	return jobID, lastEvents, lastMeta, false, nil
}

// paginateEventSnapshot serves a page of events from a previously stored
// snapshot, reading the newline-delimited JSON without holding the whole
// document as a single combined structure.
func (t *internalTool) paginateEventSnapshot(ctx context.Context, resultSetID string, offset, limit int) (map[string]any, error) {
	if t.storage == nil {
		return nil, goerr.New("result set pagination is unavailable because storage is not configured",
			goerr.T(errutil.TagInvalidState), goerr.V("result_set_id", resultSetID))
	}

	path := t.eventSnapshotPath(resultSetID)
	r, err := t.storage.GetObject(ctx, path)
	if err != nil {
		msg.Warn(ctx, "⚠️ *[Falcon]* Result set `%s` not found; re-run the search without result_set_id", resultSetID)
		return nil, goerr.Wrap(err, "failed to open event snapshot",
			goerr.T(errutil.TagNotFound), goerr.V("result_set_id", resultSetID))
	}
	defer safe.Close(ctx, r)

	events, err := decodeNDJSON(r)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to read event snapshot",
			goerr.T(errutil.TagInternal), goerr.V("result_set_id", resultSetID))
	}

	page, returned, hasMore := paginate(events, offset, limit)
	msg.Trace(ctx, "📄 Returning events page from result set `%s` (offset: %d, returned: %d, total: %d)", resultSetID, offset, returned, len(events))
	return buildEventResult(resultSetID, page, len(events), offset, limit, returned, hasMore, true), nil
}

// writeEventSnapshot streams the events to object storage as newline-delimited
// JSON (one event per line) so later pages can be read back without re-running
// the query. Events are encoded one line at a time rather than as one combined
// document to avoid building a large intermediate payload.
func (t *internalTool) writeEventSnapshot(ctx context.Context, resultSetID string, events []any) error {
	w := t.storage.PutObject(ctx, t.eventSnapshotPath(resultSetID))

	enc := json.NewEncoder(w)
	for _, ev := range events {
		if err := enc.Encode(ev); err != nil {
			// Skip Close so the partial object is never committed (GCS and the
			// in-memory client both commit on Close).
			return goerr.Wrap(err, "failed to encode event to snapshot", goerr.T(errutil.TagInternal))
		}
	}

	if err := w.Close(); err != nil {
		return goerr.Wrap(err, "failed to finalize event snapshot", goerr.T(errutil.TagInternal))
	}
	return nil
}

// eventSnapshotPath builds the storage object key for an event result set,
// honoring the warren-wide storage prefix.
func (t *internalTool) eventSnapshotPath(resultSetID string) string {
	return fmt.Sprintf("%sfalcon/events/%s.ndjson", t.storagePrefix, resultSetID)
}

// decodeNDJSON reads newline-delimited JSON events from r, decoding one value
// at a time off the stream.
func decodeNDJSON(r io.Reader) ([]any, error) {
	events := []any{}
	dec := json.NewDecoder(r)
	for {
		var ev any
		if err := dec.Decode(&ev); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, goerr.Wrap(err, "failed to decode snapshot event")
		}
		events = append(events, ev)
	}
	return events, nil
}

// parseLimit returns the page size bounded to [1, maxEventLimit], defaulting to
// defaultEventLimit for missing or invalid input.
func parseLimit(args map[string]any) int {
	n, ok := parseIntArg(args["limit"])
	if !ok || n < 1 {
		return defaultEventLimit
	}
	if n > maxEventLimit {
		return maxEventLimit
	}
	return n
}

// parseOffset returns a non-negative pagination offset, defaulting to 0.
func parseOffset(args map[string]any) int {
	n, ok := parseIntArg(args["offset"])
	if !ok || n < 0 {
		return 0
	}
	return n
}

// parseIntArg interprets a tool argument as an int. gollem may deliver numbers
// as float64 or as strings, so both are accepted.
func parseIntArg(v any) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	case string:
		trimmed := strings.TrimSpace(n)
		if trimmed == "" {
			return 0, false
		}
		i, err := strconv.Atoi(trimmed)
		if err != nil {
			return 0, false
		}
		return i, true
	default:
		return 0, false
	}
}

// paginate returns events[offset:offset+limit] along with the number returned
// and whether any events remain beyond the page.
func paginate(events []any, offset, limit int) (page []any, returned int, hasMore bool) {
	total := len(events)
	if offset >= total {
		return []any{}, 0, false
	}
	end := min(offset+limit, total)
	page = events[offset:end]
	return page, len(page), end < total
}

// eventCountFromMetadata extracts the total matched event count from the query
// job metaData, when present.
func eventCountFromMetadata(metadata any) (int, bool) {
	m, ok := metadata.(map[string]any)
	if !ok {
		return 0, false
	}
	switch ec := m["eventCount"].(type) {
	case float64:
		return int(ec), true
	case int:
		return ec, true
	default:
		return 0, false
	}
}

// buildEventResult assembles the common paginated event response fields.
func buildEventResult(resultSetID string, page []any, total, offset, limit, returned int, hasMore, done bool) map[string]any {
	if page == nil {
		page = []any{}
	}
	result := map[string]any{
		"done":     done,
		"events":   page,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
		"returned": returned,
		"has_more": hasMore,
	}
	if resultSetID != "" {
		result["result_set_id"] = resultSetID
	}
	return result
}

// buildQueryParams constructs URL query parameters from tool arguments.
func buildQueryParams(args map[string]any, keys ...string) string {
	params := url.Values{}
	for _, key := range keys {
		if val, ok := args[key]; ok {
			switch v := val.(type) {
			case string:
				if v != "" {
					params.Set(key, v)
				}
			case float64:
				params.Set(key, fmt.Sprintf("%d", int(v)))
			}
		}
	}
	return params.Encode()
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
