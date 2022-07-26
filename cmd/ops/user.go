package main

import (
	"context"
	"encoding/json"
	"fmt"
	"versionary-api/pkg/event"
	"versionary-api/pkg/user"

	"github.com/voxtechnica/tuid-go"

	"github.com/spf13/cobra"
)

// initUserCmd initializes the user commands.
func initUserCmd(root *cobra.Command) {
	userCmd := &cobra.Command{
		Use:   "user",
		Short: "Manage users",
	}
	root.AddCommand(userCmd)

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new user",
		Long:  "Create a new user with the specified name, email address, and password.",
		RunE:  createUser,
	}
	createCmd.Flags().StringP("env", "e", "", "Operating environment: dev | test | staging | prod")
	createCmd.Flags().StringP("firstname", "f", "", "First Name")
	createCmd.Flags().StringP("lastname", "l", "", "Last Name")
	createCmd.Flags().StringP("email", "m", "", "Email Address")
	createCmd.Flags().StringP("password", "p", "", "Password")
	createCmd.Flags().StringP("org", "o", "", "Organization ID")
	createCmd.Flags().BoolP("admin", "a", false, "Admin Role?")
	_ = createCmd.MarkFlagRequired("env")
	_ = createCmd.MarkFlagRequired("email")
	_ = createCmd.MarkFlagRequired("password")
	userCmd.AddCommand(createCmd)

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List users",
		Long:  "List all users.",
		RunE:  listUsers,
	}
	listCmd.Flags().StringP("env", "e", "", "Operating environment: dev | test | staging | prod")
	listCmd.Flags().BoolP("reverse", "r", false, "Reverse chronological order?")
	listCmd.Flags().IntP("limit", "n", 100, "Limit: max the number of results")
	listCmd.Flags().StringP("offset", "i", "", "Offset: last ID received")
	_ = listCmd.MarkFlagRequired("env")
	userCmd.AddCommand(listCmd)

	readCmd := &cobra.Command{
		Use:   "read <userID or email>",
		Short: "Read specified user",
		Long:  "Read the specified user account, by email address or ID.",
		Args:  cobra.MinimumNArgs(1),
		RunE:  readUser,
	}
	readCmd.Flags().StringP("env", "e", "", "Operating environment: dev | test | staging | prod")
	_ = readCmd.MarkFlagRequired("env")
	userCmd.AddCommand(readCmd)
}

// createUser creates a new user.
func createUser(cmd *cobra.Command, args []string) error {
	// Initialize the application
	err := ops.Init(cmd.Flag("env").Value.String())
	if err != nil {
		return fmt.Errorf("error initializing application: %w", err)
	}
	ctx := context.Background()

	// Parse flags for user information
	u := user.User{
		FirstName: cmd.Flag("firstname").Value.String(),
		LastName:  cmd.Flag("lastname").Value.String(),
		Email:     cmd.Flag("email").Value.String(),
		Password:  cmd.Flag("password").Value.String(),
		OrgID:     cmd.Flag("org").Value.String(),
	}
	if admin, _ := cmd.Flags().GetBool("admin"); admin {
		u.Roles = append(u.Roles, "admin")
	}

	// Validate the Organization
	if u.OrgID != "" {
		org, err := ops.OrgService.Read(ctx, u.OrgID)
		if err != nil {
			return fmt.Errorf("error reading Organization %s: %w", u.OrgID, err)
		}
		if org.Name != "" {
			u.OrgName = org.Name
		}
	}

	// Create the User
	u, problems, err := ops.UserService.Create(ctx, u)
	if len(problems) > 0 && err != nil {
		return fmt.Errorf("unprocessable entity: %w", err)
	}
	if err != nil {
		e, _ := ops.EventService.Create(ctx, event.Event{
			EntityID:   u.ID,
			EntityType: u.Type(),
			LogLevel:   event.ERROR,
			Message:    fmt.Errorf("create user %s %s: %w", u.ID, u.Email, err).Error(),
			Err:        err,
		})
		return e
	}
	fmt.Printf("Created User %s %s\n", u.ID, u.Email)
	j, err := json.MarshalIndent(u, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshalling JSON User %s: %w", u.ID, err)
	}
	fmt.Println(string(j))
	return nil
}

// listUsers lists a batch of users.
func listUsers(cmd *cobra.Command, args []string) error {
	// Initialize the application
	err := ops.Init(cmd.Flag("env").Value.String())
	if err != nil {
		return fmt.Errorf("error initializing application: %w", err)
	}
	ctx := context.Background()

	// Read a batch of User account(s)
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
	users := ops.UserService.ReadUsers(ctx, reverse, limit, offset)
	j, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshalling JSON users: %w", err)
	}
	fmt.Println(string(j))
	return nil
}

// readUser reads the specified user(s).
func readUser(cmd *cobra.Command, args []string) error {
	// Initialize the application
	err := ops.Init(cmd.Flag("env").Value.String())
	if err != nil {
		return fmt.Errorf("error initializing application: %w", err)
	}
	ctx := context.Background()

	// Read the specified User account(s)
	for _, arg := range args {
		u, err := ops.UserService.Read(ctx, arg)
		if err != nil {
			return fmt.Errorf("error reading User %s: %w", arg, err)
		}
		j, err := json.MarshalIndent(u, "", "  ")
		if err != nil {
			return fmt.Errorf("error marshalling JSON User %s: %w", arg, err)
		}
		fmt.Println(string(j))
	}
	return nil
}
