package metric

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/voxtechnica/tuid-go"
	"github.com/voxtechnica/versionary"
)

var (
	// Metric Service
	ctx     = context.Background()
	service = NewMockService("test")

	// Known timestamps
	t1 = time.Date(2022, time.April, 1, 12, 0, 0, 0, time.Local)
	t2 = time.Date(2022, time.April, 1, 13, 0, 0, 0, time.Local)
	t3 = time.Date(2022, time.April, 1, 14, 0, 0, 0, time.Local)
	t4 = time.Date(2022, time.April, 2, 15, 0, 0, 0, time.Local)

	// Metric IDs
	id1      = tuid.NewIDWithTime(t1).String()
	id2      = tuid.NewIDWithTime(t2).String()
	id3      = tuid.NewIDWithTime(t3).String()
	id4      = tuid.NewIDWithTime(t4).String()
	knownIDs = []string{id1, id2, id3, id4}

	// Known entity IDs
	email1   = tuid.NewIDWithTime(t1).String()
	email2   = tuid.NewIDWithTime(t2).String()
	content1 = tuid.NewIDWithTime(t3).String()

	// Known Metrics
	m1 = Metric{
		ID:         id1,
		CreatedAt:  t1,
		ExpiresAt:  t1.AddDate(1, 0, 0),
		Title:      "Send Email Latency",
		EntityID:   email1,
		EntityType: "Email",
		Tags:       []string{"aws", "ses", "latency"},
		Value:      75.4,
		Units:      "ms",
	}
	m2 = Metric{
		ID:         id2,
		CreatedAt:  t2,
		ExpiresAt:  t2.AddDate(1, 0, 0),
		Title:      "Send Email Latency",
		EntityID:   email2,
		EntityType: "Email",
		Tags:       []string{"aws", "ses", "latency"},
		Value:      48.6,
		Units:      "ms",
	}
	m3 = Metric{
		ID:         id3,
		CreatedAt:  t3,
		ExpiresAt:  t3.AddDate(1, 0, 0),
		Title:      "Read Content Latency",
		EntityID:   content1,
		EntityType: "Content",
		Tags:       []string{"api", "latency"},
		Value:      102.5,
		Units:      "ms",
	}
	m4 = Metric{
		ID:         id4,
		CreatedAt:  t4,
		ExpiresAt:  t4.AddDate(1, 0, 0),
		Title:      "Read Content Latency",
		EntityID:   content1,
		EntityType: "Content",
		Tags:       []string{"api", "latency"},
		Value:      39.6,
		Units:      "ms",
	}
	knownMetrics = []Metric{m1, m2, m3, m4}
)

func TestMain(m *testing.M) {
	// Check the table/row definition
	if !service.Table.IsValid() {
		log.Fatal("invalid table definition")
	}
	// Write known Metrics to the database
	for _, m := range knownMetrics {
		if _, err := service.Write(ctx, m); err != nil {
			log.Fatal(err)
		}
	}
	// Run the tests
	m.Run()
}

func TestCreateReadAndDeleteMetric(t *testing.T) {
	expect := assert.New(t)
	// Create a new Metric with a current timestamp.
	// This should be the last Metric in chronological order.
	m, problems, err := service.Create(ctx, Metric{
		Title:      "User Login Latency",
		EntityID:   tuid.NewID().String(),
		EntityType: "User",
		Value:      222.2,
		Units:      "ms",
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

		// Read the Metric IDs from the database
		ids, err := service.ReadMetricIDs(ctx, true, 1, "")
		if expect.NoError(err) {
			expect.Equal(1, len(ids))
			expect.Equal(m.ID, ids[0])
		}

		// Read Metric ID/label pairs from the database
		labels, err := service.ReadMetricLabels(ctx, true, 1, "")
		if expect.NoError(err) {
			expect.Equal(1, len(labels))
			expect.Equal(m.ID, labels[0].Key)
			expect.Equal(m.String(), labels[0].Value)
		}

		// Read Metrics from the database
		metrics := service.ReadMetrics(ctx, true, 1, "")
		if expect.NotEmpty(metrics) {
			expect.Equal(1, len(metrics))
			expect.Equal(m, metrics[0])
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
		expect.Equal(m1, m)
	}
	// Read a Metric that does not exist
	_, err = service.Read(ctx, tuid.NewID().String())
	if expect.Error(err) {
		expect.Equal(versionary.ErrNotFound, err)
	}
}

func TestMetricExists(t *testing.T) {
	expect := assert.New(t)
	// Metric exists in the database
	mExist := service.Exists(ctx, id1)
	expect.True(mExist)
	// Metric does not exist in the database
	mExist = service.Exists(ctx, tuid.NewID().String())
	expect.False(mExist)
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
	ids, err := service.ReadMetricIDs(ctx, false, 10, "")
	if expect.NoError(err) {
		expect.GreaterOrEqual(len(ids), 4)
		expect.Subset(ids, knownIDs)
	}
}

func TestReadMetrics(t *testing.T) {
	expect := assert.New(t)
	// Read the Metrics from the database
	metrics := service.ReadMetrics(ctx, false, 10, "")
	if expect.NotEmpty(metrics) {
		expect.GreaterOrEqual(len(metrics), 4)
		expect.Subset(metrics, knownMetrics)
	}
}

//------------------------------------------------------------------------------
// Metrics by Entity ID
//------------------------------------------------------------------------------

func TestReadAllEntityIDs(t *testing.T) {
	expect := assert.New(t)
	ids, err := service.ReadAllEntityIDs(ctx)
	if expect.NoError(err) {
		expect.Contains(ids, email1)
		expect.Contains(ids, email2)
		expect.Contains(ids, content1)
	}
}

func TestReadMetricsByEntityID(t *testing.T) {
	expect := assert.New(t)
	metrics, err := service.ReadMetricsByEntityID(ctx, content1, false, 10, "")
	if expect.NoError(err) && expect.NotEmpty(metrics) {
		expect.Equal(2, len(metrics))
		expect.Contains(metrics, m3)
		expect.Contains(metrics, m4)
	}
}

func TestReadMetricsByEntityIDAsJSON(t *testing.T) {
	expect := assert.New(t)
	metricsJSON, err := service.ReadMetricsByEntityIDAsJSON(ctx, content1, false, 10, "")
	if expect.NoError(err) && expect.NotEmpty(metricsJSON) {
		expect.Contains(string(metricsJSON), content1)
		expect.Contains(string(metricsJSON), m3.ID)
		expect.Contains(string(metricsJSON), m4.ID)
	}
}

func TestReadAllMetricsByEntityID(t *testing.T) {
	expect := assert.New(t)
	metrics, err := service.ReadAllMetricsByEntityID(ctx, content1)
	if expect.NoError(err) && expect.NotEmpty(metrics) {
		expect.Equal(2, len(metrics))
		expect.Contains(metrics, m3)
		expect.Contains(metrics, m4)
	}
}

func TestReadAllMetricsByEntityIDAsJSON(t *testing.T) {
	expect := assert.New(t)
	metricsJSON, err := service.ReadAllMetricsByEntityIDAsJSON(ctx, content1)
	if expect.NoError(err) && expect.NotEmpty(metricsJSON) {
		expect.Contains(string(metricsJSON), content1)
		expect.Contains(string(metricsJSON), m3.ID)
		expect.Contains(string(metricsJSON), m4.ID)
	}
}

func TestReadMetricRangeByEntityID(t *testing.T) {
	expect := assert.New(t)
	metrics, err := service.ReadMetricRangeByEntityID(ctx, content1, "2022-04-01", "2022-04-02", false)
	if expect.NoError(err) && expect.NotEmpty(metrics) {
		expect.Equal(1, len(metrics))
		expect.Contains(metrics, m3)
	}
}

func TestReadMetricStatByEntityID(t *testing.T) {
	expect := assert.New(t)
	stat, err := service.ReadMetricStatByEntityID(ctx, content1)
	if expect.NoError(err) && expect.NotEmpty(stat) {
		expect.Equal(content1, stat.EntityID)
		expect.Equal(int64(2), stat.Count)
		expect.Equal(m3.Value+m4.Value, stat.Sum)
	}
}

func TestReadMetricStatRangeByEntityID(t *testing.T) {
	expect := assert.New(t)
	stat, err := service.ReadMetricStatRangeByEntityID(ctx, content1, "2022-04-01", "2022-04-03")
	if expect.NoError(err) && expect.NotEmpty(stat) {
		expect.Equal(content1, stat.EntityID)
		expect.Equal(int64(2), stat.Count)
		expect.Equal(m3.CreatedAt, stat.FromTime)
		expect.Equal(m4.CreatedAt, stat.ToTime)
	}
}

//------------------------------------------------------------------------------
// Metrics by Entity Type
//------------------------------------------------------------------------------

func TestReadAllEntityTypes(t *testing.T) {
	expect := assert.New(t)
	types, err := service.ReadAllEntityTypes(ctx)
	if expect.NoError(err) && expect.NotEmpty(types) {
		expect.GreaterOrEqual(len(types), 2)
		expect.Contains(types, "Email")
		expect.Contains(types, "Content")
	}
}

func TestReadMetricsByEntityType(t *testing.T) {
	expect := assert.New(t)
	metric, err := service.ReadMetricsByEntityType(ctx, "Email", false, 10, "")
	if expect.NoError(err) && expect.NotEmpty(metric) {
		expect.Equal(2, len(metric))
		expect.Equal(m1, metric[0])
		expect.Equal(m2, metric[1])
	}
}

func TestReadMetricsByEntityTypeAsJSON(t *testing.T) {
	expect := assert.New(t)
	metricsJSON, err := service.ReadMetricsByEntityTypeAsJSON(ctx, "Email", false, 10, "")
	if expect.NoError(err) && expect.NotEmpty(metricsJSON) {
		expect.Contains(string(metricsJSON), "Email")
		expect.Contains(string(metricsJSON), m1.ID)
		expect.Contains(string(metricsJSON), m2.ID)
	}
}

func TestReadAllMetricsByEntityType(t *testing.T) {
	expect := assert.New(t)
	metrics, err := service.ReadAllMetricsByEntityType(ctx, "Email")
	if expect.NoError(err) && expect.NotEmpty(metrics) {
		expect.Equal(2, len(metrics))
		expect.Equal(m1, metrics[0])
		expect.Equal(m2, metrics[1])
	}
}

func TestReadAllMetricsByEntityTypeAsJSON(t *testing.T) {
	expect := assert.New(t)
	metricsJSON, err := service.ReadAllMetricsByEntityTypeAsJSON(ctx, "Email")
	if expect.NoError(err) && expect.NotEmpty(metricsJSON) {
		expect.Contains(string(metricsJSON), "Email")
		expect.Contains(string(metricsJSON), m1.ID)
		expect.Contains(string(metricsJSON), m2.ID)
	}
}

func TestReadMetricRangeByEntityType(t *testing.T) {
	expect := assert.New(t)
	metrics, err := service.ReadMetricRangeByEntityType(ctx, "Email", "2022-04-01", "2022-04-02", false)
	if expect.NoError(err) && expect.NotEmpty(metrics) {
		expect.Equal(2, len(metrics))
		expect.Equal(m1, metrics[0])
		expect.Equal(m2, metrics[1])
	}
}

func TestReadMetricStatByEntityType(t *testing.T) {
	expect := assert.New(t)
	stat, err := service.ReadMetricStatByEntityType(ctx, "Email")
	if expect.NoError(err) && expect.NotEmpty(stat) {
		expect.Equal(stat.EntityType, "Email")
		expect.Equal(int64(2), stat.Count)
		expect.Equal(m1.Value+m2.Value, stat.Sum)
		expect.Equal(m1.CreatedAt, stat.FromTime)
		expect.Equal(m2.CreatedAt, stat.ToTime)
	}
}

func TestReadMetricStatRangeByEntityType(t *testing.T) {
	expect := assert.New(t)
	stat, err := service.ReadMetricStatRangeByEntityType(ctx, "Content", "2022-04-01", "2022-04-02")
	if expect.NoError(err) && expect.NotEmpty(stat) {
		expect.Equal(stat.EntityType, "Content")
		expect.Equal(int64(1), stat.Count)
		expect.Equal(m3.Value, stat.Min)
		expect.Equal(m3.Value, stat.Max)
		expect.Equal(m3.Value, stat.Median)
		expect.Equal(m3.Value, stat.Mean)
		expect.Equal(m3.Value, stat.Sum)
		expect.Equal(float64(0), stat.StdDev)
		expect.Equal(m3.CreatedAt, stat.FromTime)
		expect.Equal(m3.CreatedAt, stat.ToTime)
	}
}

//------------------------------------------------------------------------------
// Metrics by Tag
//------------------------------------------------------------------------------

func TestReadAllTags(t *testing.T) {
	expect := assert.New(t)
	tags, err := service.ReadAllTags(ctx)
	if expect.NoError(err) && expect.NotEmpty(tags) {
		expect.Equal(4, len(tags))
		expect.Contains(tags, "aws")
		expect.Contains(tags, "ses")
		expect.Contains(tags, "latency")
		expect.Contains(tags, "api")
	}
}

func TestReadMetricsByTag(t *testing.T) {
	expect := assert.New(t)
	metrics, err := service.ReadMetricsByTag(ctx, "latency", false, 10, "")
	if expect.NoError(err) && expect.NotEmpty(metrics) {
		expect.Equal(4, len(metrics))
		expect.Equal(metrics, knownMetrics)
	}
}

func TestReadMetricsByTagAsJSON(t *testing.T) {
	expect := assert.New(t)
	metricsJSON, err := service.ReadMetricsByTagAsJSON(ctx, "latency", false, 10, "")
	if expect.NoError(err) && expect.NotEmpty(metricsJSON) {
		expect.Contains(string(metricsJSON), "latency")
		expect.Contains(string(metricsJSON), m1.ID)
		expect.Contains(string(metricsJSON), m2.ID)
		expect.Contains(string(metricsJSON), m3.ID)
		expect.Contains(string(metricsJSON), m4.ID)
	}
}

func TestReadAllMetricsByTag(t *testing.T) {
	expect := assert.New(t)
	metrics, err := service.ReadAllMetricsByTag(ctx, "aws")
	if expect.NoError(err) && expect.NotEmpty(metrics) {
		expect.Equal(2, len(metrics))
		expect.Equal(metrics, knownMetrics[:2])
	}
}

func TestReadAllMetricsByTagAsJSON(t *testing.T) {
	expect := assert.New(t)
	metricsJSON, err := service.ReadAllMetricsByTagAsJSON(ctx, "aws")
	if expect.NoError(err) && expect.NotEmpty(metricsJSON) {
		expect.Contains(string(metricsJSON), "aws")
		expect.Contains(string(metricsJSON), m1.ID)
		expect.Contains(string(metricsJSON), m2.ID)
	}
}

func TestReadMetricRangeByTag(t *testing.T) {
	expect := assert.New(t)
	metrics, err := service.ReadMetricRangeByTag(ctx, "latency", "2022-04-01", "2022-04-02", false)
	if expect.NoError(err) && expect.NotEmpty(metrics) {
		expect.Equal(3, len(metrics))
		expect.Equal(metrics, knownMetrics[:3])
	}
}

func TestReadMetricStatByTag(t *testing.T) {
	expect := assert.New(t)
	stat, err := service.ReadMetricStatByTag(ctx, "latency")
	if expect.NoError(err) && expect.NotEmpty(stat) {
		expect.Equal(int64(4), stat.Count)
		expect.Equal(m1.Value+m2.Value+m3.Value+m4.Value, stat.Sum)
		expect.Equal(m1.CreatedAt, stat.FromTime)
		expect.Equal(m4.CreatedAt, stat.ToTime)
	}
}

func TestReadMetricStatRangeByTag(t *testing.T) {
	expect := assert.New(t)
	stat, err := service.ReadMetricStatRangeByTag(ctx, "latency", "2022-04-01", "2022-04-02")
	if expect.NoError(err) && expect.NotEmpty(stat) {
		expect.Equal(int64(3), stat.Count)
		expect.Equal(m1.Value+m2.Value+m3.Value, stat.Sum)
		expect.Equal(m1.CreatedAt, stat.FromTime)
		expect.Equal(m3.CreatedAt, stat.ToTime)
	}
}
