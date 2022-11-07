package bucket

import (
	"context"
	"errors"
	"image"
	_ "image/png"
	"io"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPrivateMemBucket(t *testing.T) {
	expect := assert.New(t)

	// Configure a test bucket with a globally unique name
	ctx := context.Background()
	ts := strconv.FormatInt(time.Now().UnixMilli(), 36)
	mb := MemBucket{
		EntityType: "Image",
		BucketName: "versionary-test-private-" + ts,
		Public:     false,
	}

	// Verify that the bucket does not exist
	exists := mb.BucketExists()
	expect.False(exists)

	// Initialize the bucket
	mb.FileSet = &MemFileSet{}

	// Verify that the bucket exists
	exists = mb.BucketExists()
	expect.True(exists)

	// Check info on a file that does not exist
	_, err := mb.FileInfo(ctx, "does-not-exist")
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
	file128, err = mb.UploadFile(ctx, file128, f1)
	expect.NoError(err)

	// Check the file info
	info, err := mb.FileInfo(ctx, file128.FileName)
	expect.NoError(err)
	expect.Equal(file128, info)

	// Download the file with the S3 client
	info, rc, err := mb.DownloadFile(ctx, file128.FileName)
	expect.NoError(err)
	expect.Equal(file128, info)
	defer func(rc io.ReadCloser) { _ = rc.Close() }(rc)
	i, format, err := image.Decode(rc)
	expect.NoError(err)
	expect.Equal("png", format)
	expect.Equal(128, i.Bounds().Dx())
	expect.Equal(128, i.Bounds().Dy())

	// Download a non-existent file
	_, _, err = mb.DownloadFile(ctx, "does-not-exist")
	expect.Error(err)
	expect.True(errors.Is(err, ErrFileNotFound))

	// Copy a file (to the same bucket)
	fileCopy, err := mb.CopyFile(ctx, file128.BucketName, file128.FileName, "stack.png")
	expect.NoError(err)
	expect.Equal("stack.png", fileCopy.FileName)
	expect.Equal(file128.ContentType, fileCopy.ContentType)
	expect.Equal(file128.ContentLength, fileCopy.ContentLength)
	expect.Equal(file128.ETag, fileCopy.ETag)

	// Copy the file again (it should be a no-op)
	lastModified := fileCopy.LastModified
	fileCopy, err = mb.CopyFile(ctx, file128.BucketName, file128.FileName, "stack.png")
	expect.NoError(err)
	expect.Equal("stack.png", fileCopy.FileName)
	expect.Equal(file128.ContentType, fileCopy.ContentType)
	expect.Equal(file128.ContentLength, fileCopy.ContentLength)
	expect.Equal(file128.ETag, fileCopy.ETag)
	expect.Equal(lastModified, fileCopy.LastModified)

	// Copy a non-existent file
	_, err = mb.CopyFile(ctx, file128.BucketName, "does-not-exist", "stack.png")
	expect.Error(err)
	expect.True(errors.Is(err, ErrFileNotFound))

	// List all the files in the bucket
	files, err := mb.ListAllFiles(ctx)
	expect.NoError(err)
	expect.Equal(2, len(files))
	expect.Greater(files[1].FileName, files[0].FileName)

	// Delete the files
	for _, file := range files {
		expect.NoError(mb.DeleteFile(ctx, file.FileName))
	}

	// Delete a non-existent file (no error expected)
	expect.NoError(mb.DeleteFile(ctx, "does-not-exist"))
}
