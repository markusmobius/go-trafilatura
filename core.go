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
	"fmt"
	"io"
	nurl "net/url"
	"strings"
	"unicode/utf8"

	htmlxpath "github.com/antchfx/htmlquery"
	"github.com/go-shiori/dom"
	"github.com/markusmobius/go-trafilatura/internal/etree"
	"github.com/markusmobius/go-trafilatura/internal/lru"
	"golang.org/x/net/html"
)

// ExtractResult is the result of content extraction.
type ExtractResult struct {
	ContentNode  *html.Node
	CommentsNode *html.Node
	ContentText  string
	CommentsText string
	Metadata     Metadata
}

// Extract parses a reader and find the main readable content.
func Extract(r io.Reader, opts Options) (*ExtractResult, error) {
	// Parse HTML
	doc, err := dom.Parse(r)
	if err != nil {
		return nil, err
	}

	return ExtractDocument(doc, opts)
}

// ExtractDocument parses the specified document and find the main readable content.
func ExtractDocument(doc *html.Node, opts Options) (*ExtractResult, error) {
	//  Set default config
	if opts.Config == nil {
		opts.Config = DefaultConfig()
	}

	// Prepare cache for detecting text duplicate
	cache := lru.NewCache(opts.Config.CacheSize)

	// HTML language check
	if opts.TargetLanguage != "" && !checkHtmlLanguage(doc, opts, false) {
		return nil, fmt.Errorf("web page language is not %s", opts.TargetLanguage)
	}

	// Clone and backup document to make sure the original kept untouched
	doc = dom.Clone(doc, true)
	docBackup1 := dom.Clone(doc, true)
	docBackup2 := dom.Clone(doc, true)

	// Fetch metadata
	metadata := extractMetadata(doc, opts)

	// Check if essential metadata is missing
	if opts.HasEssentialMetadata {
		if metadata.Title == "" {
			return nil, fmt.Errorf("title is required")
		}

		if metadata.URL == "" {
			return nil, fmt.Errorf("url is required")
		}

		if metadata.Date.IsZero() {
			return nil, fmt.Errorf("date is required")
		}
	}

	// ADDITIONAL: If original URL never specified, and it found in metadata,
	// use the one from metadata.
	if opts.OriginalURL == nil && metadata.URL != "" {
		parsedURL, err := nurl.ParseRequestURI(metadata.URL)
		if err == nil {
			opts.OriginalURL = parsedURL
		}
	}

	// Clean document
	docCleaning(doc, opts)
	simplifyTags(doc, opts)

	// TODO: Here in original Trafilatura, we are supposed to convert HTML tags
	// into the one that suitable for XML. However, since we prefer the results
	// to be XML, we won't do it here.

	// Extract comments first, then remove
	var tmpComments string
	var lenComments int
	var commentsBody *html.Node

	if !opts.ExcludeComments { // Comment is included
		commentsBody, tmpComments = extractComments(doc, cache, opts)
		lenComments = utf8.RuneCountInString(tmpComments)
	} else if opts.FavorPrecision {
		doc = pruneUnwantedNodes(doc, RemovedCommentXpaths)
	}

	// Extract content
	postBody, tmpBodyText := extractContent(doc, cache, opts)

	// Use fallback if necessary
	if !opts.NoFallback || len(opts.FallbackCandidates) > 0 {
		postBody, tmpBodyText = compareExtraction(docBackup1, postBody, opts)
	}

	// Rescue: try to use original/dirty tree
	lenText := utf8.RuneCountInString(tmpBodyText)
	if lenText < opts.Config.MinExtractedSize {
		postBody, tmpBodyText = baseline(docBackup2)
	}

	// Tree size sanity check
	if opts.MaxTreeSize > 0 {
		if len(dom.Children(postBody)) > opts.MaxTreeSize {
			for tag := range formatTagCatalog {
				etree.StripTags(postBody, tag)
			}

			if nChildren := len(dom.Children(postBody)); nChildren > opts.MaxTreeSize {
				return nil, fmt.Errorf("output tree to long, discarding file : %d", nChildren)
			}
		}
	}

	// Size checks
	if lenComments < opts.Config.MinExtractedCommentSize {
		logWarn(opts, "not enough comments: %s", opts.OriginalURL)
	}

	lenText = utf8.RuneCountInString(tmpBodyText)
	if lenText < opts.Config.MinOutputSize && lenComments < opts.Config.MinOutputCommentSize {
		return nil, fmt.Errorf("text and comments are not long enough: %d %d", lenText, lenComments)
	}

	// Check duplicates at body level
	if opts.Deduplicate && duplicateTest(postBody, cache, opts) {
		return nil, fmt.Errorf("extracted body has been duplicated")
	}

	// Sanity check on language
	lang := languageClassifier(tmpBodyText, tmpComments)
	if opts.TargetLanguage != "" {
		if lang != opts.TargetLanguage {
			return nil, fmt.Errorf("wrong language, want %s got %s", opts.TargetLanguage, lang)
		}
	}

	// Put the captured language to metadata
	if lang != "" {
		metadata.Language = lang
	}

	// Post cleaning
	postCleaning(postBody)
	postCleaning(commentsBody)

	return &ExtractResult{
		ContentNode:  postBody,
		ContentText:  tmpBodyText,
		CommentsNode: commentsBody,
		CommentsText: tmpComments,
		Metadata:     metadata,
	}, nil
}

// extractComments try and extract comments out of potential sections in the HTML.
func extractComments(doc *html.Node, cache *lru.Cache, opts Options) (*html.Node, string) {
	// Prepare final container
	commentsBody := etree.Element("body")

	// Prepare potential tags
	potentialTags := duplicateMap(tagCatalog)

	// Process each selector rules
	for _, query := range CommentXpaths {
		// Capture first node that matched with the rule
		subTree := htmlxpath.FindOne(doc, query)

		// If no nodes matched, try next selector rule
		if subTree == nil {
			continue
		}

		// Prune
		subTree = pruneUnwantedNodes(subTree, DiscardedCommentXpaths)
		etree.StripTags(subTree, "a", "span")

		// Extract comments
		var processedElems []*html.Node
		for _, elem := range dom.GetElementsByTagName(subTree, "*") {
			processed := processCommentsNode(elem, potentialTags, cache, opts)
			if processed != nil {
				processedElems = append(processedElems, processed)
			}
		}
		etree.Extend(commentsBody, processedElems...)

		// Control
		if len(dom.Children(commentsBody)) > 0 {
			etree.Remove(subTree)
			break
		}
	}

	tmpComments := etree.IterText(commentsBody, " ")
	if tmpComments != "" {
		return commentsBody, tmpComments
	}

	return nil, ""
}

func processCommentsNode(elem *html.Node, potentialTags map[string]struct{}, cache *lru.Cache, opts Options) *html.Node {
	// Make sure node is one of the potential tags
	if _, isPotential := potentialTags[dom.TagName(elem)]; !isPotential {
		return nil
	}

	// Make sure node is not empty and not duplicated
	processedNode := handleTextNode(elem, cache, true, opts)
	if processedNode != nil {
		processedNode.Attr = nil
		return processedNode
	}

	return nil
}

// extractContent find the main content of a page using a set of selectors, then
// extract relevant elements, strip them of unwanted subparts and convert them.
func extractContent(doc *html.Node, cache *lru.Cache, opts Options) (*html.Node, string) {
	backupDoc := dom.Clone(doc, true)
	resultBody := dom.CreateElement("body")

	// Prepare potential tags
	potentialTags := duplicateMap(tagCatalog)

	if !opts.ExcludeTables {
		potentialTags["table"] = struct{}{}
		potentialTags["tr"] = struct{}{}
		potentialTags["th"] = struct{}{}
		potentialTags["td"] = struct{}{}
	}

	if opts.IncludeImages {
		potentialTags["img"] = struct{}{}
	}

	if opts.IncludeLinks {
		potentialTags["a"] = struct{}{}
	}

	// Iterate each selector rule
	for _, query := range ContentXpaths {
		// Capture first node that matched with the rule
		subTree := htmlxpath.FindOne(doc, query)

		// If no nodes matched, try next selector rule
		if subTree == nil {
			continue
		}

		// Prune the subtree
		subTree = pruneUnwantedSections(subTree, opts)
		// TODO: second pass?
		// deleteByLinkDensity(subTree, opts, false, listXmlListTags...)

		// Define iteration strategy
		_, tableIsPotentialTag := potentialTags["table"]
		if tableIsPotentialTag || opts.FavorPrecision {
			tables := etree.Iter(subTree, "table")
			for i := len(tables) - 1; i >= 0; i-- {
				if linkDensityTestTables(tables[i], opts) {
					etree.Remove(tables[i])
				}
			}
		}

		// If sub tree now empty, try other selector
		if len(dom.Children(subTree)) == 0 {
			continue
		}

		// Check if there are enough <p> with text
		var paragraphText string
		for _, p := range dom.GetElementsByTagName(doc, "p") {
			paragraphText += dom.TextContent(p)
		}

		factor := 3
		if opts.FavorRecall {
			factor = 5
		} else if opts.FavorPrecision {
			factor = 1
		}

		if paragraphText == "" ||
			utf8.RuneCountInString(paragraphText) < opts.Config.MinExtractedSize*factor {
			potentialTags["div"] = struct{}{}
		}

		// Polish list of potential tags
		if _, exist := potentialTags["a"]; !exist {
			etree.StripTags(subTree, "a")
		}

		if _, exist := potentialTags["span"]; !exist {
			etree.StripTags(subTree, "span")
		}

		// Fetch sub elements
		subElements := dom.GetElementsByTagName(subTree, "*")

		// Check if all sub elements are line break
		subTagTracker := map[string]struct{}{}
		for _, e := range subElements {
			subTagTracker[dom.TagName(e)] = struct{}{}
		}

		if _, hasLineBreak := subTagTracker["br"]; len(subTagTracker) == 1 && hasLineBreak {
			subElements = []*html.Node{subTree}
		}

		// Populate result body
		var processedElems []*html.Node
		for _, elem := range subElements {
			processed := handleTextElem(elem, potentialTags, cache, opts)
			if processed != nil {
				processedElems = append(processedElems, processed)
			}
		}
		etree.Extend(resultBody, processedElems...)

		// Remove trailing titles
		finalChildren := dom.Children(resultBody)
		for i := len(finalChildren) - 1; i >= 0; i-- {
			tagName := dom.TagName(finalChildren[i])
			if inMap(tagName, mapXmlHeadTags) || inMap(tagName, mapXmlRefTags) {
				etree.Remove(finalChildren[i])
				continue
			}
			break
		}

		// Exit the loop if the result has children
		if len(dom.Children(resultBody)) > 1 {
			break
		}
	}

	// Try parsing wild <p> elements if nothing found or text too short
	tmpText := trim(etree.IterText(resultBody, " "))
	tmpTextLength := utf8.RuneCountInString(tmpText)

	if len(dom.Children(resultBody)) == 0 || tmpTextLength < opts.Config.MinExtractedSize {
		if opts.FavorRecall {
			potentialTags = duplicateMap(potentialTags)
			potentialTags["div"] = struct{}{}
		}

		resultBody = dom.CreateElement("body")
		recoverWildText(backupDoc, resultBody, potentialTags, cache, opts)
		tmpText = trim(etree.IterText(resultBody, " "))
	}

	// Filter output
	etree.StripElements(resultBody, false, "done")
	etree.StripTags(resultBody, "div")

	return resultBody, tmpText
}

// handleTextElem process text element and determine how to deal with its content.
func handleTextElem(element *html.Node, potentialTags map[string]struct{}, cache *lru.Cache, opts Options) *html.Node {
	tagName := dom.TagName(element)

	if inMap(tagName, mapXmlListTags) {
		return handleLists(element, cache, opts)
	} else if inMap(tagName, mapXmlQuoteTags) || tagName == "code" {
		return handleQuotes(element, cache, opts)
	} else if inMap(tagName, mapXmlHeadTags) {
		return handleTitles(element, cache, opts)
	} else if tagName == "p" {
		return handleParagraphs(element, potentialTags, cache, opts)
	} else if inMap(tagName, mapXmlLbTags) {
		if textCharsTest(etree.Tail(element)) {
			if element = processNode(element, cache, opts); element != nil {
				newElement := etree.Element("p")
				etree.SetText(newElement, etree.Tail(element))
				return newElement
			}
		}
	} else if inMap(tagName, mapXmlHiTags) || inMap(tagName, mapXmlRefTags) || tagName == "span" {
		return handleFormatting(element, cache, opts)
	} else if tagName == "table" {
		if _, exist := potentialTags["table"]; exist {
			return handleTable(element, potentialTags, cache, opts)
		}
	} else if inMap(tagName, mapXmlGraphicTags) {
		if _, exist := potentialTags["img"]; exist {
			return handleImage(element)
		}
	}

	return handleOtherElement(element, potentialTags, cache, opts)
}

// handleLists process lists elements
func handleLists(element *html.Node, cache *lru.Cache, opts Options) *html.Node {
	var newChildElem *html.Node
	processedElement := etree.Element(dom.TagName(element))

	if text := strings.TrimSpace(etree.Text(element)); text != "" {
		newChildElem = etree.SubElement(processedElement, "li")
		etree.SetText(newChildElem, text)
	}

	for _, child := range etree.Iter(element, listXmlItemTags...) {
		newChildElem = dom.CreateElement(dom.TagName(child))

		if len(dom.Children(child)) == 0 {
			processedChild := processNode(child, cache, opts)
			if processedChild != nil {
				newText := etree.Text(processedChild)
				if tail := strings.TrimSpace(etree.Tail(processedChild)); tail != "" {
					newText += " " + tail
				}

				etree.SetText(newChildElem, newText)
				etree.Append(processedElement, newChildElem)
			}
		} else {
			etree.SetText(newChildElem, etree.Text(child))

			for _, subElement := range dom.GetElementsByTagName(child, "*") {
				// Beware of nested list
				var processedSubChild *html.Node
				if inMap(dom.TagName(subElement), mapXmlListTags) {
					processedSubChild = handleLists(subElement, cache, opts)
					if processedSubChild != nil {
						dom.AppendChild(newChildElem, processedSubChild)
					}
				} else {
					processedSubChild := handleTextNode(subElement, cache, false, opts)
					if processedSubChild != nil {
						subChildElement := etree.SubElement(newChildElem, dom.TagName(processedSubChild))
						etree.SetText(subChildElement, etree.Text(processedSubChild))
						etree.SetTail(subChildElement, etree.Tail(processedSubChild))
						subChildElement.Attr = append([]html.Attribute{}, subElement.Attr...)
					}
				}

				subElement.Data = "done"
			}

			// etree.StripTags(newChild, "dd", "dt", "li")
			if tail := strings.TrimSpace(etree.Tail(child)); tail != "" {
				var newChildElemChildren []*html.Node
				for _, nc := range dom.Children(newChildElem) {
					if dom.TagName(nc) != "done" {
						newChildElemChildren = append(newChildElemChildren, nc)
					}
				}

				if len(newChildElemChildren) > 0 {
					lastSubChild := newChildElemChildren[len(newChildElemChildren)-1]
					if lastTail := strings.TrimSpace(etree.Tail(lastSubChild)); lastTail == "" {
						etree.SetTail(lastSubChild, tail)
					} else {
						etree.SetTail(lastSubChild, lastTail+" "+tail)
					}
				}
			}
		}

		if etree.Text(newChildElem) != "" || len(dom.Children(newChildElem)) > 0 {
			etree.Append(processedElement, newChildElem)
		}

		child.Data = "done"
	}

	element.Data = "done"

	// Test if it has children and text. Avoid double tags??
	if len(dom.Children(processedElement)) > 0 && textCharsTest(etree.IterText(processedElement, "")) {
		return processedElement
	}

	return nil
}

// handleQuotes process quotes elements.
func handleQuotes(element *html.Node, cache *lru.Cache, opts Options) *html.Node {
	processedElement := etree.Element(dom.TagName(element))

	for _, child := range etree.Iter(element) {
		processedChild := processNode(child, cache, opts)
		if processedChild != nil {
			newSub := etree.SubElement(processedElement, dom.TagName(child))
			etree.SetText(newSub, etree.Text(processedChild))
			etree.SetTail(newSub, etree.Tail(processedChild))
		}
		child.Data = "done"
	}

	if len(dom.Children(processedElement)) > 0 && textCharsTest(etree.IterText(processedElement, "")) {
		etree.StripTags(processedElement, "blockquote", "pre", "q")
		return processedElement
	}

	return nil
}

// handleTitles process head elements (titles).
func handleTitles(element *html.Node, cache *lru.Cache, opts Options) *html.Node {
	// In original trafilatura, summary is treated as heading.
	// However, in XML, <h1> to <h6> is treated simply as <head>,
	// which means heading level is not important in XML. Since
	// we work mainly in HTML, we can't simply change the summary
	// into heading because heading level is important here. So,
	// here we just mark the summary as bold to show that it's an
	// important text.
	if dom.TagName(element) == "summary" {
		element.Data = "b"
	}

	var title *html.Node
	if children := dom.Children(element); len(children) == 0 {
		// TODO: maybe needs attention?
		// tail := etree.Tail(element)
		// if tail != "" && rxWords.MatchString(tail) {
		// 	logWarn(opts, "tail in title, stripping: %s", tail)
		// }
		// etree.SetTail(element, "")
		title = processNode(element, cache, opts)
	} else {
		title = dom.Clone(element, false)
		for _, child := range dom.ChildNodes(element) {
			clonedChild := dom.Clone(child, true)
			processedChild := handleTextNode(clonedChild, cache, false, opts)

			if processedChild != nil {
				dom.AppendChild(title, processedChild)
			} else {
				dom.AppendChild(title, clonedChild)
			}

			child.Data = "done"
		}
	}

	if title != nil && textCharsTest(etree.IterText(title, "")) {
		return title
	}

	return nil
}

// handleParagraphs process paragraphs (p) elements along with their children, trim and clean the content.
func handleParagraphs(element *html.Node, potentialTags map[string]struct{}, cache *lru.Cache, opts Options) *html.Node {
	// Clear attributes
	element.Attr = nil

	// Handle paragraph without children
	if len(dom.Children(element)) == 0 {
		return processNode(element, cache, opts)
	}

	// Handle with children
	var unwantedChildren []*html.Node
	processedElements := make(map[*html.Node]struct{})
	for _, child := range dom.GetElementsByTagName(element, "*") {
		childTag := dom.TagName(child)

		// Make sure child is potential element
		if _, exist := potentialTags[childTag]; !exist && childTag != "done" {
			logDebug(opts, "unexpected in p: %s %s %s", childTag, etree.Text(child), etree.Tail(child))
			unwantedChildren = append(unwantedChildren, child)
			continue
		}

		// If necessary remove duplicate child
		if opts.Deduplicate && cache != nil && duplicateTest(child, cache, opts) {
			unwantedChildren = append(unwantedChildren, child)
			continue
		}

		switch childTag {
		case "p": // nested <p>
			logWarn(opts, "extra p within p: %s %s %s", childTag, etree.Text(child), etree.Tail(child))
			etree.SetText(child, " "+etree.Text(child))
			etree.Strip(child)

		case "a": // links
			childHref := trim(dom.GetAttribute(child, "href"))
			childTarget := trim(dom.GetAttribute(child, "target"))
			child.Attr = nil

			if childHref != "" {
				dom.SetAttribute(child, "href", childHref)
			}

			if childTarget != "" {
				dom.SetAttribute(child, "target", childTarget)
			}
		}

		processedElements[child] = struct{}{}
	}

	// Remove unwanted child
	for i := len(unwantedChildren) - 1; i >= 0; i-- {
		etree.Remove(unwantedChildren[i])
	}

	// Remove empty elements. Do it backward, to make sure all children
	// is removed before its parent.
	children := dom.GetElementsByTagName(element, "*")
	for i := len(children) - 1; i >= 0; i-- {
		isEmpty := !textCharsTest(etree.Text(children[i]))
		isVoidElement := dom.IsVoidElement(children[i])
		if isEmpty && !isVoidElement {
			etree.Strip(children[i])
		}
	}

	// Clean trailing line break
	lineBreaks := dom.QuerySelectorAll(element, "br,hr")
	for i := len(lineBreaks) - 1; i >= 0; i-- {
		br := lineBreaks[i]
		if br.NextSibling == nil || etree.Tail(br) == "" {
			etree.Remove(br)
		}
	}

	// Clone the element to return
	processedElement := dom.Clone(element, true)
	etree.SetTail(processedElement, etree.Tail(element))

	// Mark processed elements as done
	for elem := range processedElements {
		elem.Data = "done"
	}

	// Finish
	elementText := etree.Text(processedElement)
	elementChildren := dom.Children(processedElement)
	if len(elementChildren) > 0 || elementText != "" {
		return processedElement
	}

	logDebug(opts, "discarding p-child: %s", trim(etree.ToString(processedElement)))
	return nil
}

// handleFormatting process formatting elements (b, i, etc) found
// outside of paragraphs.
func handleFormatting(element *html.Node, cache *lru.Cache, opts Options) *html.Node {
	formatting := processNode(element, cache, opts)
	if len(dom.Children(element)) == 0 && formatting == nil {
		return nil
	}

	// Repair orphan elements
	parent := element.Parent
	if parent == nil {
		parent = element.PrevSibling
	}

	var processedElement *html.Node
	if parentTag := dom.TagName(parent); parent == nil ||
		(!inMap(parentTag, mapXmlCellTags) &&
			!inMap(parentTag, mapXmlHeadTags) &&
			!inMap(parentTag, mapXmlHiTags) &&
			!inMap(parentTag, mapXmlItemTags) &&
			!inMap(parentTag, mapXmlQuoteTags) &&
			parentTag != "p") {
		processedElement = etree.Element("p")
		etree.Append(processedElement, formatting)
	} else {
		processedElement = formatting
	}

	return processedElement
}

// handleTable process single table element.
func handleTable(tableElement *html.Node, potentialTags map[string]struct{}, cache *lru.Cache, opts Options) *html.Node {
	newTable := etree.Element(("table"))
	newRow := etree.Element("tr")

	// Prepare potential tags with div
	potentialTagsWithDiv := duplicateMap(potentialTags)
	potentialTagsWithDiv["div"] = struct{}{}

	// TODO: we are supposed to strip structural elements here, but I'm not so sure.
	// Check it again later, I guess.
	etree.StripTags(tableElement, "thead", "tbody", "tfoot")

	// Explore sub-elements
	for _, subElement := range dom.GetElementsByTagName(tableElement, "*") {
		subElementTag := dom.TagName(subElement)
		if subElementTag == "tr" {
			if len(dom.Children(newRow)) > 0 {
				etree.Append(newTable, newRow)
				newRow = etree.Element("tr")
			}
		} else if subElementTag == "td" || subElementTag == "th" {
			newChildElem := etree.Element(subElementTag)

			// Process childless element
			if len(dom.Children(subElement)) == 0 {
				processedCell := processNode(subElement, cache, opts)
				if processedCell != nil {
					etree.SetText(newChildElem, etree.Text(processedCell))
					etree.SetTail(newChildElem, etree.Tail(processedCell))
				}
			} else {
				// Proceed with iteration, fix for nested elements
				etree.SetText(newChildElem, etree.Text(subElement))
				etree.SetTail(newChildElem, etree.Tail(subElement))
				subElement.Data = "done"

				for _, child := range dom.GetElementsByTagName(subElement, "*") {
					childTag := dom.TagName(child)

					var processedSubChild *html.Node
					if inMap(childTag, mapXmlCellTags) || inMap(childTag, mapXmlHiTags) {
						processedSubChild = handleTextNode(child, cache, true, opts)
					} else {
						processedSubChild = handleTextElem(child, potentialTagsWithDiv, cache, opts)
					}

					if processedSubChild != nil {
						subChildElement := etree.SubElement(newChildElem, dom.TagName(processedSubChild))
						etree.SetText(subChildElement, etree.Text(processedSubChild))
						etree.SetTail(subChildElement, etree.Tail(processedSubChild))
					}

					child.Data = "done"
				}
			}

			// Add to tree
			if etree.Text(newChildElem) != "" || len(dom.Children(newChildElem)) > 0 {
				dom.AppendChild(newRow, newChildElem)
			}
		} else if subElementTag == "table" {
			// beware of nested tables
			break
		}

		// Clean up
		subElement.Data = "done"
	}

	// End of processing
	if len(dom.Children(newRow)) > 0 {
		etree.Append(newTable, newRow)
	}

	if len(dom.Children(newTable)) > 0 {
		return newTable
	}

	return nil
}

// handleImage process image element.
func handleImage(element *html.Node) *html.Node {
	processedElement := etree.Element(dom.TagName(element))

	// Handle image source
	elementSrc := dom.GetAttribute(element, "src")
	elementDataSrc := dom.GetAttribute(element, "data-src")

	if isImageFile(elementDataSrc) {
		dom.SetAttribute(processedElement, "src", elementDataSrc)
	} else if isImageFile(elementSrc) {
		dom.SetAttribute(processedElement, "src", elementSrc)
	} else {
		// Take the first corresponding attribute
		for _, attr := range element.Attr {
			if strings.HasPrefix(attr.Key, "data-src") && isImageFile(attr.Val) {
				dom.SetAttribute(processedElement, "src", attr.Val)
				break
			}
		}
	}

	// Handle additional data
	if elementAlt := dom.GetAttribute(element, "alt"); elementAlt != "" {
		dom.SetAttribute(processedElement, "alt", elementAlt)
	}

	if elementTitle := dom.GetAttribute(element, "title"); elementTitle != "" {
		dom.SetAttribute(processedElement, "title", elementTitle)
	}

	// If image doesn't have any attributes or doesn't have any src, return nil
	if len(processedElement.Attr) == 0 || dom.GetAttribute(processedElement, "src") == "" {
		return nil
	}

	// Post process the URL
	url := dom.GetAttribute(processedElement, "src")
	if url != "" && strings.HasPrefix(url, "//") {
		url = "http://" + strings.TrimPrefix(url, "//")
		dom.SetAttribute(processedElement, "src", url)
	}

	return processedElement
}

// handleOtherElement handle diverse or unknown elements in the scope of relevant tags.
func handleOtherElement(element *html.Node, potentialTags map[string]struct{}, cache *lru.Cache, opts Options) *html.Node {
	// Delete non potential element
	tagName := dom.TagName(element)
	if _, exist := potentialTags[tagName]; !exist {
		return nil
	}

	// TODO: make a copy and prune it in case it contains sub-elements handled on their own?
	if tagName == "div" || tagName == "details" {
		processedElement := handleTextNode(element, cache, false, opts)
		if processedElement != nil && textCharsTest(etree.Text(processedElement)) {
			processedElement.Attr = nil
			if dom.TagName(processedElement) == "div" {
				processedElement.Data = "p"
			}

			return processedElement
		}
	}

	logDebug(opts, "unexpected element seen: %s %s", tagName, etree.Text(element))
	return nil
}

func recoverWildText(doc, resultBody *html.Node, potentialTags map[string]struct{}, cache *lru.Cache, opts Options) {
	logInfo(opts, "recovering wild text elements")

	var searchList []string
	searchList = append(searchList, listXmlQuoteTags...)
	searchList = append(searchList, "code", "p", "table")

	if opts.FavorRecall {
		potentialTags = duplicateMap(potentialTags)
		potentialTags["div"] = struct{}{}
		for _, t := range listXmlLbTags {
			potentialTags[t] = struct{}{}
		}

		searchList = append(searchList, "div")
		searchList = append(searchList, listXmlLbTags...)
		searchList = append(searchList, listXmlListTags...)
	}

	// Prune
	searchDoc := pruneUnwantedSections(doc, opts)

	// Decide if links are preserved
	if _, exist := potentialTags["a"]; !exist {
		etree.StripTags(searchDoc, "a", "ref", "span")
	} else {
		etree.StripTags(searchDoc, "span")
	}

	var processedElems []*html.Node
	for _, element := range etree.Iter(searchDoc, searchList...) {
		processedElement := handleTextElem(element, potentialTags, cache, opts)
		if processedElement != nil {
			processedElems = append(processedElems, processedElement)
		}
	}

	etree.Extend(resultBody, processedElems...)
}

// compareExtraction decide whether to choose own or external extraction based on a series
// of heuristics. In original Trafilatura, they use python-readability and justext, while
// here we use go-readability and go-domdistiller. Since there are difference in
// implementation between them, here we do it a bit differently compared to the original code.
func compareExtraction(doc, originalExtract *html.Node, opts Options) (*html.Node, string) {
	// Bypass for favor recall
	originalText := trim(etree.IterText(originalExtract, " "))
	lenOriginal := utf8.RuneCountInString(originalText)
	if opts.FavorRecall && lenOriginal > opts.Config.MinExtractedSize*10 {
		return originalExtract, originalText
	}

	// Prepare fallback candidates
	fallbackCandidates := opts.FallbackCandidates

	// Clean doc to be used for Readability and Dom Distiller
	cleanedDoc := dom.Clone(doc, true)

	// If fallback candidates are empty, populate it first
	if len(fallbackCandidates) == 0 {
		fallbackCandidates = []*html.Node{}

		readabilityExtract, err := tryReadability(cleanedDoc, opts)
		if err == nil {
			fallbackCandidates = append(fallbackCandidates, readabilityExtract)
		} else {
			logWarn(opts, "readability failed: %v", err)
		}

		// Here we append nil to fallback candidates. This nil value is used to
		// notify Trafilatura to run Go-DomDistiller for that candidate. We do it
		// this way to make sure that dom-distiller will only be run if readability
		// result is still not good enough to use.
		fallbackCandidates = append(fallbackCandidates, nil)
	}

	// Convert url to string for logging
	var originalUrl string
	if opts.OriginalURL != nil {
		originalUrl = opts.OriginalURL.String()
	}

	// Compare
	for i, candidate := range fallbackCandidates {
		// Use dom-distiller if necessary
		if candidate == nil {
			var err error
			candidate, err = tryDomDistiller(cleanedDoc, opts)
			if err != nil {
				logWarn(opts, "dom-distiller failed: %v", err)
				continue
			}
		}

		// Extract text from candidate
		candidateText := trim(dom.TextContent(candidate))
		lenCandidate := utf8.RuneCountInString(candidateText)
		logInfo(opts, "extracted length: %d (candidate-%d) %d (original)", lenCandidate, i+1, lenOriginal)

		// TODO: This part is pretty different compared to the original.
		// Check if this candidate can be used, either because it pass length check
		// or because we need to favor recall.
		var candidateUsable bool

		switch {
		case lenCandidate == 0 || lenCandidate == lenOriginal:
			candidateUsable = false
		case lenOriginal == 0 && lenCandidate > 0:
			candidateUsable = true
		case lenOriginal > 2*lenCandidate:
			candidateUsable = false
		case lenCandidate > 2*lenOriginal:
			candidateUsable = true
		default: // borderline case
			tables := dom.GetElementsByTagName(doc, "table")
			paragraphs := dom.GetElementsByTagName(doc, "p")
			nTable, nParagraph := len(tables), len(paragraphs)

			var pTextLength int
			for _, p := range paragraphs {
				pText := trim(etree.IterText(p, " "))
				pTextLength += utf8.RuneCountInString(pText)
			}

			if pTextLength == 0 && lenCandidate > opts.Config.MinExtractedSize*2 {
				candidateUsable = true
			} else if nTable > nParagraph && lenCandidate > opts.Config.MinExtractedSize*2 {
				candidateUsable = true
			} else {
				candidateUsable = false
			}
		}

		mustFavorRecall := lenOriginal < opts.Config.MinExtractedSize && opts.FavorRecall
		if candidateUsable || mustFavorRecall {
			originalExtract = candidate
			lenOriginal = lenCandidate
			logInfo(opts, "candidate-%d usable: %s", i+1, originalUrl)
		}

		if lenOriginal >= opts.Config.MinExtractedSize {
			logInfo(opts, "candidate-%d used: %s", i+1, originalUrl)
			break
		}
	}

	// Sanitize the tree
	sanitizeTree(originalExtract, opts)

	// Return data
	finalText := trim(etree.IterText(originalExtract, " "))
	return originalExtract, finalText
}

// baseline uses baseline extraction function targeting text paragraphs and/or JSON metadata.
func baseline(doc *html.Node) (*html.Node, string) {
	postBody := etree.Element("body")
	if doc == nil {
		return postBody, ""
	}

	// Scrape JSON+LD for article body
	for _, script := range dom.QuerySelectorAll(doc, `script[type="application/ld+json"]`) {
		// Get the json text inside the script
		jsonLdText := dom.TextContent(script)
		jsonLdText = strings.TrimSpace(jsonLdText)
		jsonLdText = html.UnescapeString(jsonLdText)
		if jsonLdText == "" {
			continue
		}

		// Decode JSON text, assuming it is an object
		data := map[string]any{}
		err := json.Unmarshal([]byte(jsonLdText), &data)
		if err != nil {
			continue
		}

		// Find article body recursively
		var articleBody string
		var findArticleBody func(obj map[string]any)

		findArticleBody = func(obj map[string]any) {
			for key, value := range obj {
				switch v := value.(type) {
				case string:
					v = trim(v)
					if strings.ToLower(key) == "articlebody" && v != "" {
						articleBody = v
						return
					}

				case map[string]any:
					findArticleBody(v)

				case []any:
					for _, item := range v {
						if obj, isObject := item.(map[string]any); isObject {
							findArticleBody(obj)
						}
					}
				}
			}
		}

		findArticleBody(data)
		if articleBody != "" {
			p := etree.SubElement(postBody, "p")
			etree.SetText(p, articleBody)
			return postBody, articleBody
		}
	}

	// Basic tree cleaning
	discardedElements := dom.QuerySelectorAll(doc, "aside,footer,script,style")
	for i := len(discardedElements) - 1; i >= 0; i-- {
		discardedElements[i].Parent.RemoveChild(discardedElements[i])
	}

	// Scrape from article tag
	articleElement := dom.QuerySelector(doc, "article")
	if articleElement != nil {
		tmpText := trim(dom.TextContent(articleElement))
		if utf8.RuneCountInString(tmpText) > 100 {
			p := etree.SubElement(postBody, "p")
			etree.SetText(p, tmpText)
			return postBody, tmpText
		}
	}

	// Scrape from text paragraphs
	results := make(map[string]struct{})
	for _, element := range etree.Iter(doc, "blockquote", "pre", "q", "code", "p") {
		entry := dom.TextContent(element)
		if _, exist := results[entry]; !exist {
			p := etree.SubElement(postBody, "p")
			etree.SetText(p, entry)
			results[entry] = struct{}{}
		}
	}

	tmpText := trim(etree.IterText(postBody, "\n"))
	if utf8.RuneCountInString(tmpText) > 100 {
		return postBody, tmpText
	}

	// Default strategy: clean the tree and take everything
	if body := dom.QuerySelector(doc, "body"); body != nil {
		text := trim(etree.IterText(body, "\n"))
		if utf8.RuneCountInString(text) > 100 {
			elem := etree.SubElement(postBody, "p")
			etree.SetText(elem, text)
			return postBody, text
		}
	}

	// New fallback
	text := trim(dom.TextContent(doc))
	elem := etree.SubElement(postBody, "p")
	etree.SetText(elem, text)
	return postBody, text
}

func pruneUnwantedSections(subTree *html.Node, opts Options) *html.Node {
	// Prune the rest
	subTree = pruneUnwantedNodes(subTree, OverallDiscardedContentXpaths, true)
	subTree = pruneUnwantedNodes(subTree, DiscardedPaywallXpaths)

	// Prune images
	if !opts.IncludeImages {
		subTree = pruneUnwantedNodes(subTree, DiscardedImageXpaths)
	}

	// Balance precision / recall
	if !opts.FavorRecall {
		subTree = pruneUnwantedNodes(subTree, DiscardedTeaserXpaths)
		if opts.FavorPrecision {
			subTree = pruneUnwantedNodes(subTree, PrecisionDiscardedContentXpaths)
		}
	}

	// Remove elements by link density
	deleteByLinkDensity(subTree, opts, true, "div")
	deleteByLinkDensity(subTree, opts, true, listXmlListTags...)
	deleteByLinkDensity(subTree, opts, false, "p")

	// Also filter fw/head, table and quote elements?
	if opts.FavorPrecision {
		// Delete trailing titles
		children := dom.Children(subTree)
		for i := len(children) - 1; i >= 0; i-- {
			if inMap(dom.TagName(children[i]), mapXmlHeadTags) {
				children[i].Parent.RemoveChild(children[i])
				continue
			}
			break
		}

		deleteByLinkDensity(subTree, opts, false, listXmlHeadTags...)
		deleteByLinkDensity(subTree, opts, false, listXmlQuoteTags...)
	}

	return subTree
}
