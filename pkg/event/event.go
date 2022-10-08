package event

import (
	"time"

	"github.com/voxtechnica/tuid-go"
	v "github.com/voxtechnica/versionary"
)

// Event is a model entity useful for understanding user and system activity, and for debugging when things go badly.
// Events are never updated; they have no versions. We just record what happened. The list of associated entity IDs
// should be kept short, unless they're truly helpful. If the Event was triggered by an Error, it can contain the
// wrapped Error.
type Event struct {
	ID         string    `json:"id"`
	CreatedAt  time.Time `json:"createdAt"`
	ExpiresAt  time.Time `json:"expiresAt"`
	UserID     string    `json:"userId,omitempty"`
	EntityID   string    `json:"entityId,omitempty"`
	EntityType string    `json:"entityType,omitempty"`
	OtherIDs   []string  `json:"otherIds,omitempty"`
	LogLevel   LogLevel  `json:"logLevel"`
	Message    string    `json:"message"`
	URI        string    `json:"uri,omitempty"`
	Err        error     `json:"-"`
}

// Type returns the entity type of the Event.
func (e Event) Type() string {
	return "Event"
}

// CreatedOn returns an ISO-8601 formatted string of the event's creation date.
func (e Event) CreatedOn() string {
	if e.CreatedAt.IsZero() {
		return ""
	}
	return e.CreatedAt.Format("2006-01-02")
}

// IDs returns a list of entity IDs associated with the event, excluding the Event ID.
func (e Event) IDs() []string {
	var ids []string
	if e.UserID != "" {
		ids = append(ids, e.UserID)
	}
	if e.EntityID != "" {
		ids = append(ids, e.EntityID)
	}
	for _, id := range e.OtherIDs {
		if id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

// CompressedJSON returns a compressed JSON representation of the event.
func (e Event) CompressedJSON() []byte {
	j, err := v.ToCompressedJSON(e)
	if err != nil {
		// skip persistence if we can't serialize and compress the event
		return nil
	}
	return j
}

// Error returns an error representation of the event.
func (e Event) Error() string {
	return "Event-" + e.ID + " " + e.Message
}

// Unwrap returns the Error that triggered the event.
func (e Event) Unwrap() error {
	return e.Err
}

// Validate checks whether the Event has all required fields and whether the supplied values
// are valid, returning a list of problems. If the list is empty, then the Event is valid.
func (e Event) Validate() []string {
	var problems []string
	if e.ID == "" || !tuid.IsValid(tuid.TUID(e.ID)) {
		problems = append(problems, "ID is missing or invalid")
	}
	if e.CreatedAt.IsZero() {
		problems = append(problems, "CreatedAt is missing")
	}
	if e.ExpiresAt.IsZero() {
		problems = append(problems, "ExpiresAt is missing")
	}
	if e.UserID != "" && !tuid.IsValid(tuid.TUID(e.UserID)) {
		problems = append(problems, "UserID is not a TUID")
	}
	if e.EntityID != "" && !tuid.IsValid(tuid.TUID(e.EntityID)) {
		problems = append(problems, "EntityID is not a TUID")
	}
	if !e.LogLevel.IsValid() {
		problems = append(problems, "LogLevel is missing or invalid")
	}
	if e.Message == "" {
		problems = append(problems, "Message is missing")
	}
	return problems
}
