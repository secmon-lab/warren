package service

import (
	"context"
	"errors"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/opaq"
	"github.com/secmon-lab/warren/pkg/interfaces"
	"github.com/secmon-lab/warren/pkg/model"
)

func DoTestPolicy(ctx context.Context, policy interfaces.PolicyClient, testDataSet *model.TestDataSet) []error {
	var errs []error
	for schema, dataSets := range testDataSet.Detect {
		for filename, data := range dataSets {
			var resp model.PolicyResult
			if err := policy.Query(ctx, "data.alert."+schema, data, &resp); err != nil {
				if errors.Is(err, opaq.ErrNoEvalResult) {
					errs = append(errs, goerr.Wrap(err, "should be detected, but not detected", goerr.V("schema", schema), goerr.V("data", data), goerr.V("filename", filename), goerr.T(model.ErrTagTestFailed)))
				} else {
					errs = append(errs, goerr.Wrap(err, "failed to query policy", goerr.V("schema", schema), goerr.V("data", data), goerr.V("filename", filename)))
				}
			}
		}
	}

	for schema, dataSets := range testDataSet.Ignore {
		for filename, data := range dataSets {
			var resp model.PolicyResult
			if err := policy.Query(ctx, "data.alert."+schema, data, &resp); err != nil {
				if errors.Is(err, opaq.ErrNoEvalResult) {
					continue
				}

				errs = append(errs, goerr.Wrap(err, "failed to query policy", goerr.V("schema", schema), goerr.V("data", data), goerr.V("filename", filename)))
			}

			if len(resp.Alert) > 0 {
				errs = append(errs, goerr.New("should be ignored, but detected", goerr.V("schema", schema), goerr.V("data", data), goerr.V("filename", filename), goerr.T(model.ErrTagTestFailed)))
			}
		}
	}

	return errs
}
