package ref

import (
	"errors"
	"fmt"
	"strings"

	"github.com/voxtechnica/tuid-go"
)

// RefID represents a reference to a specific entity. If the VersionID is empty, the
// reference is to the latest version of the entity. The segments are hyphenated, as
// in "Content-9Jbuygs7UM87hafJ" or "Content-9Jbuygs7UM87hafJ-9Jbv7xzh2rF9DClo".
type RefID struct {
	EntityType string `json:"entityType"` // Entity type (e.g. "Content")
	EntityID   string `json:"entityId"`   // Entity ID (a TUID)
	VersionID  string `json:"versionId"`  // Version ID (a TUID; optional)
}

// IsValid returns true if the RefID is minimally functional.
func (r RefID) IsValid() bool {
	return r.EntityType != "" && tuid.IsValid(tuid.TUID(r.EntityID)) &&
		(r.VersionID == "" || tuid.IsValid(tuid.TUID(r.VersionID)))
}

// IsEmpty returns true if the RefID has no valid reference.
func (r RefID) IsEmpty() bool {
	return !r.IsValid()
}

// String returns a string representation of the RefID.
func (r RefID) String() string {
	if !r.IsValid() {
		return ""
	}
	if r.VersionID == "" {
		return r.EntityType + "-" + r.EntityID
	}
	return r.EntityType + "-" + r.EntityID + "-" + r.VersionID
}

// NewRefID creates a new RefID from the given entity type and ID. If the version ID is
// empty, the reference is to the latest (or only) version of the entity.
func NewRefID(entityType, entityID, versionID string) (RefID, error) {
	var r RefID
	if entityType == "" || entityID == "" {
		return r, errors.New("entity type is required")
	}
	r.EntityType = entityType
	if entityID == "" {
		return r, errors.New("entity ID is required")
	}
	if !tuid.IsValid(tuid.TUID(entityID)) {
		return r, fmt.Errorf("entity ID %s is invalid", entityID)
	}
	r.EntityID = entityID
	if versionID != "" && !tuid.IsValid(tuid.TUID(versionID)) {
		return r, fmt.Errorf("version ID %s is invalid", versionID)
	}
	r.VersionID = versionID
	return r, nil
}

// Parse parses a string representation of a RefID.
func Parse(s string) (RefID, error) {
	var r RefID
	parts := strings.Split(s, "-")
	if len(parts) < 2 || len(parts) > 3 {
		return r, errors.New("invalid RefID")
	}
	r.EntityType = parts[0]
	r.EntityID = parts[1]
	if len(parts) == 3 {
		r.VersionID = parts[2]
	}
	return r, nil
}
