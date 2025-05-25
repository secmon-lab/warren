package base_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/tool/base"
)

func TestBase(t *testing.T) {
	testCases := []struct {
		name     string
		funcName string
		args     map[string]any
		wantResp map[string]any
		wantErr  bool
	}{
		{
			name:     "get alerts",
			funcName: "base.alerts.get",
			args: map[string]any{
				"limit":  float64(10),
				"offset": float64(0),
			},
			wantResp: map[string]any{
				"alerts": []string{},
				"count":  0,
				"offset": int64(0),
				"limit":  int64(10),
			},
			wantErr: false,
		},
		{
			name:     "search alerts",
			funcName: "base.alert.search",
			args: map[string]any{
				"path":   "status",
				"op":     "==",
				"value":  "open",
				"limit":  float64(10),
				"offset": float64(0),
			},
			wantResp: map[string]any{
				"alerts": map[string]any{},
				"count":  0,
				"offset": float64(0),
				"limit":  float64(10),
			},
			wantErr: false,
		},
		{
			name:     "invalid function name",
			funcName: "base.invalid",
			args:     map[string]any{},
			wantErr:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repo := repository.NewMemory()
			base := base.New(repo, []types.AlertID{}, map[string]string{}, types.NewTicketID())

			resp, err := base.Run(context.Background(), tc.funcName, tc.args)
			if tc.wantErr {
				gt.Error(t, err)
				return
			}

			gt.NoError(t, err)
			gt.NotEqual(t, resp, nil)
			gt.Value(t, resp).Equal(tc.wantResp)
		})
	}
}

func TestBase_Specs(t *testing.T) {
	repo := repository.NewMemory()
	base := base.New(repo, []types.AlertID{}, map[string]string{}, types.NewTicketID())

	specs, err := base.Specs(context.Background())
	gt.NoError(t, err)
	gt.A(t, specs).Length(4)

	for _, spec := range specs {
		switch spec.Name {
		case "base.alerts.get":
			gt.Value(t, spec.Description).Equal("Get a set of alerts with pagination support")
			gt.Map(t, spec.Parameters).HasKey("limit")
			gt.Map(t, spec.Parameters).HasKey("offset")
			gt.Value(t, spec.Parameters["limit"].Type).Equal("integer")
			gt.Value(t, spec.Parameters["offset"].Type).Equal("integer")

		case "base.alert.search":
			gt.Value(t, spec.Description).Equal("Search the alerts by the given query. You can specify the path as Firestore path, and the operation and value to filter the alerts.")
			gt.Map(t, spec.Parameters).HasKey("path")
			gt.Map(t, spec.Parameters).HasKey("op")
			gt.Map(t, spec.Parameters).HasKey("value")
			gt.Map(t, spec.Parameters).HasKey("limit")
			gt.Map(t, spec.Parameters).HasKey("offset")
			gt.Value(t, spec.Parameters["path"].Type).Equal("string")
			gt.Value(t, spec.Parameters["op"].Type).Equal("string")
			gt.Value(t, spec.Parameters["value"].Type).Equal("string")
			gt.Value(t, spec.Parameters["limit"].Type).Equal("integer")
			gt.Value(t, spec.Parameters["offset"].Type).Equal("integer")
			gt.A(t, spec.Required).Length(3)
			gt.A(t, spec.Required).Contains([]string{"path", "op", "value"})
		}
	}
}
