package action_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/action"
)

func TestActionSpec_Validate(t *testing.T) {
	testCases := []struct {
		name    string
		spec    action.ActionSpec
		args    action.Arguments
		wantErr bool
	}{
		{
			name: "valid string argument",
			spec: action.ActionSpec{
				Name: "test",
				Args: []action.ArgumentSpec{
					{Name: "test", Type: action.ArgumentTypeString, Required: true},
				},
			},
			args:    action.Arguments{"test": "test"},
			wantErr: false,
		},
		{
			name: "missing required argument",
			spec: action.ActionSpec{
				Name: "test",
				Args: []action.ArgumentSpec{
					{Name: "test", Type: action.ArgumentTypeString, Required: true},
				},
			},
			args:    action.Arguments{},
			wantErr: true,
		},
		{
			name: "invalid string argument",
			spec: action.ActionSpec{
				Name: "test",
				Args: []action.ArgumentSpec{
					{Name: "test", Type: action.ArgumentTypeString, Required: true},
				},
			},
			args:    action.Arguments{"test": ""},
			wantErr: true,
		},
		{
			name: "valid number argument",
			spec: action.ActionSpec{
				Name: "test",
				Args: []action.ArgumentSpec{
					{Name: "test", Type: action.ArgumentTypeNumber, Required: true},
				},
			},
			args:    action.Arguments{"test": float64(123)},
			wantErr: false,
		},
		{
			name: "invalid number argument",
			spec: action.ActionSpec{
				Name: "test",
				Args: []action.ArgumentSpec{
					{Name: "test", Type: action.ArgumentTypeNumber, Required: true},
				},
			},
			args:    action.Arguments{"test": "not a number"},
			wantErr: true,
		},
		{
			name: "valid boolean argument",
			spec: action.ActionSpec{
				Name: "test",
				Args: []action.ArgumentSpec{
					{Name: "test", Type: action.ArgumentTypeBoolean, Required: true},
				},
			},
			args:    action.Arguments{"test": true},
			wantErr: false,
		},
		{
			name: "invalid boolean argument",
			spec: action.ActionSpec{
				Name: "test",
				Args: []action.ArgumentSpec{
					{Name: "test", Type: action.ArgumentTypeBoolean, Required: true},
				},
			},
			args:    action.Arguments{"test": "not a boolean"},
			wantErr: true,
		},
		{
			name: "valid choice argument",
			spec: action.ActionSpec{
				Name: "test",
				Args: []action.ArgumentSpec{
					{
						Name:     "test",
						Type:     action.ArgumentTypeString,
						Required: true,
						Choices: action.ChoiceSpecs{
							{Value: "choice1", Description: "test"},
							{Value: "choice2", Description: "test"},
						},
					},
				},
			},
			args:    action.Arguments{"test": "choice1"},
			wantErr: false,
		},
		{
			name: "invalid choice argument",
			spec: action.ActionSpec{
				Name: "test",
				Args: []action.ArgumentSpec{
					{
						Name:     "test",
						Type:     action.ArgumentTypeString,
						Required: true,
						Choices: action.ChoiceSpecs{
							{Value: "choice1", Description: "test"},
							{Value: "choice2", Description: "test"},
						},
					},
				},
			},
			args:    action.Arguments{"test": "invalid"},
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
