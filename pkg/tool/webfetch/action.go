package webfetch

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/utils/safe"
	"github.com/urfave/cli/v3"
)

const (
	maxResponseBody = 1024 * 1024 // 1MB
	defaultTimeout  = 30 * time.Second
)

type fetchFunc func(ctx context.Context, url string) (string, error)

// Action implements the interfaces.Tool interface for fetching web content.
type Action struct {
	fetchFn fetchFunc
	client  *http.Client
}

var _ interfaces.Tool = &Action{}

func defaultFetch(client *http.Client) fetchFunc {
	return func(ctx context.Context, url string) (string, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return "", goerr.Wrap(err, "failed to create request", goerr.V("url", url))
		}

		resp, err := client.Do(req)
		if err != nil {
			return "", goerr.Wrap(err, "failed to fetch URL", goerr.V("url", url))
		}
		defer safe.Close(ctx, resp.Body)

		body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
		if err != nil {
			return "", goerr.Wrap(err, "failed to read response body", goerr.V("url", url))
		}

		return fmt.Sprintf("HTTP %d\n\n%s", resp.StatusCode, string(body)), nil
	}
}

func (x *Action) Helper() *cli.Command {
	return nil
}

func (x *Action) ID() string {
	return "webfetch"
}

func (x *Action) Description() string {
	return "Web page content fetching"
}

func (x *Action) Flags() []cli.Flag {
	return []cli.Flag{}
}

func (x *Action) Specs(_ context.Context) ([]gollem.ToolSpec, error) {
	return []gollem.ToolSpec{
		{
			Name:        "web_fetch",
			Description: "Fetch the content of a web page at the given URL. Returns the HTTP status code and response body.",
			Parameters: map[string]*gollem.Parameter{
				"url": {
					Type:        gollem.TypeString,
					Description: "The URL to fetch",
					Required:    true,
				},
			},
		},
	}, nil
}

func (x *Action) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	if name != "web_fetch" {
		return nil, goerr.New("invalid function name", goerr.V("name", name))
	}

	url, ok := args["url"].(string)
	if !ok || url == "" {
		return nil, goerr.New("url is required")
	}

	fn := x.fetchFn
	if fn == nil {
		c := x.client
		if c == nil {
			c = &http.Client{Timeout: defaultTimeout}
		}
		fn = defaultFetch(c)
	}

	result, err := fn(ctx, url)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"result": result,
	}, nil
}

func (x *Action) Configure(_ context.Context) error {
	return nil
}

func (x *Action) LogValue() slog.Value {
	return slog.GroupValue()
}

func (x *Action) Prompt(_ context.Context) (string, error) {
	return "", nil
}
