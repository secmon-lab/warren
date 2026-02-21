package whois

import (
	"context"
	"log/slog"
	"time"

	whoislib "github.com/likexian/whois"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/urfave/cli/v3"
)

type queryFunc func(ctx context.Context, target string) (string, error)

// Action implements the interfaces.Tool interface for WHOIS lookups.
type Action struct {
	queryFn queryFunc
}

var _ interfaces.Tool = &Action{}

func defaultQuery(ctx context.Context, target string) (string, error) {
	client := whoislib.NewClient()
	if deadline, ok := ctx.Deadline(); ok {
		client.SetTimeout(time.Until(deadline))
	}
	result, err := client.Whois(target)
	if err != nil {
		return "", goerr.Wrap(err, "failed to query whois", goerr.V("target", target))
	}
	return result, nil
}

func (x *Action) query(ctx context.Context, target string) (string, error) {
	if x.queryFn != nil {
		return x.queryFn(ctx, target)
	}
	return defaultQuery(ctx, target)
}

func (x *Action) Helper() *cli.Command {
	return nil
}

func (x *Action) Name() string {
	return "whois"
}

func (x *Action) Flags() []cli.Flag {
	return []cli.Flag{}
}

func (x *Action) Specs(_ context.Context) ([]gollem.ToolSpec, error) {
	return []gollem.ToolSpec{
		{
			Name:        "whois_domain",
			Description: "Perform a WHOIS lookup for a domain name to retrieve registration information such as owner, registrar, registration date, and expiration date.",
			Parameters: map[string]*gollem.Parameter{
				"target": {
					Type:        gollem.TypeString,
					Description: "The domain name to look up",
				},
			},
		},
		{
			Name:        "whois_ip",
			Description: "Perform a WHOIS lookup for an IP address (IPv4 or IPv6) to retrieve network registration information such as owner, ISP, and allocated range.",
			Parameters: map[string]*gollem.Parameter{
				"target": {
					Type:        gollem.TypeString,
					Description: "The IP address (IPv4 or IPv6) to look up",
				},
			},
		},
	}, nil
}

func (x *Action) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	var target string

	switch name {
	case "whois_domain", "whois_ip":
		t, ok := args["target"].(string)
		if !ok || t == "" {
			return nil, goerr.New("target is required")
		}
		target = t
	default:
		return nil, goerr.New("invalid function name", goerr.V("name", name))
	}

	result, err := x.query(ctx, target)
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
