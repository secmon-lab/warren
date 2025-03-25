package action_test

import (
	"context"
	"testing"

	"cloud.google.com/go/vertexai/genai"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/service/action"
)

func TestAction(t *testing.T) {
	cases := []struct {
		name     string
		actions  []interfaces.Action
		wantErr  bool
		errCheck func(t *testing.T, err error)
	}{
		{
			name: "success: action is configured correctly",
			actions: []interfaces.Action{
				&interfaces.ActionMock{
					NameFunc:      func() string { return "test" },
					ConfigureFunc: func(ctx context.Context) error { return nil },
					SpecsFunc:     func() []*genai.FunctionDeclaration { return nil },
				},
			},
		},
		{
			name: "error: empty action name",
			actions: []interfaces.Action{
				&interfaces.ActionMock{
					NameFunc:      func() string { return "" },
					ConfigureFunc: func(ctx context.Context) error { return nil },
					SpecsFunc:     func() []*genai.FunctionDeclaration { return nil },
				},
			},
			wantErr: true,
			errCheck: func(t *testing.T, err error) {
				gt.Error(t, err)
			},
		},
		{
			name: "error: duplicate function name",
			actions: []interfaces.Action{
				&interfaces.ActionMock{
					NameFunc:      func() string { return "test1" },
					ConfigureFunc: func(ctx context.Context) error { return nil },
					SpecsFunc:     func() []*genai.FunctionDeclaration { return []*genai.FunctionDeclaration{{Name: "func"}} },
				},
				&interfaces.ActionMock{
					NameFunc:      func() string { return "test2" },
					ConfigureFunc: func(ctx context.Context) error { return nil },
					SpecsFunc:     func() []*genai.FunctionDeclaration { return []*genai.FunctionDeclaration{{Name: "func"}} },
				},
			},
			wantErr: true,
			errCheck: func(t *testing.T, err error) {
				gt.Error(t, err).Contains("function name is conflicted")
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := action.New(t.Context(), tt.actions)
			if tt.wantErr {
				gt.Error(t, err)
				if tt.errCheck != nil {
					tt.errCheck(t, err)
				}
				return
			}
			gt.NoError(t, err)
			gt.NotNil(t, svc)
		})
	}
}
