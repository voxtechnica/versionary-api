package content

import (
	"context"
	"fmt"
	"strings"
	"versionary-api/pkg/util"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/voxtechnica/tuid-go"
	v "github.com/voxtechnica/versionary"
)

//==============================================================================
// Content Table
//==============================================================================

// rowContents is a TableRow definition for Content versions
var rowContents = v.TableRow[Content]{
	RowName:      "contents_version",
	PartKeyName:  "id",
	PartKeyValue: func(c Content) string { return c.ID },
	PartKeyLabel: func(c Content) string { return c.Title() },
	SortKeyName:  "version_id",
	SortKeyValue: func(c Content) string { return c.VersionID },
	JsonValue:    func(c Content) []byte { return c.CompressedJSON() },
}

// rowContentTitlesType is a TableRow definition for searching/browsing Content titles by Type
var rowContentTitlesType = v.TableRow[Content]{
	RowName:      "content_titles_type",
	PartKeyName:  "type",
	PartKeyValue: func(c Content) string { return c.Type.String() },
	SortKeyName:  "id",
	SortKeyValue: func(c Content) string { return c.ID },
	TextValue:    func(c Content) string { return c.Title() },
}

// rowContentTitlesAuthor is a TableRow definition for searching/browsing Content titles by Author
var rowContentTitlesAuthor = v.TableRow[Content]{
	RowName:       "content_titles_author",
	PartKeyName:   "author",
	PartKeyValues: func(c Content) []string { return c.AuthorNames() },
	SortKeyName:   "id",
	SortKeyValue:  func(c Content) string { return c.ID },
	TextValue:     func(c Content) string { return c.Title() },
}

// rowContentTitlesEditor is a TableRow definition for searching/browsing Content titles by Editor
var rowContentTitlesEditor = v.TableRow[Content]{
	RowName:      "content_titles_editor",
	PartKeyName:  "editor_id",
	PartKeyValue: func(c Content) string { return c.EditorID },
	PartKeyLabel: func(c Content) string { return c.EditorName },
	SortKeyName:  "id",
	SortKeyValue: func(c Content) string { return c.ID },
	TextValue:    func(c Content) string { return c.Title() },
}

// rowContentTitlesTag is a TableRow definition for searching/browsing Content titles by Tag.
var rowContentTitlesTag = v.TableRow[Content]{
	RowName:       "content_titles_tag",
	PartKeyName:   "tag",
	PartKeyValues: func(c Content) []string { return c.Tags },
	SortKeyName:   "id",
	SortKeyValue:  func(c Content) string { return c.ID },
	TextValue:     func(c Content) string { return c.Title() },
}

// NewTable instantiates a new DynamoDB Content table.
func NewTable(dbClient *dynamodb.Client, env string) v.Table[Content] {
	if env == "" {
		env = "dev"
	}
	return v.Table[Content]{
		Client:     dbClient,
		EntityType: "Content",
		TableName:  "contents" + "_" + env,
		TTL:        false,
		EntityRow:  rowContents,
		IndexRows: map[string]v.TableRow[Content]{
			rowContentTitlesType.RowName:   rowContentTitlesType,
			rowContentTitlesAuthor.RowName: rowContentTitlesAuthor,
			rowContentTitlesEditor.RowName: rowContentTitlesEditor,
			rowContentTitlesTag.RowName:    rowContentTitlesTag,
		},
	}
}

// NewMemTable creates an in-memory Content table for testing purposes.
func NewMemTable(table v.Table[Content]) v.MemTable[Content] {
	return v.NewMemTable(table)
}

//==============================================================================
// Content Service
//==============================================================================

// Service is a service for managing Contents of various types.
type Service struct {
	EntityType string
	Table      v.TableReadWriter[Content]
}

// NewService creates a new Content service backed by a Versionary Table for the specified environment.
func NewService(dbClient *dynamodb.Client, env string) Service {
	table := NewTable(dbClient, env)
	return Service{
		EntityType: table.EntityType,
		Table:      table,
	}
}

// NewMockService creates a new Content service backed by an in-memory table for testing purposes.
func NewMockService(env string) Service {
	table := NewMemTable(NewTable(nil, env))
	return Service{
		EntityType: table.EntityType,
		Table:      table,
	}
}

// filterTitles returns a filtered list of Content IDs and Titles for a specified row and partition key value.
// The case-insensitive contains query is split into words, and the words are compared with the value in the TextValue.
// If anyMatch is true, then a TextValue is included in the results if any of the words are found (OR filter).
// If anyMatch is false, then the TextValue must contain all the words in the query string (AND filter).
// The filtered results are sorted alphabetically by value, not by ID.
func (s Service) filterTitles(ctx context.Context, row v.TableRow[Content], key string, contains string, anyMatch bool) ([]v.TextValue, error) {
	filter, err := util.ContainsFilter(contains, anyMatch)
	if err != nil {
		return []v.TextValue{}, err
	}
	return s.Table.FilterTextValues(ctx, row, key, filter)
}

//------------------------------------------------------------------------------
// Content Versions
//------------------------------------------------------------------------------

// Create a Content in the Content table.
func (s Service) Create(ctx context.Context, c Content) (Content, []string, error) {
	t := tuid.NewID()
	at, _ := t.Time()
	c.ID = t.String()
	c.CreatedAt = at
	c.VersionID = t.String()
	c.UpdatedAt = at
	c = c.Sanitize()
	c.WordCount = c.Body.WordCount()
	c.ImageCount = c.Body.ImageCount()
	c.LinkCount = c.Body.LinkCount()
	c.SectionCount = c.Body.SectionCount()
	problems := c.Validate()
	if len(problems) > 0 {
		return c, problems, fmt.Errorf("error creating %s %s: invalid field(s): %s", s.EntityType, c.ID, strings.Join(problems, ", "))
	}
	err := s.Table.WriteEntity(ctx, c)
	if err != nil {
		return c, problems, fmt.Errorf("error creating %s %s %s: %w", s.EntityType, c.ID, c.Title(), err)
	}
	return c, problems, nil
}

// Update a Content in the Content table. If a previous version does not exist, the Content is created.
func (s Service) Update(ctx context.Context, c Content) (Content, []string, error) {
	t := tuid.NewID()
	at, _ := t.Time()
	c.VersionID = t.String()
	c.UpdatedAt = at
	c = c.Sanitize()
	c.WordCount = c.Body.WordCount()
	c.ImageCount = c.Body.ImageCount()
	c.LinkCount = c.Body.LinkCount()
	c.SectionCount = c.Body.SectionCount()
	problems := c.Validate()
	if len(problems) > 0 {
		return c, problems, fmt.Errorf("error updating %s %s: invalid field(s): %s", s.EntityType, c.ID, strings.Join(problems, ", "))
	}
	return c, problems, s.Table.UpdateEntity(ctx, c)
}

// Write a Content to the Content table. This method assumes that the Content has all the required fields.
// It would most likely be used for "refreshing" the index rows in the Content table.
func (s Service) Write(ctx context.Context, c Content) (Content, error) {
	return c, s.Table.WriteEntity(ctx, c)
}

// Delete a Content from the Content table. The deleted Content is returned.
func (s Service) Delete(ctx context.Context, id string) (Content, error) {
	return s.Table.DeleteEntityWithID(ctx, id)
}

// Delete a Content Version from the Content table. The deleted Content is returned.
func (s Service) DeleteVersion(ctx context.Context, id string, versionID string) (Content, error) {
	return s.Table.DeleteEntityVersionWithID(ctx, id, versionID)
}

// Exists checks if a Content exists in the Content table.
func (s Service) Exists(ctx context.Context, id string) bool {
	return s.Table.EntityExists(ctx, id)
}

// Read a specified Content from the Content table.
func (s Service) Read(ctx context.Context, id string) (Content, error) {
	return s.Table.ReadEntity(ctx, id)
}

// ReadAsJSON gets a specified Content from the Content table, serialized as JSON.
func (s Service) ReadAsJSON(ctx context.Context, id string) ([]byte, error) {
	return s.Table.ReadEntityAsJSON(ctx, id)
}

// VersionExists checks if a specified Content version exists in the Content table.
func (s Service) VersionExists(ctx context.Context, id, versionID string) bool {
	return s.Table.EntityVersionExists(ctx, id, versionID)
}

// ReadVersion gets a specified Content version from the Content table.
func (s Service) ReadVersion(ctx context.Context, id, versionID string) (Content, error) {
	return s.Table.ReadEntityVersion(ctx, id, versionID)
}

// ReadVersionAsJSON gets a specified Content version from the Content table, serialized as JSON.
func (s Service) ReadVersionAsJSON(ctx context.Context, id, versionID string) ([]byte, error) {
	return s.Table.ReadEntityVersionAsJSON(ctx, id, versionID)
}

// ReadVersions returns paginated versions of the specified Content.
// Sorting is chronological (or reverse). The offset is the last ID returned in a previous request.
func (s Service) ReadVersions(ctx context.Context, id string, reverse bool, limit int, offset string) ([]Content, error) {
	return s.Table.ReadEntityVersions(ctx, id, reverse, limit, offset)
}

// ReadVersionsAsJSON returns paginated versions of the specified Content, serialized as JSON.
// Sorting is chronological (or reverse). The offset is the last ID returned in a previous request.
func (s Service) ReadVersionsAsJSON(ctx context.Context, id string, reverse bool, limit int, offset string) ([]byte, error) {
	return s.Table.ReadEntityVersionsAsJSON(ctx, id, reverse, limit, offset)
}

// ReadAllVersions returns all versions of the specified Content in chronological order.
// Caution: this may be a LOT of data!
func (s Service) ReadAllVersions(ctx context.Context, id string) ([]Content, error) {
	return s.Table.ReadAllEntityVersions(ctx, id)
}

// ReadAllVersionsAsJSON returns all versions of the specified Content, serialized as JSON.
// Caution: this may be a LOT of data!
func (s Service) ReadAllVersionsAsJSON(ctx context.Context, id string) ([]byte, error) {
	return s.Table.ReadAllEntityVersionsAsJSON(ctx, id)
}

// ReadContentIDs returns a paginated list of Content IDs in the Content table.
// Sorting is chronological (or reverse). The offset is the last ID returned in a previous request.
func (s Service) ReadContentIDs(ctx context.Context, reverse bool, limit int, offset string) ([]string, error) {
	return s.Table.ReadEntityIDs(ctx, reverse, limit, offset)
}

// ReadAllContentIDs returns all Content IDs in the Content table.
// Caution: this may be a LOT of data!
func (s Service) ReadAllContentIDs(ctx context.Context) ([]string, error) {
	return s.Table.ReadAllEntityIDs(ctx)
}

// ReadTitles returns a paginated list of Content IDs and titles in the Content table.
// Sorting is chronological (or reverse). The offset is the last ID returned in a previous request.
func (s Service) ReadTitles(ctx context.Context, reverse bool, limit int, offset string) ([]v.TextValue, error) {
	return s.Table.ReadEntityLabels(ctx, reverse, limit, offset)
}

// ReadAllTitles returns all Content IDs and titles in the Content table.
// Caution: this may be a LOT of data!
func (s Service) ReadAllTitles(ctx context.Context, sortByValue bool) ([]v.TextValue, error) {
	return s.Table.ReadAllEntityLabels(ctx, sortByValue)
}

// FilterTitles returns a filtered list of Content IDs and titles in the Content table.
// The case-insensitive contains query is split into words, and the words are compared with the value in the TextValue.
// If anyMatch is true, then a TextValue is included in the results if any of the words are found (OR filter).
// If anyMatch is false, then the TextValue must contain all the words in the query string (AND filter).
// The filtered results are sorted alphabetically by value, not by ID.
func (s Service) FilterTitles(ctx context.Context, contains string, anyMatch bool) ([]v.TextValue, error) {
	filter, err := util.ContainsFilter(contains, anyMatch)
	if err != nil {
		return []v.TextValue{}, err
	}
	return s.Table.FilterEntityLabels(ctx, filter)
}

// ReadContents returns a paginated list of Contents in the Content table.
// Sorting is chronological (or reverse). The offset is the last ID returned in a previous request.
// Note that this is a best-effort attempt to return the requested Contents, retrieved individually, in parallel.
// It is probably not the best way to page through a large Content table.
func (s Service) ReadContents(ctx context.Context, reverse bool, limit int, offset string) []Content {
	ids, err := s.Table.ReadEntityIDs(ctx, reverse, limit, offset)
	if err != nil {
		return []Content{}
	}
	return s.Table.ReadEntities(ctx, ids)
}

//------------------------------------------------------------------------------
// Content Titles by Type
//------------------------------------------------------------------------------

// ReadAllTypes returns all Content types in the Content table.
func (s Service) ReadAllTypes(ctx context.Context) ([]string, error) {
	return s.Table.ReadAllPartKeyValues(ctx, rowContentTitlesType)
}

// ReadTitlesByType returns a paginated list of Content IDs and Titles for a given Content type.
func (s Service) ReadTitlesByType(ctx context.Context, t string, reverse bool, limit int, offset string) ([]v.TextValue, error) {
	return s.Table.ReadTextValues(ctx, rowContentTitlesType, t, reverse, limit, offset)
}

// ReadAllTitlesByType returns all Content IDs and Titles for a given Content type.
func (s Service) ReadAllTitlesByType(ctx context.Context, t string, sortByValue bool) ([]v.TextValue, error) {
	return s.Table.ReadAllTextValues(ctx, rowContentTitlesType, t, sortByValue)
}

// FilterTitlesByType returns a filtered list of Content IDs and Titles for a given Content type.
// The case-insensitive contains query is split into words, and the words are compared with the value in the TextValue.
// If anyMatch is true, then a TextValue is included in the results if any of the words are found (OR filter).
// If anyMatch is false, then the TextValue must contain all the words in the query string (AND filter).
// The filtered results are sorted alphabetically by value, not by ID.
func (s Service) FilterTitlesByType(ctx context.Context, t string, contains string, anyMatch bool) ([]v.TextValue, error) {
	return s.filterTitles(ctx, rowContentTitlesType, t, contains, anyMatch)
}

//------------------------------------------------------------------------------
// Content Titles by Author
//------------------------------------------------------------------------------

// ReadAllAuthors returns all Content authors in the Content table.
func (s Service) ReadAllAuthors(ctx context.Context) ([]string, error) {
	return s.Table.ReadAllPartKeyValues(ctx, rowContentTitlesAuthor)
}

// ReadTitlesByAuthor returns a paginated list of Content IDs and Titles for a given Content author.
func (s Service) ReadTitlesByAuthor(ctx context.Context, a string, reverse bool, limit int, offset string) ([]v.TextValue, error) {
	return s.Table.ReadTextValues(ctx, rowContentTitlesAuthor, a, reverse, limit, offset)
}

// ReadAllTitlesByAuthor returns all Content IDs and Titles for a given Content author.
func (s Service) ReadAllTitlesByAuthor(ctx context.Context, a string, sortByValue bool) ([]v.TextValue, error) {
	return s.Table.ReadAllTextValues(ctx, rowContentTitlesAuthor, a, sortByValue)
}

// FilterTitlesByAuthor returns a filtered list of Content IDs and Titles for a given Content author.
// The case-insensitive contains query is split into words, and the words are compared with the value in the TextValue.
// If anyMatch is true, then a TextValue is included in the results if any of the words are found (OR filter).
// If anyMatch is false, then the TextValue must contain all the words in the query string (AND filter).
// The filtered results are sorted alphabetically by value, not by ID.
func (s Service) FilterTitlesByAuthor(ctx context.Context, a string, contains string, anyMatch bool) ([]v.TextValue, error) {
	return s.filterTitles(ctx, rowContentTitlesAuthor, a, contains, anyMatch)
}

//------------------------------------------------------------------------------
// Content Titles by Editor
//------------------------------------------------------------------------------

// ReadAllEditorIDs returns all Content editor IDs in the Content table.
func (s Service) ReadAllEditorIDs(ctx context.Context) ([]string, error) {
	return s.Table.ReadAllPartKeyValues(ctx, rowContentTitlesEditor)
}

// ReadAllEditorNames returns all Content editors (userID and name) in the Content table.
func (s Service) ReadAllEditorNames(ctx context.Context, sortByValue bool) ([]v.TextValue, error) {
	return s.Table.ReadAllPartKeyLabels(ctx, rowContentTitlesEditor, sortByValue)
}

// FilterEditorNames returns a filtered list of Content editor IDs and names.
// The case-insensitive contains query is split into words, and the words are compared with the value in the TextValue.
// If anyMatch is true, then a TextValue is included in the results if any of the words are found (OR filter).
// If anyMatch is false, then the TextValue must contain all the words in the query string (AND filter).
// The filtered results are sorted alphabetically by value, not by ID.
func (s Service) FilterEditorNames(ctx context.Context, contains string, anyMatch bool) ([]v.TextValue, error) {
	filter, err := util.ContainsFilter(contains, anyMatch)
	if err != nil {
		return []v.TextValue{}, err
	}
	return s.Table.FilterPartKeyLabels(ctx, rowContentTitlesEditor, filter)
}

// ReadTitlesByEditorID returns a paginated list of Content IDs and Titles for a given Content editor.
func (s Service) ReadTitlesByEditorID(ctx context.Context, editorID string, reverse bool, limit int, offset string) ([]v.TextValue, error) {
	return s.Table.ReadTextValues(ctx, rowContentTitlesEditor, editorID, reverse, limit, offset)
}

// ReadAllTitlesByEditorID returns all Content IDs and Titles for a given Content editor.
func (s Service) ReadAllTitlesByEditorID(ctx context.Context, editorID string, sortByValue bool) ([]v.TextValue, error) {
	return s.Table.ReadAllTextValues(ctx, rowContentTitlesEditor, editorID, sortByValue)
}

// FilterTitlesByEditorID returns a filtered list of Content IDs and Titles for a given Content editor.
// The case-insensitive contains query is split into words, and the words are compared with the value in the TextValue.
// If anyMatch is true, then a TextValue is included in the results if any of the words are found (OR filter).
// If anyMatch is false, then the TextValue must contain all the words in the query string (AND filter).
// The filtered results are sorted alphabetically by value, not by ID.
func (s Service) FilterTitlesByEditorID(ctx context.Context, editorID string, contains string, anyMatch bool) ([]v.TextValue, error) {
	return s.filterTitles(ctx, rowContentTitlesEditor, editorID, contains, anyMatch)
}

//------------------------------------------------------------------------------
// Content Titles by Tag
//------------------------------------------------------------------------------

// ReadAllTags returns all Content tags in the Content table.
func (s Service) ReadAllTags(ctx context.Context) ([]string, error) {
	return s.Table.ReadAllPartKeyValues(ctx, rowContentTitlesTag)
}

// ReadTitlesByTag returns a paginated list of Content IDs and Titles for a given Content tag.
func (s Service) ReadTitlesByTag(ctx context.Context, tag string, reverse bool, limit int, offset string) ([]v.TextValue, error) {
	return s.Table.ReadTextValues(ctx, rowContentTitlesTag, tag, reverse, limit, offset)
}

// ReadAllTitlesByTag returns all Content IDs and Titles for a given Content tag.
func (s Service) ReadAllTitlesByTag(ctx context.Context, tag string, sortByValue bool) ([]v.TextValue, error) {
	return s.Table.ReadAllTextValues(ctx, rowContentTitlesTag, tag, sortByValue)
}

// FilterTitlesByTag returns a filtered list of Content IDs and Titles for a given Content tag.
// The case-insensitive contains query is split into words, and the words are compared with the value in the TextValue.
// If anyMatch is true, then a TextValue is included in the results if any of the words are found (OR filter).
// If anyMatch is false, then the TextValue must contain all the words in the query string (AND filter).
// The filtered results are sorted alphabetically by value, not by ID.
func (s Service) FilterTitlesByTag(ctx context.Context, tag string, contains string, anyMatch bool) ([]v.TextValue, error) {
	return s.filterTitles(ctx, rowContentTitlesTag, tag, contains, anyMatch)
}
