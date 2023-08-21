package main

import (
	"context"
	"encoding/json"
	"fmt"
	"versionary-api/pkg/event"
	"versionary-api/pkg/org"

	"github.com/spf13/cobra"
	"github.com/voxtechnica/tuid-go"
)

// initOrgCmd initializes the organization commands.
func initOrgCmd(root *cobra.Command) {
	orgCmd := &cobra.Command{
		Use:   "org",
		Short: "Manage organizations",
	}
	root.AddCommand(orgCmd)

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new organization",
		Long:  "Create a new organization with the specified name and status.",
		RunE:  createOrg,
	}
	createCmd.Flags().StringP("env", "e", "", "Operating environment: dev | test | staging | prod")
	createCmd.Flags().StringP("name", "n", "", "Organization name")
	createCmd.Flags().StringP("status", "s", "", "Organization status: PENDING | ENABLED | DISABLED")
	_ = createCmd.MarkFlagRequired("env")
	_ = createCmd.MarkFlagRequired("name")
	orgCmd.AddCommand(createCmd)

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List organizations",
		Long:  "List all organizations.",
		RunE:  listOrgs,
	}
	listCmd.Flags().StringP("env", "e", "", "Operating environment: dev | test | staging | prod")
	listCmd.Flags().BoolP("reverse", "r", false, "Reverse chronological order?")
	listCmd.Flags().IntP("limit", "n", 100, "Limit: max the number of results")
	listCmd.Flags().StringP("offset", "i", "", "Offset: last ID received")
	_ = listCmd.MarkFlagRequired("env")
	orgCmd.AddCommand(listCmd)

	readCmd := &cobra.Command{
		Use:   "read <orgID or name>",
		Short: "Read specified organization",
		Long:  "Read the specified organization, by name or ID.",
		Args:  cobra.MinimumNArgs(1),
		RunE:  readOrg,
	}
	readCmd.Flags().StringP("env", "e", "", "Operating environment: dev | test | staging | prod")
	orgCmd.AddCommand(readCmd)

	deleteCmd := &cobra.Command{
		Use:   "delete <orgID>",
		Short: "Delete specified organization",
		Long:  "Delete the specified organization, by ID.",
		Args:  cobra.ExactArgs(1),
		RunE:  deleteOrg,
	}
	deleteCmd.Flags().StringP("env", "e", "", "Operating environment: dev | test | staging | prod")
	_ = deleteCmd.MarkFlagRequired("env")
	orgCmd.AddCommand(deleteCmd)
}

// createOrg creates a new organization.
func createOrg(cmd *cobra.Command, args []string) error {
	// Initialize the application
	err := ops.Init(cmd.Flag("env").Value.String())
	if err != nil {
		return fmt.Errorf("error initializing application: %s", err)
	}
	ctx := context.Background()

	status := org.ENABLED
	strStatus := cmd.Flag("status").Value.String()
	// parse the string
	if strStatus != "" {
		status, err = org.ParseStatus(strStatus)
		if err != nil {
			return fmt.Errorf("create organization: %w", err)
		}
	}

	// Parse flags for organization name and status
	o := org.Organization{
		Name:   cmd.Flag("name").Value.String(),
		Status: status,
	}

	// Create the Organization
	o, problems, err := ops.OrgService.Create(ctx, o)
	if len(problems) > 0 && err != nil {
		return fmt.Errorf("unprocessable entity: %w", err)
	}
	if err != nil {
		e, _, _ := ops.EventService.Create(ctx, event.Event{
			EntityID:   o.ID,
			EntityType: o.Type(),
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("create organization %s %s: %w", o.ID, o.Name, err).Error(),
			Err:        err,
		})
		return e
	}
	fmt.Printf("Created organization %s %s\n", o.ID, o.Name)
	j, err := json.MarshalIndent(o, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling JSON Org %s: %w", o.ID, err)
	}
	fmt.Println(string(j))
	return nil
}

// listOrgs lists a batch of organizations.
func listOrgs(cmd *cobra.Command, args []string) error {
	// Initialize the application
	err := ops.Init(cmd.Flag("env").Value.String())
	if err != nil {
		return fmt.Errorf("error initializing application: %s", err)
	}
	ctx := context.Background()

	// Read a batch of Organizations
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
	orgs := ops.OrgService.ReadOrganizations(ctx, reverse, limit, offset)
	j, err := json.MarshalIndent(orgs, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling JSON Orgs: %w", err)
	}
	fmt.Println(string(j))
	return nil
}

// readOrg reads the specified organization(s).
func readOrg(cmd *cobra.Command, args []string) error {
	// Initialize the application
	err := ops.Init(cmd.Flag("env").Value.String())
	if err != nil {
		return fmt.Errorf("error initializing application: %s", err)
	}
	ctx := context.Background()

	// Read the specified Organization(s)
	for _, arg := range args {
		var org org.Organization
		if tuid.IsValid(tuid.TUID(arg)) {
			org, err = ops.OrgService.Read(ctx, arg)
		} else {
			org, err = ops.OrgService.ReadOrganizationByName(ctx, arg)
		}
		if err != nil {
			return fmt.Errorf("error reading Organization %s: %w", arg, err)
		}
		j, err := json.MarshalIndent(org, "", "  ")
		if err != nil {
			return fmt.Errorf("error marshaling JSON Org %s: %w", arg, err)
		}
		fmt.Println(string(j))
	}
	return nil
}

// deleteOrg deletes the specified organization
func deleteOrg(cmd *cobra.Command, args []string) error {
	env := cmd.Flag("env").Value.String()

	// Initialize the application
	err := ops.Init(env)
	if err != nil {
		return fmt.Errorf("error initializing application: %s", err)
	}
	ctx := context.Background()

	// Delete the specified Organization
	for _, orgID := range args {
		org, err := ops.OrgService.Delete(ctx, orgID)
		if err != nil {
			return fmt.Errorf("error deleting Organization %s: %w", orgID, err)
		}
		orgJSON, err := json.MarshalIndent(org, "", "  ")
		if err != nil {
			return fmt.Errorf("error marshaling JSON Org %s: %w", orgID, err)
		}
		fmt.Println(string(orgJSON))
	}
	return nil
}
