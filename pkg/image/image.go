package image

import (
	"fmt"
	"strings"
	"time"

	b "versionary-api/pkg/bucket"

	"github.com/voxtechnica/tuid-go"
	v "github.com/voxtechnica/versionary"
)

// Image provides metadata about an image file in the S3 object store.
type Image struct {
	ID             string    `json:"id"`
	CreatedAt      time.Time `json:"createdAt"`
	VersionID      string    `json:"versionID"`
	UpdatedAt      time.Time `json:"updatedAt"`
	Title          string    `json:"title"`                    // Image title, suitable for display as a caption
	AltText        string    `json:"altText"`                  // Image description for accessibility
	SourceURI      string    `json:"sourceURI,omitempty"`      // SourceURI is the URI of the original image
	SourceFileName string    `json:"sourceFileName,omitempty"` // SourceFileName is the name of the original image
	MediaType      MediaType `json:"mediaType"`                // Media Type (image/jpeg, image/webp, image/png, image/gif)
	FileName       string    `json:"fileName"`                 // S3 File name (ID + file extension)
	FileSize       int64     `json:"fileSize"`                 // File size in bytes
	MD5Hash        string    `json:"md5Hash"`                  // MD5 hash of the image
	PHash          string    `json:"pHash,omitempty"`          // PHash is the DTC perceptual hash of the image
	Width          int       `json:"width,omitempty"`          // Width of the image in pixels
	Height         int       `json:"height,omitempty"`         // Height of the image in pixels
	AspectRatio    float64   `json:"aspectRatio,omitempty"`    // AspectRatio is width / height
	Tags           []string  `json:"tags,omitempty"`           // Tags are associated image topics or categories
	Status         Status    `json:"status"`                   // Status is the current status of the image
}

// Type returns the entity type of the Image.
func (i Image) Type() string {
	return "Image"
}

// Label returns a text label for the Image.
func (i Image) Label() string {
	if i.Title != "" {
		return i.Title
	}
	if i.AltText != "" {
		return i.AltText
	}
	if i.SourceFileName != "" {
		return i.SourceFileName
	}
	return i.FileName
}

// FileExt returns the file extension for the image (including a leading period), based on the MediaType.
func (i Image) FileExt() string {
	return i.MediaType.FileExt()
}

// FileInfo returns a FileInfo for the Image.
func (i Image) FileInfo() b.FileInfo {
	return b.FileInfo{
		FileName:      i.FileName,
		ContentType:   i.MediaType.String(),
		ContentLength: i.FileSize,
		ETag:          i.MD5Hash,
		LastModified:  i.UpdatedAt,
	}
}

// String returns a string representation of the Image.
func (i Image) String() string {
	return fmt.Sprintf("Image %s (%s)", i.Label(), i.ID)
}

// CompressedJSON returns a compressed JSON representation of the Image.
func (i Image) CompressedJSON() []byte {
	j, err := v.ToCompressedJSON(i)
	if err != nil {
		return nil
	}
	return j
}

// Validate checks whether the Image has all required fields and whether the supplied values are valid,
// returning a list of problems. If the list is empty, then the Image is valid.
func (i Image) Validate() []string {
	var problems []string
	if i.ID == "" || !tuid.IsValid(tuid.TUID(i.ID)) {
		problems = append(problems, "ID is missing or invalid")
	}
	if i.CreatedAt.IsZero() {
		problems = append(problems, "CreatedAt is missing")
	}
	if i.VersionID == "" || !tuid.IsValid(tuid.TUID(i.VersionID)) {
		problems = append(problems, "VersionID is missing or invalid")
	}
	if i.UpdatedAt.IsZero() {
		problems = append(problems, "UpdatedAt is missing")
	}
	if !i.MediaType.IsValid() {
		expected := strings.Join(SupportedMediaTypes(), ", ")
		problems = append(problems, "MediaType is missing or unsupported. Expected: "+expected)
	}
	if i.FileName == "" {
		problems = append(problems, "FileName is missing")
	}
	if !i.Status.IsValid() {
		statuses := v.Map(Statuses, func(s Status) string { return string(s) })
		expected := strings.Join(statuses, ", ")
		problems = append(problems, "Status is missing or invalid. Expected: "+expected)
	}
	return problems
}

//------------------------------------------------------------------------------

// ImageDistance represents the perceptual distance between two images,
// along with information about the similar image.
type ImageDistance struct {
	ID       string `json:"id"`               // Image ID of the similar Image
	Label    string `json:"label"`            // Image title or alt text, if available
	Source   string `json:"source,omitempty"` // Source is the URI or file name of the original image
	FileName string `json:"fileName"`         // S3 File name (ID + file extension)
	FileSize int64  `json:"fileSize"`         // File size in bytes
	MD5Hash  string `json:"md5Hash"`          // MD5 hash of the image
	Width    int    `json:"width,omitempty"`  // Width of the image in pixels
	Height   int    `json:"height,omitempty"` // Height of the image in pixels
	PHash    string `json:"pHash,omitempty"`  // PHash is the DTC perceptual hash of the image
	Distance int    `json:"distance"`         // The number of differing perceptual hash bits
}

// Populate the ImageDistance fields from the provided Image.
func (d *ImageDistance) Populate(i Image) {
	d.ID = i.ID
	d.Label = i.Label()
	if i.SourceURI != "" {
		d.Source = i.SourceURI
	} else {
		d.Source = i.SourceFileName
	}
	d.FileName = i.FileName
	d.FileSize = i.FileSize
	d.MD5Hash = i.MD5Hash
	d.Width = i.Width
	d.Height = i.Height
	d.PHash = i.PHash
}
