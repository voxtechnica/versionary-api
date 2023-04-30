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
	u.Roles = append(u.Roles, "test")
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
		var users []user.User
		if expect.NoError(json.NewDecoder(w.Body).Decode(&users), "Decode JSON Users") {
			expect.GreaterOrEqual(len(users), 1, "Number of users")
			ids := versionary.Map(users, func(u user.User) string { return u.ID })
			expect.Contains(ids, regularUser.ID, "User ID")
		}
	}

	// Read users by status invalid
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/users?status=invalid", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusBadRequest, e.Code, "Event Code")
			expect.Contains(e.Message, "invalid status: INVALID", "Event Message")
		}
	}
	// Read users by email address
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/users?email=info%40versionary.net", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var users []user.User
		if expect.NoError(json.NewDecoder(w.Body).Decode(&users), "Decode JSON User") {
			expect.Equal(len(users), 1)
		}

	}
	// Read users by org ID
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/users?org="+adminUser.OrgID, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var users []user.User
		if expect.NoError(json.NewDecoder(w.Body).Decode(&users), "Decode JSON Users") {
			expect.GreaterOrEqual(len(users), 1, "Number of users")
			orgIDs := versionary.Map(users, func(u user.User) string { return u.OrgID })
			expect.Contains(orgIDs, adminUser.OrgID, "User Organization ID")
		}
	}
	// Read users by invalid org ID
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/users?org=invalid", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusBadRequest, e.Code, "Event Code")
			expect.Contains(e.Message, "invalid parameter, org : invalid", "Event Message")
		}
	}
	// Read users by role
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/users?role=admin", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var users []user.User
		if expect.NoError(json.NewDecoder(w.Body).Decode(&users), "Decode JSON Users") {
			expect.Equal(users[0], adminUser)
		}
	}
	// Read users by invalid role
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/users?role=invalid", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var users []user.User
		if expect.NoError(json.NewDecoder(w.Body).Decode(&users), "Decode JSON Users") {
			expect.Empty(users)
		}
	}
}

func TestReadUser(t *testing.T) {
	expect := assert.New(t)
	// Read a known user by ID
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/users/"+adminUser.ID, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var u user.User
		if expect.NoError(json.NewDecoder(w.Body).Decode(&u), "Decode JSON User") {
			expect.Equal(adminUser, u, "User")
		}
	}
	// Read a known user by email
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/users/"+adminUser.Email, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var u user.User
		if expect.NoError(json.NewDecoder(w.Body).Decode(&u), "Decode JSON User") {
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
	// Read user ID by unauthorized user
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/users/"+adminUser.ID, nil)
	req.Header.Set("Authorization", "Bearer "+regularToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusForbidden, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusForbidden, e.Code, "Event Code")
			expect.Contains(e.Message, "unauthorized: read user", "Event Message")
		}
	}
	// Read regular user own ID
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/users/"+regularUser.ID, nil)
	req.Header.Set("Authorization", "Bearer "+regularToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var u user.User
		if expect.NoError(json.NewDecoder(w.Body).Decode(&u), "Decode JSON User") {
			expect.Equal(regularUser.ID, u.ID, "User")
			expect.Empty(u.PasswordHash, "User Password Hash")
			expect.Empty(u.PasswordReset, "User Password Reset Token")
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
	// Read versions of a user that does not exist
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

func TestReadUserVersion(t *testing.T) {
	expect := assert.New(t)
	// Read a known user version
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/users/"+adminUser.ID+"/versions/"+adminUser.VersionID, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var u user.User
		if expect.NoError(json.NewDecoder(w.Body).Decode(&u), "Decode JSON User") {
			expect.Equal(adminUser, u, "User")
		}
	}
	// Read a known user version that does not exist
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/users/"+adminUser.ID+"/versions/"+tuid.NewID().String(), nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusNotFound, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusNotFound, e.Code, "Event Code")
			expect.Contains(e.Message, "not found: user "+adminUser.ID+" version", "Event Message")
		}
	}
	// Read a user version: invalid user ID
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/users/bad_id/versions/"+adminUser.VersionID, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusBadRequest, e.Code, "Event Code")
			expect.Contains(e.Message, "invalid path parameter ID", "Event Message")
		}
	}
}

func TestUserVersionExists(t *testing.T) {
	expect := assert.New(t)
	// Check an invalid user ID
	w := httptest.NewRecorder()
	req, err := http.NewRequest("HEAD", "/v1/users/bad_id/versions/bad_id", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code, "HTTP Status Code")
	}
	// Check a user that does not exist
	userID := tuid.NewID().String()
	w = httptest.NewRecorder()
	req, err = http.NewRequest("HEAD", "/v1/users/"+userID+"/versions/"+userID, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusNotFound, w.Code, "HTTP Status Code")
	}
	// Check if a known user version exists
	w = httptest.NewRecorder()
	req, err = http.NewRequest("HEAD", "/v1/users/"+adminUser.ID+"/versions/"+adminUser.VersionID, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusNoContent, w.Code, "HTTP Status Code")
	}
}

func TestUpdateUser(t *testing.T) {
	expect := assert.New(t)
	// Update a user: invalid JSON
	w := httptest.NewRecorder()
	req, err := http.NewRequest("PUT", "/v1/users/"+adminUser.ID, strings.NewReader(""))
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
	// Update a user: invalid user ID
	w = httptest.NewRecorder()
	req, err = http.NewRequest("PUT", "/v1/users/bad_id", strings.NewReader(`{"givenName": "updated_user"}`))
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
			expect.Contains(e.Message, "invalid path parameter ID: bad_id", "Event Message")
		}
	}
	// Update a user: mismatched IDs
	w = httptest.NewRecorder()
	userID := tuid.NewID().String()
	body := `{"id": "` + userID + `", "givenName": "New Name"}`
	req, err = http.NewRequest("PUT", "/v1/users/"+adminUser.ID, strings.NewReader(body))
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
			expect.Contains(e.Message, "does not match", "Event Message")
		}
	}
	// Update a user: escalation of privileges
	w = httptest.NewRecorder()
	priorRoles := make([]string, len(regularUser.Roles))
	_ = copy(priorRoles, regularUser.Roles)
	regularUser.Roles = append(regularUser.Roles, "admin")
	b, _ := json.Marshal(regularUser)
	req, err = http.NewRequest("PUT", "/v1/users/"+regularUser.ID, bytes.NewBuffer(b))
	req.Header.Set("Authorization", "Bearer "+regularToken)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		if expect.NoError(json.NewDecoder(w.Body).Decode(&regularUser), "Decode JSON User") {
			expect.Equal(priorRoles, regularUser.Roles, "Unchanged Regular User Roles")
		}
	}
}

func TestDeleteUser(t *testing.T) {
	expect := assert.New(t)
	// Delete an invalid user ID
	w := httptest.NewRecorder()
	req, err := http.NewRequest("DELETE", "/v1/users/bad_id", nil)
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
	// Delete a user that does not exist
	w = httptest.NewRecorder()
	userID := tuid.NewID().String()
	req, err = http.NewRequest("DELETE", "/v1/users/"+userID, nil)
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
	// Delete an admin user by regular user
	w = httptest.NewRecorder()
	req, err = http.NewRequest("DELETE", "/v1/users/"+adminUser.ID, nil)
	req.Header.Set("Authorization", "Bearer "+regularToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusForbidden, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusForbidden, e.Code, "Event Code")
			expect.Contains(e.Message, "unauthorized: delete user", "Event Message")
		}
	}
}

func TestReadUserIDs(t *testing.T) {
	expect := assert.New(t)
	// Get user IDs: missing email query
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/user_ids", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusBadRequest, e.Code, "Event Code")
			expect.Contains(e.Message, "missing required query parameter: email", "Event Message")
		}
	}
	//Get user IDs: invalid email address
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", `/v1/user_ids?email=bad_email`, nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusBadRequest, e.Code, "Event Code")
			expect.Contains(e.Message, "invalid query parameter email:", "Event Message")
		}
	}
	// Get user IDs: happy path
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", `/v1/user_ids?email=info%40versionary.net`, nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var ids []string
		if expect.NoError(json.NewDecoder(w.Body).Decode(&ids), "Decode JSON User IDs") {
			expect.Equal(1, len(ids), "Number of IDs")
			expect.Equal(regularUser.ID, ids[0])
		}
	}
}

func TestReadUserNames(t *testing.T) {
	knownUsers := []user.User{regularUser}
	expect := assert.New(t)
	// Get user IDs and names: missing authentication token
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/user_names?search=regular&any=true", nil)
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
	// Get user IDs and names: unauthorized token (not an admin)
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/user_names?search=regular&any=true", nil)
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
	// Get user IDs and names: invalid 'any' parameter
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", `/v1/user_names?search=regular&any=invalid`, nil)
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
	// Get user IDs and names: invalid 'limit' parameter
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", `/v1/user_names?limit=invalid`, nil)
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
	// Get user IDs and names: happy path (search and any parameters)
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/user_names?search=Regular&any=true", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var users []versionary.TextValue
		if expect.NoError(json.NewDecoder(w.Body).Decode(&users), "Decode JSON Users") {
			names := versionary.Map(users, func(e versionary.TextValue) string { return e.Value })
			knownNames := versionary.Map(knownUsers, func(u user.User) string { return u.String() })
			expect.Equal(1, len(users), "Number of Users")
			expect.Subset(names, knownNames, "User Name")
		}
	}
	// Get user IDs and names: happy path (limit, offset parameters)
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/user_names?limit=1&offset=0", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var users []versionary.TextValue
		if expect.NoError(json.NewDecoder(w.Body).Decode(&users), "Decode JSON Users") {
			expect.Equal(1, len(users), "Number of Users")
		}
	}
	// Get user IDs and names: happy path (sorted parameters)
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/user_names?sorted=true", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var users []versionary.TextValue
		if expect.NoError(json.NewDecoder(w.Body).Decode(&users), "Decode JSON Users") {
			names := versionary.Map(users, func(e versionary.TextValue) string { return e.Value })
			knownNames := versionary.Map(knownUsers, func(u user.User) string { return u.String() })
			expect.Equal(2, len(users), "Number of Users")
			expect.Equal(names[1], knownNames[0], "User Name")
		}
	}
}

func TestReadUserEmails(t *testing.T) {
	expect := assert.New(t)
	// Get users emails: missing authentication token
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/user_emails", nil)
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
	// Get users emails: unauthorized token (not an admin)
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/user_emails", nil)
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
	// Get users emails: happy path
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/user_emails", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var emails []string
		if expect.NoError(json.NewDecoder(w.Body).Decode(&emails), "Decode JSON Emails") {
			expect.GreaterOrEqual(len(emails), 2, "Number of Emails")
		}
	}
	// Get users emails: happy path (limit parameter)
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/user_emails?limit=1", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var emails []string
		if expect.NoError(json.NewDecoder(w.Body).Decode(&emails), "Decode JSON Emails") {
			expect.Equal(1, len(emails))
		}
	}
	// Get users emails: happy path (reverse parameter)
	expectedEmailAddress := "info@versionary.net"
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/user_emails?reverse=true", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var emails []string
		if expect.NoError(json.NewDecoder(w.Body).Decode(&emails), "Decode JSON Emails") {
			expect.GreaterOrEqual(len(emails), 2, "Number of Emails")
			expect.Contains(emails, expectedEmailAddress, "Email address")
		}
	}
}

func TestReadUserOrgs(t *testing.T) {
	expect := assert.New(t)
	// Get orgs ID and name: missing authentication token
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/user_orgs", nil)
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
	// Get orgs ID and name: unauthorized token (not an admin)
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/user_orgs", nil)
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
	// Get orgs ID and name: happy path
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/user_orgs", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var orgs []versionary.TextValue
		if expect.NoError(json.NewDecoder(w.Body).Decode(&orgs), "Decode JSON Organizations") {
			orgNames := versionary.Map(orgs, func(e versionary.TextValue) string { return e.Value })
			expect.GreaterOrEqual(len(orgs), 1, "Number of Organizations")
			knownOrgName := userOrg.Name
			expect.Contains(orgNames, knownOrgName, "Organization Name")
		}
	}
}

func TestReadUserRoles(t *testing.T) {
	expect := assert.New(t)
	// Get roles: missing authentication token
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/user_roles", nil)
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
	// Get roles: unauthorized token (not an admin)
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/user_roles", nil)
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
	// Get roles: happy path
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/user_roles", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var roles []string
		if expect.NoError(json.NewDecoder(w.Body).Decode(&roles), "Decode JSON User Roles") {
			expect.GreaterOrEqual(len(roles), 2, "Number of Roles")
			expect.Contains(roles, "admin", "Role Name")
			expect.Contains(roles, "creator", "Role Name")
		}
	}
}

func TestReadUserStatuses(t *testing.T) {
	expect := assert.New(t)
	// Get statuses: missing authentication token
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/user_statuses", nil)
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
	// Get statuses: unauthorized token (not an admin)
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/user_statuses", nil)
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
	// Get statuses: happy path
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/user_statuses", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var statuses []string
		if expect.NoError(json.NewDecoder(w.Body).Decode(&statuses), "Decode JSON User Statuses") {
			expect.GreaterOrEqual(len(statuses), 2, "Number of User Statuses")
			expect.Contains(statuses, string(user.PENDING), "PENDING Status exists")
			expect.Contains(statuses, string(user.ENABLED), "ENABLED Status exists")
		}
	}
}

func TestSendResetToken(t *testing.T) {
	expect := assert.New(t)
	// Valid user email
	w := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/v1/users/"+regularUser.Email+"/resets", nil)
	req.Header.Set("Content-Type", "application/json;charset=UFT-8")
	req.Header.Set("Accept", "application/json;charset=UFT-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusNoContent, w.Code, "HTTP Status Code")
	}

	// // Valid user ID
	w = httptest.NewRecorder()
	req, err = http.NewRequest("POST", "/v1/users/"+regularUser.ID+"/resets", nil)
	req.Header.Set("Content-Type", "application/json;charset=UFT-8")
	req.Header.Set("Accept", "application/json;charset=UFT-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusNoContent, w.Code, "HTTP Status Code")
	}

	// // Invalid parameter
	w = httptest.NewRecorder()
	req, err = http.NewRequest("POST", "/v1/users/bad_email/resets", nil)
	req.Header.Set("Content-Type", "application/json;charset=UFT-8")
	req.Header.Set("Accept", "application/json;charset=UFT-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code, "HTTP Status Code")
	}

	// // Unknown email address
	w = httptest.NewRecorder()
	req, err = http.NewRequest("POST", "/v1/users/unknown@address.net/resets", nil)
	req.Header.Set("Content-Type", "application/json;charset=UFT-8")
	req.Header.Set("Accept", "application/json;charset=UFT-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusNotFound, w.Code, "HTTP Status Code")
	}

	// // Unknown user ID
	w = httptest.NewRecorder()
	userID := tuid.NewID().String()
	req, err = http.NewRequest("POST", "/v1/users/"+userID+"/resets", nil)
	req.Header.Set("Content-Type", "application/json;charset=UFT-8")
	req.Header.Set("Accept", "application/json;charset=UFT-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusNotFound, w.Code, "HTTP Status Code")
	}

	// Unprocessable entity
	// w = httptest.NewRecorder()
	// body := `{"name": "", "email": "info@versionary.net"}`
	// req, err = http.NewRequest("POST", "/v1/users/"+body+"/resets", nil)
	// req.Header.Set("Content-Type", "application/json;charset=UFT-8")
	// req.Header.Set("Accept", "application/json;charset=UFT-8")
	// if expect.NoError(err) {
	// 	r.ServeHTTP(w, req)
	// 	expect.Equal(http.StatusUnprocessableEntity, w.Code, "HTTP Status Code")
	// }
}

func TestResetUserPassword(t *testing.T) {
	expect := assert.New(t)
	// Invalid token
	w := httptest.NewRecorder()
	req, err := http.NewRequest("PUT", "/v1/users/"+regularUser.Email+"/resets/bad_token", nil)
	req.Header.Set("Content-Type", "application/json;charset=UFT-8")
	req.Header.Set("Accept", "application/json;charset=UFT-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code, "HTTP Status Code")
	}

	// Unknown token
	w = httptest.NewRecorder()
	token := tuid.NewID().String()
	req, err = http.NewRequest("PUT", "/v1/users/"+regularUser.Email+"/resets/"+token, nil)
	req.Header.Set("Content-Type", "application/json;charset=UFT-8")
	req.Header.Set("Accept", "application/json;charset=UFT-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code, "HTTP Status Code")
	}

	// // Invalid parameter
	w = httptest.NewRecorder()
	req, err = http.NewRequest("POST", "/v1/users/bad_email/resets/"+token, nil)
	req.Header.Set("Content-Type", "application/json;charset=UFT-8")
	req.Header.Set("Accept", "application/json;charset=UFT-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusNotFound, w.Code, "HTTP Status Code")
	}

	// // Unknown email address
	w = httptest.NewRecorder()
	req, err = http.NewRequest("POST", "/v1/users/unknown@address.net/resets"+token, nil)
	req.Header.Set("Content-Type", "application/json;charset=UFT-8")
	req.Header.Set("Accept", "application/json;charset=UFT-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusNotFound, w.Code, "HTTP Status Code")
	}
}

func TestResetFlow(t *testing.T) {
	expect := assert.New(t)
	// Create user
	var u user.User
	w := httptest.NewRecorder()
	body1 := `{"givenName": "Password_Reset_User", "email":"PasswordReset@test.com"}`
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
			expect.True(u.Status.IsValid(), "Valid User Status")
			expect.Empty(u.PasswordReset, "PasswordReset field is empty")
		}
	}

	// Generate reset token
	w = httptest.NewRecorder()
	req, err = http.NewRequest("POST", "/v1/users/"+u.Email+"/resets", nil)
	req.Header.Set("Content-Type", "application/json;charset=UFT-8")
	req.Header.Set("Accept", "application/json;charset=UFT-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusNoContent, w.Code, "HTTP Status Code")
	}

	// Read the user
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/users/"+u.ID, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	var u2 user.User
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		if expect.NoError(json.NewDecoder(w.Body).Decode(&u2), "Decode JSON User") {
			expect.NotEmpty(u2.PasswordReset, "PasswordReset token is not empty")
			expect.NotEqual(u.PasswordReset, u2.PasswordReset)
		}
	}

	// Reset the password
	w = httptest.NewRecorder()
	body := `{"password": "new_password123"}`
	req, err = http.NewRequest("PUT", "/v1/users/"+u2.Email+"/resets/"+u2.PasswordReset, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json;charset=UFT-8")
	req.Header.Set("Accept", "application/json;charset=UFT-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusNoContent, w.Code, "HTTP Status Code")
	}

	// Read the user
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/users/"+u2.ID, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var u2 user.User
		if expect.NoError(json.NewDecoder(w.Body).Decode(&u2), "Decode JSON User") {
			expect.Empty(u2.PasswordReset, "PasswordReset token is empty")
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
		}
	}
}
