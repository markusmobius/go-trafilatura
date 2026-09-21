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
	"maps"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/go-shiori/dom"
	"github.com/markusmobius/go-trafilatura/v2/internal/etree"
	"github.com/markusmobius/go-trafilatura/v2/internal/lru"
	"github.com/markusmobius/go-trafilatura/v2/internal/selector"
	"golang.org/x/net/html"
)

// docCleaning cleans the document by discarding unwanted elements.
// In original it's named `tree_cleaning`.
func docCleaning(doc *html.Node, opts Options) {
	docCleaningMode(doc, opts, false)
}

func prepareCore(doc *html.Node, opts Options) {
	docCleaningMode(doc, opts, true)
	convertTagsMode(doc, opts, true)
}

func docCleaningMode(doc *html.Node, opts Options, core bool) {
	// Determine cleaning strategy
	cleaningList := maps.Clone(tagsToClean)
	strippingList := maps.Clone(tagsToStrip)
	if core {
		cleaningList["noindex"] = struct{}{}
	}

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
		for _, table := range dom.QuerySelectorAll(doc, `table[role="presentation"], table[role="none"]`) {
			table.Data = "div"
		}
	}

	if opts.IncludeImages {
		// Many websites have <img> inside <figure> or <picture> or <source> tag
		delete(cleaningList, "figure")
		delete(cleaningList, "picture")
		delete(cleaningList, "source")
		delete(strippingList, "img")
	}

	// Remove nodes in stripping list but keep its children
	for _, tagName := range slices.Sorted(maps.Keys(strippingList)) {
		etree.StripTags(doc, tagName)
	}

	clean := func() {
		order := slices.Sorted(maps.Keys(cleaningList))
		if core {
			order = strings.Fields("aside embed fencedframe footer form head iframe menu object script applet audio canvas figure map picture svg video area blink button datalist dialog frame frameset fieldset link input ins label legend marquee math menuitem nav noindex noscript optgroup option output param progress rp rt rtc select source style track textarea time use table td th tr")
			for _, tagName := range slices.Sorted(maps.Keys(cleaningList)) {
				if !slices.Contains(order, tagName) {
					order = append(order, tagName)
				}
			}
		}
		for _, tagName := range order {
			if !inMap(tagName, cleaningList) {
				continue
			}
			etree.StripElements(doc, true, tagName)
		}
	}

	// Prevent removal of paragraphs
	if opts.Focus == FavorRecall && len(dom.GetElementsByTagName(doc, "p")) > 0 {
		docBackup := dom.Clone(doc, true)
		clean()

		// If paragraphs is removed, revert to backup
		if len(dom.GetElementsByTagName(doc, "p")) == 0 {
			*doc = *docBackup
		}
	} else {
		// Remove nodes in cleaning list including its children
		clean()
	}

	// Remove HTML comment
	removeHtmlCommentNode(doc)
	if core {
		var empty []*html.Node
		for _, element := range dom.GetElementsByTagName(doc, "*") {
			if inMap(element.Data, emptyTagsToRemove) && element.FirstChild == nil {
				empty = append(empty, element)
			}
		}
		for _, element := range empty {
			etree.Remove(element, opts.Focus != FavorPrecision)
		}
	} else {
		pruneHTML(doc, opts)
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

// pruneHTML deletes selected empty elements to save space and processing time.
func pruneHTML(doc *html.Node, opts Options) {
	keepTail := opts.Focus != FavorPrecision
	allElements := dom.GetElementsByTagName(doc, "*")
	for i := len(allElements) - 1; i >= 0; i-- {
		subElement := allElements[i]
		tagName := dom.TagName(subElement)
		if _, exist := emptyTagsToRemove[tagName]; !exist {
			continue
		}

		if len(dom.ChildNodes(subElement)) == 0 {
			etree.Remove(subElement, keepTail)
		}
	}
}

// pruneUnwantedNodes prune the HTML tree by removing unwanted sections.
func pruneUnwantedNodes(tree *html.Node, queries []selector.Rule, withBackup ...bool) *html.Node {
	var oldLen int
	var backup *html.Node
	backupEnabled := len(withBackup) > 0 && withBackup[0]

	if backupEnabled {
		backup = dom.Clone(tree, true)
		oldLen = utf8.RuneCountInString(dom.TextContent(tree))
	}

	for _, query := range queries {
		subElements := selector.QueryAll(tree, query)
		for i := len(subElements) - 1; i >= 0; i-- {
			etree.Remove(subElements[i], true)
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
	// Image element bypass
	tagName := dom.TagName(node)
	if inMap(tagName, mapXmlGraphicTags) && isImageElement(node) {
		return node
	}

	// Make sure text is not empty
	text := etree.Text(node)
	tail := etree.Tail(node)
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
		originalTail := etree.TailSlot(node)
		text, tail = tail, ""

		// Handle differently for br/hr
		if fixComments && inMap(tagName, mapXmlLbTags) {
			node.Data = "p"
		}
		etree.SetTextSlot(node, originalTail)
		etree.SetTailSlot(node, etree.TextValue{Value: "", Present: true})
	}

	// Trim values
	if !preserveSpaces {
		text = trim(text)
		etree.SetTextSlot(node, etree.TextValue{Value: text, Present: text != ""})
		if tail != "" {
			tail = trim(tail)
			etree.SetTailSlot(node, etree.TextValue{Value: tail, Present: tail != ""})
		}
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
	nLinks := len(links)
	if nLinks == 0 {
		return nil, false
	}
	if opts.IncludeImages && dom.QuerySelector(element, "img") != nil {
		return nil, false
	}

	// Get element text
	text := trim(dom.TextContent(element))
	textLength := utf8.RuneCountInString(text)

	// Shortcut
	if nLinks == 1 {
		var threshold float64 = 100
		if opts.Focus == FavorPrecision {
			threshold = 10
		}

		linkText := trim(dom.TextContent(links[0]))
		linkTextLength := utf8.RuneCountInString(linkText)
		if linkTextLength > int(threshold) && float64(linkTextLength) > float64(textLength)*0.9 {
			return nil, true
		}
	}

	// Prepare limit
	var limitLength int
	if dom.TagName(element) == "p" {
		if dom.NextElementSibling(element) == nil {
			limitLength = 60
		} else {
			limitLength = 30
		}
	} else {
		if dom.NextElementSibling(element) == nil {
			limitLength = 300
		} else {
			limitLength = 100
		}
	}

	// Check if text of this node is within limit
	if textLength < limitLength {
		// Collect link info
		linkLength, nShortLinks, nonEmptyLinks := collectLinkInfo(links)
		nNonEmptyLinks := len(nonEmptyLinks)
		if nNonEmptyLinks == 0 {
			return nonEmptyLinks, true
		}

		// Check if links data surpass threshold
		logDebug(opts, "list link text/total: %d/%d", linkLength, textLength)
		logDebug(opts, "short elems/total: %d/%d", nShortLinks, nNonEmptyLinks)

		if float64(linkLength) > float64(textLength)*0.8 ||
			(nNonEmptyLinks > 1 && float64(nShortLinks)/float64(nNonEmptyLinks) > 0.8) {
			return nonEmptyLinks, true
		}
		return nonEmptyLinks, false
	} else if nLinks > 4 {
		linkLength, _, nonEmptyLinks := collectLinkInfo(links)
		if float64(linkLength) > float64(textLength)*0.9 && linkLength < 100*len(nonEmptyLinks) {
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
	if textLength < 200 {
		return false
	}

	// Check link info
	linkLength, _, _ := collectLinkInfo(links)

	logDebug(opts, "table link text: %d / total: %d", linkLength, textLength)

	if textLength < 1000 {
		return float64(linkLength) > float64(textLength)*0.8
	} else {
		return float64(linkLength) > float64(textLength)*0.5
	}
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
		if textLength < 10 {
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
	if tagName == "wbr" && (text != "" || tail != "") {
		element.Data = "span"
	}
	etree.SetTextSlot(element, etree.TextValue{Value: text, Present: text != ""})
	etree.SetTailSlot(element, etree.TextValue{Value: tail, Present: tail != ""})

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
		if len(grandChildren) == 0 && isEmpty && !isVoidElement && !inMap(dom.TagName(child), mapXmlCellTags) {
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

	threshold := 100
	nChildLimit := 3
	if opts.Focus == FavorPrecision {
		threshold = 200
		nChildLimit = 1
	}

	for _, elem := range etree.Iter(subTree, tagNames...) {
		nonEmptyLinks, isHighDensity := linkDensityTest(elem, opts)
		if dom.TagName(elem) == "p" && (inMap(dom.TagName(elem.Parent), mapXmlItemTags) || inMap(dom.TagName(elem.Parent), mapXmlCellTags)) {
			continue
		}

		if isHighDensity {
			nodesToDelete = append(nodesToDelete, elem)
		} else if backtracking && len(nonEmptyLinks) > 0 {
			text := trim(dom.TextContent(elem))
			textLength := utf8.RuneCountInString(text)
			if textLength > 0 && textLength < threshold && len(dom.Children(elem)) >= nChildLimit {
				nodesToDelete = append(nodesToDelete, elem)
			}
		}
	}

	for i := len(nodesToDelete) - 1; i >= 0; i-- {
		etree.Remove(nodesToDelete[i], true)
	}
}

// Simplify HTML markup.
// Here in original Trafilatura we are supposed to convert HTML tags
// into the one that suitable for XML. However, since we prefer the results
// to be HTML, we won't do it here.
func convertTags(tree *html.Node, opts Options) {
	convertTagsMode(tree, opts, false)
}

func convertTagsMode(tree *html.Node, opts Options, core bool) {
	for _, heading := range dom.QuerySelectorAll(tree, `strong[class*="schema-faq-question"]`) {
		heading.Data = "h3"
		heading.Attr = nil
	}
	for _, element := range dom.QuerySelectorAll(tree, "sub, sup") {
		if etree.Text(element) == "" && len(dom.Children(element)) == 0 {
			etree.Remove(element, true)
		}
	}

	// Delete links for faster processing
	if !opts.IncludeLinks {
		// Prepare selector
		cssSelector := "div a, ul a, ol a, dl a, p a"
		if core {
			cssSelector = "div a, li a, p a"
		}
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

	if core {
		for _, element := range dom.GetElementsByTagName(tree, "*") {
			if inMap(dom.TagName(element), mapXmlHiTags) || strIn(dom.TagName(element), "h1", "h2", "h3", "h4", "h5", "h6") {
				element.Attr = nil
			}
		}
		for _, details := range dom.GetElementsByTagName(tree, "details") {
			details.Data = "div"
			for _, summary := range dom.GetElementsByTagName(details, "summary") {
				summary.Data = "h3"
			}
		}
	}

	// Iterate over all concerned elements.
	// In this case we only care about quotes.
	for _, elem := range etree.Iter(tree, listXmlQuoteTags...) {
		if core && dom.TagName(elem) != "pre" {
			continue
		}
		var codeFlag bool

		// Pre with a single span is more likely to be code
		if dom.TagName(elem) == "pre" {
			children := dom.Children(elem)
			if len(children) == 1 && dom.TagName(children[0]) == "span" {
				codeFlag = true
			}
			for _, indicator := range []string{"{", "(\"", "('", "\n    "} {
				if strings.Contains(etree.Text(elem), indicator) {
					codeFlag = true
					break
				}
			}
		}

		// Find hljs elements to detect if it's code
		hljsSelector := `span[class*=" hljs"], span[class^="hljs"]`
		if core {
			hljsSelector = `span[class^="hljs"]`
		}
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

	if opts.IncludeImages && opts.IncludeLinks {
		for _, link := range dom.QuerySelectorAll(tree, "a") {
			if link.Parent == nil {
				continue
			}
			images := dom.QuerySelectorAll(link, "img")
			nextElement := dom.NextElementSibling(link)
			for _, image := range images {
				nodes := append([]*html.Node{image}, etree.TailNodes(image)...)
				for _, node := range nodes {
					node.Parent.RemoveChild(node)
					link.Parent.InsertBefore(node, nextElement)
				}
			}
			if len(images) > 0 && strings.TrimSpace(dom.TextContent(link)) == "" {
				etree.Remove(link, true)
			}
		}
	}
}
