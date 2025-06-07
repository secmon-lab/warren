package slack

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/slack-go/slack"
)

// AlertUpdateRequest represents a request to update an alert message
type AlertUpdateRequest struct {
	Alert alert.Alert
}

// AlertUpdater defines the interface for updating alert messages with rate limiting
type AlertUpdater interface {
	UpdateAlert(ctx context.Context, alert alert.Alert)
	Stop()
}

// RateLimitedUpdater provides rate-limited alert message updates
type RateLimitedUpdater struct {
	client      interfaces.SlackClient
	requestChan chan *AlertUpdateRequest
	once        sync.Once
	started     bool
	mu          sync.RWMutex

	// Rate limiting configuration
	interval      time.Duration
	retryInterval time.Duration // Base interval for exponential backoff

	// Graceful shutdown
	ctx    context.Context
	cancel context.CancelFunc
}

// UpdaterOption represents a configuration option for RateLimitedUpdater
type UpdaterOption func(*RateLimitedUpdater)

// WithInterval sets the rate limiting interval
func WithInterval(interval time.Duration) UpdaterOption {
	return func(r *RateLimitedUpdater) {
		r.interval = interval
	}
}

// WithRetryInterval sets the base retry interval for exponential backoff
func WithRetryInterval(interval time.Duration) UpdaterOption {
	return func(r *RateLimitedUpdater) {
		r.retryInterval = interval
	}
}

// NewRateLimitedUpdater creates a new rate-limited updater with optional configurations
func NewRateLimitedUpdater(client interfaces.SlackClient, opts ...UpdaterOption) AlertUpdater {
	ctx, cancel := context.WithCancel(context.Background())

	r := &RateLimitedUpdater{
		client:        client,
		requestChan:   make(chan *AlertUpdateRequest, 10), // Short buffer to prevent blocking
		interval:      2 * time.Second,                    // Default: 30 requests/min with safety margin
		retryInterval: 1 * time.Second,                    // Default: 1 second base for exponential backoff
		ctx:           ctx,
		cancel:        cancel,
	}

	// Apply options
	for _, opt := range opts {
		opt(r)
	}

	return r
}

// UpdateAlert queues an alert update request asynchronously
func (r *RateLimitedUpdater) UpdateAlert(ctx context.Context, alert alert.Alert) {
	// Ensure singleton goroutine is started
	r.once.Do(func() {
		r.mu.Lock()
		r.started = true
		r.mu.Unlock()
		go r.processRequests(r.ctx)
	})

	// Send request to processing goroutine asynchronously
	go func() {
		request := &AlertUpdateRequest{
			Alert: alert,
		}

		select {
		case r.requestChan <- request:
			// Request queued successfully
		case <-ctx.Done():
			// Context cancelled, log and exit
			logger := logging.From(ctx)
			logger.Warn("context cancelled while queuing alert update", "alert_id", alert.ID)
		}
	}()
}

// processRequests is the main processing loop for the singleton goroutine
func (r *RateLimitedUpdater) processRequests(ctx context.Context) {
	logger := logging.From(ctx)
	logger.Info("starting rate-limited alert updater")

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("stopping rate-limited alert updater")
			return
		case <-ticker.C:
			// Process one request per interval
			select {
			case request := <-r.requestChan:
				r.processRequest(ctx, request)
			default:
				// No requests to process
			}
		}
	}
}

// processRequest processes a single alert update request
func (r *RateLimitedUpdater) processRequest(ctx context.Context, request *AlertUpdateRequest) {
	logger := logging.From(ctx)

	defer func() {
		if r := recover(); r != nil {
			err := goerr.New("panic in alert update processing", goerr.V("panic", r))
			errs.Handle(ctx, err)
		}
	}()

	alert := request.Alert
	if alert.SlackThread == nil {
		err := goerr.New("alert has no slack thread")
		errs.Handle(ctx, err)
		return
	}

	blocks := buildAlertBlocks(alert)

	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		_, _, _, err := r.client.UpdateMessageContext(
			ctx,
			alert.SlackThread.ChannelID,
			alert.SlackThread.ThreadID,
			slack.MsgOptionBlocks(blocks...),
		)

		if err == nil {
			// Success
			logger.Debug("alert message updated successfully",
				"alert_id", alert.ID,
				"channel_id", alert.SlackThread.ChannelID,
				"thread_id", alert.SlackThread.ThreadID)
			return
		}

		// Handle rate limiting
		if r.isRateLimitError(err) {
			waitTime := r.extractRetryAfter(err)
			if waitTime == 0 {
				waitTime = time.Duration(attempt) * r.retryInterval // exponential backoff fallback
			}

			logger.Warn("rate limited, waiting before retry",
				"wait_time", waitTime,
				"attempt", attempt,
				"max_retries", maxRetries)

			time.Sleep(waitTime)
			continue
		}

		// For non-rate-limit errors, don't retry
		logger.Error("failed to update alert message",
			"error", err,
			"alert_id", alert.ID,
			"channel_id", alert.SlackThread.ChannelID,
			"thread_id", alert.SlackThread.ThreadID)

		wrappedErr := goerr.Wrap(err, "failed to update slack message",
			goerr.V("alert_id", alert.ID),
			goerr.V("channel_id", alert.SlackThread.ChannelID),
			goerr.V("thread_id", alert.SlackThread.ThreadID))

		errs.Handle(ctx, wrappedErr)
		return
	}

	// Max retries exceeded
	finalErr := goerr.New("max retries exceeded for alert update",
		goerr.V("alert_id", alert.ID),
		goerr.V("max_retries", maxRetries))

	errs.Handle(ctx, finalErr)
}

// isRateLimitError checks if the error is a rate limiting error
func (r *RateLimitedUpdater) isRateLimitError(err error) bool {
	// First check for RateLimitedError (most specific and standard)
	var rateLimitErr *slack.RateLimitedError
	if errors.As(err, &rateLimitErr) {
		return true
	}

	// Fallback to SlackErrorResponse
	var slackErr *slack.SlackErrorResponse
	if errors.As(err, &slackErr) {
		return slackErr.Err == "rate_limited"
	}

	return false
}

// extractRetryAfter extracts the retry-after duration from a rate limit error
func (r *RateLimitedUpdater) extractRetryAfter(err error) time.Duration {
	// First check for RateLimitedError which has parsed RetryAfter field
	var rateLimitErr *slack.RateLimitedError
	if errors.As(err, &rateLimitErr) {
		return rateLimitErr.RetryAfter
	}

	// Fallback to manual parsing from SlackErrorResponse
	var slackErr *slack.SlackErrorResponse
	if errors.As(err, &slackErr) {
		// Safely check if Messages slice has elements
		if len(slackErr.ResponseMetadata.Messages) > 0 {
			if retryAfterStr := slackErr.ResponseMetadata.Messages[0]; retryAfterStr != "" {
				if seconds, parseErr := strconv.Atoi(retryAfterStr); parseErr == nil {
					return time.Duration(seconds) * time.Second
				}
			}
		}
	}

	return 0
}

// Stop gracefully stops the rate-limited updater
func (r *RateLimitedUpdater) Stop() {
	if r.cancel != nil {
		r.cancel()
	}
}
