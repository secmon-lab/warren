package usecase

import (
	"context"

	"github.com/secmon-lab/warren/pkg/model"
)

func (uc *UseCases) FindSimilarAlert(ctx context.Context, alert model.Alert) (*model.Alert, error) {
	return uc.findSimilarAlert(ctx, alert)
}

var PlanAction = planAction
