package bigquery_test

import (
	"context"
	_ "embed"
	"testing"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/vertexai/genai"
	"github.com/m-mizutani/gt"
	bq "github.com/secmon-lab/warren/pkg/action/bigquery"
	"github.com/secmon-lab/warren/pkg/domain/model"
	"github.com/secmon-lab/warren/pkg/mock"
	"github.com/urfave/cli/v3"
)

func TestActionConfig(t *testing.T) {
	var action bq.Action
	app := cli.Command{
		Name:  "bigquery",
		Flags: action.Flags(),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			gt.NoError(t, action.Configure(ctx))
			return nil
		},
	}

	gt.NoError(t, app.Run(context.Background(), []string{
		"warren",
		"--bigquery-project-id", "my-project",
		"--bigquery-config", "testdata/config.yml",
	}))

	gt.Equal(t, action.ByteLimit(), 100*1000*1000)
	gt.Equal(t, action.LimitRows(), 256)
	gt.Equal(t, len(action.Tables()), 2)
}

type bqClientMock struct {
	QueryFunc       func(ctx context.Context, query string, out func(v map[string]bigquery.Value) error) error
	DryRunFunc      func(ctx context.Context, query string) (*bigquery.JobStatus, error)
	GetMetadataFunc func(ctx context.Context, datasetID, tableID string) (*bigquery.TableMetadata, error)
	CloseFunc       func() error
}

func newBQClientMock() *bqClientMock {
	return &bqClientMock{
		QueryFunc: func(ctx context.Context, query string, out func(v map[string]bigquery.Value) error) error {
			return out(map[string]bigquery.Value{"test": "data"})
		},
		DryRunFunc: func(ctx context.Context, query string) (*bigquery.JobStatus, error) {
			return &bigquery.JobStatus{
				Statistics: &bigquery.JobStatistics{
					TotalBytesProcessed: 1000,
				},
			}, nil
		},
		GetMetadataFunc: func(ctx context.Context, datasetID, tableID string) (*bigquery.TableMetadata, error) {
			return &bigquery.TableMetadata{
				Schema: bigquery.Schema{
					{
						Name:     "test",
						Required: true,
					},
				},
			}, nil
		},
		CloseFunc: func() error { return nil },
	}
}

func (x *bqClientMock) Query(ctx context.Context, query string, out func(v map[string]bigquery.Value) error) error {
	return x.QueryFunc(ctx, query, out)
}

func (x *bqClientMock) DryRun(ctx context.Context, query string) (*bigquery.JobStatus, error) {
	return x.DryRunFunc(ctx, query)
}

func (x *bqClientMock) GetMetadata(ctx context.Context, datasetID, tableID string) (*bigquery.TableMetadata, error) {
	return x.GetMetadataFunc(ctx, datasetID, tableID)
}

func (x *bqClientMock) Close() error {
	return x.CloseFunc()
}

type ssnMock struct {
	resp string
}

func (x *ssnMock) SendMessage(ctx context.Context, msg ...genai.Part) (*genai.GenerateContentResponse, error) {
	return &genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{{
			Content: &genai.Content{
				Parts: []genai.Part{
					genai.Text(x.resp),
				},
			},
		}},
	}, nil
}

func TestActionExecute(t *testing.T) {
	cases := map[string]struct {
		client  *bqClientMock
		args    model.Arguments
		want    *model.ActionResult
		wantErr bool
		ssn     *ssnMock
	}{
		"basic query": {
			client: &bqClientMock{
				GetMetadataFunc: func(ctx context.Context, datasetID, tableID string) (*bigquery.TableMetadata, error) {
					return &bigquery.TableMetadata{}, nil
				},
				DryRunFunc: func(ctx context.Context, query string) (*bigquery.JobStatus, error) {
					return &bigquery.JobStatus{
						Statistics: &bigquery.JobStatistics{
							TotalBytesProcessed: 1000,
						},
					}, nil
				},
				QueryFunc: func(ctx context.Context, query string, out func(v map[string]bigquery.Value) error) error {
					return out(map[string]bigquery.Value{"test": "data"})
				},
				CloseFunc: func() error { return nil },
			},
			args: model.Arguments{
				"table_id": "my-project.github.audit_logs_v1",
			},
			want: &model.ActionResult{
				Type:    model.ActionResultTypeJSON,
				Data:    "{\n  \"test\": \"data\"\n}\n",
				Message: "Retrieved data from my-project.github.audit_logs_v1",
			},
			ssn: &ssnMock{
				resp: `{"query":"SELECT * FROM ` + "`my-project.github.audit_logs_v1`" + ` LIMIT 1000"}`,
			},
		},
		"invalid table id": {
			args: model.Arguments{
				"table_id": "invalid",
			},
			wantErr: true,
		},
		"no query result": {
			args: model.Arguments{
				"table_id": "my-project.github.audit_logs_v1",
			},
			ssn: &ssnMock{
				resp: "",
			},
			wantErr: true,
		},
		"no table id": {
			args:    model.Arguments{},
			wantErr: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var action bq.Action
			if tc.client != nil {
				action.SetBQClientFactory(func(ctx context.Context, projectID, impersonationSA string) (bq.BigQueryClient, error) {
					return tc.client, nil
				})
			}
			threadMock := &mock.SlackThreadServiceMock{
				ReplyFunc: func(ctx context.Context, msg string) {
					// do nothing
				},
				AttachFileFunc: func(ctx context.Context, comment, filename string, content []byte) error {
					return nil
				},
			}
			cmd := &cli.Command{
				Name:  "bigquery",
				Flags: action.Flags(),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					gt.NoError(t, action.Configure(ctx))
					got, err := action.Execute(context.Background(), threadMock, tc.ssn, tc.args)
					if tc.wantErr {
						gt.Error(t, err)
						return nil
					}
					gt.NoError(t, err)
					gt.Equal(t, tc.want, got)
					return nil
				},
			}

			gt.NoError(t, cmd.Run(context.Background(), []string{
				"warren",
				"--bigquery-project-id", "my-project",
				"--bigquery-config", "testdata/config.yml",
			}))
		})
	}
}

func TestActionExecuteWithLimit(t *testing.T) {
	dryRunCount := 0
	client := newBQClientMock()
	client.DryRunFunc = func(ctx context.Context, query string) (*bigquery.JobStatus, error) {
		dryRunCount++
		result := 1000 * 1000 * 1000
		if dryRunCount > 1 {
			result = 10 * 1000 * 1000
		}
		return &bigquery.JobStatus{
			Statistics: &bigquery.JobStatistics{
				TotalBytesProcessed: int64(result),
			},
		}, nil
	}
	args := model.Arguments{
		"table_id": "my-project.github.audit_logs_v1",
	}
	ssn := &ssnMock{
		resp: `{"query":"SELECT * FROM ` + "`my-project.github.audit_logs_v1`" + ` LIMIT 1000"}`,
	}

	var action bq.Action
	action.SetBQClientFactory(func(ctx context.Context, projectID, impersonationSA string) (bq.BigQueryClient, error) {
		return client, nil
	})

	threadMock := &mock.SlackThreadServiceMock{
		ReplyFunc: func(ctx context.Context, msg string) {
			// do nothing
		},
		AttachFileFunc: func(ctx context.Context, comment, filename string, content []byte) error {
			return nil
		},
	}
	cmd := &cli.Command{
		Name:  "bigquery",
		Flags: action.Flags(),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			gt.NoError(t, action.Configure(ctx))
			got, err := action.Execute(context.Background(), threadMock, ssn, args)
			gt.NoError(t, err)
			gt.Equal(t, got.Type, model.ActionResultTypeJSON)
			gt.Equal(t, got.Data, "{\n  \"test\": \"data\"\n}\n")
			return nil
		},
	}
	gt.NoError(t, cmd.Run(context.Background(), []string{
		"warren",
		"--bigquery-project-id", "my-project",
		"--bigquery-config", "testdata/config.yml",
	}))

	gt.Equal(t, dryRunCount, 2)
}

func TestGenerateQuery(t *testing.T) {
	schema := bigquery.Schema{
		{
			Name:     "name",
			Type:     bigquery.StringFieldType,
			Required: true,
		},
		{
			Name:     "age",
			Type:     bigquery.IntegerFieldType,
			Required: false,
		},
		{
			Name:     "created_at",
			Type:     bigquery.TimestampFieldType,
			Required: false,
		},
	}

	query, err := bq.GenerateQuery("my-project.github.audit_logs_v1", schema, 1000)
	gt.NoError(t, err)
	t.Log(query)
}
