package content

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	v "github.com/voxtechnica/versionary"
)

// Set up test context and service
var (
	ctx     = context.Background()
	service = NewMockService("test")
)

func TestMain(m *testing.M) {
	// Open the test data file
	file, err := os.Open("content_test.json")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// Load test data from content_test.json
	var testData []Content
	err = json.NewDecoder(file).Decode(&testData)
	if err != nil {
		log.Fatal(err)
	}

	// Write the test data to the content table
	for _, con := range testData {
		if _, err := service.Write(ctx, con); err != nil {
			log.Fatal(err)
		}
	}

	// Run tests
	exitCode := m.Run()

	// Clean up test data from content table
	for _, con := range testData {
		if _, err := service.Delete(ctx, con.ID); err != nil {
			log.Fatal(err)
		}
	}

	// Exit
	os.Exit(exitCode)
}

func TestCRUD(t *testing.T) {
	expect := assert.New(t)
	// Create a content
	ctx := context.Background() // Declare ctx variable
	con, problems, err := service.Create(ctx, Content{
		Type:    BOOK,
		Comment: "This is a comment for a CRUD test",
		Content: Section{
			Title:    "Package content CRUD Test 1",
			Subtitle: "Package content CRUD Test Subtitle 1",
		},
	})
	expect.Empty(problems)

	if expect.NoError(err) {
		// Content exists in content table
		conExist := service.Exists(ctx, con.ID)
		expect.True(conExist)
	}
	// Read the content
	conCheck, err := service.Read(ctx, con.ID)
	if expect.NoError(err) {
		// Check the content
		expect.Equal(con, conCheck)
	}
	// read the content as JSON
	conCheckJSON, err := service.ReadAsJSON(ctx, con.ID)
	if expect.NoError(err) {
		expect.Contains(string(conCheckJSON), con.ID)
	}
	// Update the content
	con.Comment = "This is an updated comment for a CRUD test"
	conCheck, _, err = service.Update(ctx, con)
	if expect.NoError(err) {
		// Verify the version ID has changed
		expect.NotEqual(conCheck.ID, conCheck.VersionID)
		expect.NotEqual(conCheck.Comment, "This is a comment for a CRUD test")
	}
	// Delete version
	vDeleted, err := service.DeleteVersion(ctx, con.ID, con.VersionID)
	if expect.NoError(err) {
		expect.Equal(con.ID, vDeleted.ID)

		vExist := service.VersionExists(ctx, con.ID, con.VersionID)
		expect.False(vExist)
	}
	// Delete the content
	conDelete, err := service.Delete(ctx, con.ID)
	if expect.NoError(err) {
		// Check the content
		expect.Equal(conCheck, conDelete)
	}
	// Content does not exist in content table
	conExist := service.Exists(ctx, con.ID)
	expect.False(conExist)

	// Read the content
	_, err = service.Read(ctx, con.ID)
	expect.ErrorIs(err, v.ErrNotFound, "expected ErrNotFound")
}
