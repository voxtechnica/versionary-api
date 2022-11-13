package content

import "github.com/voxtechnica/versionary"

// Type indicates the type of content.
type Type string

// BOOK Type indicates that the content is a book.
const BOOK Type = "BOOK"

// CHAPTER Type indicates that the content is a chapter of a book.
const CHAPTER Type = "CHAPTER"

// ARTICLE Type indicates that the content is an article.
const ARTICLE Type = "ARTICLE"

// CATEGORY Type indicates that the content is a category (group of content).
const CATEGORY Type = "CATEGORY"

// Types is the complete list of valid content types.
var Types = []Type{BOOK, CHAPTER, ARTICLE, CATEGORY}

// IsValid returns true if the supplied Type is recognized.
func (t Type) IsValid() bool {
	for _, v := range Types {
		if t == v {
			return true
		}
	}
	return false
}

// String returns a string representation of the Type.
func (t Type) String() string {
	return string(t)
}

// SupportedTypes returns a list of the supported content types.
func SupportedTypes() []string {
	return versionary.Map(Types, func(t Type) string { return t.String() })
}
