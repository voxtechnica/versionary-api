package main

import (
	"bytes"
	"encoding/json"
	"fmt"

	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/voxtechnica/tuid-go"

	"github.com/voxtechnica/versionary"

	"versionary-api/pkg/user"
)

func TestUserCRUD(t *testing.T) {
	expect := assert.New(t)
	// Create an user
	var u user.User
	w := httptest.NewRecorder()
	body1 := `{"givenName": "crud_user", "email":"Test_User@test.com"}`
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
			expect.Equal("crud_user", u.GivenName, "User Given Name")
			expect.True(u.Status.IsValid(), "Valid User Status")
		}
	}

	// Read the user
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/users/"+u.ID, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req) 
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var u2 user.User
		if expect.NoError(json.NewDecoder(w.Body).Decode(&u2), "Decode JSON User") {
			expect.Equal(u, u2, "User")
		}
	}
	// Update the user
	w = httptest.NewRecorder()
	u.Status = user.DISABLED
	body, _ := json.Marshal(u)
	req, err = http.NewRequest("PUT", "/v1/users/"+u.ID, bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var u2 user.User
		if expect.NoError(json.NewDecoder(w.Body).Decode(&u2), "Decode JSON User") {
			expect.Equal(u.ID, u2.ID, "User ID")
			expect.NotEqual(u.VersionID, u2.VersionID, "User VersionID")
			expect.Equal(u.GivenName, u2.GivenName, "User Given Name")
			expect.Equal(u.Status, u2.Status, "User Status")
		}
	}
	// Read the user versions
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/users/"+u.ID+"/versions", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var versions []user.User
		if expect.NoError(json.NewDecoder(w.Body).Decode(&versions), "Decode JSON Versions") {
			expect.Equal(2, len(versions), "Number of Versions")
			expect.Equal(u.VersionID, versions[0].VersionID, "1st Version ID")
			expect.NotEqual(u.VersionID, versions[1].VersionID, "2nd Version ID")
		}
	}
	// Delete the user
	w = httptest.NewRecorder()
	req, err = http.NewRequest("DELETE", "/v1/users/"+u.ID, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var u2 user.User
		if expect.NoError(json.NewDecoder(w.Body).Decode(&u2), "Decode JSON User") {
			expect.Equal(u.ID, u2.ID, "User ID")
			expect.Equal(u.GivenName, u2.GivenName, "User Name")
			expect.Equal(u.Status, u2.Status, "User Status")
		}
	}
}

func TestCreateUser(t *testing.T) {
	expect := assert.New(t)
	// Create a user: invalid JSON body
	w := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/v1/users", strings.NewReader(""))
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
	// Create a user: validation errors
	w = httptest.NewRecorder()
	body := `{"givenName": "test_user_1", "email":"Test_User1@test.com", "status": "INVALID"}`
	req, err = http.NewRequest("POST", "/v1/users", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusUnprocessableEntity, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusUnprocessableEntity, e.Code, "Event Code")
			expect.Contains(e.Message, "invalid field(s)", "Event Message")
		}
	}
	// Create a user: missing authorization token
	w = httptest.NewRecorder()
	body = `{"givenName": "test_user_1", "email":"Test_User1@test.com", "status": "ENABLED"}`
	req, err = http.NewRequest("POST", "/v1/users", strings.NewReader(body))
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
	// Create a user: missing admin role
	w = httptest.NewRecorder()
	body = `{"givenName": "test_user_1", "email":"Test_User1@test.com", "status": "ENABLED"}`
	req, err = http.NewRequest("POST", "/v1/users", strings.NewReader(body))
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

func TestReadUsers(t *testing.T) {
	expect := assert.New(t)
	// Read paginated users
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/users?reverse=false&limit=10", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var users []user.User
		if expect.NoError(json.NewDecoder(w.Body).Decode(&users), "Decode JSON Users") {
			expect.GreaterOrEqual(len(users), 2, "Number of Users")
			ids := versionary.Map(users, func(u user.User) string { return u.ID })
			expect.Contains(ids, adminUser.ID, "User ID")
		}
	}
	// Read users by status
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/users?status=pending", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		fmt.Println("Server response: ", w.Body)
		fmt.Println(req)
		var users []user.User
		if expect.NoError(json.NewDecoder(w.Body).Decode(&users), "Decode JSON Users") {
			expect.GreaterOrEqual(len(users), 1, "Number of users")
			ids := versionary.Map(users, func(o user.User) string { return o.ID })
			expect.Contains(ids, regularUser.ID, "User ID")
		}
	}

	// Read users by status invalid
	// w = httptest.NewRecorder()
	// req, err = http.NewRequest("GET", "/v1/users?status=invalid", nil)
	// req.Header.Set("Authorization", "Bearer "+adminToken)
	// req.Header.Set("Accept", "application/json;charset=UTF-8")
	// if expect.NoError(err) {
	// 	r.ServeHTTP(w, req)
	// 	expect.Equal(http.StatusBadRequest, w.Code, "HTTP Status Code")
	// 	fmt.Println("Server response: ", w.Body)
	// 	fmt.Println(req)
	// 	var e APIEvent
	// 	if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Users") {
	// 		expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
	// 		expect.Equal(http.StatusBadRequest, e.Code, "Event Code")
	// 		expect.Contains(e.Message, "invalid status: INVALID", "Event Message")
	// 	}
	// }
}

func TestReadUser(t *testing.T) {
	expect := assert.New(t)
	// Read a known user
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/users/"+adminUser.ID, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var u user.User
		if expect.NoError(json.NewDecoder(w.Body).Decode(&u), "Decode JSON Organization") {
			expect.Equal(adminUser, u, "User")
		}
	}
	// Read an invalid user ID
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/users/bad_id", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusBadRequest, e.Code, "Event Code")
			expect.Contains(e.Message, "invalid path parameter ID: bad_id", "Event Message")
		}
	}
	// Read a user that does not exist
	w = httptest.NewRecorder()
	userID := tuid.NewID().String()
	req, err = http.NewRequest("GET", "/v1/users/"+userID, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusNotFound, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusNotFound, e.Code, "Event Code")
			expect.Contains(e.Message, "not found: user "+userID, "Event Message")
		}
	}
}

func TestUserExists(t *testing.T) {
	expect := assert.New(t)
	// Check an invalid user ID
	w := httptest.NewRecorder()
	req, err := http.NewRequest("HEAD", "/v1/users/bad_id", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code, "HTTP Status Code")
	}
	// Check a user that does not exist
	w = httptest.NewRecorder()
	userID := tuid.NewID().String()
	req, err = http.NewRequest("HEAD", "/v1/users/"+userID, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusNotFound, w.Code, "HTTP Status Code")
	}
	// Check if a known user exists
	w = httptest.NewRecorder()
	req, err = http.NewRequest("HEAD", "/v1/users/"+regularUser.ID, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusNoContent, w.Code, "HTTP Status Code")
	}
}

func TestReadUserVersions(t *testing.T) {
	expect := assert.New(t)
	// Read versions of a known user
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/users/"+adminUser.ID+"/versions", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var versions []user.User
		if expect.NoError(json.NewDecoder(w.Body).Decode(&versions), "Decode JSON User Versions") {
			expect.Equal(1, len(versions), "Number of Versions")
			expect.Equal(adminUser, versions[0], "User Version")
		}
	}
	// Read versions of an invalid user ID
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/users/bad_id/versions", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusBadRequest, e.Code, "Event Code")
			expect.Contains(e.Message, "invalid path parameter ID: bad_id", "Event Message")
		}
	}
	// // Read versions of a user that does not exist
	w = httptest.NewRecorder()
	userID := tuid.NewID().String()
	req, err = http.NewRequest("GET", "/v1/users/"+userID+"/versions", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusNotFound, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusNotFound, e.Code, "Event Code")
			expect.Contains(e.Message, "not found: user "+userID, "Event Message")
		}
	}
}



