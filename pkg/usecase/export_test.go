package usecase

import (
	"context"

	"github.com/secmon-lab/warren/pkg/domain/model/alert"
)

func (uc *UseCases) FindSimilarAlert(ctx context.Context, alert alert.Alert) (*alert.Alert, error) {
	return uc.findSimilarAlert(ctx, alert)
}

func (uc *UseCases) GenerateAlertMetadata(ctx context.Context, alert alert.Alert) (*alert.Alert, error) {
	return uc.generateAlertMetadata(ctx, alert)
}

var (
	PlanAction       = planAction
	FormatRegoPolicy = formatRegoPolicy
)
