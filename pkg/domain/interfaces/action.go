package interfaces

import (
	"context"
	"log/slog"

	"cloud.google.com/go/vertexai/genai"
	"github.com/secmon-lab/warren/pkg/domain/model/action"
	"github.com/urfave/cli/v3"
)

type Action interface {
	Specs() []*genai.FunctionDeclaration
	Execute(ctx context.Context, name string, args map[string]any) (*action.Result, error)
	Flags() []cli.Flag
	Configure(ctx context.Context) error
	LogValue() slog.Value
}
