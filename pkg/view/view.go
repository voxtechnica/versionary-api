package view

import (
	"net/url"
	"sort"
	"strings"
	"time"
	"versionary-api/pkg/ref"

	"github.com/voxtechnica/tuid-go"
	ua "github.com/voxtechnica/user-agent"
	"github.com/voxtechnica/versionary"
)

// Client represents the client application that generated the View.
type Client struct {
	DeviceID    string       `json:"deviceId"`              // ID of the associated Device
	UserAgent   ua.UserAgent `json:"userAgent"`             // Parsed User-Agent header
	IPAddress   string       `json:"ipAddress,omitempty"`   // Client IPv4 or IPv6 address
	CountryCode string       `json:"countryCode,omitempty"` // CloudFront-Viewer-Country (ISO 3166-1 alpha-2)
	Width       int          `json:"width,omitempty"`       // Javascript window.innerWidth
	Height      int          `json:"height,omitempty"`      // Javascript window.innerHeight
}

// Page represents the webpage that was viewed.
type Page struct {
	ID       string `json:"id,omitempty"`       // Page ID (paths change, but IDs do not)
	Type     string `json:"type,omitempty"`     // Page type (e.g. "article")
	Title    string `json:"title,omitempty"`    // Javascript document.title
	URI      string `json:"uri"`                // Javascript window.location.href
	Referrer string `json:"referrer,omitempty"` // Javascript document.referrer
}

// Path returns the path of the View Page, extracted from the URI.
func (p Page) Path() string {
	if p.URI == "" {
		return ""
	}
	u, err := url.Parse(p.URI)
	if err != nil {
		return ""
	}
	return u.EscapedPath()
}

// ReferrerDomain returns the domain (host name) of the referrer, if any.
func (p Page) ReferrerDomain() string {
	if p.Referrer == "" {
		return ""
	}
	u, err := url.Parse(p.Referrer)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}

// SearchEngine returns the name of a major search engine that referred the View, if any.
func (p Page) SearchEngine() string {
	domain := p.ReferrerDomain()
	if domain == "" {
		return ""
	}
	if strings.Contains(domain, "google.") {
		return "Google"
	}
	if strings.Contains(domain, "bing.") {
		return "Bing"
	}
	if strings.Contains(domain, "yahoo.") {
		return "Yahoo"
	}
	if strings.Contains(domain, "duckduckgo.") {
		return "DuckDuckGo"
	}
	if strings.Contains(domain, "facebook.") {
		return "Facebook"
	}
	return ""
}

// View represents a single page view by the specified client.
type View struct {
	ID        string            `json:"id"`
	CreatedAt time.Time         `json:"createdAt"`
	ExpiresAt time.Time         `json:"expiresAt"`
	Page      Page              `json:"page"`
	Client    Client            `json:"client"`
	Tags      map[string]string `json:"tags,omitempty"`
}

// Type returns the entity type of the View.
func (v View) Type() string {
	return "View"
}

// RefID returns the Reference ID of the entity.
func (v View) RefID() ref.RefID {
	r, _ := ref.NewRefID(v.Type(), v.ID, "")
	return r
}

// CreatedOn returns an ISO-8601 formatted string of the CreatedAt time.
func (v View) CreatedOn() string {
	if v.CreatedAt.IsZero() {
		return ""
	}
	return v.CreatedAt.Format("2006-01-02")
}

// TagKeys returns a sorted list of keys from the tags.
func (v View) TagKeys() []string {
	var keys []string
	for k := range v.Tags {
		if k != "" {
			keys = append(keys, k)
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})
	return keys
}

// TagValues returns a sorted list of key:value pairs from the tags.
func (v View) TagValues() []string {
	var values []string
	for k, v := range v.Tags {
		if k != "" && v != "" {
			values = append(values, k+":"+v)
		}
	}
	sort.Slice(values, func(i, j int) bool {
		return values[i] < values[j]
	})
	return values
}

// CompressedJSON returns a compressed JSON representation of the View.
func (v View) CompressedJSON() []byte {
	j, err := versionary.ToCompressedJSON(v)
	if err != nil {
		return nil
	}
	return j
}

// Validate checks whether the View has all required fields and whether the supplied values
// are valid, returning a list of problems. If the list is empty, then the View is valid.
func (v View) Validate() []string {
	var problems []string
	if v.ID == "" || !tuid.IsValid(tuid.TUID(v.ID)) {
		problems = append(problems, "ID is missing or invalid")
	}
	if v.CreatedAt.IsZero() {
		problems = append(problems, "CreatedAt is missing")
	}
	if v.ExpiresAt.IsZero() {
		problems = append(problems, "ExpiresAt is missing")
	}
	if v.Client.DeviceID == "" || !tuid.IsValid(tuid.TUID(v.Client.DeviceID)) {
		problems = append(problems, "DeviceID is missing or invalid")
	}
	if v.Page.ID != "" && !tuid.IsValid(tuid.TUID(v.Page.ID)) {
		problems = append(problems, "Page ID is invalid")
	}
	if v.Page.URI == "" {
		problems = append(problems, "Page URI is missing")
	} else {
		_, err := url.Parse(v.Page.URI)
		if err != nil {
			problems = append(problems, "Page URI is invalid")
		}
	}
	return problems
}
