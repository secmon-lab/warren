package config

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/mcp"
	helpertransport "github.com/secmon-lab/warren/pkg/service/mcp"
	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
)

const Version = "1.0.0"

// CommandConfig represents a common command execution configuration
// shared between local MCP servers and credential helpers.
type CommandConfig struct {
	Command string            `yaml:"command"`
	Args    []string          `yaml:"args,omitempty"`
	Env     map[string]string `yaml:"env,omitempty"`
}

// MCPServerConfig represents configuration for a remote MCP server
type MCPServerConfig struct {
	Name     string            `yaml:"name"`
	Type     string            `yaml:"type"` // "sse" or "http"
	URL      string            `yaml:"url"`
	Headers  map[string]string `yaml:"headers,omitempty"`
	Helper   *CommandConfig    `yaml:"helper,omitempty"`
	Disabled bool              `yaml:"disabled,omitempty"`
}

// MCPLocalConfig represents configuration for a local MCP server
type MCPLocalConfig struct {
	Name          string `yaml:"name"`
	CommandConfig `yaml:",inline"`
	Disabled      bool `yaml:"disabled,omitempty"`
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
		if serverCfg.Helper != nil {
			httpClient := buildHelperHTTPClient(serverCfg.Helper, serverCfg.Headers)
			opts = append(opts, mcp.WithSSEClient(httpClient))
		} else if len(serverCfg.Headers) > 0 {
			opts = append(opts, mcp.WithSSEHeaders(serverCfg.Headers))
		}
		opts = append(opts, mcp.WithSSEClientInfo("warren", Version))

		client, err = mcp.NewSSE(ctx, serverCfg.URL, opts...)

	case "http":
		if serverCfg.URL == "" {
			return nil, goerr.New("url is required for HTTP MCP server", goerr.V("server", serverCfg.Name))
		}

		var opts []mcp.StreamableHTTPOption
		if serverCfg.Helper != nil {
			httpClient := buildHelperHTTPClient(serverCfg.Helper, serverCfg.Headers)
			opts = append(opts, mcp.WithStreamableHTTPClient(httpClient))
		} else if len(serverCfg.Headers) > 0 {
			opts = append(opts, mcp.WithStreamableHTTPHeaders(serverCfg.Headers))
		}
		opts = append(opts, mcp.WithStreamableHTTPClientInfo("warren", Version))

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

// buildHelperHTTPClient creates an *http.Client with HelperTransport
// that dynamically injects headers from a credential helper command.
func buildHelperHTTPClient(helper *CommandConfig, staticHeaders map[string]string) *http.Client {
	cfg := helpertransport.HelperConfig{
		Command: helper.Command,
		Args:    helper.Args,
		Env:     helper.Env,
	}

	transport := helpertransport.NewHelperTransport(cfg, staticHeaders, nil)
	return &http.Client{Transport: transport}
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
		envVars := make([]string, 0, len(localCfg.Env))
		for key, value := range localCfg.Env {
			envVars = append(envVars, key+"="+value)
		}
		opts = append(opts, mcp.WithEnvVars(envVars))
	}
	opts = append(opts, mcp.WithStdioClientInfo("warren", Version))

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
	names := make([]string, 0, len(x.Servers)+len(x.Local))
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
