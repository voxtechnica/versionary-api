package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"versionary-api/pkg/event"
	"versionary-api/pkg/metric"

	"github.com/spf13/cobra"
)

// initMetricCmd initializes the metric commands.
func initMetricCmd(root *cobra.Command) {
	metricCmd := &cobra.Command{
		Use:   "metric",
		Short: "Manage metrics",
	}
	root.AddCommand(metricCmd)

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new metric",
		Long:  "Create a new metric with the specified title, value and units.",
		RunE:  createMetric,
	}
	createCmd.Flags().StringP("env", "e", "", "Operating environment: dev | test | staging | prod")
	createCmd.Flags().StringP("title", "t", "", "Metric title")
	createCmd.Flags().StringP("entity", "y", "", "Entity ID")
	createCmd.Flags().StringP("type", "p", "", "Entity type")
	createCmd.Flags().StringP("tags", "g", "", "Tags")
	createCmd.Flags().Float64P("value", "V", 0.0, "Metric value")
	createCmd.Flags().StringP("units", "u", "", "Metric units")
	_ = createCmd.MarkFlagRequired("env")
	_ = createCmd.MarkFlagRequired("title")
	_ = createCmd.MarkFlagRequired("value")
	_ = createCmd.MarkFlagRequired("units")
	metricCmd.AddCommand(createCmd)

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List metrics",
		Long:  "List paginated metrics, optionally by entity ID, type, or tag.",
		RunE:  listMetrics,
	}
	listCmd.Flags().StringP("env", "e", "", "Operating environment: dev | test | staging | prod")
	listCmd.Flags().BoolP("json", "j", false, "Verbose: full JSON output?")
	listCmd.Flags().StringP("entity", "y", "", "Entity ID")
	listCmd.Flags().StringP("type", "p", "", "Entity type")
	listCmd.Flags().StringP("tag", "g", "", "Tag")
	listCmd.Flags().BoolP("reverse", "r", false, "Reverse chronological order?")
	listCmd.Flags().IntP("limit", "n", 100, "Limit: max the number of results")
	listCmd.Flags().StringP("offset", "i", "", "Offset: last ID received")
	_ = listCmd.MarkFlagRequired("env")
	metricCmd.AddCommand(listCmd)

	readCmd := &cobra.Command{
		Use:   "read <metricID>",
		Short: "Read specified metric",
		Long:  "Read the specified metric, by ID.",
		Args:  cobra.MinimumNArgs(1),
		RunE:  readMetric,
	}
	readCmd.Flags().StringP("env", "e", "", "Operating environment: dev | test | staging | prod")
	_ = readCmd.MarkFlagRequired("env")
	metricCmd.AddCommand(readCmd)

	statsCmd := &cobra.Command{
		Use:   "stats",
		Short: "Read metric stats",
		Long:  "Read the specified metric stats, by entity ID, type, or tag.",
		RunE:  readMetricStats,
	}
	statsCmd.Flags().StringP("env", "e", "", "Operating environment: dev | test | staging | prod")
	statsCmd.Flags().StringP("entity", "y", "", "Entity ID")
	statsCmd.Flags().StringP("type", "p", "", "Entity type")
	statsCmd.Flags().StringP("tag", "g", "", "Tag")
	_ = statsCmd.MarkFlagRequired("env")
	metricCmd.AddCommand(statsCmd)
}

// createMetric creates a new metric.
func createMetric(cmd *cobra.Command, args []string) error {
	// Initialize the application
	err := ops.Init(cmd.Flag("env").Value.String())
	if err != nil {
		return fmt.Errorf("error initializing application: %s", err)
	}
	ctx := context.Background()

	// Parse flags for metric fields
	valueStr := cmd.Flag("value").Value.String()
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return fmt.Errorf("error parsing metric value %s: %w", valueStr, err)
	}
	m := metric.Metric{
		Title:      cmd.Flag("title").Value.String(),
		EntityID:   cmd.Flag("entity").Value.String(),
		EntityType: cmd.Flag("type").Value.String(),
		Value:      value,
		Units:      cmd.Flag("units").Value.String(),
	}
	tags := cmd.Flag("tags").Value.String()
	if tags != "" {
		m.Tags = strings.Split(tags, ",")
	}

	// Create the Metric
	m, problems, err := ops.MetricService.Create(ctx, m)
	if len(problems) > 0 && err != nil {
		return fmt.Errorf("unprocessable entity: %w", err)
	}
	if err != nil {
		e, _, _ := ops.EventService.Create(ctx, event.Event{
			EntityID:   m.ID,
			EntityType: m.Type(),
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("create metric %s %s: %w", m.ID, m.Title, err).Error(),
			Err:        err,
		})
		return e
	}
	fmt.Printf("Created metric %s %s\n", m.ID, m.Title)
	j, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling JSON Metric %s: %w", m.ID, err)
	}
	fmt.Println(string(j))
	return nil
}

// listMetrics lists a batch of metrics.
func listMetrics(cmd *cobra.Command, args []string) error {
	// Initialize the application
	err := ops.Init(cmd.Flag("env").Value.String())
	if err != nil {
		return fmt.Errorf("error initializing application: %s", err)
	}
	ctx := context.Background()

	// Parse parameters
	verbose, _ := cmd.Flags().GetBool("json")
	entityID, _ := cmd.Flags().GetString("entity")
	entityType, _ := cmd.Flags().GetString("type")
	tag, _ := cmd.Flags().GetString("tag")
	reverse, _ := cmd.Flags().GetBool("reverse")
	limit, _ := cmd.Flags().GetInt("limit")
	offset, _ := cmd.Flags().GetString("offset")

	// Read a batch of Metric(s)
	var metrics []metric.Metric
	if entityID != "" {
		metrics, err = ops.MetricService.ReadMetricsByEntityID(ctx, entityID, reverse, limit, offset)
		if err != nil {
			return fmt.Errorf("error reading Metrics by EntityID %s: %w", entityID, err)
		}
	} else if entityType != "" {
		metrics, err = ops.MetricService.ReadMetricsByEntityType(ctx, entityType, reverse, limit, offset)
		if err != nil {
			return fmt.Errorf("error reading Metrics by EntityType %s: %w", entityType, err)
		}
	} else if tag != "" {
		metrics, err = ops.MetricService.ReadMetricsByTag(ctx, tag, reverse, limit, offset)
		if err != nil {
			return fmt.Errorf("error reading Metrics by Tag %s: %w", tag, err)
		}
	} else {
		metrics = ops.MetricService.ReadMetrics(ctx, reverse, limit, offset)
	}

	// Output the results
	if verbose {
		j, err := json.MarshalIndent(metrics, "", "  ")
		if err != nil {
			return fmt.Errorf("error marshaling JSON Metrics: %w", err)
		}
		fmt.Println(string(j))
	} else {
		for _, m := range metrics {
			fmt.Println(m.String())
		}
	}
	return nil
}

// readMetric reads the specified metric(s).
func readMetric(cmd *cobra.Command, args []string) error {
	// Initialize the application
	err := ops.Init(cmd.Flag("env").Value.String())
	if err != nil {
		return fmt.Errorf("error initializing application: %s", err)
	}
	ctx := context.Background()

	// Read the specified Metric(s)
	for _, arg := range args {
		m, err := ops.MetricService.Read(ctx, arg)
		if err != nil {
			return fmt.Errorf("error reading Metric %s: %w", arg, err)
		}
		j, err := json.MarshalIndent(m, "", "  ")
		if err != nil {
			return fmt.Errorf("error marshaling JSON Metric %s: %w", arg, err)
		}
		fmt.Println(string(j))
	}
	return nil
}

// readMetricStats reads the specified metric stats.
func readMetricStats(cmd *cobra.Command, args []string) error {
	// Initialize the application
	err := ops.Init(cmd.Flag("env").Value.String())
	if err != nil {
		return fmt.Errorf("error initializing application: %s", err)
	}
	ctx := context.Background()

	// Parse parameters
	entityID, _ := cmd.Flags().GetString("entity")
	entityType, _ := cmd.Flags().GetString("type")
	tag, _ := cmd.Flags().GetString("tag")

	// Read the specified MetricStats
	var stats metric.MetricStat
	if entityID != "" {
		stats, err = ops.MetricService.ReadMetricStatByEntityID(ctx, entityID)
		if err != nil {
			return fmt.Errorf("error reading MetricStats by EntityID %s: %w", entityID, err)
		}
	} else if entityType != "" {
		stats, err = ops.MetricService.ReadMetricStatByEntityType(ctx, entityType)
		if err != nil {
			return fmt.Errorf("error reading MetricStats by EntityType %s: %w", entityType, err)
		}
	} else if tag != "" {
		stats, err = ops.MetricService.ReadMetricStatByTag(ctx, tag)
		if err != nil {
			return fmt.Errorf("error reading MetricStats by Tag %s: %w", tag, err)
		}
	} else {
		return fmt.Errorf("no entity, type, or tag specified")
	}

	// Output the results
	j, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling JSON MetricStat: %w", err)
	}
	fmt.Println(string(j))
	return nil
}
