package user

// Status indicates the operational state of a User or Organization
type Status string

// PENDING Status indicates that further verification is required (e.g. verified email address)
const PENDING Status = "PENDING"

// ENABLED Status indicates that the User or Organization is active
const ENABLED Status = "ENABLED"

// DISABLED Status indicates that the User or Organization is "turned off" (but not deleted)
const DISABLED Status = "DISABLED"

// Statuses is the complete list of valid User or Organization statuses
var Statuses = []string{"PENDING", "ENABLED", "DISABLED"}

// IsValid returns true if the supplied Status is recognized
func (s Status) IsValid() bool {
	for _, v := range Statuses {
		if string(s) == v {
			return true
		}
	}
	return false
}
