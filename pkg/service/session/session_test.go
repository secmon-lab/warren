package session_test

import (
	"context"
	"testing"

	"cloud.google.com/go/vertexai/genai"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/adapter/gemini"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	action_model "github.com/secmon-lab/warren/pkg/domain/model/action"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	session_model "github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/service/action"
	"github.com/secmon-lab/warren/pkg/service/session"
)

func TestSessionChat(t *testing.T) {
	ctx := t.Context()
	target := alert.New(ctx, "test-schema", alert.Metadata{
		Title:       "suspicious access",
		Description: "suspicious access to the system",
		Data: map[string]any{
			"user":      "blue",
			"remote":    "103.243.25.70",
			"timestamp": "2021-01-01 12:00:00",
		},
	})
	repo := repository.NewMemory()
	gt.NoError(t, repo.PutAlert(ctx, target)).Required()

	ssn := session_model.New(ctx, &slack.User{}, &slack.Thread{}, []types.AlertID{target.ID})
	geminiClient := gemini.NewTestClient(t)

	threatInfoMock := &interfaces.ActionMock{
		NameFunc: func() string {
			return "threat_info"
		},
		SpecsFunc: func() []*genai.FunctionDeclaration {
			return []*genai.FunctionDeclaration{
				{
					Name:        "threat_info.ipv4",
					Description: "Get threat information of the given IPv4 address",
					Parameters: &genai.Schema{
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"addr": {Type: genai.TypeString},
						},
						Required: []string{"addr"},
					},
				},
			}
		},
		ExecuteFunc: func(ctx context.Context, name string, args map[string]any) (*action_model.Result, error) {
			return &action_model.Result{
				Name: "threat_info.ipv4",
				Data: map[string]any{
					"message": "Threat info about 103.243.25.70",
					"rows": []string{
						"IP reputation: malicious",
						"Country: United States",
						"ASN: AS15169",
					},
				},
			}, nil
		},
	}

	logServiceMock := &interfaces.ActionMock{
		NameFunc: func() string {
			return "log"
		},
		SpecsFunc: func() []*genai.FunctionDeclaration {
			return []*genai.FunctionDeclaration{
				{
					Name:        "log.user",
					Description: "Get user logs",
					Parameters: &genai.Schema{
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"user": {
								Type:        genai.TypeString,
								Description: "User name",
							},
						},
						Required: []string{"user"},
					},
				},
			}
		},
		ExecuteFunc: func(ctx context.Context, name string, args map[string]any) (*action_model.Result, error) {
			return &action_model.Result{
				Name: "log.user",
				Data: map[string]any{
					"message": "User logs",
					"rows": []string{
						"2021-01-01 12:00:00: login from 10.0.2.3",
						"2021-01-01 12:00:01: login from 103.243.25.70",
					},
				},
			}, nil
		},
	}

	actionService, err := action.New(ctx, []interfaces.Action{threatInfoMock, logServiceMock})
	gt.NoError(t, err)

	svc := session.New(repo, geminiClient, actionService, ssn)

	err = svc.Chat(ctx, "How about risk of the alert?")
	gt.NoError(t, err)
}
