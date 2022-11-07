package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"versionary-api/pkg/image"

	"github.com/spf13/cobra"
)

// initImageCmd initializes the image commands.
func initImageCmd(root *cobra.Command) {
	imageCmd := &cobra.Command{
		Use:   "image",
		Short: "Manage images",
	}
	root.AddCommand(imageCmd)

	// Upload an image to S3.
	uploadCmd := &cobra.Command{
		Use:   "upload <path/filename>",
		Short: "Upload an image",
		Long:  `Create an image and upload the associated file to S3.`,
		Args:  cobra.ExactArgs(1),
		RunE:  uploadImage,
	}
	uploadCmd.Flags().StringP("env", "e", "", "Operating environment: dev | test | staging | prod")
	uploadCmd.Flags().StringP("mediatype", "m", "", "Media Type: image/jpeg | image/webp | image/png | image/gif")
	uploadCmd.Flags().StringP("title", "t", "", "Image title for display")
	uploadCmd.Flags().StringP("alt", "a", "", "Alternate text (for accessibility)")
	_ = uploadCmd.MarkFlagRequired("env")
	imageCmd.AddCommand(uploadCmd)

	// List images
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List images",
		Long:  `List all images.`,
		RunE:  listImages,
	}
	listCmd.Flags().StringP("env", "e", "", "Operating environment: dev | test | staging | prod")
	listCmd.Flags().BoolP("sorted", "s", false, "Sort by Label?")
	_ = listCmd.MarkFlagRequired("env")
	imageCmd.AddCommand(listCmd)

	// Read an image metadata
	readCmd := &cobra.Command{
		Use:   "read <imageID>",
		Short: "Read specified image",
		Long:  `Read the specified image, by ID.`,
		Args:  cobra.ExactArgs(1),
		RunE:  readImage,
	}
	readCmd.Flags().StringP("env", "e", "", "Operating environment: dev | test | staging | prod")
	_ = readCmd.MarkFlagRequired("env")
	imageCmd.AddCommand(readCmd)
}

// uploadImage uploads an image to S3.
func uploadImage(cmd *cobra.Command, args []string) error {
	// Initialize the application
	err := ops.Init(cmd.Flag("env").Value.String())
	if err != nil {
		return fmt.Errorf("error initializing application: %w", err)
	}
	ctx := context.Background()

	// Parse flags for the image
	mediaType := cmd.Flag("mediatype").Value.String()
	title := cmd.Flag("title").Value.String()
	alt := cmd.Flag("alt").Value.String()
	sourceFileName := args[0]
	i := image.Image{
		Title:          title,
		AltText:        alt,
		SourceFileName: sourceFileName,
	}

	// Determine the media type
	blob, err := ops.ImageService.FetchSourceFile(ctx, sourceFileName)
	if err != nil {
		return fmt.Errorf("error fetching source file %s: %w", sourceFileName, err)
	}
	detected := http.DetectContentType(blob)
	if detected != "application/octet-stream" && mediaType == "" {
		mediaType = detected
	}
	if mediaType == "" {
		return fmt.Errorf("error detecting media type for %s", sourceFileName)
	}
	i.MediaType = image.MediaType(mediaType)

	// Create and upload the image
	i, _, err = ops.ImageService.Create(ctx, i)
	if err != nil {
		return err
	}
	fmt.Printf("Created Image %s %s\n", i.ID, i.Label())
	j, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshalling JSON Image %s: %w", i.ID, err)
	}
	fmt.Println(string(j))
	return nil
}

// listImages lists all images.
func listImages(cmd *cobra.Command, args []string) error {
	// Initialize the application
	err := ops.Init(cmd.Flag("env").Value.String())
	if err != nil {
		return fmt.Errorf("error initializing application: %w", err)
	}
	ctx := context.Background()

	// Parse the flags
	sorted, err := cmd.Flags().GetBool("sorted")
	if err != nil {
		return fmt.Errorf("error getting sorted flag: %w", err)
	}

	// List images
	images, err := ops.ImageService.ReadAllImageLabels(ctx, sorted)
	if err != nil {
		return err
	}
	for _, i := range images {
		fmt.Printf("%s\t%s\n", i.Key, i.Value)
	}
	return nil
}

// readImage reads the specified image.
func readImage(cmd *cobra.Command, args []string) error {
	// Initialize the application
	err := ops.Init(cmd.Flag("env").Value.String())
	if err != nil {
		return fmt.Errorf("error initializing application: %w", err)
	}
	ctx := context.Background()

	// Read the image
	i, err := ops.ImageService.Read(ctx, args[0])
	if err != nil {
		return err
	}
	j, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshalling JSON Image %s: %w", i.ID, err)
	}
	fmt.Println(string(j))
	return nil
}
