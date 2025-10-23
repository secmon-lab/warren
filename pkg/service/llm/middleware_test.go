package llm_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/mock"
	"github.com/secmon-lab/warren/pkg/service/llm"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

func TestNewCompactionMiddleware(t *testing.T) {
	client := &mock.LLMClientMock{}
	logger := logging.Default()

	middleware := llm.NewCompactionMiddleware(client, logger)
	gt.V(t, middleware).NotNil()
}

func TestNewCompactionStreamMiddleware(t *testing.T) {
	client := &mock.LLMClientMock{}

	middleware := llm.NewCompactionStreamMiddleware(client)
	gt.V(t, middleware).NotNil()
}
