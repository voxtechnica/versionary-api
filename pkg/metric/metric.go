package metric

import (
	"fmt"
	"strings"
	"time"
	"versionary-api/pkg/ref"

	"github.com/voxtechnica/tuid-go"
	"github.com/voxtechnica/versionary"
)

// Metric is a model entity useful for understanding system activity in terms of performance.
// Metric is never updated; it has no version. We just record the stats.
type Metric struct {
	ID         string    `json:"id"`
	CreatedAt  time.Time `json:"createdAt"`
	ExpiresAt  time.Time `json:"expiresAt"`
	Title      string    `json:"title"`
	Label      string    `json:"label,omitempty"`
	EntityID   string    `json:"entityId,omitempty"`
	EntityType string    `json:"entityType,omitempty"`
	Tags       []string  `json:"tags,omitempty"`
	Value      float64   `json:"value"`
	Units      string    `json:"units"`
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
	if m.ExpiresAt.IsZero() {
		problems = append(problems, "ExpiresAt is missing")
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
	if m.Units == "" {
		problems = append(problems, "Units is missing")
	}
	return problems
}

// String method to include nice string representation of Metric
func (m Metric) String() string {
	return fmt.Sprintf(
		"Metric{"+
			"Title: %s, "+
			"Label: %s, "+
			"CreatedAt: %s, "+
			"EntityType: %s, "+
			"Tags: %s, "+
			"Value: %.4f, "+ // Format Value to 4 decimal places
			"Units: %s}",
		m.Title, m.Label, m.CreatedAt.Format("2006-01-02"), m.EntityType, strings.Join(m.Tags, " "), m.Value, m.Units)
}

// MetricStat is a model entity to dynamically aggregate metrics.
type MetricStat struct {
	EntityID   string  `json:"entityId,omitempty"`
	EntityType string  `json:"entityType,omitempty"`
	Tag        string  `json:"tag,omitempty"`
	Count      int64   `json:"count"`
	Sum        float64 `json:"sum"`
	Min        float64 `json:"min"`
	Max        float64 `json:"max"`
	Mean       float64 `json:"mean"`
	Median     float64 `json:"median"`
	StdDev     float64 `json:"stdDev"`
}
