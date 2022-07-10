package main

import (
	"bytes"
	"fmt"
	"net/http"
	"versionary-api/pkg/user"

	"github.com/gin-gonic/gin"
	user_agent "github.com/voxtechnica/user-agent"
)

// initDiagRoutes initializes the diagnostic routes.
func initDiagRoutes(r *gin.Engine) {
	r.Any("/echo", roleAuthorizer("admin"), echoRequest)
	r.GET("/user_agent", userAgent)
	r.GET("/commit", commit)
	r.GET("/", about)
}

// about godoc
// @Summary Basic information about the API
// @Description Basic information about the API, including the operating environment and the current git commit.
// @Accept json
// @Produce json
// @Success 200 {object} app.About
// @Failure 500 {object} gin.H
// @Router /about [get]
func about(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, api.About())
}

// commit godoc
// @Summary Redirect to the current git commit on GitHub
// @Description Redirects to the current git commit on GitHub.
// @Accept json
// @Produce json
// @Success 307 {string} string
// @Failure 503 {object} gin.H
// @Router /commit [get]
func commit(c *gin.Context) {
	url := gitCommitURL()
	if url == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"code":  http.StatusServiceUnavailable,
			"error": "unvailable (missing git hash or origin)",
		})
		return
	}
	c.Redirect(http.StatusTemporaryRedirect, url)
}

// userAgent godoc
// @Summary Echo a parsed User-Agent header
// @Description Echo a parsed User-Agent header.
// @Accept json
// @Produce json
// @Success 200 {object} user_agent.UserAgent
// @Failure 500 {object} gin.H
// @Router /user_agent [get]
func userAgent(c *gin.Context) {
	header := c.Request.Header.Get("User-Agent")
	ua := user_agent.Parse(header)
	c.IndentedJSON(http.StatusOK, ua)
}

// echoRequest godoc
// @Summary Echo the request back to the client
// @Description Echo the request back to the client, including a recognized Token and associated User.
// @Accept json
// @Produce json
// @Success 200 {object} request
// @Failure 500 {object} gin.H
// @Router /echo [get]
func echoRequest(c *gin.Context) {
	r := request{
		Method:           c.Request.Method,
		URL:              c.Request.URL.String(),
		Proto:            c.Request.Proto,
		Header:           c.Request.Header,
		Trailer:          c.Request.Trailer,
		ContentLength:    c.Request.ContentLength,
		TransferEncoding: c.Request.TransferEncoding,
		Host:             c.Request.Host,
		RemoteAddr:       c.Request.RemoteAddr,
		RequestURI:       c.Request.RequestURI,
	}
	if c.Request.Body != nil {
		defer c.Request.Body.Close()
		buf := new(bytes.Buffer)
		n, err := buf.ReadFrom(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":  http.StatusInternalServerError,
				"error": fmt.Errorf("failed to read request body: %w", err),
			})
			return
		}
		r.ContentLength = n
		r.Body = buf.String()
	}
	u, ok := c.Get("user")
	if ok {
		r.User = u.(user.User)
	}
	t, ok := c.Get("token")
	if ok {
		r.Token = t.(user.Token)
	}
	c.JSON(http.StatusOK, r)
}

// request represents an http.Request in a more readable format.
type request struct {
	Method           string      `json:"method,omitempty"`
	URL              string      `json:"url,omitempty"`
	Proto            string      `json:"proto,omitempty"`
	Header           http.Header `json:"header,omitempty"`
	Trailer          http.Header `json:"trailer,omitempty"`
	ContentLength    int64       `json:"contentLength,omitempty"`
	TransferEncoding []string    `json:"transferEncoding,omitempty"`
	Host             string      `json:"host,omitempty"`
	RemoteAddr       string      `json:"remoteAddr,omitempty"`
	RequestURI       string      `json:"requestURI,omitempty"`
	Body             string      `json:"body,omitempty"`
	Token            user.Token  `json:"token,omitempty"`
	User             user.User   `json:"user,omitempty"`
}
