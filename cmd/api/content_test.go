package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/voxtechnica/tuid-go"

	"versionary-api/pkg/content"
)

func TestContentCRUD(t *testing.T) {
	expect := assert.New(t)
	// Create a new Content object
	var con content.Content
	j, err := json.Marshal(content.Content{
		Type: content.BOOK,
		Content: content.Section{
			Title:    "Test Book 1",
			Subtitle: "A Test Book Subtitle 1",
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
		fmt.Println("BODY: ", w.Body)
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
