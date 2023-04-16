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
	nurl "net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/html"
)

func Test_Metadata_Titles(t *testing.T) {
	rawHTML := `<html><head><title>Test Title</title></head><body></body></html>`
	metadata := testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Test Title", metadata.Title)

	rawHTML = `<html><body><h1>First</h1><h1>Second</h1></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "First", metadata.Title)

	rawHTML = `<html><body><h1>   </h1><div class="post-title">Test Title</div></body></html>`
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
}

func Test_Metadata_URLs(t *testing.T) {
	rawHTML := `<html><head><meta property="og:url" content="https://example.org"/></head><body></body></html>`
	metadata := testGetMetadataFromHTML(rawHTML)
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

	opts.NoFallback = true // fast mode
	metadata = testGetMetadataFromHTML(rawHTML, opts)
	assert.True(t, metadata.Date.IsZero())

	opts.NoFallback = false // extensive mode
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

func Test_Metadata_MetaTags(t *testing.T) {
	rawHTML := `<html><head>
			<meta property="og:title" content="Open Graph Title" />
			<meta property="og:author" content="Jenny Smith" />
			<meta property="og:description" content="This is an Open Graph description" />
			<meta property="og:site_name" content="My first site" />
			<meta property="og:url" content="https://example.org/test" />
		</head><body>
			<a rel="license" href="https://creativecommons.org/">Creative Commons</a>
		</body></html>`
	metadata := testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Open Graph Title", metadata.Title)
	assert.Equal(t, "Jenny Smith", metadata.Author)
	assert.Equal(t, "This is an Open Graph description", metadata.Description)
	assert.Equal(t, "My first site", metadata.Sitename)
	assert.Equal(t, "https://example.org/test", metadata.URL)
	assert.Equal(t, "Creative Commons", metadata.License)

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

func Test_Metadata_RealPages(t *testing.T) {
	var url string
	var opts Options
	var parsedURL *nurl.URL
	var metadata Metadata

	url = "http://blog.python.org/2016/12/python-360-is-now-available.html"
	metadata = testGetMetadataFromURL(url)
	assert.Equal(t, "Python 3.6.0 is now available!", metadata.Title)
	assert.Equal(t, "Python 3.6.0 is now available! Python 3.6.0 is the newest major release of the Python language, and it contains many new features and opti...", metadata.Description)
	assert.Equal(t, "Ned Deily", metadata.Author)
	assert.Equal(t, url, metadata.URL)
	assert.Equal(t, "blog.python.org", metadata.Sitename)

	url = "https://en.blog.wordpress.com/2019/06/19/want-to-see-a-more-diverse-wordpress-contributor-community-so-do-we/"
	metadata = testGetMetadataFromURL(url)
	assert.Equal(t, "Want to See a More Diverse WordPress Contributor Community? So Do We.", metadata.Title)
	assert.Equal(t, "More diverse speakers at WordCamps means a more diverse community contributing to WordPress — and that results in better software for everyone.", metadata.Description)
	assert.Equal(t, "The WordPress.com Blog", metadata.Sitename)
	assert.Equal(t, url, metadata.URL)

	url = "https://creativecommons.org/about/"
	metadata = testGetMetadataFromURL(url)
	assert.Equal(t, "What we do - Creative Commons", metadata.Title)
	assert.Equal(t, "What is Creative Commons? Creative Commons helps you legally share your knowledge and creativity to build a more equitable, accessible, and innovative world. We unlock the full potential of the internet to drive a new era of development, growth and productivity. With a network of staff, board, and affiliates around the world, Creative Commons provides … Read More \"What we do\"", metadata.Description)
	assert.Equal(t, "Creative Commons", metadata.Sitename)
	assert.Equal(t, url, metadata.URL)

	url = "https://www.creativecommons.at/faircoin-hackathon"
	metadata = testGetMetadataFromURL(url)
	assert.Equal(t, "FairCoin hackathon beim Sommercamp", metadata.Title)

	url = "https://netzpolitik.org/2016/die-cider-connection-abmahnungen-gegen-nutzer-von-creative-commons-bildern/"
	metadata = testGetMetadataFromURL(url)
	assert.Equal(t, "Die Cider Connection: Abmahnungen gegen Nutzer von Creative-Commons-Bildern", metadata.Title)
	assert.Equal(t, "Markus Reuter", metadata.Author)
	assert.Equal(t, "Seit Dezember 2015 verschickt eine Cider Connection zahlreiche Abmahnungen wegen fehlerhafter Creative-Commons-Referenzierungen. Wir haben recherchiert und legen jetzt das Netzwerk der Abmahner offen.", metadata.Description)
	assert.Equal(t, "netzpolitik.org", metadata.Sitename)
	assert.Equal(t, url, metadata.URL)

	url = "https://www.befifty.de/home/2017/7/12/unter-uns-montauk"
	metadata = testGetMetadataFromURL(url)
	assert.Equal(t, "Das vielleicht schönste Ende der Welt: Montauk", metadata.Title)
	assert.Equal(t, "Beate Finken", metadata.Author)
	assert.Equal(t, "Ein Strand, ist ein Strand, ist ein Strand Ein Strand, ist ein Strand, ist ein Strand. Von wegen! In Italien ist alles wohl organisiert, Handtuch an Handtuch oder Liegestuhl an Liegestuhl. In der Karibik liegt man unter Palmen im Sand und in Marbella dominieren Beton und eine kerzengerade Promenade", metadata.Description)
	assert.Equal(t, "BeFifty", metadata.Sitename)
	assert.Equal(t, []string{"Travel", "Amerika"}, metadata.Categories)
	assert.Equal(t, url, metadata.URL)

	url = "https://www.soundofscience.fr/1927"
	metadata = testGetMetadataFromURL(url)
	assert.Equal(t, "Une candidature collective à la présidence du HCERES", metadata.Title)
	assert.Equal(t, "Martin Clavey", metadata.Author)
	assert.True(t, strings.HasPrefix(metadata.Description, "En réaction à la candidature du conseiller recherche"))
	assert.Equal(t, "The Sound Of Science", metadata.Sitename)
	assert.Equal(t, []string{"Politique scientifique française"}, metadata.Categories)
	assert.Equal(t, []string{"évaluation", "HCERES"}, metadata.Tags)
	assert.Equal(t, url, metadata.URL)

	url = "https://laviedesidees.fr/L-evaluation-et-les-listes-de.html"
	metadata = testGetMetadataFromURL(url)
	assert.Equal(t, "L’évaluation et les listes de revues", metadata.Title)
	assert.Equal(t, "Florence Audier", metadata.Author)
	assert.True(t, strings.HasPrefix(metadata.Description, "L'évaluation, et la place"))
	assert.Equal(t, "La Vie des idées", metadata.Sitename)
	// assert.Equal(t, []string{"Essai", "Économie"}, metadata.Categories)
	assert.Empty(t, metadata.Tags)
	assert.Equal(t, "http://www.laviedesidees.fr/L-evaluation-et-les-listes-de.html", metadata.URL)

	url = "https://www.theguardian.com/education/2020/jan/20/thousands-of-uk-academics-treated-as-second-class-citizens"
	metadata = testGetMetadataFromURL(url)
	assert.Equal(t, "Thousands of UK academics 'treated as second-class citizens'", metadata.Title)
	assert.Equal(t, "Richard Adams", metadata.Author)
	assert.True(t, strings.HasPrefix(metadata.Description, "Report claims higher education institutions"))
	assert.Equal(t, "The Guardian", metadata.Sitename)
	assert.Equal(t, []string{"Education"}, metadata.Categories)
	assert.Contains(t, metadata.Tags, "Higher education")
	// meta name="keywords"
	assert.Equal(t, "http://www.theguardian.com/education/2020/jan/20/thousands-of-uk-academics-treated-as-second-class-citizens", metadata.URL)

	url = "https://phys.org/news/2019-10-flint-flake-tool-partially-birch.html"
	metadata = testGetMetadataFromURL(url)
	assert.Equal(t, "Flint flake tool partially covered by birch tar adds to evidence of Neanderthal complex thinking", metadata.Title)
	assert.Equal(t, "Bob Yirka", metadata.Author)
	assert.Equal(t, "A team of researchers affiliated with several institutions in The Netherlands has found evidence in small a cutting tool of Neanderthals using birch tar. In their paper published in Proceedings of the National Academy of Sciences, the group describes the tool and what it revealed about Neanderthal technology.", metadata.Description)
	assert.Equal(t, "Phys.org", metadata.Sitename)
	// assert.Equal(t, []string{"Archeology", "Fossils"}, metadata.Categories)
	assert.Equal(t, []string{"Science", "Physics News", "Science news", "Technology News",
		"Physics", "Materials", "Nanotech", "Technology"}, metadata.Tags)
	assert.Equal(t, url, metadata.URL)

	url = "https://gregoryszorc.com/blog/2020/01/13/mercurial%27s-journey-to-and-reflections-on-python-3/"
	metadata = testGetMetadataFromURL(url)
	assert.Equal(t, "Mercurial's Journey to and Reflections on Python 3", metadata.Title)
	// assert metadata['title'] == "Mercurial's Journey to and Reflections on Python 3"
	// assert.Equal(t, "Gregory Szorc", metadata.Author)
	// assert.Equal(t, "Description of the experience of making Mercurial work with Python 3", metadata.Description)
	// assert.Equal(t, "gregoryszorc", metadata.Sitename)
	// assert metadata['categories'] == ['Mercurial', 'Python']

	url = "https://www.pluralsight.com/tech-blog/managing-python-environments/"
	metadata = testGetMetadataFromURL(url)
	assert.Equal(t, "Managing Python Environments", metadata.Title)
	assert.Equal(t, "John Walk", metadata.Author)
	assert.True(t, strings.HasPrefix(metadata.Description, "If you're not careful,"))
	assert.Equal(t, "pluralsight.com", metadata.Sitename) // Pluralsight
	// assert.Equal(t, []string{"practices"}, metadata.Categories)
	// assert.Equal(t, []string{"python", "docker", " getting started"}, metadata.Tags)
	assert.Equal(t, url, metadata.URL)

	url = "https://stackoverflow.blog/2020/01/20/what-is-rust-and-why-is-it-so-popular/"
	metadata = testGetMetadataFromURL(url)
	assert.Equal(t, "What is Rust and why is it so popular? - Stack Overflow Blog", metadata.Title)
	assert.Equal(t, "Jake Goulding", metadata.Author)
	assert.Equal(t, "Stack Overflow Blog", metadata.Sitename)
	assert.Equal(t, []string{"Bulletin"}, metadata.Categories)
	assert.Equal(t, []string{"programming", "rust"}, metadata.Tags)
	assert.Equal(t, url, metadata.URL)

	url = "https://www.dw.com/en/berlin-confronts-germanys-colonial-past-with-new-initiative/a-52060881"
	metadata = testGetMetadataFromURL(url)
	assert.True(t, strings.Contains(metadata.Title, "Berlin confronts Germany's colonial past with new initiative"))
	assert.Equal(t, "Deutsche Welle", metadata.Author) // "actually 'Ben Knight'
	assert.Equal(t, "The German capital has launched a five-year project to mark its part in European colonialism. Streets which still honor leaders who led the Reich's imperial expansion will be renamed — and some locals aren't happy.", metadata.Description)
	assert.Equal(t, "DW.COM", metadata.Sitename) // DW - Deutsche Welle
	assert.Contains(t, metadata.Tags, "Africa")
	assert.Equal(t, url, metadata.URL)

	url = "https://www.theplanetarypress.com/2020/01/management-of-intact-forestlands-by-indigenous-peoples-key-to-protecting-climate/"
	metadata = testGetMetadataFromURL(url)
	assert.True(t, strings.HasPrefix(metadata.Title, "Management of Intact Forestlands by Indigenous Peoples Key to Protecting Climate"))
	assert.Equal(t, "The Planetary Press", metadata.Author) // actually "Julie Mollins"
	assert.Equal(t, "The Planetary Press", metadata.Sitename)
	assert.Contains(t, metadata.Categories, "Climate")
	assert.Equal(t, url, metadata.URL)

	url = "https://wikimediafoundation.org/news/2020/01/15/access-to-wikipedia-restored-in-turkey-after-more-than-two-and-a-half-years/"
	metadata = testGetMetadataFromURL(url)
	assert.Equal(t, "Access to Wikipedia restored in Turkey after more than two and a half years", metadata.Title)
	assert.Equal(t, "Wikimedia Foundation", metadata.Author)
	assert.True(t, strings.HasPrefix(metadata.Description, "Today, on Wikipedia’s 19th birthday"))
	assert.Equal(t, "Wikimedia Foundation", metadata.Sitename)
	// assert.Equal(t, []string{"Politics", "Turkey", "Wikipedia"}, metadata.Categories)
	assert.Equal(t, url, metadata.URL)

	url = "https://www.reuters.com/article/us-awards-sag/parasite-scores-upset-at-sag-awards-boosting-oscar-chances-idUSKBN1ZI0EH"
	metadata = testGetMetadataFromURL(url)
	assert.True(t, strings.HasSuffix(metadata.Title, "scores historic upset at SAG awards, boosting Oscar chances"))
	assert.Equal(t, "Jill Serjeant", metadata.Author)
	assert.Equal(t, "2020-01-20", metadata.Date.Format("2006-01-02"))
	// assert.Equal(t, "“Parasite,” the Korean language social satire about the wealth gap in South Korea, was the first film in a foreign language to win the top prize of best cast ensemble in the 26 year-history of the SAG awards.", metadata.Description)
	assert.Contains(t, metadata.Tags, "Film")
	assert.Contains(t, metadata.Tags, "South Korea")
	assert.Equal(t, "https://www.reuters.com/article/us-awards-sag-idUSKBN1ZI0EH", metadata.URL)
	// TODO: I'm not sure where the original got "Media" as categories, so here I'll skip it.
	// assert.Contains(t, metadata.Categories, "Media")
	// TODO: It should be "Reuters", but their OpenGraph tag say otherwise.
	assert.Equal(t, "U.S.", metadata.Sitename)

	url = "https://www.nationalgeographic.co.uk/environment-and-conservation/2020/01/ravenous-wild-goats-ruled-island-over-century-now-its-being"
	metadata = testGetMetadataFromURL(url)
	// assert.Equal(t, "National Geographic", metadata.Author)
	assert.Equal(t, "Michael Hingston", metadata.Author)
	assert.Equal(t, "Ravenous wild goats ruled this island for over a century. Now, it's being reborn.", metadata.Title)
	assert.True(t, strings.HasPrefix(metadata.Description, "The rocky island of Redonda, once stripped of its flora and fauna"))
	assert.Equal(t, "National Geographic", metadata.Sitename)
	assert.Equal(t, []string{"Environment and Conservation"}, metadata.Categories) // "Goats", "Environment", "Redonda"
	assert.Equal(t, url, metadata.URL)

	url = "https://www.nature.com/articles/d41586-019-02790-3"
	metadata = testGetMetadataFromURL(url)
	assert.Equal(t, "Gigantic Chinese telescope opens to astronomers worldwide", metadata.Title)
	assert.Equal(t, "Elizabeth Gibney", metadata.Author)
	assert.Equal(t, "FAST has superior sensitivity to detect cosmic phenomena, including fast radio bursts and pulsars.", metadata.Description)
	assert.Equal(t, "Nature Publishing Group", metadata.Sitename) // Nature
	assert.Contains(t, metadata.Categories, "Exoplanets")         // "Astronomy", "Telescope", "China"
	assert.Equal(t, url, metadata.URL)

	url = "https://www.scmp.com/comment/opinion/article/3046526/taiwanese-president-tsai-ing-wens-political-playbook-should-be"
	metadata = testGetMetadataFromURL(url)
	assert.Equal(t, `Carrie Lam should study Tsai Ing-wen’s playbook`, metadata.Title)
	// author exist in JSON-LD, but it's in botched JSON so it'll be empty
	assert.Equal(t, "Alice Wu", metadata.Author)
	assert.Equal(t, url, metadata.URL)

	url = "https://www.faz.net/aktuell/wirtschaft/nutzerbasierte-abrechnung-musik-stars-fordern-neues-streaming-modell-16604622.html"
	metadata = testGetMetadataFromURL(url)
	assert.Equal(t, "Nutzerbasierte Abrechnung: Musik-Stars fordern neues Streaming-Modell", metadata.Title)
	// author overriden from JSON-LD + double name
	assert.Contains(t, strings.Split(metadata.Author, "; "), "Benjamin Fischer")
	assert.Equal(t, "Frankfurter Allgemeine Zeitung", metadata.Sitename)
	assert.Equal(t, "https://www.faz.net/1.6604622", metadata.URL)

	url = "https://boingboing.net/2013/07/19/hating-millennials-the-preju.html"
	metadata = testGetMetadataFromURL(url)
	assert.Equal(t, "Hating Millennials - the prejudice you're allowed to boast about", metadata.Title)
	assert.Equal(t, "Cory Doctorow", metadata.Author)
	assert.Equal(t, "Boing Boing", metadata.Sitename)
	assert.Equal(t, url, metadata.URL)

	url = "https://www.gofeminin.de/abnehmen/wie-kann-ich-schnell-abnehmen-s1431651.html"
	metadata = testGetMetadataFromURL(url)
	assert.Equal(t, "Wie kann ich schnell abnehmen? Der Schlachtplan zum Wunschgewicht", metadata.Title)
	assert.Equal(t, "Diane Buckstegge", metadata.Author)
	assert.Equal(t, "Gofeminin", metadata.Sitename) // originally "gofeminin"
	assert.Equal(t, url, metadata.URL)

	url = "https://github.blog/2019-03-29-leader-spotlight-erin-spiceland/"
	metadata = testGetMetadataFromURL(url)
	assert.Equal(t, "Leader spotlight: Erin Spiceland", metadata.Title)
	assert.Equal(t, "Jessica Rudder", metadata.Author)
	assert.True(t, strings.HasPrefix(metadata.Description, "We’re spending Women’s History"))
	assert.Equal(t, "The GitHub Blog", metadata.Sitename)
	assert.Equal(t, []string{"Community"}, metadata.Categories)
	assert.Equal(t, url, metadata.URL)

	url = "https://www.spiegel.de/spiegel/print/d-161500790.html"
	metadata = testGetMetadataFromURL(url)
	assert.Equal(t, "Ein Albtraum", metadata.Title)
	// print(metadata)
	// assert.Equal(t, "Clemens Höges", metadata.Author)

	url = "https://www.salon.com/2020/01/10/despite-everything-u-s-emissions-dipped-in-2019_partner/"
	metadata = testGetMetadataFromURL(url)
	assert.Equal(t, "Despite everything, U.S. emissions dipped in 2019", metadata.Title)
	// in JSON-LD
	assert.Equal(t, "Nathanael Johnson", metadata.Author)
	assert.Equal(t, "Salon.com", metadata.Sitename)
	// in header
	assert.Contains(t, metadata.Categories, "Science & Health")
	assert.Contains(t, metadata.Tags, "Gas Industry")
	assert.Contains(t, metadata.Tags, "coal emissions")
	assert.Equal(t, url, metadata.URL)

	url = "https://www.ndr.de/nachrichten/info/16-Coronavirus-Update-Wir-brauchen-Abkuerzungen-bei-der-Impfstoffzulassung,podcastcoronavirus140.html"
	parsedURL, _ = nurl.ParseRequestURI(url)
	opts = Options{OriginalURL: parsedURL}
	metadata = testGetMetadataFromURL(url, opts)
	assert.Equal(t, url, metadata.URL)
	assert.Contains(t, metadata.Author, "Korinna Hennig")
	assert.Contains(t, metadata.Tags, "Ältere Menschen")

	url = "https://www.dailymail.co.uk/news/article-9831365/UKs-daily-Covid-cases-fall-SEVENTH-day-Infections-plummet-50-23-511.html"
	metadata = testGetMetadataFromURL(url)
	assert.Equal(t, url, metadata.URL)
	assert.Equal(t, metadata.Author, "Luke Andrews; James Tapsfield")
	assert.Contains(t, metadata.Tags, "news")
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
		logrus.Panicln(err)
	}

	// Parse HTML
	doc, err := html.Parse(f)
	if err != nil {
		logrus.Panicln(err)
	}

	return extractMetadata(doc, defaultOpts)
}
