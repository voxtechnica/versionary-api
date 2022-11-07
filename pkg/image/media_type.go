package image

import (
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/webp"
)

// MediaType is a type for supported Image media types.
type MediaType string

const (
	// GIF is the media type for GIF images
	GIF MediaType = "image/gif"

	// JPEG is the media type for JPEG images
	JPEG MediaType = "image/jpeg"

	// PNG is the media type for PNG images
	PNG MediaType = "image/png"

	// WebP is the media type for WebP images
	WebP MediaType = "image/webp"
)

// MediaTypes is the complete list of valid, supported MediaTypes and their corresponding file extensions.
var MediaTypes = map[MediaType]string{
	JPEG: ".jpeg",
	WebP: ".webp",
	PNG:  ".png",
	GIF:  ".gif",
}

// IsValid returns true if the supplied MediaType is recognized.
func (m MediaType) IsValid() bool {
	_, ok := MediaTypes[m]
	return ok
}

// FileExt returns the file extension for the MediaType (including a leading period).
// Unsupported media types return an empty string.
func (m MediaType) FileExt() string {
	return MediaTypes[m]
}

// String returns a string representation of the MediaType.
func (m MediaType) String() string {
	return string(m)
}

func SupportedMediaTypes() []string {
	return []string{
		WebP.String(),
		JPEG.String(),
		PNG.String(),
		GIF.String(),
	}
}
