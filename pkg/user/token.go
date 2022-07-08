package user

import (
	"time"

	"github.com/voxtechnica/tuid-go"
	v "github.com/voxtechnica/versionary"
)

// Token models an OAuth2 Resource Owner Password Credentials Grant in this system.
// For more information, see http://tools.ietf.org/html/rfc6749#section-4.3
type Token struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
	UserID    string    `json:"userId"`
	Email     string    `json:"email,omitempty"`
}

// Type returns the entity type of the Token.
func (t Token) Type() string {
	return "Token"
}

// CompressedJSON returns a compressed JSON representation of the Token.
func (t Token) CompressedJSON() []byte {
	j, err := v.ToCompressedJSON(t)
	if err != nil {
		return nil
	}
	return j
}

func (t Token) Validate() []string {
	problems := []string{}
	if t.ID == "" || !tuid.IsValid(tuid.TUID(t.ID)) {
		problems = append(problems, "ID is missing or invalid")
	}
	if t.CreatedAt.IsZero() {
		problems = append(problems, "CreatedAt is missing")
	}
	if t.ExpiresAt.IsZero() {
		problems = append(problems, "ExpiresAt is missing")
	}
	if t.UserID == "" || !tuid.IsValid(tuid.TUID(t.UserID)) {
		problems = append(problems, "UserID is missing or invalid")
	}
	return problems
}

// TokenRequest provides a loose interpretation of the OAuth 2 Specification. It's "loose", because
// we're allowing Content-Type application/json instead of application/x-www-form-urlencoded, and
// because we're using camelCase instead of snake_case.
// For more information, see http://tools.ietf.org/html/rfc6749#section-4.3.2
type TokenRequest struct {
	GrantType string `json:"grantType"` // usually "password"
	Username  string `json:"username"`  // email or User ID
	Password  string `json:"password"`  // plaintext password
}

// TokenResponse provides a Bearer Token Response in a loose interpretation of the OAuth 2 Specification.
// It's "loose", because we're using camelCase instead of snake_case.
// Also, we're providing an expiration timestamp instead of a duration in seconds.
// For more information, see https://datatracker.ietf.org/doc/html/rfc6749#section-5.1
type TokenResponse struct {
	AccessToken string    `json:"accessToken"` // Token ID (a tuid.TUID)
	TokenType   string    `json:"tokenType"`   // usually "Bearer"
	ExpiresAt   time.Time `json:"expiresAt"`   // when the token expires (DynamoDB TTL)
}
