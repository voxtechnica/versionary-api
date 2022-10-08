package device

import (
	"time"

	"github.com/voxtechnica/tuid-go"
	ua "github.com/voxtechnica/user-agent"
	v "github.com/voxtechnica/versionary"
)

// A Device is based on a user's User-Agent header, which inspected to identify
// the device type, operating system, and client application in use.
type Device struct {
	ID         string       `json:"id"`
	CreatedAt  time.Time    `json:"createdAt"`
	VersionID  string       `json:"versionID"`
	UpdatedAt  time.Time    `json:"updatedAt"`
	LastSeenAt time.Time    `json:"lastSeen"`
	ExpiresAt  time.Time    `json:"expiresAt"`
	UserID     string       `json:"userId,omitempty"`
	UserAgent  ua.UserAgent `json:"userAgent"`
}

// Type returns the entity type of the Device.
func (d Device) Type() string {
	return "Device"
}

// LastSeenOn returns an ISO-8601 formatted string of the LastSeenAt time.
func (d Device) LastSeenOn() string {
	if d.LastSeenAt.IsZero() {
		return ""
	}
	return d.LastSeenAt.Format("2006-01-02")
}

// CompressedJSON returns a compressed JSON representation of the Device.
func (d Device) CompressedJSON() []byte {
	j, err := v.ToCompressedJSON(d)
	if err != nil {
		return nil
	}
	return j
}

// Validate checks whether the Device has all required fields and whether
// the supplied values are valid, returning a list of problems. If the list is
// empty, then the Device is valid.
func (d Device) Validate() []string {
	var problems []string
	if d.ID == "" || !tuid.IsValid(tuid.TUID(d.ID)) {
		problems = append(problems, "ID is missing or invalid")
	}
	if d.CreatedAt.IsZero() {
		problems = append(problems, "CreatedAt is missing")
	}
	if d.VersionID == "" || !tuid.IsValid(tuid.TUID(d.VersionID)) {
		problems = append(problems, "VersionID is missing or invalid")
	}
	if d.UpdatedAt.IsZero() {
		problems = append(problems, "UpdatedAt is missing")
	}
	if d.LastSeenAt.IsZero() {
		problems = append(problems, "LastSeenAt is missing")
	}
	if d.ExpiresAt.IsZero() {
		problems = append(problems, "ExpiresAt is missing")
	}
	if d.UserID != "" && !tuid.IsValid(tuid.TUID(d.UserID)) {
		problems = append(problems, "UserID is invalid")
	}
	return problems
}
