package content

import (
	"strings"
	"time"
	"versionary-api/pkg/ref"

	"github.com/voxtechnica/tuid-go"
	"github.com/voxtechnica/versionary"
)

// Content is a piece of content of a specified type (e.g. book, chapter, etc.)
type Content struct {
	Type         Type      `json:"type"`
	ID           string    `json:"id"`
	CreatedAt    time.Time `json:"createdAt"`
	VersionID    string    `json:"versionId"`
	UpdatedAt    time.Time `json:"updatedAt"`
	EditorID     string    `json:"editorId,omitempty"`
	EditorName   string    `json:"editorName,omitempty"`
	Comment      string    `json:"comment,omitempty"`
	WordCount    int       `json:"wordCount"`
	ImageCount   int       `json:"imageCount"`
	LinkCount    int       `json:"linkCount"`
	SectionCount int       `json:"sectionCount"`
	Tags         []string  `json:"tags,omitempty"`
	Authors      []Author  `json:"authors,omitempty"`
	Content      Section   `json:"content,omitempty"`
}

// RefID returns the Reference ID of this entity.
func (c Content) RefID() ref.RefID {
	r, _ := ref.NewRefID("Content", c.ID, c.VersionID)
	return r
}

// Title returns the title of the Content.
func (c Content) Title() string {
	t := strings.ToTitle(c.Type.String())
	if c.Content.Title != "" && c.Content.Subtitle != "" {
		return c.Content.Title + ": " + c.Content.Subtitle + " (" + t + ")"
	}
	if c.Content.Title != "" {
		return c.Content.Title + " (" + t + ")"
	}
	return t + " " + c.ID
}

// AuthorNames returns a list of the names of the authors of the Content.
func (c Content) AuthorNames() []string {
	names := make([]string, 0, len(c.Authors))
	for _, author := range c.Authors {
		if author.Name != "" {
			names = append(names, author.Name)
		}
	}
	return names
}

// CompressedJSON returns a compressed JSON representation of the Content.
func (c Content) CompressedJSON() []byte {
	j, err := versionary.ToCompressedJSON(c)
	if err != nil {
		return nil
	}
	return j
}

// Sanitize removes potentially dangerous HTML tags from the Content.
func (c Content) Sanitize() Content {
	c.Content = c.Content.Sanitize()
	return c
}

// IsEmpty returns true if the Content has no content.
func (c Content) IsEmpty() bool {
	return c.Content.IsEmpty()
}

// Validate checks whether the Content has all required fields and whether the supplied values are valid.
// It returns a list of problems, and if the list is empty, then the Content is valid.
func (c Content) Validate() []string {
	var problems []string
	if c.ID == "" || !tuid.IsValid(tuid.TUID(c.ID)) {
		problems = append(problems, "ID is missing or invalid")
	}
	if c.CreatedAt.IsZero() {
		problems = append(problems, "CreatedAt is missing")
	}
	if c.VersionID == "" || !tuid.IsValid(tuid.TUID(c.VersionID)) {
		problems = append(problems, "VersionID is missing or invalid")
	}
	if c.UpdatedAt.IsZero() {
		problems = append(problems, "UpdatedAt is missing")
	}
	if c.EditorID != "" && !tuid.IsValid(tuid.TUID(c.VersionID)) {
		problems = append(problems, "EditorID is invalid")
	}
	for _, author := range c.Authors {
		problems = append(problems, author.Validate()...)
	}
	if c.Content.IsEmpty() {
		problems = append(problems, "Content is empty")
	} else {
		problems = append(problems, c.Content.Validate()...)
	}
	if c.Type == "" {
		problems = append(problems, "Type is missing")
	}
	return problems
}
