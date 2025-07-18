package config

import (
	"context"
	"log/slog"
	"os"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/mcp"
	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
)

// MCPServerConfig represents configuration for a remote MCP server
type MCPServerConfig struct {
	Name     string            `yaml:"name"`
	Type     string            `yaml:"type"` // "sse" or "http"
	URL      string            `yaml:"url"`
	Headers  map[string]string `yaml:"headers,omitempty"`
	Disabled bool              `yaml:"disabled,omitempty"`
}

// MCPLocalConfig represents configuration for a local MCP server
type MCPLocalConfig struct {
	Name     string   `yaml:"name"`
	Command  string   `yaml:"command"`
	Args     []string `yaml:"args,omitempty"`
	Env      []string `yaml:"env,omitempty"`
	Disabled bool     `yaml:"disabled,omitempty"`
}

// MCPConfig represents MCP configuration
type MCPConfig struct {
	configFile string
	Servers    []MCPServerConfig `yaml:"servers,omitempty"`
	Local      []MCPLocalConfig  `yaml:"local,omitempty"`
}

func (x *MCPConfig) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "mcp-config",
			Usage:       "Path to MCP configuration YAML file",
			Sources:     cli.EnvVars("WARREN_MCP_CONFIG", "MCP_CONFIG"),
			Destination: &x.configFile,
			Category:    "MCP",
		},
	}
}

func (x MCPConfig) LogValue() slog.Value {
	attrs := []slog.Attr{
		slog.String("config_file", x.configFile),
		slog.Int("server_count", len(x.Servers)),
		slog.Int("local_count", len(x.Local)),
	}

	return slog.GroupValue(attrs...)
}

// LoadConfig loads MCP configuration from the specified YAML file
func (x *MCPConfig) LoadConfig() error {
	if x.configFile == "" {
		// No config file specified, MCP is disabled
		return nil
	}

	data, err := os.ReadFile(x.configFile)
	if err != nil {
		return goerr.Wrap(err, "failed to read MCP config file", goerr.V("file", x.configFile))
	}

	if err := yaml.Unmarshal(data, x); err != nil {
		return goerr.Wrap(err, "failed to parse MCP config file", goerr.V("file", x.configFile))
	}

	return nil
}

// CreateMCPClients creates MCP clients based on the configuration
func (x *MCPConfig) CreateMCPClients(ctx context.Context) ([]*mcp.Client, error) {
	if x.configFile == "" {
		return nil, nil
	}

	if err := x.LoadConfig(); err != nil {
		return nil, goerr.Wrap(err, "failed to load MCP configuration")
	}

	var clients []*mcp.Client

	// Process servers configuration
	for _, serverCfg := range x.Servers {
		if client, err := x.createMCPServerClient(ctx, serverCfg); err != nil {
			return nil, err
		} else if client != nil {
			clients = append(clients, client)
		}
	}

	// Process local configuration
	for _, localCfg := range x.Local {
		if client, err := x.createMCPLocalClient(ctx, localCfg); err != nil {
			return nil, err
		} else if client != nil {
			clients = append(clients, client)
		}
	}

	return clients, nil
}

// createMCPServerClient creates a remote MCP client from server configuration
func (x *MCPConfig) createMCPServerClient(ctx context.Context, serverCfg MCPServerConfig) (*mcp.Client, error) {
	if serverCfg.Disabled {
		return nil, nil
	}

	var client *mcp.Client
	var err error

	switch serverCfg.Type {
	case "sse":
		if serverCfg.URL == "" {
			return nil, goerr.New("url is required for SSE MCP server", goerr.V("server", serverCfg.Name))
		}

		var opts []mcp.SSEOption
		if len(serverCfg.Headers) > 0 {
			opts = append(opts, mcp.WithSSEHeaders(serverCfg.Headers))
		}
		opts = append(opts, mcp.WithSSEClientInfo("warren", "1.0.0"))

		client, err = mcp.NewSSE(ctx, serverCfg.URL, opts...)

	case "http":
		if serverCfg.URL == "" {
			return nil, goerr.New("url is required for HTTP MCP server", goerr.V("server", serverCfg.Name))
		}

		var opts []mcp.StreamableHTTPOption
		if len(serverCfg.Headers) > 0 {
			opts = append(opts, mcp.WithStreamableHTTPHeaders(serverCfg.Headers))
		}
		opts = append(opts, mcp.WithStreamableHTTPClientInfo("warren", "1.0.0"))

		client, err = mcp.NewStreamableHTTP(ctx, serverCfg.URL, opts...)

	default:
		return nil, goerr.New("unsupported MCP server type",
			goerr.V("server", serverCfg.Name),
			goerr.V("type", serverCfg.Type))
	}

	if err != nil {
		return nil, goerr.Wrap(err, "failed to create MCP server client",
			goerr.V("server", serverCfg.Name),
			goerr.V("type", serverCfg.Type))
	}

	return client, nil
}

// createMCPLocalClient creates a local MCP client from local configuration
func (x *MCPConfig) createMCPLocalClient(ctx context.Context, localCfg MCPLocalConfig) (*mcp.Client, error) {
	if localCfg.Disabled {
		return nil, nil
	}

	if localCfg.Command == "" {
		return nil, goerr.New("command is required for local MCP server", goerr.V("server", localCfg.Name))
	}

	var opts []mcp.StdioOption
	if len(localCfg.Env) > 0 {
		opts = append(opts, mcp.WithEnvVars(localCfg.Env))
	}
	opts = append(opts, mcp.WithStdioClientInfo("warren", "1.0.0"))

	client, err := mcp.NewStdio(ctx, localCfg.Command, localCfg.Args, opts...)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create MCP local client",
			goerr.V("server", localCfg.Name))
	}

	return client, nil
}

// CreateMCPToolSets creates gollem tool sets from MCP clients
func (x *MCPConfig) CreateMCPToolSets(ctx context.Context) ([]gollem.ToolSet, error) {
	clients, err := x.CreateMCPClients(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create MCP clients")
	}

	if len(clients) == 0 {
		return nil, nil
	}

	toolSets := make([]gollem.ToolSet, len(clients))
	for i, client := range clients {
		// MCP Client already implements ToolSet interface (Specs and Run methods)
		toolSets[i] = client
	}

	return toolSets, nil
}

// IsConfigured returns true if MCP configuration file is specified
func (x *MCPConfig) IsConfigured() bool {
	return x.configFile != ""
}

// GetServerNames returns a list of configured server names (both servers and local)
func (x *MCPConfig) GetServerNames() []string {
	var names []string
	for _, server := range x.Servers {
		if !server.Disabled {
			names = append(names, server.Name)
		}
	}
	for _, local := range x.Local {
		if !local.Disabled {
			names = append(names, local.Name)
		}
	}
	return names
}
