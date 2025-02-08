package bigquery_test

import (
	_ "embed"
)

/*
func TestActionConfig(t *testing.T) {
	var action bigquery.Action
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
*/
