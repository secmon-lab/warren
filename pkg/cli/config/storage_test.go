package config_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/cli/config"
)

func TestStorage_Bucket(t *testing.T) {
	t.Run("returns empty string when not configured", func(t *testing.T) {
		cfg := &config.Storage{}
		gt.Equal(t, "", cfg.Bucket())
	})
}

func TestStorage_IsConfigured(t *testing.T) {
	t.Run("returns false when bucket is empty", func(t *testing.T) {
		cfg := &config.Storage{}
		gt.Equal(t, false, cfg.IsConfigured())
	})
}

func TestStorage_Configure(t *testing.T) {
	t.Run("returns error when bucket is empty", func(t *testing.T) {
		cfg := &config.Storage{}
		ctx := context.Background()
		_, err := cfg.Configure(ctx)
		gt.Error(t, err)
	})
}
