package trafilatura

import (
	"strings"
	"testing"

	"github.com/go-shiori/dom"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/html"
)

func Test_cleanHTML(t *testing.T) {
	rawHTML := `<html><body>
		<table><a href="">Link</a></table>
		<img src="test.jpg" />
		<u>Underlined</u>
		<sub>Text</sub>
		<sup>Text</sup>
	</body></html>`

	doc, _ := html.Parse(strings.NewReader(rawHTML))
	cleanHTML(doc, false, true)

	assert.Empty(t, dom.QuerySelectorAll(doc, "table"))
	assert.NotEmpty(t, dom.QuerySelectorAll(doc, "img"))
}
