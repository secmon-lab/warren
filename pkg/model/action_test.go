package model_test

import (
	"testing"

	"github.com/secmon-lab/warren/pkg/model"
)

func TestArgumentSpec_Validate(t *testing.T) {
	testCases := []struct {
		name    string
		spec    []model.ArgumentSpec
		args    model.Arguments
		wantErr bool
	}{
		{
			name: "valid string argument",
			spec: []model.ArgumentSpec{
				{Name: "arg1", Type: model.ArgumentTypeString, Required: true},
			},
			args: model.Arguments{
				"arg1": "value1",
			},
			wantErr: false,
		},
		{
			name: "missing required argument",
			spec: []model.ArgumentSpec{
				{Name: "arg1", Type: model.ArgumentTypeString, Required: true},
			},
			args:    model.Arguments{},
			wantErr: true,
		},
		{
			name: "invalid boolean argument",
			spec: []model.ArgumentSpec{
				{Name: "arg1", Type: model.ArgumentTypeBoolean, Required: true},
			},
			args: model.Arguments{
				"arg1": "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid number argument",
			spec: []model.ArgumentSpec{
				{Name: "arg1", Type: model.ArgumentTypeNumber, Required: true},
			},
			args: model.Arguments{
				"arg1": "not-a-number",
			},
			wantErr: true,
		},
		{
			name: "allow optional argument",
			spec: []model.ArgumentSpec{
				{Name: "arg1", Type: model.ArgumentTypeString, Required: false},
			},
			args:    model.Arguments{},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.args.Validate(tc.spec)
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}
