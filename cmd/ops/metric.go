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
	"github.com/voxtechnica/tuid-go"
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
	createCmd.Flags().StringP("label", "l", "", "Metric label")
	createCmd.Flags().StringP("entity", "y", "", "Entity ID")
	createCmd.Flags().StringP("type", "p", "", "Entity type")
	createCmd.Flags().StringP("tags", "g", "", "Tags")
	createCmd.Flags().Float64P("value", "v", 0.0, "Metric value")
	createCmd.Flags().StringP("units", "u", "", "Metric units")
	_ = createCmd.MarkFlagRequired("env")
	_ = createCmd.MarkFlagRequired("title")
	_ = createCmd.MarkFlagRequired("value")
	_ = createCmd.MarkFlagRequired("units")
	metricCmd.AddCommand(createCmd)

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List metrics",
		Long:  "List all metrics.",
		RunE:  listMetrics,
	}
	listCmd.Flags().StringP("env", "e", "", "Operating environment: dev | test | staging | prod")
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
}

// createMetric creates a new metric.
func createMetric(cmd *cobra.Command, args []string) error {
	// Initialize the application
	err := ops.Init(cmd.Flag("env").Value.String())
	if err != nil {
		return fmt.Errorf("error initializing application: %s", err)
	}
	ctx := context.Background()

	// Parse flags for metric title, value and units
	valueStr := cmd.Flag("value").Value.String()
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return fmt.Errorf("error parsing metric value %s: %w", valueStr, err)
	}

	m := metric.Metric{
		Title: cmd.Flag("title").Value.String(),
		Value: value,
		Units: cmd.Flag("units").Value.String(),
	}

	// Parse flags for metric label, entity ID and type
	m.Label = cmd.Flag("label").Value.String()
	m.EntityID = cmd.Flag("entity").Value.String()
	m.EntityType = cmd.Flag("type").Value.String()

	// Parse flags for metric tags
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

	// Read a batch of Metric(s)
	reverse, _ := cmd.Flags().GetBool("reverse")
	limit, _ := cmd.Flags().GetInt("limit")
	offset, _ := cmd.Flags().GetString("offset")
	if offset == "" {
		if reverse {
			offset = tuid.MaxID
		} else {
			offset = tuid.MinID
		}
	}
	metrics := ops.MetricService.ReadMetrics(ctx, reverse, limit, offset)
	j, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling JSON Metrics: %w", err)
	}
	fmt.Println(string(j))
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
