package auth

import (
	"time"

	"github.com/google/uuid"
	"github.com/m-mizutani/goerr/v2"
)

type TokenID string

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
	id, err := uuid.NewV7()
	if err != nil {
		panic(err)
	}
	return TokenSecret(id.String())
}

type Token struct {
	ID        TokenID     `json:"id" firestore:"-"`
	Secret    TokenSecret `json:"secret"`
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
		ExpiresAt: now.Add(7 * 24 * time.Hour), // 1週間後
		CreatedAt: now,
	}
}
