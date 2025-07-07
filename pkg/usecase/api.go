package usecase

import (
	"context"
	"encoding/json"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

// GetUserIcon returns the user's icon image data and content type
func (u *UseCases) GetUserIcon(ctx context.Context, userID string) ([]byte, string, error) {
	if u.slackService == nil {
		return nil, "", ErrSlackServiceNotConfigured
	}

	return u.slackService.GetUserIcon(ctx, userID)
}

// GetUserProfile returns the user's profile name via Slack service
func (u *UseCases) GetUserProfile(ctx context.Context, userID string) (string, error) {
	if u.slackService == nil {
		return "", ErrSlackServiceNotConfigured
	}

	return u.slackService.GetUserProfile(ctx, userID)
}

// GenerateTicketAlertsJSONL generates JSONL data for alerts associated with a ticket
func (u *UseCases) GenerateTicketAlertsJSONL(ctx context.Context, ticketID types.TicketID) ([]byte, error) {
	// Get ticket
	ticket, err := u.repository.GetTicket(ctx, ticketID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get ticket", goerr.V("ticket_id", ticketID))
	}
	if ticket == nil {
		return nil, goerr.New("ticket not found", goerr.V("ticket_id", ticketID))
	}

	// Get alerts for the ticket
	var alerts alert.Alerts
	if len(ticket.AlertIDs) > 0 {
		alerts, err = u.repository.BatchGetAlerts(ctx, ticket.AlertIDs)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to get alerts", goerr.V("ticket_id", ticketID))
		}
	}

	// Generate JSONL content with only alert data
	var jsonlData []byte

	for _, alert := range alerts {
		if alert.Data != nil {
			// Parse the alert data
			var data interface{}
			if dataStr, ok := alert.Data.(string); ok {
				if err := json.Unmarshal([]byte(dataStr), &data); err != nil {
					// If parsing fails, use the raw data
					data = dataStr
				}
			} else {
				// Data is already an interface{}, use it directly
				data = alert.Data
			}

			// Encode only the data part to JSONL
			recordBytes, err := json.Marshal(data)
			if err != nil {
				continue // Skip this record if encoding fails
			}

			jsonlData = append(jsonlData, recordBytes...)
			jsonlData = append(jsonlData, '\n')
		}
	}

	return jsonlData, nil
}
