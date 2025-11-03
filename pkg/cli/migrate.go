package cli

import (
	"context"

	"github.com/m-mizutani/fireconf"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/cli/config"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/urfave/cli/v3"
)

func cmdMigrate() *cli.Command {
	var cfg config.Firestore
	var dryRun bool

	return &cli.Command{
		Name:    "migrate",
		Aliases: []string{"m"},
		Usage:   "Migrate Firestore indexes and configurations",
		Flags: append(cfg.Flags(),
			&cli.BoolFlag{
				Name:        "dry-run",
				Usage:       "Show what would be changed without applying",
				Destination: &dryRun,
			},
		),
		Action: func(ctx context.Context, c *cli.Command) error {
			return runMigrate(ctx, &cfg, dryRun)
		},
	}
}

func runMigrate(ctx context.Context, cfg *config.Firestore, dryRun bool) error {
	logger := logging.From(ctx)

	projectID := cfg.ProjectID()
	databaseID := cfg.DatabaseID()

	if projectID == "" {
		return goerr.New("firestore-project-id is required")
	}

	logger.Info("Starting Firestore migration",
		"project_id", projectID,
		"database_id", databaseID,
		"dry_run", dryRun,
	)

	// Get index configuration
	indexConfig := defineFirestoreIndexes()

	// Create fireconf client with options
	var opts []fireconf.Option

	opts = append(opts, fireconf.WithLogger(logger))
	if dryRun {
		logger.Info("Dry-run mode: showing planned changes without applying")
		opts = append(opts, fireconf.WithDryRun(true))
	}

	client, err := fireconf.NewClient(ctx, projectID, databaseID, opts...)
	if err != nil {
		return goerr.Wrap(err, "failed to create fireconf client",
			goerr.V("project_id", projectID),
			goerr.V("database_id", databaseID),
		)
	}

	// Apply migration
	if err := client.Migrate(ctx, indexConfig); err != nil {
		return goerr.Wrap(err, "failed to migrate indexes",
			goerr.V("project_id", projectID),
			goerr.V("database_id", databaseID),
			goerr.V("dry_run", dryRun),
		)
	}

	logger.Info("Migration completed successfully")
	return nil
}

func defineFirestoreIndexes() *fireconf.Config {
	collections := []string{"alerts", "tickets", "lists"}

	var firestoreCollections []fireconf.Collection

	// Indexes for alerts, tickets, lists (with Embedding field)
	for _, collectionName := range collections {
		var indexes []fireconf.Index

		// Single-field Embedding index
		indexes = append(indexes, fireconf.Index{
			Fields: []fireconf.IndexField{
				{
					Path: "Embedding",
					Vector: &fireconf.VectorConfig{
						Dimension: 256,
					},
				},
			},
		})

		// CreatedAt + Embedding composite index
		indexes = append(indexes, fireconf.Index{
			Fields: []fireconf.IndexField{
				{
					Path:  "CreatedAt",
					Order: fireconf.OrderDescending,
				},
				{
					Path: "Embedding",
					Vector: &fireconf.VectorConfig{
						Dimension: 256,
					},
				},
			},
		})

		// Status + CreatedAt index only for 'tickets'
		if collectionName == "tickets" {
			indexes = append(indexes, fireconf.Index{
				Fields: []fireconf.IndexField{
					{
						Path:  "Status",
						Order: fireconf.OrderAscending,
					},
					{
						Path:  "CreatedAt",
						Order: fireconf.OrderDescending,
					},
				},
			})
		}

		firestoreCollections = append(firestoreCollections, fireconf.Collection{
			Name:    collectionName,
			Indexes: indexes,
		})
	}

	// Index for memories subcollection (COLLECTION query scope)
	// This is used for agent-specific memory searches: agents/{agentID}/memories/*
	// Note: COLLECTION scope is required for queries on a specific subcollection path
	firestoreCollections = append(firestoreCollections, fireconf.Collection{
		Name: "memories",
		Indexes: []fireconf.Index{
			{
				QueryScope: fireconf.QueryScopeCollection,
				Fields: []fireconf.IndexField{
					{
						Path: "QueryEmbedding",
						Vector: &fireconf.VectorConfig{
							Dimension: 256,
						},
					},
				},
			},
		},
	})

	// Index for execution_memories/records subcollection (COLLECTION query scope)
	// This is used for schema-specific execution memory searches: execution_memories/{schemaID}/records/*
	// Note: COLLECTION scope is required for queries on a specific subcollection path
	firestoreCollections = append(firestoreCollections, fireconf.Collection{
		Name: "records",
		Indexes: []fireconf.Index{
			{
				QueryScope: fireconf.QueryScopeCollection,
				Fields: []fireconf.IndexField{
					{
						Path: "Embedding",
						Vector: &fireconf.VectorConfig{
							Dimension: 256,
						},
					},
				},
			},
		},
	})

	return &fireconf.Config{
		Collections: firestoreCollections,
	}
}
