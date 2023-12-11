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
	t4 = time.Date(2022, time.April, 1, 15, 0, 0, 0, time.UTC)

	// Metric IDs
	id1 = tuid.NewIDWithTime(t1).String()
	id2 = tuid.NewIDWithTime(t2).String()
	id3 = tuid.NewIDWithTime(t3).String()
	id4 = tuid.NewIDWithTime(t4).String()

	// Known entity IDs
	entity1 = tuid.NewIDWithTime(t1).String()
	entity2 = tuid.NewIDWithTime(t2).String()
	entity3 = tuid.NewIDWithTime(t3).String()

	// Known Metrics
	m40 = Metric{
		ID:         id4,
		CreatedAt:  t4,
		ExpiresAt:  t4.AddDate(1, 0, 0),
		Title:      "Test CPU Usage2",
		Label:      "Server-01 CPU2",
		EntityID:   entity1,
		EntityType: "Server",
		Tags:       []string{"infrastructure", "backend", "critical"},
		Value:      200.1, // This indicates 75.4% CPU usage
		Units:      "%",   // Percentage unit
	}
	m30 = Metric{
		ID:         id3,
		CreatedAt:  t3,
		ExpiresAt:  t3.AddDate(1, 0, 0),
		Title:      "Page Load Duration", // State intention
		Label:      "Homepage Load Time", // Additional context for metric title
		EntityID:   entity3,              // ID of the entity being measured
		EntityType: "Webpage",
		Tags:       []string{"frontend", "performance"}, // Tags for categorization
		Value:      2.5,                                 // Duration value
		Units:      "seconds",                           // Units in seconds
	}
	m20 = Metric{
		ID:         id2,
		CreatedAt:  t2,
		ExpiresAt:  t2.AddDate(1, 0, 0),
		Title:      "API Response Time",
		Label:      "User Registration API Latency",
		EntityID:   entity2,
		EntityType: "API",
		Tags:       []string{"backend", "user", "latency"},
		Value:      320.7, // This indicates the API took 320.7 milliseconds to respond
		Units:      "ms",  // Milliseconds unit
	}
	m10 = Metric{
		ID:         id1,
		CreatedAt:  t1,
		ExpiresAt:  t1.AddDate(1, 0, 0),
		Title:      "Test CPU Usage",
		Label:      "Server-01 CPU",
		EntityID:   entity1,
		EntityType: "Server",
		Tags:       []string{"infrastructure", "backend", "critical"},
		Value:      75.4, // This indicates 75.4% CPU usage
		Units:      "%",  // Percentage unit
	}

	knownIDs = []string{id1, id2, id3, id4}
)

func TestMain(m *testing.M) {
	// Check the table/row definition
	if !service.Table.IsValid() {
		log.Fatal("invalid table definition")
	}
	// Write known Metrics to the database
	for _, m := range []Metric{m40, m30, m20, m10} {
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
		Units: "ms",
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

func TestReadMetric(t *testing.T) {
	expect := assert.New(t)
	// Read the Metric from the database
	m, err := service.Read(ctx, id1)
	if expect.NoError(err) {
		expect.Equal(m10, m)
	}
}

func TestReadMetricAsJSON(t *testing.T) {
	expect := assert.New(t)
	// Read the Metric from the database as JSON
	mJSON, err := service.ReadAsJSON(ctx, id2)
	if expect.NoError(err) {
		expect.NotEmpty(mJSON)
		expect.Contains(string(mJSON), id2)
	}
}

func TestReadMetricIDs(t *testing.T) {
	expect := assert.New(t)
	// Read the Metric IDs from the database
	ids, err := service.ReadMetricIDs(ctx, false, 10, "-")
	if expect.NoError(err) {
		expect.GreaterOrEqual(len(ids), 3)
		expect.Subset(ids, knownIDs)
	}
}

func TestReadMetrics(t *testing.T) {
	expect := assert.New(t)
	// Read the Metrics from the database
	metrics := service.ReadMetrics(ctx, false, 10, "-")
	if expect.NotEmpty(metrics) {
		expect.GreaterOrEqual(len(metrics), 4)
	}
}

//------------------------------------------------------------------------------
// Metrics by Entity ID
//------------------------------------------------------------------------------

func TestReadAllEntityIDs(t *testing.T) {
	expect := assert.New(t)
	ids, err := service.ReadAllEntityIDs(ctx)
	if expect.NoError(err) {
		expect.Contains(ids, entity1)
		expect.Contains(ids, entity2)
		expect.Contains(ids, entity3)
	}
}

func TestReadMetricsByEntityID(t *testing.T) {
	expect := assert.New(t)
	metrics, err := service.ReadMetricsByEntityID(ctx, entity1, false, 10, "-")
	if expect.NoError(err) && expect.NotEmpty(metrics) {
		expect.Equal(2, len(metrics))
		expect.Equal(m10, metrics[0])
	}
}

func TestReadMetricsByEntityIDAsJSON(t *testing.T) {
	expect := assert.New(t)
	metricsJSON, err := service.ReadMetricsByEntityIDAsJSON(ctx, entity3, false, 10, "-")
	if expect.NoError(err) && expect.NotEmpty(metricsJSON) {
		expect.Contains(string(metricsJSON), entity3)
	}
}

func TestGenerateStatsForEntityID(t *testing.T) {
	expect := assert.New(t)
	testStat, err := service.GenerateStatsForEntityID(ctx, entity1, false, 10, "-")
	if expect.NoError(err) && expect.NotEmpty(testStat) {
		expect.Equal(entity1, testStat.EntityID)
	}
}

func TestGenerateStatsForEntityIDByDate(t *testing.T) {
	expect := assert.New(t)
	testStat, err := service.GenerateStatsForEntityIDByDate(ctx, entity1, "2021-09-25", "2023-10-09")
	startDate, _ := time.Parse("2006-01-02", "2021-09-25")
	endDate, _ := time.Parse("2006-01-02", "2023-10-09")
	if expect.NoError(err) && expect.NotEmpty(testStat) {
		expect.True(startDate.Before(testStat.FromTime))
		expect.True(endDate.After(testStat.ToTime))
	}
}

//------------------------------------------------------------------------------
// Metrics by Entity Type
//------------------------------------------------------------------------------

func TestReadAllEntityTypes(t *testing.T) {
	expect := assert.New(t)
	types, err := service.ReadAllEntityTypes(ctx)
	if expect.NoError(err) {
		expect.Contains(types, "Server")
		expect.Contains(types, "API")
		expect.Contains(types, "Webpage")
	}
}

func TestReadMetricsByEntityType(t *testing.T) {
	expect := assert.New(t)
	metric, err := service.ReadMetricsByEntityType(ctx, "Server", false, 10, "-")
	if expect.NoError(err) && expect.NotEmpty(metric) {
		expect.Equal(2, len(metric))
		expect.Equal(m10, metric[0])
	}
}

func TestReadMetricsByEntityTypeAsJSON(t *testing.T) {
	expect := assert.New(t)
	metricsJSON, err := service.ReadMetricsByEntityTypeAsJSON(ctx, "API", false, 10, "-")
	if expect.NoError(err) && expect.NotEmpty(metricsJSON) {
		expect.Contains(string(metricsJSON), "API")
	}
}

func TestGenerateStatsForEntityType(t *testing.T) {
	expect := assert.New(t)
	testStat, err := service.GenerateStatsForEntityTypeByDate(ctx, "API", "", "")
	if expect.NoError(err) && expect.NotEmpty(testStat) {
		expect.Contains(testStat.EntityType, "API")
	}
}

//------------------------------------------------------------------------------
// Metrics by Tag
//------------------------------------------------------------------------------

func TestReadAllTags(t *testing.T) {
	expect := assert.New(t)
	tags, err := service.ReadAllTags(ctx)
	if expect.NoError(err) {
		expect.Contains(tags, "backend")
		expect.Contains(tags, "frontend")
		expect.Contains(tags, "infrastructure")
		expect.Contains(tags, "performance")
		expect.Contains(tags, "user")
	}
}

func TestReadMetricsByTag(t *testing.T) {
	expect := assert.New(t)
	metrics, err := service.ReadMetricsByTag(ctx, "backend", false, 10, "-")
	if expect.NoError(err) && expect.NotEmpty(metrics) {
		expect.Equal(3, len(metrics))
		expect.Equal(m10, metrics[0])
	}
}

func TestReadMetricsByTagAsJSON(t *testing.T) {
	expect := assert.New(t)
	metricsJSON, err := service.ReadMetricsByTagAsJSON(ctx, "frontend", false, 10, "-")
	if expect.NoError(err) && expect.NotEmpty(metricsJSON) {
		expect.Contains(string(metricsJSON), "frontend")
	}
}
