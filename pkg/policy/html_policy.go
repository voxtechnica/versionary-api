package policy

import "github.com/microcosm-cc/bluemonday"

var (
	PlainText = bluemonday.StrictPolicy()
	RichText  = newPolicyRichText()
	Content   = newPolicyContent()
)

// newPolicyRichText returns an HTML Policy that allows only limited HTML tags. It is used to sanitize
// user-generated content. Reference: https://pkg.go.dev/github.com/microcosm-cc/bluemonday#section-readme
func newPolicyRichText() *bluemonday.Policy {
	p := bluemonday.NewPolicy()
	p.AllowElements("p", "b", "strong", "i", "em", "code", "s", "sup", "sub")
	return p
}

// newPolicyContent returns an HTML Policy that allows only limited HTML tags. It is used to sanitize
// user-generated content. Reference: https://pkg.go.dev/github.com/microcosm-cc/bluemonday#section-readme
func newPolicyContent() *bluemonday.Policy {
	p := bluemonday.NewPolicy()
	p.AllowElements("p", "b", "strong", "i", "em", "s", "sup", "sub", "code", "ul", "ol", "li", "a")
	p.AllowAttrs("href").OnElements("a")
	p.AllowLists()
	p.AllowRelativeURLs(true)
	p.AllowURLSchemes("mailto", "http", "https")
	p.RequireNoFollowOnLinks(false)
	return p
}
