package usecase_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/opaq"
	"github.com/secmon-lab/warren/pkg/usecase"
)

func TestDiffPolicy(t *testing.T) {
	oldPolicy, err := opaq.New(opaq.Files("testdata/diff/old"), opaq.WithRelPath("testdata/diff/old"))
	gt.NoError(t, err)
	newPolicy, err := opaq.New(opaq.Files("testdata/diff/new"), opaq.WithRelPath("testdata/diff/new"))
	gt.NoError(t, err)

	diff := usecase.DiffPolicy(oldPolicy.Sources(), newPolicy.Sources())
	println(diff)
}
