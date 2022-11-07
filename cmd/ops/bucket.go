package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"versionary-api/pkg/bucket"
	"versionary-api/pkg/image"

	"github.com/spf13/cobra"
)

// initBucketCmd initializes the bucket commands.
func initBucketCmd(root *cobra.Command) {
	bucketCmd := &cobra.Command{
		Use:   "bucket",
		Short: "Manage S3 buckets",
	}
	bucketCmd.Flags().StringP("env", "e", "", "Operating environment: dev | test | staging | prod")
	root.AddCommand(bucketCmd)

	checkCmd := &cobra.Command{
		Use:   "check [entityType...]",
		Short: "Ensure that S3 bucket(s) exist",
		Long: `Check each specified S3 bucket, creating them if they do not exist.
If no entity types are specified, all buckets in the specified environment will be checked.`,
		RunE: checkBuckets,
	}
	checkCmd.Flags().StringP("env", "e", "", "Operating environment: dev | test | staging | prod")
	_ = checkCmd.MarkFlagRequired("env")
	bucketCmd.AddCommand(checkCmd)

	deleteCmd := &cobra.Command{
		Use:   "delete [entityType...]",
		Short: "Delete S3 bucket(s)",
		Long: `Delete each specified S3 bucket in a non-production environment.
If no entity types are specified, all buckets in the specified environment will be deleted.
Note that the buckets will be emptied before they are deleted.`,
		RunE: deleteBuckets,
	}
	deleteCmd.Flags().StringP("env", "e", "", "Operating environment: dev | test | staging")
	_ = deleteCmd.MarkFlagRequired("env")
	bucketCmd.AddCommand(deleteCmd)
}

// checkBuckets checks each bucket in the specified environment.
func checkBuckets(cmd *cobra.Command, args []string) error {
	err := ops.Init(cmd.Flag("env").Value.String())
	if err != nil {
		return fmt.Errorf("error initializing application: %w", err)
	}
	ctx := context.Background()

	// Check each bucket
	var buckets []string
	if len(args) > 0 {
		// If bucket names were specified, only check those.
		buckets = args
	} else {
		// Otherwise, check all buckets.
		buckets = []string{"Image"}
	}
	fmt.Printf("Checking %d Bucket(s) in %s: %s\n", len(buckets), ops.Environment, strings.Join(buckets, ", "))
	for _, entity := range buckets {
		// TODO: add new S3 buckets here
		switch entity {
		case "Image":
			checkBucket(ctx, image.NewBucket(ops.S3Client, ops.Environment))
		default:
			fmt.Println("Skipping unknown entity type:", entity)
		}
	}
	return nil
}

// checkBucket creates a S3 bucket if it does not already exist.
func checkBucket(ctx context.Context, b bucket.Bucket) {
	if !b.IsValid() {
		log.Println("bucket", b.BucketName, "INVALID bucket definition - skipping")
		return
	}
	exists, err := b.BucketExists(ctx)
	if err != nil {
		log.Println("bucket", b.BucketName, "ERROR checking bucket:", err)
		return
	}
	if exists {
		log.Println("bucket", b.BucketName, "EXISTS")
	} else {
		if err := b.CreateBucket(ctx); err != nil {
			log.Println("bucket", b.BucketName, "ERROR creating bucket:", err)
		}
	}
}

// deleteBuckets deletes each bucket in the specified environment.
func deleteBuckets(cmd *cobra.Command, args []string) error {
	err := ops.Init(cmd.Flag("env").Value.String())
	if err != nil {
		return fmt.Errorf("error initializing application: %w", err)
	}
	ctx := context.Background()

	// Not for use in production!
	if ops.Environment == "prod" {
		return errors.New("delete buckets in production? Really? Use the AWS console instead")
	}

	// Delete each bucket
	var buckets []string
	if len(args) > 0 {
		// If bucket names were specified, only delete those.
		buckets = args
	} else {
		// Otherwise, delete all buckets.
		buckets = []string{"Image"}
	}
	fmt.Printf("Deleting %d Bucket(s) in %s: %s\n", len(buckets), ops.Environment, strings.Join(buckets, ", "))
	for _, entity := range buckets {
		// TODO: add new S3 buckets here
		switch entity {
		case "Image":
			deleteBucket(ctx, image.NewBucket(ops.S3Client, ops.Environment))
		default:
			fmt.Println("Skipping unknown entity type:", entity)
		}
	}
	return nil
}

// deleteBucket deletes an S3 bucket.
func deleteBucket(ctx context.Context, b bucket.Bucket) {
	// Verify that the bucket exists
	exists, err := b.BucketExists(ctx)
	if err != nil {
		log.Println("bucket", b.BucketName, "ERROR checking bucket:", err)
		return
	}
	if !exists {
		log.Println("bucket", b.BucketName, "MISSING - skipping")
		return
	}
	// Empty the bucket
	if err := b.EmptyBucket(ctx); err != nil {
		log.Println("bucket", b.BucketName, "ERROR emptying bucket:", err)
		return
	}
	// Delete the bucket
	if err := b.DeleteBucket(ctx); err != nil {
		log.Println("bucket", b.BucketName, "ERROR deleting bucket:", err)
	}
}
