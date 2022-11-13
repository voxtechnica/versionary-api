package content

import (
	"strings"
	"versionary-api/pkg/image"
	"versionary-api/pkg/policy"

	"github.com/voxtechnica/tuid-go"
)

// Section represents a section of HTML content with titles, text, images, links, and subsections.
// All fields are optional, but don't leave empty sections lying about.
type Section struct {
	ID       string        `json:"id,omitempty"`       // Section ID used for client application state management
	Title    string        `json:"title,omitempty"`    // Section title (heading; optional)
	Subtitle string        `json:"subtitle,omitempty"` // Section subtitle (subheading; optional)
	Text     string        `json:"text,omitempty"`     // Section text (HTML content; optional)
	Images   []image.Image `json:"images,omitempty"`   // Section images (optional)
	Links    []Link        `json:"links,omitempty"`    // Section links (optional)
	Sections []Section     `json:"sections,omitempty"` // Nested subsections (optional)
}

// Type returns the EntityType of the Section.
func (s Section) Type() string {
	return "Section"
}

// Sanitize removes potentially dangerous HTML tags from the Section and all subsections.
// If the ID is missing, a new one is generated.
func (s Section) Sanitize() Section {
	if s.ID == "" {
		s.ID = tuid.NewID().String()
	}
	s.Title = policy.PlainText.Sanitize(s.Title)
	s.Subtitle = policy.PlainText.Sanitize(s.Subtitle)
	s.Text = policy.Content.Sanitize(s.Text)
	for i, link := range s.Links {
		s.Links[i] = link.Sanitize()
	}
	for i, section := range s.Sections {
		s.Sections[i] = section.Sanitize()
	}
	return s
}

// IsEmpty returns true if the Section has no titles, text, images, links, or subsections.
func (s Section) IsEmpty() bool {
	return strings.TrimSpace(s.Title+s.Subtitle+s.Text) == "" &&
		len(s.Images) == 0 && len(s.Links) == 0 && len(s.Sections) == 0
}

// IsValid returns true if the Section is minimally functional.
func (s Section) IsValid() bool {
	return !s.IsEmpty()
}

// Validate checks whether the Section has all required fields and whether the supplied values are valid.
// It returns a list of problems, and if the list is empty, then the Section is valid.
// The entire structure is traversed and validated, including any subsections.
func (s Section) Validate() []string {
	var problems []string
	if s.ID != "" && !tuid.IsValid(tuid.TUID(s.ID)) {
		problems = append(problems, "Section ID is invalid")
	}
	for _, img := range s.Images {
		problems = append(problems, img.Validate()...)
	}
	for _, link := range s.Links {
		problems = append(problems, link.Validate()...)
	}
	for _, section := range s.Sections {
		problems = append(problems, section.Validate()...)
	}
	if s.IsEmpty() {
		problems = append(problems, "Section is empty")
	}
	return problems
}

// WordCount returns the total number of words in this Section and all subsections.
func (s Section) WordCount() int {
	count := len(strings.Fields(s.Title)) + len(strings.Fields(s.Subtitle))
	count += len(strings.Fields(policy.PlainText.Sanitize(s.Text)))
	for _, img := range s.Images {
		count += img.WordCount()
	}
	for _, link := range s.Links {
		count += link.WordCount()
	}
	for _, section := range s.Sections {
		count += section.WordCount()
	}
	return count
}

// ImageCount returns the total number of images in this Section and all subsections.
func (s Section) ImageCount() int {
	count := len(s.Images)
	for _, section := range s.Sections {
		count += section.ImageCount()
	}
	return count
}

// LinkCount returns the total number of links in this Section and all subsections.
func (s Section) LinkCount() int {
	count := 0
	for _, link := range s.Links {
		count += link.LinkCount()
	}
	for _, section := range s.Sections {
		count += section.LinkCount()
	}
	return count
}

// SectionCount returns the total number of sections, including this Section and all subsections.
func (s Section) SectionCount() int {
	count := 1
	for _, section := range s.Sections {
		count += section.SectionCount()
	}
	return count
}
