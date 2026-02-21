package whois_test

import (
	"context"
	"errors"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/tool/whois"
)

func TestWhois_Domain(t *testing.T) {
	action := &whois.Action{}
	action.SetQueryFunc(func(_ context.Context, target string) (string, error) {
		gt.Value(t, target).Equal("example.com")
		return "Domain Name: EXAMPLE.COM\nRegistrar: Example Registrar\n", nil
	})

	resp, err := action.Run(context.Background(), "whois_domain", map[string]any{
		"target": "example.com",
	})
	gt.NoError(t, err)
	gt.NotEqual(t, resp, nil)
	gt.Value(t, resp["result"]).Equal("Domain Name: EXAMPLE.COM\nRegistrar: Example Registrar\n")
}

func TestWhois_IP(t *testing.T) {
	action := &whois.Action{}
	action.SetQueryFunc(func(_ context.Context, target string) (string, error) {
		gt.Value(t, target).Equal("8.8.8.8")
		return "NetRange: 8.8.8.0 - 8.8.8.255\nOrganization: Google LLC\n", nil
	})

	resp, err := action.Run(context.Background(), "whois_ip", map[string]any{
		"target": "8.8.8.8",
	})
	gt.NoError(t, err)
	gt.NotEqual(t, resp, nil)
	gt.Value(t, resp["result"]).Equal("NetRange: 8.8.8.0 - 8.8.8.255\nOrganization: Google LLC\n")
}

func TestWhois_IPv6(t *testing.T) {
	action := &whois.Action{}
	action.SetQueryFunc(func(_ context.Context, target string) (string, error) {
		gt.Value(t, target).Equal("2001:4860:4860::8888")
		return "inet6num: 2001:4860::/32\nOrganisation: Google LLC\n", nil
	})

	resp, err := action.Run(context.Background(), "whois_ip", map[string]any{
		"target": "2001:4860:4860::8888",
	})
	gt.NoError(t, err)
	gt.NotEqual(t, resp, nil)
	gt.Value(t, resp["result"]).Equal("inet6num: 2001:4860::/32\nOrganisation: Google LLC\n")
}

func TestWhois_EmptyTarget(t *testing.T) {
	action := &whois.Action{}

	_, err := action.Run(context.Background(), "whois_domain", map[string]any{
		"target": "",
	})
	gt.Error(t, err)
}

func TestWhois_MissingTarget(t *testing.T) {
	action := &whois.Action{}

	_, err := action.Run(context.Background(), "whois_domain", map[string]any{})
	gt.Error(t, err)
}

func TestWhois_QueryError(t *testing.T) {
	action := &whois.Action{}
	action.SetQueryFunc(func(_ context.Context, _ string) (string, error) {
		return "", errors.New("connection refused")
	})

	_, err := action.Run(context.Background(), "whois_domain", map[string]any{
		"target": "example.com",
	})
	gt.Error(t, err)
}

func TestWhois_InvalidFunctionName(t *testing.T) {
	action := &whois.Action{}

	_, err := action.Run(context.Background(), "unknown_func", map[string]any{
		"target": "example.com",
	})
	gt.Error(t, err)
}

func TestWhois_Configure(t *testing.T) {
	action := &whois.Action{}
	err := action.Configure(context.Background())
	gt.NoError(t, err)
}

func TestWhois_Specs(t *testing.T) {
	action := &whois.Action{}
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

func TestWhois_Flags(t *testing.T) {
	action := &whois.Action{}
	flags := action.Flags()
	gt.A(t, flags).Length(0)
}

func TestWhois_Name(t *testing.T) {
	action := &whois.Action{}
	gt.Value(t, action.Name()).Equal("whois")
}
