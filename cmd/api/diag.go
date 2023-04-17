package main

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	user_agent "github.com/voxtechnica/user-agent"

	"versionary-api/cmd/api/docs"
	"versionary-api/pkg/token"
	"versionary-api/pkg/user"
)

// registerDiagRoutes initializes the diagnostic routes.
func registerDiagRoutes(r *gin.Engine) {
	// Swagger 2.0 Meta Information
	docs.SwaggerInfo.Title = api.Name
	docs.SwaggerInfo.Description = api.Description
	docs.SwaggerInfo.Version = api.GitHash
	docs.SwaggerInfo.BasePath = "/"
	r.GET("/docs", swaggerDocs)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Diagnostic routes
	r.Any("/echo", roleAuthorizer("admin"), echoRequest)
	r.GET("/user_agent", userAgent)
	r.GET("/commit", commit)
	r.GET("/about", about)
	r.GET("/", about)
}

// swaggerDocs initializes Swagger and redirects to the Swagger API documentation.
//
// @Summary Show API documentation
// @Description Show API documentation
// @Description Show Swagger API documentation, generated from annotations in the running code.
// @Tags Diagnostic
// @Produce html
// @Success 307 {string} string
// @Router /docs [get]
func swaggerDocs(c *gin.Context) {
	c.Redirect(http.StatusFound, "/swagger/index.html")
}

// about provides basic information about the API, including the operating environment and the current git commit.
//
// @Summary About the API
// @Description Basic information about the API
// @Description Basic information about the API, including the operating environment and the current git commit.
// @Tags Diagnostic
// @Produce json
// @Success 200 {object} app.About "Information about the running application"
// @Router /about [get]
func about(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, api.About())
}

// commit redirects to the current git commit on GitHub.
//
// @Summary Show Git Commit
// @Description Redirect to the current git commit on GitHub
// @Description Redirects to the current git commit on GitHub.
// @Tags Diagnostic
// @Produce html,json
// @Success 307 {string} string "Redirect URL"
// @Failure 503 {object} APIEvent "git commit URL unavailable"
// @Header 307 {string} Location "git commit URL"
// @Router /commit [get]
func commit(c *gin.Context) {
	url := gitCommitURL()
	if url == "" {
		abortWithError(c, http.StatusServiceUnavailable, errors.New("git commit URL unavailable"))
	} else {
		c.Redirect(http.StatusTemporaryRedirect, url)
	}
}

// userAgent echoes a parsed client User-Agent header.
//
// @Summary Parse User-Agent Header
// @Description Echo a parsed User-Agent header
// @Description Echo a parsed User-Agent header.
// @Tags Diagnostic
// @Produce json
// @Param user-agent header string false "User-Agent header"
// @Success 200 {object} user_agent.UserAgent "Parsed User-Agent header"
// @Router /user_agent [get]
func userAgent(c *gin.Context) {
	header := c.Request.Header.Get("User-Agent")
	ua := user_agent.Parse(header)
	c.IndentedJSON(http.StatusOK, ua)
}

// echoRequest echos the request back to the client, including a recognized Token and associated User.
//
// @Summary Echo Request
// @Description Echo the request back to the client
// @Description Echo the request back to the client, including the provided Token and associated User.
// @Tags Diagnostic
// @Accept plain,json
// @Produce json
// @Param authorization header string true "OAuth Bearer Token (Administrator)"
// @Param body body string false "Request body"
// @Success 200 {object} request "Echoed request information"
// @Failure 400 {object} APIEvent "Bad Request (invalid parameter)"
// @Failure 401 {object} APIEvent "Unauthenticated (missing or invalid Authorization header)"
// @Failure 403 {object} APIEvent "Unauthorized (not an Administrator)"
// @Failure 500 {object} APIEvent "Error reading request body"
// @Router /echo [post]
func echoRequest(c *gin.Context) {
	reverse, limit, offset, err := paginationParams(c, false, 100)
	if err != nil {
		abortWithError(c, http.StatusBadRequest, err)
		return
	}
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
		Params:           params{Reverse: reverse, Limit: limit, Offset: offset},
	}
	if c.Request.Body != nil {
		defer c.Request.Body.Close()
		buf := new(bytes.Buffer)
		n, err := buf.ReadFrom(c.Request.Body)
		if err != nil {
			abortWithError(c, http.StatusInternalServerError, fmt.Errorf("error reading request body: %w", err))
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
		r.Token = t.(token.Token)
	}
	c.JSON(http.StatusOK, r)
}

// request represents an http.Request in a more readable format.
type request struct {
	Method           string              `json:"method,omitempty"`
	URL              string              `json:"url,omitempty"`
	Proto            string              `json:"proto,omitempty"`
	Header           map[string][]string `json:"header,omitempty"`
	Trailer          map[string][]string `json:"trailer,omitempty"`
	ContentLength    int64               `json:"contentLength,omitempty"`
	TransferEncoding []string            `json:"transferEncoding,omitempty"`
	Host             string              `json:"host,omitempty"`
	RemoteAddr       string              `json:"remoteAddr,omitempty"`
	RequestURI       string              `json:"requestURI,omitempty"`
	Params           params              `json:"params,omitempty"`
	Body             string              `json:"body,omitempty"`
	Token            token.Token         `json:"token,omitempty"`
	User             user.User           `json:"user,omitempty"`
}

// params represents pagination parameters, specified as query parameters.
type params struct {
	Reverse bool   `json:"reverse"`
	Limit   int    `json:"limit"`
	Offset  string `json:"offset"`
}
