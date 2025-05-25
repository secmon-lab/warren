package http_test

import "github.com/secmon-lab/warren/pkg/domain/interfaces"

type useCaseInterface struct {
	interfaces.AlertUsecases
	interfaces.SlackEventUsecases
	interfaces.SlackInteractionUsecases
}
