package metric

import (
	"time"
	"versionary-api/pkg/ref"

	"github.com/voxtechnica/tuid-go"
	"github.com/voxtechnica/versionary"
)

// Metric is a model entity useful for understanding system activity in terms of performance.
type Metric struct {
	ID         string    `json:"id"`
	CreatedAt  time.Time `json:"createdAt"`
	Title      string    `json:"title"`
	Label      string    `json:"label,omitempty"`
	EntityID   string    `json:"entityId,omitempty"`
	EntityType string    `json:"entityType,omitempty"`
	Tags       []string  `json:"tags,omitempty"`
	Value      float64   `json:"value"`
	Units      string    `json:"units,omitempty"`
}

// Type returns the entity type of the Metric.
func (m Metric) Type() string {
	return "Metric"
}

// RefID returns the Reference ID of the entity.
func (m Metric) RefID() ref.RefID {
	r, _ := ref.NewRefID(m.Type(), m.ID, "")
	return r
}

// CompressedJSON returns a compressed JSON representation of the Metric.
func (m Metric) CompressedJSON() []byte {
	j, err := versionary.ToCompressedJSON(m)
	if err != nil {
		return nil
	}
	return j
}

// IDs returns a list of entity IDs associated with the metric, excluding the Metric ID.
func (m Metric) IDs() []string {
	var ids []string
	if m.EntityID != "" {
		ids = append(ids, m.EntityID)
	}
	return ids
}

// Validate checks whether the Metric has all required fields and whether
// the supplied values are valid, returning a list of problems. If the list is
// empty, then the Metric is valid.
func (m Metric) Validate() []string {
	var problems []string
	if m.ID == "" || !tuid.IsValid(tuid.TUID(m.ID)) {
		problems = append(problems, "ID is missing")
	}
	if m.CreatedAt.IsZero() {
		problems = append(problems, "CreatedAt is missing")
	}
	if m.Title == "" {
		problems = append(problems, "Title is missing")
	}
	if m.EntityID != "" && !tuid.IsValid(tuid.TUID(m.EntityID)) {
		problems = append(problems, "EntityID is not a TUID")
	}
	if m.Value == 0.0 {
		problems = append(problems, "Value is missing")
	}
	return problems
}
