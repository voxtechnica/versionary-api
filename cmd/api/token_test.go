package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"versionary-api/pkg/token"
	"versionary-api/pkg/user"

	"github.com/stretchr/testify/assert"
	"github.com/voxtechnica/tuid-go"
	"github.com/voxtechnica/versionary"
)

func TestTokenCRUD(t *testing.T) {
	expect := assert.New(t)
	// Create new user
	var u user.User
	w := httptest.NewRecorder()
	body1 := `{"givenName": "token_user", "email":"token_user@test.com", "password": "tokenabcd1234"}`
	req, err := http.NewRequest("POST", "/v1/users", strings.NewReader(body1))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json;charset=UFT-8")
	req.Header.Set("Accept", "application/json;charset=UFT-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusCreated, w.Code, "HTTP Status Code")
		if expect.NoError(json.NewDecoder(w.Body).Decode(&u), "Decode JSON User") {
			expect.True(tuid.IsValid(tuid.TUID(u.ID)), "Valid User ID")
			expect.Equal(u.ID, u.VersionID, "ID and VersionID match")
			expect.Equal("token_user", u.GivenName, "User Given Name")
			expect.True(u.Status.IsValid(), "Valid User Status")
		}
	}
	// Create token request
	j, err := json.Marshal(token.Request{
		GrantType: "password",
		Username:  "token_user@test.com",
		Password:  "tokenabcd1234",
	})
	if err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	req, err = http.NewRequest("POST", "/v1/tokens", bytes.NewBuffer(j))
	// req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	var token token.Response
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusCreated, w.Code, "HTTP Status Code")
		if expect.NoError(json.NewDecoder(w.Body).Decode(&token), "Decode JSON Token") {
			expect.True(tuid.IsValid(tuid.TUID(token.AccessToken)), "Valid Token ID")
		}
	}

	// Read token
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "v1/token/"+token.AccessToken, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
	}

	// Delete token
	w = httptest.NewRecorder()
	req, err = http.NewRequest("DELETE", "v1/tokens/"+token.AccessToken, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
	}

	// Read token
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "v1/token/"+token.AccessToken, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusNotFound, w.Code, "HTTP Status Code")
	}
}

func TestReadTokens(t *testing.T) {
	expect := assert.New(t)
	// Read tokens by admin user
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/tokens", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
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
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
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
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
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
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
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
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var tokens []token.Token
		if expect.NoError(json.NewDecoder(w.Body).Decode(&tokens), "Decode JSON Token") {
			expect.Contains(tokens[0].UserID, regularUser.ID, "User ID")
		}
	}
}

func TestTokenExists(t *testing.T) {
	expect := assert.New(t)
	// Token exists
	w := httptest.NewRecorder()
	req, err := http.NewRequest("HEAD", "/v1/tokens/"+adminToken, nil)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusNoContent, w.Code, "HTTP Status Code")
	}

	// Bad token parameter
	w = httptest.NewRecorder()
	req, err = http.NewRequest("HEAD", "/v1/tokens/bad_token", nil)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code, "HTTP Status Code")
	}

	// Token does not exist
	tokenID := tuid.NewID().String()
	w = httptest.NewRecorder()
	req, err = http.NewRequest("HEAD", "/v1/tokens/"+tokenID, nil)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusNotFound, w.Code, "HTTP Status Code")
	}
}

func TestDeleteToken(t *testing.T) {
	expect := assert.New(t)
	// Bad token parameter
	w := httptest.NewRecorder()
	req, err := http.NewRequest("DELETE", "/v1/tokens/bad_token", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code, "HTTP Status Code")
	}

	// Missing Authorization header
	w = httptest.NewRecorder()
	req, err = http.NewRequest("DELETE", "/v1/tokens/"+adminToken, nil)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusUnauthorized, w.Code, "HTTP Status Code")
	}

	// Forbidden (not admin)
	w = httptest.NewRecorder()
	req, err = http.NewRequest("DELETE", "/v1/tokens/"+adminToken, nil)
	req.Header.Set("Authorization", "Bearer "+regularToken)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusForbidden, w.Code, "HTTP Status Code")
	}

	// Token not found
	tokenID := tuid.NewID().String()
	w = httptest.NewRecorder()
	req, err = http.NewRequest("DELETE", "/v1/tokens/"+tokenID, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusNotFound, w.Code, "HTTP Status Code")
	}
}

func TestReadTokenIDs(t *testing.T) {
	expect := assert.New(t)
	// Read token IDs
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/token_ids", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var ids []versionary.TextValue
		if expect.NoError(json.NewDecoder(w.Body).Decode(&ids), "Decode JSON Token IDs") {
			userIds := versionary.Map(ids, func(v versionary.TextValue) string { return v.Value })
			expect.Equal(3, len(ids), "Number of Token/Users ID pairs")
			expect.Contains(userIds, adminUser.ID)
			expect.Contains(userIds, regularUser.ID)
		}
	}

	// Invalid query parameter
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/token_ids?sorted=invalid", nil)
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
			expect.Contains(e.Message, "invalid parameter", "Event Message")
		}
	}

	// Missing authentication token
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/token_ids", nil)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
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
	// Unauthorized token (not an admin)
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/token_ids", nil)
	req.Header.Set("Authorization", "Bearer "+regularToken)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
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
}

func TestTokenUsers(t *testing.T) {
	expect := assert.New(t)
	// Read users with tokens
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/token_users", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var users []versionary.TextValue
		if expect.NoError(json.NewDecoder(w.Body).Decode(&users), "Decode JSON User IDs") {
			userEmails := versionary.Map(users, func(v versionary.TextValue) string { return v.Value })
			expect.Equal(3, len(users), "Number of Users with Tokens")
			expect.Contains(userEmails, adminUser.Email)
			expect.Contains(userEmails, regularUser.Email)
		}
	}

	// Get users using search query
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/token_users?search=Admin&any=false", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var users []versionary.TextValue
		if expect.NoError(json.NewDecoder(w.Body).Decode(&users), "Decode JSON User IDs") {
			userEmails := versionary.Map(users, func(v versionary.TextValue) string { return v.Value })
			expect.Equal(1, len(users), "Number of Users")
			expect.Contains(userEmails, adminUser.Email)
		}
	}

	// Unauthorized token (not an admin)
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/token_users", nil)
	req.Header.Set("Authorization", "Bearer "+regularToken)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
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

	// Invalid query parameter
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/token_users?sorted=invalid", nil)
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
			expect.Contains(e.Message, "invalid parameter", "Event Message")
		}
	}

	// Missing authentication token
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/token_users", nil)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
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
}
