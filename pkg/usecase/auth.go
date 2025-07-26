package usecase

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/auth"
	"github.com/secmon-lab/warren/pkg/service/slack"
)

type AuthUseCase struct {
	repo         interfaces.Repository
	slackSvc     *slack.Service
	clientID     string
	clientSecret string
	callbackURL  string
	cache        *authCache
}

func NewAuthUseCase(repo interfaces.Repository, slackSvc *slack.Service, clientID, clientSecret, callbackURL string) *AuthUseCase {
	return &AuthUseCase{
		repo:         repo,
		slackSvc:     slackSvc,
		clientID:     clientID,
		clientSecret: clientSecret,
		callbackURL:  callbackURL,
		cache:        newAuthCache(),
	}
}

// OpenIDConfiguration represents Slack's OpenID Connect configuration
type OpenIDConfiguration struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	UserinfoEndpoint                  string   `json:"userinfo_endpoint"`
	JWKSURI                           string   `json:"jwks_uri"`
	ScopesSupported                   []string `json:"scopes_supported"`
	ResponseTypesSupported            []string `json:"response_types_supported"`
	ResponseModesSupported            []string `json:"response_modes_supported"`
	GrantTypesSupported               []string `json:"grant_types_supported"`
	SubjectTypesSupported             []string `json:"subject_types_supported"`
	IDTokenSigningAlgValuesSupported  []string `json:"id_token_signing_alg_values_supported"`
	ClaimsSupported                   []string `json:"claims_supported"`
	ClaimsParameterSupported          bool     `json:"claims_parameter_supported"`
	RequestParameterSupported         bool     `json:"request_parameter_supported"`
	RequestURIParameterSupported      bool     `json:"request_uri_parameter_supported"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
}

// GetAuthURL returns the URL for Slack OAuth
func (uc *AuthUseCase) GetAuthURL(state string) string {
	params := url.Values{}
	params.Set("client_id", uc.clientID)
	params.Set("scope", "openid,email,profile")
	params.Set("redirect_uri", uc.callbackURL)
	params.Set("response_type", "code")
	params.Set("state", state)
	if uc.slackSvc != nil {
		params.Set("team", uc.slackSvc.TeamID())
	}

	return "https://slack.com/openid/connect/authorize?" + params.Encode()
}

// IsNoAuthn returns false for regular AuthUseCase
func (uc *AuthUseCase) IsNoAuthn() bool {
	return false
}

// SlackTokenResponse represents the response from Slack token exchange
type SlackTokenResponse struct {
	OK          bool   `json:"ok"`
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	BotUserID   string `json:"bot_user_id"`
	AppID       string `json:"app_id"`
	Team        struct {
		Name string `json:"name"`
		ID   string `json:"id"`
	} `json:"team"`
	Enterprise interface{} `json:"enterprise"`
	AuthedUser struct {
		ID          string `json:"id"`
		Scope       string `json:"scope"`
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	} `json:"authed_user"`
	IDToken string `json:"id_token"`
	Error   string `json:"error"`
}

// SlackIDToken represents the decoded ID token from Slack
type SlackIDToken struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// HandleCallback processes the OAuth callback
func (uc *AuthUseCase) HandleCallback(ctx context.Context, code string) (*auth.Token, error) {
	// Exchange code for access token
	tokenResp, err := uc.exchangeCodeForToken(code)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to exchange code for token")
	}

	if !tokenResp.OK || tokenResp.Error != "" {
		return nil, goerr.New("slack oauth error", goerr.V("error", tokenResp.Error))
	}

	// Decode and verify ID token
	idToken, err := uc.decodeIDToken(ctx, tokenResp.IDToken)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to decode ID token")
	}

	// Create and store token
	token := auth.NewToken(idToken.Sub, idToken.Email, idToken.Name)
	if err := uc.repo.PutToken(ctx, token); err != nil {
		return nil, goerr.Wrap(err, "failed to store token")
	}

	return token, nil
}

// exchangeCodeForToken exchanges the authorization code for an access token
func (uc *AuthUseCase) exchangeCodeForToken(code string) (*SlackTokenResponse, error) {
	data := url.Values{}
	data.Set("client_id", uc.clientID)
	data.Set("client_secret", uc.clientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", uc.callbackURL)

	resp, err := http.PostForm("https://slack.com/api/openid.connect.token", data)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to make token request")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to read response body")
	}

	var tokenResp SlackTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, goerr.Wrap(err, "failed to parse token response")
	}

	return &tokenResp, nil
}

// getOpenIDConfiguration fetches Slack's OpenID Connect configuration
func (uc *AuthUseCase) getOpenIDConfiguration(ctx context.Context) (*OpenIDConfiguration, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://slack.com/.well-known/openid-configuration", nil)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create request")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to fetch OpenID configuration")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, goerr.New("failed to fetch OpenID configuration", goerr.V("status", resp.StatusCode))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to read OpenID configuration response")
	}

	var config OpenIDConfiguration
	if err := json.Unmarshal(body, &config); err != nil {
		return nil, goerr.Wrap(err, "failed to parse OpenID configuration")
	}

	return &config, nil
}

// decodeIDToken decodes and verifies the ID token using Slack's public keys
func (uc *AuthUseCase) decodeIDToken(ctx context.Context, idToken string) (*SlackIDToken, error) {
	// Get OpenID Connect configuration to find JWKS URI
	config, err := uc.getOpenIDConfiguration(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get OpenID configuration")
	}

	// Fetch Slack's public JWK set from the discovered URI
	keySet, err := jwk.Fetch(ctx, config.JWKSURI)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to fetch Slack's public keys", goerr.V("jwks_uri", config.JWKSURI))
	}

	// Parse and verify the JWT token
	token, err := jwt.Parse([]byte(idToken), jwt.WithKeySet(keySet), jwt.WithValidate(true), jwt.WithAudience(uc.clientID))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to parse or verify JWT token")
	}

	// Extract claims
	sub, ok := token.Get("sub")
	if !ok {
		return nil, goerr.New("sub claim not found in token")
	}

	email, ok := token.Get("email")
	if !ok {
		return nil, goerr.New("email claim not found in token")
	}

	name, ok := token.Get("name")
	if !ok {
		return nil, goerr.New("name claim not found in token")
	}

	// Convert to string values
	subStr, ok := sub.(string)
	if !ok {
		return nil, goerr.New("sub claim is not a string")
	}

	emailStr, ok := email.(string)
	if !ok {
		return nil, goerr.New("email claim is not a string")
	}

	nameStr, ok := name.(string)
	if !ok {
		return nil, goerr.New("name claim is not a string")
	}

	return &SlackIDToken{
		Sub:   subStr,
		Email: emailStr,
		Name:  nameStr,
	}, nil
}

// ValidateToken validates the token and returns user info
func (uc *AuthUseCase) ValidateToken(ctx context.Context, tokenID auth.TokenID, tokenSecret auth.TokenSecret) (*auth.Token, error) {
	return uc.validateTokenWithCache(ctx, tokenID, tokenSecret)
}

// Logout deletes the token
func (uc *AuthUseCase) Logout(ctx context.Context, tokenID auth.TokenID) error {
	// Remove from cache first
	uc.cache.remove(tokenID)

	// Then remove from repository
	return uc.repo.DeleteToken(ctx, tokenID)
}
