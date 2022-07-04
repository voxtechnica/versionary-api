package main

import (
	"context"
	"encoding/json"
	"fmt"
	"versionary-api/pkg/user"

	"github.com/spf13/cobra"
)

// initTokenCmd initializes the token commands.
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
	_ = createCmd.MarkFlagRequired("env")
	tokenCmd.AddCommand(createCmd)

	listCmd := &cobra.Command{
		Use:   "list <userID or email>",
		Short: "List all tokens",
		Long:  "List all tokens for the specified user.",
		Args:  cobra.ExactArgs(1),
		RunE:  listTokens,
	}
	listCmd.Flags().StringP("env", "e", "", "Operating environment: dev | test | staging | prod")
	_ = listCmd.MarkFlagRequired("env")
	tokenCmd.AddCommand(listCmd)

	readCmd := &cobra.Command{
		Use:   "read <tokenID>",
		Short: "Read specified token",
		Long:  "Read the specified token, by ID.",
		Args:  cobra.MinimumNArgs(1),
		RunE:  readToken,
	}
	readCmd.Flags().StringP("env", "e", "", "Operating environment: dev | test | staging | prod")
	_ = readCmd.MarkFlagRequired("env")
	tokenCmd.AddCommand(readCmd)
}

// createToken creates a new token for the specified user.
func createToken(cmd *cobra.Command, args []string) error {
	// Initialize the application
	err := ops.Init(cmd.Flag("env").Value.String())
	if err != nil {
		return fmt.Errorf("error initializing application: %w", err)
	}
	ctx := context.Background()

	// Read the specified User account
	u, err := ops.UserService.Read(ctx, args[0])
	if err != nil {
		return fmt.Errorf("error reading user %s: %w", args[0], err)
	}

	// Create the Token
	t, err := ops.TokenService.Create(ctx, user.Token{
		UserID: u.ID,
		Email:  u.Email,
	})
	if err != nil {
		return fmt.Errorf("error creating token for user %s: %w", u.ID, err)
	}
	fmt.Printf("Created Token %s for User %s %s\n", t.ID, u.ID, u.Email)
	j, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshalling JSON token %s: %w", t.ID, err)
	}
	fmt.Println(string(j))
	return nil
}

// listTokens lists all tokens for the specified user.
func listTokens(cmd *cobra.Command, args []string) error {
	// Initialize the application
	err := ops.Init(cmd.Flag("env").Value.String())
	if err != nil {
		return fmt.Errorf("error initializing application: %w", err)
	}
	ctx := context.Background()

	// Read the specified User account
	u, err := ops.UserService.Read(ctx, args[0])
	if err != nil {
		return fmt.Errorf("error reading user %s: %w", args[0], err)
	}

	// List all Tokens for the specified User
	tokens, err := ops.TokenService.ReadAllTokensByUserID(ctx, u.ID)
	if err != nil {
		return fmt.Errorf("error reading tokens for user %s: %w", u.ID, err)
	}
	j, err := json.MarshalIndent(tokens, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshalling JSON tokens for user %s: %w", u.ID, err)
	}
	fmt.Println(string(j))
	return nil
}

// readToken reads the specified token.
func readToken(cmd *cobra.Command, args []string) error {
	// Initialize the application
	err := ops.Init(cmd.Flag("env").Value.String())
	if err != nil {
		return fmt.Errorf("error initializing application: %w", err)
	}
	ctx := context.Background()

	// Read the specified Token(s)
	for _, arg := range args {
		t, err := ops.TokenService.Read(ctx, arg)
		if err != nil {
			return fmt.Errorf("error reading token %s: %w", arg, err)
		}
		j, err := json.MarshalIndent(t, "", "  ")
		if err != nil {
			return fmt.Errorf("error marshalling JSON token %s: %w", arg, err)
		}
		fmt.Println(string(j))
	}
	return nil
}
