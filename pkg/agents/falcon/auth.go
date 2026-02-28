package falcon

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/safe"
)

// tokenProvider manages OAuth2 Client Credentials Flow for CrowdStrike Falcon API.
// It handles token acquisition and automatic refresh before expiry.
type tokenProvider struct {
	clientID     string
	clientSecret string
	baseURL      string
	httpClient   *http.Client

	mu     sync.Mutex
	token  string
	expiry time.Time
}

// tokenResponse represents the OAuth2 token response from CrowdStrike.
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

// newTokenProvider creates a new tokenProvider for CrowdStrike OAuth2 authentication.
func newTokenProvider(clientID, clientSecret, baseURL string) *tokenProvider {
	return &tokenProvider{
		clientID:     clientID,
		clientSecret: clientSecret,
		baseURL:      baseURL,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

// getToken returns a valid bearer token, refreshing if necessary.
func (tp *tokenProvider) getToken(ctx context.Context) (string, error) {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	// Return cached token if still valid (with 30-second buffer)
	if tp.token != "" && time.Now().Add(30*time.Second).Before(tp.expiry) {
		return tp.token, nil
	}

	if err := tp.refreshToken(ctx); err != nil {
		return "", err
	}

	return tp.token, nil
}

// clearToken invalidates the cached token, forcing a refresh on next getToken call.
func (tp *tokenProvider) clearToken() {
	tp.mu.Lock()
	defer tp.mu.Unlock()
	tp.token = ""
	tp.expiry = time.Time{}
}

// refreshToken acquires a new OAuth2 token from the CrowdStrike API.
func (tp *tokenProvider) refreshToken(ctx context.Context) error {
	log := logging.From(ctx)
	log.Debug("Refreshing CrowdStrike OAuth2 token",
		"base_url", tp.baseURL,
		"client_id", tp.clientID,
	)

	form := url.Values{
		"client_id":     {tp.clientID},
		"client_secret": {tp.clientSecret},
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		tp.baseURL+"/oauth2/token",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return goerr.Wrap(err, "failed to create token request")
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := tp.httpClient.Do(req)
	if err != nil {
		return goerr.Wrap(err, "failed to send token request")
	}
	defer safe.Close(ctx, resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return goerr.Wrap(err, "failed to read token response body")
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		log.Warn("CrowdStrike OAuth2 token request failed",
			"status", resp.StatusCode,
			"body", string(body),
		)
		return goerr.New("OAuth2 token request failed",
			goerr.V("status", resp.StatusCode),
			goerr.V("body", string(body)),
		)
	}

	var tokenResp tokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return goerr.Wrap(err, "failed to parse token response")
	}

	tp.token = tokenResp.AccessToken
	tp.expiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	log.Debug("CrowdStrike OAuth2 token refreshed", "expires_in", tokenResp.ExpiresIn)

	return nil
}
