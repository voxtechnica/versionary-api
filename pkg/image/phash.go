package image

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"slices"
	"strings"

	"github.com/corona10/goimagehash"
	"github.com/voxtechnica/versionary"
)

// PHash is a 256-bit perceptual hash of an image, segmented into four 64-bit blocks.
type PHash string

// String returns the string representation of the PHash.
func (h PHash) String() string {
	return string(h)
}

// Distance returns the Hamming distance between two PHashes.
// The values will range from 0 to 256, with 0 being identical.
func (h PHash) Distance(o PHash) (int, error) {
	a, err := h.parse()
	if err != nil {
		return 256, fmt.Errorf("hash distance: %w", err)
	}
	b, err := o.parse()
	if err != nil {
		return 256, fmt.Errorf("hash distance: %w", err)
	}
	return a.Distance(b)
}

// Distances returns the Hamming distance between a PHash and a list of PHashes.
// This is intended for finding similar images, supporting a limit and a maximum distance.
// The text values are an image ID and a PHash, retrieved from the database.
// The results are sorted by distance (closest first), then by ID.
func (h PHash) Distances(o []versionary.TextValue, max, limit int) ([]Distance, error) {
	var distances []Distance
	// Parse this hash
	a, err := h.parse()
	if err != nil {
		return distances, fmt.Errorf("hash distances: %w", err)
	}
	// Parse those hashes, filtering out invalid hashes and distances
	for _, v := range o {
		ph := PHash(v.Value)
		b, err := ph.parse()
		if err != nil {
			continue // skip invalid hashes
		}
		d, err := a.Distance(b)
		if err != nil {
			continue // skip invalid distances
		}
		if d <= max {
			distances = append(distances, Distance{
				ID:       v.Key,
				PHash:    ph,
				Distance: d,
			})
		}
	}
	// Sort the results by distance, then by ID
	slices.SortFunc(distances, func(i, j Distance) int {
		if i.Distance == j.Distance {
			return strings.Compare(i.ID, j.ID)
		}
		return i.Distance - j.Distance
	})
	// Limit the results
	if limit > 0 && len(distances) > limit {
		distances = distances[:limit]
	}
	return distances, nil
}

// parse returns the 256-bit hash as a slice of 64-bit blocks.
func (h PHash) parse() (*goimagehash.ExtImageHash, error) {
	var bits []uint64
	for _, b := range strings.Split(string(h), ":") {
		u, err := decode(b)
		if err != nil {
			return nil, fmt.Errorf("parse hash %s: %w", h, err)
		}
		bits = append(bits, u)
	}
	if len(bits) != 4 {
		return nil, fmt.Errorf("parse hash %s: invalid length", h)
	}
	return goimagehash.NewExtImageHash(bits, goimagehash.PHash, len(bits)*64), nil
}

// NewPHash returns a 256-bit perceptual hash of an image.
func NewPHash(i image.Image) (PHash, error) {
	h, err := goimagehash.ExtPerceptionHash(i, 16, 16) // 16x16 pixels, 256-bit hash
	if err != nil {
		return "", fmt.Errorf("hash image: %w", err)
	}
	var s []string
	for _, b := range h.GetHash() {
		s = append(s, encode(b))
	}
	return PHash(strings.Join(s, ":")), nil
}

// base is the number of unique digits in the encoding
var base uint64 = 62

// digits is the set of unique digits in the encoding
var digits = []byte("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz")

// encode the provided unsigned integer into a base-62 encoded string
func encode(value uint64) string {
	var result []byte
	for value > 0 {
		q := value / base
		r := value % base
		d := digits[r]
		result = append([]byte{d}, result...) // prepend the new digit
		value = q
	}
	if len(result) == 0 {
		return string(digits[0])
	}
	return string(result)
}

// decode the provided base-62 encoded string into an unsigned integer
func decode(text string) (uint64, error) {
	textBytes := []byte(text)
	size := len(textBytes)
	if size == 0 {
		return 0, errors.New("base 62 decoding error: no digits")
	}
	var result uint64
	for i := 0; i < size; i++ {
		b := textBytes[size-1-i] // examine digits from right to left
		j := bytes.IndexByte(digits, b)
		if j == -1 {
			return 0, fmt.Errorf("base 62 decoding error: invalid digit `%s` in %s",
				string(b), string(textBytes))
		}
		result += uint64(j) * power(base, uint64(i))
	}
	return result, nil
}

// power computes a**b using a binary powering algorithm
// See Donald Knuth, The Art of Computer Programming, Volume 2, Section 4.6.3
func power(a, b uint64) uint64 {
	var p uint64 = 1
	for b > 0 {
		if b&1 != 0 {
			p *= a
		}
		b >>= 1
		a *= a
	}
	return p
}
