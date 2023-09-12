package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/voxtechnica/tuid-go"
	"github.com/voxtechnica/versionary"

	"versionary-api/pkg/content"
	"versionary-api/pkg/util"
)

func TestContentCRUD(t *testing.T) {
	expect := assert.New(t)
	// Create a new Content object
	var con content.Content
	j, err := json.Marshal(content.Content{
		Type: content.BOOK,
		Body: content.Section{
			Title:    "CRUD Test 1",
			Subtitle: "CRUD Test Subtitle 1",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/v1/contents", bytes.NewBuffer(j))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusCreated, w.Code, "HTTP Status Code")
		if expect.NoError(json.NewDecoder(w.Body).Decode(&con), "Decode JSON Content") {
			expect.True(tuid.IsValid(tuid.TUID(con.ID)), "Valid TUID")
			expect.Equal(con.ID, con.VersionID, "ID and Version ID match")
		}
	}
	// Read the Content object
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/contents/"+con.ID, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var con2 content.Content
		if expect.NoError(json.NewDecoder(w.Body).Decode(&con2), "Decode JSON Content") {
			expect.Equal(con, con2, "Content")
		}
	}

	// Update the Content object
	w = httptest.NewRecorder()
	con.Comment = "This is a test comment 1"
	body, _ := json.Marshal(con)
	req, err = http.NewRequest("PUT", "/v1/contents/"+con.ID, bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var con2 content.Content
		if expect.NoError(json.NewDecoder(w.Body).Decode(&con2), "Decode JSON Content") {
			expect.Equal(con.ID, con2.ID, "Content ID")
			expect.Equal(con.Comment, con2.Comment, "Content Comment")
		}
	}

	// Delete the Content object
	w = httptest.NewRecorder()
	req, err = http.NewRequest("DELETE", "/v1/contents/"+con.ID, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var con2 content.Content
		if expect.NoError(json.NewDecoder(w.Body).Decode(&con2), "Decode JSON Content") {
			expect.Equal(con.ID, con2.ID, "Content ID")
			expect.Equal(con.EditorName, con2.EditorName, "Editor Name")
		}
	}
}

func TestCreateContent(t *testing.T) {
	expect := assert.New(t)
	// Create a new Content object: happy path
	var con content.Content
	j, err := json.Marshal(content.Content{
		Type: content.ARTICLE,
		Body: content.Section{
			Title:    "Article 1",
			Subtitle: "Article 1 Subtitle",
			Text:     "This is the text of Article 1.",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/v1/contents", bytes.NewBuffer(j))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusCreated, w.Code, "HTTP Status Code")
		if expect.NoError(json.NewDecoder(w.Body).Decode(&con), "Decode JSON Content") {
			expect.True(tuid.IsValid(tuid.TUID(con.ID)), "Valid TUID")
			expect.Equal(con.ID, con.VersionID, "ID and Version ID match")
		}
	}
	// Create a new Content object: missing required fields
	w = httptest.NewRecorder()
	req, err = http.NewRequest("POST", "/v1/contents", strings.NewReader(""))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code, "HTTP Status Code")
	}
	var e APIEvent
	if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON APIEvent") {
		expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
		expect.Equal(http.StatusBadRequest, e.Code, "Event Code")
		expect.Contains(e.Message, "invalid JSON", "Event Message")
	}
}

func TestReadContents(t *testing.T) {
	expect := assert.New(t)
	// Read all Content
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/contents", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var con []content.Content
		if expect.NoError(json.NewDecoder(w.Body).Decode(&con), "Decode JSON Content") {
			expect.GreaterOrEqual(len(con), 1, "Content Count")
		}
	}

	// Read all Content: missing authorization token
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/contents", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusUnauthorized, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusUnauthorized, e.Code, "Event Code")
			expect.Contains(e.Message, "unauthenticated", "Event Message")
		}
	}

	// Read all Content: missing admin role
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/contents", nil)
	req.Header.Set("Authorization", "Bearer "+regularToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusForbidden, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusForbidden, e.Code, "Event Code")
			expect.Contains(e.Message, "unauthorized", "Event Message")
		}
	}

	// Read all Content: invalid parameter
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/contents?limit=invalid", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusBadRequest, e.Code, "Event Code")
			expect.Contains(e.Message, "invalid parameter", "Event Message")
		}
	}
}

func TestReadContent(t *testing.T) {
	expect := assert.New(t)
	// Read a Content object by ID
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/contents/"+contentOne.ID, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var con content.Content
		if expect.NoError(json.NewDecoder(w.Body).Decode(&con), "Decode JSON Content") {
			expect.Equal(contentOne, con, "Content")
			expect.Equal(contentOne.ID, con.ID, "Content ID")
			expect.Equal(contentOne.VersionID, con.VersionID, "Content VersionID")
			expect.Equal(contentOne.Type, con.Type, "Content Type")
			expect.Equal(contentOne.Body, con.Body, "Content Content")
		}
	}

	// Read a Content object by invalid ID
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/contents/fake_id", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusBadRequest, e.Code, "Event Code")
			expect.Contains(e.Message, "bad request: invalid path parameter ID: entity ID fake_id is invalid", "Event Message")
		}
	}

	// Read a non-existent Content object by ID to get a 404
	w = httptest.NewRecorder()
	contentID := tuid.NewID().String()
	req, err = http.NewRequest("GET", "/v1/contents/"+contentID, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusNotFound, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusNotFound, e.Code, "Event Code")
			expect.Contains(e.Message, "not found: Content-"+contentID, "Event Message")
		}
	}
}

func TestContentExists(t *testing.T) {
	expect := assert.New(t)
	// Check whether a Content object exists by ID
	w := httptest.NewRecorder()
	req, err := http.NewRequest("HEAD", "/v1/contents/"+contentTwo.ID, nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusNoContent, w.Code, "HTTP Status Code")
	}

	// Check an invalid Content ID
	w = httptest.NewRecorder()
	req, err = http.NewRequest("HEAD", "/v1/contents/bad_id", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code, "HTTP Status Code")
	}

	// Check Content ID that doesn't exist
	w = httptest.NewRecorder()
	contentID := tuid.NewID().String()
	req, err = http.NewRequest("HEAD", "/v1/contents/"+contentID, nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusNotFound, w.Code, "HTTP Status Code")
	}
}

func TestReadContentVersions(t *testing.T) {
	expect := assert.New(t)
	// Read all versions of a Content object by ID
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/contents/"+contentOne.ID+"/versions", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var con []content.Content
		if expect.NoError(json.NewDecoder(w.Body).Decode(&con), "Decode JSON Content") {
			expect.GreaterOrEqual(len(con), 1, "Content Count")
		}
	}

	// Read versions of an invalid Content ID
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/contents/fake_id/versions", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusBadRequest, e.Code, "Event Code")
			expect.Contains(e.Message, "bad request: invalid path parameter ID: entity ID fake_id is invalid", "Event Message")
		}
	}

	// Read all versions of a Content: missing authorization token
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/contents/"+contentOne.ID+"/versions", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusUnauthorized, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusUnauthorized, e.Code, "Event Code")
			expect.Contains(e.Message, "unauthenticated", "Event Message")
		}
	}

	// Read all versions of a Content: missing admin role
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/contents/"+contentOne.ID+"/versions", nil)
	req.Header.Set("Authorization", "Bearer "+regularToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusForbidden, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusForbidden, e.Code, "Event Code")
			expect.Contains(e.Message, "unauthorized", "Event Message")
		}
	}

	// Read all versions of a Content that doesn't exist
	w = httptest.NewRecorder()
	contentID := tuid.NewID().String()
	req, err = http.NewRequest("GET", "/v1/contents/"+contentID+"/versions", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusNotFound, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusNotFound, e.Code, "Event Code")
			expect.Contains(e.Message, "not found: Content-"+contentID, "Event Message")
		}
	}
}

func TestContentVersionExists(t *testing.T) {
	expect := assert.New(t)
	// Check a Content object by ID and VersionID
	w := httptest.NewRecorder()
	req, err := http.NewRequest("HEAD", "/v1/contents/"+contentOne.ID+"/versions/"+contentOne.VersionID, nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusNoContent, w.Code, "HTTP Status Code")
	}

	// Check a Content object that doesn't exist
	contentID := tuid.NewID().String()
	w = httptest.NewRecorder()
	req, err = http.NewRequest("HEAD", "/v1/contents/"+contentID+"/versions/"+contentID, nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusNotFound, w.Code, "HTTP Status Code")
	}

	// Check a Content object with an invalid version ID
	w = httptest.NewRecorder()
	req, err = http.NewRequest("HEAD", "/v1/contents/fake_id/versions/fake_id", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code, "HTTP Status Code")
	}
}

func TestReadContentVersion(t *testing.T) {
	expect := assert.New(t)
	// Read a known Content version
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/contents/"+contentOne.ID+"/versions/"+contentOne.VersionID, nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var con content.Content
		if expect.NoError(json.NewDecoder(w.Body).Decode(&con), "Decode JSON Organization") {
			expect.Equal(contentOne, con, "Content")
		}
	}

	// Read a Content version: invalid Content ID
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/contents/fake_id/versions/"+contentOne.VersionID, nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusBadRequest, e.Code, "Event Code")
			expect.Contains(e.Message, "bad request: invalid path parameter ID: entity ID fake_id is invalid", "Event Message")
		}
	}

	// Read a Content version that doesn't exist
	w = httptest.NewRecorder()
	contentID := tuid.NewID().String()
	req, err = http.NewRequest("GET", "/v1/contents/"+contentID+"/versions/"+contentID, nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusNotFound, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusNotFound, e.Code, "Event Code")
			expect.Contains(e.Message, "not found: Content-"+contentID, "Event Message")
		}
	}
}

func TestUpdateContent(t *testing.T) {
	expect := assert.New(t)
	// Update a Content object happy path covered in TestContentCRUD
	// Update a Content object: invalid JSON
	w := httptest.NewRecorder()
	req, err := http.NewRequest("PUT", "/v1/contents/"+contentOne.ID, strings.NewReader("invalid JSON"))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusBadRequest, e.Code, "Event Code")
			expect.Contains(e.Message, "invalid JSON", "Event Message")
		}
	}

	// Update a Content object: missing authorization token
	w = httptest.NewRecorder()
	body := `{"comment":"This is a test comment 2"}`
	req, err = http.NewRequest("PUT", "/v1/contents/"+contentOne.ID, strings.NewReader(body))
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusUnauthorized, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusUnauthorized, e.Code, "Event Code")
			expect.Contains(e.Message, "unauthenticated", "Event Message")
		}
	}

	// Update a Content object: missing admin role
	w = httptest.NewRecorder()
	body = `{"comment":"This is a test comment 3"}`
	req, err = http.NewRequest("PUT", "/v1/contents/"+contentOne.ID, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+regularToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusForbidden, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusForbidden, e.Code, "Event Code")
			expect.Contains(e.Message, "unauthorized", "Event Message")
		}
	}

	// Update a Content object: missing type
	w = httptest.NewRecorder()
	body = `{"id":"` + contentOne.ID + `","type":""}`
	req, err = http.NewRequest("PUT", "/v1/contents/"+contentOne.ID, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusUnprocessableEntity, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusUnprocessableEntity, e.Code, "Event Code")
			expect.Contains(e.Message, "missing", "Event Message")
		}
	}
}

func TestDeleteContent(t *testing.T) {
	expect := assert.New(t)
	// Delete a Content object happy path covered in TestContentCRUD
	// Delete a Content object: invalid path parameter
	w := httptest.NewRecorder()
	req, err := http.NewRequest("DELETE", "/v1/contents/fake_id", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusBadRequest, e.Code, "Event Code")
			expect.Contains(e.Message, "bad request: invalid path parameter ID: entity ID fake_id is invalid", "Event Message")
		}
	}

	// Delete a Content object: missing authorization token
	w = httptest.NewRecorder()
	req, err = http.NewRequest("DELETE", "/v1/contents/"+contentOne.ID, nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusUnauthorized, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusUnauthorized, e.Code, "Event Code")
			expect.Contains(e.Message, "unauthenticated", "Event Message")
		}
	}

	// Delete a Content object: missing admin role
	w = httptest.NewRecorder()
	req, err = http.NewRequest("DELETE", "/v1/contents/"+contentOne.ID, nil)
	req.Header.Set("Authorization", "Bearer "+regularToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusForbidden, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusForbidden, e.Code, "Event Code")
			expect.Contains(e.Message, "unauthorized", "Event Message")
		}
	}

	// Delete a Content object: not found
	w = httptest.NewRecorder()
	contentID := tuid.NewID().String()
	req, err = http.NewRequest("DELETE", "/v1/contents/"+contentID, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusNotFound, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusNotFound, e.Code, "Event Code")
			expect.Contains(e.Message, "not found: Content-"+contentID, "Event Message")
		}
	}
}

func TestReadContentTypes(t *testing.T) {
	expect := assert.New(t)
	// Read all Content types
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/content_types", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var types []string
		if expect.NoError(json.NewDecoder(w.Body).Decode(&types), "Decode JSON Content Types") {
			expect.GreaterOrEqual(len(types), 2, "Content Type Count")
			expect.Contains(types, string(contentOne.Type), "Specified Content Type")
		}
	}

	// Read all Content types: missing authorization token
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/content_types", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusUnauthorized, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusUnauthorized, e.Code, "Event Code")
			expect.Contains(e.Message, "unauthenticated", "Event Message")
		}
	}

	// Read all Content types: missing admin role
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/content_types", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	req.Header.Set("Authorization", "Bearer "+regularToken)
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusForbidden, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusForbidden, e.Code, "Event Code")
			expect.Contains(e.Message, "unauthorized", "Event Message")
		}
	}
}

func TestReadContentAuthors(t *testing.T) {
	expect := assert.New(t)
	// Read all Content authors
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/content_authors", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var authors []string
		if expect.NoError(json.NewDecoder(w.Body).Decode(&authors), "Decode JSON Content Authors") {
			expect.GreaterOrEqual(len(authors), 2, "Content Authors Count")
			expect.Contains(authors, contentOne.Authors[0].Name, "Specified Content Author")
		}
	}

	// Read all Content authors: missing authorization token
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/content_authors", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusUnauthorized, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusUnauthorized, e.Code, "Event Code")
			expect.Contains(e.Message, "unauthenticated", "Event Message")
		}
	}

	// Read all Content authors: missing admin role
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/content_authors", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	req.Header.Set("Authorization", "Bearer "+regularToken)
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusForbidden, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusForbidden, e.Code, "Event Code")
			expect.Contains(e.Message, "unauthorized", "Event Message")
		}
	}
}

func TestReadContentEditors(t *testing.T) {
	expect := assert.New(t)
	// Read all Content editors
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/content_editors", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var editors []versionary.TextValue
		if expect.NoError(json.NewDecoder(w.Body).Decode(&editors), "Decode JSON Content Editors") {
			expect.GreaterOrEqual(len(editors), 1, "Content Editors Count")
			m := util.TextValuesMap(editors)
			name, ok := m[adminUser.ID]
			expect.True(ok, "Specified Content Editor")
			expect.Equal(adminUser.FullName(), name, "Specified Content Editor")
		}
	}

	// Read all Content editors: missing authorization token
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/content_editors", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusUnauthorized, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusUnauthorized, e.Code, "Event Code")
			expect.Contains(e.Message, "unauthenticated", "Event Message")
		}
	}

	// Read all Content editors: missing admin role
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/content_editors", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	req.Header.Set("Authorization", "Bearer "+regularToken)
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusForbidden, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusForbidden, e.Code, "Event Code")
			expect.Contains(e.Message, "unauthorized", "Event Message")
		}
	}
}

func TestReadContentTags(t *testing.T) {
	expect := assert.New(t)
	// Read all Content tags
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/content_tags", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var tags []string
		if expect.NoError(json.NewDecoder(w.Body).Decode(&tags), "Decode JSON Content Tags") {
			expect.GreaterOrEqual(len(tags), 5, "Content Tags Count")
			expect.Contains(tags, contentOne.Tags[0], "Specified Content Tag")
		}
	}

	// Read all Content tags: missing authorization token
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/content_tags", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusUnauthorized, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusUnauthorized, e.Code, "Event Code")
			expect.Contains(e.Message, "unauthenticated", "Event Message")
		}
	}

	// Read all Content tags: missing admin role
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/content_tags", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	req.Header.Set("Authorization", "Bearer "+regularToken)
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusForbidden, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusForbidden, e.Code, "Event Code")
			expect.Contains(e.Message, "unauthorized", "Event Message")
		}
	}
}

func TestReadContentTitles(t *testing.T) {
	expect := assert.New(t)
	// Read all Content titles
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/content_titles?sorted=true", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var titles []versionary.TextValue
		if expect.NoError(json.NewDecoder(w.Body).Decode(&titles), "Decode JSON Content Titles") {
			expect.GreaterOrEqual(len(titles), 2, "Content Titles Count")
		}
	}

	// Read Content titles by type
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/content_titles?type="+contentOne.Type.String(), nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var titles []versionary.TextValue
		if expect.NoError(json.NewDecoder(w.Body).Decode(&titles), "Decode JSON Content Titles") {
			expect.GreaterOrEqual(len(titles), 1, "Content Titles Count")
			m := util.TextValuesMap(titles)
			title, ok := m[contentOne.ID]
			expect.True(ok, "Content Title Exists")
			expect.Contains(title, contentOne.Title(), "Content Title")
		}
	}

	// Read Content titles by tag
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/content_titles?tag=v2", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var titles []versionary.TextValue
		if expect.NoError(json.NewDecoder(w.Body).Decode(&titles), "Decode JSON Content Titles") {
			expect.Equal(2, len(titles), "Content Titles Count")
		}
	}

	// Read Content titles by author
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/content_titles?author=Test Author 1", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var titles []versionary.TextValue
		if expect.NoError(json.NewDecoder(w.Body).Decode(&titles), "Decode JSON Content Titles") {
			expect.Equal(1, len(titles), "Content Titles Count")
		}
	}

	// Read Content titles by editor ID
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/content_titles?editor="+regularUser.ID, nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var titles []versionary.TextValue
		if expect.NoError(json.NewDecoder(w.Body).Decode(&titles), "Decode JSON Content Titles") {
			expect.Equal(0, len(titles), "Content Titles Count")
		}
	}

	// Read Content titles by invalid editor ID
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/content_titles?editor=fake_id", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusBadRequest, e.Code, "Event Code")
			expect.Contains(e.Message, "invalid editor ID: fake_id", "Event Message")
		}
	}

	// Read Content titles by search term
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/content_titles?search=learn&any=true", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var titles []versionary.TextValue
		if expect.NoError(json.NewDecoder(w.Body).Decode(&titles), "Decode JSON Content Titles") {
			expect.GreaterOrEqual(len(titles), 1, "Content Titles Count")
		}
	}

	// Read Content titles by search terms
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/content_titles?search=learn chapter&any=true", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var titles []versionary.TextValue
		if expect.NoError(json.NewDecoder(w.Body).Decode(&titles), "Decode JSON Content Titles") {
			expect.Equal(2, len(titles), "Content Titles Count")
		}
	}

	// Read Content titles by limit parameter
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/content_titles?limit=2", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var titles []versionary.TextValue
		if expect.NoError(json.NewDecoder(w.Body).Decode(&titles), "Decode JSON Content Titles") {
			expect.Equal(2, len(titles), "Content Titles Count")
		}
	}

	// Read Content titles by invalid reverse parameter
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/content_titles?reverse=invalid", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusBadRequest, e.Code, "Event Code")
			expect.Contains(e.Message, "invalid syntax", "Event Message")
		}
	}

	// Read Content titles: missing authorization token
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/content_titles", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusUnauthorized, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusUnauthorized, e.Code, "Event Code")
			expect.Contains(e.Message, "unauthenticated", "Event Message")
		}
	}

	// Read Content titles: missing admin role
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/content_titles", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	req.Header.Set("Authorization", "Bearer "+regularToken)
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusForbidden, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusForbidden, e.Code, "Event Code")
			expect.Contains(e.Message, "unauthorized", "Event Message")
		}
	}
}
