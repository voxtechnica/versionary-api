package bucket

import (
	"sort"
	"time"
)

// MemFile contains information for a file stored in a MemBucket.
type MemFile struct {
	FileName      string    `json:"fileName"`
	ContentType   string    `json:"contentType,omitempty"`
	ContentLength int64     `json:"contentLength,omitempty"`
	ETag          string    `json:"etag,omitempty"`
	LastModified  time.Time `json:"lastModified,omitempty"`
	Blob          []byte    `json:"-"`
}

// FileInfo converts a MemFile to a FileInfo.
func (mf MemFile) FileInfo(bucketName string) FileInfo {
	return FileInfo{
		BucketName:    bucketName,
		FileName:      mf.FileName,
		ContentType:   mf.ContentType,
		ContentLength: mf.ContentLength,
		ETag:          mf.ETag,
		LastModified:  mf.LastModified,
	}
}

// MemFileSet provides an in-memory data structure for storing a set of MemFiles,
// used for lightweight testing purposes.
type MemFileSet map[string]MemFile

// AddFile adds a file to the MemFileSet.
func (mfs *MemFileSet) AddFile(mf MemFile) {
	if *mfs == nil {
		*mfs = make(MemFileSet)
	}
	(*mfs)[mf.FileName] = mf
}

// FileExists returns true if the file exists in the MemFileSet.
func (mfs *MemFileSet) FileExists(fileName string) bool {
	if *mfs == nil {
		*mfs = make(MemFileSet)
		return false
	}
	_, ok := (*mfs)[fileName]
	return ok
}

// GetFile returns a file from the MemFileSet.
func (mfs *MemFileSet) GetFile(fileName string) (MemFile, bool) {
	if *mfs == nil {
		*mfs = make(MemFileSet)
		return MemFile{}, false
	}
	mf, ok := (*mfs)[fileName]
	return mf, ok
}

// DeleteFile deletes a file from the MemFileSet.
func (mfs *MemFileSet) DeleteFile(fileName string) {
	if *mfs == nil {
		*mfs = make(MemFileSet)
		return
	}
	delete(*mfs, fileName)
}

// ListFiles returns a complete list of files from the MemFileSet, sorted by FileName.
func (mfs *MemFileSet) ListFiles() []MemFile {
	if *mfs == nil {
		*mfs = make(MemFileSet)
		return nil
	}
	var files []MemFile
	for _, mf := range *mfs {
		files = append(files, mf)
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].FileName < files[j].FileName
	})
	return files
}
