package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/secmon-lab/warren/pkg/utils/safe"
	"github.com/urfave/cli/v3"
)

const (
	slackAPIURL    = "https://slack.com/api/search.messages"
	defaultTimeout = 30 * time.Second
	maxRetries     = 3
)

type Action struct {
	oauthToken string
	client     interfaces.SlackClient // for future extensions
	baseURL    string                  // for testing
}

var _ interfaces.Tool = &Action{}

func (x *Action) Helper() *cli.Command {
	return nil
}

func (x *Action) Name() string {
	return "slack_message_search"
}

func (x *Action) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "slack-tool-user-token",
			Usage:       "Slack User token for search API access (must be User token, not Bot token)",
			Destination: &x.oauthToken,
			Category:    "Tool",
			Sources:     cli.EnvVars("WARREN_SLACK_TOOL_USER_TOKEN"),
		},
	}
}

func (x *Action) SetSlackClient(client interfaces.SlackClient) {
	x.client = client
}

func (x *Action) Specs(ctx context.Context) ([]gollem.ToolSpec, error) {
	return []gollem.ToolSpec{
		{
			Name:        "slack_message_search",
			Description: "Search for messages in Slack workspace using the search.messages API",
			Parameters: map[string]*gollem.Parameter{
				"query": {
					Type:        gollem.TypeString,
					Description: "The search query (e.g., 'from:@user', 'in:general', 'has:link')",
				},
				"sort": {
					Type:        gollem.TypeString,
					Description: "Sort order: 'score' (relevance) or 'timestamp' (newest first)",
				},
				"sort_dir": {
					Type:        gollem.TypeString,
					Description: "Sort direction: 'asc' or 'desc'",
				},
				"count": {
					Type:        gollem.TypeNumber,
					Description: "Number of results to return (default: 20, max: 100)",
				},
				"page": {
					Type:        gollem.TypeNumber,
					Description: "Page number for pagination (default: 1)",
				},
			},
			Required: []string{"query"},
		},
	}, nil
}

func (x *Action) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	if x.oauthToken == "" {
		return nil, goerr.New("Slack OAuth User token is required")
	}

	// Parse arguments
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return nil, goerr.New("query is required")
	}

	// Build search options
	opts := &SearchOptions{
		Query: query,
		Count: 20, // default
		Page:  1,  // default
	}

	// Optional parameters
	if sort, ok := args["sort"].(string); ok {
		opts.Sort = sort
	}
	if sortDir, ok := args["sort_dir"].(string); ok {
		opts.SortDir = sortDir
	}
	if count, ok := args["count"].(float64); ok {
		opts.Count = int(count)
	}
	if page, ok := args["page"].(float64); ok {
		opts.Page = int(page)
	}

	// Execute search
	resp, err := x.searchMessages(ctx, opts)
	if err != nil {
		return nil, err
	}

	// Format output
	output := x.formatOutput(resp)

	// Convert to map[string]any for gollem
	messages := make([]any, len(output.Messages))
	for i, msg := range output.Messages {
		messages[i] = map[string]any{
			"channel":        msg.Channel,
			"channel_name":   msg.ChannelName,
			"user":          msg.User,
			"user_name":     msg.UserName,
			"text":          msg.Text,
			"timestamp":     msg.Timestamp,
			"permalink":     msg.Permalink,
			"formatted_time": msg.FormattedTime,
		}
	}
	
	result := map[string]any{
		"total":    float64(output.Total),
		"messages": messages,
	}

	return result, nil
}

// SetOAuthToken sets the OAuth token (for testing)
func (x *Action) SetOAuthToken(token string) {
	x.oauthToken = token
}

// SetTestURL sets a custom base URL (for testing)
func (x *Action) SetTestURL(url string) {
	x.baseURL = url
}

// searchMessages performs the actual API call to Slack
func (x *Action) searchMessages(ctx context.Context, opts *SearchOptions) (*SearchResponse, error) {
	// Build query parameters
	params := url.Values{}
	params.Set("query", opts.Query)
	if opts.Sort != "" {
		params.Set("sort", opts.Sort)
	}
	if opts.SortDir != "" {
		params.Set("sort_dir", opts.SortDir)
	}
	params.Set("count", strconv.Itoa(opts.Count))
	params.Set("page", strconv.Itoa(opts.Page))
	if opts.Highlight {
		params.Set("highlight", "true")
	}

	// Prepare request with retries
	var resp *SearchResponse
	var lastErr error

	for i := range maxRetries {
		select {
		case <-ctx.Done():
			return nil, goerr.Wrap(ctx.Err(), "context cancelled")
		default:
		}

		// Create request
		apiURL := slackAPIURL
		if x.baseURL != "" {
			apiURL = x.baseURL + "/search.messages"
		}
		reqURL := fmt.Sprintf("%s?%s", apiURL, params.Encode())
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to create request")
		}

		// Set headers
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", x.oauthToken))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		// Execute request
		client := &http.Client{Timeout: defaultTimeout}
		httpResp, err := client.Do(req)
		if err != nil {
			lastErr = goerr.Wrap(err, "failed to execute request")
			continue
		}
		defer safe.Close(ctx, httpResp.Body)

		// Handle rate limiting
		if httpResp.StatusCode == http.StatusTooManyRequests {
			retryAfter := httpResp.Header.Get("Retry-After")
			if retryAfter != "" {
				if seconds, err := strconv.Atoi(retryAfter); err == nil {
					slog.InfoContext(ctx, "Rate limited, retrying after", "seconds", seconds)
					time.Sleep(time.Duration(seconds) * time.Second)
					continue
				}
			}
			// Default retry wait
			time.Sleep(time.Duration(i+1) * time.Second)
			continue
		}

		// Read response
		body, err := io.ReadAll(httpResp.Body)
		if err != nil {
			lastErr = goerr.Wrap(err, "failed to read response body")
			continue
		}

		// Parse response
		resp = &SearchResponse{}
		if err := json.Unmarshal(body, resp); err != nil {
			return nil, goerr.Wrap(err, "failed to parse response", goerr.V("body", string(body)))
		}

		// Check API response
		if !resp.OK {
			lastErr = goerr.New("Slack API error", goerr.V("error", resp.Error))
			// Don't retry on API errors (except rate limit)
			if resp.Error != "rate_limited" {
				return nil, lastErr
			}
			continue
		}

		// Success
		return resp, nil
	}

	return nil, goerr.Wrap(lastErr, "failed after retries", goerr.V("retries", maxRetries))
}

// formatOutput formats the search response for output
func (x *Action) formatOutput(resp *SearchResponse) SearchOutput {
	output := SearchOutput{
		Total:    resp.Messages.Total,
		Messages: make([]SearchMessageItem, 0, len(resp.Messages.Matches)),
	}

	for _, msg := range resp.Messages.Matches {
		item := SearchMessageItem{
			Channel:     msg.Channel.ID,
			ChannelName: msg.Channel.Name,
			User:        msg.User,
			UserName:    msg.Username,
			Text:        msg.Text,
			Timestamp:   msg.Timestamp,
			Permalink:   msg.Permalink,
		}

		// Parse timestamp if possible
		if ts, err := strconv.ParseFloat(msg.Timestamp, 64); err == nil {
			item.FormattedTime = time.Unix(int64(ts), 0)
		}

		output.Messages = append(output.Messages, item)
	}

	return output
}

func (x *Action) Configure(ctx context.Context) error {
	if x.oauthToken == "" {
		return errs.ErrActionUnavailable
	}
	return nil
}

func (x *Action) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Int("oauth_token.len", len(x.oauthToken)),
	)
}

// Prompt returns additional instructions for the system prompt
func (x *Action) Prompt(ctx context.Context) (string, error) {
	return "", nil
}