// This file is part of go-trafilatura, Go package for extracting readable
// content, comments and metadata from a web page. Source available in
// <https://github.com/markusmobius/go-trafilatura>.
// Copyright (C) 2021 Markus Mobius
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by the
// Free Software Foundation, either version 3 of the License, or (at your
// option) any later version.
//
// This program is distributed in the hope that it will be useful, but
// WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY
// or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License
// for more details.
//
// You should have received a copy of the GNU General Public License along
// with this program. If not, see <https://www.gnu.org/licenses/>.

// Code in this file is ported from <https://github.com/adbar/trafilatura>
// which available under GNU GPL v3 license.

package trafilatura

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	nurl "net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-shiori/dom"
	"github.com/markusmobius/go-trafilatura/internal/etree"
	"github.com/markusmobius/go-trafilatura/internal/lru"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/html"
)

var (
	exampleURL, _ = nurl.ParseRequestURI("https://example.org")

	trafilaturaMockFiles = map[string]string{
		"http://exotic_tags": "exotic_tags.html",
	}

	zeroOpts = Options{
		FallbackCandidates: &FallbackConfig{},
		OriginalURL:        exampleURL,
		Config:             zeroConfig,
	}

	zeroConfig = &Config{
		MinOutputSize:    0,
		MinExtractedSize: 0,
	}

	defaultOpts = Options{
		Config: DefaultConfig(),
	}
)

func Test_Trim(t *testing.T) {
	// Test string trimming
	assert.Equal(t, "Test", trim("	Test  "))
	assert.Equal(t, "Test Test", trim("\t\tTest  Test\r\n"))

	elem := etree.Element("body")
	etree.SetText(elem, "Test Text")
	assert.False(t, textFilter(elem))

	etree.SetText(elem, "Instagram")
	assert.True(t, textFilter(elem))

	etree.SetText(elem, "\t\t")
	assert.True(t, textFilter(elem))
}

func Test_ExoticTags(t *testing.T) {
	var result *ExtractResult
	var htmlString string
	var opts Options

	// Cover some edge cases with a specially crafted file
	result = extractMockFile(trafilaturaMockFiles, "http://exotic_tags")
	assert.Contains(t, result.ContentText, "Teletype text")
	assert.Contains(t, result.ContentText, "My new car is silver.")

	// Misformed HTML declaration
	htmlString = `<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01 Transitional//EN" 2012"http://www.w3.org/TR/html4/loose.dtd"><html><head></head><body><p>ABC</p></body></html>`
	result, err := Extract(strings.NewReader(htmlString), zeroOpts)
	assert.Nil(t, err)
	assert.Contains(t, result.ContentText, "ABC")

	// Quotes
	potentialTags := duplicateMap(tagCatalog)
	assert.Nil(t, handleQuotes(etree.Element("blockquote"), nil, zeroOpts))
	assert.Nil(t, handleTable(etree.Element("table"), potentialTags, nil, zeroOpts))

	// Nested <p> with trailing line break
	element, second := etree.Element("p"), etree.Element("p")
	etree.SetText(element, "1st part.")
	etree.SetText(second, "2nd part.")
	etree.Append(element, second)
	etree.SubElement(element, "br")

	converted := handleParagraphs(element, map[string]struct{}{"p": {}}, nil, zeroOpts)
	assert.Equal(t, "<p>1st part. 2nd part.</p>", etree.ToString(converted))

	// Naked div with <br>
	opts = Options{Config: zeroConfig}
	htmlString = `<html><body><main><div>1.<br/>2.<br/>3.<br/></div></main></body></html>`
	result, _ = Extract(strings.NewReader(htmlString), opts)
	assert.Contains(t, result.ContentText, "1. 2. 3.")

	// HTML5: <details>
	opts = Options{Config: zeroConfig}
	htmlString = `<html><body><article><details><summary>Epcot Center</summary><p>Epcot is a theme park at Walt Disney World Resort featuring exciting attractions, international pavilions, award-winning fireworks and seasonal special events.</p></details></article></body></html>`
	result, _ = Extract(strings.NewReader(htmlString), opts)
	assert.Contains(t, result.ContentText, "Epcot Center")
	assert.Contains(t, result.ContentText, "award-winning fireworks")

	htmlString = `<html><body><article><details><summary>Epcot Center</summary><p>Epcot is a theme park at Walt Disney World Resort featuring exciting attractions, international pavilions, award-winning fireworks and seasonal special events.</p></details></article></body></html>`
	result, _ = Extract(strings.NewReader(htmlString), opts)
	assert.Contains(t, result.ContentText, "Epcot Center")
	assert.Contains(t, result.ContentText, "award-winning fireworks")

	// Paywalls
	opts = Options{Config: zeroConfig}
	htmlString = `<html><body><main><p>1</p><p id="paywall">2</p><p>3</p></main></body></html>`
	result, _ = Extract(strings.NewReader(htmlString), opts)
	assert.Equal(t, "1 3", result.ContentText)

	opts = Options{Config: zeroConfig}
	htmlString = `<html><body><main><p>1</p><p id="paywall">2</p><p>3</p></main></body></html>`
	result, _ = Extract(strings.NewReader(htmlString), opts)
	assert.Equal(t, "1 3", result.ContentText)

	// Edge cases
	htmlString = `
	<!DOCTYPE html>
	<html>
	<head>
		<meta charset="UTF-8">
		<title>A weird bug</title>
	</head>
	<body>
		<div>
			<h1>Lorem ipsum dolor sit amet, consectetur adipiscing elit.</h1>
			<h2>Sed et interdum lectus.</h2>
			<p>Quisque molestie nunc eu arcu condimentum fringilla.</p>
			<!-- strong can be changed to b, em, i, u, or kbd -->
			<strong><a></a></strong>
			<h2>Aliquam eget interdum elit, id posuere ipsum.</h2>
			<p>Phasellus lectus erat, hendrerit sed tortor ac, dignissim vehicula metus.</p>
		</div>
	</body>
	</html>`
	opts = Options{IncludeLinks: true, IncludeImages: true}
	result, _ = Extract(strings.NewReader(htmlString), opts)
	assert.NotEmpty(t, result.ContentText)

	htmlString = `
	<html>
	<head>
		<meta charset="UTF-8">
		<title>A weird bug</title>
	</head>
	<body>
		<div id="content">
			<h1>A header</h1>
			<h2>Very specific bug so odd</h2>
			<h3>Nested header</h3>
			<p>Some "hyphenated-word quote" followed by a bit more text line.</p>
			<em>
				<p>em improperly wrapping p here</p>
			</em>
			<p>Text here</p>
		</div>
	</body>
	</html>`
	opts = Options{IncludeLinks: true, IncludeImages: true}
	result, _ = Extract(strings.NewReader(htmlString), opts)
	assert.NotEmpty(t, result.ContentText)
}

func Test_LanguageClassifier(t *testing.T) {
	var lang string

	lang = languageClassifier("Hier ist ein Text auf Deutsch", "")
	assert.Equal(t, "de", lang)

	lang = languageClassifier("Hier ist ein Text auf Deutsch", "")
	assert.NotEqual(t, "en", lang)

	lang = languageClassifier("Hier ist ein Text auf Deutsch", "Die Kommentare sind aber etwas länger.")
	assert.Equal(t, "de", lang)
}

func Test_Cache(t *testing.T) {
	cache := lru.NewCache(2)

	div1 := etree.Element("div")
	p1 := etree.SubElement(div1, "p")
	etree.SetText(p1, "AAAA BBBB AAAA BBBB AAAA BBBB AAAA BBBB AAAA BBBB AAAA BBBB AAAA BBBB AAAA BBBB AAAA BBBB AAAA BBBB AAAA BBBB AAAA BBBB AAAA BBBB")

	assert.False(t, duplicateTest(p1, cache, defaultOpts))
	assert.False(t, duplicateTest(p1, cache, defaultOpts))
	assert.False(t, duplicateTest(div1, cache, defaultOpts))
	assert.True(t, duplicateTest(p1, cache, defaultOpts))

	div2 := etree.Element("div")
	p2 := etree.SubElement(div2, "p")
	etree.SetText(p2, "CCCC DDDD CCCC DDDD CCCC DDDD CCCC DDDD CCCC DDDD CCCC DDDD CCCC DDDD CCCC DDDD CCCC DDDD CCCC DDDD CCCC DDDD")

	assert.False(t, duplicateTest(div2, cache, defaultOpts))
	assert.False(t, duplicateTest(p2, cache, defaultOpts))
	assert.False(t, duplicateTest(div2, cache, defaultOpts))
	assert.True(t, duplicateTest(p2, cache, defaultOpts))

	div3 := etree.Element("div")
	p3 := etree.SubElement(div3, "p")
	etree.SetText(p3, "EEEE FFFF EEEE FFFF EEEE FFFF EEEE FFFF EEEE FFFF EEEE FFFF EEEE FFFF EEEE FFFF EEEE FFFF EEEE FFFF EEEE FFFF EEEE FFFF EEEE FFFF")

	assert.False(t, duplicateTest(div3, cache, defaultOpts))
	assert.False(t, duplicateTest(div3, cache, defaultOpts))
	assert.False(t, duplicateTest(div3, cache, defaultOpts))

	// Since cache haven't been cleared, try the old nodes
	assert.True(t, duplicateTest(p2, cache, defaultOpts))
	assert.True(t, duplicateTest(p3, cache, defaultOpts))
	assert.False(t, duplicateTest(p1, cache, defaultOpts))

	// Clear the cache then try again
	cache.Clear()
	assert.False(t, duplicateTest(p2, cache, defaultOpts))

	// Get wrong key
	val, exist := cache.Get("tralala")
	assert.Zero(t, val)
	assert.False(t, exist)
}

func Test_Formatting(t *testing.T) {
	var r io.Reader
	var opts Options
	var result *ExtractResult
	fnHtml := func(r *ExtractResult) string {
		return etree.ToString(r.ContentNode)
	}

	// Trailing line break
	r = strings.NewReader("<html><body><p>This here is the text.<br/></p></body></html>")
	result, _ = Extract(r, zeroOpts)
	assert.NotContains(t, fnHtml(result), "<br/>")

	// Simple
	r = strings.NewReader("<html><body><p><b>This here is in bold font.</b></p></body></html>")
	result, _ = Extract(r, zeroOpts)
	assert.Contains(t, fnHtml(result), "<p><b>This here is in bold font.</b></p>")

	// Nested
	r = strings.NewReader("<html><body><p><b>This here is in bold and <i>italic</i> font.</b></p></body></html>")
	result, _ = Extract(r, zeroOpts)
	assert.Contains(t, fnHtml(result), "<p><b>This here is in bold and <i>italic</i> font.</b></p>")

	// Empty
	r = strings.NewReader("<html><body><p><b><i></i></b></p></body></html>")
	result, _ = Extract(r, zeroOpts)
	assert.Contains(t, fnHtml(result), "<body></body>")

	// Wild div
	r = strings.NewReader("<html><body><article><div><strong>Wild text</strong></div></article></body></html>")
	result, _ = Extract(r, zeroOpts)
	assert.Contains(t, fnHtml(result), "<p>")
	assert.Contains(t, fnHtml(result), "<strong>Wild text</strong>")
	assert.Equal(t, "Wild text", dom.TextContent(result.ContentNode))

	// Links
	r = strings.NewReader(`<html><body><p><a href="">Link text</a></p></body></html>`)
	result, _ = Extract(r, zeroOpts)
	assert.Equal(t, "Link text", dom.TextContent(result.ContentNode))

	// Line breaks
	r = strings.NewReader(`<html><body><p><br/></p></body></html>`)
	result, _ = Extract(r, zeroOpts)
	assert.Equal(t, "", dom.TextContent(result.ContentNode))

	r = strings.NewReader(`<html><body><p><br/>Here is the text.</p></body></html>`)
	result, _ = Extract(r, zeroOpts)
	assert.Equal(t, "Here is the text.", dom.TextContent(result.ContentNode))

	// Handle formatting tails
	body := etree.Element("body")
	element := etree.SubElement(body, "b")
	etree.SetText(element, "Here is the text.")
	etree.SetTail(element, "And a tail.")

	converted := handleFormatting(element, nil, zeroOpts)
	assert.Equal(t, "<p><b>Here is the text.</b>And a tail.</p>", etree.ToString(converted))

	// Empty elements
	r = strings.NewReader("<html><body><div>\t\n</div><div>There is text here.</div></body></html>")
	result, _ = Extract(r, zeroOpts)
	assert.Equal(t, "<div><p>There is text here.</p></div>", fnHtml(result))

	// List with links
	opts = Options{IncludeLinks: true, Config: zeroConfig}
	r = strings.NewReader(`<html><body><article><ul><li>Number 1</li><li>Number <a href="test.html">2</a></li><li>Number 3</li><p>Test</p></article></body></html>`)
	result, _ = Extract(r, opts)
	assert.Contains(t, fnHtml(result), `<li>Number <a href="test.html">2</a></li>`)

	// (Markdown) formatting within <p>-tag
	rawHTML := `<html><body><p><b>bold</b>, <i>italics</i>, <tt>tt</tt>, <strike>deleted</strike>, <u>underlined</u>, <a href="test.html">link</a> and additional text to bypass detection.</p></body></html>`

	opts = Options{IncludeLinks: false, Config: zeroConfig}
	result, _ = Extract(strings.NewReader(rawHTML), opts)
	assert.Equal(t, "bold, italics, tt, deleted, underlined, link and additional text to bypass detection.", dom.TextContent(result.ContentNode))
	assert.Contains(t, dom.OuterHTML(result.ContentNode), `<p><b>bold</b>, <i>italics</i>, <tt>tt</tt>, <strike>deleted</strike>, <u>underlined</u>, link and additional text to bypass detection.</p>`)

	opts = Options{IncludeLinks: true, Config: zeroConfig}
	result, _ = Extract(strings.NewReader(rawHTML), opts)
	assert.Contains(t, dom.OuterHTML(result.ContentNode), `<p><b>bold</b>, <i>italics</i>, <tt>tt</tt>, <strike>deleted</strike>, <u>underlined</u>, <a href="test.html">link</a> and additional text to bypass detection.</p>`)

	// Line break following formatting
	r = strings.NewReader("<html><body><article><p><strong>Staff Review of the Financial Situation</strong><br>Domestic financial conditions remained accommodative over the intermeeting period.</p></article></body></html>")
	result, _ = Extract(r, zeroOpts)
	assert.Equal(t, "Staff Review of the Financial Situation\nDomestic financial conditions remained accommodative over the intermeeting period.", dom.InnerText(result.ContentNode))

	// Title with formatting
	r = strings.NewReader(`
		<html><body>
			<article>
				<h4 id="1theinoperator">1) The <code>in</code> Operator</h4>
				<p>The easiest way to check if a Python string contains a substring is to use the <code>in</code> operator. The <code>in</code> operator is used to check data structures for membership in Python. It returns a Boolean (either <code>True</code> or <code>False</code>) and can be used as follows:</p>
			</article>
		</body></html>`)
	result, _ = Extract(r, zeroOpts)
	assert.Contains(t, fnHtml(result), `<h4>1) The <code>in</code> Operator</h4>`)
	assert.Contains(t, fnHtml(result), `<p>The easiest way to check if a Python string contains a substring is to use the <code>in</code> operator.`)
	assert.Contains(t, fnHtml(result), `The <code>in</code> operator is used to check data structures for membership in Python.`)
	assert.Contains(t, fnHtml(result), `It returns a Boolean (either <code>True</code> or <code>False</code>) and can be used as follows:`)

	// Double <p> elems
	r = strings.NewReader("<html><body><p>AAA, <p>BBB</p>, CCC.</p></body></html>")
	result, _ = Extract(r, Options{IncludeLinks: true, Config: zeroConfig})
	assert.Contains(t, result.ContentText, "AAA")
	assert.Contains(t, result.ContentText, "BBB")
	assert.Contains(t, result.ContentText, "CCC")
}

func Test_Baseline(t *testing.T) {
	// Blank document
	doc := docFromStr("")
	_, result := baseline(doc)
	assert.Equal(t, "", result)

	// Extract articleBody in JSON+LD
	doc = docFromStr(`<html><body><script type="application/ld+json">{"description":"In letzter Zeit kam man am Begriff \"Hygge\", was so viel wie \"angenehm\" oder \"gemütlich\" bedeutet, ja nicht vorbei. Jetzt macht ihm ein neuer Glücks-Trend ...","image":[{"name":"Mit der Ikigai-Methode wirst du glücklicher","url":"https:\/\/image.brigitte.de\/10973004\/uncropped-0-0\/7d00b2658fd0a3b19e1b161f4657cc20\/Xw\/ikigai--1-.jpg","width":"2048","height":"1366","@type":"ImageObject"},{"name":"Mit der Ikigai-Methode wirst du glücklicher","url":"https:\/\/image.brigitte.de\/10973004\/16x9-1280-720\/bf947c7c24167d7c0adae0be10942d57\/Uf\/ikigai--1-.jpg","width":"1280","height":"720","@type":"ImageObject"},{"name":"Mit der Ikigai-Methode wirst du glücklicher","url":"https:\/\/image.brigitte.de\/10973004\/16x9-938-528\/bf947c7c24167d7c0adae0be10942d57\/JK\/ikigai--1-.jpg","width":"938","height":"528","@type":"ImageObject"},{"name":"Mit der Ikigai-Methode wirst du glücklicher","url":"https:\/\/image.brigitte.de\/10973004\/large1x1-622-622\/f5544b7d67e1be04f7729b130e7e0485\/KN\/ikigai--1-.jpg","width":"622","height":"622","@type":"ImageObject"}],"mainEntityOfPage":{"@id":"https:\/\/www.brigitte.de\/liebe\/persoenlichkeit\/ikigai-macht-dich-sofort-gluecklicher--10972896.html","@type":"WebPage"},"headline":"Ikigai macht dich sofort glücklicher!","datePublished":"2019-06-19T14:29:08+0000","dateModified":"2019-06-19T14:29:10+0000","author":{"name":"BRIGITTE.de","@type":"Organization"},"publisher":{"name":"BRIGITTE.de","logo":{"url":"https:\/\/image.brigitte.de\/11476842\/uncropped-0-0\/f19537e97b9189bf0f25ce924168bedb\/kK\/bri-logo-schema-org.png","width":"167","height":"60","@type":"ImageObject"},"@type":"Organization"},"articleBody":"In letzter Zeit kam man am Begriff \"Hygge\" (\"gemütlich\" oder \"angenehm\") nicht vorbei. Jetzt macht ihm ein neuer Glücks-Trend Konkurrenz: \"Ikigai\". Bist du glücklich? Schwierige Frage, nicht wahr? Viele von uns müssen da erst mal überlegen.","@type":"NewsArticle"}</script></body></html>`)
	_, result = baseline(doc)
	assert.True(t, strings.HasPrefix(result, "In letzter Zeit kam man"))
	assert.True(t, strings.HasSuffix(result, "erst mal überlegen."))

	// Extract from <article> tag
	doc = docFromStr("<html><body><article><b>The article consists of this text.</b></article></body></html>")
	_, result = baseline(doc)
	assert.NotEmpty(t, result)
	assert.Equal(t, "The article consists of this text.", result)

	// Extract from quote
	doc = docFromStr("<html><body><blockquote>This is only a quote but it is better than nothing.</blockquote></body></html>")
	_, result = baseline(doc)
	assert.NotEmpty(t, result)
	assert.Equal(t, "This is only a quote but it is better than nothing.", result)

	doc = docFromStr("<html><body><div>   Document body...   </div><script> console.log('Hello world') </script></body></html>")
	_, result = baseline(doc)
	assert.Equal(t, "Document body...", result)
}

func Test_Language(t *testing.T) {
	// Main text
	assert.Equal(t, "de", languageClassifier("Hier ist ein Text auf Deutsch", ""))

	// Comments text
	assert.Equal(t, "de", languageClassifier("This is English.", "Die Kommentare sind aber etwas länger."))
}

func Test_Filters(t *testing.T) {
	// Helper function
	rRepeatElement := func(element string, repeat int) io.Reader {
		str := fmt.Sprintf("<html><body>%s</body></html>", strings.Repeat(element, repeat))
		return strings.NewReader(str)
	}

	// Recursion limit
	p1 := "<p>abc</p>"
	p2 := "<p><i>abc</i></p>"
	opts := Options{MaxTreeSize: 500}

	result, _ := Extract(rRepeatElement(p1, 50), opts)
	assert.NotNil(t, result)

	result, _ = Extract(rRepeatElement(p1, 501), opts)
	assert.Nil(t, result)

	result, _ = Extract(rRepeatElement(p2, 501), opts)
	assert.Nil(t, result)

	result, _ = Extract(rRepeatElement(p2, 499), opts)
	assert.NotNil(t, result)

	// HTML lang filter
	// No lang
	opts = Options{TargetLanguage: "en"}
	doc := docFromStr(`<html><body></body></html>`)
	assert.True(t, checkHtmlLanguage(doc, opts, false))

	// Lang detection on content
	str := `html><body><article><p>How many ages hence/Shall this our lofty scene be acted over,/In states unborn and accents yet unknown!</p></article></body></html>`

	opts = Options{TargetLanguage: "de"}
	result, _ = Extract(strings.NewReader(str), opts)
	assert.Nil(t, result)

	opts = Options{TargetLanguage: "en"}
	result, _ = Extract(strings.NewReader(str), opts)
	assert.NotNil(t, result)

	// TODO: In original Trafilatura, the value of p3 is set to "In sleep a king,
	// but waking no such matter." which is part of Sonnet 87, classic English poem
	// by Shakespear. Unfortunately, whatlanggo struggle to detect its language.
	// However, when I added the entire closure of Sonnet 87, it works. Need to
	// investigate later.
	p3 := "<p>Thus have I had thee as a dream doth flatter, In sleep a king, but waking no such matter.</p>"
	str = `<html lang="en-US"><body>` + strings.Repeat(p3, 50) + `</body></html>`

	opts = Options{TargetLanguage: "en"}
	result, _ = Extract(strings.NewReader(str), opts)
	assert.NotNil(t, result)

	opts = Options{TargetLanguage: "de"}
	result, _ = Extract(strings.NewReader(str), opts)
	assert.Nil(t, result)

	str = `<html lang="de-DE"><body>` + strings.Repeat(p3, 50) + `</body></html>`
	opts = Options{TargetLanguage: "de"}
	result, _ = Extract(strings.NewReader(str), opts)
	assert.Nil(t, result)

	// http-equiv="content-language"
	opts.TargetLanguage = "en"
	doc = docFromStr(`<html><head><meta http-equiv="content-language" content="en"></head><body></body></html>`)
	assert.True(t, checkHtmlLanguage(doc, opts, false))

	opts.TargetLanguage = "de"
	doc = docFromStr(`<html><head><meta http-equiv="content-language" content="en"></head><body></body></html>`)
	assert.False(t, checkHtmlLanguage(doc, opts, false))

	opts.TargetLanguage = "de"
	doc = docFromStr(`<html><head><meta http-equiv="content-language" content="DE"></head><body></body></html>`)
	assert.True(t, checkHtmlLanguage(doc, opts, false))

	// HTML lang attribute superseded by og:locale
	doc = docFromStr(`<html lang="en-US"><head><meta property="og:locale" content="de_DE" /></head><body></body></html>`)

	opts.TargetLanguage = "de"
	assert.True(t, checkHtmlLanguage(doc, opts, false))

	opts.TargetLanguage = "en"
	assert.False(t, checkHtmlLanguage(doc, opts, false))

	// Last choice: HTML lang attribute
	doc = docFromStr(`<html lang="de_DE, en_US"><body></body></html>`)

	opts.TargetLanguage = "de"
	assert.True(t, checkHtmlLanguage(doc, opts, false))
	assert.True(t, checkHtmlLanguage(doc, opts, true))

	opts.TargetLanguage = "en"
	assert.True(t, checkHtmlLanguage(doc, opts, false))
	assert.True(t, checkHtmlLanguage(doc, opts, true))

	// If strict, lang attribute in <html> should be checked
	opts.TargetLanguage = "it"
	doc = docFromStr(`<html lang="en"><body></body></html>`)
	assert.False(t, checkHtmlLanguage(doc, opts, true)) // is strict
	assert.True(t, checkHtmlLanguage(doc, opts, false)) // not strict

	// Even if in strict mode, lang in <html> attribute is the last choice
	opts.TargetLanguage = "de"
	doc = docFromStr(`<html lang="en-US"><head><meta property="og:locale" content="de_DE" /></head><body></body></html>`)
	assert.True(t, checkHtmlLanguage(doc, opts, true))  // is strict
	assert.True(t, checkHtmlLanguage(doc, opts, false)) // not strict
}

func Test_External(t *testing.T) {
	var f io.Reader
	var opts Options
	var doc *html.Node
	var result *ExtractResult

	// Remove unwanted elements
	doc = docFromStr(`<html><body><footer>Test text</footer></body></html>`)
	sanitizeTree(doc, defaultOpts)
	assert.Empty(t, etree.IterText(doc, " "))

	doc = docFromStr(`<html><body><table><th>Test text</th><tr><td>Test</td></tr></table></body></html>`)
	sanitizeTree(doc, defaultOpts)
	assert.NotEmpty(t, etree.IterText(doc, " "))

	// Strip fancy tags while excluding links and images
	doc = docFromStr(`<html><body><p>Text here <fancy>Test text</fancy><a href="">with a link</a>.</p><img src="test.jpg"/></body></html>`)
	sanitizeTree(doc, defaultOpts)

	mainTree := dom.QuerySelector(doc, "body")
	assert.Len(t, dom.Children(mainTree), 1)

	// Strip fancy tags while including links and images
	opts = Options{IncludeLinks: true, IncludeImages: true}
	doc = docFromStr(`<html><body><p>Text here <fancy>Test text</fancy><a href="">with a link</a>.</p><img src="test.jpg"/></body></html>`)
	sanitizeTree(doc, opts)

	mainTree = dom.QuerySelector(doc, "body")
	aNodes := dom.GetElementsByTagName(mainTree, "a")
	imgNodes := dom.GetElementsByTagName(mainTree, "img")

	assert.Len(t, dom.Children(mainTree), 2)
	assert.NotZero(t, len(aNodes))
	assert.NotZero(t, len(imgNodes))

	// Test language
	opts = Options{TargetLanguage: "en"}
	str := `<html><body>` + strings.Repeat("<p>Non è inglese.</p>", 20) + `</body></html>`
	result, _ = Extract(strings.NewReader(str), opts)
	assert.Nil(t, result)

	// No tables
	f, _ = os.Open(filepath.Join("test-files", "simple", "apache.html"))
	doc, _ = html.Parse(f)

	opts = Options{ExcludeTables: false}
	result, _ = ExtractDocument(doc, opts)
	assert.Contains(t, result.ContentText, "localhost:80")

	opts = Options{ExcludeTables: true}
	result, _ = ExtractDocument(doc, opts)
	assert.NotContains(t, result.ContentText, "localhost:80")

	// Table sub elements
	f, _ = os.Open(filepath.Join("test-files", "simple", "scam.html"))
	doc, _ = html.Parse(f)

	opts = Options{ExcludeTables: true, Config: zeroConfig}
	result, _ = ExtractDocument(doc, opts)
	assert.Empty(t, result.ContentText)

	opts = Options{ExcludeTables: true, FallbackCandidates: &FallbackConfig{}, Config: zeroConfig}
	result, _ = ExtractDocument(doc, opts)
	assert.NotEmpty(t, result.ContentText)
	assert.NotContains(t, result.ContentText, "Uncensored Hosting")
	assert.NotContains(t, result.ContentText, "ChooseBetter")
}

func Test_Images(t *testing.T) {
	// File type
	assert.True(t, isImageFile("test.jpg"))
	assert.False(t, isImageFile("test.txt"))

	// Image handler
	img := handleImage(etree.FromString(`<img src="test.jpg"/>`))
	assert.NotNil(t, img)

	img = handleImage(etree.FromString(`<img data-src="test.jpg" alt="text" title="a title"/>`))
	assert.NotNil(t, img)

	img = handleImage(etree.FromString(`<img other="test.jpg"/>`))
	assert.Nil(t, img)

	// Extension checker
	assert.True(t, isImageFile("test.jpg"))
	assert.False(t, isImageFile("test.txt"))

	// Text element handler
	assert.Nil(t, handleTextElem(etree.Element("img"), nil, nil, defaultOpts))

	// From file
	f, _ := os.Open(filepath.Join("test-files", "simple", "http_sample.html"))
	bt, _ := ioutil.ReadAll(f)

	opts := defaultOpts
	result, _ := Extract(bytes.NewReader(bt), opts)
	contentHtml := dom.OuterHTML(result.ContentNode)
	assert.NotContains(t, contentHtml, `<img src="test.jpg" title="Example image"/>`)

	opts.IncludeImages = true
	result, _ = Extract(bytes.NewReader(bt), opts)
	contentHtml = dom.OuterHTML(result.ContentNode)
	assert.Contains(t, contentHtml, `<img src="test.jpg" title="Example image"/>`)

	// CNN example
	f, _ = os.Open(filepath.Join("test-files", "simple", "cnn-image.html"))
	doc, _ := html.Parse(f)
	img = handleImage(dom.QuerySelector(doc, "img"))
	assert.NotNil(t, img)
	assert.True(t, dom.HasAttribute(img, "alt"))
	assert.True(t, dom.HasAttribute(img, "src"))

	// Modified CNN example
	f, _ = os.Open(filepath.Join("test-files", "simple", "cnn-image-modified.html"))
	doc, _ = html.Parse(f)
	img = handleImage(dom.QuerySelector(doc, "img"))
	assert.NotNil(t, img)
	assert.True(t, dom.HasAttribute(img, "alt"))
	assert.True(t, dom.HasAttribute(img, "src"))
	assert.True(t, strings.HasPrefix(dom.GetAttribute(img, "src"), "http"))
}

func Test_Links(t *testing.T) {
	// Prepare options
	linkOpts := Options{
		IncludeLinks: true,
		Config:       zeroConfig,
	}

	// Test handleTextElem
	processed := handleTextElem(etree.Element("a"), nil, nil, defaultOpts)
	assert.Nil(t, processed)

	// Formatting link
	element := etree.FromString(`<a href="testlink.html">Test link text.</a>`)
	processed = handleFormatting(element, nil, zeroOpts)
	assert.NotNil(t, processed)

	// Extracting links with target
	htmlStr := `<html><body><p><a href="testlink.html">Test link text.</a>This part of the text has to be long enough.</p></body></html>`
	result, _ := Extract(strings.NewReader(htmlStr), zeroOpts)
	assert.NotContains(t, dom.OuterHTML(result.ContentNode), "testlink.html")

	result, _ = Extract(strings.NewReader(htmlStr), linkOpts)
	assert.Contains(t, dom.OuterHTML(result.ContentNode), `<a href="testlink.html">Test link text.</a>This part of the text has to be long enough.`)

	// Relative link conversion
	originalURL, _ := nurl.ParseRequestURI("https://www.example.com")
	result, _ = Extract(strings.NewReader(htmlStr), Options{
		IncludeLinks: true,
		Config:       zeroConfig,
		OriginalURL:  originalURL})
	assert.Contains(t, dom.OuterHTML(result.ContentNode), `<a href="https://www.example.com/testlink.html">Test link text.</a>This part of the text has to be long enough.`)

	// Extracting links without target
	htmlStr = `<html><body><p><a>Test link text.</a>This part of the text has to be long enough.</p></body></html>`
	result, _ = Extract(strings.NewReader(htmlStr), linkOpts)
	assert.Contains(t, dom.OuterHTML(result.ContentNode), `<a>Test link text.</a>This part of the text has to be long enough.`)

	htmlStr = `<html><body><article><a>Segment 1</a><h1><a>Segment 2</a></h1><p>Segment 3</p></article></body></html>`
	result, _ = Extract(strings.NewReader(htmlStr), linkOpts)
	assert.Contains(t, result.ContentText, "1")
	assert.Contains(t, result.ContentText, "2")
	assert.Contains(t, result.ContentText, "3")

	// Extracting document with links, from file
	f, _ := os.Open(filepath.Join("test-files", "simple", "http_sample.html"))
	bt, _ := ioutil.ReadAll(f)

	result, _ = Extract(bytes.NewReader(bt), zeroOpts)
	assert.NotContains(t, dom.OuterHTML(result.ContentNode), "testlink.html")

	result, _ = Extract(bytes.NewReader(bt), linkOpts)
	assert.Contains(t, dom.OuterHTML(result.ContentNode), "testlink.html")

	// Test license link
	htmlStr = `<html><body><p>Test text under <a rel="license" href="">CC BY-SA license</a>.</p></body></html>`
	result, _ = Extract(strings.NewReader(htmlStr), linkOpts)
	assert.Contains(t, dom.OuterHTML(result.ContentNode), "<a>CC BY-SA license</a>")
}

func Test_ExtractionOptions(t *testing.T) {
	var opts Options
	var result *ExtractResult
	htmlStr := `<html>
		<head>
			<meta http-equiv="content-language" content="EN" />
		</head>
		<body>
			<div="article-body">
				<p>Text.<!-- comment --></p>
			</div>
		</body>
	</html>`

	opts = Options{Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlStr), opts)
	assert.NotNil(t, result)

	opts = Options{HasEssentialMetadata: true, Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlStr), opts)
	assert.Nil(t, result)

	opts = Options{TargetLanguage: "de", Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlStr), opts)
	assert.Nil(t, result)
}

func Test_PrecisionRecall(t *testing.T) {
	var opts Options
	var result *ExtractResult
	var htmlStr string

	// Basic test
	htmlStr = `<html><body><p>This here is the text.</p></body></html>`

	opts = Options{FavorPrecision: true, Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlStr), opts)
	assert.NotNil(t, result)

	opts = Options{FavorRecall: true, Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlStr), opts)
	assert.NotNil(t, result)

	// Teaser text
	htmlStr = `<html><body>
		<div class="article-body">
			<div class="teaser-content">
				<p>This here is a teaser text.</p>
			</div>
			<p>This here is the text.</p>
		</div>
	</body></html>`

	opts = Options{FavorRecall: true, Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlStr), opts)
	assert.Contains(t, result.ContentText, "teaser text")

	opts = Options{FavorRecall: false, Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlStr), opts)
	assert.NotContains(t, result.ContentText, "teaser text")

	opts = Options{FavorPrecision: true, Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlStr), opts)
	assert.NotContains(t, result.ContentText, "teaser text")

	// Never extracted
	htmlStr = `<html><body><article><div><p>
		<a href="test.html">1.</a>
		<br />
		<a href="test2.html">2.</a>
	</p></div></article></body></html>`

	opts = Options{FavorRecall: true, Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlStr), opts)
	assert.NotContains(t, result.ContentText, "1")

	opts = Options{FavorPrecision: true, Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlStr), opts)
	assert.NotContains(t, result.ContentText, "1")
}

func Test_TableProcessing(t *testing.T) {
	var opts Options
	var processedTable *html.Node
	var nodeValues []string
	potentialTags := duplicateMap(tagCatalog)
	iterNodeValues := func(root *html.Node) []string {
		var nodeValues []string
		for _, node := range etree.Iter(root) {
			nodeTag := dom.TagName(node)
			nodeText := trim(etree.Text(node))
			if nodeText == "" {
				nodeValues = append(nodeValues, nodeTag)
			} else {
				nodeValues = append(nodeValues, nodeTag+"-"+nodeText)
			}
		}
		return nodeValues
	}

	// Simple table
	tableSimpleCell := etree.FromString(`<table><tr><td>cell1</td><td>cell2</td></tr><tr><td>cell3</td><td>cell4</td></tr></table>`)
	processedTable = handleTable(tableSimpleCell, potentialTags, nil, defaultOpts)
	assert.Equal(t, []string{"table", "tr", "td-cell1", "td-cell2", "tr", "td-cell3", "td-cell4"}, iterNodeValues(processedTable))

	// If a cell contains 'exotic' tags, they are cleaned during the extraction
	// Process and the content is merged with the parent e.g. <td>
	tableCellWithChildren := etree.FromString(`<table><tr><td><p>text</p><p>more text</p></td></tr></table>`)
	processedTable = handleTable(tableCellWithChildren, potentialTags, nil, defaultOpts)
	assert.Equal(t, `<table><tr><td><p>text</p><p>more text</p></td></tr></table>`, dom.OuterHTML(processedTable))

	// Complex table that hasn't been cleaned yet
	complexPage := docFromStr(`
	<html><body>
		<article>
			<table>
			<tbody>
				<tr>
				<td><small>text<br></small>
					<h4>more_text</h4>
				</td>
				<td><a href='link'>linktext</a></td>
				</tr>
			</tbody>
			</table>
		</article>
	</body></html>`)
	opts = Options{IncludeLinks: true, Config: zeroConfig}
	result, _ := ExtractDocument(complexPage, opts)
	assert.Contains(t, dom.OuterHTML(result.ContentNode), `<table><tr><td>text<h4>more_text</h4></td></tr></table>`)

	// Table cell with text and child
	tableCellWithTextAndChild := etree.FromString(`<table><tr><td>text<lb/><p>more text</p></td></tr></table>`)
	processedTable = handleTable(tableCellWithTextAndChild, potentialTags, nil, defaultOpts)
	assert.Equal(t, `<table><tr><td>text<p>more text</p></td></tr></table>`, dom.OuterHTML(processedTable))

	// Table cell with link
	tableCellWithLink := etree.FromString(`<table><tr><td><a href='test'>link</a></td></tr></table>`)
	processedTable = handleTable(tableCellWithLink, potentialTags, nil, defaultOpts)
	nodeValues = iterNodeValues(dom.QuerySelector(processedTable, "td"))
	assert.Equal(t, []string{"td", "p"}, nodeValues)

	// Table with head
	tableWithHead := etree.FromString(`
	<table>
		<tr><th>Month</th><th>Days</th></tr>
		<tr><td>January</td><td>31</td></tr>
		<tr><td>February</td><td>28</td></tr>
	</table>`)
	processedTable = handleTable(tableWithHead, potentialTags, nil, defaultOpts)
	assert.Equal(t, 3, len(dom.Children(processedTable)))

	firstRow := dom.Children(processedTable)[0]
	firstRowCells := dom.Children(firstRow)
	assert.Equal(t, 2, len(firstRowCells))
	assert.Equal(t, "th", dom.TagName(firstRowCells[0]))
	assert.Equal(t, "th", dom.TagName(firstRowCells[1]))
	assert.Equal(t, "Month", dom.TextContent(firstRowCells[0]))
	assert.Equal(t, "Days", dom.TextContent(firstRowCells[1]))

	// Table with head span
	tableWithHeadSpan := etree.FromString(`
	<table>
		<tr>
			<th>Name</th>
			<th>Adress</th>
			<th colspan="2">Phone</th>
		</tr>
		<tr>
			<td>Jane Doe</td>
			<td>test@example.com</td>
			<td>phone 1</td>
			<td>phone 2</td>
		</tr>
	</table>`)
	processedTable = handleTable(tableWithHeadSpan, potentialTags, nil, defaultOpts)
	assert.Equal(t, 2, len(dom.Children(processedTable)))

	firstRow = dom.Children(processedTable)[0]
	firstRowCells = dom.Children(firstRow)
	assert.Equal(t, 3, len(firstRowCells))
	assert.Equal(t, "th", dom.TagName(firstRowCells[0]))
	assert.Equal(t, "th", dom.TagName(firstRowCells[1]))
	assert.Equal(t, "th", dom.TagName(firstRowCells[2]))

	// Table cell with formatting
	tableCellWithFormatting := etree.FromString(`<table><tr><td><mark>highlighted text</mark></td></tr></table>`)
	processedTable = handleTable(tableCellWithFormatting, potentialTags, nil, defaultOpts)
	firstCell := dom.QuerySelector(processedTable, "td")
	assert.NotNil(t, firstCell)
	assert.Equal(t, dom.OuterHTML(firstCell), `<td><mark>highlighted text</mark></td>`)

	// Table cell with span
	tableCellWithSpan := etree.FromString(`<table><tr><td><span style='sth'>span text</span></td></tr></table>`)
	processedTable = handleTable(tableCellWithSpan, potentialTags, nil, defaultOpts)
	firstCell = dom.QuerySelector(processedTable, "td")
	assert.NotNil(t, firstCell)
	assert.Equal(t, dom.OuterHTML(firstCell), `<td><p></p></td>`)

	// Table with nested elements
	tableNestedElements := docFromStr(`
	<html><body>
		<article>
			<table>
				<tr>
					<td><b>Present Tense</b></td>
					<td>I buy</td>
					<td>you buy</td>
					<td>he/she/it buys</td>
					<td>we buy</td>
					<td>you buy</td>
					<td>they buy</td>
				</tr>
			</table>
		</article>
	</body></html>`)
	opts = Options{IncludeLinks: true, Config: zeroConfig}
	result, _ = ExtractDocument(tableNestedElements, opts)
	assert.Contains(t, dom.OuterHTML(result.ContentNode), ``+
		`<tr>`+
		`<td><b>Present Tense</b></td>`+
		`<td>I buy</td>`+
		`<td>you buy</td>`+
		`<td>he/she/it buys</td>`+
		`<td>we buy</td>`+
		`<td>you buy</td>`+
		`<td>they buy</td>`+
		`</tr>`)

	// Table with links
	// TODO: further tests and adjustsments
	tableWithLinks := docFromStr(`` +
		`<html><body><article><table><tr><td><a href="test.html">` +
		strings.Repeat("ABCD", 100) +
		`</a></td></tr></table></article></body></html>`)
	opts = Options{IncludeLinks: true, Config: zeroConfig}
	result, _ = ExtractDocument(tableWithLinks, opts)
	assert.NotContains(t, result.ContentText, "ABCD")

	// Nested table 1
	tableNested1 := docFromStr(`
	<html><body><article>
		<table><th>1</th><table><tr><td>2</td></tr></table></table>
	</article></body></html>`)
	opts = Options{IncludeLinks: true, Config: zeroConfig}
	result, _ = ExtractDocument(tableNested1, opts)
	// TODO: all elements are there, but output not nested
	// TODO: th conversion
	assert.Contains(t, dom.OuterHTML(result.ContentNode), "<th>1</th>")
	assert.Contains(t, dom.OuterHTML(result.ContentNode), "<td>2</td>")

	// Nested table 2
	tableNested2 := etree.FromString(`
	<table><tr><td>
		<table><tr><td>1</td></tr></table>
	</td></tr></table>`)
	processedTable = handleTable(tableNested2, potentialTags, nil, defaultOpts)
	assert.Equal(t, []string{"table", "tr", "td", "td-1"}, iterNodeValues(processedTable))

	// Nested table - complex
	tableNestedComplex := etree.FromString(`
	<table>
		<tr>
			<td>
				<table><tr><td>1</td></tr></table>
			</td>
			<td>text1</td>
		</tr>
		<tr>
			<td>text2</td>
		</tr>
	</table>`)
	processedTable = handleTable(tableNestedComplex, potentialTags, nil, defaultOpts)
	assert.Equal(t, []string{"table", "tr", "td", "td-1", "td-text1", "tr", "td-text2"}, iterNodeValues(processedTable))

	// Table with list
	tableWithList := etree.FromString(`
	<table>
		<tr><td>
			<p>a list</p>
			<ul>
				<li>one</li>
				<li>two</li>
			</ul>
		</td></tr>
	</table>`)
	processedTable = handleTable(tableWithList, potentialTags, nil, defaultOpts)
	assert.Equal(t, []string{"table", "tr", "td", "p-a list", "ul"}, iterNodeValues(processedTable))

	// Broken table 1 (broken as in uncommon structure)
	tableBroken1 := etree.FromString(`<table><td>cell1</td><tr><td>cell2</td></tr></table>`)
	processedTable = handleTable(tableBroken1, potentialTags, nil, defaultOpts)
	assert.Equal(t, []string{"table", "tr", "td-cell1", "tr", "td-cell2"}, iterNodeValues(processedTable))

	// Broken table 2
	tableBroken2 := docFromStr(`<table><tr><p>text</p></tr><tr><td>cell</td></tr></table>`)
	tableBroken2 = dom.QuerySelector(tableBroken2, "table")
	processedTable = handleTable(tableBroken2, potentialTags, nil, defaultOpts)
	assert.Equal(t, []string{"table", "tr", "td-cell"}, iterNodeValues(processedTable))

	// Table nested in figure https://github.com/adbar/trafilatura/issues/301
	tableInFigure := docFromStr(`<html><body><article><figure><table><th>1</th><tr><td>2</td></tr></table></figure></article></body></html>`)
	result, _ = ExtractDocument(tableInFigure, zeroOpts)
	assert.Contains(t, dom.OuterHTML(result.ContentNode), "<th>1</th>")
	assert.Contains(t, dom.OuterHTML(result.ContentNode), "<td>2</td>")
}

func Test_ListProcessing(t *testing.T) {
	var opts Options
	var processedList *html.Node
	var result *ExtractResult
	var strResult string
	iterNodeValues := func(root *html.Node) []string {
		var nodeValues []string
		for _, node := range etree.Iter(root) {
			nodeTag := dom.TagName(node)
			nodeText := trim(etree.Text(node))
			if nodeText == "" {
				nodeValues = append(nodeValues, nodeTag)
			} else {
				nodeValues = append(nodeValues, nodeTag+"-"+nodeText)
			}
		}
		return nodeValues
	}

	// Malformed lists (common error)
	listMalformed := etree.FromString(`
	<ul>Description of the list:
		<li>List item 1</li>
		<li>List item 2</li>
		<li>List item 3</li>
	</ul>`)
	opts = Options{Config: zeroConfig}
	processedList = handleLists(listMalformed, nil, opts)
	strResult = etree.ToString(processedList)
	assert.Equal(t, 3, strings.Count(strResult, "List item"))
	assert.Contains(t, strResult, "Description")

	// Nested list
	listNested := docFromStr(`
	<html><body><article>
		<ul>
			<li>Coffee</li>
			<li>Tea
				<ul>
					<li>Black tea</li>
					<li>Green tea</li>
				</ul>
			</li>
			<li>Milk</li>
		</ul>
	</article></body></html>`)
	opts = Options{Config: zeroConfig}
	result, _ = ExtractDocument(listNested, opts)
	assert.Contains(t, noSpace(dom.OuterHTML(result.ContentNode)), noSpace(`
	<ul>
		<li>Coffee</li>
		<li>Tea
			<ul>
				<li>Black tea</li>
				<li>Green tea</li>
			</ul>
		</li>
		<li>Milk</li>
	</ul>`))

	// Description list
	listDescription := docFromStr(`
	<html><body><article>
		<dl>
			<dt>Coffee</dt>
			<dd>Black hot drink</dd>
			<dt>Milk</dt>
			<dd>White cold drink</dd>
		</dl>
	</article></body></html>`)
	opts = Options{Config: zeroConfig}
	result, _ = ExtractDocument(listDescription, opts)
	assert.Contains(t, noSpace(dom.OuterHTML(result.ContentNode)), noSpace(`
	<dl>
		<dt>Coffee</dt>
		<dd>Black hot drink</dd>
		<dt>Milk</dt>
		<dd>White cold drink</dd>
	</dl>`))

	// Item with child
	listItemWithChild := etree.FromString(`<ul><li><p>text</p></li></ul>`)
	processedList = handleLists(listItemWithChild, nil, defaultOpts)
	assert.Equal(t, []string{"ul", "li", "p-text"}, iterNodeValues(processedList))

	listItemWithTextAndChild := etree.FromString(`<ul><li>text1<p>text2</p></li></ul>`)
	processedList = handleLists(listItemWithTextAndChild, nil, defaultOpts)
	assert.Equal(t, []string{"ul", "li-text1", "p-text2"}, iterNodeValues(processedList))

	listItemWithBr := etree.FromString(`<ul><li>text<br/>more text</li></ul>`)
	processedList = handleLists(listItemWithBr, nil, defaultOpts)
	assert.Equal(t, []string{"ul", "li-text", "br"}, iterNodeValues(processedList))

	// List with text outside item
	listWithTextOutside := etree.FromString(`<ul>header<li>text</li></ul>`)
	processedList = handleLists(listWithTextOutside, nil, defaultOpts)
	assert.Equal(t, []string{"ul", "li-header", "li-text"}, iterNodeValues(processedList))

	// Simple list
	listSimple := etree.FromString(`<ul>   <li>text</li></ul>`)
	processedList = handleLists(listSimple, nil, defaultOpts)
	assert.Len(t, dom.Children(processedList), 1)

	// List item with tail
	listItemWithTail := etree.FromString(`<ul><li>text</li>tail</ul>`)
	processedList = handleLists(listItemWithTail, nil, defaultOpts)
	children := dom.Children(processedList)
	assert.Len(t, children, 1)
	assert.Equal(t, "text tail", dom.TextContent(children[0]))

	// List item with child and tail #1
	listItemWithChildAndTail := etree.FromString(`<ul><li><p>text</p></li>tail</ul>`)
	processedList = handleLists(listItemWithChildAndTail, nil, defaultOpts)
	children = dom.Children(processedList)
	assert.Len(t, children, 1)

	firstItem := children[0]
	assert.Empty(t, etree.Tail(firstItem))
	assert.Equal(t, "tail", etree.Tail(dom.Children(firstItem)[0]))

	// List item with child and tail #2
	listItemWithChildAndTail = etree.FromString(`<ul><li><p>text</p>tail1</li>tail</ul>`)
	processedList = handleLists(listItemWithChildAndTail, nil, defaultOpts)
	children = dom.Children(processedList)
	assert.Len(t, children, 1)

	firstItem = children[0]
	assert.Empty(t, etree.Tail(firstItem))
	assert.Equal(t, "tail1 tail", etree.Tail(dom.Children(firstItem)[0]))

	// List item with child and tail #3
	listItemWithChildAndTail = etree.FromString("<ul><li><p>text</p>\n</li>tail</ul>")
	processedList = handleLists(listItemWithChildAndTail, nil, defaultOpts)
	children = dom.Children(processedList)
	assert.Len(t, children, 1)

	firstItem = children[0]
	assert.Empty(t, etree.Tail(firstItem))
	assert.Equal(t, "tail", etree.Tail(dom.Children(firstItem)[0])) // TODO: TIFU

	// List item with tail and nested list
	listItemWithTailAndNestedList := etree.FromString(`
	<ul>
		<li><ul><li>text</li></ul></li>
		tail
	</ul>`)
	processedList = handleLists(listItemWithTailAndNestedList, nil, defaultOpts)
	assert.Equal(t, "tail", etree.Tail(dom.QuerySelector(processedList, "li ul")))
}

func Test_CodeBlocks(t *testing.T) {
	var opts Options
	var result *ExtractResult
	var htmlInput string
	var htmlOutput string

	// Highlight js
	htmlInput = `` +
		`<div class="s-prose js-post-body" itemprop="text">` +
		`<p>Code:</p>` +
		`<pre class="lang-sql s-code-block"><code class="hljs language-sql">code\n` +
		`<span class="hljs-keyword">highlighted</span> more <span class="hljs-keyword">code</span>` +
		`</code></pre>` +
		`</div>`

	opts = Options{Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlInput), opts)

	htmlOutput = etree.ToString(result.ContentNode)
	assert.Contains(t, htmlOutput, `<code>code\nhighlighted more code</code>`)
	assert.NotContains(t, htmlOutput, `<q>`)

	// Github
	htmlInput = `` +
		`<div class="highlight highlight-source-shell notranslate position-relative overflow-auto" dir="auto"><pre>$ pip install PyGithub</pre><div class="zeroclipboard-container position-absolute right-0 top-0">` +
		`<clipboard-copy aria-label="Copy" class="ClipboardButton btn js-clipboard-copy m-2 p-0 tooltipped-no-delay" data-copy-feedback="Copied!" data-tooltip-direction="w" value="$ pip install PyGithub" tabindex="0" role="button" style="display: inherit;">` +
		`<svg aria-hidden="true" height="16" viewBox="0 0 16 16" version="1.1" width="16" data-view-component="true" class="octicon octicon-copy js-clipboard-copy-icon m-2">` +
		`<path d="M0 6.75C0 5.784.784 5 1.75 5h1.5a.75.75 0 0 1 0 1.5h-1.5a.25.25 0 0 0-.25.25v7.5c0 .138.112.25.25.25h7.5a.25.25 0 0 0 .25-.25v-1.5a.75.75 0 0 1 1.5 0v1.5A1.75 1.75 0 0 1 9.25 16h-7.5A1.75 1.75 0 0 1 0 14.25Z"></path><path d="M5 1.75C5 .784 5.784 0 6.75 0h7.5C15.216 0 16 .784 16 1.75v7.5A1.75 1.75 0 0 1 14.25 11h-7.5A1.75 1.75 0 0 1 5 9.25Zm1.75-.25a.25.25 0 0 0-.25.25v7.5c0 .138.112.25.25.25h7.5a.25.25 0 0 0 .25-.25v-7.5a.25.25 0 0 0-.25-.25Z"></path>` +
		`</svg>` +
		`<svg aria-hidden="true" height="16" viewBox="0 0 16 16" version="1.1" width="16" data-view-component="true" class="octicon octicon-check js-clipboard-check-icon color-fg-success d-none m-2">` +
		`<path d="M13.78 4.22a.75.75 0 0 1 0 1.06l-7.25 7.25a.75.75 0 0 1-1.06 0L2.22 9.28a.751.751 0 0 1 .018-1.042.751.751 0 0 1 1.042-.018L6 10.94l6.72-6.72a.75.75 0 0 1 1.06 0Z"></path>` +
		`</svg>` +
		`</clipboard-copy>` +
		`</div></div>`

	opts = Options{Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlInput), opts)

	htmlOutput = etree.ToString(result.ContentNode)
	assert.Contains(t, htmlOutput, `<code>$ pip install PyGithub</code>`)
	assert.NotContains(t, htmlOutput, `<q>`)

	// Inline code
	htmlInput = `<div><p>paragraph</p><p>here is <code>some</code> code</p></div>`

	opts = Options{Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlInput), opts)

	htmlOutput = etree.ToString(result.ContentNode)
	assert.Contains(t, htmlOutput, `<code>some</code>`)
	assert.NotContains(t, htmlOutput, `<q>`)

	// W3 Schools
	htmlInput = `` +
		`<div class="w3-example"><h3>Example</h3>` +
		`<p>Create a class named Person, use the __init__() function to assign values ` +
		`for name and age:</p>` +
		`<div class="w3-code notranslate pythonHigh"><span class="pythoncolor" style="color:black"><span class="pythonnumbercolor" style="color:red">` +
		`</span>  <span class="pythonkeywordcolor" style="color:mediumblue">class</span> Person:<br>&nbsp; <span class="pythonkeywordcolor" style="color:mediumblue">def</span> __init__(self, name, age):<br>&nbsp;&nbsp;&nbsp; <span class="pythonnumbercolor" style="color:red">` +
		`</span>  self.name = name<br>&nbsp;&nbsp;&nbsp; self.age = age<br><br>p1 = Person(<span class="pythonstringcolor" style="color:brown">"John"</span>, <span class="pythonnumbercolor" style="color:red">` +
		`</span>  <span class="pythonnumbercolor" style="color:red">36</span>)<br><span class="pythonnumbercolor" style="color:red">` +
		`</span>  <br><span class="pythonkeywordcolor" style="color:mediumblue">print</span>(p1.name)<br><span class="pythonkeywordcolor" style="color:mediumblue">print</span>(p1.age) </span></div>` +
		`</div>`

	opts = Options{Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlInput), opts)

	htmlOutput = dom.OuterHTML(result.ContentNode)
	// TODO: this one is different than the original, because the formatting
	// is different. Investigate later.
	assert.Contains(t, htmlOutput, `<code>class Person:`)
	assert.NotContains(t, htmlOutput, `<q>`)
}
