package bucket

import (
	"bytes"
	"context"
	"errors"
	"image"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
)

func TestPrivateBucket(t *testing.T) {
	expect := assert.New(t)

	// Configure a test bucket with a globally unique name
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		t.Fatal("error loading AWS config:", err)
	}
	ts := strconv.FormatInt(time.Now().UnixMilli(), 36)
	bucket := Bucket{
		Client:     s3.NewFromConfig(cfg),
		EntityType: "Image",
		BucketName: "versionary-test-private-" + ts,
		Public:     false,
	}

	// Verify that the bucket does not exist
	exists, err := bucket.BucketExists(ctx)
	expect.NoError(err)
	expect.False(exists)

	// Create the bucket
	err = bucket.CreateBucket(ctx)
	expect.NoError(err)

	// Verify that the bucket exists
	exists, err = bucket.BucketExists(ctx)
	expect.NoError(err)
	expect.True(exists)

	// Try to create the bucket again
	err = bucket.CreateBucket(ctx)
	expect.NoError(err)

	// Check the bucket policy to ensure it is private
	policy, err := bucket.GetBucketPolicy(ctx)
	expect.NoError(err)
	expect.Equal("", policy)

	// Check info on a file that does not exist
	_, err = bucket.FileInfo(ctx, "does-not-exist")
	expect.Error(err)
	expect.True(errors.Is(err, ErrFileNotFound))

	// Upload a test file. Credits: <a href="https://freeicons.io/profile/730">Anu Rocks</a>
	// on <a href="https://freeicons.io">freeicons.io</a>
	file128 := FileInfo{
		FileName:    "stack.128.png",
		ContentType: "image/png",
	}
	f1, err := os.Open("testdata/" + file128.FileName)
	expect.NoError(err)
	defer func(f *os.File) { _ = f.Close() }(f1)
	file128, err = bucket.UploadFile(ctx, file128, f1)
	expect.NoError(err)

	// Check the file info
	info, err := bucket.FileInfo(ctx, file128.FileName)
	expect.NoError(err)
	expect.Equal(file128, info)

	// Try to download the file without a pre-signed URL
	httpClient := http.Client{
		Timeout: 30 * time.Second,
	}
	req1, err := http.NewRequestWithContext(ctx, http.MethodGet, bucket.BucketURL()+file128.FileName, nil)
	expect.NoError(err)
	resp1, err := httpClient.Do(req1)
	expect.NoError(err)
	defer func(r *http.Response) { _ = r.Body.Close() }(resp1)
	expect.Equal(http.StatusForbidden, resp1.StatusCode)

	// Generate a pre-signed GET URL
	// Example: https://versionary-test-l9czcumv.s3.us-west-2.amazonaws.com/stack.128.png
	// ?X-Amz-Algorithm=AWS4-HMAC-SHA256
	// &X-Amz-Credential=AKIAJB5M3DLEBQXMKS5A%2F20221017%2Fus-west-2%2Fs3%2Faws4_request
	// &X-Amz-Date=20221017T161856Z
	// &X-Amz-Expires=600
	// &X-Amz-SignedHeaders=host
	// &x-id=GetObject
	// &X-Amz-Signature=7ba8349c1423ab102200d9f5fe581781e078ec252d0e7b25bd1fc951ec074a99
	psu1, err := bucket.GetDownloadURL(ctx, file128.FileName, 10*time.Minute)
	expect.NoError(err)
	expect.NotEmpty(psu1.URL)

	// Download the file with the pre-signed URL
	req2, err := http.NewRequestWithContext(ctx, http.MethodGet, psu1.URL, nil)
	expect.NoError(err)
	resp2, err := httpClient.Do(req2)
	expect.NoError(err)
	defer func(resp *http.Response) { _ = resp.Body.Close() }(resp2)
	expect.Equal(http.StatusOK, resp2.StatusCode)
	expect.Equal(file128.ContentType, resp2.Header.Get("Content-Type"))
	expect.Equal(file128.ETag, strings.ReplaceAll(resp2.Header.Get("ETag"), "\"", ""))
	expect.Equal(file128.ContentLength, resp2.ContentLength)
	img, _, err := image.Decode(resp2.Body)
	expect.NoError(err)
	expect.NotNil(img)

	// Download the file with the S3 client
	info, rc, err := bucket.DownloadFile(ctx, file128.FileName)
	expect.NoError(err)
	expect.Equal(file128, info)
	defer func(rc io.ReadCloser) { _ = rc.Close() }(rc)
	i, format, err := image.Decode(rc)
	expect.NoError(err)
	expect.Equal("png", format)
	expect.Equal(128, i.Bounds().Dx())
	expect.Equal(128, i.Bounds().Dy())

	// Download a non-existent file
	_, _, err = bucket.DownloadFile(ctx, "does-not-exist")
	expect.Error(err)
	expect.True(errors.Is(err, ErrFileNotFound))

	// Generate a pre-signed PUT URL
	// Example: https://versionary-test-l9d6kr5e.s3.us-west-2.amazonaws.com/stack.64.png
	// ?X-Amz-Algorithm=AWS4-HMAC-SHA256
	// &X-Amz-Credential=AKIAJB5M3DLEBQXMKS5A%2F20221017%2Fus-west-2%2Fs3%2Faws4_request
	// &X-Amz-Date=20221017T194023Z
	// &X-Amz-Expires=600
	// &X-Amz-SignedHeaders=host
	// &x-id=PutObject
	// &X-Amz-Signature=1a406c99341927a353d51644a35cfa6e66259dd26a9b6c3c0804abb78064159f
	file64 := FileInfo{
		BucketName:    bucket.BucketName,
		FileName:      "stack.64.png",
		ETag:          "93ce174c4a9c1bd44e73e15f0799e746",
		ContentType:   "image/png",
		ContentLength: 1015,
	}
	psu2, err := bucket.GetUploadURL(ctx, file64.FileName, file64.ContentType, 10*time.Minute)
	expect.NoError(err)
	expect.NotEmpty(psu2)

	// Upload a file with the pre-signed URL
	f2, err := os.Open("testdata/" + file64.FileName)
	expect.NoError(err)
	defer func(f *os.File) { _ = f.Close() }(f2)
	expect.NoError(err)
	body, err := os.ReadFile("testdata/" + file64.FileName)
	expect.NoError(err)
	req3, err := http.NewRequestWithContext(ctx, http.MethodPut, psu2.URL, bytes.NewReader(body))
	expect.NoError(err)
	req3.Header.Set("Host", psu2.Host)
	req3.Header.Set("Content-Type", file64.ContentType)
	req3.Header.Set("Content-Length", strconv.FormatInt(int64(len(body)), 10))
	resp3, err := httpClient.Do(req3)
	expect.NoError(err)
	defer func(resp *http.Response) { _ = resp.Body.Close() }(resp3)
	expect.Equal(http.StatusOK, resp3.StatusCode)
	expect.Equal(file64.ETag, strings.ReplaceAll(resp3.Header.Get("ETag"), "\"", ""))

	// Check the file info after the upload
	info, err = bucket.FileInfo(ctx, file64.FileName)
	expect.NoError(err)
	file64.LastModified = info.LastModified
	expect.Equal(file64, info)

	// Copy a file (to the same bucket, but it could be different)
	fileCopy, err := bucket.CopyFile(ctx, file128.BucketName, file128.FileName, "stack.png")
	expect.NoError(err)
	expect.Equal("stack.png", fileCopy.FileName)
	expect.Equal(file128.ContentType, fileCopy.ContentType)
	expect.Equal(file128.ContentLength, fileCopy.ContentLength)
	expect.Equal(file128.ETag, fileCopy.ETag)

	// Copy the file again (it should be a no-op)
	lastModified := fileCopy.LastModified
	fileCopy, err = bucket.CopyFile(ctx, file128.BucketName, file128.FileName, "stack.png")
	expect.NoError(err)
	expect.Equal("stack.png", fileCopy.FileName)
	expect.Equal(file128.ContentType, fileCopy.ContentType)
	expect.Equal(file128.ContentLength, fileCopy.ContentLength)
	expect.Equal(file128.ETag, fileCopy.ETag)
	expect.Equal(lastModified, fileCopy.LastModified)

	// Copy a non-existent file
	_, err = bucket.CopyFile(ctx, file128.BucketName, "does-not-exist", "stack.png")
	expect.Error(err)
	expect.True(errors.Is(err, ErrFileNotFound))

	// List all the files in the bucket
	files, err := bucket.ListAllFiles(ctx)
	expect.NoError(err)
	expect.Equal(3, len(files))

	// Delete one file
	expect.NoError(bucket.DeleteFile(ctx, fileCopy.FileName))

	// Delete a non-existent file (no error expected)
	expect.NoError(bucket.DeleteFile(ctx, "does-not-exist"))

	// Delete multiple files, emptying the bucket
	expect.NoError(bucket.EmptyBucket(ctx))

	// Empty an empty bucket (no error expected)
	expect.NoError(bucket.EmptyBucket(ctx))

	// Delete the bucket
	expect.NoError(bucket.DeleteBucket(ctx))

	// Verify that the bucket does not exist
	exists, err = bucket.BucketExists(ctx)
	expect.NoError(err)
	expect.False(exists)

	// Try to delete the bucket again
	expect.NoError(bucket.DeleteBucket(ctx))
}

func TestPublicBucket(t *testing.T) {
	expect := assert.New(t)

	// Configure a test bucket with a globally unique name
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		t.Fatal("error loading AWS config:", err)
	}
	ts := strconv.FormatInt(time.Now().UnixMilli(), 36)
	bucket := Bucket{
		Client:     s3.NewFromConfig(cfg),
		EntityType: "Image",
		BucketName: "versionary-test-public-" + ts,
		Public:     true,
	}

	// Create the bucket
	err = bucket.CreateBucket(ctx)
	expect.NoError(err)

	// Configure the bucket to be publicly readable
	policy, err := bucket.SetPublicAccessPolicy(ctx)
	expect.NoError(err)
	expect.True(strings.Contains(policy, "Allow"))

	// Configure the bucket to as a website
	err = bucket.SetBucketAsWebsite(ctx)
	expect.NoError(err)

	// Verify that the index file exists
	index, err := bucket.FileInfo(ctx, "index.png")
	expect.NoError(err)
	expect.Equal("image/png", index.ContentType)

	// Download the index file as a publicly readable object
	httpClient := http.Client{
		Timeout: 30 * time.Second,
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, bucket.BucketURL()+index.FileName, nil)
	expect.NoError(err)
	res, err := httpClient.Do(req)
	expect.NoError(err)
	defer func(r *http.Response) { _ = r.Body.Close() }(res)
	expect.Equal(http.StatusOK, res.StatusCode)

	// Delete the index file
	expect.NoError(bucket.DeleteFile(ctx, index.FileName))

	// Delete the bucket
	expect.NoError(bucket.DeleteBucket(ctx))
}
