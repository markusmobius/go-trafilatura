// This file is part of go-trafilatura, Go package for extracting readable
// content, comments and metadata from a web page. Source available in
// <https://github.com/markusmobius/go-trafilatura>.
//
// Copyright (C) 2021 Markus Mobius
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code in this file is ported from <https://github.com/adbar/trafilatura>
// which available under Apache 2.0 license.

package trafilatura

import (
	"bytes"
	"fmt"
	"io"
	"maps"
	nurl "net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-shiori/dom"
	"github.com/markusmobius/go-htmldate"
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
		EnableFallback: true,
		OriginalURL:    exampleURL,
		Config:         zeroConfig,
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
	potentialTags := maps.Clone(tagCatalog)
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
			<p>Phasellus lectus erat, hendrerit sed tortor ac, dignissim vehicula metus.<br/></p>
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
			<p>Text here<br/></p>
			<h3>More articles</h3>
		</div>
	</body>
	</html>`

	opts = Options{IncludeLinks: true, IncludeImages: true}
	for _, focus := range []ExtractionFocus{Balanced, FavorRecall, FavorPrecision} {
		opts.Focus = focus
		result, _ = Extract(strings.NewReader(htmlString), opts)
		assert.Contains(t, result.ContentText, "em improperly wrapping p here")
		assert.True(t, strings.HasSuffix(result.ContentText, "Text here"))
	}
}

func Test_HtmlProcessing(t *testing.T) {
	var opts Options
	var node *html.Node
	var htmlString string
	var result *ExtractResult

	strToNode := func(s string) *html.Node {
		n, _ := dom.FastParse(strings.NewReader(s))
		return n
	}

	// Paywalls
	opts = Options{Config: zeroConfig}
	htmlString = `<html><body><main><p>1</p><p id="premium">2</p><p>3</p></main></body></html>`
	result, _ = Extract(strings.NewReader(htmlString), opts)
	assert.Equal(t, "1 3", result.ContentText)

	// Test tail of node deleted if set as text
	node = strToNode(`<div><p></p>tail</div>`)
	node = processNode(dom.QuerySelector(node, "p"), nil, defaultOpts)
	assert.Equal(t, "tail", etree.Text(node))
	assert.Equal(t, "", etree.Tail(node))

	node = strToNode(`<ol><li></li>text in tail</ol>`)
	node = processNode(dom.QuerySelector(node, "li"), nil, defaultOpts)
	assert.Equal(t, "text in tail", etree.Text(node))
	assert.Equal(t, "", etree.Tail(node))

	node = strToNode(`<p><br/>tail</p>`)
	node = processNode(dom.QuerySelector(node, "br"), nil, defaultOpts)
	assert.Equal(t, "", etree.Text(node))
	assert.Equal(t, "tail", etree.Tail(node))

	node = strToNode(`<div><p>some text</p>tail</div>`)
	node = processNode(dom.QuerySelector(node, "p"), nil, defaultOpts)
	assert.Equal(t, "some text", etree.Text(node))
	assert.Equal(t, "tail", etree.Tail(node))

	// Text node handler
	node = strToNode(`<p><a href="url"><strong>bold</strong>inner</a>outer</p>`)
	node = handleTextNode(dom.QuerySelector(node, "a"), nil, false, false, defaultOpts)
	assert.Equal(t, "outer", etree.Tail(node))

	node = strToNode(`<p><a href="url">text</a>tail</p>`)
	node = handleTextNode(dom.QuerySelector(node, "a"), nil, false, false, defaultOpts)
	assert.Equal(t, "text", etree.Text(node))
	assert.Equal(t, "tail", etree.Tail(node))

	node = strToNode(`<p><a href="url"></a>tail</p>`)
	node = handleTextNode(dom.QuerySelector(node, "a"), nil, false, false, defaultOpts)
	assert.Equal(t, "tail", etree.Text(node))
	assert.Equal(t, "", etree.Tail(node))

	node = strToNode(`<p><a href="url">text<strong>bold</strong></a>tail</p>`)
	node = handleTextNode(dom.QuerySelector(node, "a"), nil, false, false, defaultOpts)
	assert.Equal(t, "text", etree.Text(node))
	assert.Equal(t, "tail", etree.Tail(node))
}

func Test_LanguageClassifier(t *testing.T) {
	var lang string
	var htmlInput string
	var result *ExtractResult

	// Content text only
	lang = languageClassifier("Hier ist ein Text auf Deutsch", "")
	assert.Equal(t, "de", lang)

	lang = languageClassifier("Hier ist ein Text auf Deutsch", "")
	assert.NotEqual(t, "en", lang)

	// Comments text
	lang = languageClassifier("Hier ist ein Text auf Deutsch", "Die Kommentare sind aber etwas länger.")
	assert.Equal(t, "de", lang)

	lang = languageClassifier("This is English.", "Die Kommentare sind aber etwas länger.")
	assert.Equal(t, "de", lang)

	// Extraction result
	htmlInput = `<html><body><p>Texto en español</p></body></html>`
	result, _ = Extract(strings.NewReader(htmlInput), zeroOpts)
	assert.Equal(t, "es", result.Metadata.Language)

	// Originally they use "Texte en français" here but it's too short which
	// make whatlanggo confuse it with Afrikaans, so here I pick a longer sentence.
	htmlInput = `<html><body><p>Après la pluie, le beau temps.</p></body></html>`
	result, _ = Extract(strings.NewReader(htmlInput), zeroOpts)
	assert.Equal(t, "fr", result.Metadata.Language)
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

	// Title
	r = strings.NewReader("<html><body><article><h3>Title</h3><p><b>This here is in bold font.</b></p></article></body></html>")
	result, _ = Extract(r, zeroOpts)
	assert.Contains(t, fnHtml(result), "<h3>Title</h3>")
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
	assert.Equal(t, "Wild text", result.ContentText)

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

	opts = Options{ExcludeTables: true, EnableFallback: true, Config: zeroConfig}
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
	img := handleImage(nil)
	assert.Nil(t, img)

	img = handleImage(etree.FromString(`<img src="test.jpg"/>`))
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
	bt, _ := io.ReadAll(f)

	// Comparing between includeImages = true and false
	opts := defaultOpts
	opts.Config = zeroConfig

	result, _ := Extract(bytes.NewReader(bt), opts)
	contentHtml := dom.OuterHTML(result.ContentNode)
	assert.NotContains(t, contentHtml, `<img src="test.jpg" title="Example image"/>`)

	opts.IncludeImages = true
	result, _ = Extract(bytes.NewReader(bt), opts)
	contentHtml = dom.OuterHTML(result.ContentNode)
	assert.Contains(t, contentHtml, `<img src="test.jpg" title="Example image"/>`)

	// From string
	var str string
	str = `<html><body><article><p><img data-src="test.jpg" alt="text" title="a title"/></p></article></body></html>`
	result, _ = Extract(strings.NewReader(str), opts)
	contentHtml = dom.OuterHTML(result.ContentNode)
	assert.Contains(t, contentHtml, `<img src="test.jpg" alt="text" title="a title"/>`)

	str = `<html><body><article><p><img other="test.jpg" alt="text" title="a title"/></p></article></body></html>`
	result, _ = Extract(strings.NewReader(str), opts)
	contentHtml = dom.OuterHTML(result.ContentNode)
	assert.Equal(t, `<body></body>`, contentHtml)

	str = `<html><body><article><div><p><img data-src="test.jpg" alt="text" title="a title"/></p></div></article></body></html>`
	result, _ = Extract(strings.NewReader(str), opts)
	contentHtml = dom.OuterHTML(result.ContentNode)
	assert.Contains(t, contentHtml, `<img src="test.jpg" alt="text" title="a title"/>`)

	str = `<html><body><article><div><p><img data-src-small="test.jpg" alt="text" title="a title"/></p></div></article></body></html>`
	result, _ = Extract(strings.NewReader(str), opts)
	contentHtml = dom.OuterHTML(result.ContentNode)
	assert.Contains(t, contentHtml, `<img src="test.jpg" alt="text" title="a title"/>`)

	str = `<img src="data:image/jpeg;base64,iVBORw0KGgoAAAANSUhEUgAAAAUAAAAFCAYAAACNbyblAAAAHElEQVQI12P4//8/w38GIAXDIBKE0DHxgljNBAAO9TXL0Y4OHwAAAABJRU5ErkJggg==" alt="text"></img>`
	result, _ = Extract(strings.NewReader(str), opts)
	contentHtml = dom.OuterHTML(result.ContentNode)
	assert.Equal(t, `<body></body>`, contentHtml)

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
	bt, _ := io.ReadAll(f)

	result, _ = Extract(bytes.NewReader(bt), zeroOpts)
	assert.NotContains(t, dom.OuterHTML(result.ContentNode), "testlink.html")

	result, _ = Extract(bytes.NewReader(bt), linkOpts)
	assert.Contains(t, dom.OuterHTML(result.ContentNode), "testlink.html")

	// Test license link
	htmlStr = `<html><body><p>Test text under <a rel="license" href="">CC BY-SA license</a>.</p></body></html>`
	result, _ = Extract(strings.NewReader(htmlStr), linkOpts)
	assert.Contains(t, dom.OuterHTML(result.ContentNode), "<a>CC BY-SA license</a>")

	// Link in p, length threshold
	// var opts Options
	htmlStr = `<html><body><article><p><a>` + strings.Repeat("abcd", 20) + `</a></p></article></body></html>`

	opts := Options{Config: zeroConfig, Focus: Balanced}
	result, _ = Extract(strings.NewReader(htmlStr), opts)
	assert.Contains(t, dom.TextContent(result.ContentNode), "abcd")

	opts = Options{Config: zeroConfig, Focus: FavorPrecision}
	result, _ = Extract(strings.NewReader(htmlStr), opts)
	assert.Empty(t, dom.TextContent(result.ContentNode))
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

	// Try HtmlDate config
	htmlStr = `<html><head/><body>` +
		strings.Repeat(`<p>ABC def ghi jkl.</p>`, 1000) +
		`<p>Posted on 1st Dec 2019<.</p></body></html>`
	doc, _ := dom.FastParse(strings.NewReader(htmlStr))

	dateOpts := htmldate.Options{UseOriginalDate: true}
	opts = Options{Config: zeroConfig, HtmlDateOptions: &dateOpts}

	meta := extractMetadata(doc, opts)
	assert.NotZero(t, meta.Date)

	dateOpts.SkipExtensiveSearch = true
	meta = extractMetadata(doc, opts)
	assert.Zero(t, meta.Date)
}

func Test_PrecisionRecall(t *testing.T) {
	var opts Options
	var result *ExtractResult
	var htmlStr string

	// Basic test
	htmlStr = `<html><body><p>This here is the text.</p></body></html>`

	opts = Options{Focus: FavorPrecision, Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlStr), opts)
	assert.NotNil(t, result)

	opts = Options{Focus: FavorRecall, Config: zeroConfig}
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

	opts = Options{Focus: FavorRecall, Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlStr), opts)
	assert.Contains(t, result.ContentText, "teaser text")

	opts = Options{Focus: Balanced, Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlStr), opts)
	assert.NotContains(t, result.ContentText, "teaser text")

	opts = Options{Focus: FavorPrecision, Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlStr), opts)
	assert.NotContains(t, result.ContentText, "teaser text")

	// Never extracted
	htmlStr = `<html><body><article><div><p>
		<a href="test.html">1.</a>
		<br />
		<a href="test2.html">2.</a>
	</p></div></article></body></html>`

	opts = Options{Focus: FavorRecall, Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlStr), opts)
	assert.NotContains(t, result.ContentText, "1")

	opts = Options{Focus: FavorPrecision, Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlStr), opts)
	assert.NotContains(t, result.ContentText, "1")

	// Only found when favor recall
	htmlStr = `<html><body>
		<div class="article-body">
			<p>content</p>
			<p class="link">Test</p>
		</div>
	</body></html>`

	opts = Options{Focus: FavorRecall, Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlStr), opts)
	assert.Contains(t, result.ContentText, "content")
	assert.Contains(t, result.ContentText, "Test")

	opts = Options{Focus: FavorPrecision, Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlStr), opts)
	assert.Contains(t, result.ContentText, "content")
	assert.NotContains(t, result.ContentText, "Test")

	htmlStr = `<html><body><article>
		<aside><p>Here is the text.</p></aside>
	</article></body></html>`

	opts = Options{Focus: Balanced, Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlStr), opts)
	assert.NotEqual(t, "Here is the text.", result.ContentText)

	opts = Options{Focus: FavorRecall, Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlStr), opts)
	assert.Equal(t, "Here is the text.", result.ContentText)

	htmlStr = `<html><body><div>
		<h2>Title</h2>
		<small>Text.</small>
	</div></body></html>`
	opts = Options{Focus: FavorRecall, Config: zeroConfig, EnableFallback: true}
	result, _ = Extract(strings.NewReader(htmlStr), opts)
	assert.NotEmpty(t, result.ContentText)

	htmlStr = `<html><body><div>
		<span>Text.</span>
	</div></body></html>`

	opts = Options{Focus: FavorPrecision, Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlStr), opts)
	assert.Empty(t, result.ContentText)

	opts = Options{Focus: FavorRecall, Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlStr), opts)
	assert.Equal(t, "Text.", result.ContentText)
}

func Test_TableProcessing(t *testing.T) {
	var opts Options
	var processedTable *html.Node
	var nodeValues []string
	potentialTags := maps.Clone(tagCatalog)
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
	processedTable = handleTable(dom.Clone(tableWithList, true), potentialTags, nil, defaultOpts)
	assert.Equal(t, []string{"table", "tr", "td", "p-a list", "ul"}, iterNodeValues(processedTable))

	recallOpts := Options{Config: DefaultConfig(), Focus: FavorRecall}
	processedTable = handleTable(dom.Clone(tableWithList, true), potentialTags, nil, recallOpts)
	assert.Equal(t, []string{"table", "tr", "td", "p-a list", "ul", "li-one", "li-two"}, iterNodeValues(processedTable))

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
	listItemWithTailAndNestedList := etree.FromString(`` +
		`<ul>` +
		`<li><ul><li>text</li></ul></li>` +
		`tail` +
		`</ul>`)
	processedList = handleLists(listItemWithTailAndNestedList, nil, defaultOpts)
	assert.Equal(t, "tail", etree.Tail(dom.QuerySelector(processedList, "li ul")))
}

func Test_CodeBlocks(t *testing.T) {
	var opts Options
	var result *ExtractResult
	var htmlInput string
	var htmlOutput string
	var expected string

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

	htmlOutput = trim(dom.OuterHTML(result.ContentNode))
	expected = `` +
		`<code> ` +
		`class Person:<br/> ` +
		`def __init__(self, name, age):<br/> ` +
		`self.name = name<br/> ` +
		`self.age = age<br/>` +
		`<br/>p1 = Person(&#34;John&#34;, 36)<br/> ` +
		`<br/>print(p1.name)<br/>print(p1.age) ` +
		`</code>`
	assert.Contains(t, htmlOutput, expected)
	assert.NotContains(t, htmlOutput, `<q>`)

	// Pip
	htmlInput = `
	<div>
		<p>Code:</p>
		<pre lang="python3">
			<span class="kn">import</span>
			<span class="nn">openai</span>
			<span class="kn">from</span>
			<span class="nn">openai_function_call</span>
			<span class="kn">import</span>
			<span class="n">openai_function</span>
		</pre>
	</div>`

	opts = Options{Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlInput), opts)

	htmlOutput = trim(dom.OuterHTML(result.ContentNode))
	expected = `<code> import openai from openai_function_call import openai_function </code>`
	assert.Contains(t, htmlOutput, expected)
	assert.NotContains(t, htmlOutput, `<q>`)

	// Medium JS
	htmlInput = `
	<div>
		<p>Code:</p>
		<pre class="lw lx ly lz ma nq nr ns bo nt ba bj">
			<span id="fe48" class="nu mo ev nr b bf nv nw l nx ny" data-selectable-paragraph="">
				<span class="hljs-keyword">import</span> openai_function<br><br>
				<span class="hljs-meta">@openai_function</span>
			</span>
		</pre>
	</div>`

	opts = Options{Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlInput), opts)

	htmlOutput = trim(dom.OuterHTML(result.ContentNode))
	expected = `<code> import openai_function<br/><br/> @openai_function </code>`
	assert.Contains(t, htmlOutput, expected)
	assert.NotContains(t, htmlOutput, `<q>`)

	// Medium SSR
	htmlInput = `
	<div>
		<p>Code:</p>
		<pre class="lw lx ly lz ma nq nr ns bo nt ba bj">
			<span id="fe48" class="nu mo ev nr b bf nv nw l nx ny">
				import openai_function<br><br>
				@openai_functiondef sum(a:int, b:int):<br/>
				&quot;&quot;&quot;Sum description adds a + b&quot;&quot;&quot;
			</span>
		</pre>
	</div>`

	opts = Options{Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlInput), opts)

	htmlOutput = trim(dom.OuterHTML(result.ContentNode))
	expected = `<code> import openai_function<br/><br/> @openai_functiondef sum(a:int, b:int):<br/> &#34;&#34;&#34;Sum description adds a + b&#34;&#34;&#34; </code>`
	assert.Contains(t, htmlOutput, expected)
	assert.NotContains(t, htmlOutput, `<q>`)

	// Code element
	htmlInput = `<div><p>Code:</p><pre><code><span>my code</span></code></pre>`

	opts = Options{Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlInput), opts)

	htmlOutput = trim(dom.OuterHTML(result.ContentNode))
	assert.Contains(t, htmlOutput, `<code>my code</code>`)
	assert.NotContains(t, htmlOutput, `<q>`)
}

func Test_PruneSelector(t *testing.T) {
	// Helper function
	createDoc := func(strContent string) *html.Node {
		str := fmt.Sprintf(`<html><body>%s</body></html>`, strContent)
		doc, _ := dom.FastParse(strings.NewReader(str))
		return doc
	}

	// Variable helper
	var result *ExtractResult
	opts := Options{
		Config:         zeroConfig,
		EnableFallback: true,
	}

	// Example HTML
	p := `<p>abc</p>`
	h1 := `<h1>ABC</h1>`
	h2 := `<h2>42</h2>`
	doc1 := createDoc(strings.Repeat(p, 50))
	doc2 := createDoc(h1 + strings.Repeat(p, 50))
	doc3 := createDoc(h1 + h2 + strings.Repeat(p, 50))

	// Sanity check
	result, _ = ExtractDocument(doc1, opts)
	assert.NotEmpty(t, dom.OuterHTML(result.ContentNode))

	result, _ = ExtractDocument(doc2, opts)
	assert.NotEmpty(t, dom.OuterHTML(result.ContentNode))

	result, _ = ExtractDocument(doc3, opts)
	assert.NotEmpty(t, dom.OuterHTML(result.ContentNode))

	// With prune selector
	opts.PruneSelector = "p"
	result, _ = ExtractDocument(doc1, opts)
	assert.Equal(t, "", result.ContentText)

	opts.PruneSelector = "p"
	result, _ = ExtractDocument(doc2, opts)
	assert.Equal(t, "ABC", result.ContentText)

	opts.PruneSelector = "p, h1"
	result, _ = ExtractDocument(doc2, opts)
	assert.Equal(t, "", result.ContentText)

	opts.PruneSelector = "p, h1"
	result, _ = ExtractDocument(doc3, opts)
	assert.Equal(t, "42", result.ContentText)
}

func Test_MixedContentExtraction(t *testing.T) {
	htmlContent := `<html><body><p>Text here</p><img src="img.jpg"/><video src="video.mp4"/></body></html>`
	result, _ := Extract(strings.NewReader(htmlContent), zeroOpts)
	assert.Equal(t, "Text here", result.ContentText)
}

func Test_NonStdHtmlEntities(t *testing.T) {
	htmlContent := `<html><body><p>Text &customentity; more text</p></body></html>`
	result, _ := Extract(strings.NewReader(htmlContent), zeroOpts)
	assert.Equal(t, "Text &customentity; more text", result.ContentText)
}

func Test_LargeDocPerformance(t *testing.T) {
	htmlContent := `<html><body>` + strings.Repeat(`<p>Sample text</p>`, 1000) + `</body></html>`
	start := time.Now()
	Extract(strings.NewReader(htmlContent), zeroOpts)
	assert.LessOrEqual(t, time.Since(start), 5*time.Second)
}
