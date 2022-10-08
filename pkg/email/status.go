package email

// Status indicates the operational state of a message
type Status string

// PENDING Status indicates that the message has been created, but not yet sent.
const PENDING Status = "PENDING"

// SENT Status indicates that the message has been sent.
const SENT Status = "SENT"

// UNSENT Status indicates that the message has been created, but we chose not to send it.
// Some messages are created for informational purposes only, and are not sent (e.g. in non-production environments).
const UNSENT Status = "UNSENT"

// ERROR Status indicates that an error occurred and the message could not be sent.
const ERROR Status = "ERROR"

// Statuses is the complete list of valid User or Organization statuses
var Statuses = []Status{PENDING, SENT, UNSENT, ERROR}

// IsValid returns true if the supplied Status is recognized
func (s Status) IsValid() bool {
	for _, v := range Statuses {
		if s == v {
			return true
		}
	}
	return false
}

// String returns a string representation of the Status
func (s Status) String() string {
	return string(s)
}
