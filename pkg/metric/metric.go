package metric

import (
	"math"
	"slices"
	"strconv"
	"strings"
	"time"
	"versionary-api/pkg/ref"

	"github.com/voxtechnica/tuid-go"
	v "github.com/voxtechnica/versionary"
)

// Metric is a model entity useful for understanding system activity in terms of performance.
// Metric is never updated; it has no version. We just record the stats.
type Metric struct {
	ID         string    `json:"id"`
	CreatedAt  time.Time `json:"createdAt"`
	ExpiresAt  time.Time `json:"expiresAt"`
	Title      string    `json:"title"`
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

// CreatedOn returns an ISO-8601 formatted string of the Metric's creation date.
func (m Metric) CreatedOn() string {
	if m.CreatedAt.IsZero() {
		return ""
	}
	return m.CreatedAt.Format("2006-01-02")
}

// CompressedJSON returns a compressed JSON representation of the Metric.
func (m Metric) CompressedJSON() []byte {
	j, err := v.ToCompressedJSON(m)
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
		problems = append(problems, "EntityID is not a valid TUID")
	}
	if m.Units == "" {
		problems = append(problems, "Units are missing")
	}
	return problems
}

// String method to include nice string representation of Metric for console output.
func (m Metric) String() string {
	s := "Metric " + m.ID + ": " + strconv.FormatFloat(m.Value, 'f', -1, 64) + " " + m.Units
	s += " at " + m.CreatedAt.Format("2006-01-02 15:04:05")
	if m.EntityType != "" && m.EntityID != "" {
		s += " for " + m.EntityType + "-" + m.EntityID
	} else if m.EntityType != "" {
		s += " for " + m.EntityType
	} else if m.EntityID != "" {
		s += " for " + m.EntityID
	}
	if len(m.Tags) > 0 {
		s += " tagged " + strings.Join(m.Tags, ", ")
	}
	s += " (" + m.Title + ")"
	return s
}

// MetricStat is a model entity to dynamically aggregate metrics.
type MetricStat struct {
	EntityID   string    `json:"entityId,omitempty"`
	EntityType string    `json:"entityType,omitempty"`
	Tag        string    `json:"tag,omitempty"`
	FromTime   time.Time `json:"fromTime"`
	ToTime     time.Time `json:"toTime"`
	Count      int64     `json:"count"`
	Sum        float64   `json:"sum"`
	Min        float64   `json:"min"`
	Max        float64   `json:"max"`
	Mean       float64   `json:"mean"`
	Median     float64   `json:"median"`
	StdDev     float64   `json:"stdDev"`
}

// NewMetricStat creates a new MetricStat from a slice of ID/value pairs.
func NewMetricStat(entityID, entityType, tag string, metrics []v.NumValue) MetricStat {
	var ms MetricStat
	ms.EntityID = entityID
	ms.EntityType = entityType
	ms.Tag = tag
	ms.Count = int64(len(metrics))
	if len(metrics) == 0 {
		return ms
	}

	// Use the embedded timestamps for from/to times
	fromID := tuid.TUID(metrics[0].Key)
	fromTime, err := fromID.Time()
	if err == nil {
		ms.FromTime = fromTime
	}
	toID := tuid.TUID(metrics[len(metrics)-1].Key)
	toTime, err := toID.Time()
	if err == nil {
		ms.ToTime = toTime
	}

	// Generate basic descriptive statistics
	var values []float64
	for i, m := range metrics {
		values = append(values, m.Value)
		if m.Value < ms.Min || i == 0 {
			ms.Min = m.Value
		}
		if m.Value > ms.Max || i == 0 {
			ms.Max = m.Value
		}
		ms.Sum += m.Value
	}
	ms.Mean = ms.Sum / float64(ms.Count)

	// Identify the median value
	slices.Sort(values)
	if ms.Count == 1 {
		ms.Median = values[0]
	} else if ms.Count%2 == 0 {
		ms.Median = (values[ms.Count/2-1] + values[ms.Count/2]) / 2
	} else {
		ms.Median = values[ms.Count/2]
	}

	// Calculate the standard deviation, which is the square root of the variance,
	// which is the average of the squared distances to the mean.
	var sumSquares float64
	for _, v := range values {
		sumSquares += (v - ms.Mean) * (v - ms.Mean)
	}
	variance := sumSquares / float64(len(values))
	ms.StdDev = math.Sqrt(variance)
	return ms
}
