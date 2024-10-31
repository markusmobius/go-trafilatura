package trafilatura

import (
	"strings"
	"unicode/utf8"

	"github.com/go-shiori/dom"
	"github.com/markusmobius/go-trafilatura/internal/etree"
	"github.com/markusmobius/go-trafilatura/internal/lru"
	"github.com/markusmobius/go-trafilatura/internal/selector"
	"golang.org/x/net/html"
)

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
			processedChild := handleTextNode(clonedChild, cache, false, false, opts)

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

// addSubElement adds a sub-element to an existing child element.
func addSubElement(newChildElement, subElement, processedSubChild *html.Node) *html.Node {
	subChildElement := etree.SubElement(newChildElement, dom.TagName(processedSubChild))
	etree.SetText(subChildElement, etree.Text(processedSubChild))
	etree.SetTail(subChildElement, etree.Tail(processedSubChild))
	subChildElement.Attr = append(subChildElement.Attr, subElement.Attr...)
	return subChildElement
}

// processNestedElement iterates through an element child and rewire its descendants.
func processNestedElement(child, newChildElement *html.Node, cache *lru.Cache, opts Options) {
	etree.SetText(newChildElement, etree.Text(child))
	for _, subElement := range etree.IterDescendants(child) {
		if inMap(dom.TagName(subElement), mapXmlListTags) {
			processedSubChild := handleLists(subElement, cache, opts)
			if processedSubChild != nil {
				dom.AppendChild(newChildElement, processedSubChild)
			}
		} else {
			processedSubChild := handleTextNode(subElement, cache, false, false, opts)
			if processedSubChild != nil {
				addSubElement(newChildElement, subElement, processedSubChild)
			}
		}
		subElement.Data = "done"
	}
}

// isTextElement checks if the element contains text.
func isTextElement(element *html.Node) bool {
	return element != nil && textCharsTest(etree.IterText(element, ""))
}

// defineNewElement creates a new sub-element if necessary.
func defineNewElement(processedElement, originalElement *html.Node) {
	if processedElement != nil {
		childElement := etree.SubElement(originalElement, dom.TagName(processedElement))
		etree.SetText(childElement, etree.Text(processedElement))
		etree.SetTail(childElement, etree.Tail(processedElement))
	}
}

// handleLists process lists elements including their descendants.
func handleLists(element *html.Node, cache *lru.Cache, opts Options) *html.Node {
	var newChildElem *html.Node
	processedElement := etree.Element(dom.TagName(element))

	if text := strings.TrimSpace(etree.Text(element)); text != "" {
		newChildElem = etree.SubElement(processedElement, "li")
		etree.SetText(newChildElem, text)
	}

	for _, child := range etree.IterDescendants(element, listXmlItemTags...) {
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
			processNestedElement(child, newChildElem, cache, opts)

			if tail := etree.Tail(child); strings.TrimSpace(tail) != "" {
				var newChildElemChildren []*html.Node
				for _, el := range dom.Children(newChildElem) {
					if dom.TagName(el) != "done" {
						newChildElemChildren = append(newChildElemChildren, el)
					}
				}

				if nNewChildElemChildren := len(newChildElemChildren); nNewChildElemChildren > 0 {
					lastSubChild := newChildElemChildren[nNewChildElemChildren-1]
					if lastTail := etree.Tail(lastSubChild); strings.TrimSpace(lastTail) == "" {
						etree.SetTail(lastSubChild, etree.Tail(child))
					} else {
						newTail := lastTail + " " + etree.Tail(child)
						etree.SetTail(lastSubChild, newTail)
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
	if isTextElement(processedElement) {
		return processedElement
	}

	return nil
}

// isCodeBlockElement check if it is a code element according to common structural markers.
func isCodeBlockElement(element *html.Node) bool {
	// Pip
	if dom.GetAttribute(element, "lang") != "" || dom.TagName(element) == "code" {
		return true
	}

	// GitHub
	parent := element.Parent
	if parent != nil && strings.Contains(dom.ClassName(parent), "highlight") {
		return true
	}

	// Highlight.js
	code := dom.QuerySelector(element, "code")
	if code != nil && len(dom.Children(element)) == 1 {
		return true
	}

	return false
}

// handleCodeBlocks turn element into a properly tagged code block.
func handleCodeBlocks(element *html.Node) *html.Node {
	processedElement := dom.Clone(element, true)
	for _, child := range etree.Iter(element) {
		child.Data = "done"
	}

	processedElement.Data = "code"
	for _, child := range etree.Iter(processedElement) {
		child.Attr = nil
	}

	return processedElement
}

// handleQuotes process quotes elements.
func handleQuotes(element *html.Node, cache *lru.Cache, opts Options) *html.Node {
	// Handle code block first
	if isCodeBlockElement(element) {
		return handleCodeBlocks(element)
	}

	processedElement := etree.Element(dom.TagName(element))
	for _, child := range etree.Iter(element) {
		processedChild := processNode(child, cache, opts)
		defineNewElement(processedChild, processedElement)
		child.Data = "done"
	}

	if isTextElement(processedElement) {
		etree.StripTags(processedElement, listXmlQuoteTags...)
		return processedElement
	}

	return nil
}

// handleOtherElements handle diverse or unknown elements in the scope of relevant tags.
func handleOtherElements(element *html.Node, potentialTags map[string]struct{}, cache *lru.Cache, opts Options) *html.Node {
	// Handle W3Schools Code
	tagName := dom.TagName(element)
	if tagName == "div" && strings.Contains(dom.ClassName(element), "w3-code") {
		return handleCodeBlocks(element)
	}

	// Delete non potential element
	if _, exist := potentialTags[tagName]; !exist {
		if tagName != "done" {
			logDebug(opts, "discarding element: %s %q", tagName, dom.TextContent(element))
		}
		return nil
	}

	// TODO: make a copy and prune it in case it contains sub-elements handled on their own?
	if tagName == "div" || tagName == "details" {
		processedElement := handleTextNode(element, cache, false, true, opts)
		if processedElement != nil && textCharsTest(etree.Text(processedElement)) {
			processedElement.Attr = nil
			if dom.TagName(processedElement) == "div" {
				processedElement.Data = "p"
			}

			return processedElement
		}
	}

	logDebug(opts, "unexpected element seen: %s %q", tagName, etree.Text(element))
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
			logDebug(opts, "unexpected in p: %s %q %q", childTag, etree.Text(child), etree.Tail(child))
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
			logWarn(opts, "extra p within p: %s %q %q", childTag, etree.Text(child), etree.Tail(child))
			childText := etree.Text(child)
			parentText := etree.Text(child.Parent)
			if parentText != "" && childText != "" {
				etree.SetText(child, " "+etree.Text(child))
			}
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
	for _, subElement := range etree.IterDescendants(tableElement) {
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

				for _, child := range etree.IterDescendants(subElement) {
					childTag := dom.TagName(child)

					var processedSubChild *html.Node
					if inMap(childTag, mapXmlCellTags) || inMap(childTag, mapXmlHiTags) {
						processedSubChild = handleTextNode(child, cache, true, false, opts)
					} else if inMap(childTag, mapXmlListTags) && opts.Focus == FavorRecall {
						processedSubChild = handleLists(child, cache, opts)
						if processedSubChild != nil {
							etree.Append(newChildElem, dom.Clone(processedSubChild, true))
							processedSubChild = nil
						}
					} else {
						processedSubChild = handleTextElem(child, potentialTagsWithDiv, cache, opts)
					}

					// Add child element to processed_element
					defineNewElement(processedSubChild, newChildElem)
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

// handleImage process image element and their relevant attributes.
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
			if processedLb := processNode(element, cache, opts); processedLb != nil {
				newElement := etree.Element("p")
				etree.SetText(newElement, etree.Tail(processedLb))
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

	return handleOtherElements(element, potentialTags, cache, opts)
}

// recoverWildText Look for all previously unconsidered wild elements, including
// outside of the determined frame and throughout the document to recover potentially
// missing text parts.
func recoverWildText(doc, resultBody *html.Node, potentialTags map[string]struct{}, cache *lru.Cache, opts Options) {
	logInfo(opts, "recovering wild text elements")

	var selectorList []string
	selectorList = append(selectorList, listXmlQuoteTags...)
	selectorList = append(selectorList, "code", "p", "table", `div[class*="w3-code"]`)

	if opts.Focus == FavorRecall {
		potentialTags = duplicateMap(potentialTags)
		potentialTags["div"] = struct{}{}
		for _, t := range listXmlLbTags {
			potentialTags[t] = struct{}{}
		}

		selectorList = append(selectorList, "div")
		selectorList = append(selectorList, listXmlLbTags...)
		selectorList = append(selectorList, listXmlListTags...)
	}

	// Prune
	searchDoc := pruneUnwantedSections(doc, potentialTags, opts)

	// Decide if links are preserved
	if _, exist := potentialTags["a"]; !exist {
		etree.StripTags(searchDoc, "a", "ref", "span")
	} else {
		etree.StripTags(searchDoc, "span")
	}

	var processedElems []*html.Node
	selectors := strings.Join(selectorList, ", ")
	for _, element := range dom.QuerySelectorAll(searchDoc, selectors) {
		processedElement := handleTextElem(element, potentialTags, cache, opts)
		if processedElement != nil {
			processedElems = append(processedElems, processedElement)
		}
	}

	etree.Extend(resultBody, processedElems...)
}

// pruneUnwantedSections is rule-based deletion of targeted document sections.
func pruneUnwantedSections(subTree *html.Node, potentialTags map[string]struct{}, opts Options) *html.Node {
	// Prune the rest
	subTree = pruneUnwantedNodes(subTree, selector.OverallDiscardedContent, true)

	// Prune images
	if !opts.IncludeImages {
		subTree = pruneUnwantedNodes(subTree, selector.DiscardedImage)
	}

	// Balance precision / recall
	if opts.Focus != FavorRecall {
		subTree = pruneUnwantedNodes(subTree, selector.DiscardedTeaser)
		if opts.Focus == FavorPrecision {
			subTree = pruneUnwantedNodes(subTree, selector.PrecisionDiscardedContent)
		}
	}

	// Remove elements by link density, several passes
	for i := 0; i < 2; i++ {
		deleteByLinkDensity(subTree, opts, true, "div")
		deleteByLinkDensity(subTree, opts, false, listXmlListTags...)
		deleteByLinkDensity(subTree, opts, false, "p")
	}

	// Remove tables by link density
	if _, potential := potentialTags["table"]; potential || opts.Focus == FavorPrecision {
		tables := etree.Iter(subTree, "table")
		for i := len(tables) - 1; i >= 0; i-- {
			if linkDensityTestTables(tables[i], opts) {
				etree.Remove(tables[i])
			}
		}
	}

	// Also filter fw/head, table and quote elements?
	if opts.Focus == FavorPrecision {
		// Delete trailing titles
		children := dom.Children(subTree)
		for i := len(children) - 1; i >= 0; i-- {
			if inMap(dom.TagName(children[i]), mapXmlHeadTags) {
				etree.Remove(children[i])
				continue
			}
			break
		}

		deleteByLinkDensity(subTree, opts, false, listXmlHeadTags...)
		deleteByLinkDensity(subTree, opts, false, listXmlQuoteTags...)
	}

	return subTree
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
	for _, query := range selector.Content {
		// Capture first node that matched with the rule
		subTree := selector.Query(doc, query)

		// If no nodes matched, try next selector rule
		if subTree == nil {
			continue
		}

		// Prune the subtree
		subTree = pruneUnwantedSections(subTree, potentialTags, opts)
		// TODO: second pass?
		// deleteByLinkDensity(subTree, opts, false, listXmlListTags...)

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
		if opts.Focus == FavorPrecision {
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
		resultBody = dom.CreateElement("body")
		recoverWildText(backupDoc, resultBody, potentialTags, cache, opts)
		tmpText = trim(etree.IterText(resultBody, " "))
	}

	// Filter output
	etree.StripElements(resultBody, false, "done")
	etree.StripTags(resultBody, "div")

	return resultBody, tmpText
}

// processCommentsNode process and determine how to deal with comment's content.
func processCommentsNode(elem *html.Node, potentialTags map[string]struct{}, cache *lru.Cache, opts Options) *html.Node {
	// Make sure node is one of the potential tags
	if _, isPotential := potentialTags[dom.TagName(elem)]; !isPotential {
		return nil
	}

	// Make sure node is not empty and not duplicated
	processedNode := handleTextNode(elem, cache, true, false, opts)
	if processedNode != nil {
		processedNode.Attr = nil
		return processedNode
	}

	return nil
}

// extractComments try and extract comments out of potential sections in the HTML.
func extractComments(doc *html.Node, cache *lru.Cache, opts Options) (*html.Node, string) {
	// Prepare final container
	commentsBody := etree.Element("body")

	// Prepare potential tags
	potentialTags := duplicateMap(tagCatalog)

	// Process each selector rules
	for _, query := range selector.Comments {
		// Capture first node that matched with the rule
		subTree := selector.Query(doc, query)

		// If no nodes matched, try next selector rule
		if subTree == nil {
			continue
		}

		// Prune
		subTree = pruneUnwantedNodes(subTree, selector.DiscardedComments)
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
