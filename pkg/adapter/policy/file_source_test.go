package policy_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/adapter/policy"
)

func TestFileSource_Snapshot(t *testing.T) {
	t.Run("loads files from a directory", func(t *testing.T) {
		src, err := policy.NewFileSource([]string{"testdata"})
		gt.NoError(t, err)

		snap, err := src.Snapshot(context.Background())
		gt.NoError(t, err)
		gt.NotNil(t, snap)
		gt.M(t, snap.Files).Length(1)
		gt.True(t, snap.Version != "")
	})

	t.Run("returns identical snapshot on repeated calls", func(t *testing.T) {
		src, err := policy.NewFileSource([]string{"testdata"})
		gt.NoError(t, err)

		first, err := src.Snapshot(context.Background())
		gt.NoError(t, err)
		second, err := src.Snapshot(context.Background())
		gt.NoError(t, err)

		gt.Equal(t, first.Version, second.Version)
	})

	t.Run("fails on non-existent path", func(t *testing.T) {
		_, err := policy.NewFileSource([]string{"testdata/does-not-exist"})
		gt.Error(t, err)
	})
}
