package user

import (
	"crypto/sha256"
	"encoding/hex"
	"net/mail"
	"strings"
	"time"
	"versionary-api/pkg/ref"

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
	GivenName     string    `json:"givenName"`
	FamilyName    string    `json:"familyName"`
	Email         string    `json:"email"`
	Password      string    `json:"password,omitempty"`
	PasswordHash  string    `json:"passwordHash,omitempty"`
	PasswordReset string    `json:"passwordReset,omitempty"`
	Roles         []string  `json:"roles,omitempty"`
	OrgID         string    `json:"orgID,omitempty"`
	OrgName       string    `json:"orgName,omitempty"`
	AvatarURL     string    `json:"avatarURL,omitempty"`
	WebsiteURL    string    `json:"websiteURL,omitempty"`
	Status        Status    `json:"status"`
}

// Type returns the entity type of the User.
func (u User) Type() string {
	return "User"
}

// RefID returns the Reference ID of the entity.
func (u User) RefID() ref.RefID {
	r, _ := ref.NewRefID(u.Type(), u.ID, u.VersionID)
	return r
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

// RestoreScrubbed restores the User's sensitive information from the supplied User version.
// Note that Password is never stored in the database, and does not need to be restored.
func (u User) RestoreScrubbed(user User) User {
	u.PasswordHash = user.PasswordHash
	u.PasswordReset = user.PasswordReset
	return u
}

// FullName returns the User's full name.
func (u User) FullName() string {
	return strings.TrimSpace(u.GivenName + " " + u.FamilyName)
}

// String returns a minimal string representation of the User (an RFC 5322 email address).
func (u User) String() string {
	f := u.FullName()
	if f == "" {
		return u.Email
	}
	if u.Email == "" {
		return f
	}
	return f + " <" + u.Email + ">"
}

// Validate checks whether the User has all required fields and whether the supplied values are valid,
// returning a list of problems. If the list is empty, then the User is valid.
func (u User) Validate() []string {
	var problems []string
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
		statuses := v.Map(Statuses, func(s Status) string { return string(s) })
		expected := strings.Join(statuses, ", ")
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

// ValidPassword checks the supplied clear-text password against stored password hash.
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

// StandardizeEmail returns the User's email address in a standard format.
// This method is used primarily to standardize email addresses for indexing.
func StandardizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
