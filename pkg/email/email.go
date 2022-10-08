package email

import (
	"errors"
	"fmt"
	"github.com/voxtechnica/tuid-go"
	v "github.com/voxtechnica/versionary"
	"net/mail"
	"strings"
	"time"
)

// Identity represents an email address and optional name.
type Identity struct {
	Name    string `json:"name,omitempty"` // Full name of the person/identity
	Address string `json:"address"`        // Email address of the person/identity
}

// IsValid checks whether the Email Identity has all required fields.
// Use NewIdentity to verify that the email address itself is valid.
func (i Identity) IsValid() bool {
	return i.Address != ""
}

// String returns a string representation of the Email Address.
func (i Identity) String() string {
	if i.Name != "" {
		return i.Name + " <" + i.Address + ">"
	}
	return i.Address
}

// NewIdentity parses the supplied RFC 5322/6532 email address into an Identity.
// These email addresses contain an optional display name and an email address.
// A supplied name is preferred over a parsed display name.
func NewIdentity(name, address string) (Identity, error) {
	i := Identity{
		Name:    strings.TrimSpace(name),
		Address: strings.TrimSpace(address),
	}
	if i.Address == "" {
		return i, errors.New("missing email address")
	}
	a, err := mail.ParseAddress(i.Address)
	if err != nil {
		return i, fmt.Errorf("invalid email address %s: %w", address, err)
	}
	if i.Name == "" {
		i.Name = (*a).Name
	}
	i.Address = strings.ToLower((*a).Address)
	return i, nil
}

// Email represents an email message.
type Email struct {
	ID           string     `json:"id"`
	CreatedAt    time.Time  `json:"createdAt"`
	VersionID    string     `json:"versionID"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	From         Identity   `json:"from"`
	To           []Identity `json:"to"`
	CC           []Identity `json:"cc,omitempty"`
	BCC          []Identity `json:"bcc,omitempty"`
	Subject      string     `json:"subject"`
	BodyText     string     `json:"bodyText"`
	BodyHTML     string     `json:"bodyHTML,omitempty"`
	EventMessage string     `json:"eventMessage,omitempty"`
	Status       Status     `json:"status"`
}

// Type returns the entity type of the Email.
func (e Email) Type() string {
	return "Email"
}

// AllAddresses returns a list of all email addresses in the Email message.
func (e Email) AllAddresses() []string {
	var addresses []string
	if e.From.Address != "" {
		addresses = append(addresses, e.From.Address)
	}
	for _, i := range e.To {
		if i.Address != "" {
			addresses = append(addresses, i.Address)
		}
	}
	for _, i := range e.CC {
		if i.Address != "" {
			addresses = append(addresses, i.Address)
		}
	}
	for _, i := range e.BCC {
		if i.Address != "" {
			addresses = append(addresses, i.Address)
		}
	}
	return addresses
}

// IsParticipant checks whether the supplied email address is a participant in the Email.
func (e Email) IsParticipant(address string) bool {
	a := strings.ToLower(strings.TrimSpace(address))
	if a == "" {
		return false
	}
	for _, i := range e.To {
		if i.Address == a {
			return true
		}
	}
	for _, i := range e.CC {
		if i.Address == a {
			return true
		}
	}
	for _, i := range e.BCC {
		if i.Address == a {
			return true
		}
	}
	return false
}

// CompressedJSON returns a compressed JSON representation of the Email.
func (e Email) CompressedJSON() []byte {
	j, err := v.ToCompressedJSON(e)
	if err != nil {
		return nil
	}
	return j
}

// Validate checks whether the Email has all required fields and whether
// the supplied values are valid, returning a list of problems. If the list is
// empty, then the Email is valid.
func (e Email) Validate() []string {
	var problems []string
	if e.ID == "" || !tuid.IsValid(tuid.TUID(e.ID)) {
		problems = append(problems, "ID is missing or invalid")
	}
	if e.CreatedAt.IsZero() {
		problems = append(problems, "CreatedAt is missing")
	}
	if e.VersionID == "" || !tuid.IsValid(tuid.TUID(e.VersionID)) {
		problems = append(problems, "VersionID is missing or invalid")
	}
	if e.UpdatedAt.IsZero() {
		problems = append(problems, "UpdatedAt is missing")
	}
	if !e.From.IsValid() {
		problems = append(problems, "From address is missing or invalid")
	}
	if len(e.To) == 0 {
		problems = append(problems, "To recipients are missing")
	}
	for _, i := range e.To {
		if !i.IsValid() {
			problems = append(problems, "To recipient address is missing or invalid")
		}
	}
	for _, i := range e.CC {
		if !i.IsValid() {
			problems = append(problems, "CC recipient address is missing or invalid")
		}
	}
	for _, i := range e.BCC {
		if !i.IsValid() {
			problems = append(problems, "BCC recipient address is missing or invalid")
		}
	}
	if e.Subject == "" {
		problems = append(problems, "Subject is missing")
	}
	if e.BodyText == "" {
		problems = append(problems, "Body is missing")
	}
	if !e.Status.IsValid() {
		problems = append(problems, "Status is missing or invalid")
	}
	return problems
}
