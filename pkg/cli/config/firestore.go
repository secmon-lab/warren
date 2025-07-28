package config

import (
	"context"
	"log/slog"

	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/urfave/cli/v3"
)

type Firestore struct {
	projectID  string
	databaseID string
}

func (c *Firestore) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "firestore-project-id",
			Usage:       "Firestore project ID",
			Required:    false,
			Destination: &c.projectID,
			Category:    "Firestore",
			Sources:     cli.EnvVars("WARREN_FIRESTORE_PROJECT_ID"),
		},
		&cli.StringFlag{
			Name:        "firestore-database-id",
			Usage:       "Firestore database ID",
			Destination: &c.databaseID,
			Category:    "Firestore",
			Sources:     cli.EnvVars("WARREN_FIRESTORE_DATABASE_ID"),
			Value:       "(default)",
		},
	}
}

func (c Firestore) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("project_id", c.projectID),
		slog.String("database_id", c.databaseID),
	)
}

func (c *Firestore) Configure(ctx context.Context) (*repository.Firestore, error) {
	return repository.NewFirestore(ctx, c.projectID, c.databaseID)
}

// ProjectID returns the project ID (exported for serve command)
func (c *Firestore) ProjectID() string {
	return c.projectID
}

// IsConfigured returns true if Firestore is configured
func (c *Firestore) IsConfigured() bool {
	return c.projectID != ""
}
