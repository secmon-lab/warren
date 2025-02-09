package urlscan_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/action/urlscan"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/utils/test"
	"github.com/urfave/cli/v3"
)

func TestURLScan(t *testing.T) {
	testCases := []struct {
		name            string
		args            model.Arguments
		scanResp        string
		resultResponses []string
		wantResp        string
		wantErr         bool
	}{
		{
			name: "valid response",
			args: model.Arguments{
				"url": "https://example.com",
			},
			scanResp: `{
				"uuid": "test-uuid",
				"message": "Submission successful",
				"result": "https://urlscan.io/result/test-uuid"
			}`,
			resultResponses: []string{
				`{"data": "test result"}`,
			},
			wantResp: `{"data": "test result"}`,
			wantErr:  false,
		},
		{
			name:    "missing url",
			args:    model.Arguments{},
			wantErr: true,
		},
		{
			name: "scan error response",
			args: model.Arguments{
				"url": "https://example.com",
			},
			scanResp: `{"error": "invalid request"}`,
			wantErr:  true,
		},
		{
			name: "result not ready",
			args: model.Arguments{
				"url": "https://example.com",
			},
			scanResp: `{
				"uuid": "test-uuid",
				"message": "Submission successful",
				"result": "https://urlscan.io/result/test-uuid"
			}`,
			resultResponses: []string{
				"", "", "", "", "", // 5 not found responses
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resultCallCount := 0
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

				if r.Method == "POST" {
					gt.Value(t, r.Header.Get("API-Key")).Equal("test-key")
					gt.Value(t, r.Header.Get("Content-Type")).Equal("application/json")

					if tc.scanResp == "" {
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					w.WriteHeader(http.StatusOK)
					if _, err := w.Write([]byte(tc.scanResp)); err != nil {
						t.Fatal("failed to write response:", err)
					}
					return
				}

				if r.Method == "GET" {
					if resultCallCount >= len(tc.resultResponses) {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					resp := tc.resultResponses[resultCallCount]
					resultCallCount++
					if resp == "" {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					w.WriteHeader(http.StatusOK)
					if _, err := w.Write([]byte(resp)); err != nil {
						t.Fatal("failed to write response:", err)
					}
					return
				}
			}))
			defer ts.Close()

			var action urlscan.Action
			cmd := cli.Command{
				Name:  "urlscan",
				Flags: action.Flags(),
				Action: func(ctx context.Context, c *cli.Command) error {
					resp, err := action.Execute(ctx, nil, nil, tc.args)
					if tc.wantErr {
						gt.Error(t, err)
						return nil
					}

					gt.NoError(t, err)
					gt.Value(t, resp.Data).Equal(tc.wantResp)
					return nil
				},
			}

			gt.NoError(t, cmd.Run(context.Background(), []string{
				"urlscan",
				"--urlscan-api-key", "test-key",
				"--urlscan-base-url", ts.URL,
				"--urlscan-backoff", "0.01s",
			}))
		})
	}
}

func TestURLScan_Enabled(t *testing.T) {
	var action urlscan.Action

	t.Setenv("WARREN_URLSCAN_API_KEY", "")
	cmd := cli.Command{
		Name:  "urlscan",
		Flags: action.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			gt.Equal(t, action.Configure(ctx), model.ErrActionUnavailable)
			return nil
		},
	}

	gt.NoError(t, cmd.Run(context.Background(), []string{
		"urlscan",
		// "--urlscan-api-key", "test-key",
		"--urlscan-base-url", "https://urlscan.io",
		"--urlscan-backoff", "0.01s",
	}))
}

func TestSendRequest(t *testing.T) {
	var action urlscan.Action

	vars := test.NewEnvVars(t, "TEST_URLSCAN_API_KEY")
	cmd := cli.Command{
		Name:  "urlscan",
		Flags: action.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			resp, err := action.Execute(ctx, nil, nil, model.Arguments{"url": "https://example.com"})
			gt.NoError(t, err)
			gt.NotEqual(t, resp, nil)
			gt.Equal(t, resp.Type, model.ActionResultTypeJSON)
			gt.String(t, resp.Data).Contains("https://example.com")
			return nil
		},
	}

	gt.NoError(t, cmd.Run(context.Background(), []string{
		"urlscan",
		"--urlscan-api-key", vars.Get("TEST_URLSCAN_API_KEY"),
	}))
}
