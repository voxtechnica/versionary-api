package email

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewIdentity(t *testing.T) {
	expect := assert.New(t)

	// Normal, full email address (standardized to lowercase)
	i, err := NewIdentity("", "Test Name <Test@Example.com>")
	if expect.NoError(err) {
		expect.Equal("Test Name", i.Name)
		expect.Equal("test@example.com", i.Address)
	}

	// Supplied name overrides parsed name
	i, err = NewIdentity("test name", "Test Name <Test@Example.com>")
	if expect.NoError(err) {
		expect.Equal("test name", i.Name)
		expect.Equal("test@example.com", i.Address)
	}

	// Spurious whitespace is trimmed
	i, err = NewIdentity(" test name ", " Test Name <Test@Example.com> ")
	if expect.NoError(err) {
		expect.Equal("test name", i.Name)
		expect.Equal("test@example.com", i.Address)
	}

	// Missing name
	i, err = NewIdentity("", "Test@Example.com")
	if expect.NoError(err) {
		expect.Equal("", i.Name)
		expect.Equal("test@example.com", i.Address)
	}

	// Missing address
	i, err = NewIdentity("", "")
	if expect.Error(err) {
		expect.Equal("missing email address", err.Error())
	}

	// Invalid address (missing domain)
	i, err = NewIdentity("", "Test Name <test@>")
	if expect.Error(err) {
		expect.Contains(err.Error(), "invalid email address")
	}

	// Invalid address (missing local part)
	i, err = NewIdentity("", "Test Name <@example.com>")
	if expect.Error(err) {
		expect.Contains(err.Error(), "invalid email address")
	}

	// Invalid address (space in local part)
	i, err = NewIdentity("", "Test Name <test @example.com>")
	if expect.Error(err) {
		expect.Contains(err.Error(), "invalid email address")
	}

	// Invalid address (space in domain)
	i, err = NewIdentity("", "Test Name <test@example .com>")
	if expect.Error(err) {
		expect.Contains(err.Error(), "invalid email address")
	}
}
