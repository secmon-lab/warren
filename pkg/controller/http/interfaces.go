package http

import "github.com/secmon-lab/warren/pkg/usecase"

type useCase interface {
	usecase.Alert
	usecase.SlackEvent
	usecase.SlackInteraction
}
