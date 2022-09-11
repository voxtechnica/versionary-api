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

	"versionary-api/pkg/org"
	"versionary-api/pkg/user"
)

func TestOrganizationCRUD(t *testing.T) {
	expect := assert.New(t)
	// Create an organization
	var o org.Organization
	w := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/v1/organizations", strings.NewReader(`{"name":"Test Organization"}`))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusCreated, w.Code, "HTTP Status Code")
		if expect.NoError(json.NewDecoder(w.Body).Decode(&o), "Decode JSON Organization") {
			expect.True(tuid.IsValid(tuid.TUID(o.ID)), "Valid Organization ID")
			expect.Equal(o.ID, o.VersionID, "ID and VersionID match")
			expect.Equal("Test Organization", o.Name, "Organization Name")
			expect.True(o.Status.IsValid(), "Valid Organization Status")
		}
	}
	// Read the organization
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/organizations/"+o.ID, nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var o2 org.Organization
		if expect.NoError(json.NewDecoder(w.Body).Decode(&o2), "Decode JSON Organization") {
			expect.Equal(o, o2, "Organization")
		}
	}
	// Update the organization
	w = httptest.NewRecorder()
	o.Status = org.DISABLED
	body, _ := json.Marshal(o)
	req, err = http.NewRequest("PUT", "/v1/organizations/"+o.ID, bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var o2 org.Organization
		if expect.NoError(json.NewDecoder(w.Body).Decode(&o2), "Decode JSON Organization") {
			expect.Equal(o.ID, o2.ID, "Organization ID")
			expect.NotEqual(o.VersionID, o2.VersionID, "Organization VersionID")
			expect.Equal(o.Name, o2.Name, "Organization Name")
			expect.Equal(o.Status, o2.Status, "Organization Status")
		}
	}
	// Read the organization versions
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/organizations/"+o.ID+"/versions", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var versions []org.Organization
		if expect.NoError(json.NewDecoder(w.Body).Decode(&versions), "Decode JSON Versions") {
			expect.Equal(2, len(versions), "Number of Versions")
			expect.Equal(o.VersionID, versions[0].VersionID, "1st Version ID")
			expect.NotEqual(o.VersionID, versions[1].VersionID, "2nd Version ID")
		}
	}
	// Delete the organization
	w = httptest.NewRecorder()
	req, err = http.NewRequest("DELETE", "/v1/organizations/"+o.ID, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var o2 org.Organization
		if expect.NoError(json.NewDecoder(w.Body).Decode(&o2), "Decode JSON Organization") {
			expect.Equal(o.ID, o2.ID, "Organization ID")
			expect.Equal(o.Name, o2.Name, "Organization Name")
			expect.Equal(o.Status, o2.Status, "Organization Status")
		}
	}
}

func TestCreateOrganization(t *testing.T) {
	expect := assert.New(t)
	// Create an organization: invalid JSON body
	w := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/v1/organizations", strings.NewReader(""))
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
	// Create an organization: validation errors
	w = httptest.NewRecorder()
	body := `{"name": "", "status": "INVALID"}`
	req, err = http.NewRequest("POST", "/v1/organizations", strings.NewReader(body))
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
	// Create an organization: missing authorization token
	w = httptest.NewRecorder()
	body = `{"name": "Test Organization", "status": "ENABLED"}`
	req, err = http.NewRequest("POST", "/v1/organizations", strings.NewReader(body))
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
	// Create an organization: missing admin role
	w = httptest.NewRecorder()
	body = `{"name": "Test Organization", "status": "ENABLED"}`
	req, err = http.NewRequest("POST", "/v1/organizations", strings.NewReader(body))
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
	// Create an organization: covered in the CRUD test above
}

func TestReadOrganizations(t *testing.T) {
	expect := assert.New(t)
	// Read paginated organizations
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/organizations?reverse=false&limit=10", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var orgs []org.Organization
		if expect.NoError(json.NewDecoder(w.Body).Decode(&orgs), "Decode JSON Organizations") {
			expect.GreaterOrEqual(len(orgs), 1, "Number of Organizations")
			ids := versionary.Map(orgs, func(o org.Organization) string { return o.ID })
			expect.Contains(ids, userOrg.ID, "Organization ID")
		}
	}
	// Read organizations by status
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/organizations?status=enabled", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var orgs []org.Organization
		if expect.NoError(json.NewDecoder(w.Body).Decode(&orgs), "Decode JSON Organizations") {
			expect.GreaterOrEqual(len(orgs), 1, "Number of Organizations")
			ids := versionary.Map(orgs, func(o org.Organization) string { return o.ID })
			expect.Contains(ids, userOrg.ID, "Organization ID")
		}
	}
	// Read organizations by invalid status
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/organizations?status=invalid", nil)
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
}

func TestReadOrganization(t *testing.T) {
	expect := assert.New(t)
	// Read a known organization
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/organizations/"+userOrg.ID, nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var o org.Organization
		if expect.NoError(json.NewDecoder(w.Body).Decode(&o), "Decode JSON Organization") {
			expect.Equal(userOrg, o, "Organization")
		}
	}
	// Read an invalid organization ID
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/organizations/bad_id", nil)
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
	// Read an organization that does not exist
	w = httptest.NewRecorder()
	orgID := tuid.NewID().String()
	req, err = http.NewRequest("GET", "/v1/organizations/"+orgID, nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusNotFound, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusNotFound, e.Code, "Event Code")
			expect.Contains(e.Message, "not found: organization "+orgID, "Event Message")
		}
	}
}

func TestOrganizationExists(t *testing.T) {
	expect := assert.New(t)
	// Check an invalid organization ID
	w := httptest.NewRecorder()
	req, err := http.NewRequest("HEAD", "/v1/organizations/bad_id", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code, "HTTP Status Code")
	}
	// Check an organization that does not exist
	w = httptest.NewRecorder()
	orgID := tuid.NewID().String()
	req, err = http.NewRequest("HEAD", "/v1/organizations/"+orgID, nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusNotFound, w.Code, "HTTP Status Code")
	}
	// Check if a known organization exists
	w = httptest.NewRecorder()
	req, err = http.NewRequest("HEAD", "/v1/organizations/"+userOrg.ID, nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusNoContent, w.Code, "HTTP Status Code")
	}
}

func TestReadOrganizationVersions(t *testing.T) {
	expect := assert.New(t)
	// Read versions of a known organization
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/organizations/"+userOrg.ID+"/versions", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var versions []org.Organization
		if expect.NoError(json.NewDecoder(w.Body).Decode(&versions), "Decode JSON Organization Versions") {
			expect.Equal(1, len(versions), "Number of Versions")
			expect.Equal(userOrg, versions[0], "Organization Version")
		}
	}
	// Read versions of an invalid organization ID
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/organizations/bad_id/versions", nil)
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
	// Read versions of an organization that does not exist
	w = httptest.NewRecorder()
	orgID := tuid.NewID().String()
	req, err = http.NewRequest("GET", "/v1/organizations/"+orgID+"/versions", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusNotFound, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusNotFound, e.Code, "Event Code")
			expect.Contains(e.Message, "not found: organization "+orgID, "Event Message")
		}
	}
}

func TestReadOrganizationVersion(t *testing.T) {
	expect := assert.New(t)
	// Read a known organization version
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/organizations/"+userOrg.ID+"/versions/"+userOrg.VersionID, nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var o org.Organization
		if expect.NoError(json.NewDecoder(w.Body).Decode(&o), "Decode JSON Organization") {
			expect.Equal(userOrg, o, "Organization")
		}
	}
	// Read a known organization version that does not exist
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/organizations/"+userOrg.ID+"/versions/"+tuid.NewID().String(), nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusNotFound, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusNotFound, e.Code, "Event Code")
			expect.Contains(e.Message, "not found: organization "+userOrg.ID+" version", "Event Message")
		}
	}
	// Read an organization version: invalid organization ID
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/v1/organizations/bad_id/versions/"+userOrg.VersionID, nil)
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

func TestOrganizationVersionExists(t *testing.T) {
	expect := assert.New(t)
	// Check an invalid organization ID
	w := httptest.NewRecorder()
	req, err := http.NewRequest("HEAD", "/v1/organizations/bad_id/versions/bad_id", nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusBadRequest, w.Code, "HTTP Status Code")
	}
	// Check an organization that does not exist
	orgID := tuid.NewID().String()
	w = httptest.NewRecorder()
	req, err = http.NewRequest("HEAD", "/v1/organizations/"+orgID+"/versions/"+orgID, nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusNotFound, w.Code, "HTTP Status Code")
	}
	// Check if a known organization version exists
	w = httptest.NewRecorder()
	req, err = http.NewRequest("HEAD", "/v1/organizations/"+userOrg.ID+"/versions/"+userOrg.VersionID, nil)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusNoContent, w.Code, "HTTP Status Code")
	}
}

func TestUpdateOrganization(t *testing.T) {
	expect := assert.New(t)
	// Update an organization: invalid JSON
	w := httptest.NewRecorder()
	req, err := http.NewRequest("PUT", "/v1/organizations/"+userOrg.ID, strings.NewReader(""))
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
	// Update an organization: invalid organization ID
	w = httptest.NewRecorder()
	req, err = http.NewRequest("PUT", "/v1/organizations/bad_id", strings.NewReader(`{"name": "New Name"}`))
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
	// Update an organization: mismatched IDs
	w = httptest.NewRecorder()
	orgID := tuid.NewID().String()
	body := `{"id": "` + orgID + `", "name": "New Name"}`
	req, err = http.NewRequest("PUT", "/v1/organizations/"+userOrg.ID, strings.NewReader(body))
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
	// Update an organization: missing name
	w = httptest.NewRecorder()
	body = `{"id": "` + userOrg.ID + `", "name": ""}`
	req, err = http.NewRequest("PUT", "/v1/organizations/"+userOrg.ID, strings.NewReader(body))
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
			expect.Contains(e.Message, "missing", "Event Message")
		}
	}
	// Update a known organization: covered in the CRUD test above
}

func TestDeleteOrganization(t *testing.T) {
	expect := assert.New(t)
	// Delete an invalid organization ID
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/organizations/bad_id", nil)
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
	// Delete an organization that does not exist
	w = httptest.NewRecorder()
	orgID := tuid.NewID().String()
	req, err = http.NewRequest("DELETE", "/v1/organizations/"+orgID, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusNotFound, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event LogLevel")
			expect.Equal(http.StatusNotFound, e.Code, "Event Code")
			expect.Contains(e.Message, "not found: organization "+orgID, "Event Message")
		}
	}
	// Delete a known organization: covered in the CRUD test above
}

func TestReadOrganizationStatuses(t *testing.T) {
	expect := assert.New(t)
	// Read organization statuses in use
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/v1/organization_statuses", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var statuses []string
		if expect.NoError(json.NewDecoder(w.Body).Decode(&statuses), "Decode JSON Organization Statuses") {
			expect.GreaterOrEqual(len(statuses), 1, "Organization Statuses")
			expect.Contains(statuses, string(user.ENABLED), "ENABLED Status exists")
		}
	}
}
