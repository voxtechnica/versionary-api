package org

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/voxtechnica/tuid-go"
	v "github.com/voxtechnica/versionary"
)

var (
	// Organization Service
	ctx      = context.Background()
	service  = NewMockService("test")

	// Known timestamps
	t1 = time.Date(2022, time.April, 1, 12, 0, 0, 0, time.UTC)
	t2 = time.Date(2022, time.April, 1, 13, 0, 0, 0, time.UTC)
	t3 = time.Date(2022, time.April, 1, 14, 0, 0, 0, time.UTC)
	t4 = time.Date(2022, time.April, 2, 12, 0, 0, 0, time.UTC)

	// Organization IDs
	id1 = tuid.NewIDWithTime(t1).String()
	id2 = tuid.NewIDWithTime(t2).String()
	id3 = tuid.NewIDWithTime(t3).String()
	id4 = tuid.NewIDWithTime(t4).String()

	// Known Organizations
	o10 = Organization{
		ID:         id1,
		VersionID:  id1,
		CreatedAt:  t1,
		UpdatedAt:  t1,
		Name: "Test Organization 1",
		Status: PENDING,
	}
	o20 = Organization{
		ID:         id2,
		VersionID:  id2,
		CreatedAt:  t2,
		UpdatedAt:  t2,
		Name: "Test Organization 2",
		Status: ENABLED,
	}
	o30 = Organization{
		ID:         id3,
		VersionID:  id3,
		CreatedAt:  t3,
		UpdatedAt:  t3,
		Name: "Test Organization 3",
		Status: DISABLED,
	}
	o11 = Organization{
		ID:         id1,
		VersionID:  id4,
		CreatedAt:  t1,
		UpdatedAt:  t4,
		Name: "Test Organization 1.1",
		Status: ENABLED,
	}
	knownOrgs = []Organization{o11, o20, o30}
	knownIDs = []string{id1, id2, id3}
)

func TestMain(m *testing.M) {
	// Check the table/row definitions
	if !service.Table.IsValid() {
		log.Fatal("invalid table configuration")
	}
	// Write known organizations
	for _, e := range []Organization{o10, o11, o20, o30} {
		if _, err := service.Write(ctx, e); err != nil {
			log.Fatal(err)
		}
	}
	// Run the tests
	m.Run()
}

func TestCreateReadUpdateDelete(t *testing.T) {
	expect := assert.New(t)
	// Create an organization
	o, problems, err := service.Create(ctx, Organization{
		Name:   "Test Organization 4",
		Status: PENDING,
	})
	expect.Empty(problems)
	if expect.NoError(err) {
		// Read the organization
		oCheck, err := service.Read(ctx, o.ID)
		if expect.NoError(err) {
			// Check the organization
			expect.Equal(o, oCheck)
		}
		// Read the organization as JSON
		oCheckJSON, err := service.ReadAsJSON(ctx, o.ID)
		if expect.NoError(err) {
			expect.Contains(string(oCheckJSON), o.ID)
		}
		// Update the organization
		o.Status = ENABLED
		oUpdated, _, err := service.Update(ctx, o)
		if expect.NoError(err) {
			// Verify that the version ID has changed
			expect.NotEqual(oUpdated.ID, oUpdated.VersionID)
			expect.NotEqual(oUpdated.Status, PENDING)
		}
		// Delete the organization
		oDelete, err := service.Delete(ctx, o.ID)
		if expect.NoError(err) {
			// Check the organization
			expect.Equal(oUpdated, oDelete)
		}
		// Organization does not exist
		oExist := service.Exists(ctx, o.ID)
		expect.False(oExist)

		// Read the organization
		_, err = service.Read(ctx, o.ID)
		expect.ErrorIs(err, v.ErrNotFound, "expected ErrNotFound")
	}
}
