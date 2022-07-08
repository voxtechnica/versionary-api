package user

import (
	"crypto/sha256"
	"encoding/hex"
	"net/mail"
	"strings"
	"time"

	"github.com/voxtechnica/tuid-go"
	v "github.com/voxtechnica/versionary"
)

// User represents a person or system that accesses this information system.
// It plays a key role in authenticating identity and authorizing actions (roles).
type User struct {
	ID            string    `json:"id"`
	CreatedAt     time.Time `json:"createdAt"`
	VersionID     string    `json:"versionID"`
	UpdatedAt     time.Time `json:"updatedAt"`
	FirstName     string    `json:"firstName"`
	LastName      string    `json:"lastName"`
	Email         string    `json:"email"`
	Password      string    `json:"password,omitempty"`
	PasswordHash  string    `json:"passwordHash,omitempty"`
	PasswordReset string    `json:"passwordReset,omitempty"`
	Roles         []string  `json:"roles,omitempty"`
	OrgID         string    `json:"orgID"`
	OrgName       string    `json:"orgName"`
	Status        Status    `json:"status"`
}

// Type returns the entity type of the User.
func (u User) Type() string {
	return "User"
}

// CompressedJSON returns a compressed JSON representation of the User.
func (u User) CompressedJSON() []byte {
	j, err := v.ToCompressedJSON(u)
	if err != nil {
		return nil
	}
	return j
}

// HasRole returns true if the user has the specified role.
// Administrators are like janitors... they have all the keys.
func (u User) HasRole(role string) bool {
	for _, r := range u.Roles {
		if r == role || r == "admin" {
			return true
		}
	}
	return false
}

// Scrub removes sensitive information from the User.
func (u User) Scrub() User {
	u.Password = ""
	u.PasswordHash = ""
	u.PasswordReset = ""
	return u
}

// String returns a minimal string representation of the User (an RFC 5322 email address).
func (u User) String() string {
	if u.FirstName == "" && u.LastName == "" {
		return u.Email
	}
	return u.FirstName + " " + u.LastName + " <" + u.Email + ">"
}

// Validate checks whether the User has all required fields and whether the supplied values are valid,
// returning a list of problems. If the list is empty, then the User is valid.
func (u User) Validate() []string {
	problems := []string{}
	if u.ID == "" || !tuid.IsValid(tuid.TUID(u.ID)) {
		problems = append(problems, "ID is missing or invalid")
	}
	if u.CreatedAt.IsZero() {
		problems = append(problems, "CreatedAt is missing")
	}
	if u.VersionID == "" || !tuid.IsValid(tuid.TUID(u.VersionID)) {
		problems = append(problems, "VersionID is missing or invalid")
	}
	if u.UpdatedAt.IsZero() {
		problems = append(problems, "UpdatedAt is missing")
	}
	if !u.ValidEmail() {
		problems = append(problems, "Email is missing or invalid")
	}
	if u.OrgID != "" && !tuid.IsValid(tuid.TUID(u.OrgID)) {
		problems = append(problems, "OrgID is not a TUID")
	}
	if u.Status == "" || !u.Status.IsValid() {
		expected := strings.Join(Statuses, ", ")
		problems = append(problems, "Status is missing or invalid. Expecting: "+expected)
	}
	return problems
}

// ValidEmail returns true if the User's email address is valid.
func (u User) ValidEmail() bool {
	if u.Email == "" {
		return false
	}
	_, err := mail.ParseAddress(u.Email)
	return err == nil
}

// ValidPassword checks the supplied clear-text pashword against stored password hash.
func (u User) ValidPassword(password string) bool {
	return u.ID != "" && password != "" && u.PasswordHash != "" &&
		hashPassword(u.ID, password) == u.PasswordHash
}

// hashPassword produces a salted SHA256 hash of a User's ID and Password.
func hashPassword(id, password string) string {
	if id == "" && password == "" {
		return ""
	}
	hash := sha256.Sum256([]byte(id + password))
	return hex.EncodeToString(hash[:])
}

// standardizeEmail returns the User's email address in a standard format.
// This method is used primarily to standardize email addresses for indexing.
func standardizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
