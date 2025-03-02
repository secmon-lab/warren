package usecase_test

import (
	"io"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/service"
	"github.com/secmon-lab/warren/pkg/usecase"
)

func TestRunCommand(t *testing.T) {
	uc := usecase.New()
	th := service.NewConsole(io.Discard).NewThread(model.SlackThread{
		ChannelID: "C07000000000000000",
		ThreadID:  "T07000000000000000",
	})
	gt.NoError(t, uc.RunCommand(t.Context(), []string{"warren", "help"}, nil, th)).Must()
}
