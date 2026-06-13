package whois_test

import (
	"context"
	"os"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/tool/whois"
)

func TestWhois_ConfigureReturnsNil(t *testing.T) {
	action := &whois.Action{}
	err := action.Configure(context.Background())
	gt.NoError(t, err)
}

func TestWhois_Specs(t *testing.T) {
	action := &whois.Action{}
	gt.NoError(t, action.Configure(context.Background()))

	specs, err := action.Specs(context.Background())
	gt.NoError(t, err)
	gt.A(t, specs).Length(2)

	var foundDomain, foundIP bool
	for _, spec := range specs {
		switch spec.Name {
		case "whois_domain":
			foundDomain = true
			gt.Map(t, spec.Parameters).HasKey("target")
			gt.Value(t, spec.Parameters["target"].Type).Equal("string")
		case "whois_ip":
			foundIP = true
			gt.Map(t, spec.Parameters).HasKey("target")
			gt.Value(t, spec.Parameters["target"].Type).Equal("string")
		}
	}
	gt.True(t, foundDomain)
	gt.True(t, foundIP)
}

func TestWhois_SpecsWithoutConfigure(t *testing.T) {
	action := &whois.Action{}
	_, err := action.Specs(context.Background())
	gt.Error(t, err)
}

func TestWhois_RunWithoutConfigure(t *testing.T) {
	action := &whois.Action{}
	_, err := action.Run(context.Background(), "whois_domain", map[string]any{"target": "example.com"})
	gt.Error(t, err)
}

func TestWhois_Flags(t *testing.T) {
	action := &whois.Action{}
	flags := action.Flags()
	gt.A(t, flags).Length(0)
}

func TestWhois_ID(t *testing.T) {
	action := &whois.Action{}
	gt.Value(t, action.ID()).Equal("whois")
}

// TestWhois_RunLive performs live WHOIS lookups when TEST_WHOIS_DOMAIN or
// TEST_WHOIS_IP env vars are set. Skipped if both are absent (missing env vars
// are the only acceptable skip reason per project rules).
func TestWhois_RunLive(t *testing.T) {
	domain := os.Getenv("TEST_WHOIS_DOMAIN")
	ip := os.Getenv("TEST_WHOIS_IP")
	if domain == "" && ip == "" {
		t.Skip("TEST_WHOIS_DOMAIN and TEST_WHOIS_IP are not set")
	}

	action := &whois.Action{}
	gt.NoError(t, action.Configure(context.Background()))

	if domain != "" {
		resp, err := action.Run(context.Background(), "whois_domain", map[string]any{"target": domain})
		gt.NoError(t, err)
		gt.NotEqual(t, resp, nil)
		result, ok := resp["result"].(string)
		gt.True(t, ok)
		gt.True(t, len(result) > 0)
	}

	if ip != "" {
		resp, err := action.Run(context.Background(), "whois_ip", map[string]any{"target": ip})
		gt.NoError(t, err)
		gt.NotEqual(t, resp, nil)
		result, ok := resp["result"].(string)
		gt.True(t, ok)
		gt.True(t, len(result) > 0)
	}
}
