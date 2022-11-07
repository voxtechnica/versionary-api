package bucket

import (
	"bytes"
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"time"
)

// MemBucket is a in-memory bucket implementation used for managing a collection of files.
// It implements the BucketReader and BucketWriter interfaces, and is intended to be used
// for testing purposes only.
type MemBucket struct {
	FileSet    *MemFileSet
	EntityType string
	BucketName string
	Public     bool
}

// NewMemBucket returns a new MemBucket.
func NewMemBucket(b Bucket) MemBucket {
	return MemBucket{
		FileSet:    &MemFileSet{},
		EntityType: b.EntityType,
		BucketName: b.BucketName,
		Public:     b.Public,
	}
}

// IsValid returns true if the bucket is configured properly.
func (mb MemBucket) IsValid() bool {
	return mb.FileSet != nil && mb.EntityType != "" && mb.BucketName != ""
}

// BucketExists returns true if the bucket exists.
func (mb MemBucket) BucketExists() bool {
	return mb.FileSet != nil
}

// FileExists returns true if the file exists in the bucket.
func (mb MemBucket) FileExists(ctx context.Context, fileName string) (bool, error) {
	return mb.FileSet.FileExists(fileName), nil
}

// FileInfo returns information about a file in the bucket.
func (mb MemBucket) FileInfo(ctx context.Context, fileName string) (FileInfo, error) {
	mf, ok := mb.FileSet.GetFile(fileName)
	if !ok {
		return FileInfo{}, ErrFileNotFound
	}
	return FileInfo{
		BucketName:    mb.BucketName,
		FileName:      mf.FileName,
		ContentType:   mf.ContentType,
		ContentLength: mf.ContentLength,
		ETag:          mf.ETag,
		LastModified:  mf.LastModified,
	}, nil
}

// UploadFile uploads a file to the bucket.
func (mb MemBucket) UploadFile(ctx context.Context, info FileInfo, file io.Reader) (FileInfo, error) {
	blob, err := io.ReadAll(file)
	if err != nil {
		return info, fmt.Errorf("upload %s to %s: %w", info.FileName, mb.BucketName, err)
	}
	mf := MemFile{
		FileName:      info.FileName,
		ContentType:   info.ContentType,
		ContentLength: int64(len(blob)),
		ETag:          fmt.Sprintf("%x", md5.Sum(blob)),
		LastModified:  time.Now(),
		Blob:          blob,
	}
	mb.FileSet.AddFile(mf)
	return mf.FileInfo(mb.BucketName), nil
}

// DownloadFile downloads a file from the bucket.
func (mb MemBucket) DownloadFile(ctx context.Context, fileName string) (FileInfo, io.ReadCloser, error) {
	mf, ok := mb.FileSet.GetFile(fileName)
	if !ok {
		return FileInfo{BucketName: mb.BucketName, FileName: fileName}, nil, ErrFileNotFound
	}
	info := mf.FileInfo(mb.BucketName)
	rc := io.NopCloser(bytes.NewReader(mf.Blob))
	return info, rc, nil
}

// GetUploadURL returns a pre-signed URL for uploading a file to the bucket.
// MemBucket does not support pre-signed URLs, so this function returns localhost URL that will not work.
func (mb MemBucket) GetUploadURL(ctx context.Context, fileName string, contentType string, expires time.Duration) (PreSignedURL, error) {
	psu := PreSignedURL{
		BucketName:  mb.BucketName,
		FileName:    fileName,
		ContentType: contentType,
		ExpiresAt:   time.Now().Add(expires),
	}
	psu.Method = "PUT"
	psu.Host = "localhost"
	psu.URL = fmt.Sprintf("http://%s/%s/%s", psu.Host, psu.BucketName, psu.FileName)
	return psu, nil
}

// GetDownloadURL returns a pre-signed URL for downloading a file from the bucket.
// MemBucket does not support pre-signed URLs, so this function returns localhost URL that will not work.
func (mb MemBucket) GetDownloadURL(ctx context.Context, fileName string, expires time.Duration) (PreSignedURL, error) {
	psu := PreSignedURL{
		BucketName: mb.BucketName,
		FileName:   fileName,
		ExpiresAt:  time.Now().Add(expires),
	}
	mf, ok := mb.FileSet.GetFile(fileName)
	if !ok {
		return PreSignedURL{}, ErrFileNotFound
	}
	psu.ContentType = mf.ContentType
	psu.ContentLength = mf.ContentLength
	psu.ETag = mf.ETag
	psu.Method = "GET"
	psu.Host = "localhost"
	psu.URL = fmt.Sprintf("http://%s/%s/%s", psu.Host, psu.BucketName, psu.FileName)
	return psu, nil
}

// CopyFile copies a file from one bucket to another. Note that for a MemBucket,
// this function only copies files in the same bucket.
func (mb MemBucket) CopyFile(ctx context.Context, fromBucketName string, fromFileName string, toFileName string) (FileInfo, error) {
	mf, ok := mb.FileSet.GetFile(fromFileName)
	if !ok {
		return FileInfo{BucketName: mb.BucketName, FileName: fromFileName}, ErrFileNotFound
	}
	mf.FileName = toFileName
	mb.FileSet.AddFile(mf)
	return mf.FileInfo(mb.BucketName), nil
}

// DeleteFile deletes a file from the bucket.
func (mb MemBucket) DeleteFile(ctx context.Context, fileName string) error {
	mb.FileSet.DeleteFile(fileName)
	return nil
}

// DeleteFiles deletes the specified files from the bucket.
func (mb MemBucket) DeleteFiles(ctx context.Context, fileNames []string) error {
	// Maximum 1000 files can be deleted at once
	if len(fileNames) > 1000 {
		return ErrTooManyFiles
	}
	// Delete files
	for _, fileName := range fileNames {
		mb.FileSet.DeleteFile(fileName)
	}
	return nil
}

// ListAllFiles returns a list of files in the bucket. This may be a large list!
func (mb MemBucket) ListAllFiles(ctx context.Context) ([]FileInfo, error) {
	var files []FileInfo
	for _, mf := range *mb.FileSet {
		files = append(files, mf.FileInfo(mb.BucketName))
	}
	return files, nil
}
