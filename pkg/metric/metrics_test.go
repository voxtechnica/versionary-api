package metric

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/voxtechnica/tuid-go"
)

var (
	// Metric Service
	ctx     = context.Background()
	service = NewMockService("test")

	// Known timestamps
	t1 = time.Date(2022, time.April, 1, 12, 0, 0, 0, time.UTC)
	t2 = time.Date(2022, time.April, 1, 13, 0, 0, 0, time.UTC)
	t3 = time.Date(2022, time.April, 1, 14, 0, 0, 0, time.UTC)

	// Metric IDs
	id1 = tuid.NewIDWithTime(t1).String()
	id2 = tuid.NewIDWithTime(t2).String()
	id3 = tuid.NewIDWithTime(t3).String()

	// Known Metrics
	m30 = Metric{
		ID:        id3,
		CreatedAt: t3,
		Title:     "Page Load Duration",                // State intention
		Label:     "Homepage Load Time",                // Additional context for metric title
		Tags:      []string{"frontend", "performance"}, // Tags for categorization
		Value:     2.5,                                 // Duration value
		Units:     "seconds",                           // Units in seconds
	}
	m20 = Metric{
		ID:        id2,
		CreatedAt: t2,
		Title:     "API Response Time",
		Label:     "User Registration API Latency",
		Tags:      []string{"backend", "user", "latency"},
		Value:     320.7, // This indicates the API took 320.7 milliseconds to respond
		Units:     "ms",  // Milliseconds unit
	}
	m10 = Metric{
		ID:        id1,
		CreatedAt: t1,
		Title:     "Test CPU Usage",
		Label:     "Server-01 CPU",
		Tags:      []string{"infrastructure", "backend", "critical"},
		Value:     75.4, // This indicates 75.4% CPU usage
		Units:     "%",  // Percentage unit
	}
)

func testMain(m *testing.M) {
	// Check the table/row definition
	if !service.Table.IsValid() {
		log.Fatal("invalid table definition")
	}
	// Write known Metrics to the database
	for _, m := range []Metric{m10, m20, m30} {
		if _, err := service.Write(ctx, m); err != nil {
			log.Fatal(err)
		}
	}
	// Run the tests
	m.Run()
}

func TestCreateReadAndDeleteMetric(t *testing.T) {
	expect := assert.New(t)
	// Create a new Metric
	m, problems, err := service.Create(ctx, Metric{
		Title: "Loading Time",
		Value: 222.2,
	})
	expect.Empty(problems)
	if expect.NoError(err) {
		// Metric exists in the database
		mExist := service.Exists(ctx, m.ID)
		expect.True(mExist)

		// Read the Metric from the database
		mRead, err := service.Read(ctx, m.ID)
		if expect.NoError(err) {
			expect.Equal(m, mRead)
		}

		// Read the Metric from the database as JSON
		mJSON, err := service.ReadAsJSON(ctx, m.ID)
		if expect.NoError(err) {
			expect.NotEmpty(mJSON)
			expect.Contains(string(mJSON), m.ID)
		}

		// Delete the Metric from the database
		mDelete, err := service.Delete(ctx, m.ID)
		if expect.NoError(err) {
			expect.Equal(m, mDelete)
		}

		// Metric no longer exists in the database
		mExist = service.Exists(ctx, m.ID)
		expect.False(mExist)
	}
}
