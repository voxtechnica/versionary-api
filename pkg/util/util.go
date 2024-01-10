package util

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/voxtechnica/tuid-go"
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

// DateRangeIDs returns a pair of first IDs (TUIDs) for the specified date range.
// The start date is inclusive, and the end date is effectively exclusive.
// The expected date format is YYYY-MM-DD.
func DateRangeIDs(startDate, endDate string) (string, string, error) {
	var start, end string
	startTime, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return start, end, fmt.Errorf("invalid date %s (expect yyyy-mm-dd): %w", startDate, err)
	}
	start = tuid.FirstIDWithTime(startTime).String()
	endTime, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return start, end, fmt.Errorf("invalid date %s (expect yyyy-mm-dd): %w", endDate, err)
	}
	end = tuid.FirstIDWithTime(endTime).String()
	return start, end, nil
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
