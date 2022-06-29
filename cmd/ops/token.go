package main

import (
	"context"
	"encoding/json"
	"fmt"
	"versionary-api/pkg/app"
	"versionary-api/pkg/user"

	"github.com/spf13/cobra"
)

func initTokenCmd(root *cobra.Command) {
	tokenCmd := &cobra.Command{
		Use:   "token",
		Short: "Manage OAuth Bearer tokens",
	}
	root.AddCommand(tokenCmd)

	createCmd := &cobra.Command{
		Use:   "create <userID or email>",
		Short: "Create a new token",
		Long:  "Create a new token for the specified user (email or ID).",
		Args:  cobra.ExactArgs(1),
		RunE:  createToken,
	}
	createCmd.Flags().StringP("env", "e", "", "Operating environment: dev | test | staging | prod")
	createCmd.MarkFlagRequired("env")
	tokenCmd.AddCommand(createCmd)

	readCmd := &cobra.Command{
		Use:   "read <tokenID>",
		Short: "Read specified token",
		Long:  "Read the specified token, by ID.",
		Args:  cobra.MinimumNArgs(1),
		RunE:  readToken,
	}
	readCmd.Flags().StringP("env", "e", "", "Operating environment: dev | test | staging | prod")
	readCmd.MarkFlagRequired("env")
	tokenCmd.AddCommand(readCmd)
}

func createToken(cmd *cobra.Command, args []string) error {
	// Initialize the application
	var a app.Application
	err := a.Init(cmd.Flag("env").Value.String())
	if err != nil {
		return fmt.Errorf("error initializing application: %w", err)
	}
	ctx := context.Background()

	// Read the specified User account
	u, err := a.UserService.Read(ctx, args[0])
	if err != nil {
		return fmt.Errorf("error reading user: %w", err)
	}

	// Create the Token
	t, err := a.TokenService.Create(ctx, user.Token{
		UserID: u.ID,
		Email:  u.Email,
	})
	if err != nil {
		return err
	}
	fmt.Printf("Created Token %s for User %s %s\n", t.ID, u.ID, u.Email)
	j, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(j))
	return nil
}

func readToken(cmd *cobra.Command, args []string) error {
	// Initialize the application
	var a app.Application
	err := a.Init(cmd.Flag("env").Value.String())
	if err != nil {
		return fmt.Errorf("error initializing application: %w", err)
	}
	ctx := context.Background()

	// Read the specified Token(s)
	for _, arg := range args {
		t, err := a.TokenService.Read(ctx, arg)
		if err != nil {
			return err
		}
		j, err := json.MarshalIndent(t, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(j))
	}
	return nil
}
