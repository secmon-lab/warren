package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	firestoreadmin "cloud.google.com/go/firestore/apiv1/admin"
	adminpb "cloud.google.com/go/firestore/apiv1/admin/adminpb"
	"github.com/m-mizutani/fireconf"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/cli/config"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/safe"
	"github.com/urfave/cli/v3"
	"google.golang.org/api/iterator"
)

type migrationJob struct {
	Name        string
	Description string
	Run         func(ctx context.Context, projectID, databaseID string, dryRun bool) error
}

const defaultMigrationJob = "index"

var migrationJobs = []migrationJob{
	{
		Name:        "index",
		Description: "Sync Firestore composite indexes to match the application schema (requires Firestore Admin API permission)",
		Run:         migrateIndexes,
	},
	{
		Name:        "backfill-alert-status",
		Description: "Backfill Status field on pre-v0.10.0 alerts that lack the field or have the old 'unbound' value, setting them to 'active' so they appear in Firestore queries",
		Run:         backfillAlertStatus,
	},
}

func findMigrationJob(name string) (*migrationJob, bool) {
	for i := range migrationJobs {
		if migrationJobs[i].Name == name {
			return &migrationJobs[i], true
		}
	}
	return nil, false
}

func cmdMigrate() *cli.Command {
	var cfg config.Firestore
	var dryRun bool
	var listJobs bool
	var jobNames []string

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
			&cli.BoolFlag{
				Name:        "list",
				Aliases:     []string{"l"},
				Usage:       "List available migration jobs",
				Destination: &listJobs,
			},
			&cli.StringSliceFlag{
				Name:        "job",
				Aliases:     []string{"j"},
				Usage:       "Additional migration jobs to run (use --list to see available jobs)",
				Destination: &jobNames,
			},
		),
		Action: func(ctx context.Context, c *cli.Command) error {
			if listJobs {
				printMigrationJobs()
				return nil
			}
			return runMigrate(ctx, &cfg, dryRun, jobNames)
		},
	}
}

func printMigrationJobs() {
	fmt.Println("Available migration jobs:")
	fmt.Println()
	for _, job := range migrationJobs {
		fmt.Printf("  %s\n", job.Name)
		// Wrap description to keep it readable
		for _, line := range wrapText(job.Description, 60) {
			fmt.Printf("      %s\n", line)
		}
		fmt.Println()
	}
}

func wrapText(text string, width int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}

	var lines []string
	current := words[0]
	for _, word := range words[1:] {
		if len(current)+1+len(word) > width {
			lines = append(lines, current)
			current = word
		} else {
			current += " " + word
		}
	}
	lines = append(lines, current)
	return lines
}

func runMigrate(ctx context.Context, cfg *config.Firestore, dryRun bool, jobNames []string) error {
	logger := logging.From(ctx)

	// Default to index migration when no jobs specified
	if len(jobNames) == 0 {
		jobNames = []string{defaultMigrationJob}
	}

	// Validate all job names before starting any work
	for _, name := range jobNames {
		if _, ok := findMigrationJob(name); !ok {
			var available []string
			for _, j := range migrationJobs {
				available = append(available, j.Name)
			}
			return goerr.New("unknown migration job (use --list to see available jobs)",
				goerr.V("job", name),
				goerr.V("available", available),
			)
		}
	}

	projectID := cfg.ProjectID()
	databaseID := cfg.DatabaseID()

	if projectID == "" {
		return goerr.New("firestore-project-id is required")
	}

	logger.Info("Starting Firestore migration",
		"project_id", projectID,
		"database_id", databaseID,
		"dry_run", dryRun,
		"jobs", jobNames,
	)

	// Run migration jobs
	for _, name := range jobNames {
		job, _ := findMigrationJob(name) // already validated above
		logger.Info("Running migration job", "job", job.Name)
		if err := job.Run(ctx, projectID, databaseID, dryRun); err != nil {
			return goerr.Wrap(err, "migration job failed",
				goerr.V("job", job.Name))
		}
	}

	logger.Info("Migration completed successfully")
	return nil
}

func migrateIndexes(ctx context.Context, projectID, databaseID string, dryRun bool) error {
	logger := logging.From(ctx)

	indexConfig := defineFirestoreIndexes()

	var opts []fireconf.Option
	opts = append(opts, fireconf.WithLogger(logger))
	if dryRun {
		opts = append(opts, fireconf.WithDryRun(true))
	}

	client, err := fireconf.NewClient(ctx, projectID, databaseID, opts...)
	if err != nil {
		return goerr.Wrap(err, "failed to create fireconf client")
	}

	if err := client.Migrate(ctx, indexConfig); err != nil {
		return goerr.Wrap(err, "failed to migrate indexes")
	}

	if !dryRun {
		// Wait for all indexes to become READY.
		// fireconf's LRO wait may return before vector indexes are actually usable,
		// so we poll the Admin API directly until every index is in READY state.
		if err := waitForIndexesReady(ctx, projectID, databaseID, indexConfig, logger.With("phase", "wait_ready")); err != nil {
			return goerr.Wrap(err, "indexes did not become ready")
		}
	}

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

func backfillAlertStatus(ctx context.Context, projectID, databaseID string, dryRun bool) error {
	logger := logging.From(ctx)
	logger.Info("Starting alert status backfill")

	db, err := firestore.NewClientWithDatabase(ctx, projectID, databaseID)
	if err != nil {
		return goerr.Wrap(err, "failed to create firestore data client")
	}
	defer safe.Close(ctx, db)

	iter := db.Collection("alerts").Documents(ctx)

	var targets []*firestore.DocumentRef
	var totalCount int
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return goerr.Wrap(err, "failed to iterate alerts")
		}
		totalCount++

		var a alert.Alert
		if err := doc.DataTo(&a); err != nil {
			return goerr.Wrap(err, "failed to deserialize alert",
				goerr.V("doc_id", doc.Ref.ID))
		}

		if a.Status == "" || a.Status == "unbound" {
			targets = append(targets, doc.Ref)
		}
	}

	logger.Info("Alert status backfill scan completed",
		"total_alerts", totalCount,
		"needs_backfill", len(targets),
	)

	if len(targets) == 0 {
		logger.Info("No alerts need status backfill")
		return nil
	}

	if dryRun {
		logger.Info("Dry-run mode: skipping backfill write",
			"would_update", len(targets),
		)
		return nil
	}

	bw := db.BulkWriter(ctx)
	var jobs []*firestore.BulkWriterJob
	for _, ref := range targets {
		job, err := bw.Update(ref, []firestore.Update{
			{
				Path:  "Status",
				Value: string(alert.AlertStatusActive),
			},
		})
		if err != nil {
			return goerr.Wrap(err, "failed to enqueue status update",
				goerr.V("doc_id", ref.ID))
		}
		jobs = append(jobs, job)
	}
	bw.End()

	for _, job := range jobs {
		if _, err := job.Results(); err != nil {
			return goerr.Wrap(err, "failed to commit status update")
		}
	}

	logger.Info("Alert status backfill completed",
		"updated", len(targets),
	)
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
