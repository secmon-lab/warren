package model_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model"
)

func TestActionSpec_Validate(t *testing.T) {
	testCases := []struct {
		name    string
		spec    model.ActionSpec
		args    model.Arguments
		wantErr bool
	}{
		{
			name: "valid string argument",
			spec: model.ActionSpec{
				Name: "test",
				Args: []model.ArgumentSpec{
					{Name: "test", Type: model.ArgumentTypeString, Required: true},
				},
			},
			args:    model.Arguments{"test": "test"},
			wantErr: false,
		},
		{
			name: "missing required argument",
			spec: model.ActionSpec{
				Name: "test",
				Args: []model.ArgumentSpec{
					{Name: "test", Type: model.ArgumentTypeString, Required: true},
				},
			},
			args:    model.Arguments{},
			wantErr: true,
		},
		{
			name: "invalid string argument",
			spec: model.ActionSpec{
				Name: "test",
				Args: []model.ArgumentSpec{
					{Name: "test", Type: model.ArgumentTypeString, Required: true},
				},
			},
			args:    model.Arguments{"test": ""},
			wantErr: true,
		},
		{
			name: "valid number argument",
			spec: model.ActionSpec{
				Name: "test",
				Args: []model.ArgumentSpec{
					{Name: "test", Type: model.ArgumentTypeNumber, Required: true},
				},
			},
			args:    model.Arguments{"test": float64(123)},
			wantErr: false,
		},
		{
			name: "invalid number argument",
			spec: model.ActionSpec{
				Name: "test",
				Args: []model.ArgumentSpec{
					{Name: "test", Type: model.ArgumentTypeNumber, Required: true},
				},
			},
			args:    model.Arguments{"test": "not a number"},
			wantErr: true,
		},
		{
			name: "valid boolean argument",
			spec: model.ActionSpec{
				Name: "test",
				Args: []model.ArgumentSpec{
					{Name: "test", Type: model.ArgumentTypeBoolean, Required: true},
				},
			},
			args:    model.Arguments{"test": true},
			wantErr: false,
		},
		{
			name: "invalid boolean argument",
			spec: model.ActionSpec{
				Name: "test",
				Args: []model.ArgumentSpec{
					{Name: "test", Type: model.ArgumentTypeBoolean, Required: true},
				},
			},
			args:    model.Arguments{"test": "not a boolean"},
			wantErr: true,
		},
		{
			name: "valid choice argument",
			spec: model.ActionSpec{
				Name: "test",
				Args: []model.ArgumentSpec{
					{
						Name:     "test",
						Type:     model.ArgumentTypeString,
						Required: true,
						Choices: model.ChoiceSpecs{
							{Value: "choice1", Description: "test"},
							{Value: "choice2", Description: "test"},
						},
					},
				},
			},
			args:    model.Arguments{"test": "choice1"},
			wantErr: false,
		},
		{
			name: "invalid choice argument",
			spec: model.ActionSpec{
				Name: "test",
				Args: []model.ArgumentSpec{
					{
						Name:     "test",
						Type:     model.ArgumentTypeString,
						Required: true,
						Choices: model.ChoiceSpecs{
							{Value: "choice1", Description: "test"},
							{Value: "choice2", Description: "test"},
						},
					},
				},
			},
			args:    model.Arguments{"test": "invalid"},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.spec.Validate(tc.args)
			if tc.wantErr {
				gt.Error(t, err)
			} else {
				gt.NoError(t, err)
			}
		})
	}
}
