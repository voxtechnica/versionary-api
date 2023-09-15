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

// vCheck object has <nil> value included in []image.Image(nil) and []content.Section(nil)
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

// allVersions object has <nil> value included in []image.Image(nil) and []content.Section(nil)
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
		expect.GreaterOrEqual(len(onlyTitles), 3)
		expect.Subset(onlyTitles, expectedTitles)
	}
}

func TestReadAllTitles(t *testing.T) {
	expect := assert.New(t)
	knownContentObjects := []Content{chapter1}
	expectedTitle := v.Map(knownContentObjects, func(c Content) string { return c.Body.Title + ": " + c.Body.Subtitle + " (" + string(c.Type) + ")" })
	allIDsAndTitles, err := service.ReadAllTitles(ctx, true)
	if expect.NoError(err) {
		onlyTitles := v.Map(allIDsAndTitles, func(entry v.TextValue) string { return entry.Value })
		expect.Subset(onlyTitles, expectedTitle)
	}
}

func TestFilterTitles(t *testing.T) {
	expect := assert.New(t)
	filteredTitle, err := service.FilterTitles(ctx, "workstation", true)
	expectedTitle := []v.TextValue{
		{
			Key:   chapter1.ID,
			Value: chapter1.Body.Title + ": " + chapter1.Body.Subtitle + " (" + string(chapter1.Type) + ")",
		},
	}
	if expect.NoError(err) {
		expect.Equal(filteredTitle, expectedTitle)
	}
}

func TestReadContents(t *testing.T) {
	expect := assert.New(t)
	contents := service.ReadContents(ctx, false, 10, "")
	if expect.NotEmpty(contents) {

		expect.GreaterOrEqual(len(contents), 3)
	}
}

//------------------------------------------------------------------------------
// Content Titles by Type
//------------------------------------------------------------------------------

func TestReadAllTypes(t *testing.T) {
	expect := assert.New(t)
	allTypes, err := service.ReadAllTypes(ctx)
	if expect.NoError(err) {
		expect.Contains(allTypes, string(BOOK))
		expect.Contains(allTypes, string(CHAPTER))
	}
}

func TestReadTitlesByType(t *testing.T) {
	expect := assert.New(t)
	idsAndTitles, err := service.ReadTitlesByType(ctx, "CHAPTER", false, 10, "")
	if expect.NoError(err) {
		onlyTitles := v.Map(idsAndTitles, func(entry v.TextValue) string { return entry.Value })
		expect.Equal(len(onlyTitles), 2)
	}
}

func TestReadAllTitlesByType(t *testing.T) {
	expect := assert.New(t)
	idsAndTitles, err := service.ReadAllTitlesByType(ctx, "BOOK", true)
	if expect.NoError(err) {
		onlyTitles := v.Map(idsAndTitles, func(entry v.TextValue) string { return entry.Value })
		expect.Equal(len(onlyTitles), 1)

	}
}

func TestFilterTitlesByType(t *testing.T) {
	expect := assert.New(t)
	filteredTitle, err := service.FilterTitlesByType(ctx, "BOOK", "easy", true)
	expectedTitle := []v.TextValue{
		{
			Key:   book.ID,
			Value: book.Body.Title + ": " + book.Body.Subtitle + " (" + string(book.Type) + ")",
		},
	}
	if expect.NoError(err) {
		expect.Equal(filteredTitle, expectedTitle)
	}
}

//------------------------------------------------------------------------------
// Content Titles by Author
//------------------------------------------------------------------------------

func TestAllAuthors(t *testing.T) {
	expect := assert.New(t)
	// knownAuthors := []string{"Editorial Team"}
	authors, err := service.ReadAllAuthors(ctx)
	if expect.NoError(err) {
		expect.GreaterOrEqual(len(authors), 1)
	}
}

func TestReadTitlesByAuthor(t *testing.T) {
	expect := assert.New(t)
	idsAndTitles, err := service.ReadTitlesByAuthor(ctx, "Editorial Team", false, 10, "")
	if expect.NoError(err) {
		onlyTitles := v.Map(idsAndTitles, func(entry v.TextValue) string { return entry.Value })
		expect.Equal(len(onlyTitles), 3)
		expect.Contains(onlyTitles, book.Body.Title+": "+book.Body.Subtitle+" ("+string(book.Type)+")")
	}
}

func TestFilterTitlesByAuthor(t *testing.T) {
	expect := assert.New(t)
	filteredTitle, err := service.FilterTitlesByAuthor(ctx, "Editorial Team", "Happy", true)
	expectedTitle := []v.TextValue{
		{
			Key:   chapter2.ID,
			Value: chapter2.Body.Title + ": " + chapter2.Body.Subtitle + " (" + string(chapter2.Type) + ")",
		},
	}
	if expect.NoError(err) {
		expect.Equal(filteredTitle, expectedTitle)
	}
}

//------------------------------------------------------------------------------
// Content Titles by Editor
//------------------------------------------------------------------------------

func TestReadAllEditorIDs(t *testing.T) {
	expect := assert.New(t)
	allEditorIDs, err := service.ReadAllEditorIDs(ctx)
	if expect.NoError(err) && expect.NotEmpty(allEditorIDs) {
		expect.Equal(len(allEditorIDs), 1)
	}
}

func TestReadAllEditorNames(t *testing.T) {
	expect := assert.New(t)
	allEditorIDsAndNames, err := service.ReadAllEditorNames(ctx, true)
	if expect.NoError(err) {
		editorNames := v.Map(allEditorIDsAndNames, func(entry v.TextValue) string { return entry.Value })
		expect.Equal(editorNames, []string{"Editor-in-chief"})
	}
}

func TestFilterEditorNames(t *testing.T) {
	expect := assert.New(t)
	filteredEditorIDAndName, err := service.FilterEditorNames(ctx, "chief", true)
	expectedEditorIDAndName := []v.TextValue{
		{
			Key:   "9StQoyU5SxGDdGqF",
			Value: "Editor-in-chief",
		},
	}
	if expect.NoError(err) {
		expect.Equal(filteredEditorIDAndName, expectedEditorIDAndName)
	}
}

// content ID in this test keeps on changing when I try to hardcode it from book.json
// func TestReadTitlesByEditorID(t *testing.T) {
// 	expect := assert.New(t)
// 	contentIDsAndTitles, err := service.ReadTitlesByEditorID(ctx, book.EditorID, false, 10, "")
// 	expectedContentIDsAndTitles := []v.TextValue{
// 		{
// 			Key:   "9SoxTDVQoAc45HCi",
// 			Value: book.Body.Title + ": " + book.Body.Subtitle + " (" + string(book.Type) + ")",
// 		},
// 	}
// 	if expect.NoError(err) {
// 		expect.Equal(contentIDsAndTitles, expectedContentIDsAndTitles)
// 		fmt.Println(contentIDsAndTitles)
// 	}
// }

func TestReadAllTitlesByEditorID(t *testing.T) {
	expect := assert.New(t)
	contentIDsAndTitles, err := service.ReadAllTitlesByEditorID(ctx, book.EditorID, true)
	if expect.NoError(err) {
		onlyTitles := v.Map(contentIDsAndTitles, func(entry v.TextValue) string { return entry.Value })
		expect.Equal(onlyTitles, []string{book.Body.Title + ": " + book.Body.Subtitle + " (" + string(book.Type) + ")"})
	}
}

func TestFilterTitlesByEditorID(t *testing.T) {
	expect := assert.New(t)
	filteredContentIDsAndTitles, err := service.FilterTitlesByEditorID(ctx, book.EditorID, "easy", true)
	expectedContentIDsAndTitles := []v.TextValue{
		{
			Key:   book.ID,
			Value: book.Body.Title + ": " + book.Body.Subtitle + " (" + string(book.Type) + ")",
		},
	}
	if expect.NoError(err) {
		expect.Equal(filteredContentIDsAndTitles, expectedContentIDsAndTitles)
	}
}

//------------------------------------------------------------------------------
// Content Titles by Tag
//------------------------------------------------------------------------------

func TestReadAllTags(t *testing.T) {
	expect := assert.New(t)
	allTags, err := service.ReadAllTags(ctx)
	if expect.NoError(err) && expect.NotEmpty(allTags) {
		expect.Contains(allTags, "book")
		expect.Contains(allTags, "chapter2")
	}
}

func TestReadTitlesByTag(t *testing.T) {
	expect := assert.New(t)
	idsAndTitles, err := service.ReadTitlesByTag(ctx, "chapter1", false, 10, "")
	expectedTitle := []string{chapter1.Body.Title + ": " + chapter1.Body.Subtitle + " (" + string(chapter1.Type) + ")"}
	if expect.NoError(err) {
		onlyTitles := v.Map(idsAndTitles, func(entry v.TextValue) string { return entry.Value })
		expect.Equal(len(onlyTitles), 1)
		expect.Equal(onlyTitles, expectedTitle)
	}
}

func TestReadAllTitlesByTag(t *testing.T) {
	expect := assert.New(t)
	idsAndTitles, err := service.ReadAllTitlesByTag(ctx, "test", true)
	if expect.NoError(err) {
		onlyTitles := v.Map(idsAndTitles, func(entry v.TextValue) string { return entry.Value })
		expect.Equal(len(onlyTitles), 3)
	}
}

func TestFilterTitlesByTag(t *testing.T) {
	expect := assert.New(t)
	filteredTitle, err := service.FilterTitlesByTag(ctx, "test", "easy", true)
	expectedTitle := []v.TextValue{
		{
			Key:   book.ID,
			Value: book.Body.Title + ": " + book.Body.Subtitle + " (" + string(book.Type) + ")",
		},
	}
	if expect.NoError(err) {
		expect.Equal(filteredTitle, expectedTitle)
	}
}
