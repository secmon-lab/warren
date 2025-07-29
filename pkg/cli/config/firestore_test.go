package config_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/cli/config"
)

func TestFirestore_ProjectID(t *testing.T) {
	t.Run("returns empty string when not configured", func(t *testing.T) {
		cfg := &config.Firestore{}
		gt.Equal(t, "", cfg.ProjectID())
	})
}

func TestFirestore_IsConfigured(t *testing.T) {
	t.Run("returns false when project ID is empty", func(t *testing.T) {
		cfg := &config.Firestore{}
		gt.Equal(t, false, cfg.IsConfigured())
	})
}

func TestFirestore_Configure(t *testing.T) {
	t.Run("returns error when project ID is empty", func(t *testing.T) {
		cfg := &config.Firestore{}
		ctx := context.Background()
		_, err := cfg.Configure(ctx)
		gt.Error(t, err)
	})
}
