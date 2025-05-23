package usecase_test

import (
	_ "embed"
)

/*
type geminiClient struct {
	model *genai.GenerativeModel
}

func (c *geminiClient) StartChat() interfaces.LLMSession {
	return c.model.StartChat()
}

func (c *geminiClient) SendMessage(ctx context.Context, msg ...genai.Part) (*genai.GenerateContentResponse, error) {
	return c.model.GenerateContent(ctx, msg...)
}

func genGeminiClient(t *testing.T) *geminiClient {
	vars := test.NewEnvVars(t, "TEST_GEMINI_PROJECT_ID", "TEST_GEMINI_LOCATION")
	client, err := genai.NewClient(t.Context(), vars.Get("TEST_GEMINI_PROJECT_ID"), vars.Get("TEST_GEMINI_LOCATION"))
	gt.NoError(t, err)
	geminiModel := client.GenerativeModel("gemini-2.0-flash-exp")
	geminiModel.GenerationConfig.ResponseMIMEType = "application/json"
	return &geminiClient{model: geminiModel}
}

func TestFindSimilarAlert(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	uc := usecase.New(usecase.WithLLMClient(genGeminiClient(t)), usecase.WithRepository(repo))

	newAlert := alert.NewAlert(ctx, "my_schema", model.PolicyAlert{
		Title: "test alert 1",
		Attrs: []model.Attribute{{Key: "test", Value: "test"}},
		Data:  map[string]any{"test": "test"},
	})

	alert1 := alert.NewAlert(ctx, "my_schema", model.PolicyAlert{
		Title: "test alert 0",
		Attrs: []model.Attribute{{Key: "test", Value: "test"}},
		Data:  map[string]any{"test": "test"},
	})
	alert2 := model.NewAlert(ctx, "some_other_schema", model.PolicyAlert{
		Title: "this is different alert",
		Attrs: []model.Attribute{{Key: "color", Value: "red"}},
		Data:  map[string]any{"test": "test"},
	})
	alert3 := model.NewAlert(ctx, "some_big_schema", model.PolicyAlert{
		Title: "more different alert",
		Attrs: []model.Attribute{{Key: "taste", Value: "sweet"}},
		Data:  map[string]any{"test": "test"},
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
			SpecFunc: func() action.ActionSpec {
				return action.ActionSpec{
					Name: "bigquery",
					Args: []action.ArgumentSpec{
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

	alert := model.NewAlert(ctx, "aws.guardduty", model.PolicyAlert{
		Title: "Amazon GuardDuty finding",
		Data:  guarddutyJSON,
	})

	prePrompt, err := prompt.BuildInitPrompt(ctx, alert, 3)
	gt.NoError(t, err)

	resp, err := usecase.PlanAction(ctx, ssn, prePrompt, actionSvc)
	gt.NoError(t, err)
	gt.NotEqual(t, resp, nil)
	gt.Equal(t, resp.Action, "bigquery")
	gt.Equal(t, resp.Args, action.Arguments{"table_id": "vpc_flow_logs"})
}

func TestGenerateAlertMetadata(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	uc := usecase.New(usecase.WithLLMClient(genGeminiClient(t)), usecase.WithRepository(repo))

	alert := model.NewAlert(ctx, "aws.guardduty", model.PolicyAlert{
		Title: "Amazon GuardDuty finding",
		Data:  guarddutyJSON,
		Attrs: []model.Attribute{{Key: "test", Value: "test"}},
	})

	newAlert, err := uc.GenerateAlertMetadata(ctx, alert)
	gt.NoError(t, err)
	// Title is not changed
	gt.Equal(t, newAlert.Title, alert.Title)
	// Description is not empty
	gt.NotEqual(t, newAlert.Description, "")
	// Attributes are not empty
	gt.A(t, newAlert.Attributes).Longer(2)
}
*/
