package config_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/cli/config"
	"gopkg.in/yaml.v3"
)

func TestMCPServerConfig_HelperYAMLParse(t *testing.T) {
	input := `
servers:
  - name: "iap-server"
    type: "http"
    url: "https://iap.example.com/mcp"
    headers:
      Content-Type: "application/json"
    helper:
      command: "./scripts/iap-token.sh"
      args: ["https://iap.example.com"]
      env:
        GOOGLE_APPLICATION_CREDENTIALS: "/path/to/key.json"
`
	var cfg config.MCPConfig
	gt.NoError(t, yaml.Unmarshal([]byte(input), &cfg))

	gt.A(t, cfg.Servers).Length(1)
	srv := cfg.Servers[0]
	gt.Value(t, srv.Name).Equal("iap-server")
	gt.Value(t, srv.Type).Equal("http")
	gt.Value(t, srv.URL).Equal("https://iap.example.com/mcp")
	gt.Value(t, srv.Headers["Content-Type"]).Equal("application/json")

	gt.Value(t, srv.Helper).NotEqual(nil)
	gt.Value(t, srv.Helper.Command).Equal("./scripts/iap-token.sh")
	gt.A(t, srv.Helper.Args).Length(1)
	gt.Value(t, srv.Helper.Args[0]).Equal("https://iap.example.com")
	gt.Value(t, srv.Helper.Env["GOOGLE_APPLICATION_CREDENTIALS"]).Equal("/path/to/key.json")
}

func TestMCPServerConfig_NoHelper(t *testing.T) {
	input := `
servers:
  - name: "simple-server"
    type: "sse"
    url: "https://example.com/mcp"
    headers:
      Authorization: "Bearer static-token"
`
	var cfg config.MCPConfig
	gt.NoError(t, yaml.Unmarshal([]byte(input), &cfg))

	gt.A(t, cfg.Servers).Length(1)
	srv := cfg.Servers[0]
	gt.Value(t, srv.Name).Equal("simple-server")
	gt.Value(t, srv.Helper == nil).Equal(true)
	gt.Value(t, srv.Headers["Authorization"]).Equal("Bearer static-token")
}

func TestMCPLocalConfig_CommandConfigInline(t *testing.T) {
	input := `
local:
  - name: "fs-server"
    command: "npx"
    args: ["@modelcontextprotocol/server-filesystem", "/tmp"]
    env:
      NODE_ENV: "production"
`
	var cfg config.MCPConfig
	gt.NoError(t, yaml.Unmarshal([]byte(input), &cfg))

	gt.A(t, cfg.Local).Length(1)
	local := cfg.Local[0]
	gt.Value(t, local.Name).Equal("fs-server")
	gt.Value(t, local.Command).Equal("npx")
	gt.A(t, local.Args).Length(2)
	gt.Value(t, local.Args[0]).Equal("@modelcontextprotocol/server-filesystem")
	gt.Value(t, local.Args[1]).Equal("/tmp")
	gt.Value(t, local.Env["NODE_ENV"]).Equal("production")
	gt.Value(t, local.Disabled).Equal(false)
}

func TestMCPLocalConfig_DisabledField(t *testing.T) {
	input := `
local:
  - name: "disabled-server"
    command: "npx"
    disabled: true
`
	var cfg config.MCPConfig
	gt.NoError(t, yaml.Unmarshal([]byte(input), &cfg))

	gt.A(t, cfg.Local).Length(1)
	gt.Value(t, cfg.Local[0].Disabled).Equal(true)
}
