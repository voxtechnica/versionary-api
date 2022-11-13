package content

import (
	"net/url"
	"versionary-api/pkg/email"
)

// Author is a person who has contributed to a book.
type Author struct {
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
	URL   string `json:"url,omitempty"`
}

// IsEmpty returns true if the Author has no content.
func (a Author) IsEmpty() bool {
	return a.Name == ""
}

// Validate checks whether the Author has all required fields and whether the supplied values are valid.
// It returns a list of problems, and if the list is empty, then the Author is valid.
func (a Author) Validate() []string {
	var problems []string
	if a.Name == "" {
		problems = append(problems, "Author Name is missing")
	}
	if a.Email != "" {
		_, err := email.NewIdentity(a.Name, a.Email)
		if err != nil {
			problems = append(problems, "Author Email is invalid")
		}
	}
	if a.URL != "" {
		_, err := url.Parse(a.URL)
		if err != nil {
			problems = append(problems, "Author URL is invalid")
		}
	}
	return problems
}
