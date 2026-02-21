package cli

import (
	"context"
	"time"

	firestoreadmin "cloud.google.com/go/firestore/apiv1/admin"
	adminpb "cloud.google.com/go/firestore/apiv1/admin/adminpb"
	"github.com/m-mizutani/fireconf"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/cli/config"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/safe"
	"github.com/urfave/cli/v3"
	"google.golang.org/api/iterator"
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

	if !dryRun {
		// Wait for all indexes to become READY.
		// fireconf's LRO wait may return before vector indexes are actually usable,
		// so we poll the Admin API directly until every index is in READY state.
		if err := waitForIndexesReady(ctx, projectID, databaseID, indexConfig, logger.With("phase", "wait_ready")); err != nil {
			return goerr.Wrap(err, "indexes did not become ready",
				goerr.V("project_id", projectID),
				goerr.V("database_id", databaseID),
			)
		}
	}

	logger.Info("Migration completed successfully")
	return nil
}

// waitForIndexesReady polls Firestore Admin API until all managed indexes are READY.
func waitForIndexesReady(ctx context.Context, projectID, databaseID string, cfg *fireconf.Config, logger interface{ Info(string, ...any) }) error {
	adminClient, err := firestoreadmin.NewFirestoreAdminClient(ctx)
	if err != nil {
		return goerr.Wrap(err, "failed to create firestore admin client")
	}
	defer safe.Close(ctx, adminClient)

	// Collect the collection names we care about.
	var collections []string
	for _, col := range cfg.Collections {
		collections = append(collections, col.Name)
	}

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		allReady := true

		for _, collectionName := range collections {
			parent := "projects/" + projectID + "/databases/" + databaseID + "/collectionGroups/" + collectionName

			it := adminClient.ListIndexes(ctx, &adminpb.ListIndexesRequest{Parent: parent})
			for {
				idx, err := it.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					return goerr.Wrap(err, "failed to list indexes",
						goerr.V("collection", collectionName))
				}

				state := idx.GetState()
				if state == adminpb.Index_CREATING || state == adminpb.Index_NEEDS_REPAIR {
					allReady = false
					logger.Info("Index not yet ready, waiting",
						"collection", collectionName,
						"index", idx.GetName(),
						"state", state.String(),
					)
				}
			}
		}

		if allReady {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func defineFirestoreIndexes() *fireconf.Config {
	collections := []string{"alerts", "tickets", "lists"}

	var firestoreCollections []fireconf.Collection

	// Indexes for alerts, tickets, lists (with Embedding field)
	for _, collectionName := range collections {
		var indexes []fireconf.Index

		// Single-field Embedding index
		indexes = append(indexes, fireconf.Index{
			QueryScope: fireconf.QueryScopeCollection,
			Fields: []fireconf.IndexField{
				{
					Path: "Embedding",
					Vector: &fireconf.VectorConfig{
						Dimension: 256,
					},
				},
			},
		})

		// __name__ + Embedding composite index (required for DistanceResultField queries)
		// Note: vector field must be last in composite index
		indexes = append(indexes, fireconf.Index{
			QueryScope: fireconf.QueryScopeCollection,
			Fields: []fireconf.IndexField{
				{
					Path:  "__name__",
					Order: fireconf.OrderAscending,
				},
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
			QueryScope: fireconf.QueryScopeCollection,
			Fields: []fireconf.IndexField{
				{
					Path:  "CreatedAt",
					Order: fireconf.OrderDescending,
				},
				{
					Path:  "__name__",
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

		// Status + CreatedAt + __name__ index only for 'tickets'
		if collectionName == "tickets" {
			indexes = append(indexes, fireconf.Index{
				QueryScope: fireconf.QueryScopeCollection,
				Fields: []fireconf.IndexField{
					{
						Path:  "Status",
						Order: fireconf.OrderAscending,
					},
					{
						Path:  "CreatedAt",
						Order: fireconf.OrderDescending,
					},
					{
						Path:  "__name__",
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
			// __name__ + QueryEmbedding composite index (required for DistanceResultField queries)
			// Note: vector field must be last in composite index
			{
				QueryScope: fireconf.QueryScopeCollection,
				Fields: []fireconf.IndexField{
					{
						Path:  "__name__",
						Order: fireconf.OrderAscending,
					},
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
