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
	"unicode/utf8"

	"github.com/go-shiori/dom"
	"github.com/markusmobius/go-trafilatura/internal/etree"
	"github.com/markusmobius/go-trafilatura/internal/lru"
	"github.com/markusmobius/go-trafilatura/internal/selector"
	"golang.org/x/net/html"
)

// docCleaning cleans the document by discarding unwanted elements.
// In iriginal it's named `tree_cleaning`.
func docCleaning(doc *html.Node, opts Options) {
	// Determine cleaning strategy
	cleaningList := duplicateMap(tagsToClean)
	strippingList := duplicateMap(tagsToStrip)

	if opts.ExcludeTables {
		cleaningList["table"] = struct{}{}
		cleaningList["td"] = struct{}{}
		cleaningList["th"] = struct{}{}
		cleaningList["tr"] = struct{}{}
	} else {
		for _, figure := range dom.QuerySelectorAll(doc, "figure") {
			var hasTableDescendant bool
			for _, child := range dom.GetElementsByTagName(figure, "*") {
				if dom.TagName(child) == "table" {
					hasTableDescendant = true
					break
				}
			}

			if hasTableDescendant {
				figure.Data = "div"
			}
		}
	}

	if opts.IncludeImages {
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

// Simplify relevant HTML tags for faster processing.
func simplifyTags(doc *html.Node, opts Options) {
	if !opts.IncludeLinks {
		links := dom.GetElementsByTagName(doc, "a")
		for i := len(links) - 1; i >= 0; i-- {
			link := links[i]
			linkParent := link.Parent

			// Check its parent
			if linkParent != nil {
				if ancestorIs(link, "div") || ancestorIs(link, "ul") ||
					(!opts.ExcludeTables && ancestorIs(link, "table")) {
					continue
				}
			}

			// Strip the link
			etree.Strip(link)
		}
	}
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
	allElements := dom.GetElementsByTagName(doc, "*")
	for i := len(allElements) - 1; i >= 0; i-- {
		subElement := allElements[i]
		tagName := dom.TagName(subElement)
		if _, exist := emptyTagsToRemove[tagName]; !exist {
			continue
		}

		if len(dom.ChildNodes(subElement)) == 0 {
			subElement.Parent.RemoveChild(subElement)
		}
	}
}

// pruneUnwantedNodes prune the HTML tree by removing unwanted sections.
func pruneUnwantedNodes(tree *html.Node, queries []selector.Rule, withBackup ...bool) *html.Node {
	var oldLen int
	var backup *html.Node
	backupEnabled := len(withBackup) > 0 && withBackup[0]

	tree = dom.Clone(tree, true)
	if backupEnabled {
		backup = dom.Clone(tree, true)
		oldLen = utf8.RuneCountInString(dom.TextContent(tree))
	}

	for _, query := range queries {
		subElements := selector.QueryAll(tree, query)
		for i := len(subElements) - 1; i >= 0; i-- {
			subElement := subElements[i]

			// Preserve tail text from deletion
			tail := etree.Tail(subElement)
			if tail != "" {
				previous := dom.PreviousElementSibling(subElement)
				if previous == nil {
					previous = subElement.Parent
				}

				if previous != nil {
					// There is a previous node, append text to its tail
					previousTail := etree.Tail(previous)
					if previousTail != "" {
						etree.SetTail(previous, previousTail+" "+tail)
					} else {
						etree.SetTail(previous, tail)
					}
				}
			}

			etree.Remove(subElement)
		}
	}

	if backupEnabled {
		newLen := utf8.RuneCountInString(dom.TextContent(tree))
		if newLen <= oldLen/7 {
			return backup
		}
	}

	return tree
}

// handleTextNode converts, formats and probes potential text elements.
func handleTextNode(node *html.Node, cache *lru.Cache, fixComments, preserveSpaces bool, opts Options) *html.Node {
	// Make sure text is not empty
	text := etree.Text(node)
	tail := etree.Tail(node)
	tagName := dom.TagName(node)
	children := dom.Children(node)
	if tagName == "done" || (len(children) == 0 && text == "" && tail == "") {
		return nil
	}

	// Line break bypass
	if !fixComments && inMap(tagName, mapXmlLbTags) {
		if !preserveSpaces {
			etree.SetTail(node, trim(tail))
		}
		return node
	}

	// If text is empty, try tail
	if text == "" && len(children) == 0 {
		text, tail = tail, ""
		etree.SetText(node, text)
		etree.SetTail(node, tail)

		// Handle differently for br/hr
		if fixComments && inMap(tagName, mapXmlLbTags) {
			node.Data = "p"
		}
	}

	// Trim values
	if !preserveSpaces {
		text, tail = trim(text), trim(tail)
		etree.SetText(node, text)
		etree.SetTail(node, tail)
	}

	if text == "" && textFilter(node) {
		return nil
	}

	if opts.Deduplicate && cache != nil && duplicateTest(node, cache, opts) {
		return nil
	}

	return node
}

// linkDensityTest check whether sections will be removed because it's rich in
// links (probably boilerplate)
func linkDensityTest(element *html.Node, opts Options) ([]*html.Node, bool) {
	// Fetch links in node
	links := dom.GetElementsByTagName(element, "a")
	if len(links) == 0 {
		return nil, false
	}

	// Prepare limit and threshold
	var limitLength int
	var threshold float64

	if dom.TagName(element) == "p" {
		if !opts.FavorPrecision {
			if dom.NextElementSibling(element) == nil {
				limitLength, threshold = 60, 0.8
			} else {
				limitLength, threshold = 30, 0.8
			}
		} else {
			limitLength, threshold = 200, 0.8
		}
	} else {
		if dom.NextElementSibling(element) == nil {
			limitLength, threshold = 300, 0.8
		} else {
			limitLength, threshold = 100, 0.8
		}
	}

	// Check if text of this node is within limit
	text := trim(dom.TextContent(element))
	textLength := utf8.RuneCountInString(text)
	if textLength < limitLength {
		// Collect link info
		linkLength, nShortLinks, nonEmptyLinks := collectLinkInfo(links, opts)
		nNonEmptyLinks := len(nonEmptyLinks)
		if nNonEmptyLinks == 0 {
			return nonEmptyLinks, true
		}

		// Check if links data surpass threshold
		if float64(linkLength) > threshold*float64(textLength) ||
			(nNonEmptyLinks > 1 && float64(nShortLinks)/float64(nNonEmptyLinks) > 0.8) {
			return nonEmptyLinks, true
		}
	}

	return nil, false
}

// linkDensityTestTables check whether a table will be removed because
// it's rich in links (probably boilerplate)
func linkDensityTestTables(table *html.Node, opts Options) bool {
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
		linkLength, _, nonEmptyLinks := collectLinkInfo(links, opts)
		nNonEmptyLinks := len(nonEmptyLinks)
		if nNonEmptyLinks == 0 {
			return true
		}

		if (textLength <= 1000 && float64(linkLength) > float64(textLength)*0.8) ||
			(textLength > 1000 && float64(linkLength) > float64(textLength)*0.5) {
			return true
		}

		// TODO: it seems to does more harm than good
		// if float64(nShortLinks) > float64(len(links))*0.66 {
		// 	return true
		// }
	}

	return false
}

// collectLinkInfo collects heuristics on link text.
func collectLinkInfo(links []*html.Node, opts Options) (linkLength, nShortLinks int, nonEmptyLinks []*html.Node) {
	// Longer strings impact recall in favor of precision
	threshold := 10
	if opts.FavorPrecision {
		threshold = 50
	}

	for _, link := range links {
		text := trim(dom.TextContent(link))
		textLength := utf8.RuneCountInString(text)
		if textLength == 0 {
			continue
		}

		linkLength += textLength
		if textLength < threshold {
			nShortLinks++
		}

		nonEmptyLinks = append(nonEmptyLinks, link)
	}

	return
}

// processNode converts, formats, and probes potential text elements (light format).
func processNode(element *html.Node, cache *lru.Cache, opts Options) *html.Node {
	text := etree.Text(element)
	tail := etree.Tail(element)
	tagName := dom.TagName(element)
	children := dom.Children(element)
	if tagName == "done" || (len(children) == 0 && text == "" && tail == "") {
		return nil
	}

	// Trim
	text, tail = trim(text), trim(tail)
	etree.SetText(element, text)
	etree.SetTail(element, tail)

	// Adapt content string
	if !inMap(tagName, mapXmlLbTags) && text == "" && tail != "" {
		text, tail = tail, ""
		etree.SetText(element, text)
		etree.SetTail(element, tail)
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

// ADDITIONAL:
// postCleaning is used to clean the extracted content.
// This is additional function that doesn't exist in original.
func postCleaning(doc *html.Node) {
	if doc == nil {
		return
	}

	// Remove empty nodes. Do it backward, to make sure all children
	// is removed before its parent.
	children := dom.GetElementsByTagName(doc, "*")
	for i := len(children) - 1; i >= 0; i-- {
		child := children[i]

		grandChildren := dom.Children(child)
		isVoidElement := dom.IsVoidElement(child)
		isEmpty := !textCharsTest(etree.Text(child))
		if len(grandChildren) == 0 && isEmpty && !isVoidElement {
			etree.Strip(child)
		}
	}

	// Remove useless attributes
	for _, element := range etree.Iter(doc) {
		tagName := dom.TagName(element)
		finalAttrs := []html.Attribute{}
		_, elementAllowedToHaveSize := elementWithSizeAttr[tagName]

		for _, attr := range element.Attr {
			// Exclude identification and presentational attributes.
			switch attr.Key {
			case "id", "class", "align", "background", "bgcolor", "border", "cellpadding",
				"cellspacing", "frame", "hspace", "rules", "style", "valign", "vspace":
				continue
			case "width", "height":
				if !elementAllowedToHaveSize {
					continue
				}
			}

			// Exclude unsafe attributes
			if _, allowed := allowedAttributes[attr.Key]; !allowed {
				continue
			}

			finalAttrs = append(finalAttrs, attr)
		}

		element.Attr = finalAttrs
	}
}

// deleteByLinkDensity determines the link density of elements with respect to
// their length, and remove the elements identified as boilerplate.
func deleteByLinkDensity(subTree *html.Node, opts Options, backtracking bool, tagNames ...string) {
	var nodesToDelete []*html.Node
	textNodes := make(map[string][]*html.Node)

	for _, elem := range etree.Iter(subTree, tagNames...) {
		nonEmptyLinks, isHighDensity := linkDensityTest(elem, opts)

		if isHighDensity {
			nodesToDelete = append(nodesToDelete, elem)
			continue
		}

		if backtracking && len(nonEmptyLinks) > 0 {
			text := trim(dom.TextContent(elem))
			textNodes[text] = append(textNodes[text], elem)
		}
	}

	if backtracking {
		threshold := 100
		if opts.FavorPrecision {
			threshold = 200
		}

		for text, nodes := range textNodes {
			textLength := utf8.RuneCountInString(text)
			if textLength > 0 && textLength < threshold && len(nodes) >= 3 {
				nodesToDelete = append(nodesToDelete, nodes...)
			}
		}
	}

	for i := len(nodesToDelete) - 1; i >= 0; i-- {
		etree.Remove(nodesToDelete[i])
	}
}

// ancestorIs checks if a given node has one of its ancestor tag name matching the provided one.
func ancestorIs(node *html.Node, tag string) bool {
	for node.Parent != nil {
		if dom.TagName(node.Parent) == tag {
			return true
		}
		node = node.Parent
	}
	return false
}

// Simplify HTML markup.
// Here in original Trafilatura we are supposed to convert HTML tags
// into the one that suitable for XML. However, since we prefer the results
// to be HTML, we won't do it here.
func convertTags(tree *html.Node, opts Options) {
	// Delete links for faster processing
	if !opts.IncludeLinks {
		// Prepare selector
		cssSelector := "div a, ul a, ol a" // "p a" ?
		if !opts.ExcludeTables {
			cssSelector += ", table a"
		}

		// Temporary change tags
		importantLinks := dom.QuerySelectorAll(tree, cssSelector)
		for _, elem := range importantLinks {
			elem.Data = "protected-a"
		}

		// Strip the rest of links
		etree.StripTags(tree, "a")

		// Revert back
		for _, elem := range importantLinks {
			elem.Data = "a"
		}
	} else {
		// Convert relative URL to absolute
		for _, elem := range dom.QuerySelectorAll(tree, "a") {
			// Extract link
			href := trim(dom.GetAttribute(elem, "href"))
			target := trim(dom.GetAttribute(elem, "target"))

			// Clear up existing attributes
			elem.Attr = nil

			// Convert relative URL to absolute
			if href != "" {
				href = createAbsoluteURL(href, opts.OriginalURL)
				dom.SetAttribute(elem, "href", href)
			}

			if target != "" {
				target = createAbsoluteURL(target, opts.OriginalURL)
				dom.SetAttribute(elem, "target", target)
			}
		}
	}

	// Iterate over all concerned elements.
	// In this case we only care about quotes.
	for _, elem := range etree.Iter(tree, listXmlQuoteTags...) {
		var codeFlag bool

		// Pre with a single span is more likely to be code
		if dom.TagName(elem) == "pre" {
			children := dom.Children(elem)
			if len(children) == 1 && dom.TagName(children[0]) == "span" {
				codeFlag = true
			}
		}

		// Find hljs elements to detect if it's code
		hljsSelector := `span[class*=" hljs"], span[class^="hljs"]`
		hljsElems := dom.QuerySelectorAll(elem, hljsSelector)
		if len(hljsElems) > 0 {
			codeFlag = true
			for _, hljsElem := range hljsElems {
				hljsElem.Attr = nil
			}
		}

		if codeFlag {
			elem.Data = "code"
		}
	}
}
