package main

import (
	"context"
	"encoding/json"
	"fmt"
	"versionary-api/pkg/app"
	"versionary-api/pkg/user"

	"github.com/spf13/cobra"
)

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
	createCmd.MarkFlagRequired("env")
	createCmd.MarkFlagRequired("email")
	createCmd.MarkFlagRequired("password")
	userCmd.AddCommand(createCmd)

	readCmd := &cobra.Command{
		Use:   "read <userID or email>",
		Short: "Read specified user",
		Long:  "Read the specified user account, by email address or ID.",
		Args:  cobra.MinimumNArgs(1),
		RunE:  readUser,
	}
	readCmd.Flags().StringP("env", "e", "", "Operating environment: dev | test | staging | prod")
	readCmd.MarkFlagRequired("env")
	userCmd.AddCommand(readCmd)
}

func createUser(cmd *cobra.Command, args []string) error {
	// Initialize the application
	var a app.Application
	err := a.Init(cmd.Flag("env").Value.String())
	if err != nil {
		return fmt.Errorf("error initializing application: %w", err)
	}
	ctx := context.Background()

	// Parse flags for user information
	firstname, err := cmd.Flags().GetString("firstname")
	if err != nil {
		return err
	}
	lastname, err := cmd.Flags().GetString("lastname")
	if err != nil {
		return err
	}
	email, err := cmd.Flags().GetString("email")
	if err != nil {
		return err
	}
	password, err := cmd.Flags().GetString("password")
	if err != nil {
		return err
	}
	orgID, err := cmd.Flags().GetString("org")
	if err != nil {
		return err
	}
	admin, err := cmd.Flags().GetBool("admin")
	if err != nil {
		return err
	}
	u := user.User{
		FirstName: firstname,
		LastName:  lastname,
		Email:     email,
		Password:  password,
		OrgID:     orgID,
	}
	if admin {
		u.Roles = append(u.Roles, "admin")
	}

	// Validate the Organization
	if orgID != "" {
		org, err := a.OrgService.Read(ctx, orgID)
		if err != nil {
			return err
		}
		if org.Name != "" {
			u.OrgName = org.Name
		}
	}

	// Create the User
	u, err = a.UserService.Create(ctx, u)
	if err != nil {
		return err
	}
	fmt.Printf("Created User %s %s\n", u.ID, u.Email)
	j, err := json.MarshalIndent(u, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(j))
	return nil
}

func readUser(cmd *cobra.Command, args []string) error {
	// Initialize the application
	var a app.Application
	err := a.Init(cmd.Flag("env").Value.String())
	if err != nil {
		return fmt.Errorf("error initializing application: %w", err)
	}
	ctx := context.Background()

	// Read the specified User account(s)
	for _, arg := range args {
		u, err := a.UserService.Read(ctx, arg)
		if err != nil {
			return err
		}
		j, err := json.MarshalIndent(u, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(j))
	}
	return nil
}
