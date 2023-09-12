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
		return c, fmt.Errorf("error unmarshal JSON file %s: %w", path, err)
	}
	return c, nil
}

func TestMain(m *testing.M) {
	// Load test data from JSON files
	var err error
	chapter1, err = readJSONContent("testdata/chapter1.json")
	if err != nil {
		log.Fatal(err)
	}
	chapter1, _, err = service.Create(ctx, chapter1)
	if err != nil {
		log.Fatal(err)
	}

	chapter2, err = readJSONContent("testdata/chapter2.json")
	if err != nil {
		log.Fatal(err)
	}
	chapter2, _, err = service.Create(ctx, chapter2)
	if err != nil {
		log.Fatal(err)
	}

	book, err = readJSONContent("testdata/book.json")
	if err != nil {
		log.Fatal(err)
	}
	book, _, err = service.Create(ctx, book)
	if err != nil {
		log.Fatal(err)
	}

	// Run tests
	exitCode := m.Run()

	// Exit
	os.Exit(exitCode)
}

func TestCRUD(t *testing.T) {
	expect := assert.New(t)
	// Create a content
	ctx := context.Background() // Declare ctx variable
	con, problems, err := service.Create(ctx, Content{
		Type:    ARTICLE,
		Comment: "This is a comment for a CRUD test",
		Body: Section{
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

func TestReadAsJSON(t *testing.T) {
	expect := assert.New(t)
	conJSON, err := service.ReadAsJSON(ctx, book.ID)
	if expect.NoError(err) {
		expect.Contains(string(conJSON), book.ID)
	}
}

func TestVersionExists(t *testing.T) {
	expect := assert.New(t)
	// Version exists
	vExist := service.VersionExists(ctx, book.ID, book.VersionID)
	expect.True(vExist)
	// Version does not exist
	vExist = service.VersionExists(ctx, book.ID, "00000000000000000000000000000000")
	expect.False(vExist)
}

// func TestReadVersion(t *testing.T) {
// 	expect := assert.New(t)
// 	// Read the content
// 	vCheck, err := service.ReadVersion(ctx, book.ID, book.VersionID)
// 	if expect.NoError(err) {
// 		// Check the content version
// 		expect.Equal(book, vCheck)
// 		fmt.Println(vCheck.VersionID)
// 		fmt.Println(book.ID)
// 		fmt.Println(book.VersionID)
// 	}
// }

func TestReadVersionAsJSON(t *testing.T) {
	expect := assert.New(t)
	vJSON, err := service.ReadVersionAsJSON(ctx, book.ID, book.VersionID)
	if expect.NoError(err) {
		expect.Contains(string(vJSON), book.ID)
	}
}

func TestReadVersions(t *testing.T) {
	expect := assert.New(t)
	versions, err := service.ReadVersions(ctx, book.ID, false, 2, "")
	if expect.NoError(err) && expect.NotEmpty(versions) {
		expect.Equal(book.VersionID, versions[0].VersionID)
	}
}

func TestReadVersionsAsJSON(t *testing.T) {
	expect := assert.New(t)
	vJSON, err := service.ReadVersionsAsJSON(ctx, book.ID, false, 2, "")
	if expect.NoError(err) {
		expect.Contains(string(vJSON), book.ID)
	}
}

// func TestReadAllVersions(t *testing.T) {
// 	expect := assert.New(t)
// 	allVersions, err := service.ReadAllVersions(ctx, book.ID)
// 	if expect.NoError(err) && expect.NotEmpty(allVersions) {
// 		fmt.Println(allVersions[0])
// 		fmt.Println(book)
// 		expect.Equal(book, allVersions[0])
// 	}
// }

func TestReadAllVersionsAsJSON(t *testing.T) {
	expect := assert.New(t)
	versionsJSON, err := service.ReadAllVersionsAsJSON(ctx, book.ID)
	if expect.NoError(err) {
		expect.Contains(string(versionsJSON), book.ID)
	}
}

func TestReadContentIDs(t *testing.T) {
	expect := assert.New(t)
	conIDs, err := service.ReadContentIDs(ctx, false, 3, "")
	if expect.NoError(err) {
		expect.GreaterOrEqual(len(conIDs), 3)
	}
}

func TestReadAllContentIDs(t *testing.T) {
	expect := assert.New(t)
	// Read the content
	conIDs, err := service.ReadAllContentIDs(ctx)
	if expect.NoError(err) {
		// Check the content IDs
		expect.GreaterOrEqual(len(conIDs), 3)
	}
}

func TestReadTitles(t *testing.T) {
	expect := assert.New(t)
	knownContentObjects := []Content{book, chapter1, chapter2}
	expectedTitles := v.Map(knownContentObjects, func(c Content) string { return c.Body.Title + ": " + c.Body.Subtitle + " (" + string(c.Type) + ")" })
	idsAndTitles, err := service.ReadTitles(ctx, false, 10, "")
	if expect.NoError(err) {
		onlyTitles := v.Map(idsAndTitles, func(entry v.TextValue) string { return entry.Value })
		expect.Subset(onlyTitles, expectedTitles)
		// fmt.Println("ONLY TITLES: ", onlyTitles)
	}
}
