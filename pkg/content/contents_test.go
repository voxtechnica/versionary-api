package content

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	v "github.com/voxtechnica/versionary"
)

// Set up test context and service
var (
	ctx      = context.Background()
	service  = NewMockService("test")
	book     Content
	chapter1 Content
	chapter2 Content
)

// readJSONContent reads a JSON file into a Content struct.
func readJSONContent(path string) (Content, error) {
	var c Content
	// A file path is required.
	if path == "" {
		return c, fmt.Errorf("error fetching JSON: no file path provided")
	}
	// The file must exist.
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return c, fmt.Errorf("error fetching JSON: file %s does not exist", path)
	}
	// Read the file into a byte slice.
	blob, err := os.ReadFile(path)
	if err != nil {
		return c, fmt.Errorf("error fetching JSON file %s: %w", path, err)
	}
	// Unmarshal the byte slice into a Content struct.
	if err := json.Unmarshal(blob, &c); err != nil {
		return c, fmt.Errorf("error unmarshaling JSON file %s: %w", path, err)
	}
	return c, nil
}

func TestMain(m *testing.M) {
	// Load test data from JSON files
	var err error
	book, err = readJSONContent("testdata/book.json")
	if err != nil {
		log.Fatal(err)
	}
	chapter1, err = readJSONContent("testdata/chapter1.json")
	if err != nil {
		log.Fatal(err)
	}
	chapter2, err = readJSONContent("testdata/chapter2.json")
	if err != nil {
		log.Fatal(err)
	}

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
