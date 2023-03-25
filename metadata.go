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
	"encoding/json"
	nurl "net/url"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	htmlxpath "github.com/antchfx/htmlquery"
	"github.com/forPelevin/gomoji"
	"github.com/go-shiori/dom"
	"github.com/markusmobius/go-htmldate"
	"github.com/markusmobius/go-trafilatura/internal/etree"
	"golang.org/x/net/html"
)

var (
	rxCommaSeparator = regexp.MustCompile(`\s*[,;]\s*`)
	rxTitleCleaner   = regexp.MustCompile(`(?i)^(.+)?\s+[-|]\s+(.+)$`) // part without dots?
	rxJsonSymbol     = regexp.MustCompile(`[{\\}]`)
	rxNameJson       = regexp.MustCompile(`(?i)"name?\\?": ?\\?"([^"\\]+)`)
	rxUrlCheck       = regexp.MustCompile(`(?i)https?://|/`)
	rxDomainFinder   = regexp.MustCompile(`(?i)https?://[^/]+`)
	rxSitenameFinder = regexp.MustCompile(`(?i)https?://(?:www\.|w[0-9]+\.)?([^/]+)`)
	rxHtmlStripTag   = regexp.MustCompile(`(?i)(<!--.*?-->|<[^>]*>)`)
	rxCategoryHref   = regexp.MustCompile(`(?i)/categor(?:y|ies)/`)
	rxTagHref        = regexp.MustCompile(`(?i)/tags?/`)

	rxAuthorPrefix       = regexp.MustCompile(`(?i)^([a-zäöüß]+(ed|t))? ?(written by|words by|by|von) `)
	rxAuthorDigits       = regexp.MustCompile(`(?i)\d.+?$`)
	rxAuthorSpecialChars = regexp.MustCompile(`(?i)[:()?*$#!%/<>{}~.]`)
	rxAuthorSpaceChars   = regexp.MustCompile(`(?i)[._+]`)
	rxAuthorSocialMedia  = regexp.MustCompile(`(?i)@\S+`)
	rxAuthorPreposition  = regexp.MustCompile(`(?i)[^\w]+$|\b\s+(am|on|for|at|in|to|from|of|via|with)\b\s+(.*)`)
	rxAuthorSeparator    = regexp.MustCompile(`(?i)/|;|,|\||&|(?:^|\W)[u|a]nd(?:$|\W)`)
	rxPrefixHttp         = regexp.MustCompile(`(?i)^http`)

	metaNameAuthor      = []string{"author", "byl", "dc.creator", "dcterms.creator", "sailthru.author", "citation_author"} // twitter:creator
	metaNameTitle       = []string{"title", "dc.title", "dcterms.title", "fb_title", "sailthru.title", "twitter:title", "citation_title"}
	metaNameDescription = []string{"description", "dc.description", "dcterms.description", "dc:description", "sailthru.description", "twitter:description"}
	metaNamePublisher   = []string{"copyright", "dc.publisher", "dcterms.publisher", "publisher", "citation_journal_title"}

	fastHtmlDateOpts      = htmldate.Options{UseOriginalDate: true, SkipExtensiveSearch: true}
	extensiveHtmlDateOpts = htmldate.Options{UseOriginalDate: true, SkipExtensiveSearch: false}
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
	License     string
}

func extractMetadata(doc *html.Node, opts Options) Metadata {
	// Extract metadata from <meta> tags
	metadata := examineMeta(doc)

	// Extract metadata from JSON-LD and override
	metadata = extractJsonLd(doc, metadata)

	// Try extracting from DOM element using selectors
	// Title
	if metadata.Title == "" {
		metadata.Title = extractDomTitle(doc)
	}

	// Author
	if metadata.Author == "" {
		metadata.Author = extractDomAuthor(doc)
	}

	// URL
	if metadata.URL == "" {
		metadata.URL = extractDomURL(doc, opts.OriginalURL)
	}

	// Hostname
	if metadata.URL != "" {
		metadata.Hostname = extractDomainURL(metadata.URL)
	}

	// Publish date
	var htmlDateOpts htmldate.Options
	if opts.HtmlDateOptions != nil {
		htmlDateOpts = *opts.HtmlDateOptions
	} else {
		if opts.NoFallback { // No fallback means we want it fast
			htmlDateOpts = fastHtmlDateOpts
		} else {
			htmlDateOpts = extensiveHtmlDateOpts
		}
	}

	htmlDateOpts.URL = metadata.URL
	publishDate, err := htmldate.FromDocument(doc, htmlDateOpts)
	if err == nil && !publishDate.IsZero() {
		metadata.Date = publishDate.DateTime
	}

	// Sitename
	if metadata.Sitename == "" {
		metadata.Sitename = extractDomSitename(doc)
	}

	if metadata.Sitename != "" {
		// Scrap Twitter ID
		metadata.Sitename = strings.TrimPrefix(metadata.Sitename, "@")

		// Capitalize
		firstRune := getRune(metadata.Sitename, 0)
		if !strings.Contains(metadata.Sitename, ".") && !unicode.IsUpper(firstRune) {
			metadata.Sitename = strings.Title(metadata.Sitename)
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
	for _, element := range dom.QuerySelectorAll(doc, `a[rel="license"]`) {
		if text := trim(etree.Text(element)); text != "" {
			metadata.License = text
			break
		}
	}

	return metadata
}

// examineMeta search meta tags for relevant information
func examineMeta(doc *html.Node) Metadata {
	// Bootstrap metadata from OpenGraph tags
	metadata := extractOpenGraphMeta(doc)

	// If all OpenGraph metadata have been assigned, we can return it
	if metadata.Title != "" && metadata.Author != "" && metadata.URL != "" &&
		metadata.Description != "" && metadata.Sitename != "" {
		return metadata
	}

	// Scan all <meta> nodes that has attribute "content"
	var tmpSitename string
	for _, node := range dom.QuerySelectorAll(doc, "meta[content]") {
		// Make sure content is not empty
		content := dom.GetAttribute(node, "content")
		content = rxHtmlStripTag.ReplaceAllString(content, "")
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
			}
			continue
		}

		// Handle name attribute
		name := dom.GetAttribute(node, "name")
		name = strings.ToLower(name)
		name = trim(name)

		if name != "" {
			if strIn(name, metaNameAuthor...) {
				content = rxHtmlStripTag.ReplaceAllString(content, "")
				metadata.Author = normalizeAuthors(metadata.Author, content)
			} else if strIn(name, metaNameTitle...) {
				metadata.Title = strOr(metadata.Title, content)
			} else if strIn(name, metaNameDescription...) {
				metadata.Description = strOr(metadata.Description, content)
			} else if strIn(name, metaNamePublisher...) {
				metadata.Sitename = strOr(metadata.Sitename, content)
			} else if strIn(name, "twitter:site", "application-name") || strings.Contains(name, "twitter:app:name") {
				tmpSitename = content
			} else if name == "twitter:url" {
				if isAbs, _ := isAbsoluteURL(content); metadata.URL == "" && isAbs {
					metadata.URL = content
				}
			} else if name == "keywords" { // "page-topic"
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

	// Clean up author
	metadata.Author = validateMetadataAuthor(metadata.Author)
	return metadata
}

// extractOpenGraphMeta search meta tags following the OpenGraph guidelines (https://ogp.me/)
func extractOpenGraphMeta(doc *html.Node) Metadata {
	var metadata Metadata

	// Scan all <meta> nodes whose property starts with "og:"
	for _, node := range dom.QuerySelectorAll(doc, `head > meta[property^="og:"]`) {
		// Get property name
		propName := dom.GetAttribute(node, "property")
		propName = trim(propName)

		// Make sure node has content attribute
		content := dom.GetAttribute(node, "content")
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
			metadata.Author = content
		case "og:url":
			if isAbs, _ := isAbsoluteURL(content); isAbs {
				metadata.URL = content
			}
		}
	}

	return metadata
}

func validateMetadataAuthor(author string) string {
	if author == "" {
		return author
	}

	if !strings.Contains(author, " ") || strings.HasPrefix(author, "http") {
		return ""
	}

	// Make sure author doesn't contain JSON symbols (in case JSON+LD has wrong format)
	if rxJsonSymbol.MatchString(author) {
		return ""
	}

	return author
}

// extractJsonLd search metadata from JSON+LD data following the Schema.org guidelines
// (https://schema.org). Here we don't really care about error here, so if parse failed
// we just return the original metadata.
func extractJsonLd(doc *html.Node, originalMetadata Metadata) Metadata {
	// Find all script nodes that contain JSON+Ld schema
	scriptNodes1 := dom.QuerySelectorAll(doc, `script[type="application/ld+json"]`)
	scriptNodes2 := dom.QuerySelectorAll(doc, `script[type="application/settings+json"]`)
	scriptNodes := append(scriptNodes1, scriptNodes2...)

	// Process each script node
	var metadata Metadata
	for _, script := range scriptNodes {
		// Get the json text inside the script
		jsonLdText := dom.TextContent(script)
		jsonLdText = strings.TrimSpace(jsonLdText)
		if jsonLdText == "" {
			continue
		}

		// Decode JSON text, assuming it is an object
		data := map[string]interface{}{}
		err := json.Unmarshal([]byte(jsonLdText), &data)
		if err != nil {
			continue
		}

		// Find articles and persons inside JSON+LD recursively
		persons := make([]map[string]interface{}, 0)
		articles := make([]map[string]interface{}, 0)

		var findImportantObjects func(obj map[string]interface{})
		findImportantObjects = func(obj map[string]interface{}) {
			// First check if this object type matches with our need.
			if objType, hasType := obj["@type"]; hasType {
				if strObjType, isString := objType.(string); isString {
					isPerson := strObjType == "Person"
					isArticle := strings.Contains(strObjType, "Article") ||
						strObjType == "SocialMediaPosting" ||
						strObjType == "Report"

					switch {
					case isArticle:
						articles = append(articles, obj)
						return

					case isPerson:
						persons = append(persons, obj)
						return
					}
				}
			}

			// If not, look in its children
			for _, value := range obj {
				switch v := value.(type) {
				case map[string]interface{}:
					findImportantObjects(v)

				case []interface{}:
					for _, item := range v {
						itemObject, isObject := item.(map[string]interface{})
						if isObject {
							findImportantObjects(itemObject)
						}
					}
				}
			}
		}

		findImportantObjects(data)

		// Extract metadata from each article
		for _, article := range articles {
			if metadata.Author == "" {
				// For author, if taken from schema, we only want it from schema with type "Person"
				metadata.Author = extractJsonArticleThingName(article, "author", "Person")
				metadata.Author = validateMetadataAuthor(metadata.Author)
			}

			if metadata.Sitename == "" {
				metadata.Sitename = extractJsonArticleThingName(article, "publisher")
			}

			if len(metadata.Categories) == 0 {
				if section, exist := article["articleSection"]; exist {
					category := extractJsonString(section)
					metadata.Categories = append(metadata.Categories, category)
				}
			}

			if metadata.Title == "" {
				if name, exist := article["name"]; exist {
					metadata.Title = extractJsonString(name)
				}
			}

			// If title is empty or only consist of one word, try to look in headline
			if metadata.Title == "" || strWordCount(metadata.Title) == 1 {
				for key, value := range article {
					if !strings.Contains(strings.ToLower(key), "headline") {
						continue
					}

					title := extractJsonString(value)
					if title == "" || strings.Contains(title, "...") {
						continue
					}

					metadata.Title = title
					break
				}
			}
		}

		// If author not found, look in persons
		if metadata.Author == "" {
			names := []string{}
			for _, person := range persons {
				personName := extractJsonThingName(person)
				personName = validateMetadataAuthor(personName)
				if personName != "" {
					names = append(names, personName)
				}
			}

			if len(names) > 0 {
				metadata.Author = strings.Join(names, "; ")
			}
		}

		// Stop if all metadata found
		if metadata.Author != "" && metadata.Sitename != "" &&
			len(metadata.Categories) != 0 && metadata.Title != "" {
			break
		}
	}

	// If available, override author and categories in original metadata
	originalMetadata.Author = strOr(metadata.Author, originalMetadata.Author)
	if len(metadata.Categories) > 0 {
		originalMetadata.Categories = metadata.Categories
	}

	// If the new sitename exist and longer, override the original
	if utf8.RuneCountInString(metadata.Sitename) > utf8.RuneCountInString(originalMetadata.Sitename) {
		originalMetadata.Sitename = metadata.Sitename
	}

	// The new title is only used if original metadata doesn't have any title
	if originalMetadata.Title == "" {
		originalMetadata.Title = metadata.Title
	}

	// Clean up authors
	originalMetadata.Author = normalizeAuthors("", originalMetadata.Author)

	return originalMetadata
}

func extractJsonArticleThingName(article map[string]interface{}, key string, allowedTypes ...string) string {
	// Fetch value from the key
	value, exist := article[key]
	if !exist {
		return ""
	}

	return extractJsonThingName(value, allowedTypes...)
}

func extractJsonThingName(iface interface{}, allowedTypes ...string) string {
	// Decode the value of interface
	switch val := iface.(type) {
	case string:
		// There are some case where the string contains an unescaped
		// JSON, so try to handle it here
		if rxJsonSymbol.MatchString(val) {
			matches := rxNameJson.FindStringSubmatch(val)
			if len(matches) == 0 {
				return ""
			}
			val = matches[1]
		}

		// Clean up the string
		return trim(val)

	case map[string]interface{}:
		// If it's object, make sure its type allowed
		if len(allowedTypes) > 0 {
			if objType, hasType := val["@type"]; hasType {
				if strObjType, isString := objType.(string); isString {
					if !strIn(strObjType, allowedTypes...) {
						return ""
					}
				}
			}
		}

		// Return its name
		if iName, exist := val["name"]; exist {
			return extractJsonString(iName)
		}

	case []interface{}:
		// If it's array, merge names into one
		names := []string{}
		for _, entry := range val {
			switch entryVal := entry.(type) {
			case string:
				entryVal = trim(entryVal)
				names = append(names, entryVal)

			case map[string]interface{}:
				if iName, exist := entryVal["name"]; exist {
					if name := extractJsonString(iName); name != "" {
						names = append(names, name)
					}
				}
			}
		}

		if len(names) > 0 {
			return strings.Join(names, "; ")
		}
	}

	return ""
}

func extractJsonString(iface interface{}) string {
	if s, isString := iface.(string); isString {
		return trim(s)
	}

	return ""
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
	title := extractDomMetaSelectors(doc, 200, MetaTitleXpaths)
	if title != "" {
		return title
	}

	// Look in <title> tag
	titleNode := dom.QuerySelector(doc, "head > title")
	if titleNode != nil {
		title := dom.TextContent(titleNode)
		title = trim(title)

		matches := rxTitleCleaner.FindStringSubmatch(title)
		if len(matches) > 0 {
			if !strings.Contains(matches[1], ".") {
				title = matches[1]
			} else if !strings.Contains(matches[2], ".") {
				title = matches[2]
			}
		}

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
	author := extractDomMetaSelectors(doc, 120, MetaAuthorXpaths)
	if author != "" {
		return normalizeAuthors("", author)
	}

	return ""
}

// extractDomURL extracts the document URL from the canonical <link>.
func extractDomURL(doc *html.Node, defaultURL *nurl.URL) string {
	var url string

	// Try canonical link first
	linkNode := dom.QuerySelector(doc, `head link[rel="canonical"]`)
	if linkNode != nil {
		href := dom.GetAttribute(linkNode, "href")
		href = trim(href)
		if href != "" && rxUrlCheck.MatchString(href) {
			url = href
		}
	} else {
		// Now try default language link
		linkNodes := dom.QuerySelectorAll(doc, `head link[rel="alternate"]`)
		for _, node := range linkNodes {
			hreflang := dom.GetAttribute(node, "hreflang")
			if hreflang == "x-default" {
				href := dom.GetAttribute(node, "href")
				href = trim(href)
				if href != "" && rxUrlCheck.MatchString(href) {
					url = href
				}
			}
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
				domainMatches := rxDomainFinder.FindStringSubmatch(nodeContent)
				if len(domainMatches) > 0 {
					url = domainMatches[0] + url
					break
				}
			}
		}
	}

	// Validate URL
	if url != "" {
		// If it's already an absolute URL, return it
		if isAbs, _ := isAbsoluteURL(url); isAbs {
			return url
		}

		// If not, try to convert it into absolute URL using default URL
		// instead of using domain name
		newURL := createAbsoluteURL(url, defaultURL)
		if isAbs, _ := isAbsoluteURL(newURL); isAbs {
			return newURL
		}
	}

	// At this point, URL is either empty or not absolute, so just return the default URL
	if defaultURL != nil {
		return defaultURL.String()
	}

	// If default URL is not specified, just give up
	return ""
}

// extractDomSitename extracts the name of a site from the main title (if it exists).
func extractDomSitename(doc *html.Node) string {
	titleNode := dom.QuerySelector(doc, "head > title")
	if titleNode == nil {
		return ""
	}

	titleText := trim(dom.TextContent(titleNode))
	if titleText == "" {
		return ""
	}

	matches := rxTitleCleaner.FindStringSubmatch(titleText)
	if len(matches) > 0 {
		if strings.Contains(matches[1], ".") {
			return matches[1]
		} else if strings.Contains(matches[2], ".") {
			return matches[2]
		}
	}

	return ""
}

// extractDomCategories returns the categories of the document.
func extractDomCategories(doc *html.Node) []string {
	categories := []string{}

	// Try using selectors
	for _, query := range MetaCategoriesXpaths {
		for _, node := range htmlxpath.Find(doc, query) {
			href := dom.GetAttribute(node, "href")
			href = strings.TrimSpace(href)
			if href != "" && rxCategoryHref.MatchString(href) {
				text := dom.TextContent(node)
				text = trim(text)
				if text != "" {
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
		node := dom.QuerySelector(doc, `head meta[property="article:section"]`)
		if node != nil {
			content := dom.GetAttribute(node, "content")
			content = trim(content)
			if content != "" {
				categories = append(categories, content)
			}
		}
	}

	return categories
}

// extractDomTags returns the tags of the document.
func extractDomTags(doc *html.Node) []string {
	tags := []string{}

	// Try using selectors
	for _, query := range MetaTagsXpaths {
		for _, node := range htmlxpath.Find(doc, query) {
			href := dom.GetAttribute(node, "href")
			href = strings.TrimSpace(href)
			if href != "" && rxTagHref.MatchString(href) {
				text := dom.TextContent(node)
				text = trim(text)
				if text != "" {
					tags = append(tags, text)
				}
			}
		}

		if len(tags) > 0 {
			break
		}
	}

	return tags
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

func extractDomMetaSelectors(doc *html.Node, limit int, queries []string) string {
	for _, query := range queries {
		for _, node := range htmlxpath.Find(doc, query) {
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

func normalizeAuthors(authors string, input string) string {
	// Clean up input string
	input = trim(input)
	input = gomoji.RemoveEmojis(input)
	input = rxAuthorDigits.ReplaceAllString(input, "")
	input = rxAuthorSocialMedia.ReplaceAllString(input, "")
	input = rxAuthorSpecialChars.ReplaceAllString(input, "")
	input = rxAuthorSpaceChars.ReplaceAllString(input, " ")
	input = rxAuthorPreposition.ReplaceAllString(input, "")

	// Prepare list of current authors
	listAuthor := strings.Split(authors, "; ")
	if len(listAuthor) == 1 && listAuthor[0] == "" {
		listAuthor = []string{}
	}

	// Track the existing authors
	tracker := sliceToMap(listAuthor...)

	// Save the new authors
	for _, a := range rxAuthorSeparator.Split(input, -1) {
		a = trim(a)
		if a == "" {
			continue
		}

		a = strings.Title(a)
		a = rxAuthorPrefix.ReplaceAllString(a, "")
		_, tracked := tracker[a]
		if !strings.Contains(authors, a) && !rxPrefixHttp.MatchString(a) && !tracked {
			listAuthor = append(listAuthor, a)
		}
	}

	return strings.Join(listAuthor, "; ")
}
