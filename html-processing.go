package trafilatura

import (
	"regexp"
	"unicode/utf8"

	"github.com/go-shiori/dom"
	"github.com/markusmobius/go-trafilatura/internal/etree"
	"github.com/markusmobius/go-trafilatura/internal/selector"
	"golang.org/x/net/html"
)

var rxWords = regexp.MustCompile(`\w`)

// docCleaning cleans the document by discarding unwanted elements
func docCleaning(doc *html.Node, excludeTables, includeImages bool) {
	// Determine cleaning strategy
	cleaningList := duplicateMap(tagsToClean)
	strippingList := duplicateMap(tagsToStrip)

	if excludeTables {
		cleaningList["table"] = struct{}{}
	}

	if includeImages {
		// Many websites have <img> inside <figure> or <picture> or <source> tag
		delete(cleaningList, "figure")
		delete(cleaningList, "picture")
		delete(cleaningList, "source")
		delete(strippingList, "img")
	}

	// Remove nodes in cleaning list including its children
	for tagName := range cleaningList {
		etree.StripElements(doc, false, tagName)
	}

	// Remove nodes in stripping list but keep its children
	for tagName := range strippingList {
		etree.StripTags(doc, tagName)
	}

	// Remove HTML comment
	removeHtmlCommentNode(doc)
	pruneHTML(doc)
}

// removeHtmlCommentNode removes all `html.CommentNode` in document.
func removeHtmlCommentNode(doc *html.Node) {
	// Find all comment nodes
	var finder func(*html.Node)
	var commentNodes []*html.Node

	finder = func(node *html.Node) {
		if node.Type == html.CommentNode {
			commentNodes = append(commentNodes, node)
		}

		for child := node.FirstChild; child != nil; child = child.NextSibling {
			finder(child)
		}
	}

	for child := doc.FirstChild; child != nil; child = child.NextSibling {
		finder(child)
	}

	// Remove it
	dom.RemoveNodes(commentNodes, nil)
}

// pruneHTML deletes selected empty elements
func pruneHTML(doc *html.Node) {
	for _, subElement := range dom.GetElementsByTagName(doc, "*") {
		tagName := dom.TagName(subElement)
		if _, exist := emptyTagsToRemove[tagName]; !exist {
			continue
		}

		if len(dom.ChildNodes(subElement)) == 0 {
			etree.Remove(subElement)
		}
	}
}

// discardUnwanted deletes unwanted sections.
func discardUnwanted(tree *html.Node) {
	for _, subElement := range dom.GetElementsByTagName(tree, "*") {
		for _, rule := range selector.DiscardedContentRules {
			if rule(subElement) {
				etree.Remove(subElement)
				break
			}
		}
	}
}

// discardUnwantedComments deletes unwanted comment sections.
func discardUnwantedComments(tree *html.Node) {
	for _, subElement := range dom.GetElementsByTagName(tree, "*") {
		for _, rule := range selector.DiscardedCommentsRules {
			if rule(subElement) {
				etree.Remove(subElement)
				break
			}
		}
	}
}

// handleTextNode converts, formats and probes potential text elements.
func handleTextNode(node *html.Node, cache *Cache, fixComments bool, opts Options) *html.Node {
	// Make sure text is not empty
	text := etree.Text(node)
	tail := etree.Tail(node)
	if text == "" && tail == "" {
		return nil
	}

	// Line break bypass
	tagName := dom.TagName(node)
	if !fixComments && (tagName == "br" || tagName == "hr") {
		etree.SetTail(node, trim(tail))
		return node
	}

	// If text is empty, try tail
	if text == "" {
		text = tail
		etree.SetText(node, tail)
		etree.SetTail(node, "")

		// Handle differently for br/hr
		if fixComments && (tagName == "br" || tagName == "hr") {
			node.Data = "p"
		}
	}

	// Trim values
	text, tail = trim(text), trim(tail)
	etree.SetText(node, text)
	etree.SetTail(node, tail)

	if rxWords.MatchString(text) {
		if textFilter(node) {
			return nil
		}

		if opts.Deduplicate && cache != nil && duplicateTest(node, cache, opts) {
			return nil
		}
	} else {
		return nil
	}

	return node
}

// linkDensityTest check whether sections will be removed because it's rich in
// links (probably boilerplate)
func linkDensityTest(element *html.Node) ([]*html.Node, bool) {
	// Fetch links in node
	links := dom.GetElementsByTagName(element, "a")
	if len(links) == 0 {
		return nil, false
	}

	// Prepare limit and threshold
	var limitLength int
	var threshold float64

	switch {
	case dom.TagName(element) == "p":
		limitLength, threshold = 25, 0.9
	case element.NextSibling == nil:
		limitLength, threshold = 200, 0.66
	default:
		limitLength, threshold = 100, 0.66
	}

	// Check if text of this node is within limit
	text := trim(dom.TextContent(element))
	textLength := utf8.RuneCountInString(text)
	if textLength < limitLength {
		// Collect link info
		linkLength, nShortLinks, nonEmptyLinks := collectLinkInfo(links)
		nNonEmptyLinks := len(nonEmptyLinks)
		if nNonEmptyLinks == 0 {
			return nonEmptyLinks, true
		}

		// Check if links data surpass threshold
		if float64(linkLength) >= threshold*float64(textLength) ||
			float64(nShortLinks)/float64(nNonEmptyLinks) >= threshold {
			return nonEmptyLinks, true
		}
	}

	return nil, false
}

// linkDensityTestTables check whether a table will be removed because
// it's rich in links (probably boilerplate)
func linkDensityTestTables(table *html.Node) bool {
	// Fetch links in table
	links := dom.GetElementsByTagName(table, "a")
	if len(links) == 0 {
		return false
	}

	// Check text length
	text := trim(dom.TextContent(table))
	textLength := utf8.RuneCountInString(text)
	if textLength > 250 {
		// Collect link info
		linkLength, nShortLinks, nonEmptyLinks := collectLinkInfo(links)
		nNonEmptyLinks := len(nonEmptyLinks)
		if nNonEmptyLinks == 0 {
			return true
		}

		if (textLength <= 1000 && float64(linkLength) > float64(textLength)*0.8) ||
			(textLength > 1000 && float64(linkLength) > float64(textLength)*0.5) {
			return true
		}

		if float64(nShortLinks) > float64(len(links))*0.66 {
			return true
		}
	}

	return false
}

// collectLinkInfo collects heuristics on link text.
func collectLinkInfo(links []*html.Node) (linkLength, nShortLinks int, nonEmptyLinks []*html.Node) {
	for _, link := range links {
		text := trim(dom.TextContent(link))
		textLength := utf8.RuneCountInString(text)
		if textLength == 0 {
			continue
		}

		linkLength += textLength
		if textLength < 100 {
			nShortLinks++
		}

		nonEmptyLinks = append(nonEmptyLinks, link)
	}

	return
}

// processNode converts, formats, and probes potential text elements (light format).
func processNode(element *html.Node, cache *Cache, opts Options) *html.Node {
	tagName := dom.TagName(element)
	if tagName == "done" {
		return nil
	}

	text, tail := etree.Text(element), etree.Tail(element)
	if len(dom.Children(element)) == 0 && text == "" && tail == "" {
		return nil
	}

	// Trim
	text, tail = trim(text), trim(tail)
	etree.SetText(element, text)
	etree.SetTail(element, tail)

	// Adapt content string
	if (tagName != "br" && tagName != "hr") && text == "" && tail != "" {
		etree.SetText(element, tail)
		text = tail
	}

	// Content checks
	if text != "" || tail != "" {
		if textFilter(element) {
			return nil
		}

		if cache != nil && opts.Deduplicate && duplicateTest(element, cache, opts) {
			return nil
		}
	}

	return element
}

// postCleaning is used to clean the extracted content from useless attribute.
// This is additional function that doesn't exist in original.
func postCleaning(doc *html.Node) {
	for _, element := range etree.Iter(doc) {
		newAttr := []html.Attribute{}
		for _, attr := range element.Attr {
			if attr.Key == "id" || attr.Key == "class" {
				continue
			}

			_, isStyling := presentationalAttributes[attr.Key]
			if isStyling {
				continue
			}

			newAttr = append(newAttr, attr)
		}

		element.Attr = newAttr
	}
}
