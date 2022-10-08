package org

import (
	"strings"
	"time"

	"github.com/voxtechnica/tuid-go"
	v "github.com/voxtechnica/versionary"
)

type Organization struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	VersionID string    `json:"versionID"`
	UpdatedAt time.Time `json:"updatedAt"`
	Name      string    `json:"name"`
	Status    Status    `json:"status"`
}

// Type returns the entity type of the Organization.
func (o Organization) Type() string {
	return "Organization"
}

// CompressedJSON returns a compressed JSON representation of the Organization.
func (o Organization) CompressedJSON() []byte {
	j, err := v.ToCompressedJSON(o)
	if err != nil {
		return nil
	}
	return j
}

// Validate checks whether the Organization has all required fields and whether
// the supplied values are valid, returning a list of problems. If the list is
// empty, then the Organization is valid.
func (o Organization) Validate() []string {
	var problems []string
	if o.ID == "" || !tuid.IsValid(tuid.TUID(o.ID)) {
		problems = append(problems, "ID is missing or invalid")
	}
	if o.CreatedAt.IsZero() {
		problems = append(problems, "CreatedAt is missing")
	}
	if o.VersionID == "" || !tuid.IsValid(tuid.TUID(o.VersionID)) {
		problems = append(problems, "VersionID is missing or invalid")
	}
	if o.UpdatedAt.IsZero() {
		problems = append(problems, "UpdatedAt is missing")
	}
	if o.Name == "" {
		problems = append(problems, "Name is missing")
	}
	if o.Status == "" || !o.Status.IsValid() {
		statuses := v.Map(Statuses, func(s Status) string { return string(s) })
		expected := strings.Join(statuses, ", ")
		problems = append(problems, "Status is missing or invalid. Expecting: "+expected)
	}
	return problems
}
