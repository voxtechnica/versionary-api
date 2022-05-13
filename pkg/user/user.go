package user

import (
	"crypto/sha256"
	"encoding/hex"
	"net/mail"
	"strings"
	"time"

	v "github.com/voxtechnica/versionary"
)

type User struct {
	ID            string    `json:"id"`
	UpdateID      string    `json:"updateId"`
	FirstName     string    `json:"firstName"`
	LastName      string    `json:"lastName"`
	Email         string    `json:"email"`
	Password      string    `json:"password"`
	PasswordHash  string    `json:"passwordHash"`
	PasswordReset string    `json:"passwordReset"`
	Roles         []string  `json:"roles"`
	GroupIDs      []string  `json:"groups"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// CompressedJSON returns a compressed JSON representation of the User.
func (u User) CompressedJSON() []byte {
	j, err := v.ToCompressedJSON(u)
	if err != nil {
		// skip persistence if we can't serialize and compress the event
		return nil
	}
	return j
}

// EmailStandardized returns the user's email address in a consistent format, avoiding case-sensitivity issues.
func (u User) EmailStandardized() string {
	return strings.ToLower(strings.TrimSpace(u.Email))
}

// HasRole returns true if the user has the specified role.
func (u User) HasRole(role string) bool {
	for _, r := range u.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HashPassword produces a SHA256 hash of a User's Password.
func HashPassword(password string) string {
	if password == "" {
		return ""
	}
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

// ValidEmail returns true if the User's email address is valid.
func (u User) ValidEmail() bool {
	_, err := mail.ParseAddress(u.Email)
	return err == nil
}
