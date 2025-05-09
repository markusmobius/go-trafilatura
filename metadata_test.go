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
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/html"
)

func Test_Metadata(t *testing.T) {
	rawHTML := `
	<html>

	<head>
		<title>Test Title</title>
		<meta itemprop="author" content="Jenny Smith" />
		<meta property="og:url" content="https://example.org" />
		<meta itemprop="description" content="Description" />
		<meta property="og:published_time" content="2017-09-01" />
		<meta name="article:publisher" content="The Newspaper" />
		<meta property="image" content="https://example.org/example.jpg" />
	</head>

	<body>
		<p class="entry-categories">
			<a href="https://example.org/category/cat1/">Cat1</a>,
			<a href="https://example.org/category/cat2/">Cat2</a>
		</p>
		<p>
			<a href="https://creativecommons.org/licenses/by-sa/4.0/" rel="license">CC BY-SA</a>
		</p>
	</body>

	</html>`

	metadata := testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Test Title", metadata.Title)
	assert.Equal(t, "Jenny Smith", metadata.Author)
	assert.Equal(t, "https://example.org", metadata.URL)
	assert.Equal(t, "Description", metadata.Description)
	assert.Equal(t, "The Newspaper", metadata.Sitename)
	assert.Equal(t, "2017-09-01", metadata.Date.Format("2006-01-02"))
	assert.Equal(t, []string{"Cat1", "Cat2"}, metadata.Categories)
	assert.Equal(t, "CC BY-SA 4.0", metadata.License)
	assert.Equal(t, "https://example.org/example.jpg", metadata.Image)
}

func Test_Metadata_Titles(t *testing.T) {
	var rawHTML string
	isEqual := func(rawHTML string, expected any) {
		metadata := testGetMetadataFromHTML(rawHTML)
		assert.Equal(t, expected, metadata.Title)
	}

	rawHTML = `<html><body><h3 class="title">T</h3><h3 id="title"></h3></body></html>`
	isEqual(rawHTML, "")

	rawHTML = `<html><head><title>Test Title</title><meta property="og:title" content=" " /></head><body><h1>First</h1></body></html>`
	isEqual(rawHTML, "First")

	rawHTML = `<html><head><title>Test Title</title><meta name="title" content=" " /></head><body><h1>First</h1></body></html>`
	isEqual(rawHTML, "First")

	rawHTML = `<html><head><title>Test Title</title></head><body></body></html>`
	isEqual(rawHTML, "Test Title")

	rawHTML = `<html><body><h1>First</h1><h1>Second</h1></body></html>`
	isEqual(rawHTML, "First")

	rawHTML = `<html><body><h1>   </h1><div class="post-title">Test Title</div></body></html>`
	isEqual(rawHTML, "Test Title")

	rawHTML = `<html><body><h2 class="block-title">Main menu</h2><h1 class="article-title">Test Title</h1></body></html>`
	isEqual(rawHTML, "Test Title")

	rawHTML = `<html><body><h2>First</h2><h1>Second</h1></body></html>`
	isEqual(rawHTML, "Second")

	rawHTML = `<html><body><h2>First</h2><h2>Second</h2></body></html>`
	isEqual(rawHTML, "First")

	rawHTML = `<html><body><title></title></body></html>`
	isEqual(rawHTML, "")

	rawHTML = `<html><head><title> - Home</title></head><body/></html>`
	isEqual(rawHTML, "- Home")

	rawHTML = `<html><head><title>My Title » My Website</title></head><body/></html>`
	isEqual(rawHTML, "My Title") // TODO: and metadata.sitename == "My Website"

	// Try from file
	metadata := testGetMetadataFromFile("simple/metadata-title.html")
	assert.Equal(t, "Semantic satiation", metadata.Title)
}

func Test_Metadata_normalizeAuthors(t *testing.T) {
	// Alias for shorter test
	na := normalizeAuthors
	isEqual := assert.Equal

	isEqual(t, "Abc", na("", "abc"))
	isEqual(t, "Steve Steve", na("", "Steve Steve 123"))
	isEqual(t, "Steve Steve", na("", "By Steve Steve"))
	isEqual(t, "Seán Federico O'Murchú", na("", "Seán Federico O'Murchú"))
	isEqual(t, "John Doe", na("", "John Doe"))
	isEqual(t, "Alice; Bob; John Doe", na("Alice; Bob", "John Doe"))
	isEqual(t, "Alice; Bob", na("Alice; Bob", "john.doe@example.com"))
	isEqual(t, "Étienne", na("", "\u00e9tienne"))
	isEqual(t, "Étienne", na("", "&#233;tienne"))
	isEqual(t, "Alice; Bob", na("", "Alice &amp; Bob"))
	isEqual(t, "John Doe", na("", "<b>John Doe</b>"))
	isEqual(t, "John Doe", na("", "John 😊 Doe"))
	isEqual(t, "John Doe", na("", "words by John Doe"))
	isEqual(t, "John Doe", na("", "John Doe123"))
	isEqual(t, "John Doe", na("", "John_Doe"))
	isEqual(t, "John Doe", na("", "John Doe* "))
	isEqual(t, "John Doe", na("", "John Doe of John Doe"))
	isEqual(t, "John Doe", na("", "John Doe — John Doe"))
	isEqual(t, "John Doe", na("", `John "The King" Doe`))
}

func Test_Metadata_Authors(t *testing.T) {
	var opts Options
	var rawHTML string
	var metadata Metadata

	isEqual := func(rawHTML string, expected string) {
		metadata := testGetMetadataFromHTML(rawHTML)
		assert.Equal(t, expected, metadata.Author)
	}

	headHTML := func(s string) string {
		return `<html><head>` + s + `</head><body></body></html>`
	}

	bodyHTML := func(s string) string {
		return `<html><body>` + s + `</body></html>`
	}

	// Extraction from head
	rawHTML = headHTML(`<meta itemprop="author" content="Jenny Smith"/>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = headHTML(`<meta itemprop="author" content="Jenny Smith"/><meta itemprop="author" content="John Smith"/>`)
	isEqual(rawHTML, "Jenny Smith; John Smith")

	rawHTML = headHTML(`<meta itemprop="author" content="Jenny Smith und John Smith"/>`)
	isEqual(rawHTML, "Jenny Smith; John Smith")

	rawHTML = headHTML(`<meta name="author" content="Jenny Smith"/><meta name="author" content="John Smith"/>`)
	isEqual(rawHTML, "Jenny Smith; John Smith")

	rawHTML = headHTML(`<meta name="author" content="Jenny Smith and John Smith"/>`)
	isEqual(rawHTML, "Jenny Smith; John Smith")

	rawHTML = headHTML(`<meta name="author" content="Jenny Smith"/>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = headHTML(`<meta name="author" content="Hank O&#39;Hop"/>`)
	isEqual(rawHTML, "Hank O'Hop")

	rawHTML = headHTML(`<meta name="author" content="Jenny Smith ❤️"/>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = headHTML(`<meta name="citation_author" content="Jenny Smith and John Smith"/>`)
	isEqual(rawHTML, "Jenny Smith; John Smith")

	rawHTML = headHTML(`<meta property="author" content="Jenny Smith"/><meta property="author" content="John Smith"/>`)
	isEqual(rawHTML, "Jenny Smith; John Smith")

	rawHTML = headHTML(`<meta itemprop="author" content="Jenny Smith and John Smith"/>`)
	isEqual(rawHTML, "Jenny Smith; John Smith")

	rawHTML = headHTML(`<meta name="article:author" content="Jenny Smith"/>`)
	isEqual(rawHTML, "Jenny Smith")

	// Extraction from body
	rawHTML = bodyHTML(`<a href="" rel="author">Jenny Smith</a>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<a href="" rel="author">Jenny "The Author" Smith</a>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<span class="author">Jenny Smith</span>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<h4 class="author">Jenny Smith</h4>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<h4 class="author">Jenny Smith — Trafilatura</h4>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<span class="wrapper--detail__writer">Jenny Smith</span>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<span id="author-name">Jenny Smith</span>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<figure data-component="Figure"><div class="author">Jenny Smith</div></figure>`)
	isEqual(rawHTML, "")

	rawHTML = bodyHTML(`<div class="sidebar"><div class="author">Jenny Smith</div></figure>`)
	isEqual(rawHTML, "")

	rawHTML = bodyHTML(`<div class="quote"><p>My quote here</p><p class="quote-author"><span>—</span> Jenny Smith</p></div>`)
	isEqual(rawHTML, "")

	rawHTML = bodyHTML(`<span class="author">Jenny Smith and John Smith</span>`)
	isEqual(rawHTML, "Jenny Smith; John Smith")

	rawHTML = bodyHTML(`<a class="author">Jenny Smith</a>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<a class="author">Jenny Smith <div class="title">Editor</div></a>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<a class="author">Jenny Smith from Trafilatura</a>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<meta itemprop="author" content="Fake Author"/><a class="author">Jenny Smith from Trafilatura</a>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<a class="username">Jenny Smith</a>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<div class="submitted-by"><a>Jenny Smith</a></div>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<div class="byline-content"><div class="byline"><a>Jenny Smith</a></div><time>July 12, 2021 08:05</time></div>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<h3 itemprop="author">Jenny Smith</h3>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<div class="article-meta article-meta-byline article-meta-with-photo article-meta-author-and-reviewer" itemprop="author" itemscope="" itemtype="http://schema.org/Person"><span class="article-meta-photo-wrap"><img src="" alt="Jenny Smith" itemprop="image" class="article-meta-photo"></span><span class="article-meta-contents"><span class="article-meta-author">By <a href="" itemprop="url"><span itemprop="name">Jenny Smith</span></a></span><span class="article-meta-date">May 18 2022</span><span class="article-meta-reviewer">Reviewed by <a href="">Robert Smith</a></span></span></div>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<div data-component="Byline">Jenny Smith</div>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<span id="author">Jenny Smith</span>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<span id="author">Jenny Smith – The Moon</span>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<span id="author">Jenny_Smith</span>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<span itemprop="author name">Shannon Deery, Mitch Clarke, Susie O’Brien, Laura Placella, Kara Irving, Jordy Atkinson, Suzan Delibasic</span>`)
	isEqual(rawHTML, "Shannon Deery; Mitch Clarke; Susie O’Brien; Laura Placella; Kara Irving; Jordy Atkinson; Suzan Delibasic")

	rawHTML = bodyHTML(`<address class="author">Jenny Smith</address>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<author>Jenny Smith</author>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<div class="author"><span class="profile__name"> Jenny Smith </span> <a href="https://twitter.com/jenny_smith" class="profile__social" target="_blank"> @jenny_smith </a> <span class="profile__extra lg:hidden"> 11:57AM </span> </div>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<p class="author-section byline-plain">By <a class="author" rel="nofollow">Jenny Smith For Daily Mail Australia</a></p>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<div class="o-Attribution__a-Author"><span class="o-Attribution__a-Author--Label">By:</span><span class="o-Attribution__a-Author--Prefix"><span class="o-Attribution__a-Name"><a href="//web.archive.org/web/20210707074846/https://www.discovery.com/profiles/ian-shive">Ian Shive</a></span></span></div>`)
	isEqual(rawHTML, "Ian Shive")

	rawHTML = bodyHTML(`<div class="ArticlePage-authors"><div class="ArticlePage-authorName" itemprop="name"><span class="ArticlePage-authorBy">By&nbsp;</span><a aria-label="Ben Coxworth" href="https://newatlas.com/author/ben-coxworth/"><span>Ben Coxworth</span></a></div></div>`)
	isEqual(rawHTML, "Ben Coxworth")

	rawHTML = bodyHTML(`<div><strong><a class="d1dba0c3091a3c30ebd6" data-testid="AuthorURL" href="/by/p535y1">AUTHOR NAME</a></strong></div`)
	isEqual(rawHTML, "AUTHOR NAME")

	rawHTML = `<html><head><meta data-rh="true" property="og:author" content="By &lt;a href=&quot;/profiles/amir-vera&quot;&gt;Amir Vera&lt;/a&gt;, Seán Federico O&#x27;Murchú, &lt;a href=&quot;/profiles/tara-subramaniam&quot;&gt;Tara Subramaniam&lt;/a&gt; and Adam Renton, CNN"/></head><body>f{end}`
	isEqual(rawHTML, "Amir Vera; Seán Federico O'Murchú; Tara Subramaniam; Adam Renton; CNN")

	// Blacklist
	opts = Options{BlacklistedAuthors: []string{"Jenny Smith"}}
	rawHTML = `<html><head><meta itemprop="author" content="Jenny Smith"/></head><body></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML, opts)
	assert.Equal(t, "", metadata.Author)

	opts = Options{BlacklistedAuthors: []string{"A", "b"}}
	assert.Equal(t, "c; d", removeBlacklistedAuthors("a; B; c; d", opts))
	assert.Equal(t, "c; d", removeBlacklistedAuthors("a;B;c;d", opts))
}

func Test_Metadata_URLs(t *testing.T) {
	var rawHTML string
	expected := "https://example.org"
	isEqual := func(rawHTML string, expected string, customOpts ...Options) {
		metadata := testGetMetadataFromHTML(rawHTML, customOpts...)
		assert.Equal(t, expected, metadata.URL)
	}

	rawHTML = `<html><head><meta property="og:url" content="https://example.org"/></head><body></body></html>`
	isEqual(rawHTML, expected)

	rawHTML = `<html><head><link rel="canonical" href="https://example.org"/></head><body></body></html>`
	isEqual(rawHTML, expected)

	rawHTML = `<html><head><meta name="twitter:url" content="https://example.org"/></head><body></body></html>`
	isEqual(rawHTML, expected)

	rawHTML = `<html><head><link rel="alternate" hreflang="x-default" href="https://example.org"/></head><body></body></html>`
	isEqual(rawHTML, expected)

	// Test on partial URLs
	rawHTML = `<html><head><link rel="canonical" href="/article/medical-record"/><meta name="twitter:url" content="https://example.org"/></head><body></body></html>`
	assert.Equal(t, "https://example.org/article/medical-record", extractDomURL(docFromStr(rawHTML)))

	rawHTML = `<html><head><base href="https://example.org" target="_blank"/></head><body></body></html>`
	isEqual(rawHTML, expected)
}

func Test_Metadata_Descriptions(t *testing.T) {
	rawHTML := `<html><head><meta itemprop="description" content="Description"/></head><body></body></html>`
	metadata := testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Description", metadata.Description)

	rawHTML = `<html><head><meta property="og:description" content="&amp;#13; A Northern Territory action plan, which includes plans to support development and employment on Aboriginal land, has received an update. &amp;#13..." /></head><body></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "A Northern Territory action plan, which includes plans to support development and employment on Aboriginal land, has received an update. ...", metadata.Description)
}

func Test_Metadata_Dates(t *testing.T) {
	var rawHTML string
	isEqual := func(rawHTML string, expected string, customOpts ...Options) {
		metadata := testGetMetadataFromHTML(rawHTML, customOpts...)
		assert.Equal(t, expected, metadata.Date.Format("2006-01-02"))
	}

	rawHTML = `<html><head><meta property="og:published_time" content="2017-09-01"/></head><body></body></html>`
	isEqual(rawHTML, "2017-09-01")

	rawHTML = `<html><head><meta property="og:url" content="https://example.org/2017/09/01/content.html"/></head><body></body></html>`
	isEqual(rawHTML, "2017-09-01")

	// Compare extensive mode
	opts := defaultOpts
	rawHTML = `<html><body><p>Veröffentlicht am 1.9.17</p></body></html>`

	opts.EnableFallback = false // fast mode
	isEqual(rawHTML, "2017-09-01", opts)

	opts.EnableFallback = true // extensive mode
	isEqual(rawHTML, "2017-09-01", opts)

}

func Test_Metadata_Categories(t *testing.T) {
	var rawHTML string
	isEqual := func(rawHTML string, expected ...string) {
		metadata := testGetMetadataFromHTML(rawHTML)
		assert.Equal(t, expected, metadata.Categories)
	}

	rawHTML = `<html><body>
		<p class="entry-categories">
			<a href="https://example.org/category/cat1/">Cat1</a>,
			<a href="https://example.org/category/cat2/">Cat2</a>
		</p></body></html>`
	isEqual(rawHTML, "Cat1", "Cat2")

	rawHTML = `<html><body>
		<div class="postmeta"><a href="https://example.org/category/cat1/">Cat1</a></div>
	</body></html>`
	isEqual(rawHTML, "Cat1")
}

func Test_Metadata_Tags(t *testing.T) {
	var rawHTML string
	isEqual := func(rawHTML string, expected ...string) {
		metadata := testGetMetadataFromHTML(rawHTML)
		assert.Equal(t, expected, metadata.Tags)
	}

	rawHTML = `<html><body>
		<p class="entry-tags">
			<a href="https://example.org/tags/tag1/">Tag1</a>,
			<a href="https://example.org/tags/tag2/">Tag2</a>
		</p></body></html>`
	isEqual(rawHTML, "Tag1", "Tag2")

	rawHTML = `<html><body>
		<p class="entry-tags">
			<a href="https://example.org/tags/tag1/">    Tag1   </a>,
			<a href="https://example.org/tags/tag2/"> 1 &amp; 2 </a>
		</p></body></html>`
	isEqual(rawHTML, "Tag1", "1 & 2")

	rawHTML = `<html><head>
		<meta name="keywords" content="sodium, salt, paracetamol, blood, pressure, high, heart, &amp;quot, intake, warning, study, &amp;quot, medicine, dissolvable, cardiovascular" />
	</head></html>`
	isEqual(rawHTML, "sodium", "salt", "paracetamol", "blood", "pressure", "high", "heart", "intake", "warning", "study", "medicine", "dissolvable", "cardiovascular")
}

func Test_Metadata_Sitename(t *testing.T) {
	var rawHTML string
	isEqual := func(rawHTML string, expected string) {
		metadata := testGetMetadataFromHTML(rawHTML)
		assert.Equal(t, expected, metadata.Sitename)
	}

	rawHTML = `<html><head><meta name="article:publisher" content="@"/></head><body/></html>`
	isEqual(rawHTML, "")

	rawHTML = `<html><head><meta name="article:publisher" content="The Newspaper"/></head><body/></html>`
	isEqual(rawHTML, "The Newspaper")

	rawHTML = `<html><head><meta property="article:publisher" content="The Newspaper"/></head><body/></html>`
	isEqual(rawHTML, "The Newspaper")

	rawHTML = `<html><head><title>sitemaps.org - Home</title></head><body/></html>`
	isEqual(rawHTML, "sitemaps.org")
}

func Test_Metadata_License(t *testing.T) {
	// From <a> rel
	rawHTML := `<html><body><p><a href="https://creativecommons.org/licenses/by-sa/4.0/" rel="license">CC BY-SA</a></p></body></html>`
	metadata := testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "CC BY-SA 4.0", metadata.License)

	rawHTML = `<html><body><p><a href="https://licenses.org/unknown" rel="license">Unknown</a></p></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Unknown", metadata.License)

	// Footer
	rawHTML = `<html><body><footer><a href="https://creativecommons.org/licenses/by-sa/4.0/">CC BY-SA</a></footer></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "CC BY-SA 4.0", metadata.License)

	// Real world footer test: netzpolitik.org
	rawHTML = `<html><body>
	<div class="footer__navigation">
		<p class="footer__licence">
			<strong>Lizenz: </strong>
			Die von uns verfassten Inhalte stehen, soweit nicht anders vermerkt, unter der Lizenz
			<a href="http://creativecommons.org/licenses/by-nc-sa/4.0/">Creative Commons BY-NC-SA 4.0.</a>
		</p>
	</div></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "CC BY-NC-SA 4.0", metadata.License)

	// This is not a license
	rawHTML = `<html><body><footer class="entry-footer">
		<span class="cat-links">Posted in <a href="https://sallysbakingaddiction.com/category/seasonal/birthday/" rel="category tag">Birthday</a></span>
	</footer></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Empty(t, metadata.License)

	// This is a license
	rawHTML = `<html><body><footer class="entry-footer">
		<span>The license is <a href="https://example.org/1">CC BY-NC</a></span>
	</footer></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "CC BY-NC", metadata.License)
}

func Test_Metadata_MetaImages(t *testing.T) {
	var rawHTML string
	exampleURL, _ := url.ParseRequestURI("http://example.org")
	isEqual := func(rawHTML string, expected string) {
		metadata := testGetMetadataFromHTML(rawHTML, Options{
			Config:      DefaultConfig(),
			OriginalURL: exampleURL,
		})
		assert.Equal(t, expected, metadata.Image)
	}

	// Image extraction from meta SEO tags
	rawHTML = `<html><head><meta property="image" content="https://example.org/example.jpg"></html>`
	isEqual(rawHTML, "https://example.org/example.jpg")

	rawHTML = `<html><head><meta property="og:image:url" content="example.jpg"></html>`
	isEqual(rawHTML, "http://example.org/example.jpg")

	rawHTML = `<html><head><meta property="og:image" content="https://example.org/example-opengraph.jpg" /><body/></html>`
	isEqual(rawHTML, "https://example.org/example-opengraph.jpg")

	rawHTML = `<html><head><meta property="twitter:image" content="https://example.org/example-twitter.jpg"></html>`
	isEqual(rawHTML, "https://example.org/example-twitter.jpg")

	rawHTML = `<html><head><meta property="twitter:image:src" content="example-twitter.jpg"></html>`
	isEqual(rawHTML, "http://example.org/example-twitter.jpg")

	// Without image
	rawHTML = `<html><head><meta name="robots" content="index, follow, max-image-preview:large, max-snippet:-1, max-video-preview:-1" /></html>`
	isEqual(rawHTML, "")
}

func Test_Metadata_MetaTags(t *testing.T) {
	rawHTML := `<html>
		<head>
			<meta property="og:title" content="Open Graph Title" />
			<meta property="og:author" content="Jenny Smith" />
			<meta property="og:description" content="This is an Open Graph description" />
			<meta property="og:site_name" content="My first site" />
			<meta property="og:url" content="https://example.org/test" />
			<meta property="og:type" content="Open Graph Type" />
		</head>
		<body><a rel="license" href="https://creativecommons.org/">Creative Commons</a></body>
	</html>`
	metadata := testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Open Graph Title", metadata.Title)
	assert.Equal(t, "Jenny Smith", metadata.Author)
	assert.Equal(t, "This is an Open Graph description", metadata.Description)
	assert.Equal(t, "My first site", metadata.Sitename)
	assert.Equal(t, "https://example.org/test", metadata.URL)
	assert.Equal(t, "Creative Commons", metadata.License)
	assert.Equal(t, "Open Graph Type", metadata.PageType)

	rawHTML = `<html><head>
			<meta name="dc.title" content="Open Graph Title" />
			<meta name="dc.creator" content="Jenny Smith" />
			<meta name="dc.description" content="This is an Open Graph description" />
		</head></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Open Graph Title", metadata.Title)
	assert.Equal(t, "Jenny Smith", metadata.Author)
	assert.Equal(t, "This is an Open Graph description", metadata.Description)

	rawHTML = `<html><head>
			<meta itemprop="headline" content="Title" />
		</head></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Title", metadata.Title)

	// Test error
	isEmpty := func(meta Metadata) bool {
		return meta.Title == "" &&
			meta.Author == "" &&
			meta.URL == "" &&
			meta.Hostname == "" &&
			meta.Description == "" &&
			meta.Sitename == "" &&
			meta.Date.IsZero() &&
			len(meta.Categories) == 0 &&
			len(meta.Tags) == 0
	}

	metadata = testGetMetadataFromHTML("")
	assert.True(t, isEmpty(metadata))

	metadata = testGetMetadataFromHTML("<html><title></title></html>")
	assert.True(t, isEmpty(metadata))
}

func testGetMetadataFromHTML(rawHTML string, customOpts ...Options) Metadata {
	// Parse raw html
	doc, err := html.Parse(strings.NewReader(rawHTML))
	if err != nil {
		panic(err)
	}

	if len(customOpts) > 0 {
		return extractMetadata(doc, customOpts[0])
	}

	return extractMetadata(doc, defaultOpts)
}

func testGetMetadataFromURL(url string, customOpts ...Options) Metadata {
	doc := parseMockFile(metadataMockFiles, url)
	if len(customOpts) > 0 {
		return extractMetadata(doc, customOpts[0])
	}
	return extractMetadata(doc, defaultOpts)
}

func testGetMetadataFromFile(path string) Metadata {
	// Open file
	path = filepath.Join("test-files", path)
	f, err := os.Open(path)
	if err != nil {
		log.Panic().Err(err)
	}

	// Parse HTML
	doc, err := html.Parse(f)
	if err != nil {
		log.Panic().Err(err)
	}

	return extractMetadata(doc, defaultOpts)
}
