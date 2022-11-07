package bucket

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

//go:embed index.png
var indexFile []byte

// indexFileInfo provides information about the default public bucket index file.
var indexFileInfo = FileInfo{
	FileName:      "index.png",
	ContentType:   "image/png",
	ContentLength: int64(len(indexFile)),
}

// ErrBucketNotConfigured is returned when the bucket is not configured.
var ErrBucketNotConfigured = errors.New("bucket not configured")

// ErrFileNotFound is returned when the file is not found.
var ErrFileNotFound = errors.New("file not found")

// ErrTooManyFiles is returned when there are too many files in the request.
var ErrTooManyFiles = errors.New("too many files")

// BucketWriter is the interface for making changes to the contents of an S3 bucket.
type BucketWriter interface {
	UploadFile(ctx context.Context, info FileInfo, file io.Reader) (FileInfo, error)
	GetUploadURL(ctx context.Context, fileName string, contentType string, expires time.Duration) (PreSignedURL, error)
	CopyFile(ctx context.Context, fromBucketName string, fromFileName string, toFileName string) (FileInfo, error)
	DeleteFile(ctx context.Context, fileName string) error
	DeleteFiles(ctx context.Context, fileNames []string) error
}

// BucketReader is the interface for reading the contents of an S3 bucket.
type BucketReader interface {
	IsValid() bool
	FileExists(ctx context.Context, fileName string) (bool, error)
	FileInfo(ctx context.Context, fileName string) (FileInfo, error)
	DownloadFile(ctx context.Context, fileName string) (FileInfo, io.ReadCloser, error)
	GetDownloadURL(ctx context.Context, fileName string, expires time.Duration) (PreSignedURL, error)
	ListAllFiles(ctx context.Context) ([]FileInfo, error)
}

// BucketReadWriter is the interface for reading and writing the contents of an S3 bucket.
type BucketReadWriter interface {
	BucketReader
	BucketWriter
}

// FileInfo provides basic information about a file in a bucket.
type FileInfo struct {
	BucketName    string    `json:"bucketName"`
	FileName      string    `json:"fileName"`
	ContentType   string    `json:"contentType,omitempty"`
	ContentLength int64     `json:"contentLength,omitempty"`
	ETag          string    `json:"etag,omitempty"`
	LastModified  time.Time `json:"lastModified,omitempty"`
}

// PreSignedURL provides information for uploading or downloading a file.
type PreSignedURL struct {
	BucketName    string    `json:"bucketName"`
	FileName      string    `json:"fileName"`
	ContentType   string    `json:"contentType,omitempty"`
	ContentLength int64     `json:"contentLength,omitempty"`
	ETag          string    `json:"etag,omitempty"`
	ExpiresAt     time.Time `json:"expiresAt"`
	Method        string    `json:"method"`
	Host          string    `json:"host"`
	URL           string    `json:"url"`
}

// Bucket is a struct for interacting with S3 buckets.
type Bucket struct {
	Client     *s3.Client
	EntityType string // Entity that contains metadata about files in the bucket.
	BucketName string // Bucket name, including environment suffix.
	Public     bool   // If true, all files in the bucket are accessible by anonymous users.
}

// IsValid returns true if the bucket is configured properly.
func (b Bucket) IsValid() bool {
	return b.Client != nil && b.EntityType != "" && b.BucketName != ""
}

// BucketURL returns the base URL for the bucket, with a trailing slash.
func (b Bucket) BucketURL() string {
	if b.BucketName == "" {
		return ""
	}
	return "https://" + b.BucketName + ".s3.us-west-2.amazonaws.com/"
}

// BucketExists returns true if the bucket exists and the client has permission to access it.
func (b Bucket) BucketExists(ctx context.Context) (bool, error) {
	if b.Client == nil || b.BucketName == "" {
		return false, ErrBucketNotConfigured
	}
	output, err := b.Client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: &b.BucketName,
	})
	if err != nil {
		// Not Found is an expected error.
		var nf *types.NotFound
		if errors.As(err, &nf) {
			return false, nil
		}
		return false, fmt.Errorf("bucket %s exists: %w", b.BucketName, err)
	}
	return output != nil, nil
}

// CreateBucket creates the bucket if it does not already exist.
func (b Bucket) CreateBucket(ctx context.Context) error {
	startTime := time.Now()
	if b.Client == nil || b.BucketName == "" {
		return ErrBucketNotConfigured
	}
	// Check if the bucket already exists
	exists, err := b.BucketExists(ctx)
	if err != nil {
		return fmt.Errorf("create bucket %s: %w", b.BucketName, err)
	}
	if exists {
		log.Println("bucket", b.BucketName, "EXISTS", time.Since(startTime))
		return nil
	}
	// Create the bucket
	req := s3.CreateBucketInput{
		Bucket: &b.BucketName,
		CreateBucketConfiguration: &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraintUsWest2,
		},
	}
	if b.Public {
		req.ACL = types.BucketCannedACLPublicRead
	} else {
		req.ACL = types.BucketCannedACLPrivate
	}
	_, err = b.Client.CreateBucket(ctx, &req)
	if err != nil {
		return fmt.Errorf("create bucket %s: %w", b.BucketName, err)
	}
	// Wait for the bucket to be available
	waiter := s3.NewBucketExistsWaiter(b.Client, func(options *s3.BucketExistsWaiterOptions) {
		options.MinDelay = 3 * time.Second
		options.MaxDelay = 120 * time.Second
	})
	err = waiter.Wait(ctx, &s3.HeadBucketInput{Bucket: &b.BucketName}, 120*time.Second)
	if err != nil {
		return fmt.Errorf("create bucket %s: %w", b.BucketName, err)
	}
	// Set up public access, if needed
	if b.Public {
		_, err = b.SetPublicAccessPolicy(ctx)
		if err != nil {
			return fmt.Errorf("create bucket %s: %w", b.BucketName, err)
		}
		err = b.SetBucketAsWebsite(ctx)
		if err != nil {
			return fmt.Errorf("create bucket %s: %w", b.BucketName, err)
		}
	}
	log.Println("bucket", b.BucketName, "CREATED", time.Since(startTime))
	return nil
}

// EmptyBucket deletes all files in the bucket. It does not delete the bucket itself.
// Caution: This is a destructive operation. Use with care.
func (b Bucket) EmptyBucket(ctx context.Context) error {
	startTime := time.Now()
	if b.Client == nil || b.BucketName == "" {
		return ErrBucketNotConfigured
	}
	// Get a list of all files in the bucket
	files, err := b.ListAllFiles(ctx)
	if err != nil {
		return fmt.Errorf("empty bucket %s: %w", b.BucketName, err)
	}
	if len(files) == 0 {
		return nil
	}
	// Generate batches of a maximum of 1000 files
	batches := make([][]string, 0)
	batch := make([]string, 0)
	for _, file := range files {
		batch = append(batch, file.FileName)
		if len(batch) == 1000 {
			batches = append(batches, batch)
			batch = make([]string, 0)
		}
	}
	if len(batch) > 0 {
		batches = append(batches, batch)
	}
	// Delete all objects in the bucket
	for _, batch = range batches {
		err = b.DeleteFiles(ctx, batch)
		if err != nil {
			return fmt.Errorf("empty bucket %s: %w", b.BucketName, err)
		}
	}
	log.Println("bucket", b.BucketName, "EMPTIED", time.Since(startTime), len(files), "file(s)")
	return nil
}

// DeleteBucket deletes the bucket if it exists.
// Note that the bucket must be empty before it can be deleted.
func (b Bucket) DeleteBucket(ctx context.Context) error {
	startTime := time.Now()
	if b.Client == nil || b.BucketName == "" {
		return ErrBucketNotConfigured
	}
	// Check if the bucket exists
	exists, err := b.BucketExists(ctx)
	if err != nil {
		return fmt.Errorf("delete bucket %s: %w", b.BucketName, err)
	}
	if !exists {
		log.Println("bucket", b.BucketName, "NOT FOUND", time.Since(startTime))
		return nil
	}
	// Delete the bucket
	_, err = b.Client.DeleteBucket(ctx, &s3.DeleteBucketInput{
		Bucket: &b.BucketName,
	})
	if err != nil {
		return fmt.Errorf("delete bucket %s: %w", b.BucketName, err)
	}
	// Wait for the bucket to be deleted
	waiter := s3.NewBucketNotExistsWaiter(b.Client, func(options *s3.BucketNotExistsWaiterOptions) {
		options.MinDelay = 3 * time.Second
		options.MaxDelay = 120 * time.Second
	})
	err = waiter.Wait(ctx, &s3.HeadBucketInput{Bucket: &b.BucketName}, 120*time.Second)
	if err != nil {
		return fmt.Errorf("delete bucket %s: %w", b.BucketName, err)
	}
	log.Println("bucket", b.BucketName, "DELETED", time.Since(startTime))
	return nil
}

// GetBucketPolicy returns the current bucket policy text, if any.
func (b Bucket) GetBucketPolicy(ctx context.Context) (string, error) {
	if b.Client == nil || b.BucketName == "" {
		return "", ErrBucketNotConfigured
	}
	output, err := b.Client.GetBucketPolicy(ctx, &s3.GetBucketPolicyInput{Bucket: &b.BucketName})
	if err != nil {
		// Not Found is an expected error.
		var nf *smithy.GenericAPIError
		if errors.As(err, &nf) {
			if nf.ErrorCode() == "NoSuchBucketPolicy" {
				return "", nil
			}
		}
		return "", fmt.Errorf("get bucket %s policy: %w", b.BucketName, err)
	}
	return *output.Policy, nil
}

// SetPublicAccessPolicy sets the bucket policy to public read access.
func (b Bucket) SetPublicAccessPolicy(ctx context.Context) (string, error) {
	if b.Client == nil || b.BucketName == "" {
		return "", ErrBucketNotConfigured
	}
	// Set the bucket policy
	policy :=
		`{
			"Version": "2012-10-17",
			"Statement": [
				{
					"Sid": "` + b.BucketName + `-public-read",
					"Effect": "Allow",
					"Principal": "*",
					"Action": "s3:GetObject",
					"Resource": "arn:aws:s3:::` + b.BucketName + `/*"
				}
			]
		}`
	_, err := b.Client.PutBucketPolicy(ctx, &s3.PutBucketPolicyInput{
		Bucket: &b.BucketName,
		Policy: aws.String(policy),
	})
	if err != nil {
		return "", fmt.Errorf("set bucket %s policy: %w", b.BucketName, err)
	}
	return policy, nil
}

// SetBucketAsWebsite sets the bucket as a website with public read access.
func (b Bucket) SetBucketAsWebsite(ctx context.Context) error {
	if b.Client == nil || b.BucketName == "" {
		return ErrBucketNotConfigured
	}
	// Set the bucket policy to public read access
	if policy, _ := b.GetBucketPolicy(ctx); policy == "" {
		_, err := b.SetPublicAccessPolicy(ctx)
		if err != nil {
			return fmt.Errorf("set bucket %s as website: %w", b.BucketName, err)
		}
	}
	// Upload the default index file
	if exists, _ := b.FileExists(ctx, indexFileInfo.FileName); !exists {
		_, err := b.UploadFile(ctx, indexFileInfo, bytes.NewReader(indexFile))
		if err != nil {
			return fmt.Errorf("set bucket %s as website: %w", b.BucketName, err)
		}
	}
	// Configure the bucket as a website
	_, err := b.Client.PutBucketWebsite(ctx, &s3.PutBucketWebsiteInput{
		Bucket: &b.BucketName,
		WebsiteConfiguration: &types.WebsiteConfiguration{
			IndexDocument: &types.IndexDocument{Suffix: &indexFileInfo.FileName},
			ErrorDocument: &types.ErrorDocument{Key: &indexFileInfo.FileName},
		},
	})
	if err != nil {
		return fmt.Errorf("set bucket %s as website: %w", b.BucketName, err)
	}
	return nil
}

// FileExists returns true if the file exists in the bucket.
func (b Bucket) FileExists(ctx context.Context, fileName string) (bool, error) {
	if b.Client == nil || b.BucketName == "" {
		return false, ErrBucketNotConfigured
	}
	output, err := b.Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: &b.BucketName,
		Key:    &fileName,
	})
	if err != nil {
		// Not Found is an expected error.
		if strings.Contains(err.Error(), "NotFound") {
			return false, nil
		}
		return false, fmt.Errorf("file %s exists: %w", fileName, err)
	}
	return output != nil, nil
}

// FileInfo returns information about the file in the bucket.
func (b Bucket) FileInfo(ctx context.Context, fileName string) (FileInfo, error) {
	info := FileInfo{
		BucketName: b.BucketName,
		FileName:   fileName,
	}
	if b.Client == nil || b.BucketName == "" {
		return info, ErrBucketNotConfigured
	}
	output, err := b.Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: &b.BucketName,
		Key:    &fileName,
	})
	if err != nil {
		if strings.Contains(err.Error(), "NotFound") {
			return info, ErrFileNotFound
		}
		return info, fmt.Errorf("file %s info in %s: %w", fileName, b.BucketName, err)
	}
	info.ETag = strings.ReplaceAll(*output.ETag, "\"", "") // remove quotes
	info.ContentType = *output.ContentType
	info.ContentLength = output.ContentLength
	info.LastModified = *output.LastModified
	return info, nil
}

// UploadFile uploads a file to the bucket.
func (b Bucket) UploadFile(ctx context.Context, info FileInfo, file io.Reader) (FileInfo, error) {
	if b.Client == nil || b.BucketName == "" {
		return info, ErrBucketNotConfigured
	}
	// Generate a request with optional fields for additional validation
	info.BucketName = b.BucketName
	req := s3.PutObjectInput{
		Bucket: &b.BucketName,
		Key:    &info.FileName,
		Body:   file,
	}
	if info.ContentType != "" {
		req.ContentType = &info.ContentType
	}
	if info.ContentLength != 0 {
		req.ContentLength = info.ContentLength
	}
	// Upload the file
	res, err := b.Client.PutObject(ctx, &req)
	if err != nil {
		return info, fmt.Errorf("upload %s to %s: %w", info.FileName, b.BucketName, err)
	}
	info.ETag = strings.ReplaceAll(*res.ETag, "\"", "") // remove quotes
	// Return complete file info, on a best-effort basis
	f, err := b.FileInfo(ctx, info.FileName)
	if err != nil {
		return info, nil
	}
	return f, nil
}

// DownloadFile downloads a file from the bucket.
func (b Bucket) DownloadFile(ctx context.Context, fileName string) (FileInfo, io.ReadCloser, error) {
	info := FileInfo{
		BucketName: b.BucketName,
		FileName:   fileName,
	}
	if b.Client == nil || b.BucketName == "" {
		return info, nil, ErrBucketNotConfigured
	}
	// Download the file
	res, err := b.Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &b.BucketName,
		Key:    &fileName,
	})
	if err != nil {
		var nf *types.NoSuchKey
		if errors.As(err, &nf) {
			return info, nil, ErrFileNotFound
		}
		return info, nil, fmt.Errorf("download %s from %s: %w", fileName, b.BucketName, err)
	}
	// Return complete file info, as available
	info.ContentLength = res.ContentLength
	if res.ETag != nil {
		info.ETag = strings.ReplaceAll(*res.ETag, "\"", "") // remove unnecessary quotes
	}
	if res.ContentType != nil {
		info.ContentType = *res.ContentType
	}
	if res.LastModified != nil {
		info.LastModified = *res.LastModified
	}
	return info, res.Body, nil
}

// GetUploadURL returns a pre-signed URL for uploading a file to the bucket.
// The maximum duration before expiration is 6 hours for an IAM instance profile.
func (b Bucket) GetUploadURL(ctx context.Context, fileName string, contentType string, expires time.Duration) (PreSignedURL, error) {
	psu := PreSignedURL{
		BucketName:  b.BucketName,
		FileName:    fileName,
		ContentType: contentType,
		ExpiresAt:   time.Now().Add(expires),
	}
	if b.Client == nil || b.BucketName == "" {
		return psu, ErrBucketNotConfigured
	}
	psClient := s3.NewPresignClient(b.Client)
	params := &s3.PutObjectInput{
		Bucket:      &b.BucketName,
		Key:         &fileName,
		ContentType: &contentType,
	}
	duration := func(po *s3.PresignOptions) { po.Expires = expires }
	url, err := psClient.PresignPutObject(ctx, params, duration)
	if err != nil {
		return psu, fmt.Errorf("upload url for %s in %s: %w", fileName, b.BucketName, err)
	}
	psu.Method = url.Method
	psu.Host = url.SignedHeader.Get("Host")
	psu.URL = url.URL
	return psu, nil
}

// GetDownloadURL returns a pre-signed URL for downloading a file from the bucket.
// The maximum duration before expiration is 6 hours for an IAM instance profile.
func (b Bucket) GetDownloadURL(ctx context.Context, fileName string, expires time.Duration) (PreSignedURL, error) {
	psu := PreSignedURL{
		BucketName: b.BucketName,
		FileName:   fileName,
		ExpiresAt:  time.Now().Add(expires),
	}
	info, err := b.FileInfo(ctx, fileName)
	if err != nil {
		return psu, err
	}
	psu.ContentType = info.ContentType
	psu.ContentLength = info.ContentLength
	psu.ETag = info.ETag
	psClient := s3.NewPresignClient(b.Client)
	params := &s3.GetObjectInput{
		Bucket: &b.BucketName,
		Key:    &fileName,
	}
	duration := func(po *s3.PresignOptions) { po.Expires = expires }
	url, err := psClient.PresignGetObject(ctx, params, duration)
	if err != nil {
		return psu, fmt.Errorf("download url for %s in %s: %w", fileName, b.BucketName, err)
	}
	psu.Method = url.Method
	psu.Host = url.SignedHeader.Get("Host")
	psu.URL = url.URL
	return psu, nil
}

// CopyFile copies a file from one bucket to another.
func (b Bucket) CopyFile(ctx context.Context, fromBucketName string, fromFileName string, toFileName string) (FileInfo, error) {
	src := FileInfo{
		BucketName: fromBucketName,
		FileName:   fromFileName,
	}
	if b.Client == nil || b.BucketName == "" {
		return src, ErrBucketNotConfigured
	}

	// Verify that the source file exists
	head, err := b.Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: &fromBucketName,
		Key:    &fromFileName,
	})
	if err != nil {
		if strings.Contains(err.Error(), "NotFound") {
			return src, ErrFileNotFound
		}
		return src, fmt.Errorf("copy file %s in %s info: %w", fromFileName, fromBucketName, err)
	}
	src.ETag = strings.ReplaceAll(*head.ETag, "\"", "") // remove quotes
	src.ContentType = *head.ContentType
	src.ContentLength = head.ContentLength
	src.LastModified = *head.LastModified

	// Check if the destination file exists, with a matching ETag (no need to copy)
	dest, err := b.FileInfo(ctx, toFileName)
	if err != nil && !errors.Is(err, ErrFileNotFound) {
		return dest, fmt.Errorf("copy file %s in %s info: %w", fromFileName, fromBucketName, err)
	}
	if err == nil && dest.ETag == src.ETag && dest.ContentType == src.ContentType {
		return dest, nil
	}
	dest.ContentType = src.ContentType
	dest.ContentLength = src.ContentLength

	// Copy the file
	req := s3.CopyObjectInput{
		CopySource: aws.String(fromBucketName + "/" + fromFileName),
		Bucket:     &b.BucketName,
		Key:        &toFileName,
	}
	if src.ContentType != "" {
		req.ContentType = &src.ContentType
	}
	res, err := b.Client.CopyObject(ctx, &req)
	if err != nil {
		return dest, fmt.Errorf("copy %s from %s to %s: %w", fromFileName, fromBucketName, b.BucketName, err)
	}
	if res.CopyObjectResult != nil {
		if res.CopyObjectResult.ETag != nil {
			dest.ETag = strings.ReplaceAll(*res.CopyObjectResult.ETag, "\"", "") // remove quotes
		}
		if res.CopyObjectResult.LastModified != nil {
			dest.LastModified = *res.CopyObjectResult.LastModified
		}
	}
	return dest, nil
}

// DeleteFile deletes a file from the bucket.
func (b Bucket) DeleteFile(ctx context.Context, fileName string) error {
	if b.Client == nil || b.BucketName == "" {
		return ErrBucketNotConfigured
	}
	// Delete the file
	_, err := b.Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &b.BucketName,
		Key:    &fileName,
	})
	// No error occurs if the file does not exist
	if err != nil {
		return fmt.Errorf("delete file %s from %s: %w", fileName, b.BucketName, err)
	}
	return nil
}

// DeleteFiles deletes multiple (maximum 1000) files from the bucket.
func (b Bucket) DeleteFiles(ctx context.Context, fileNames []string) error {
	if b.Client == nil || b.BucketName == "" {
		return ErrBucketNotConfigured
	}
	// Maximum 1000 files can be deleted at once
	if len(fileNames) > 1000 {
		return ErrTooManyFiles
	}
	// Generate list of Object Identifiers
	objects := make([]types.ObjectIdentifier, len(fileNames))
	for i, fileName := range fileNames {
		objects[i] = types.ObjectIdentifier{
			Key:       aws.String(fileName),
			VersionId: nil,
		}
	}
	// Delete the files
	req := s3.DeleteObjectsInput{
		Bucket: &b.BucketName,
		Delete: &types.Delete{
			Objects: objects,
			Quiet:   true,
		},
	}
	res, err := b.Client.DeleteObjects(ctx, &req)
	if err != nil {
		// Note that no error occurs if the file does not exist
		return fmt.Errorf("delete files from %s: %w", b.BucketName, err)
	}
	if res.Errors != nil && len(res.Errors) > 0 {
		var errs []string
		for _, err := range res.Errors {
			errs = append(errs, errorString(err))
		}
		return fmt.Errorf("delete files from %s:\n%s", b.BucketName, strings.Join(errs, "\n"))
	}
	return nil
}

// errorString provides a string representation of an S3 Error.
func errorString(err types.Error) string {
	var msg string
	if err.Code != nil {
		msg += *err.Code + ": "
	}
	if err.Key != nil {
		msg += *err.Key + ": "
	}
	if err.Message != nil {
		msg += *err.Message
	}
	if err.VersionId != nil {
		msg += " (version " + *err.VersionId + ")"
	}
	return msg
}

// ListAllFiles returns a list of files in the bucket. This may be a large list!
func (b Bucket) ListAllFiles(ctx context.Context) ([]FileInfo, error) {
	if b.Client == nil || b.BucketName == "" {
		return nil, ErrBucketNotConfigured
	}
	// List the files using a paginator
	var files []FileInfo
	p := s3.NewListObjectsV2Paginator(b.Client, &s3.ListObjectsV2Input{Bucket: &b.BucketName})
	for p.HasMorePages() {
		page, err := p.NextPage(ctx)
		if err != nil {
			return files, fmt.Errorf("list files in %s: %w", b.BucketName, err)
		}
		for _, f := range page.Contents {
			files = append(files, FileInfo{
				BucketName:    b.BucketName,
				FileName:      *f.Key,
				ContentLength: f.Size,
				ETag:          strings.ReplaceAll(*f.ETag, "\"", ""), // remove unnecessary quotes
				LastModified:  *f.LastModified,
			})
		}
	}
	return files, nil
}
