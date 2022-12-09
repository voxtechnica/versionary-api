package image

import (
	"bytes"
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"image"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	b "versionary-api/pkg/bucket"
	"versionary-api/pkg/util"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/azr/phash"
	"github.com/voxtechnica/tuid-go"
	v "github.com/voxtechnica/versionary"
)

// httpClient is the HTTP client used to fetch images from URLs.
var httpClient = http.Client{Timeout: 30 * time.Second}

//==============================================================================
// Image Table
//==============================================================================

// rowImages is a TableRow definition for Image versions.
var rowImages = v.TableRow[Image]{
	RowName:      "images_version",
	PartKeyName:  "id",
	PartKeyValue: func(i Image) string { return i.ID },
	PartKeyLabel: func(i Image) string { return i.Label() },
	SortKeyName:  "version_id",
	SortKeyValue: func(i Image) string { return i.VersionID },
	JsonValue:    func(i Image) []byte { return i.CompressedJSON() },
}

// rowImagesStatus is a TableRow definition for searching/browsing Images by status.
var rowImagesStatus = v.TableRow[Image]{
	RowName:      "images_status",
	PartKeyName:  "status",
	PartKeyValue: func(i Image) string { return i.Status.String() },
	SortKeyName:  "id",
	SortKeyValue: func(i Image) string { return i.ID },
	JsonValue:    func(i Image) []byte { return i.CompressedJSON() },
}

// rowImagesTag is a TableRow definition for searching/browsing Images by tag.
var rowImagesTag = v.TableRow[Image]{
	RowName:       "images_tag",
	PartKeyName:   "tag",
	PartKeyValues: func(i Image) []string { return i.Tags },
	SortKeyName:   "id",
	SortKeyValue:  func(i Image) string { return i.ID },
	JsonValue:     func(i Image) []byte { return i.CompressedJSON() },
}

// rowImageHashes is a TableRow definition for Image IDs and associated perceptual hashes.
// This is used for identifying similar images.
var rowImageHashes = v.TableRow[Image]{
	RowName:      "image_hashes",
	PartKeyName:  "type",
	PartKeyValue: func(i Image) string { return i.Type() },
	SortKeyName:  "id",
	SortKeyValue: func(i Image) string { return i.ID },
	TextValue:    func(i Image) string { return i.PHash },
}

// NewTable instantiates a new DynamoDB Image table.
func NewTable(dbClient *dynamodb.Client, env string) v.Table[Image] {
	if env == "" {
		env = "dev"
	}
	return v.Table[Image]{
		Client:     dbClient,
		EntityType: "Image",
		TableName:  "images" + "_" + env,
		TTL:        false,
		EntityRow:  rowImages,
		IndexRows: map[string]v.TableRow[Image]{
			rowImagesStatus.RowName: rowImagesStatus,
			rowImagesTag.RowName:    rowImagesTag,
			rowImageHashes.RowName:  rowImageHashes,
		},
	}
}

// NewMemTable creates an in-memory Image table for testing purposes.
func NewMemTable(table v.Table[Image]) v.MemTable[Image] {
	return v.NewMemTable(table)
}

//==============================================================================
// Image Bucket
//==============================================================================

// NewBucket instantiates a new S3 Image bucket.
func NewBucket(s3Client *s3.Client, env string) b.Bucket {
	if env == "" {
		env = "dev"
	}
	return b.Bucket{
		Client:     s3Client,
		EntityType: "Image",
		BucketName: "versionary-images-" + env,
		Public:     true,
	}
}

// NewMemBucket creates an in-memory Image bucket for testing purposes.
func NewMemBucket(bucket b.Bucket) b.MemBucket {
	return b.NewMemBucket(bucket)
}

//==============================================================================
// Image Service
//==============================================================================

// Service is a service for managing Images.
type Service struct {
	EntityType string
	Bucket     b.BucketReadWriter
	Table      v.TableReadWriter[Image]
}

// NewService instantiates a new Image service, backed by DynamoDB and S3.
func NewService(dbClient *dynamodb.Client, s3Client *s3.Client, env string) Service {
	bucket := NewBucket(s3Client, env)
	table := NewTable(dbClient, env)
	return Service{
		EntityType: table.EntityType,
		Bucket:     bucket,
		Table:      table,
	}
}

// NewMockService instantiates a new Image service with in-memory storage for testing purposes.
func NewMockService(env string) Service {
	bucket := NewMemBucket(NewBucket(nil, env))
	table := NewMemTable(NewTable(nil, env))
	return Service{
		EntityType: table.EntityType,
		Bucket:     bucket,
		Table:      table,
	}
}

// FetchSourceURI fetches the source image from the given URI and returns the image blob.
func (s Service) FetchSourceURI(ctx context.Context, uri string) ([]byte, error) {
	// A source URI is required.
	if uri == "" {
		return nil, fmt.Errorf("error fetching image source: no URI provided")
	}
	// The source URI must be a valid URL.
	_, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("error fetching image source: invalid URI %s: %w", uri, err)
	}
	// Fetch the image from the source URI.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, fmt.Errorf("error fetching image source %s: %w", uri, err)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching image source %s: %w", uri, err)
	}
	defer func(r *http.Response) { _ = r.Body.Close() }(resp)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error fetching image source %s: %s", uri, resp.Status)
	}
	// Read response body into byte buffer.
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error fetching image source %s: %w", uri, err)
	}
	return buf.Bytes(), nil
}

// FetchSourceFile fetches the source image from the given file path and returns the image blob.
func (s Service) FetchSourceFile(path string) ([]byte, error) {
	// A source file path is required.
	if path == "" {
		return nil, fmt.Errorf("error fetching image source: no file path provided")
	}
	// The source file must exist.
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("error fetching image source: file %s does not exist", path)
	}
	// Read the file into a byte slice.
	blob, err := os.ReadFile(path)
	if err != nil {
		return blob, fmt.Errorf("error fetching image source %s: %w", path, err)
	}
	return blob, nil
}

// FetchImageFile fetches the image file from the S3 bucket and returns the image blob.
func (s Service) FetchImageFile(ctx context.Context, fileName string) ([]byte, error) {
	// An image file name is required.
	if fileName == "" {
		return nil, fmt.Errorf("error fetching image: no file name provided")
	}
	// Fetch the image from the S3 bucket.
	_, rc, err := s.Bucket.DownloadFile(ctx, fileName)
	if err != nil {
		return nil, fmt.Errorf("error fetching image %s: %w", fileName, err)
	}
	defer func(rc io.ReadCloser) { _ = rc.Close() }(rc)
	// Read the image into byte buffer.
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(rc)
	if err != nil {
		return nil, fmt.Errorf("error fetching image %s: %w", fileName, err)
	}
	return buf.Bytes(), nil
}

// Analyze analyzes the given image blob and returns an updated Image struct.
func (s Service) Analyze(i Image, blob []byte) (Image, error) {
	// Decode the image
	img, format, err := image.Decode(bytes.NewReader(blob))
	if err != nil {
		return i, fmt.Errorf("analyze Image %s: %w", i.ID, err)
	}
	// Gather available image metadata
	i.MediaType = MediaType("image/" + format)
	i.FileName = i.ID + i.FileExt()
	i.FileSize = int64(len(blob))
	i.MD5Hash = fmt.Sprintf("%x", md5.Sum(blob))
	i.PHash = strconv.FormatUint(phash.DTC(img), 36)
	i.Width = img.Bounds().Dx()
	i.Height = img.Bounds().Dy()
	i.AspectRatio = float64(i.Width) / float64(i.Height)
	return i, nil
}

//------------------------------------------------------------------------------
// Image Versions
//------------------------------------------------------------------------------

// Create an Image in the Image table.
func (s Service) Create(ctx context.Context, i Image) (Image, []string, error) {
	// Initialize and validate the Image.
	t := tuid.NewID()
	at, _ := t.Time()
	i.ID = t.String()
	i.CreatedAt = at
	i.VersionID = t.String()
	i.UpdatedAt = at
	i.FileName = i.ID + i.FileExt()
	if i.Status == "" {
		i.Status = PENDING
	}
	problems := i.Validate()
	if len(problems) > 0 {
		i.Status = ERROR
		return i, problems, fmt.Errorf("error creating %s %s: invalid field(s): %s", s.EntityType, i.ID, strings.Join(problems, ", "))
	}
	// Fetch the image source, if available.
	var blob []byte
	var err error
	if i.SourceURI != "" {
		blob, err = s.FetchSourceURI(ctx, i.SourceURI)
		if err != nil {
			i.Status = ERROR
			return i, problems, fmt.Errorf("error creating %s %s: %w", s.EntityType, i.ID, err)
		}
	} else if i.SourceFileName != "" {
		blob, err = s.FetchSourceFile(i.SourceFileName)
		if err != nil {
			i.Status = ERROR
			return i, problems, fmt.Errorf("error creating %s %s: %w", s.EntityType, i.ID, err)
		}
	}
	if len(blob) > 0 {
		// Analyze the image, if available.
		i, err = s.Analyze(i, blob)
		if err != nil {
			i.Status = ERROR
			return i, problems, fmt.Errorf("error analyzing %s %s: %w", s.EntityType, i.ID, err)
		}
		// Upload the image to the S3 bucket, if available.
		_, err = s.Bucket.UploadFile(ctx, i.FileInfo(), bytes.NewReader(blob))
		if err != nil {
			i.Status = ERROR
			return i, problems, fmt.Errorf("error uploading %s %s %s: %w", s.EntityType, i.ID, i.Label(), err)
		}
		i.Status = COMPLETE
	}
	// Create the image in the database.
	err = s.Table.WriteEntity(ctx, i)
	if err != nil {
		i.Status = ERROR
		return i, problems, fmt.Errorf("error creating %s %s %s: %w", s.EntityType, i.ID, i.Label(), err)
	}
	return i, problems, nil
}

// Update an Image in the Image table. If a previous version does not exist, the Image is created.
// If the file size is zero, the image is fetched and analyzed, and uploaded to S3 if needed.
// Update can be used to analyze an image that was uploaded to S3 using a pre-signed URL.
func (s Service) Update(ctx context.Context, i Image) (Image, []string, error) {
	// Initialize and validate the Image.
	t := tuid.NewID()
	at, _ := t.Time()
	i.VersionID = t.String()
	i.UpdatedAt = at
	problems := i.Validate()
	if len(problems) > 0 {
		return i, problems, fmt.Errorf("error updating %s %s: invalid field(s): %s", s.EntityType, i.ID, strings.Join(problems, ", "))
	}
	// Fetch, analyze, and upload the source image, if needed.
	if i.FileSize == 0 {
		var blob []byte
		var err error
		exists, _ := s.Bucket.FileExists(ctx, i.FileName)
		if exists {
			// Fetch the image blob from S3 for analysis.
			blob, err = s.FetchImageFile(ctx, i.FileName)
			if err != nil {
				i.Status = ERROR
				return i, problems, fmt.Errorf("error updating %s %s: %w", s.EntityType, i.ID, err)
			}
		} else {
			// Fetch the image source, if available.
			if i.SourceURI != "" {
				blob, err = s.FetchSourceURI(ctx, i.SourceURI)
				if err != nil {
					i.Status = ERROR
					return i, problems, fmt.Errorf("error updating %s %s: %w", s.EntityType, i.ID, err)
				}
			} else if i.SourceFileName != "" {
				blob, err = s.FetchSourceFile(i.SourceFileName)
				if err != nil {
					i.Status = ERROR
					return i, problems, fmt.Errorf("error updating %s %s: %w", s.EntityType, i.ID, err)
				}
			}
		}
		// Analyze the image, if available.
		if len(blob) > 0 {
			i, err = s.Analyze(i, blob)
			if err != nil {
				i.Status = ERROR
				return i, problems, fmt.Errorf("error analyzing %s %s: %w", s.EntityType, i.ID, err)
			}
			// Upload the image to the S3 bucket, if needed.
			if !exists {
				_, err = s.Bucket.UploadFile(ctx, i.FileInfo(), bytes.NewReader(blob))
				if err != nil {
					i.Status = ERROR
					return i, problems, fmt.Errorf("error uploading %s %s %s: %w", s.EntityType, i.ID, i.Label(), err)
				}
			}
			i.Status = COMPLETE
		}
	}
	// Update the image in the database.
	return i, problems, s.Table.UpdateEntity(ctx, i)
}

// Write an Image to the Image table. This method assumes that the Image has all the required fields.
// It would most likely be used for "refreshing" the index rows in the Image table.
func (s Service) Write(ctx context.Context, i Image) (Image, error) {
	return i, s.Table.WriteEntity(ctx, i)
}

// UploadURL returns an S3 upload URL for the specified Image.
func (s Service) UploadURL(ctx context.Context, id string, expires time.Duration) (b.PreSignedURL, error) {
	// Read the Image from the Image table to get the file name and verify its existence.
	i, err := s.Read(ctx, id)
	if err != nil {
		return b.PreSignedURL{}, fmt.Errorf("upload URL for Image %s: %w", id, err)
	}
	// Maximum expiration time is 6 hours (AWS S3 limit)
	if expires > 6*time.Hour {
		expires = 6 * time.Hour
	} else if expires <= 0 {
		expires = 1 * time.Hour
	}
	return s.Bucket.GetUploadURL(ctx, i.FileName, i.MediaType.String(), expires)
}

// DownloadURL returns the S3 download URL for the specified Image.
func (s Service) DownloadURL(ctx context.Context, id string, expires time.Duration) (b.PreSignedURL, error) {
	// Read the Image from the Image table to get the file name and verify its existence.
	i, err := s.Read(ctx, id)
	if err != nil {
		return b.PreSignedURL{}, fmt.Errorf("download URL for Image %s: %w", id, err)
	}
	// Maximum expiration time is 6 hours (AWS S3 limit)
	if expires > 6*time.Hour {
		expires = 6 * time.Hour
	} else if expires <= 0 {
		expires = 1 * time.Hour
	}
	return s.Bucket.GetDownloadURL(ctx, i.FileName, expires)
}

// Delete an Image from the Image table and bucket. The deleted Image is returned.
func (s Service) Delete(ctx context.Context, id string) (Image, error) {
	i, err := s.Table.DeleteEntityWithID(ctx, id)
	if err != nil {
		return i, err
	}
	return i, s.Bucket.DeleteFile(ctx, i.FileName)
}

// DeleteVersion deletes a specific version of an Image from the Image table. The deleted Image is returned.
// Note: DeleteVersion does not delete the image file from the S3 bucket.
func (s Service) DeleteVersion(ctx context.Context, id string, versionid string) (Image, error) {
	return s.Table.DeleteEntityVersionWithID(ctx, id, versionid)
}

// Exists checks if an Image exists in the Image table.
func (s Service) Exists(ctx context.Context, id string) bool {
	return s.Table.EntityExists(ctx, id)
}

// FileInfo returns the S3 file info for an Image.
func (s Service) FileInfo(ctx context.Context, id string) (b.FileInfo, error) {
	i, err := s.Read(ctx, id)
	if err != nil {
		return b.FileInfo{}, err
	}
	return s.Bucket.FileInfo(ctx, i.FileName)
}

// FileBlob returns a byte slice containing the image file blob (binary large object).
// This is a "best effort" method. If an error occurs, the byte slice will be empty.
func (s Service) FileBlob(ctx context.Context, i Image) []byte {
	// First, try looking for the image file in S3.
	if i.FileName != "" {
		_, rc, err := s.Bucket.DownloadFile(ctx, i.FileName)
		if err == nil {
			defer func(rc io.ReadCloser) { _ = rc.Close() }(rc)
			buf := new(bytes.Buffer)
			_, err = buf.ReadFrom(rc)
			if err == nil {
				return buf.Bytes()
			}
		}
	}
	// Next, try looking for the image at a remote source URL.
	if i.SourceURI != "" {
		httpClient := http.Client{Timeout: 30 * time.Second}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, i.SourceURI, nil)
		if err == nil {
			resp, err := httpClient.Do(req)
			if err == nil {
				defer func(r *http.Response) { _ = r.Body.Close() }(resp)
				if resp.StatusCode == http.StatusOK {
					buf := new(bytes.Buffer)
					_, err = buf.ReadFrom(resp.Body)
					if err == nil {
						return buf.Bytes()
					}
				}
			}
		}
	}
	// Finally, try looking for the image file in the local file system.
	if i.SourceFileName != "" {
		blob, err := os.ReadFile(i.SourceFileName)
		if err == nil {
			return blob
		}
	}
	// If there was a problem finding the file, return an empty byte slice.
	return []byte{}
}

// Read a specified Image from the Image table.
func (s Service) Read(ctx context.Context, id string) (Image, error) {
	return s.Table.ReadEntity(ctx, id)
}

// ReadAsJSON gets a specified Image from the Image table, serialized as JSON.
func (s Service) ReadAsJSON(ctx context.Context, id string) ([]byte, error) {
	return s.Table.ReadEntityAsJSON(ctx, id)
}

// VersionExists checks if a specified Image version exists in the Image table.
func (s Service) VersionExists(ctx context.Context, id, versionID string) bool {
	return s.Table.EntityVersionExists(ctx, id, versionID)
}

// ReadVersion gets a specified Image version from the Image table.
func (s Service) ReadVersion(ctx context.Context, id, versionID string) (Image, error) {
	return s.Table.ReadEntityVersion(ctx, id, versionID)
}

// ReadVersionAsJSON gets a specified Image version from the Image table, serialized as JSON.
func (s Service) ReadVersionAsJSON(ctx context.Context, id, versionID string) ([]byte, error) {
	return s.Table.ReadEntityVersionAsJSON(ctx, id, versionID)
}

// ReadVersions returns paginated versions of the specified Image.
// Sorting is chronological (or reverse). The offset is the last ID returned in a previous request.
func (s Service) ReadVersions(ctx context.Context, id string, reverse bool, limit int, offset string) ([]Image, error) {
	return s.Table.ReadEntityVersions(ctx, id, reverse, limit, offset)
}

// ReadVersionsAsJSON returns paginated versions of the specified Image, serialized as JSON.
// Sorting is chronological (or reverse). The offset is the last ID returned in a previous request.
func (s Service) ReadVersionsAsJSON(ctx context.Context, id string, reverse bool, limit int, offset string) ([]byte, error) {
	return s.Table.ReadEntityVersionsAsJSON(ctx, id, reverse, limit, offset)
}

// ReadAllVersions returns all versions of the specified Image in chronological order.
// Caution: this may be a LOT of data!
func (s Service) ReadAllVersions(ctx context.Context, id string) ([]Image, error) {
	return s.Table.ReadAllEntityVersions(ctx, id)
}

// ReadAllVersionsAsJSON returns all versions of the specified Image, serialized as JSON.
// Caution: this may be a LOT of data!
func (s Service) ReadAllVersionsAsJSON(ctx context.Context, id string) ([]byte, error) {
	return s.Table.ReadAllEntityVersionsAsJSON(ctx, id)
}

// ReadImageIDs returns a paginated list of Image IDs in the Image table.
// Sorting is chronological (or reverse). The offset is the last ID returned in a previous request.
func (s Service) ReadImageIDs(ctx context.Context, reverse bool, limit int, offset string) ([]string, error) {
	return s.Table.ReadEntityIDs(ctx, reverse, limit, offset)
}

// ReadImages returns a paginated list of Images in the Image table.
// Sorting is chronological (or reverse). The offset is the last ID returned in a previous request.
// Note that this is a best-effort attempt to return the requested Images, retrieved individually, in parallel.
// It is probably not the best way to page through a large Image table.
func (s Service) ReadImages(ctx context.Context, reverse bool, limit int, offset string) []Image {
	ids, err := s.Table.ReadEntityIDs(ctx, reverse, limit, offset)
	if err != nil {
		return []Image{}
	}
	return s.Table.ReadEntities(ctx, ids)
}

// ReadImageMap returns a map of Images in the Image table, keyed by ID.
// Note that this is a best-effort attempt to return the requested Images, retrieved individually, in parallel.
func (s Service) ReadImageMap(ctx context.Context, ids []string) map[string]Image {
	m := make(map[string]Image, len(ids))
	var batches [][]string
	maxBatchSize := 10 // concurrent request limit
	batches = v.Batch(ids, maxBatchSize)
	for _, batch := range batches {
		batchSize := len(batch)
		results := make(chan struct {
			ID     string
			entity Image
		}, batchSize)
		var wg sync.WaitGroup
		wg.Add(batchSize)
		for _, entityID := range batch {
			go func(id string) {
				defer wg.Done()
				entity, err := s.Table.ReadEntity(ctx, id)
				if err == nil {
					results <- struct {
						ID     string
						entity Image
					}{ID: id, entity: entity}
				}
			}(entityID)
		}
		wg.Wait()
		close(results)
		for r := range results {
			m[r.ID] = r.entity
		}
	}
	return m
}

//------------------------------------------------------------------------------
// Image Labels
//------------------------------------------------------------------------------

// ReadImageLabels returns a paginated list of Image IDs with associated labels.
func (s Service) ReadImageLabels(ctx context.Context, reverse bool, limit int, offset string) ([]v.TextValue, error) {
	return s.Table.ReadEntityLabels(ctx, reverse, limit, offset)
}

// ReadAllImageLabels returns all Image labels in the Image table, optionally
// sorted by ascending key (Image ID) or value (Label).
func (s Service) ReadAllImageLabels(ctx context.Context, sortByValue bool) ([]v.TextValue, error) {
	return s.Table.ReadAllEntityLabels(ctx, sortByValue)
}

// FilterImageLabels returns a filtered list of Image IDs with associated labels.
// The list contains only those labels that match the provided filter query.
// The contains query string is split into words, and the words are compared with the Image label.
// If anyMatch is true, then an Image label is included if any of the words are found (an OR filter).
// If anyMatch is false, then the Image label must contain all the words in the query string (an AND filter).
// The filtered results are sorted alphabetically by label, not by ID.
func (s Service) FilterImageLabels(ctx context.Context, contains string, anyMatch bool) ([]v.TextValue, error) {
	filter, err := util.ContainsFilter(contains, anyMatch)
	if err != nil {
		return []v.TextValue{}, err
	}
	return s.Table.FilterEntityLabels(ctx, filter)
}

//------------------------------------------------------------------------------
// Images by Tag
//------------------------------------------------------------------------------

// ReadAllTags returns a complete, alphabetical Tag list for which there are Images in the Image table.
func (s Service) ReadAllTags(ctx context.Context) ([]string, error) {
	return s.Table.ReadAllPartKeyValues(ctx, rowImagesTag)
}

// ReadImagesByTag returns paginated Images by Tag. Sorting is chronological (or reverse).
// The offset is the ID of the last Image returned in a previous request.
func (s Service) ReadImagesByTag(ctx context.Context, tag string, reverse bool, limit int, offset string) ([]Image, error) {
	return s.Table.ReadEntitiesFromRow(ctx, rowImagesTag, tag, reverse, limit, offset)
}

// ReadImagesByTagAsJSON returns paginated JSON Images by Tag. Sorting is chronological (or reverse).
// The offset is the ID of the last Image returned in a previous request.
func (s Service) ReadImagesByTagAsJSON(ctx context.Context, tag string, reverse bool, limit int, offset string) ([]byte, error) {
	return s.Table.ReadEntitiesFromRowAsJSON(ctx, rowImagesTag, tag, reverse, limit, offset)
}

// ReadAllImagesByTag returns the complete list of Images, sorted chronologically by CreatedAt timestamp.
// Caution: this may be a LOT of data!
func (s Service) ReadAllImagesByTag(ctx context.Context, tag string) ([]Image, error) {
	return s.Table.ReadAllEntitiesFromRow(ctx, rowImagesTag, tag)
}

// ReadAllImagesByTagAsJSON returns the complete list of Images, serialized as JSON.
// Caution: this may be a LOT of data!
func (s Service) ReadAllImagesByTagAsJSON(ctx context.Context, tag string) ([]byte, error) {
	return s.Table.ReadAllEntitiesFromRowAsJSON(ctx, rowImagesTag, tag)
}

//------------------------------------------------------------------------------
// Images by Status
//------------------------------------------------------------------------------

// ReadAllStatuses returns a complete, alphabetical Status list for which there are Images in the Image table.
func (s Service) ReadAllStatuses(ctx context.Context) ([]string, error) {
	return s.Table.ReadAllPartKeyValues(ctx, rowImagesStatus)
}

// ReadImagesByStatus returns paginated Images by Status. Sorting is chronological (or reverse).
// The offset is the ID of the last Image returned in a previous request.
func (s Service) ReadImagesByStatus(ctx context.Context, status string, reverse bool, limit int, offset string) ([]Image, error) {
	return s.Table.ReadEntitiesFromRow(ctx, rowImagesStatus, status, reverse, limit, offset)
}

// ReadImagesByStatusAsJSON returns paginated JSON Images by Status. Sorting is chronological (or reverse).
// The offset is the ID of the last Image returned in a previous request.
func (s Service) ReadImagesByStatusAsJSON(ctx context.Context, status string, reverse bool, limit int, offset string) ([]byte, error) {
	return s.Table.ReadEntitiesFromRowAsJSON(ctx, rowImagesStatus, status, reverse, limit, offset)
}

// ReadAllImagesByStatus returns the complete list of Images, sorted chronologically by CreatedAt timestamp.
// Caution: this may be a LOT of data!
func (s Service) ReadAllImagesByStatus(ctx context.Context, status string) ([]Image, error) {
	return s.Table.ReadAllEntitiesFromRow(ctx, rowImagesStatus, status)
}

// ReadAllImagesByStatusAsJSON returns the complete list of Images, serialized as JSON.
// Caution: this may be a LOT of data!
func (s Service) ReadAllImagesByStatusAsJSON(ctx context.Context, status string) ([]byte, error) {
	return s.Table.ReadAllEntitiesFromRowAsJSON(ctx, rowImagesStatus, status)
}

//------------------------------------------------------------------------------
// Image Hashes
//------------------------------------------------------------------------------

// ReadImageHashes returns a paginated list of Image IDs with associated perceptual hash values.
func (s Service) ReadImageHashes(ctx context.Context, reverse bool, limit int, offset string) ([]v.TextValue, error) {
	return s.Table.ReadTextValues(ctx, rowImageHashes, s.EntityType, reverse, limit, offset)
}

// ReadAllImageHashes returns all Image perceptual hash values in the Image table.
func (s Service) ReadAllImageHashes(ctx context.Context) ([]v.TextValue, error) {
	return s.Table.ReadAllTextValues(ctx, rowImageHashes, s.EntityType, false)
}

// FindSimilarImages returns a Distance slice, ordered in increasing distance, of
// images having perceptual hash values within the specified distance of the query image.
// maxDistance must be between 0 and 64 (inclusive). 16 might be a reasonable value.
// For performance reasons (Image data are fetched in parallel), the maximum limit is 100.
func (s Service) FindSimilarImages(ctx context.Context, pHash string, maxDistance int, limit int) ([]Distance, error) {
	// Validate the provided parameters.
	if pHash == "" {
		return nil, errors.New("find similar images: empty perceptual hash")
	}
	if maxDistance < 0 || maxDistance > 64 {
		return nil, fmt.Errorf("find similar images: maxDistance %d must be between 0 and 64 (inclusive)", maxDistance)
	}
	if limit < 1 || limit > 100 {
		return nil, fmt.Errorf("find similar images: limit %d must be between 1 and 100 (inclusive)", limit)
	}
	// Read all the perceptual hash values from the Image table.
	hashes, err := s.ReadAllImageHashes(ctx)
	if err != nil {
		return nil, fmt.Errorf("find similar images: %w", err)
	}
	// Calculate perceptual distances, keeping only those within the specified range.
	var distances []struct {
		id   string
		dist int
	}
	for _, hash := range hashes {
		d, err := PHashDistance(pHash, hash.Value)
		if err == nil && d <= maxDistance {
			distances = append(distances, struct {
				id   string
				dist int
			}{id: hash.Key, dist: d})
		}
	}
	// Sort the results by distance, then by ID.
	sort.Slice(distances, func(i, j int) bool {
		if distances[i].dist == distances[j].dist {
			return distances[i].id < distances[j].id
		}
		return distances[i].dist < distances[j].dist
	})
	// Truncate the results to the specified limit.
	if len(distances) > limit {
		distances = distances[:limit]
	}
	// Fetch the similar images in parallel.
	ids := make([]string, len(distances))
	for i := 0; i < len(distances); i++ {
		ids[i] = distances[i].id
	}
	images := s.ReadImageMap(ctx, ids)
	// Generate Distance results.
	results := make([]Distance, len(distances))
	for i := 0; i < len(distances); i++ {
		results[i].ID = distances[i].id
		results[i].Distance = distances[i].dist
		img, ok := images[results[i].ID]
		if ok {
			results[i].Populate(img)
		}
	}
	return results, nil
}

// PHashDistance returns the number of bits that are different between two perceptual hash values.
func PHashDistance(h1, h2 string) (int, error) {
	if h1 == "" || h2 == "" {
		return 100, errors.New("image distance: missing pHash value")
	}
	if h1 == h2 {
		return 0, nil
	}
	pHash1, err := strconv.ParseUint(h1, 36, 64)
	if err != nil {
		return 100, fmt.Errorf("image distance: error parsing pHash %s: %w", h1, err)
	}
	pHash2, err := strconv.ParseUint(h2, 36, 64)
	if err != nil {
		return 100, fmt.Errorf("image distance: error parsing pHash %s: %w", h2, err)
	}
	return phash.Distance(pHash1, pHash2), nil
}
