package org

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/voxtechnica/tuid-go"
	v "github.com/voxtechnica/versionary"
)

var (
	// Organization Service
	ctx     = context.Background()
	service = NewMockService("test")

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
		ID:        id1,
		VersionID: id1,
		CreatedAt: t1,
		UpdatedAt: t1,
		Name:      "Test Organization 1",
		Status:    PENDING,
	}
	o20 = Organization{
		ID:        id2,
		VersionID: id2,
		CreatedAt: t2,
		UpdatedAt: t2,
		Name:      "Test Organization 2",
		Status:    ENABLED,
	}
	o30 = Organization{
		ID:        id3,
		VersionID: id3,
		CreatedAt: t3,
		UpdatedAt: t3,
		Name:      "Test Organization 3",
		Status:    DISABLED,
	}
	o11 = Organization{
		ID:        id1,
		VersionID: id4,
		CreatedAt: t1,
		UpdatedAt: t4,
		Name:      "Test Organization 1.1",
		Status:    ENABLED,
	}
	knownOrgs  = []Organization{o11, o20, o30}
	knownIDs   = []string{id1, id2, id3}
	knownNames = []string{o11.Name, o20.Name, o30.Name}
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

		// Organization exists in organization table
		oExist := service.Exists(ctx, o.ID)
		expect.True(oExist)

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
		oExist = service.Exists(ctx, o.ID)
		expect.False(oExist)

		// Read the organization
		_, err = service.Read(ctx, o.ID)
		expect.ErrorIs(err, v.ErrNotFound, "expected ErrNotFound")
	}
}

func TestReadIDs(t *testing.T) {
	expect := assert.New(t)
	ids, err := service.ReadIDs(ctx, false, 10, tuid.MinID)
	if expect.NoError(err) {
		expect.GreaterOrEqual(len(ids), 3)
		expect.Subset(ids, knownIDs)
	}
}

func TestReadAllIDs(t *testing.T) {
	expect := assert.New(t)
	allIDs, err := service.ReadAllIDs(ctx)
	if expect.NoError(err) {
		expect.Subset(allIDs, knownIDs)
	}
}

func TestVersionExists(t *testing.T) {
	expect := assert.New(t)
	vExists := service.VersionExists(ctx, id2, id2)
	expect.True(vExists)	
}

func TestReadVersion(t *testing.T) {
	expect := assert.New(t)
	vExist, err := service.ReadVersion(ctx, id1, id1)
	if expect.NoError(err) {
		expect.Equal(o10, vExist)
	}
}

func TestReadVersionAsJSON(t *testing.T) {
	expect := assert.New(t)
	vJSON, err := service.ReadVersionAsJSON(ctx, id2, id2)
	if expect.NoError(err) {
		expect.Contains(string(vJSON), id2)
	}
}

func TestReadAllVersions(t *testing.T) {
	expect := assert.New(t)
	allVersions, err := service.ReadAllVersions(ctx, id2)
	if expect.NoError(err) {
		expect.Equal(o20, allVersions[0])
	}
}

func TestReadAllVersionsAsJSON(t *testing.T) {
	expect := assert.New(t)
	versionsCheckJSON, err := service.ReadAllVersionsAsJSON(ctx, id3)
	if expect.NoError(err) {
		expect.Contains(string(versionsCheckJSON), id3)
	}
}


// I don't fully believe in a reliability of this test
// talk to Dave about it on a next meeting
// Ask for v.Map(), how it works and how to implement it in this test
func TestReadNames(t *testing.T) {
	expect := assert.New(t)
	idsAndNames, err := service.ReadNames(ctx, false, 4, "") 
	fmt.Println("SHOW ME: ", idsAndNames)
	if expect.NoError(err) {
		// onlyNames := make([]string, 0, len(idsAndNames))
		// for _, kv := range idsAndNames {
		// 	onlyNames = append(onlyNames, kv.Value)
		// }

		onlyNames := v.Map(idsAndNames, func (entry v.TextValue)  string { return entry.Value})
		fmt.Println("SHOW ME: ", onlyNames)
		expect.Equal(onlyNames, knownNames)
	}
}

// work with Dave on this one
func TestReadAllNames(t *testing.T) {
	expect := assert.New(t)
	idsAndNames, err := service.ReadAllNames(ctx, true) 
	fmt.Println("SHOW ME: ", idsAndNames)

	// expectedNames := []v.TextValue{
	// 	{Value: o11.Name},
	// 	{Value: o20.Name},
	// 	{Value: o30.Name},
	// }

	expectedNames := v.Map(idsAndNames, func(entry v.TextValue) string {return entry.Value})
	fmt.Println("SHOW ME: ", expectedNames)
	if expect.NoError(err) {
		reflect.DeepEqual(idsAndNames, expectedNames)
		expect.Subset(knownNames, expectedNames)
	}
}

func TestFilterNames(t *testing.T) {
	expect := assert.New(t)
	filteredName, err := service.FilterNames(ctx, "1.1", false)
	
	expectedName := []v.TextValue{
		{
			Key:   o11.ID,
			Value: o11.Name,
		},
	}

	if expect.NoError(err) {
		expect.Equal(filteredName, expectedName)
	}
}

func TestReadStatuses(t *testing.T) {
	expect := assert.New(t)
	// Read all statuses
	allStatuses := []string{"PENDING", "ENABLED", "DISABLED"}
	statuses, err := service.ReadStatuses(ctx, false, 3, tuid.MinID)
	if expect.NoError(err) {
		expect.GreaterOrEqual(len(statuses), len(allStatuses))
		expect.Subset(allStatuses, statuses)
	}
}

func TestReadAllStatuses(t *testing.T) {
	expect := assert.New(t)
	// Read all statuses
	allStatuses := []string{"PENDING", "ENABLED", "DISABLED"}
	statuses, err := service.ReadAllStatuses(ctx)
	if expect.NoError(err) {
		expect.Subset(allStatuses, statuses)
	}
}

func TestReadOrganizationsByStatus(t *testing.T) {
	expect := assert.New(t)
	checkOrgs, err := service.ReadOrganizationsByStatus(ctx, "DISABLED", false, 1, "")
	if expect.NoError(err) && expect.NotEmpty(checkOrgs){
		expect.Equal(o30, checkOrgs[0])
	}
}

func TestReadOrganizationByStatusAsJSON(t *testing.T) {
	expect := assert.New(t)
	checkOrgs, err := service.ReadOrganizationsByStatusAsJSON(ctx, "PENDING", false, 1, "")
	if expect.NoError(err) {
		expect.Contains(string(checkOrgs), o10.Status)
	}
}

func TestReadAllOrganizationsByStatus(t *testing.T) {
	expect := assert.New(t)
	checkOrgs, err := service.ReadAllOrganizationsByStatus(ctx, "ENABLED")
	if expect.NoError(err) && expect.NotEmpty(checkOrgs){		
		var numOfOrgs  []int
		var orgsIds []string
		for i, v := range checkOrgs {
			numOfOrgs = append(numOfOrgs, i)
			orgsIds = append(orgsIds, v.ID)
		}
		expect.GreaterOrEqual(len(numOfOrgs), 2)
		expect.Subset(knownIDs, orgsIds)
	}
}

func TestReadAllOrganizationsByStatusAsJSON(t *testing.T) {
	expect := assert.New(t)
	checkOrgs, err := service.ReadAllOrganizationsByStatusAsJSON(ctx, "DISABLED")
	if expect.NoError(err) {
		expect.Contains(string(checkOrgs), o30.Status)
	}
}

func TestDeleteVersion(t *testing.T) {
	expect := assert.New(t)
	vDeleted, err := service.DeleteVersion(ctx, id1, id4)
	if expect.NoError(err) {
		expect.Equal(o11.VersionID, vDeleted.VersionID)

		vExists := service.VersionExists(ctx, id1, id4)
		expect.False(vExists)
	}
}



