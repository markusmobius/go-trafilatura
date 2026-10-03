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

package trafilatura

import (
	"strings"
	"testing"

	"github.com/go-shiori/dom"
	"github.com/stretchr/testify/assert"
)

// mdFromStr renders the <body> of a raw HTML string into Markdown. It bypasses
// the extraction pipeline so that the converter can be tested in isolation.
func mdFromStr(s string) string {
	doc := docFromStr(s)
	body := dom.QuerySelector(doc, "body")
	return htmlToMarkdown(body)
}

func Test_Markdown_Headings(t *testing.T) {
	assert.Equal(t, "# Title", mdFromStr("<body><h1>Title</h1></body>"))
	assert.Equal(t, "## Title", mdFromStr("<body><h2>Title</h2></body>"))
	assert.Equal(t, "###### Title", mdFromStr("<body><h6>Title</h6></body>"))

	// Heading followed by a paragraph, separated by a blank line.
	got := mdFromStr("<body><h2>Title</h2><p>Hello world.</p></body>")
	assert.Equal(t, "## Title\n\nHello world.", got)
}

func Test_Markdown_InlineFormatting(t *testing.T) {
	assert.Equal(t, "This is **bold** text.",
		mdFromStr("<body><p>This is <b>bold</b> text.</p></body>"))

	assert.Equal(t, "This is *italic* text.",
		mdFromStr("<body><p>This is <i>italic</i> text.</p></body>"))

	assert.Equal(t, "This is `code` text.",
		mdFromStr("<body><p>This is <code>code</code> text.</p></body>"))

	assert.Equal(t, "This is ~~deleted~~ text.",
		mdFromStr("<body><p>This is <del>deleted</del> text.</p></body>"))

	// Combination of several inline styles within a paragraph.
	got := mdFromStr(`<body><p><b>bold</b>, <i>italics</i>, <tt>tt</tt>, <strike>deleted</strike>, <u>underlined</u>.</p></body>`)
	assert.Equal(t, "**bold**, *italics*, `tt`, ~~deleted~~, __underlined__.", got)
}

func Test_Markdown_Links(t *testing.T) {
	got := mdFromStr(`<body><p>See <a href="https://example.com">this link</a> now.</p></body>`)
	assert.Equal(t, "See [this link](https://example.com) now.", got)

	// Link without target falls back to the bracketed text only.
	got = mdFromStr(`<body><p>See <a>this link</a> now.</p></body>`)
	assert.Equal(t, "See [this link] now.", got)
}

func Test_Markdown_Images(t *testing.T) {
	got := mdFromStr(`<body><p><img src="/logo.png" alt="logo"></p></body>`)
	assert.Equal(t, "![logo](/logo.png)", got)

	// Title is prepended to the alt text.
	got = mdFromStr(`<body><p><img src="/logo.png" title="Brand" alt="logo"></p></body>`)
	assert.Equal(t, "![Brand logo](/logo.png)", got)
}

func Test_Markdown_Lists(t *testing.T) {
	got := mdFromStr("<body><ul><li>Item 1</li><li>Item 2</li><li>Item 3</li></ul></body>")
	assert.Equal(t, "- Item 1\n- Item 2\n- Item 3", got)

	// List item containing inline formatting and a link.
	got = mdFromStr(`<body><ul><li>Number <a href="test.html">2</a></li></ul></body>`)
	assert.Equal(t, "- Number [2](test.html)", got)
}

func Test_Markdown_CodeBlock(t *testing.T) {
	got := mdFromStr("<body><code>line1<br>line2</code></body>")
	assert.Equal(t, "```\nline1\nline2\n```", got)
}

func Test_Markdown_Table(t *testing.T) {
	got := mdFromStr("<body><table><tr><th>A</th><th>B</th></tr><tr><td>1</td><td>2</td></tr></table></body>")

	// Header row, separator and body row should all be present.
	assert.Contains(t, got, "| A")
	assert.Contains(t, got, "| B")
	assert.Contains(t, got, "|---|---|")
	assert.Contains(t, got, "| 1")
	assert.Contains(t, got, "| 2")

	// Separator must sit between the header and the body row.
	headIdx := strings.Index(got, "|---|---|")
	bodyIdx := strings.Index(got, "| 1")
	assert.True(t, headIdx >= 0 && bodyIdx > headIdx)
}

func Test_Markdown_ExtractResult(t *testing.T) {
	rawHTML := `<html><body><article>
		<h2>Heading</h2>
		<p>This is <b>bold</b> and additional text to bypass the extraction size detection.</p>
		<ul><li>First</li><li>Second</li></ul>
	</article></body></html>`

	result, err := Extract(strings.NewReader(rawHTML), zeroOpts)
	assert.NoError(t, err)

	md := result.ContentMarkdown()
	assert.Contains(t, md, "## Heading")
	assert.Contains(t, md, "**bold**")
	assert.Contains(t, md, "- First")
	assert.Contains(t, md, "- Second")

	// Empty result must not panic.
	var empty *ExtractResult
	assert.Equal(t, "", empty.ContentMarkdown())
}
