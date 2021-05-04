package trafilatura

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/go-shiori/dom"
	"golang.org/x/net/html"
)

var rxWords = regexp.MustCompile(`\w`)

// docCleaning cleans the document by discarding unwanted elements
func docCleaning(doc *html.Node, includeTables, includeImages bool) {
	// Prepare list to be cleaned
	killList := duplicateMap(tagsToKill)
	removeList := duplicateMap(tagsToRemove)

	if !includeTables {
		killList["table"] = struct{}{}
	}

	if includeImages {
		// Many websites have <img> inside <figure> or <picture> or <source> tag
		delete(killList, "figure")
		delete(killList, "picture")
		delete(killList, "source")
		delete(removeList, "img")
	}

	// Remove HTML comment
	removeHtmlComments(doc)

	// Remove nodes in kill list including its children
	for tagName := range killList {
		nodes := dom.GetElementsByTagName(doc, tagName)
		removeNodes(nodes)
	}

	// Remove nodes in remove list but keep its children
	for tagName := range removeList {
		nodes := dom.GetElementsByTagName(doc, tagName)
		stripNodes(nodes)
	}

	pruneHTML(doc)
}

// pruneHTML deletes selected empty elements
func pruneHTML(doc *html.Node) {
	// Find all empty nodes
	var emptyNodes []*html.Node
	for _, node := range dom.GetElementsByTagName(doc, "*") {
		tagName := dom.TagName(node)
		if _, exist := emptyTagsToRemove[tagName]; !exist {
			continue
		}

		if len(dom.Children(node)) == 0 {
			emptyNodes = append(emptyNodes, node)
		}
	}

	removeNodes(emptyNodes)
}

func discardUnwanted(doc *html.Node) {
	var discardedNodes []*html.Node
	for _, n := range dom.GetElementsByTagName(doc, "*") {
		for _, discardRule := range discardedContentSelectorRules {
			if discardRule(n) {
				discardedNodes = append(discardedNodes, n)
				break
			}
		}
	}

	if len(discardedNodes) > 0 {
		removeNodes(discardedNodes)
	}
}

// removeHtmlComments find all `html.CommentNode` in document then remove it.
func removeHtmlComments(doc *html.Node) {
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
	removeNodes(commentNodes)
}

func handleTextNode(node *html.Node, deduplicate bool, cache *Cache) *html.Node {
	// Make sure text is not empty
	text := dom.TextContent(node)
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	// If text doesn't contain any word, stop
	if !rxWords.MatchString(text) {
		return nil
	}

	// Check filter
	if textFilter(node) {
		return nil
	}

	if deduplicate && cache != nil && duplicateTest(node, cache) {
		return nil
	}

	return node
}

func linkDensityTest(n *html.Node) ([]*html.Node, bool) {
	// Fetch links in node
	links := dom.GetElementsByTagName(n, "a")
	if len(links) == 0 {
		return nil, false
	}

	// Prepare limit and threshold
	var limitLength int
	var threshold float64

	switch {
	case dom.TagName(n) == "p":
		limitLength, threshold = 25, 0.9
	case n.NextSibling == nil:
		limitLength, threshold = 200, 0.66
	default:
		limitLength, threshold = 100, 0.66
	}

	// Check if text of this node is within limit
	text := dom.TextContent(n)
	text = strings.TrimSpace(text)
	if utf8.RuneCountInString(text) >= limitLength {
		return nil, false
	}

	// Collect link info
	linkLength, nShortLinks, nonEmptyLinks := collectLinkInfo(links)
	nNonEmptyLinks := len(nonEmptyLinks)
	if nNonEmptyLinks == 0 {
		return nil, true
	}

	// Check if links data surpass threshold
	if float64(linkLength) >= threshold*float64(nNonEmptyLinks) ||
		float64(nShortLinks)/float64(nNonEmptyLinks) >= threshold {
		return nonEmptyLinks, true
	}

	return nil, false
}

func linkDensityTestTables(table *html.Node) bool {
	// Fetch links in table
	links := dom.GetElementsByTagName(table, "a")
	if len(links) == 0 {
		return false
	}

	// Check text length
	text := dom.TextContent(table)
	text = strNormalize(text)
	textLength := utf8.RuneCountInString(text)
	if textLength <= 250 {
		return false
	}

	// Collect link info
	linkTextLength, nShortLinks, nonEmptyLinks := collectLinkInfo(links)
	nNonEmptyLinks := len(nonEmptyLinks)
	if nNonEmptyLinks == 0 {
		// In this case, there are only empty links which obviously means
		// that this node doesn't have high link density. However, since it's
		// empty might as well delete it, so here we return true.
		return true
	}

	if (textLength <= 1000 && float64(linkTextLength) > float64(textLength)*0.8) ||
		(textLength > 1000 && float64(linkTextLength) > float64(textLength)*0.5) {
		return true
	}

	if float64(nShortLinks) > float64(len(links))*0.66 {
		return true
	}

	return false
}

func collectLinkInfo(links []*html.Node) (linkLength, nShortLinks int, nonEmptyLinks []*html.Node) {
	for _, link := range links {
		text := dom.TextContent(link)
		text = strings.TrimSpace(text)

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

func processNode(element *html.Node, cache *Cache, deduplicate bool) *html.Node {
	if dom.TagName(element) == "done" {
		return nil
	}

	text := dom.TextContent(element)
	text = strings.TrimSpace(text)
	if len(dom.Children(element)) == 0 && text != "" {
		return nil
	}

	if text != "" {
		if textFilter(element) {
			return nil
		}
		if cache != nil && deduplicate && duplicateTest(element, cache) {
			return nil
		}
	}

	return element
}

// removeNodes iterates over a nodeList and remove each of them.
func removeNodes(nodeList []*html.Node) {
	for i := len(nodeList) - 1; i >= 0; i-- {
		node := nodeList[i]
		parentNode := node.Parent
		if parentNode != nil {
			parentNode.RemoveChild(node)
		}
	}
}

// stripNodes iterates over a nodeList and remove each of them
// while still keeping the children.
func stripNodes(nodeList []*html.Node) {
	for i := len(nodeList) - 1; i >= 0; i-- {
		node := nodeList[i]

		// Make sure node has parent
		if node.Parent == nil {
			continue
		}

		// Make sure node has children
		childNodes := dom.ChildNodes(node)
		if len(childNodes) == 0 {
			continue
		}

		// Move children to parent
		for _, child := range childNodes {
			clone := dom.Clone(child, true)
			node.Parent.InsertBefore(clone, node)
		}

		// Remove the node itself
		node.Parent.RemoveChild(node)
	}
}
