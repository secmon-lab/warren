package intune_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	extintune "github.com/gollem-dev/tools/intune"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/tool/intune"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/secmon-lab/warren/pkg/utils/test"
	"github.com/urfave/cli/v3"
)

// setupTestServers creates an httptest token server and an httptest graph server,
// returning (graphURL, tokenURL). Callers configure the action via ConfigureWithOpts.
func setupTestServers(t *testing.T, handler http.HandlerFunc) (graphURL, tokenURL string) {
	t.Helper()

	graphServer := httptest.NewServer(handler)
	t.Cleanup(graphServer.Close)

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gt.Value(t, r.Method).Equal("POST")
		gt.Value(t, r.Header.Get("Content-Type")).Equal("application/x-www-form-urlencoded")

		resp := map[string]any{
			"access_token": "test-token",
			"expires_in":   3600,
			"token_type":   "Bearer",
		}
		w.Header().Set("Content-Type", "application/json")
		gt.NoError(t, json.NewEncoder(w).Encode(resp))
	}))
	t.Cleanup(tokenServer.Close)

	return graphServer.URL, tokenServer.URL
}

// configureAction sets the flag-bound fields on action then calls ConfigureWithOpts
// so that network calls go to the provided test servers.
func configureAction(t *testing.T, action *intune.Action, graphURL, tokenURL string) {
	t.Helper()
	cmd := cli.Command{
		Name:  "intune",
		Flags: action.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			return action.ConfigureWithOpts(tokenURL, graphURL)
		},
	}
	gt.NoError(t, cmd.Run(context.Background(), []string{
		"intune",
		"--intune-tenant-id", "test-tenant",
		"--intune-client-id", "test-client",
		"--intune-client-secret", "test-secret",
	}))
}

func TestIntune_DevicesByUser(t *testing.T) {
	graphURL, tokenURL := setupTestServers(t, func(w http.ResponseWriter, r *http.Request) {
		gt.Value(t, r.Header.Get("Authorization")).Equal("Bearer test-token")

		switch r.URL.Path {
		case "/deviceManagement/managedDevices":
			resp := map[string]any{
				"value": []any{
					map[string]any{
						"id":                "device-001",
						"userPrincipalName": "user@example.com",
						"userDisplayName":   "Test User",
						"deviceName":        "LAPTOP-001",
						"operatingSystem":   "Windows",
						"osVersion":         "10.0.19045",
						"complianceState":   "compliant",
						"isEncrypted":       true,
					},
				},
			}
			gt.NoError(t, json.NewEncoder(w).Encode(resp))

		case "/auditLogs/signIns":
			resp := map[string]any{
				"value": []any{
					map[string]any{
						"ipAddress":       "203.0.113.1",
						"createdDateTime": "2026-02-27T10:25:00Z",
						"clientAppUsed":   "Microsoft Edge",
					},
				},
			}
			gt.NoError(t, json.NewEncoder(w).Encode(resp))

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	var action intune.Action
	configureAction(t, &action, graphURL, tokenURL)

	resp, err := action.Run(context.Background(), "intune_devices_by_user", map[string]any{
		"user_principal_name": "user@example.com",
	})
	gt.NoError(t, err)

	devices, ok := resp["devices"].([]any)
	gt.Value(t, ok).Equal(true)
	gt.A(t, devices).Length(1)

	device := devices[0].(map[string]any)
	gt.Value(t, device["userPrincipalName"]).Equal("user@example.com")
	gt.Value(t, device["deviceName"]).Equal("LAPTOP-001")
	gt.Value(t, device["complianceState"]).Equal("compliant")
	gt.Value(t, device["isEncrypted"]).Equal(true)

	signIns, ok := resp["signInHistory"].([]any)
	gt.Value(t, ok).Equal(true)
	gt.A(t, signIns).Length(1)

	signIn := signIns[0].(map[string]any)
	gt.Value(t, signIn["ipAddress"]).Equal("203.0.113.1")

	gt.Value(t, resp["totalDevices"]).Equal(1)
}

func TestIntune_DevicesByUser_NoDevices(t *testing.T) {
	graphURL, tokenURL := setupTestServers(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/deviceManagement/managedDevices":
			gt.NoError(t, json.NewEncoder(w).Encode(map[string]any{"value": []any{}}))

		case "/auditLogs/signIns":
			gt.NoError(t, json.NewEncoder(w).Encode(map[string]any{"value": []any{}}))

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	var action intune.Action
	configureAction(t, &action, graphURL, tokenURL)

	resp, err := action.Run(context.Background(), "intune_devices_by_user", map[string]any{
		"user_principal_name": "unknown@example.com",
	})
	gt.NoError(t, err)

	devices, ok := resp["devices"].([]any)
	gt.Value(t, ok).Equal(true)
	gt.A(t, devices).Length(0)
	gt.Value(t, resp["totalDevices"]).Equal(0)
}

func TestIntune_DevicesByUser_MultipleDevices(t *testing.T) {
	graphURL, tokenURL := setupTestServers(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/deviceManagement/managedDevices":
			resp := map[string]any{
				"value": []any{
					map[string]any{
						"id":                "device-001",
						"userPrincipalName": "user@example.com",
						"deviceName":        "LAPTOP-001",
					},
					map[string]any{
						"id":                "device-002",
						"userPrincipalName": "user@example.com",
						"deviceName":        "PHONE-001",
					},
				},
			}
			gt.NoError(t, json.NewEncoder(w).Encode(resp))

		case "/auditLogs/signIns":
			gt.NoError(t, json.NewEncoder(w).Encode(map[string]any{"value": []any{}}))

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	var action intune.Action
	configureAction(t, &action, graphURL, tokenURL)

	resp, err := action.Run(context.Background(), "intune_devices_by_user", map[string]any{
		"user_principal_name": "user@example.com",
	})
	gt.NoError(t, err)

	devices, ok := resp["devices"].([]any)
	gt.Value(t, ok).Equal(true)
	gt.A(t, devices).Length(2)
	gt.Value(t, resp["totalDevices"]).Equal(2)

	d0 := devices[0].(map[string]any)
	gt.Value(t, d0["deviceName"]).Equal("LAPTOP-001")
	d1 := devices[1].(map[string]any)
	gt.Value(t, d1["deviceName"]).Equal("PHONE-001")
}

func TestIntune_DevicesByHostname(t *testing.T) {
	graphURL, tokenURL := setupTestServers(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/deviceManagement/managedDevices":
			resp := map[string]any{
				"value": []any{
					map[string]any{
						"id":                "device-001",
						"userPrincipalName": "user@example.com",
						"deviceName":        "LAPTOP-001",
						"operatingSystem":   "Windows",
					},
				},
			}
			gt.NoError(t, json.NewEncoder(w).Encode(resp))

		case "/auditLogs/signIns":
			resp := map[string]any{
				"value": []any{
					map[string]any{
						"ipAddress":       "10.0.0.1",
						"createdDateTime": "2026-02-27T09:00:00Z",
					},
				},
			}
			gt.NoError(t, json.NewEncoder(w).Encode(resp))

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	var action intune.Action
	configureAction(t, &action, graphURL, tokenURL)

	resp, err := action.Run(context.Background(), "intune_devices_by_hostname", map[string]any{
		"device_name": "LAPTOP-001",
	})
	gt.NoError(t, err)

	devices, ok := resp["devices"].([]any)
	gt.Value(t, ok).Equal(true)
	gt.A(t, devices).Length(1)

	device := devices[0].(map[string]any)
	gt.Value(t, device["deviceName"]).Equal("LAPTOP-001")
	gt.Value(t, device["operatingSystem"]).Equal("Windows")

	signIns, ok := resp["signInHistory"].([]any)
	gt.Value(t, ok).Equal(true)
	gt.A(t, signIns).Length(1)

	signIn := signIns[0].(map[string]any)
	gt.Value(t, signIn["ipAddress"]).Equal("10.0.0.1")

	gt.Value(t, resp["totalDevices"]).Equal(1)
}

func TestIntune_DevicesByHostname_NoDevices(t *testing.T) {
	graphURL, tokenURL := setupTestServers(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/deviceManagement/managedDevices" {
			gt.NoError(t, json.NewEncoder(w).Encode(map[string]any{"value": []any{}}))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	var action intune.Action
	configureAction(t, &action, graphURL, tokenURL)

	resp, err := action.Run(context.Background(), "intune_devices_by_hostname", map[string]any{
		"device_name": "UNKNOWN-HOST",
	})
	gt.NoError(t, err)

	devices, ok := resp["devices"].([]any)
	gt.Value(t, ok).Equal(true)
	gt.A(t, devices).Length(0)
	gt.Value(t, resp["totalDevices"]).Equal(0)

	// No sign-in logs should be fetched when no devices found
	gt.Value(t, resp["signInHistory"]).Equal([]any(nil))
}

func TestIntune_TokenRetryOn401(t *testing.T) {
	callCount := 0

	graphServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/deviceManagement/managedDevices":
			callCount++
			if callCount == 1 {
				w.WriteHeader(http.StatusUnauthorized)
				_, err := w.Write([]byte(`{"error":{"code":"InvalidAuthenticationToken"}}`))
				gt.NoError(t, err)
				return
			}
			resp := map[string]any{
				"value": []any{
					map[string]any{
						"id":         "device-001",
						"deviceName": "LAPTOP-001",
					},
				},
			}
			gt.NoError(t, json.NewEncoder(w).Encode(resp))

		case "/auditLogs/signIns":
			gt.NoError(t, json.NewEncoder(w).Encode(map[string]any{"value": []any{}}))

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(graphServer.Close)

	tokenCallCount := 0
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenCallCount++
		resp := map[string]any{
			"access_token": "new-token",
			"expires_in":   3600,
		}
		gt.NoError(t, json.NewEncoder(w).Encode(resp))
	}))
	t.Cleanup(tokenServer.Close)

	var action intune.Action
	cmd := cli.Command{
		Name:  "intune",
		Flags: action.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			return action.ConfigureWithOpts(tokenServer.URL, graphServer.URL)
		},
	}
	gt.NoError(t, cmd.Run(context.Background(), []string{
		"intune",
		"--intune-tenant-id", "test-tenant",
		"--intune-client-id", "test-client",
		"--intune-client-secret", "test-secret",
	}))

	resp, err := action.Run(context.Background(), "intune_devices_by_user", map[string]any{
		"user_principal_name": "user@example.com",
	})
	gt.NoError(t, err)

	devices, ok := resp["devices"].([]any)
	gt.Value(t, ok).Equal(true)
	gt.A(t, devices).Length(1)

	// Token should have been fetched at least twice (initial + retry)
	gt.Value(t, tokenCallCount >= 2).Equal(true)
}

func TestIntune_SignInLogFailure_Fallback(t *testing.T) {
	graphURL, tokenURL := setupTestServers(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/deviceManagement/managedDevices":
			resp := map[string]any{
				"value": []any{
					map[string]any{
						"id":                "device-001",
						"userPrincipalName": "user@example.com",
						"deviceName":        "LAPTOP-001",
					},
				},
			}
			gt.NoError(t, json.NewEncoder(w).Encode(resp))

		case "/auditLogs/signIns":
			w.WriteHeader(http.StatusForbidden)
			_, err := w.Write([]byte(`{"error":{"code":"Authorization_RequestDenied"}}`))
			gt.NoError(t, err)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	var action intune.Action
	configureAction(t, &action, graphURL, tokenURL)

	resp, err := action.Run(context.Background(), "intune_devices_by_user", map[string]any{
		"user_principal_name": "user@example.com",
	})
	gt.NoError(t, err)

	// Devices should still be returned even if sign-in logs fail
	devices, ok := resp["devices"].([]any)
	gt.Value(t, ok).Equal(true)
	gt.A(t, devices).Length(1)

	gt.Value(t, resp["totalDevices"]).Equal(1)
}

func TestIntune_TokenRequestFailure(t *testing.T) {
	graphServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(graphServer.Close)

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, err := w.Write([]byte(`{"error":"invalid_client","error_description":"Invalid client credentials"}`))
		gt.NoError(t, err)
	}))
	t.Cleanup(tokenServer.Close)

	var action intune.Action
	cmd := cli.Command{
		Name:  "intune",
		Flags: action.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			return action.ConfigureWithOpts(tokenServer.URL, graphServer.URL)
		},
	}
	gt.NoError(t, cmd.Run(context.Background(), []string{
		"intune",
		"--intune-tenant-id", "test-tenant",
		"--intune-client-id", "test-client",
		"--intune-client-secret", "test-secret",
	}))

	_, err := action.Run(context.Background(), "intune_devices_by_user", map[string]any{
		"user_principal_name": "user@example.com",
	})
	gt.Error(t, err)
}

func TestIntune_Specs(t *testing.T) {
	// Specs only needs a configured inner toolset; no network calls are made.
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	t.Cleanup(tokenServer.Close)
	graphServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	t.Cleanup(graphServer.Close)

	var action intune.Action
	cmd := cli.Command{
		Name:  "intune",
		Flags: action.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			return action.ConfigureWithOpts(tokenServer.URL, graphServer.URL)
		},
	}
	gt.NoError(t, cmd.Run(context.Background(), []string{
		"intune",
		"--intune-tenant-id", "test-tenant",
		"--intune-client-id", "test-client",
		"--intune-client-secret", "test-secret",
	}))

	specs, err := action.Specs(context.Background())
	gt.NoError(t, err)
	gt.A(t, specs).Length(2)

	for _, spec := range specs {
		switch spec.Name {
		case "intune_devices_by_user":
			gt.Map(t, spec.Parameters).HasKey("user_principal_name")
			gt.Value(t, spec.Parameters["user_principal_name"].Type).Equal("string")
		case "intune_devices_by_hostname":
			gt.Map(t, spec.Parameters).HasKey("device_name")
			gt.Value(t, spec.Parameters["device_name"].Type).Equal("string")
		default:
			t.Errorf("unexpected spec name: %s", spec.Name)
		}
	}
}

func TestIntune_Enabled(t *testing.T) {
	var action intune.Action

	cmd := cli.Command{
		Name:  "intune",
		Flags: action.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			gt.Equal(t, action.Configure(ctx), errutil.ErrActionUnavailable)
			return nil
		},
	}

	t.Setenv("WARREN_INTUNE_TENANT_ID", "")
	t.Setenv("WARREN_INTUNE_CLIENT_ID", "")
	t.Setenv("WARREN_INTUNE_CLIENT_SECRET", "")
	gt.NoError(t, cmd.Run(t.Context(), []string{"intune"}))
}

func TestIntune_InvalidFunctionName(t *testing.T) {
	graphURL, tokenURL := setupTestServers(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	var action intune.Action
	configureAction(t, &action, graphURL, tokenURL)

	_, err := action.Run(context.Background(), "intune_unknown_function", map[string]any{})
	gt.Error(t, err)
}

// TestIntune_OptionAppended verifies the --intune-base-url flag Action appends
// WithBaseURL carrying the provided value, by applying the accumulated options
// to a fresh external ToolSet and reading its unexported field.
func TestIntune_OptionAppended(t *testing.T) {
	var action intune.Action
	cmd := cli.Command{
		Name:   "intune",
		Flags:  action.Flags(),
		Action: func(context.Context, *cli.Command) error { return nil },
	}
	gt.NoError(t, cmd.Run(context.Background(), []string{
		"intune",
		"--intune-tenant-id", "t",
		"--intune-client-id", "c",
		"--intune-client-secret", "s",
		"--intune-base-url", "https://example.test/graph",
	}))

	var ts extintune.ToolSet
	for _, o := range action.Opts() {
		o(&ts)
	}
	gt.Value(t, test.PrivateField(t, &ts, "baseURL")).Equal("https://example.test/graph")
}

func TestIntune_Integration(t *testing.T) {
	vars := test.NewEnvVars(t, "TEST_INTUNE_TENANT_ID", "TEST_INTUNE_CLIENT_ID", "TEST_INTUNE_CLIENT_SECRET")

	var action intune.Action
	cmd := cli.Command{
		Name:  "intune",
		Flags: action.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			gt.NoError(t, action.Configure(ctx))

			resp, err := action.Run(ctx, "intune_devices_by_user", map[string]any{
				"user_principal_name": "test@example.com",
			})
			gt.NoError(t, err)
			gt.NotEqual(t, resp, nil)
			return nil
		},
	}

	gt.NoError(t, cmd.Run(context.Background(), []string{
		"intune",
		"--intune-tenant-id", vars.Get("TEST_INTUNE_TENANT_ID"),
		"--intune-client-id", vars.Get("TEST_INTUNE_CLIENT_ID"),
		"--intune-client-secret", vars.Get("TEST_INTUNE_CLIENT_SECRET"),
	}))
}
