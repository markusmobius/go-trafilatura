package trafilatura

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/html"
)

func Test_checkPageLanguage(t *testing.T) {
	raw := `<html lang="de_DE, en_US"><body></body></html>`
	doc, _ := html.Parse(strings.NewReader(raw))
	assert.Equal(t, checkPageLanguage(doc, "de"), true)

	raw = `<html lang="en"><body></body></html>`
	doc, _ = html.Parse(strings.NewReader(raw))
	assert.Equal(t, checkPageLanguage(doc, "it"), false)

	raw = `<html><head><meta http-equiv="content-language" content="en"></head><body></body></html>`
	doc, _ = html.Parse(strings.NewReader(raw))
	assert.Equal(t, checkPageLanguage(doc, "en"), true)

	raw = `<html><head><meta http-equiv="content-language" content="en"></head><body></body></html>`
	doc, _ = html.Parse(strings.NewReader(raw))
	assert.Equal(t, checkPageLanguage(doc, "de"), false)
}
