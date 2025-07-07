package http

import "github.com/secmon-lab/warren/pkg/domain/interfaces"

type useCase interface {
	interfaces.AlertUsecases
	interfaces.SlackEventUsecases
	interfaces.SlackInteractionUsecases
}
