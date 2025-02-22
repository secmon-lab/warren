package usecase

import (
	"context"

	"github.com/secmon-lab/warren/pkg/model"
)

func (uc *UseCases) FindSimilarAlert(ctx context.Context, alert model.Alert) (*model.Alert, error) {
	return uc.findSimilarAlert(ctx, alert)
}

func (uc *UseCases) GenerateAlertMetadata(ctx context.Context, alert model.Alert) (*model.Alert, error) {
	return uc.generateAlertMetadata(ctx, alert)
}

var (
	PlanAction = planAction
	DiffPolicy = diffPolicy
)
