package trafilatura

import (
	"regexp"

	"github.com/go-shiori/dom"
	"github.com/markusmobius/go-trafilatura/etree"
	"github.com/markusmobius/go-trafilatura/selector"
	"golang.org/x/net/html"
)

var rxWords = regexp.MustCompile(`\w`)

// docCleaning cleans the document by discarding unwanted elements
func docCleaning(doc *html.Node, includeTables, includeImages bool) {
	// Determine cleaning strategy
	cleaningList := duplicateMap(tagsToClean)
	strippingList := duplicateMap(tagsToStrip)

	if !includeTables {
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
	subElements := dom.GetElementsByTagName(doc, "*")
	for i := len(subElements) - 1; i >= 0; i-- {
		tagName := dom.TagName(subElements[i])
		if _, exist := emptyTagsToRemove[tagName]; !exist {
			continue
		}

		if len(dom.ChildNodes(subElements[i])) == 0 {
			etree.Remove(subElements[i])
		}
	}
}

func discardUnwantedComments(tree *html.Node) {
	subElements := dom.GetElementsByTagName(tree, "*")
	for i := len(subElements) - 1; i >= 0; i-- {
		subElement := subElements[i]
		for _, rule := range selector.DiscardedCommentsRule {
			if rule(subElement) {
				etree.Remove(subElement)
				break
			}
		}
	}
}

// handleTextNode converts, formats and probes potential text elements.
func handleTextNode(node *html.Node, cache *Cache, fixComments, deduplicate bool) *html.Node {
	// Make sure text is not empty
	text := etree.Text(node)
	tail := etree.Tail(node)
	if text == "" || tail == "" {
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

		if deduplicate && cache != nil && duplicateTest(node, cache) {
			return nil
		}
	} else {
		return nil
	}

	return node
}
