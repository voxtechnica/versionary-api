package util

import (
	"errors"
	"regexp"
	"strings"

	"github.com/voxtechnica/versionary"
)

// ErrEmptyFilter is returned when the filter string is empty.
var ErrEmptyFilter = errors.New("empty filter")

// ContainsFilter returns a function that can be used to filter TextValues.
// The case-insensitive contains query is split into words, and the words are compared with the value in the TextValue.
// If anyMatch is true, then a TextValue is included in the results if any of the words are found (OR filter).
// If anyMatch is false, then the TextValue must contain all the words in the query string (AND filter).
// TODO: if this is generally useful, move it to versionary.
func ContainsFilter(contains string, anyMatch bool) (func(tv versionary.TextValue) bool, error) {
	terms := strings.Fields(strings.ToLower(contains))
	if len(terms) == 0 {
		return func(tv versionary.TextValue) bool { return false }, ErrEmptyFilter
	}
	if anyMatch {
		return func(tv versionary.TextValue) bool { return tv.ContainsAny(terms) }, nil
	} else {
		return func(tv versionary.TextValue) bool { return tv.ContainsAll(terms) }, nil
	}
}

// dateRegex is a regular expression that matches a date in the format YYYY-MM-DD.
var dateRegex = regexp.MustCompile(`^\d{4}-(0[1-9]|1[012])-(0[1-9]|[12][0-9]|3[01])$`)

// IsValidDate returns true if the supplied string is a valid date in the format YYYY-MM-DD.
func IsValidDate(date string) bool {
	return date != "" && dateRegex.MatchString(date)
}

// TextValuesMap converts a slice of TextValues into a key/value map.
func TextValuesMap(textValues []versionary.TextValue) map[string]string {
	m := make(map[string]string)
	for _, tv := range textValues {
		m[tv.Key] = tv.Value
	}
	return m
}

// NumValuesMap converts a slice of NumValues into a key/value map.
func NumValuesMap(numericValues []versionary.NumValue) map[string]float64 {
	m := make(map[string]float64)
	for _, nv := range numericValues {
		m[nv.Key] = nv.Value
	}
	return m
}
