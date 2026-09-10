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
	"strings"
	"testing"

	"github.com/go-shiori/dom"
	"github.com/markusmobius/go-trafilatura/internal/etree"
	"github.com/markusmobius/go-trafilatura/internal/selector"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/html"
)

func Test_processNode(t *testing.T) {
	var node *html.Node

	node = etree.FromString(`<div><p></p>tail</div>`)
	node = dom.QuerySelector(node, "p")
	node = processNode(node, nil, defaultOpts)
	assert.Equal(t, "tail", etree.Text(node))
	assert.Equal(t, "", etree.Tail(node))

	node = etree.FromString(`<ul><li></li>text in tail</ul>`)
	node = dom.QuerySelector(node, "li")
	node = processNode(node, nil, defaultOpts)
	assert.Equal(t, "text in tail", etree.Text(node))
	assert.Equal(t, "", etree.Tail(node))

	node = etree.FromString(`<p><br/>tail</p>`)
	node = dom.QuerySelector(node, "br")
	node = processNode(node, nil, defaultOpts)
	assert.Equal(t, "", etree.Text(node))
	assert.Equal(t, "tail", etree.Tail(node))

	node = etree.FromString(`<div><p>some text</p>tail</div>`)
	node = dom.QuerySelector(node, "p")
	node = processNode(node, nil, defaultOpts)
	assert.Equal(t, "some text", etree.Text(node))
	assert.Equal(t, "tail", etree.Tail(node))
}

func Test_pruneUnwantedNodes(test *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "first child tail stays in parent",
			input:    `<div><p><span>span</span> span tail</p> p tail </div>`,
			expected: `<div><p> span tail</p> p tail </div>`,
		},
		{
			name:     "parent text is preserved",
			input:    `<div>start<span>discard</span>end<p>keep</p></div>`,
			expected: `<div>startend<p>keep</p></div>`,
		},
		{
			name:     "previous sibling tail is preserved",
			input:    `<div><p>first</p> between<span>discard</span> after<p>last</p></div>`,
			expected: `<div><p>first</p> between after<p>last</p></div>`,
		},
		{
			name:     "nested discarded nodes",
			input:    `<div><span>outer<span>inner</span>inner tail</span>outer tail</div>`,
			expected: `<div>outer tail</div>`,
		},
	}
	queries := []selector.Rule{func(node *html.Node) bool {
		return dom.TagName(node) == "span"
	}}
	for _, testCase := range testCases {
		test.Run(testCase.name, func(test *testing.T) {
			node := etree.FromString(testCase.input)
			result := pruneUnwantedNodes(node, queries)
			assert.Equal(test, testCase.expected, etree.ToString(result))
			assert.Same(test, node, result)
		})
	}
}

func Test_HTMLParser_Upstream22(test *testing.T) {
	for _, prefix := range []string{
		`<!DOCTYPE html>`,
		`<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">`,
		`<!DOCTYPE html SYSTEM "https://example.org/document/type.dtd">`,
	} {
		result, err := Extract(strings.NewReader(prefix+`<html><body><article><p>Content after the doctype.</p></article></body></html>`), Options{Config: zeroConfig})
		if assert.NoError(test, err) {
			assert.Equal(test, "Content after the doctype.", result.ContentText)
		}
	}
}

func Test_Selectors_Upstream22(test *testing.T) {
	for _, testCase := range []struct {
		input   string
		discard bool
	}{
		{`<div id="article" class="sharbuttons">text</div>`, true},
		{`<div id="article" class="social">text</div>`, true},
		{`<div id="social" class="article">text</div>`, true},
		{`<div id="soc" class="ial">text</div>`, false},
		{`<div class="xg1">text</div>`, false},
		{`<div id="article" class="cookie-recipe">text</div>`, false},
		{`<div class="cookie-recipe" id="article">text</div>`, true},
	} {
		node := etree.FromString(testCase.input)
		discard := false
		for _, rule := range selector.OverallDiscardedContent {
			discard = discard || rule(node)
		}
		assert.Equal(test, testCase.discard, discard, testCase.input)
	}
	for _, class := range []string{"permalink", "headline-link", "linkage", "linked-text"} {
		node := etree.FromString(`<p class="` + class + `">text</p>`)
		assert.False(test, selector.PrecisionDiscardedContent[1](node), class)
	}
	for _, class := range []string{"link", "other link", "link other"} {
		node := etree.FromString(`<p class="` + class + `">text</p>`)
		assert.True(test, selector.PrecisionDiscardedContent[1](node), class)
	}
	node := etree.FromString(`<div class="FullText">text</div>`)
	assert.True(test, selector.Content[2](node))
	for _, class := range []string{"author-name", "authorname", "AuthorName", "authorName"} {
		node := etree.FromString(`<span class="` + class + `">Author Name</span>`)
		assert.True(test, selector.MetaAuthor[0](node), class)
	}
	node = etree.FromString(`<span class="author name">Author Name</span>`)
	assert.False(test, selector.MetaAuthor[0](node))
	assert.True(test, selector.MetaAuthor[1](node))
	metadata := extractMetadata(docFromStr(`<html><body><span class="author name">Generic Author</span><span class="authorname">Specific Author</span></body></html>`), defaultOpts)
	assert.Equal(test, "Specific Author", metadata.Author)
}

func Test_convertTags_LinkedImages(test *testing.T) {
	node := etree.FromString(`<div><a href="/x"><picture><img src="a.jpg"><img src="b.jpg"></picture><img src="c.jpg"></a></div>`)
	opts := Options{IncludeImages: true, IncludeLinks: true}
	convertTags(node, opts)
	assert.Equal(test, `<div><img src="a.jpg"/><img src="b.jpg"/><img src="c.jpg"/></div>`, etree.ToString(node))

	node = etree.FromString(`<div><a href="/x">image link<img src="a.jpg"></a></div>`)
	convertTags(node, opts)
	assert.Equal(test, `<div><a href="/x">image link</a><img src="a.jpg"/></div>`, etree.ToString(node))
	_, discard := linkDensityTest(node, opts)
	assert.False(test, discard)
}

func Test_CodeBlocks_Indicators(test *testing.T) {
	for _, code := range []string{`print("hello")`, "if ready {\n    run()\n}", "value = 1\n    value += 1"} {
		test.Run(code, func(test *testing.T) {
			node := etree.FromString("<div><pre>" + code + "</pre></div>")
			convertTags(node, zeroOpts)
			assert.NotNil(test, dom.QuerySelector(node, "code"))
			assert.Equal(test, code, dom.TextContent(node))
		})
	}
	node := etree.FromString("<div><pre>ordinary quoted prose</pre></div>")
	convertTags(node, zeroOpts)
	assert.Nil(test, dom.QuerySelector(node, "code"))
}

func Test_FencedFrames(test *testing.T) {
	for _, fallback := range []bool{false, true} {
		for _, input := range []string{
			`<html><body><article><p>real text</p><fencedframe><p>advertising payload</p></fencedframe><p>more text</p></article></body></html>`,
			`<html><body><fencedframe><article><h1>advertising payload</h1><p>advertising payload</p></article></fencedframe></body></html>`,
		} {
			result, err := Extract(strings.NewReader(input), Options{Config: zeroConfig, EnableFallback: fallback})
			if assert.NoError(test, err) && assert.NotNil(test, result) {
				assert.NotContains(test, result.ContentText, "advertising payload")
			}
		}
	}
}

func Test_Cleaning_Upstream22(test *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"nobr text", `<div><p>before <nobr>unbroken words</nobr> after</p></div>`, `<div><p>before unbroken words after</p></div>`},
		{"removed element tail", `<div><p>before<script>discard</script>after</p></div>`, `<div><p>beforeafter</p></div>`},
		{"empty sub and sup", `<div><p>before<sup></sup>middle<sub></sub>after</p></div>`, `<div><p>beforemiddleafter</p></div>`},
		{"FAQ heading", `<div><strong class="schema-faq-question">The question?</strong><p>The answer.</p></div>`, `<div><h3>The question?</h3><p>The answer.</p></div>`},
	}
	for _, testCase := range testCases {
		test.Run(testCase.name, func(test *testing.T) {
			node := etree.FromString(testCase.input)
			docCleaning(node, zeroOpts)
			convertTags(node, zeroOpts)
			assert.Equal(test, testCase.expected, etree.ToString(node))
		})
	}
	for _, role := range []string{"presentation", "none"} {
		node := etree.FromString(`<div><table role="` + role + `"><tr><td><p>layout text</p></td></tr></table></div>`)
		docCleaning(node, zeroOpts)
		assert.Nil(test, dom.QuerySelector(node, "table"))
		assert.Contains(test, dom.TextContent(node), "layout text")
	}
}

func Test_LinkDensity_Upstream22(test *testing.T) {
	for _, input := range []string{
		`<ul><li><p><a href="/x">Linked list item</a></p></li></ul>`,
		`<table><tr><td><p><a href="/x">Linked table cell</a></p></td></tr></table>`,
	} {
		node := etree.FromString(input)
		deleteByLinkDensity(node, zeroOpts, false, "p")
		assert.NotNil(test, dom.QuerySelector(node, "p"))
	}
	node := etree.FromString(`<div>` + strings.Repeat(`<a href="/news">A moderately long news headline with enough text</a>`, 10) + `</div>`)
	_, discard := linkDensityTest(node, zeroOpts)
	assert.True(test, discard)

	node = etree.FromString(`<div>` + strings.Repeat(`<a href="/article">`+strings.Repeat("Editorial sentence. ", 10)+`</a>`, 5) + `</div>`)
	_, discard = linkDensityTest(node, zeroOpts)
	assert.False(test, discard)

	node = etree.FromString(`<table><tr><td>` + strings.Repeat("Informative cell content. ", 20) + `<a href="/flag"><img src="flag.png"></a></td></tr></table>`)
	assert.False(test, linkDensityTestTables(node, zeroOpts))
}
