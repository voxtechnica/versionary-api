package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"versionary-api/pkg/token"

	"github.com/stretchr/testify/assert"
)

func _TestTokenCRUD(t *testing.T) {

}

func _TestCreateTokens(t *testing.T) {

}

func TestReadTokens(t *testing.T) {
	expect := assert.New(t)
	// Read tokens by admin user
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/tokens", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var tokens []token.Token
		if expect.NoError(json.NewDecoder(w.Body).Decode(&tokens), "Decode JSON Token") {
			expect.Equal(1, len(tokens), "Token Count")
		}
	}

	// Read tokens by regular user
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/tokens", nil)
	req.Header.Set("Authorization", "Bearer "+regularToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var tokens []token.Token
		if expect.NoError(json.NewDecoder(w.Body).Decode(&tokens), "Decode JSON Token") {
			expect.Equal(1, len(tokens), "Token Count")
		}
	}

	// Read any user tokens by administrator
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/tokens?user=info%40versionary.net", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var tokens []token.Token
		if expect.NoError(json.NewDecoder(w.Body).Decode(&tokens), "Decode JSON Token") {
			expect.Contains(tokens[0].UserID, regularUser.ID, "User ID")
		}
	}

	// Read admin user tokens by regular user
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/tokens?user=admin%40versionary.net", nil)
	req.Header.Set("Authorization", "Bearer "+regularToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusForbidden, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusForbidden, e.Code, "Event Code")
			expect.Contains(e.Message, "unauthorized: read tokens", "Event Message")
		}
	}

	// Read tokens for specified non admin user
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/tokens?user=info%40versionary.net", nil)
	req.Header.Set("Authorization", "Bearer "+regularToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var tokens []token.Token
		if expect.NoError(json.NewDecoder(w.Body).Decode(&tokens), "Decode JSON Token") {
			expect.Contains(tokens[0].UserID, regularUser.ID, "User ID")
		}
	}
}

func TestReadToken(t *testing.T) {
	expect := assert.New(t)
	// Read tokens
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/tokens/"+adminToken, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var t token.Token
		if expect.NoError(json.NewDecoder(w.Body).Decode(&t), "Decode JSON Token") {
			expect.Equal(adminToken, t.ID, "Token")
		}
	}

	// Read tokens for specified user
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/tokens/"+regularToken, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var t token.Token
		if expect.NoError(json.NewDecoder(w.Body).Decode(&t), "Decode JSON Token") {
			expect.Equal(regularToken, t.ID, "Token")
		}
	}
}
