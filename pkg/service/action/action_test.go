package action_test

import (
	"context"
	"testing"

	"cloud.google.com/go/vertexai/genai"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/mock"
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
				&mock.ActionMock{
					NameFunc:      func() string { return "test" },
					ConfigureFunc: func(ctx context.Context) error { return nil },
					ToolsFunc:     func() []*genai.FunctionDeclaration { return nil },
				},
			},
		},
		{
			name: "error: empty action name",
			actions: []interfaces.Action{
				&mock.ActionMock{
					NameFunc:      func() string { return "" },
					ConfigureFunc: func(ctx context.Context) error { return nil },
					ToolsFunc:     func() []*genai.FunctionDeclaration { return nil },
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
				&mock.ActionMock{
					NameFunc:      func() string { return "test1" },
					ConfigureFunc: func(ctx context.Context) error { return nil },
					ToolsFunc:     func() []*genai.FunctionDeclaration { return []*genai.FunctionDeclaration{{Name: "func"}} },
				},
				&mock.ActionMock{
					NameFunc:      func() string { return "test2" },
					ConfigureFunc: func(ctx context.Context) error { return nil },
					ToolsFunc:     func() []*genai.FunctionDeclaration { return []*genai.FunctionDeclaration{{Name: "func"}} },
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

func TestServiceWith(t *testing.T) {
	cases := []struct {
		name      string
		base      []interfaces.Action
		new       []interfaces.Action
		wantErr   bool
		errCheck  func(t *testing.T, err error)
		specCheck func(t *testing.T, specs []*genai.FunctionDeclaration)
	}{
		{
			name: "success: add new action",
			base: []interfaces.Action{
				&mock.ActionMock{
					NameFunc:      func() string { return "base" },
					ConfigureFunc: func(ctx context.Context) error { return nil },
					ToolsFunc: func() []*genai.FunctionDeclaration {
						return []*genai.FunctionDeclaration{{Name: "base.func"}}
					},
				},
			},
			new: []interfaces.Action{
				&mock.ActionMock{
					NameFunc:      func() string { return "new" },
					ConfigureFunc: func(ctx context.Context) error { return nil },
					ToolsFunc: func() []*genai.FunctionDeclaration {
						return []*genai.FunctionDeclaration{{Name: "new.func"}}
					},
				},
			},
			specCheck: func(t *testing.T, specs []*genai.FunctionDeclaration) {
				gt.Equal(t, 2, len(specs))
				gt.Equal(t, "base.func", specs[0].Name)
				gt.Equal(t, "new.func", specs[1].Name)
			},
		},
		{
			name: "error: duplicate function name between base and new",
			base: []interfaces.Action{
				&mock.ActionMock{
					NameFunc:      func() string { return "base" },
					ConfigureFunc: func(ctx context.Context) error { return nil },
					ToolsFunc: func() []*genai.FunctionDeclaration {
						return []*genai.FunctionDeclaration{{Name: "func"}}
					},
				},
			},
			new: []interfaces.Action{
				&mock.ActionMock{
					NameFunc:      func() string { return "new" },
					ConfigureFunc: func(ctx context.Context) error { return nil },
					ToolsFunc: func() []*genai.FunctionDeclaration {
						return []*genai.FunctionDeclaration{{Name: "func"}}
					},
				},
			},
			wantErr: true,
			errCheck: func(t *testing.T, err error) {
				gt.Error(t, err).Contains("function name is conflicted")
			},
		},
		{
			name: "error: empty action name in new actions",
			base: []interfaces.Action{
				&mock.ActionMock{
					NameFunc:      func() string { return "base" },
					ConfigureFunc: func(ctx context.Context) error { return nil },
					ToolsFunc: func() []*genai.FunctionDeclaration {
						return []*genai.FunctionDeclaration{{Name: "base.func"}}
					},
				},
			},
			new: []interfaces.Action{
				&mock.ActionMock{
					NameFunc:      func() string { return "" },
					ConfigureFunc: func(ctx context.Context) error { return nil },
					ToolsFunc:     func() []*genai.FunctionDeclaration { return nil },
				},
			},
			wantErr: true,
			errCheck: func(t *testing.T, err error) {
				gt.Error(t, err).Contains("action name is required")
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			baseSvc, err := action.New(t.Context(), tt.base)
			gt.NoError(t, err)

			newSvc, err := baseSvc.With(t.Context(), tt.new...)
			if tt.wantErr {
				gt.Error(t, err)
				if tt.errCheck != nil {
					tt.errCheck(t, err)
				}
				return
			}

			gt.NoError(t, err)
			gt.NotNil(t, newSvc)

			if tt.specCheck != nil {
				tt.specCheck(t, newSvc.Tools())
			}
		})
	}
}
