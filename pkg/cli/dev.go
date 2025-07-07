package cli

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/secmon-lab/warren/pkg/adapter/storage"
	"github.com/secmon-lab/warren/pkg/cli/config"
	server "github.com/secmon-lab/warren/pkg/controller/http"
	"github.com/secmon-lab/warren/pkg/domain/mock"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	slack_svc "github.com/secmon-lab/warren/pkg/service/slack"
	"github.com/secmon-lab/warren/pkg/usecase"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/urfave/cli/v3"

	slack_sdk "github.com/slack-go/slack"
)

func cmdDev() *cli.Command {
	var (
		addr      string
		webUICfg  config.WebUI
		geminiCfg config.GeminiCfg
	)

	flags := joinFlags(
		[]cli.Flag{
			&cli.StringFlag{
				Name:        "addr",
				Aliases:     []string{"a"},
				Sources:     cli.EnvVars("WARREN_ADDR"),
				Usage:       "Listen address (default: 127.0.0.1:8080)",
				Value:       "127.0.0.1:8080",
				Destination: &addr,
			},
		},
		webUICfg.Flags(),
		geminiCfg.Flags(),
	)

	return &cli.Command{
		Name:    "dev",
		Aliases: []string{"d"},
		Usage:   "Run server in development mode with dummy data",
		Flags:   flags,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			logging.Default().Info("starting development server",
				"addr", addr,
				"web-ui", webUICfg,
				"gemini", geminiCfg,
			)

			geminiModel, err := geminiCfg.Configure(ctx)
			if err != nil {
				return err
			}

			// Create mock policy
			policyClient := &mock.PolicyClientMock{}

			// Create mock Slack service
			var slackOpts []slack_svc.ServiceOption
			if webUICfg.GetFrontendURL() != "" {
				slackOpts = append(slackOpts, slack_svc.WithFrontendURL(webUICfg.GetFrontendURL()))
			}

			slackSvc, err := slack_svc.New(&mock.SlackClientMock{
				AuthTestFunc: func() (*slack_sdk.AuthTestResponse, error) {
					return &slack_sdk.AuthTestResponse{
						User:   "U0000000000",
						Team:   "T0000000000",
						URL:    "https://slack.com",
						TeamID: "T0000000000",
						UserID: "U0000000000",
						BotID:  "B0000000000",
					}, nil
				},
			}, "C_DEV_CHANNEL", slackOpts...)
			if err != nil {
				return err
			}

			// Create mock storage client
			storageClient := storage.NewMock()

			// Create repository with memory implementation and mock services
			repo := repository.NewMemory()

			// Generate dummy data if in development mode
			if err := populateDummyData(ctx, repo); err != nil {
				return fmt.Errorf("failed to generate dummy data: %w", err)
			}

			ucOptions := []usecase.Option{
				usecase.WithLLMClient(geminiModel),
				usecase.WithPolicyClient(policyClient),
				usecase.WithRepository(repo),
				usecase.WithSlackService(slackSvc),
				usecase.WithStorageClient(storageClient),
			}

			uc := usecase.New(ucOptions...)

			// Build HTTP server options
			serverOptions := []server.Options{
				server.WithSlackVerifier(nil), // No verifier needed for dev mode
				server.WithGraphQLRepo(repo),
				server.WithGraphiQL(true),
				server.WithSlackService(slackSvc),
			}

			// Add AuthUseCase if authentication options are provided
			authUC, err := webUICfg.Configure(repo, slackSvc)
			if err != nil {
				return err
			}
			if authUC != nil {
				serverOptions = append(serverOptions, server.WithAuthUseCase(authUC))
			}

			httpServer := http.Server{
				Addr:              addr,
				Handler:           server.New(uc, serverOptions...),
				ReadTimeout:       30 * time.Second,
				ReadHeaderTimeout: 10 * time.Second,
				BaseContext: func(l net.Listener) context.Context {
					return ctx
				},
			}

			errCh := make(chan error, 1)
			go func() {
				defer close(errCh)
				if err := httpServer.ListenAndServe(); err != nil {
					errCh <- err
				}
			}()

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

			select {
			case err := <-errCh:
				return err
			case <-sigCh:
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				return httpServer.Shutdown(ctx)
			}
		},
	}
}

// populateDummyData creates dummy tickets and alerts for development
func populateDummyData(ctx context.Context, repo *repository.Memory) error {
	logging.Default().Info("populating dummy data...")

	// Define dummy data sets
	alertTitles := []string{
		"Suspicious Network Activity Detected",
		"Malware Detection Alert",
		"Failed Login Attempts",
		"Unauthorized File Access",
		"Anomalous Data Transfer",
		"Privilege Escalation Attempt",
		"SQL Injection Detected",
		"Phishing Email Detected",
		"Brute Force Attack",
		"Data Exfiltration Alert",
		"Suspicious Process Execution",
		"Network Intrusion Detected",
		"Ransomware Activity",
		"Insider Threat Alert",
		"Command Injection Attempt",
	}

	ticketTitles := []string{
		"Investigation: Network Security Incident",
		"Malware Outbreak Response",
		"Security Policy Violation",
		"Data Breach Investigation",
		"Suspicious User Activity",
		"External Attack Investigation",
		"Compliance Violation Alert",
		"System Compromise Analysis",
		"Threat Intelligence Investigation",
		"Incident Response Activity",
	}

	assignees := []string{
		"alice.security",
		"bob.analyst",
		"carol.admin",
		"david.ops",
		"eve.investigator",
	}

	commentPatterns := []string{
		"Initial investigation started. Gathering logs and evidence.",
		"Escalating to security team for further analysis.",
		"False positive confirmed. Closing ticket.",
		"Mitigation actions applied. Monitoring for additional activity.",
		"Root cause identified. Implementing preventive measures.",
		"Coordinating with network team for additional controls.",
		"User contacted for verification. Awaiting response.",
		"Threat neutralized. System restored to normal operation.",
	}

	statuses := []types.TicketStatus{
		types.TicketStatusOpen,
		types.TicketStatusPending,
		types.TicketStatusResolved,
		types.TicketStatusArchived,
	}

	// Time range for dummy data
	startTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)
	timeRange := endTime.Sub(startTime)

	// Create alerts first
	var alerts []*alert.Alert
	for i := 0; i < 50; i++ {
		// Random time within 2025
		randomDuration := time.Duration(rand.Float64() * float64(timeRange))
		alertTime := startTime.Add(randomDuration)

		ctx = clock.With(ctx, func() time.Time { return alertTime })

		alertTitle := alertTitles[rand.IntN(len(alertTitles))]

		newAlert := alert.New(ctx, types.AlertSchema("dev.alert.v1"), map[string]any{
			"severity":       []string{"low", "medium", "high", "critical"}[rand.IntN(4)],
			"source_ip":      generateRandomIP(),
			"destination_ip": generateRandomIP(),
			"event_type":     alertTitle,
			"timestamp":      alertTime.Format(time.RFC3339),
		}, alert.Metadata{
			Title:       alertTitle,
			Description: "Development environment dummy alert for testing purposes",
			Attributes: []alert.Attribute{
				{Key: "environment", Value: "development"},
				{Key: "source", Value: "dummy_generator"},
			},
		})

		alerts = append(alerts, &newAlert)
		if err := repo.PutAlert(ctx, newAlert); err != nil {
			return err
		}
	}

	// Create tickets and bind some alerts
	for i := 0; i < 15; i++ {
		// Random time within 2025
		randomDuration := time.Duration(rand.Float64() * float64(timeRange))
		ticketTime := startTime.Add(randomDuration)

		ctx = clock.With(ctx, func() time.Time { return ticketTime })

		// Select random alerts to bind (1-10 alerts per ticket)
		numAlerts := rand.IntN(10) + 1
		var selectedAlertIDs []types.AlertID
		for j := 0; j < numAlerts && j < len(alerts); j++ {
			alertIndex := rand.IntN(len(alerts))
			if alerts[alertIndex].TicketID == types.EmptyTicketID {
				selectedAlertIDs = append(selectedAlertIDs, alerts[alertIndex].ID)
				alerts[alertIndex].TicketID = types.NewTicketID() // Temporary assignment
			}
		}

		if len(selectedAlertIDs) == 0 {
			continue // Skip if no available alerts
		}

		// Create thread
		thread := &slack.Thread{
			ChannelID: "C0DEV123456",
			ThreadID:  generateRandomThreadID(),
		}

		// Create ticket
		newTicket := ticket.New(ctx, selectedAlertIDs, thread)
		newTicket.Metadata.Title = ticketTitles[rand.IntN(len(ticketTitles))]
		newTicket.Metadata.Description = "Development environment dummy ticket for testing purposes"
		newTicket.Metadata.Summary = "This is a summary of the security incident being investigated"
		newTicket.Status = statuses[rand.IntN(len(statuses))]
		assigneeName := assignees[rand.IntN(len(assignees))]
		newTicket.Assignee = &slack.User{
			ID:   generateRandomUserID(),
			Name: assigneeName,
		}

		// Update alert ticket IDs
		for _, alertID := range selectedAlertIDs {
			for _, a := range alerts {
				if a.ID == alertID {
					a.TicketID = newTicket.ID
					if err := repo.PutAlert(ctx, *a); err != nil {
						return err
					}
					break
				}
			}
		}

		if err := repo.PutTicket(ctx, newTicket); err != nil {
			return err
		}

		// Add some comments to the ticket
		numComments := rand.IntN(3) + 1
		for j := 0; j < numComments; j++ {
			assigneeName := assignees[rand.IntN(len(assignees))]
			comment := ticket.Comment{
				ID:             types.NewCommentID(),
				TicketID:       newTicket.ID,
				Comment:        commentPatterns[rand.IntN(len(commentPatterns))],
				SlackMessageID: generateRandomMessageID(),
				User: &slack.User{
					ID:   generateRandomUserID(),
					Name: assigneeName,
				},
				CreatedAt: ticketTime.Add(time.Duration(j) * time.Hour),
				Prompted:  rand.IntN(2) == 0, // 50% chance of being prompted
			}
			if err := repo.PutTicketComment(ctx, comment); err != nil {
				return err
			}
		}
	}

	logging.Default().Info("dummy data population completed",
		"alerts", len(alerts),
		"tickets", 15,
	)

	return nil
}

func generateRandomIP() string {
	return fmt.Sprintf("%d.%d.%d.%d",
		rand.IntN(256),
		rand.IntN(256),
		rand.IntN(256),
		rand.IntN(256),
	)
}

func generateRandomThreadID() string {
	return fmt.Sprintf("%d.%06d", time.Now().Unix(), rand.IntN(1000000))
}

func generateRandomUserID() string {
	return fmt.Sprintf("U%09d", rand.IntN(1000000000))
}

func generateRandomMessageID() string {
	return fmt.Sprintf("%d.%06d", time.Now().Unix(), rand.IntN(1000000))
}
