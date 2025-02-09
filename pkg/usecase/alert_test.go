package usecase_test

import (
	"context"
	_ "embed"
	"testing"

	"cloud.google.com/go/vertexai/genai"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/interfaces"
	"github.com/secmon-lab/warren/pkg/mock"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/prompt"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/service"
	"github.com/secmon-lab/warren/pkg/usecase"
	"github.com/secmon-lab/warren/pkg/utils/test"
)

func TestFindSimilarAlert(t *testing.T) {
	ctx := context.Background()
	vars := test.NewEnvVars(t, "TEST_GEMINI_PROJECT_ID", "TEST_GEMINI_LOCATION")
	client, err := genai.NewClient(ctx, vars.Get("TEST_GEMINI_PROJECT_ID"), vars.Get("TEST_GEMINI_LOCATION"))
	gt.NoError(t, err)
	geminiModel := client.GenerativeModel("gemini-2.0-flash-exp")

	repo := repository.NewMemory()
	uc := usecase.New(func() interfaces.GenAIChatSession {
		return geminiModel.StartChat()
	}, usecase.WithRepository(repo))

	newAlert := model.NewAlert(ctx, "my_schema", map[string]any{"test": "test"}, model.PolicyAlert{
		Title: "test alert 1",
		Attrs: []model.Attribute{{Key: "test", Value: "test"}},
	})

	alert1 := model.NewAlert(ctx, "my_schema", map[string]any{"test": "test"}, model.PolicyAlert{
		Title: "test alert 0",
		Attrs: []model.Attribute{{Key: "test", Value: "test"}},
	})
	alert2 := model.NewAlert(ctx, "some_other_schema", map[string]any{"color": "red"}, model.PolicyAlert{
		Title: "this is different alert",
		Attrs: []model.Attribute{{Key: "color", Value: "red"}},
	})
	alert3 := model.NewAlert(ctx, "some_big_schema", map[string]any{"test": "test"}, model.PolicyAlert{
		Title: "more different alert",
		Attrs: []model.Attribute{{Key: "taste", Value: "sweet"}},
	})
	if err := repo.PutAlert(ctx, alert1); err != nil {
		t.Fatal("failed to put alert1:", err)
	}
	if err := repo.PutAlert(ctx, alert2); err != nil {
		t.Fatal("failed to put alert2:", err)
	}
	if err := repo.PutAlert(ctx, alert3); err != nil {
		t.Fatal("failed to put alert3:", err)
	}

	alert, err := uc.FindSimilarAlert(ctx, newAlert)
	gt.NoError(t, err)
	gt.NotEqual(t, alert, nil)
	gt.Equal(t, alert.ID, alert1.ID)
}

//go:embed testdata/guardduty.json
var guarddutyJSON []byte

func TestPlanAction(t *testing.T) {
	ctx := context.Background()
	vars := test.NewEnvVars(t, "TEST_GEMINI_PROJECT_ID", "TEST_GEMINI_LOCATION")
	client, err := genai.NewClient(ctx, vars.Get("TEST_GEMINI_PROJECT_ID"), vars.Get("TEST_GEMINI_LOCATION"))
	gt.NoError(t, err)
	geminiModel := client.GenerativeModel("gemini-2.0-flash-exp")
	geminiModel.GenerationConfig.ResponseMIMEType = "application/json"
	ssn := geminiModel.StartChat()

	actionSvc := service.NewActionService([]interfaces.Action{
		&mock.ActionMock{
			SpecFunc: func() model.ActionSpec {
				return model.ActionSpec{
					Name: "bigquery",
					Args: []model.ArgumentSpec{
						{
							Name:        "table_id",
							Type:        "string",
							Description: "The name of the BigQuery table to query",
							Required:    true,
							Choices: []model.ChoiceSpec{
								{
									Value:       "cloudtrail_logs",
									Description: "stored CloudTrail logs",
								},
								{
									Value:       "vpc_flow_logs",
									Description: "stored VPC flow logs",
								},
								{
									Value:       "s3_access_logs",
									Description: "stored S3 access logs",
								},
							},
						},
					},
				}
			},
		},
	})

	alert := model.NewAlert(ctx, "aws.guardduty", guarddutyJSON, model.PolicyAlert{
		Title: "Amazon GuardDuty finding",
	})

	prePrompt, err := prompt.BuildInitPrompt(alert)
	gt.NoError(t, err)

	resp, err := usecase.PlanAction(ctx, ssn, prePrompt, actionSvc)
	gt.NoError(t, err)
	gt.NotEqual(t, resp, nil)
	gt.Equal(t, resp.Action, "bigquery")
	gt.Equal(t, resp.Args, model.Arguments{"table_id": "vpc_flow_logs"})
}
