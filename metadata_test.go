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
	var metadata Metadata

	rawHTML = `<html><body><h3 class="title">T</h3><h3 id="title"></h3></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Empty(t, metadata.Title)

	rawHTML = `<html><head><title>Test Title</title></head><body></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Test Title", metadata.Title)

	rawHTML = `<html><body><h1>First</h1><h1>Second</h1></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "First", metadata.Title)

	rawHTML = `<html><body><h1>   </h1><div class="post-title">Test Title</div></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Test Title", metadata.Title)

	rawHTML = `<html><body><h2 class="block-title">Main menu</h2><h1 class="article-title">Test Title</h1></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Test Title", metadata.Title)

	rawHTML = `<html><body><h2>First</h2><h1>Second</h1></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Second", metadata.Title)

	rawHTML = `<html><body><h2>First</h2><h2>Second</h2></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "First", metadata.Title)

	rawHTML = `<html><body><title></title></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Empty(t, metadata.Title)

	metadata = testGetMetadataFromFile("simple/metadata-title.html")
	assert.Equal(t, "Semantic satiation", metadata.Title)

	rawHTML = `<html><head><title> - Home</title></head><body/></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "- Home", metadata.Title)

	rawHTML = `<html><head><title>My Title » My Website</title></head><body/></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "My Title", metadata.Title) // TODO: and metadata.sitename == "My Website"
}

func Test_Metadata_Authors(t *testing.T) {
	var opts Options
	var rawHTML string
	var metadata Metadata

	// Normalization
	assert.Equal(t, "Abc", normalizeAuthors("", "abc"))
	assert.Equal(t, "Steve Steve", normalizeAuthors("", "Steve Steve 123"))
	assert.Equal(t, "Steve Steve", normalizeAuthors("", "By Steve Steve"))

	// Extraction
	rawHTML = `<html><head><meta itemprop="author" content="Jenny Smith"/></head><body></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Jenny Smith", metadata.Author)

	rawHTML = `<html><head><meta name="author" content="Jenny Smith"/></head><body></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Jenny Smith", metadata.Author)

	rawHTML = `<html><head><meta name="author" content="Hank O&#39;Hop"/></head><body></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Hank O'Hop", metadata.Author)

	rawHTML = `<html><body><a href="" rel="author">Jenny Smith</a></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Jenny Smith", metadata.Author)

	rawHTML = `<html><body><a href="" rel="author">Jenny "The Author" Smith</a></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Jenny Smith", metadata.Author)

	rawHTML = `<html><body><span class="author">Jenny Smith</span></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Jenny Smith", metadata.Author)

	rawHTML = `<html><body><h4 class="author">Jenny Smith</h4></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Jenny Smith", metadata.Author)

	rawHTML = `<html><body><h4 class="author">Jenny Smith — Trafilatura</h4></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Jenny Smith", metadata.Author)

	rawHTML = `<html><body><span class="wrapper--detail__writer">Jenny Smith</span></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Jenny Smith", metadata.Author)

	rawHTML = `<html><body><span id="author-name">Jenny Smith</span></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Jenny Smith", metadata.Author)

	rawHTML = `<html><body><figure data-component="Figure"><div class="author">Jenny Smith</div></figure></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Empty(t, metadata.Author)

	rawHTML = `<html><body><div class="sidebar"><div class="author">Jenny Smith</div></figure></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Empty(t, metadata.Author)

	rawHTML = `<html><body>
		<div class="quote">
			<p>My quote here</p>
			<p class="quote-author"><span>—</span> Jenny Smith</p>
		</div>
	</body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Empty(t, metadata.Author)

	rawHTML = `<html><body><a class="author">Jenny Smith</a></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Jenny Smith", metadata.Author)

	rawHTML = `<html><body><a class="author">Jenny Smith <div class="title">Editor</div></a></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Jenny Smith", metadata.Author)

	rawHTML = `<html><body><a class="author">Jenny Smith from Trafilatura</a></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Jenny Smith", metadata.Author)

	rawHTML = `<html><body><a class="username">Jenny Smith</a></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Jenny Smith", metadata.Author)

	rawHTML = `<html><body><div class="submitted-by"><a>Jenny Smith</a></div></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Jenny Smith", metadata.Author)

	rawHTML = `<html><body><div class="byline-content"><div class="byline"><a>Jenny Smith</a></div><time>July 12, 2021 08:05</time></div></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Jenny Smith", metadata.Author)

	rawHTML = `<html><body><h3 itemprop="author">Jenny Smith</h3></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Jenny Smith", metadata.Author)

	rawHTML = `<html><body>
		<div class="article-meta article-meta-byline article-meta-with-photo article-meta-author-and-reviewer" itemprop="author" itemscope="" itemtype="http://schema.org/Person">
			<span class="article-meta-photo-wrap">
				<img src="" alt="Jenny Smith" itemprop="image" class="article-meta-photo">
			</span>
			<span class="article-meta-contents">
				<span class="article-meta-author">By <a href="" itemprop="url"><span itemprop="name">Jenny Smith</span></a></span>
				<span class="article-meta-date">May 18 2022</span>
				<span class="article-meta-reviewer">Reviewed by <a href="">Robert Smith</a></span>
			</span>
		</div>
	</body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Jenny Smith", metadata.Author)

	rawHTML = `<html><body><div data-component="Byline">Jenny Smith</div></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Jenny Smith", metadata.Author)

	rawHTML = `<html><body><span id="author">Jenny Smith</span></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Jenny Smith", metadata.Author)

	rawHTML = `<html><body><span id="author">Jenny_Smith</span></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Jenny Smith", metadata.Author)

	rawHTML = `<html><body><address class="author">Jenny Smith</address></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Jenny Smith", metadata.Author)

	rawHTML = `<html><body><author>Jenny Smith</author></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Jenny Smith", metadata.Author)

	rawHTML = `<html><body><div class="author"><span class="profile__name"> Jenny Smith </span> <a href="https://twitter.com/jenny_smith" class="profile__social" target="_blank"> @jenny_smith </a> <span class="profile__extra lg:hidden"> 11:57AM </span> </div></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Jenny Smith", metadata.Author)

	rawHTML = `<html><body><p class="author-section byline-plain">By <a class="author" rel="nofollow">Jenny Smith For Daily Mail Australia</a></p></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Jenny Smith", metadata.Author)

	rawHTML = `<html><body><div class="o-Attribution__a-Author"><span class="o-Attribution__a-Author--Label">By:</span><span class="o-Attribution__a-Author--Prefix"><span class="o-Attribution__a-Name"><a href="//web.archive.org/web/20210707074846/https://www.discovery.com/profiles/ian-shive">Ian Shive</a></span></span></div></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Ian Shive", metadata.Author)

	rawHTML = `<html><body><div class="ArticlePage-authors"><div class="ArticlePage-authorName" itemprop="name"><span class="ArticlePage-authorBy">By&nbsp;</span><a aria-label="Ben Coxworth" href="https://newatlas.com/author/ben-coxworth/"><span>Ben Coxworth</span></a></div></div></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Ben Coxworth", metadata.Author)

	// No emoji
	rawHTML = `<html><head><meta name="author" content="Jenny Smith ❤️"/></head><body></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Jenny Smith", metadata.Author)

	// Multi authors
	rawHTML = `<html><head><meta itemprop="author" content="Jenny Smith"/><meta itemprop="author" content="John Smith"/></head><body></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Jenny Smith; John Smith", metadata.Author)

	rawHTML = `<html><head><meta itemprop="author" content="Jenny Smith und John Smith"/></head><body></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Jenny Smith; John Smith", metadata.Author)

	rawHTML = `<html><head><meta name="author" content="Jenny Smith"/><meta name="author" content="John Smith"/></head><body></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Jenny Smith; John Smith", metadata.Author)

	rawHTML = `<html><head><meta name="author" content="Jenny Smith and John Smith"/></head><body></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Jenny Smith; John Smith", metadata.Author)

	rawHTML = `<html><head><meta property="author" content="Jenny Smith"/><meta property="author" content="John Smith"/></head><body></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Jenny Smith; John Smith", metadata.Author)

	rawHTML = `<html><body><span class="author">Jenny Smith and John Smith</span></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Jenny Smith; John Smith", metadata.Author)

	rawHTML = `<html><body><span itemprop="author name">Shannon Deery, Mitch Clarke, Susie O’Brien, Laura Placella, Kara Irving, Jordy Atkinson, Suzan Delibasic</span></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Shannon Deery; Mitch Clarke; Susie O’Brien; Laura Placella; Kara Irving; Jordy Atkinson; Suzan Delibasic", metadata.Author)

	// Google Scholar citation
	rawHTML = `<html><head><meta name="citation_author" content="Jenny Smith and John Smith"/></head><body></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Jenny Smith; John Smith", metadata.Author)

	// Blacklist
	opts = Options{BlacklistedAuthors: []string{"Fake Author"}}
	rawHTML = `<html><body><meta itemprop="author" content="Fake Author"/><a class="author">Jenny Smith from Trafilatura</a></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML, opts)
	assert.Equal(t, "Jenny Smith", metadata.Author)

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
	var metadata Metadata

	rawHTML = `<html><head><meta property="og:url" content="https://example.org"/></head><body></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "https://example.org", metadata.URL)

	rawHTML = `<html><head><link rel="canonical" href="https://example.org"/></head><body></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "https://example.org", metadata.URL)

	rawHTML = `<html><head><meta name="twitter:url" content="https://example.org"/></head><body></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "https://example.org", metadata.URL)

	rawHTML = `<html><head><link rel="alternate" hreflang="x-default" href="https://example.org"/></head><body></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "https://example.org", metadata.URL)

	rawHTML = `<html><head><link rel="canonical" href="/article/medical-record"/><meta name="twitter:url" content="https://example.org"/></head><body></body></html>`
	assert.Equal(t, "https://example.org/article/medical-record", extractDomURL(docFromStr(rawHTML)))
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
	rawHTML := `<html><head><meta property="og:published_time" content="2017-09-01"/></head><body></body></html>`
	metadata := testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "2017-09-01", metadata.Date.Format("2006-01-02"))

	rawHTML = `<html><head><meta property="og:url" content="https://example.org/2017/09/01/content.html"/></head><body></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "2017-09-01", metadata.Date.Format("2006-01-02"))

	// Compare extensive mode
	opts := defaultOpts
	rawHTML = `<html><body><p>Veröffentlicht am 1.9.17</p></body></html>`

	opts.FallbackCandidates = nil // fast mode
	metadata = testGetMetadataFromHTML(rawHTML, opts)
	assert.True(t, metadata.Date.IsZero())

	opts.FallbackCandidates = &FallbackConfig{} // extensive mode
	metadata = testGetMetadataFromHTML(rawHTML, opts)
	assert.Equal(t, "2017-09-01", metadata.Date.Format("2006-01-02"))

}

func Test_Metadata_Categories(t *testing.T) {
	var rawHTML string
	var metadata Metadata
	var expected []string

	rawHTML = `<html><body>
		<p class="entry-categories">
			<a href="https://example.org/category/cat1/">Cat1</a>,
			<a href="https://example.org/category/cat2/">Cat2</a>
		</p></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	expected = []string{"Cat1", "Cat2"}
	assert.Equal(t, expected, metadata.Categories)

	rawHTML = `<html><body>
		<div class="postmeta"><a href="https://example.org/category/cat1/">Cat1</a></div>
	</body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	expected = []string{"Cat1"}
	assert.Equal(t, expected, metadata.Categories)

	rawHTML = `<html><head>
		<meta name="keywords" content="sodium, salt, paracetamol, blood, pressure, high, heart, &amp;quot, intake, warning, study, &amp;quot, medicine, dissolvable, cardiovascular" />
	</head></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	expected = []string{"sodium", "salt", "paracetamol", "blood", "pressure", "high", "heart", "intake", "warning", "study", "medicine", "dissolvable", "cardiovascular"}
	assert.Equal(t, expected, metadata.Tags)
}

func Test_Metadata_Tags(t *testing.T) {
	rawHTML := `<html><body>
		<p class="entry-tags">
			<a href="https://example.org/tags/tag1/">Tag1</a>,
			<a href="https://example.org/tags/tag2/">Tag2</a>
		</p></body></html>`
	metadata := testGetMetadataFromHTML(rawHTML)
	expected := []string{"Tag1", "Tag2"}
	assert.Equal(t, expected, metadata.Tags)
}

func Test_Metadata_Sitename(t *testing.T) {
	var rawHTML string
	var metadata Metadata

	rawHTML = `<html><head><meta name="article:publisher" content="The Newspaper"/></head><body/></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "The Newspaper", metadata.Sitename)

	rawHTML = `<html><head><meta property="article:publisher" content="The Newspaper"/></head><body/></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "The Newspaper", metadata.Sitename)

	rawHTML = `<html><head><title>sitemaps.org - Home</title></head><body/></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "sitemaps.org", metadata.Sitename)
	assert.Equal(t, "Home", metadata.Title)

	rawHTML = `<html><head><meta name="article:publisher" content="@"/></head><body/></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Empty(t, metadata.Sitename)
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
	var metadata Metadata

	// Image extraction from meta SEO tags
	rawHTML = `<html><head><meta property="image" content="https://example.org/example.jpg"></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "https://example.org/example.jpg", metadata.Image)

	rawHTML = `<html><head><meta property="og:image" content="https://example.org/example-opengraph.jpg" /><body/></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "https://example.org/example-opengraph.jpg", metadata.Image)

	rawHTML = `<html><head><meta property="twitter:image" content="https://example.org/example-twitter.jpg"></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "https://example.org/example-twitter.jpg", metadata.Image)

	// Without image
	rawHTML = `<html><head><meta name="robots" content="index, follow, max-image-preview:large, max-snippet:-1, max-video-preview:-1" /></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Empty(t, metadata.Image)
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
