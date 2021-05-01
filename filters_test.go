package trafilatura

import (
	"strings"
	"testing"

	"github.com/go-shiori/dom"
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

func Test_textFilter(t *testing.T) {
	div := dom.CreateElement("div")
	dom.SetTextContent(div, "Test Text")
	assert.False(t, textFilter(div))

	dom.SetTextContent(div, "Instagram")
	assert.True(t, textFilter(div))

	dom.SetTextContent(div, "\t\t")
	assert.True(t, textFilter(div))
}

func Test_duplicateFilter(t *testing.T) {
	cache := NewCache(2)

	div1 := dom.CreateElement("div")
	p1 := dom.CreateElement("p")
	dom.SetTextContent(p1, "AAAA BBBB AAAA BBBB AAAA BBBB AAAA BBBB AAAA BBBB AAAA BBBB AAAA BBBB AAAA BBBB AAAA BBBB AAAA BBBB AAAA BBBB AAAA BBBB AAAA BBBB")
	dom.AppendChild(div1, p1)

	assert.False(t, duplicateFilter(cache, p1))
	assert.False(t, duplicateFilter(cache, p1))
	assert.False(t, duplicateFilter(cache, div1))
	assert.True(t, duplicateFilter(cache, p1))

	div2 := dom.CreateElement("div")
	p2 := dom.CreateElement("p")
	dom.SetTextContent(p2, "CCCC DDDD CCCC DDDD CCCC DDDD CCCC DDDD CCCC DDDD CCCC DDDD CCCC DDDD CCCC DDDD CCCC DDDD CCCC DDDD CCCC DDDD")
	dom.AppendChild(div2, p2)

	assert.False(t, duplicateFilter(cache, div2))
	assert.False(t, duplicateFilter(cache, p2))
	assert.False(t, duplicateFilter(cache, div2))
	assert.True(t, duplicateFilter(cache, p2))

	div3 := dom.CreateElement("div")
	p3 := dom.CreateElement("p")
	dom.SetTextContent(p3, "EEEE FFFF EEEE FFFF EEEE FFFF EEEE FFFF EEEE FFFF EEEE FFFF EEEE FFFF EEEE FFFF EEEE FFFF EEEE FFFF EEEE FFFF EEEE FFFF EEEE FFFF")
	dom.AppendChild(div3, p3)

	assert.False(t, duplicateFilter(cache, div3))
	assert.False(t, duplicateFilter(cache, div3))
	assert.False(t, duplicateFilter(cache, div3))

	// Since cache haven't been cleared, try the old nodes
	assert.True(t, duplicateFilter(cache, p2))
	assert.True(t, duplicateFilter(cache, p3))
	assert.False(t, duplicateFilter(cache, p1))

	// Clear the cache then try again
	cache.Clear()
	assert.False(t, duplicateFilter(cache, p2))
}
