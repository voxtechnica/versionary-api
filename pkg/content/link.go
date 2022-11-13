package content

import (
	"net/url"
	"strings"
	"versionary-api/pkg/image"
	"versionary-api/pkg/policy"

	"github.com/voxtechnica/tuid-go"
)

// Link represents a hyperlink to a resource, with optional entity information.
// The Link may be simple, with only a URL, or it may be more complex,
// including additional information about the entity to which it points.
type Link struct {
	// ID is used for client application state management
	ID string `json:"id"`

	// Title (anchor tag body), useful for landscape view (required)
	Title string `json:"title"`

	// ShortTitle (anchor tag body), useful for portrait view (optional)
	ShortTitle string `json:"shortTitle,omitempty"`

	// URL (anchor tag href), may be relative or absolute (required)
	URL string `json:"url"`

	// EntityType of the destination resource (optional)
	EntityType string `json:"entityType,omitempty"`

	// EntityID (a TUID) of the destination resource (optional)
	EntityID string `json:"entityId,omitempty"`

	// Image(s) for the link destination (optional). Thumbnail images of different sizes may be provided.
	Images []image.Image `json:"images,omitempty"`

	// Description is a brief rich text (HTML) description of the destination (optional)
	Description string `json:"description,omitempty"`
}

// Type returns the EntityType of the Link.
func (l Link) Type() string {
	return "Link"
}

// Sanitize removes potentially dangerous HTML tags from the Link.
// If the ID is missing, a new one is generated.
func (l Link) Sanitize() Link {
	if l.ID == "" {
		l.ID = tuid.NewID().String()
	}
	l.Title = policy.PlainText.Sanitize(l.Title)
	l.ShortTitle = policy.PlainText.Sanitize(l.ShortTitle)
	l.Description = policy.RichText.Sanitize(l.Description)
	return l
}

// IsEmpty returns true if the Link has no valid link.
func (l Link) IsEmpty() bool {
	return !l.IsValid()
}

// IsValid returns true if the Link is minimally functional.
func (l Link) IsValid() bool {
	if l.URL != "" {
		_, err := url.Parse(l.URL)
		if err != nil {
			return false
		}
	}
	return strings.TrimSpace(l.Title) != "" && l.URL != ""
}

// Validate checks whether the Link has all required fields and whether the supplied values are valid.
// It returns a list of problems, and if the list is empty, then the Link is valid.
func (l Link) Validate() []string {
	var problems []string
	if l.ID != "" && !tuid.IsValid(tuid.TUID(l.ID)) {
		problems = append(problems, "Link ID is invalid")
	}
	if l.Title == "" {
		problems = append(problems, "Link Title is missing")
	}
	if l.URL == "" {
		problems = append(problems, "Link URL is missing")
	} else {
		_, err := url.Parse(l.URL)
		if err != nil {
			problems = append(problems, "Link URL is invalid")
		}
	}
	if l.EntityType != "" && l.EntityID == "" {
		problems = append(problems, "Link EntityID is missing")
	}
	if l.EntityID != "" && l.EntityType == "" {
		problems = append(problems, "Link EntityType is missing")
	}
	if l.EntityID != "" && !tuid.IsValid(tuid.TUID(l.EntityID)) {
		problems = append(problems, "Link EntityID is invalid")
	}
	for _, img := range l.Images {
		problems = append(problems, img.Validate()...)
	}
	return problems
}

// WordCount returns the total number of words in this Link
func (l Link) WordCount() int {
	count := len(strings.Fields(l.Title))
	count += len(strings.Fields(policy.PlainText.Sanitize(l.Description)))
	for _, img := range l.Images {
		count += img.WordCount()
	}
	return count
}

// ImageCount returns the total number of images in this Link
func (l Link) ImageCount() int {
	return len(l.Images)
}

// LinkCount returns 1 for this Link, or 0 if it's empty
func (l Link) LinkCount() int {
	if l.IsEmpty() {
		return 0
	}
	return 1
}
