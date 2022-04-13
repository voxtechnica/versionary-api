package event

import (
	"time"

	v "github.com/voxtechnica/versionary"
)

// Event is a model entity useful for understanding user and system activity, and for debugging when things go badly.
// Events are never updated; they have no versions. We just record what happened. The list of associated entity IDs
// should be kept short, unless they're truly meaningful (e.g. if someone gets a paginated list, don’t necessarily
// record every item in the list; maybe just the parent call).
type Event struct {
	ID         string    `json:"id"`
	UserID     string    `json:"userId"`
	EntityID   string    `json:"entityId"`
	EntityType string    `json:"entityType"`
	OtherIDs   []string  `json:"otherIds"`
	LogLevel   LogLevel  `json:"logLevel"`
	Message    string    `json:"message"`
	CreatedAt  time.Time `json:"createdAt"`
	ExpiresAt  time.Time `json:"expiresAt"`
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

// Error returns an error representation of the event.
func (e Event) Error() string {
	return e.Message
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
