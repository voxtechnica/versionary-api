package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"versionary-api/pkg/app"

	"github.com/stretchr/testify/assert"
	user_agent "github.com/voxtechnica/user-agent"
)

func TestAbout(t *testing.T) {
	expect := assert.New(t)
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/about", nil)
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var about app.About
		if expect.NoError(json.NewDecoder(w.Body).Decode(&about), "Decode JSON About") {
			expect.Equal(api.About(), about, "About")
		}
	}
}

func TestCommit(t *testing.T) {
	expect := assert.New(t)
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/commit", nil)
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		if w.Code == http.StatusTemporaryRedirect {
			l := w.Result().Header.Get("Location")
			expect.NotEmpty(l, "Location header exists")
		} else if w.Code == http.StatusServiceUnavailable {
			var e APIEvent
			if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
				expect.Equal("ERROR", e.LogLevel, "Event Log Level")
				expect.Contains(e.Message, "unavailable", "Event Message")
			}
		} else {
			expect.Fail("Unexpected HTTP Status Code")
		}
	}
}

func TestDocs(t *testing.T) {
	expect := assert.New(t)
	// Redirect to Swagger UI
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/docs", nil)
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusFound, w.Code, "HTTP Status Code")
		l := w.Result().Header.Get("Location")
		expect.Equal("/swagger/index.html", l, "Location header")
	}
	// Note that the Swagger UI is not available during testing.
	// It's an embedded resource in the compiled binary.
}

func TestEcho(t *testing.T) {
	expect := assert.New(t)
	// Anonymous user (no bearer token)
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/echo", nil)
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusUnauthorized, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event Log Level")
			expect.Equal("unauthenticated", e.Message, "Event Message")
		}
	}
	// Regular authenticated user (bearer token)
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/echo", nil)
	req.Header.Set("Authorization", "Bearer "+regularToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusForbidden, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event Log Level")
			expect.Equal("unauthorized", e.Message, "Event Message")
		}
	}
	// Admin user (bearer token with admin role)
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/echo", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Accept", "application/json;charset=UTF-8")
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var r request
		if expect.NoError(json.NewDecoder(w.Body).Decode(&r), "Decode JSON Response") {
			expect.Equal("GET", r.Method, "Request Method")
			expect.Equal(params{false, 100, "-"}, r.Params, "Pagination Parameters")
			expect.Equal(adminToken, r.Token.ID, "Bearer Token")
		}
	}
}

func TestNotFound(t *testing.T) {
	expect := assert.New(t)
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/not_found", nil)
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusNotFound, w.Code, "HTTP Status Code")
		var e APIEvent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&e), "Decode JSON Event") {
			expect.Equal("ERROR", e.LogLevel, "Event Log Level")
			expect.Equal("not found: API endpoint", e.Message, "Event Message")
		}
	}
}

func TestUserAgent(t *testing.T) {
	expect := assert.New(t)
	h := "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/104.0.0.0 Safari/537.36"
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/user_agent", nil)
	req.Header.Set("User-Agent", h)
	if expect.NoError(err) {
		r.ServeHTTP(w, req)
		expect.Equal(http.StatusOK, w.Code, "HTTP Status Code")
		var ua user_agent.UserAgent
		if expect.NoError(json.NewDecoder(w.Body).Decode(&ua), "Decode JSON UserAgent") {
			expect.Equal("Browser", ua.ClientType, "Client Type")
			expect.Equal("Chrome", ua.ClientName, "Client Name")
			expect.Equal("104.0", ua.ClientVersion, "Client Version")
		}
	}
}
