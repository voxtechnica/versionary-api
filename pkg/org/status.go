package org

import (
	"fmt"
	"strings"
)

// Status indicates the operational state of a User or Organization
type Status string

// PENDING Status indicates that further verification is required (e.g. verified email address)
const PENDING Status = "PENDING"

// ENABLED Status indicates that the User or Organization is active
const ENABLED Status = "ENABLED"

// DISABLED Status indicates that the User or Organization is "turned off" (but not deleted)
const DISABLED Status = "DISABLED"

// Statuses is the complete list of valid User or Organization statuses
var Statuses = []Status{PENDING, ENABLED, DISABLED}

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

// ParseStatus returns a Status from a string representation.
// It validates the string before returning the Status.
func ParseStatus(s string) (Status, error) {
	status := Status(strings.ToUpper(s))
	if status.IsValid() {
		return status, nil
	}
	return "", fmt.Errorf("invalid status: %s", s)
}
