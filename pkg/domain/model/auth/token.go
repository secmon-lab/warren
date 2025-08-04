package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"time"

	"github.com/google/uuid"
	"github.com/m-mizutani/goerr/v2"
)

// Anonymous user constants
const (
	// Slack user IDs always start with "U", so "anonymous" won't conflict
	AnonymousUserID    = "anonymous"
	AnonymousUserName  = "Anonymous"
	AnonymousUserEmail = "anonymous@localhost"
)

type TokenID string

const TokenExpireDuration = 7 * 24 * time.Hour

func (x TokenID) String() string {
	return string(x)
}

func NewTokenID() TokenID {
	id, err := uuid.NewV7()
	if err != nil {
		panic(err)
	}
	return TokenID(id.String())
}

func (x TokenID) Validate() error {
	if x == "" {
		return goerr.New("empty token ID")
	}
	if _, err := uuid.Parse(string(x)); err != nil {
		return goerr.Wrap(err, "invalid token ID format")
	}
	return nil
}

type TokenSecret string

func (x TokenSecret) String() string {
	return string(x)
}

func NewTokenSecret() TokenSecret {
	// Generate 32 bytes (256 bits) of cryptographically secure random data
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		panic(goerr.Wrap(err, "failed to generate random token secret"))
	}

	// Encode to base64 URL-safe string (no padding)
	tokenString := base64.RawURLEncoding.EncodeToString(randomBytes)
	return TokenSecret(tokenString)
}

type Token struct {
	ID        TokenID     `json:"id"`
	Secret    TokenSecret `json:"secret" masq:"secret"`
	Sub       string      `json:"sub"`
	Email     string      `json:"email"`
	Name      string      `json:"name"`
	ExpiresAt time.Time   `json:"expires_at"`
	CreatedAt time.Time   `json:"created_at"`
}

func (x *Token) Validate() error {
	if err := x.ID.Validate(); err != nil {
		return goerr.Wrap(err, "invalid token ID")
	}
	if x.Secret == "" {
		return goerr.New("empty token secret")
	}
	if x.Sub == "" {
		return goerr.New("empty sub")
	}
	if x.Email == "" {
		return goerr.New("empty email")
	}
	if x.Name == "" {
		return goerr.New("empty name")
	}
	if x.ExpiresAt.IsZero() {
		return goerr.New("empty expires_at")
	}
	if x.CreatedAt.IsZero() {
		return goerr.New("empty created_at")
	}
	return nil
}

func (x *Token) IsExpired() bool {
	return time.Now().After(x.ExpiresAt)
}

func NewToken(sub, email, name string) *Token {
	now := time.Now()
	return &Token{
		ID:        NewTokenID(),
		Secret:    NewTokenSecret(),
		Sub:       sub,
		Email:     email,
		Name:      name,
		ExpiresAt: now.Add(TokenExpireDuration),
		CreatedAt: now,
	}
}

type ctxTokenKey struct{}

func TokenFromContext(ctx context.Context) (*Token, error) {
	token, ok := ctx.Value(ctxTokenKey{}).(*Token)
	if !ok {
		return nil, goerr.New("token not found in context")
	}
	return token, nil
}

func ContextWithToken(ctx context.Context, token *Token) context.Context {
	return context.WithValue(ctx, ctxTokenKey{}, token)
}

// NewAnonymousUser creates a new anonymous user token
func NewAnonymousUser() *Token {
	now := time.Now()
	return &Token{
		ID:        NewTokenID(),
		Secret:    NewTokenSecret(),
		Sub:       AnonymousUserID,
		Email:     AnonymousUserEmail,
		Name:      AnonymousUserName,
		ExpiresAt: now.Add(TokenExpireDuration),
		CreatedAt: now,
	}
}

// IsAnonymous returns true if the token represents an anonymous user
func (x *Token) IsAnonymous() bool {
	return x.Sub == AnonymousUserID
}
