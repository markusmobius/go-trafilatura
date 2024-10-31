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
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/forPelevin/gomoji"
	"github.com/go-shiori/dom"
	"github.com/markusmobius/go-htmldate"
	"github.com/markusmobius/go-trafilatura/internal/etree"
	"github.com/markusmobius/go-trafilatura/internal/selector"
	"golang.org/x/net/html"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	rxCommaSeparator = regexp.MustCompile(`\s*[,;]\s*`)
	rxTitleCleaner   = regexp.MustCompile(`(?i)^(.+)?\s+[–•·—|⁄*⋆~‹«<›»>:-]\s+(.+)$`) // part without dots?
	rxJsonSymbol     = regexp.MustCompile(`[{\\}]`)
	rxNameJson       = regexp.MustCompile(`(?i)"name\\?":\s*\\?"([^"\\]+)`)
	rxUrlCheck       = regexp.MustCompile(`(?i)https?://`)
	rxSitenameFinder = regexp.MustCompile(`(?i)https?://(?:www\.|w[0-9]+\.)?([^/]+)`)
	rxHtmlStripTag   = regexp.MustCompile(`(?i)(<!--.*?-->|<[^>]*>)`)
	rxCategoryHref   = regexp.MustCompile(`(?i)/categor(?:y|ies)/`)
	rxTagHref        = regexp.MustCompile(`(?i)/tags?/`)

	rxCcLicense     = regexp.MustCompile(`(?i)/(by-nc-nd|by-nc-sa|by-nc|by-nd|by-sa|by|zero)/([1-9]\.[0-9])`)
	rxCcLicenseText = regexp.MustCompile(`(?i)(cc|creative commons) (by-nc-nd|by-nc-sa|by-nc|by-nd|by-sa|by|zero) ?([1-9]\.[0-9])?`)

	rxAuthorPrefix       = regexp.MustCompile(`(?i)^([a-zäöüß]+(ed|t))? ?(written by|words by|words|by|von|from) `)
	rxAuthorDigits       = regexp.MustCompile(`(?i)\pN.+?$`)
	rxAuthorSocialMedia  = regexp.MustCompile(`(?i)@\S+`)
	rxAuthorSpaceChars   = regexp.MustCompile(`(?i)[._+]`)
	rxAuthorNickname     = regexp.MustCompile(`(?i)["‘({\[’\'][^"]+?[‘’"\')\]}]`)
	rxAuthorSpecialChars = regexp.MustCompile(`(?i)[^\pL\pM\pN_]+$|[:()?*$#!%/<>{}~¿]`)
	rxAuthorPreposition  = regexp.MustCompile(`(?i)\b\s+(am|on|for|at|in|to|from|of|via|with|—|-|–)\s+(.*)`)
	rxAuthorEmail        = regexp.MustCompile(`(?i)\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`)
	rxAuthorSeparator    = regexp.MustCompile(`(?i)/|;|,|\||&|(?:^|[^\pL\pM\pN_])[u|a]nd(?:$|[^\pL\pM\pN_])`)
	rxAuthorHTML         = regexp.MustCompile(`(?i)<[^>]+>`)

	metaNameAuthor = sliceToMap(
		"article:author", "atc-metaauthor", "author", "authors", "byl", "citation_author",
		"creator", "dc.creator", "dc.creator.aut", "dc:creator",
		"dcterms.creator", "dcterms.creator.aut", "dcsext.author", "parsely-author",
		"rbauthors", "sailthru.author", "shareaholic:article_author_name") // questionable: twitter:creator
	metaNameTitle = sliceToMap(
		"citation_title", "dc.title", "dcterms.title", "fb_title",
		"headline", "parsely-title", "sailthru.title", "shareaholic:title",
		"rbtitle", "title", "twitter:title")
	metaNameDescription = sliceToMap(
		"dc.description", "dc:description",
		"dcterms.abstract", "dcterms.description",
		"description", "sailthru.description", "twitter:description")
	metaNamePublisher = sliceToMap(
		"article:publisher", "citation_journal_title", "copyright",
		"dc.publisher", "dc:publisher", "dcterms.publisher",
		"publisher", "sailthru.publisher", "rbpubname", "twitter:site") // questionable: citation_publisher
	metaNameTag = sliceToMap(
		"citation_keywords", "dcterms.subject", "keywords", "parsely-tags",
		"shareaholic:keywords", "tags")
	metaNameImage = sliceToMap(
		"image", "og:image", "og:image:url", "og:image:secure_url",
		"twitter:image", "twitter:image:src")
	// metaNameUrl   = sliceToMap("rbmainurl", "twitter:url")

	fastHtmlDateOpts      = &htmldate.Options{UseOriginalDate: true, SkipExtensiveSearch: true}
	extensiveHtmlDateOpts = &htmldate.Options{UseOriginalDate: true, SkipExtensiveSearch: false}
)

// Metadata is the metadata of the page.
type Metadata struct {
	Title       string
	Author      string
	URL         string
	Hostname    string
	Description string
	Sitename    string
	Date        time.Time
	Categories  []string
	Tags        []string
	ID          string
	Fingerprint string
	License     string
	Language    string
	Image       string
	PageType    string
}

func extractMetadata(doc *html.Node, opts Options) Metadata {
	// Extract metadata from <meta> tags
	metadata := examineMeta(doc)
	metadata.Author = removeBlacklistedAuthors(metadata.Author, opts)

	// TODO: in original trafilatura, if author name is a single word,
	// it's regarded as invalid and author name will be set to empty.
	// However, in some Asia country there are case hwere the name is
	// only a single word, so I decided to not implement it here.

	// Extract metadata from JSON-LD and override
	metadata = extractJsonLd(opts, doc, metadata)
	metadata.Author = removeBlacklistedAuthors(metadata.Author, opts)

	// Try extracting from DOM element using selectors
	// Title
	if metadata.Title == "" {
		metadata.Title = extractDomTitle(doc)
	}

	// Author
	if metadata.Author == "" {
		metadata.Author = extractDomAuthor(doc)
		metadata.Author = removeBlacklistedAuthors(metadata.Author, opts)
	}

	// URL
	if metadata.URL == "" {
		metadata.URL = extractDomURL(doc)
	}

	// Validate URL
	// If URL exist, it must be absolute. If not absolute, just remove it.
	if metadata.URL != "" {
		validURL, isAbs := validateURL(metadata.URL, opts.OriginalURL)
		if validURL != "" && isAbs {
			metadata.URL = validURL
		} else {
			metadata.URL = ""
		}
	}

	// If URL not found but original URL specified, just use that.
	if metadata.URL == "" && opts.OriginalURL != nil {
		metadata.URL = opts.OriginalURL.String()
	}

	// Hostname
	if metadata.URL != "" {
		metadata.Hostname = getDomainURL(metadata.URL)
	}

	// Validate image URL, it must be absolute. If not absolute, just remove it.
	if metadata.Image != "" {
		validURL, isAbs := validateURL(metadata.Image, opts.OriginalURL)
		if validURL != "" && isAbs {
			metadata.Image = validURL
		} else {
			metadata.Image = ""
		}
	}

	// Publish date
	if opts.HtmlDateOverride != nil { // User has his own HtmlDate result
		if opts.HtmlDateOverride.HasTime {
			metadata.Date = opts.HtmlDateOverride.DateTime
		}
	} else {
		var optsPointer *htmldate.Options

		switch {
		case opts.HtmlDateOptions != nil: // User has his own HtmlDate options
			optsPointer = opts.HtmlDateOptions

		case opts.HtmlDateMode == Default:
			if opts.EnableFallback {
				optsPointer = extensiveHtmlDateOpts
			} else {
				optsPointer = fastHtmlDateOpts
			}

		case opts.HtmlDateMode == Fast:
			optsPointer = fastHtmlDateOpts

		case opts.HtmlDateMode == Extensive:
			optsPointer = extensiveHtmlDateOpts

		case opts.HtmlDateMode == Disabled:
			optsPointer = nil
		}

		if optsPointer != nil {
			htmlDateOpts := *optsPointer
			htmlDateOpts.URL = metadata.URL
			publishDate, err := htmldate.FromDocument(doc, htmlDateOpts)
			if err == nil && !publishDate.IsZero() {
				metadata.Date = publishDate.DateTime
			}
		}
	}

	// Sitename
	if metadata.Sitename == "" {
		metadata.Sitename = extractDomSitename(doc)
	}

	if metadata.Sitename != "" {
		// Scrap Twitter ID
		metadata.Sitename = strings.TrimPrefix(metadata.Sitename, "@")

		// Capitalize
		firstRune, _ := utf8.DecodeRuneInString(metadata.Sitename)
		if !strings.Contains(metadata.Sitename, ".") && !unicode.IsUpper(firstRune) {
			metadata.Sitename = cases.Title(language.English).String(metadata.Sitename)
		}
	} else if metadata.URL != "" {
		matches := rxSitenameFinder.FindStringSubmatch(metadata.URL)
		if len(matches) > 0 {
			metadata.Sitename = matches[1]
		}
	}

	// Categories
	if len(metadata.Categories) == 0 {
		metadata.Categories = extractDomCategories(doc)
	}

	if len(metadata.Categories) != 0 {
		metadata.Categories = cleanCatTags(metadata.Categories)
	}

	// Tags
	if len(metadata.Tags) == 0 {
		metadata.Tags = extractDomTags(doc)
	}

	if len(metadata.Tags) != 0 {
		metadata.Tags = cleanCatTags(metadata.Tags)
	}

	// License
	metadata.License = extractLicense(doc)

	return metadata
}

// examineMeta search meta tags for relevant information
func examineMeta(doc *html.Node) Metadata {
	// Bootstrap metadata from OpenGraph tags
	metadata := extractOpenGraphMeta(doc)

	// If all OpenGraph metadata have been assigned, we can return it
	if metadata.Title != "" && metadata.Author != "" &&
		metadata.URL != "" && metadata.Description != "" &&
		metadata.Sitename != "" && metadata.Image != "" && metadata.PageType != "" {
		return metadata
	}

	// Scan all <meta> nodes that has attribute "content"
	var tmpSitename string
	for _, node := range dom.QuerySelectorAll(doc, "head meta[content]") {
		// Make sure content is not empty
		content := dom.GetAttribute(node, "content")
		content = rxHtmlStripTag.ReplaceAllString(content, "")
		content = html.UnescapeString(content)
		content = trim(content)
		if content == "" {
			continue
		}

		// Handle property attribute
		property := dom.GetAttribute(node, "property")
		property = trim(property)

		if property != "" {
			switch {
			case strings.HasPrefix(property, "og:"):
				// We already handle OpenGraph before
			case property == "article:tag":
				metadata.Tags = append(metadata.Tags, content)
			case strIn(property, "author", "article:author"):
				metadata.Author = normalizeAuthors(metadata.Author, content)
			case property == "article:publisher":
				metadata.Sitename = strOr(metadata.Sitename, content)
			case inMap(property, metaNameImage):
				metadata.Image = strOr(metadata.Image, content)
			}
			continue
		}

		// Handle name attribute
		name := dom.GetAttribute(node, "name")
		name = strings.ToLower(name)
		name = trim(name)

		if name != "" {
			if inMap(name, metaNameAuthor) {
				content = rxHtmlStripTag.ReplaceAllString(content, "")
				metadata.Author = normalizeAuthors(metadata.Author, content)
			} else if inMap(name, metaNameTitle) {
				metadata.Title = strOr(metadata.Title, content)
			} else if inMap(name, metaNameDescription) {
				metadata.Description = strOr(metadata.Description, content)
			} else if inMap(name, metaNamePublisher) {
				metadata.Sitename = strOr(metadata.Sitename, content)
			} else if strIn(name, "twitter:site", "application-name") || strings.Contains(name, "twitter:app:name") {
				tmpSitename = content
			} else if name == "twitter:url" {
				if isAbs, _ := isAbsoluteURL(content); metadata.URL == "" && isAbs {
					metadata.URL = content
				}
			} else if inMap(name, metaNameTag) { // "page-topic"
				metadata.Tags = append(metadata.Tags, content)
			}
			continue
		}

		// Handle itemprop attribute
		itemprop := dom.GetAttribute(node, "itemprop")
		itemprop = trim(itemprop)

		if itemprop != "" {
			switch itemprop {
			case "author":
				metadata.Author = normalizeAuthors(metadata.Author, content)
			case "description":
				metadata.Description = strOr(metadata.Description, content)
			case "headline":
				metadata.Title = strOr(metadata.Title, content)
			}
			continue
		}
	}

	// Use temporary site name if necessary
	if metadata.Sitename == "" && tmpSitename != "" {
		metadata.Sitename = tmpSitename
	}

	// Clean up author and tags
	metadata.Author = validateMetadataName(metadata.Author)
	metadata.Categories = uniquifyLists(metadata.Categories...)
	metadata.Tags = uniquifyLists(metadata.Tags...)
	return metadata
}

// extractOpenGraphMeta search meta tags following the OpenGraph guidelines (https://ogp.me/)
func extractOpenGraphMeta(doc *html.Node) Metadata {
	var metadata Metadata

	// Scan all <meta> nodes whose property starts with "og:"
	for _, node := range dom.QuerySelectorAll(doc, `meta[property^="og:"]`) {
		// Get property name
		propName := dom.GetAttribute(node, "property")
		propName = trim(propName)

		// Make sure node has content attribute
		content := dom.GetAttribute(node, "content")
		content = html.UnescapeString(content)
		content = trim(content)
		if content == "" {
			continue
		}

		// Fill metadata
		switch propName {
		case "og:site_name":
			metadata.Sitename = content
		case "og:title":
			metadata.Title = content
		case "og:description":
			metadata.Description = content
		case "og:author", "og:article:author":
			metadata.Author = normalizeAuthors("", content)
		case "og:image", "og:image:url", "og:image:secure_url":
			metadata.Image = content
		case "og:url":
			if isAbs, _ := isAbsoluteURL(content); isAbs {
				metadata.URL = content
			}
		case "og:article:tag":
			metadata.Tags = uniquifyLists(content)
		case "og:type":
			metadata.PageType = content
		}
	}

	return metadata
}

func validateMetadataName(name string) string {
	if name == "" {
		return name
	}

	if !strings.Contains(name, " ") || strings.HasPrefix(name, "http") {
		return ""
	}

	// Make sure author doesn't contain JSON symbols (in case JSON+LD has wrong format)
	if rxJsonSymbol.MatchString(name) {
		return ""
	}

	return name
}

func examineTitleElement(doc *html.Node) (title, first, second string) {
	titleNode := dom.QuerySelector(doc, "head > title")
	if titleNode != nil {
		title = trim(dom.TextContent(titleNode))

		if title != "" {
			matches := rxTitleCleaner.FindStringSubmatch(title)
			if len(matches) > 0 {
				first, second = matches[1], matches[2]
			}
		}
	}

	return
}

// extractDomTitle returns the document title from DOM elements.
func extractDomTitle(doc *html.Node) string {
	// If there are only one H1, use it as title
	h1Nodes := dom.QuerySelectorAll(doc, "h1")
	if len(h1Nodes) == 1 {
		title := trim(dom.TextContent(h1Nodes[0]))
		if title != "" {
			return title
		}
	}

	// Look for title using several CSS selectors
	title := extractDomMetaSelectors(doc, 200, selector.MetaTitle)
	if title != "" {
		return title
	}

	// Look in <title> tag
	title, first, second := examineTitleElement(doc)
	if first != "" && !strings.Contains(first, ".") {
		title = first
	} else if second != "" && !strings.Contains(second, ".") {
		title = second
	}

	if title != "" {
		return title
	}

	// If still not found, just use the first H1 as it is
	if len(h1Nodes) > 0 {
		title := dom.TextContent(h1Nodes[0])
		return trim(title)
	}

	// If STILL not found, use the first H2 as it is
	h2Node := dom.QuerySelector(doc, "h2")
	if h2Node != nil {
		title := dom.TextContent(h2Node)
		return trim(title)
	}

	return ""
}

// extractDomAuthor returns the document author from DOM elements.
func extractDomAuthor(doc *html.Node) string {
	clone := dom.Clone(doc, true)
	clone = pruneUnwantedNodes(clone, selector.MetaAuthorDiscard)

	author := extractDomMetaSelectors(clone, 120, selector.MetaAuthor)
	if author != "" {
		return normalizeAuthors("", author)
	}

	return ""
}

// extractDomURL extracts the document URL from the canonical <link>.
func extractDomURL(doc *html.Node) string {
	var url string

	// Try canonical link first
	linkNode := dom.QuerySelector(doc, `link[rel="canonical"][href]`)
	if linkNode == nil {
		// Now try default language link
		linkNode = dom.QuerySelector(doc, `link[rel="alternate"][hreflang="x-default"]`)
	}

	if linkNode != nil {
		if href := trim(dom.GetAttribute(linkNode, "href")); href != "" {
			url = href
		}
	}

	// Add domain name if it's missing
	if url != "" && strings.HasPrefix(url, "/") {
		for _, node := range dom.QuerySelectorAll(doc, "head meta[content]") {
			nodeName := trim(dom.GetAttribute(node, "name"))
			nodeProperty := trim(dom.GetAttribute(node, "property"))

			attrType := strOr(nodeName, nodeProperty)
			if attrType == "" {
				continue
			}

			if strings.HasPrefix(attrType, "og:") || strings.HasPrefix(attrType, "twitter:") {
				nodeContent := trim(dom.GetAttribute(node, "content"))
				if baseURL := getBaseURL(nodeContent); baseURL != "" {
					url = baseURL + url
					break
				}
			}
		}
	}

	// If default URL is not specified, just give up
	return url
}

// extractDomSitename extracts the name of a site from the main title (if it exists).
func extractDomSitename(doc *html.Node) string {
	_, first, second := examineTitleElement(doc)
	if first != "" && strings.Contains(first, ".") {
		return first
	} else if second != "" && strings.Contains(second, ".") {
		return second
	}

	return ""
}

// extractDomCategories returns the categories of the document.
func extractDomCategories(doc *html.Node) []string {
	categories := []string{}

	// Try using selectors
	for _, query := range selector.MetaCategories {
		for _, node := range selector.QueryAll(doc, query) {
			href := trim(dom.GetAttribute(node, "href"))
			if href != "" && rxCategoryHref.MatchString(href) {
				if text := trim(dom.TextContent(node)); text != "" {
					categories = append(categories, text)
				}
			}
		}

		if len(categories) > 0 {
			break
		}
	}

	// Fallback
	if len(categories) == 0 {
		selectors := []string{
			`head meta[property="article:section"]`,
			`head meta[name*="subject"]`}
		mergedSelector := strings.Join(selectors, ", ")

		for _, node := range dom.QuerySelectorAll(doc, mergedSelector) {
			if content := trim(dom.GetAttribute(node, "content")); content != "" {
				categories = append(categories, content)
			}
		}
	}

	return uniquifyLists(categories...)
}

// extractDomTags returns the tags of the document.
func extractDomTags(doc *html.Node) []string {
	tags := []string{}

	// Try using selectors
	for _, query := range selector.MetaTags {
		for _, node := range selector.QueryAll(doc, query) {
			href := trim(dom.GetAttribute(node, "href"))
			if href != "" && rxTagHref.MatchString(href) {
				if text := trim(dom.TextContent(node)); text != "" {
					tags = append(tags, text)
				}
			}
		}

		if len(tags) > 0 {
			break
		}
	}

	return uniquifyLists(tags...)
}

func cleanCatTags(catTags []string) []string {
	cleanedEntries := []string{}
	for _, entry := range catTags {
		for _, item := range rxCommaSeparator.Split(entry, -1) {
			if item = trim(item); item != "" {
				cleanedEntries = append(cleanedEntries, item)
			}
		}
	}
	return cleanedEntries
}

func extractDomMetaSelectors(doc *html.Node, limit int, queries []selector.Rule) string {
	for _, query := range queries {
		for _, node := range selector.QueryAll(doc, query) {
			text := etree.IterText(node, " ")
			text = trim(text)

			lenText := utf8.RuneCountInString(text)
			if lenText > 2 && lenText < limit {
				return text
			}
		}
	}

	return ""
}

// extractLicense search the HTML code for license information and parse it.
func extractLicense(doc *html.Node) string {
	// Look for links labeled as license
	for _, a := range dom.QuerySelectorAll(doc, `a[rel="license"][href]`) {
		if result := parseLicenseElement(a, false); result != "" {
			return result
		}
	}

	// Probe footer elements for CC links
	selector := `footer a[href], div[class*="footer"] a[href], div[id*="footer"] a[href]`
	for _, node := range dom.QuerySelectorAll(doc, selector) {
		if result := parseLicenseElement(node, true); result != "" {
			return result
		}
	}

	return ""
}

// parseLicenseElement probes a link for identifiable free license cues.
// Parse the href attribute first and then the link text.
func parseLicenseElement(node *html.Node, strict bool) string {
	// Check in href for CC license
	if href := trim(dom.GetAttribute(node, "href")); href != "" {
		if parts := rxCcLicense.FindStringSubmatch(href); len(parts) > 0 {
			return fmt.Sprintf("CC %s %s", strings.ToUpper(parts[1]), parts[2])
		}
	}

	// Check in text
	if text := trim(etree.Text(node)); text != "" {
		if !strict {
			return text
		}

		if parts := rxCcLicenseText.FindStringSubmatch(text); len(parts) > 0 {
			return parts[0]
		}
	}

	return ""
}

func normalizeAuthors(authors string, input string) string {
	if rxUrlCheck.MatchString(input) || rxAuthorEmail.MatchString(input) {
		return authors
	}

	// Clean up input string
	input = trim(input)
	input = html.UnescapeString(input)
	input = gomoji.RemoveEmojis(input)
	input = rxAuthorDigits.ReplaceAllString(input, "")
	input = rxAuthorSocialMedia.ReplaceAllString(input, "")
	input = rxAuthorSpaceChars.ReplaceAllString(input, " ")

	// Fix HTML entities
	if strings.Contains(input, "&#") || strings.Contains(input, "&amp;") {
		input = html.UnescapeString(input)
	}

	// Remove HTML tags
	input = rxAuthorHTML.ReplaceAllString(input, "")

	// Prepare list of current authors
	listAuthor := strings.Split(authors, "; ")
	if len(listAuthor) == 1 && listAuthor[0] == "" {
		listAuthor = []string{}
	}

	// Track the existing authors
	tracker := sliceToMap(listAuthor...)

	// Save the new authors
	for _, a := range rxAuthorSeparator.Split(input, -1) {
		// Clean the author
		a = rxAuthorNickname.ReplaceAllString(a, "")
		a = rxAuthorSpecialChars.ReplaceAllString(a, "")
		a = rxAuthorPrefix.ReplaceAllString(a, "")
		a = rxAuthorPreposition.ReplaceAllString(a, "")
		a = trim(a)

		// Stop if author is empty, or single word but too long.
		// The max length 23 is taken from ISO IEC-7813.
		length := utf8.RuneCountInString(a)
		hasDash := strings.Contains(a, "-")
		hasSpace := strings.Contains(a, " ")
		if length == 0 || (!hasDash && !hasSpace && length >= 50) {
			continue
		}

		// If necessary, convert to title
		firstRune, _ := utf8.DecodeRuneInString(a)
		if !unicode.IsUpper(firstRune) || strings.ToLower(a) == a {
			a = cases.Title(language.English).String(a)
		}

		// Save to list
		_, tracked := tracker[a]
		if !strings.Contains(authors, a) && !tracked {
			tracker[a] = struct{}{}
			listAuthor = append(listAuthor, a)
		}
	}

	return strings.Join(listAuthor, "; ")
}

func removeBlacklistedAuthors(current string, opts Options) string {
	if current == "" {
		return current
	}

	blacklisted := make(map[string]struct{})
	for _, ab := range opts.BlacklistedAuthors {
		blacklisted[strings.ToLower(ab)] = struct{}{}
	}

	var allowedAuthors []string
	for _, author := range strings.Split(current, ";") {
		author = strings.TrimSpace(author)
		if _, exist := blacklisted[strings.ToLower(author)]; !exist {
			allowedAuthors = append(allowedAuthors, author)
		}
	}

	if len(allowedAuthors) > 0 {
		return strings.Join(allowedAuthors, "; ")
	}

	return ""
}
