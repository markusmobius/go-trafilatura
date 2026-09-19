package trafilatura

import (
	"maps"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/go-shiori/dom"
	"github.com/markusmobius/go-trafilatura/v2/internal/etree"
	"github.com/markusmobius/go-trafilatura/v2/internal/lru"
	"github.com/markusmobius/go-trafilatura/v2/internal/selector"
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
		title = cloneForExtraction(element)
		for _, child := range dom.Children(element) {
			processedChild := handleTextNode(child, cache, false, false, opts)
			if processedChild != nil {
				etree.Append(title, processedChild)
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
	if formatting == nil {
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

// processNestedElement iterates through an element child and rewire its descendants.
func processNestedElement(child, newChildElement *html.Node, cache *lru.Cache, opts Options) {
	etree.SetTextSlot(newChildElement, etree.TextSlot(child))
	for subElement := range etree.MutableDescendants(child) {
		if inMap(dom.TagName(subElement), mapXmlListTags) {
			processedSubChild := handleLists(subElement, cache, opts)
			if processedSubChild != nil {
				dom.AppendChild(newChildElement, processedSubChild)
			}
		} else if inMap(dom.TagName(subElement), inlineCarriedTags) {
			defineNewElement(subElement, newChildElement, true)
		} else {
			processedSubChild := handleTextNode(subElement, cache, false, false, opts)
			if processedSubChild != nil {
				defineNewElement(processedSubChild, newChildElement)
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
func defineNewElement(processedElement, originalElement *html.Node, keepChildren ...bool) {
	if processedElement != nil {
		childElement := etree.SubElement(originalElement, dom.TagName(processedElement))
		etree.SetTextSlot(childElement, etree.TextSlot(processedElement))
		etree.SetTailSlot(childElement, etree.TailSlot(processedElement))
		for _, attr := range processedElement.Attr {
			if strIn(attr.Key, "href", "target", "src", "alt", "title", "lang") {
				childElement.Attr = append(childElement.Attr, attr)
			}
		}
		if len(keepChildren) > 0 && keepChildren[0] {
			for _, child := range dom.Children(processedElement) {
				if inMap(dom.TagName(child), inlineCarriedTags) || inMap(dom.TagName(child), mapXmlLbTags) {
					defineNewElement(child, childElement, true)
					for _, carried := range etree.Iter(child) {
						carried.Data = "done"
					}
				}
			}
		}
	}
}

// handleLists process lists elements including their descendants.
func handleLists(element *html.Node, cache *lru.Cache, opts Options) *html.Node {
	var newChildElem *html.Node
	processedElement := etree.Element(dom.TagName(element))

	if text := etree.Text(element); strings.TrimSpace(text) != "" {
		newChildElem = etree.SubElement(processedElement, "li")
		etree.SetText(newChildElem, text)
	}

	for child := range etree.MutableDescendants(element, listXmlItemTags...) {
		tag := dom.TagName(child)
		if tag == "done" {
			tag = "li"
		}
		newChildElem = dom.CreateElement(tag)

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
	if code != nil && len(dom.Children(element)) == 1 && strings.TrimSpace(etree.Text(element)) == "" && strings.TrimSpace(etree.Tail(code)) == "" {
		return true
	}

	return false
}

// handleCodeBlocks turn element into a properly tagged code block.
func cloneForExtraction(element *html.Node) *html.Node {
	processedElement := dom.Clone(element, true)
	container := etree.Element("div")
	dom.AppendChild(container, processedElement)
	etree.SetTailSlot(processedElement, etree.TailSlot(element))
	return processedElement
}

func handleCodeBlocks(element *html.Node) *html.Node {
	processedElement := cloneForExtraction(element)
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
	etree.SetTextSlot(processedElement, etree.TextSlot(element))
	for child := range etree.MutableDescendants(element) {
		childTag := dom.TagName(child)
		if childTag == "img" {
			defineNewElement(handleImage(child, opts), processedElement)
		} else if childTag == "p" && len(dom.Children(child)) > 0 {
			potentialTags := maps.Clone(tagCatalog)
			potentialTags["a"], potentialTags["img"] = struct{}{}, struct{}{}
			if paragraph := handleParagraphs(child, potentialTags, cache, opts); paragraph != nil {
				etree.Append(processedElement, paragraph)
			}
		} else if inMap(childTag, inlineCarriedTags) {
			defineNewElement(child, processedElement, true)
		} else {
			defineNewElement(processNode(child, cache, opts), processedElement)
		}
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
	element.Attr = nil
	if len(dom.Children(element)) == 0 {
		return processNode(element, cache, opts)
	}

	processedElement := etree.Element(dom.TagName(element))
	for child := range etree.MutableElements(element) {
		childTag := dom.TagName(child)
		if childTag == "done" || !inMap(childTag, potentialTags) {
			continue
		}

		processedChild := handleTextNode(child, cache, false, true, opts)
		if processedChild == nil {
			child.Data = "done"
			continue
		}

		if childTag == "p" {
			if text := etree.Text(processedElement); text != "" {
				etree.SetText(processedElement, text+" "+etree.Text(processedChild))
			} else {
				etree.SetTextSlot(processedElement, etree.TextSlot(processedChild))
			}
		} else if childTag == "img" {
			if image := handleImage(processedChild, opts); image != nil {
				etree.Append(processedElement, image)
			}
		} else {
			formatting := inMap(childTag, mapXmlHiTags) || childTag == "a"
			keepChildren := false
			if formatting && len(dom.Children(processedChild)) > 0 {
				wrapsInline := childTag == "a"
				for _, nested := range dom.Children(processedChild) {
					wrapsInline = wrapsInline || inMap(dom.TagName(nested), inlineCarriedTags)
				}
				keepChildren = wrapsInline
				if !wrapsInline {
					for nested := range etree.MutableChildren(processedChild) {
						if inMap(dom.TagName(nested), mapXmlLbTags) && etree.Tail(nested) != "" {
							etree.SetTail(nested, " "+strings.TrimLeftFunc(etree.Tail(nested), unicode.IsSpace))
						} else if text := etree.Text(nested); textCharsTest(text) {
							etree.SetText(nested, " "+text)
						}
						etree.StripTagsInPlace(processedChild, dom.TagName(nested))
					}
					keepChildren = false
				}
			}
			defineNewElement(processedChild, processedElement, keepChildren)
		}
		child.Data = "done"
	}

	children := dom.Children(processedElement)
	if len(children) > 0 {
		last := children[len(children)-1]
		if inMap(dom.TagName(last), mapXmlLbTags) && !etree.TailSlot(last).Present {
			etree.Remove(last)
		}
		return processedElement
	}
	if etree.Text(processedElement) != "" {
		return processedElement
	}
	return nil
}

const maxTableSpan = 100

func tableSpan(cell *html.Node, attribute string) int {
	value := dom.GetAttribute(cell, attribute)
	if value == "" {
		return 1
	}
	span := 0
	for _, digit := range value {
		digitValue := int(digit - '0')
		if digit < '0' || digit > '9' {
			digitValue = -1
			for _, digitRange := range unicode.Digit.R16 {
				if digit >= rune(digitRange.Lo) && digit <= rune(digitRange.Hi) && (digit-rune(digitRange.Lo))%rune(digitRange.Stride) == 0 {
					digitValue = int((digit-rune(digitRange.Lo))/rune(digitRange.Stride)) % 10
					break
				}
			}
			for _, digitRange := range unicode.Digit.R32 {
				if digit >= rune(digitRange.Lo) && digit <= rune(digitRange.Hi) && (digit-rune(digitRange.Lo))%rune(digitRange.Stride) == 0 {
					digitValue = int((digit-rune(digitRange.Lo))/rune(digitRange.Stride)) % 10
					break
				}
			}
			if digitValue < 0 {
				return 1
			}
		}
		span = min(span*10+digitValue, maxTableSpan)
	}
	return span
}

func flushRowspanCells(row *html.Node, rowspans map[int]int) {
	for column := len(dom.Children(row)); rowspans[column] > 0; column++ {
		etree.SubElement(row, "td")
		rowspans[column]--
		if rowspans[column] == 0 {
			delete(rowspans, column)
		}
	}
}

func finalizeTableRow(table, row *html.Node, rowspans map[int]int, maxColumns int) {
	flushRowspanCells(row, rowspans)
	for len(dom.Children(row)) < maxColumns {
		etree.SubElement(row, "td")
	}
	for _, cell := range dom.Children(row) {
		if etree.Text(cell) != "" || len(dom.Children(cell)) > 0 {
			etree.Append(table, row)
			return
		}
	}
}

func fillTableCell(target, cell *html.Node, nestedElements map[*html.Node]struct{}, potentialTags map[string]struct{}, cache *lru.Cache, opts Options) {
	if len(dom.Children(cell)) == 0 {
		if processed := processNode(cell, cache, opts); processed != nil {
			etree.SetTextSlot(target, etree.TextSlot(processed))
			etree.SetTailSlot(target, etree.TailSlot(processed))
		}
		return
	}
	etree.SetTextSlot(target, etree.TextSlot(cell))
	etree.SetTailSlot(target, etree.TailSlot(cell))
	cell.Data = "done"
	for child := range etree.MutableDescendants(cell) {
		tag := dom.TagName(child)
		if tag == "done" {
			continue
		}
		if _, nested := nestedElements[child]; nested {
			if tag == "table" && etree.Tail(child) != "" {
				children := dom.Children(target)
				if len(children) > 0 {
					last := children[len(children)-1]
					etree.SetTail(last, etree.Tail(last)+etree.Tail(child))
				} else {
					etree.SetText(target, etree.Text(target)+etree.Tail(child))
				}
			}
			continue
		}
		var processed *html.Node
		if inMap(tag, mapXmlCellTags) || inMap(tag, mapXmlHiTags) || strIn(tag, "a", "del", "s", "strike") {
			processed = handleTextNode(child, cache, false, true, opts)
			if processed == nil && len(dom.Children(child)) > 0 {
				defineNewElement(child, target, true)
				for _, carried := range etree.Iter(child) {
					carried.Data = "done"
				}
				continue
			}
		} else if inMap(tag, mapXmlListTags) && opts.Focus == FavorRecall {
			if list := handleLists(child, cache, opts); list != nil {
				etree.Append(target, list)
			}
			child.Data = "done"
			continue
		} else {
			processed = handleTextElem(child, potentialTags, cache, opts)
		}
		defineNewElement(processed, target, true)
		child.Data = "done"
	}
}

func handleTable(tableElement *html.Node, potentialTags map[string]struct{}, cache *lru.Cache, opts Options) *html.Node {
	newTable := etree.Element("table")
	potentialTagsWithDiv := maps.Clone(potentialTags)
	potentialTagsWithDiv["div"] = struct{}{}
	etree.StripTags(tableElement, "thead", "tbody", "tfoot")

	nestedElements := make(map[*html.Node]struct{})
	for _, nestedTable := range etree.IterDescendants(tableElement, "table") {
		for _, nested := range etree.Iter(nestedTable) {
			nestedElements[nested] = struct{}{}
		}
	}
	maxColumns := 0
	for _, row := range dom.Children(tableElement) {
		if dom.TagName(row) != "tr" {
			continue
		}
		columns := 0
		for _, cell := range dom.Children(row) {
			if inMap(dom.TagName(cell), mapXmlCellTags) {
				columns = min(columns+tableSpan(cell, "colspan"), maxTableSpan)
			}
		}
		maxColumns = max(maxColumns, columns)
	}
	for _, caption := range dom.Children(tableElement) {
		if dom.TagName(caption) != "caption" {
			continue
		}
		if text := etree.ExtractionText(caption); text != "" {
			row := etree.SubElement(newTable, "tr")
			etree.SetText(etree.SubElement(row, "th"), text)
			for len(dom.Children(row)) < maxColumns {
				etree.SubElement(row, "td")
			}
		}
		caption.Data = "done"
	}

	headerEmitted, rowHasHeader := false, false
	newRow := etree.Element("tr")
	rowspans := make(map[int]int)
	for _, element := range dom.Children(tableElement) {
		var cells []*html.Node
		if dom.TagName(element) == "tr" {
			if len(dom.Children(newRow)) > 0 {
				finalizeTableRow(newTable, newRow, rowspans, maxColumns)
				headerEmitted = headerEmitted || rowHasHeader
			}
			newRow = etree.Element("tr")
			rowHasHeader = false
			flushRowspanCells(newRow, rowspans)
			cells = dom.Children(element)
		} else if inMap(dom.TagName(element), mapXmlCellTags) {
			cells = []*html.Node{element}
		} else {
			if dom.TagName(element) != "table" {
				element.Data = "done"
			}
			continue
		}
		for _, cell := range cells {
			if !inMap(dom.TagName(cell), mapXmlCellTags) {
				continue
			}
			isHeader := dom.TagName(cell) == "th" && !headerEmitted
			rowHasHeader = rowHasHeader || isHeader
			flushRowspanCells(newRow, rowspans)
			colspan, rowspan := tableSpan(cell, "colspan"), tableSpan(cell, "rowspan")
			if rowspan > 1 {
				column := len(dom.Children(newRow))
				for offset := range colspan {
					rowspans[column+offset] = rowspan - 1
				}
			}
			tag := "td"
			if isHeader {
				tag = "th"
			}
			newCell := etree.SubElement(newRow, tag)
			fillTableCell(newCell, cell, nestedElements, potentialTagsWithDiv, cache, opts)
			for offset := 1; offset < colspan; offset++ {
				etree.SubElement(newRow, tag)
			}
			cell.Data = "done"
		}
		element.Data = "done"
	}
	finalizeTableRow(newTable, newRow, rowspans, maxColumns)
	if len(dom.Children(newTable)) > 0 {
		return newTable
	}
	return nil
}

// handleImage process image element and their relevant attributes.
func handleImage(element *html.Node, options ...Options) *html.Node {
	if element == nil {
		return nil
	}

	processedElement := etree.SubElement(etree.Element("div"), dom.TagName(element))

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
	if len(options) > 0 && options[0].OriginalURL != nil {
		if resolved, err := options[0].OriginalURL.Parse(url); err == nil {
			url = resolved.String()
		}
	} else if strings.HasPrefix(url, "//") {
		url = "http://" + strings.TrimPrefix(url, "//")
	}
	dom.SetAttribute(processedElement, "src", url)
	etree.SetTailSlot(processedElement, etree.TailSlot(element))

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
	} else if inMap(tagName, mapXmlHiTags) || inMap(tagName, mapXmlRefTags) || strIn(tagName, "span", "del", "s", "strike") {
		return handleFormatting(element, cache, opts)
	} else if tagName == "table" {
		if _, exist := potentialTags["table"]; exist {
			return handleTable(element, potentialTags, cache, opts)
		}
	} else if inMap(tagName, mapXmlGraphicTags) {
		if _, exist := potentialTags["img"]; exist {
			return handleImage(element, opts)
		}
	}

	return handleOtherElements(element, potentialTags, cache, opts)
}

// recoverWildText Look for all previously unconsidered wild elements, including
// outside of the determined frame and throughout the document to recover potentially
// missing text parts.
func recoverWildText(doc, resultBody *html.Node, potentialTags map[string]struct{}, cache *lru.Cache, opts Options) {
	logInfo(opts, "recovering wild text elements")
	if potentialTags == nil {
		potentialTags = tagCatalog
	}
	potentialTags = maps.Clone(potentialTags)

	var selectorList []string
	selectorList = append(selectorList, listXmlQuoteTags...)
	selectorList = append(selectorList, "code", "p", "table", `div[class*="w3-code"]`)

	if opts.Focus == FavorRecall {
		potentialTags["div"] = struct{}{}
		for _, t := range listXmlLbTags {
			potentialTags[t] = struct{}{}
		}

		selectorList = append(selectorList, "div")
		selectorList = append(selectorList, listXmlLbTags...)
		selectorList = append(selectorList, listXmlListTags...)
	}

	// Prune
	searchDoc := pruneUnwantedSections(doc, potentialTags, opts, !opts.EnableFallback)

	// Decide if links are preserved
	if _, exist := potentialTags["a"]; !exist {
		etree.StripTags(searchDoc, "a", "ref", "span")
	} else {
		etree.StripTags(searchDoc, "span")
	}

	var existing strings.Builder
	existingLength := 0
	existingElements := make(map[string]struct{})
	for _, element := range dom.Children(resultBody) {
		text := trim(dom.TextContent(element))
		existingElements[text] = struct{}{}
		if text != "" {
			existing.WriteString(text)
			existing.WriteByte('\n')
			existingLength += utf8.RuneCountInString(text) + 1
		}
	}
	selectors := strings.Join(selectorList, ", ")
	for _, element := range dom.QuerySelectorAll(searchDoc, selectors) {
		processedElement := handleTextElem(element, potentialTags, cache, opts)
		if processedElement == nil {
			continue
		}
		text := trim(dom.TextContent(processedElement))
		_, duplicate := existingElements[text]
		underCap := existingLength <= dedupeScanCap
		if text != "" && (duplicate || (utf8.RuneCountInString(text) > minDuplicateLength && underCap && strings.Contains(existing.String(), text))) {
			continue
		}
		etree.Append(resultBody, processedElement)
		if underCap {
			existing.WriteString(text)
			existing.WriteByte('\n')
			existingLength += utf8.RuneCountInString(text) + 1
		}
		existingElements[text] = struct{}{}
	}
}

// pruneUnwantedSections is rule-based deletion of targeted document sections.
func pruneUnwantedSections(subTree *html.Node, potentialTags map[string]struct{}, opts Options, keepTeasers ...bool) *html.Node {
	// Prune the rest
	subTree = pruneUnwantedNodes(subTree, selector.OverallDiscardedContent, true)

	// Prune images
	if !opts.IncludeImages {
		subTree = pruneUnwantedNodes(subTree, selector.DiscardedImage)
	}

	// Balance precision / recall
	if opts.Focus != FavorRecall {
		if len(keepTeasers) == 0 || !keepTeasers[0] {
			subTree = pruneUnwantedNodes(subTree, selector.DiscardedTeaser)
		}
		if opts.Focus == FavorPrecision {
			subTree = pruneUnwantedNodes(subTree, selector.PrecisionDiscardedContent)
		}
	}

	// Remove elements by link density, several passes
	for range 2 {
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
	potentialTags := maps.Clone(tagCatalog)

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
		selectedTree := subTree
		subTree = pruneUnwantedSections(subTree, potentialTags, opts)
		// TODO: second pass?
		// deleteByLinkDensity(subTree, opts, false, listXmlListTags...)

		// If sub tree now empty, try other selector
		if len(dom.Children(subTree)) == 0 {
			continue
		}

		// Check if there are enough <p> with text
		var paragraphText string
		paragraphRoot := subTree
		if subTree == selectedTree {
			paragraphRoot = doc
		}
		for paragraphRoot.Parent != nil {
			paragraphRoot = paragraphRoot.Parent
		}
		for _, paragraph := range dom.GetElementsByTagName(paragraphRoot, "p") {
			paragraphText += dom.TextContent(paragraph)
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
		textElements := 0
		for _, element := range dom.Children(resultBody) {
			if dom.TagName(element) != "img" {
				textElements++
			}
		}
		if textElements > 1 {
			break
		}
	}

	// Try parsing wild <p> elements if nothing found or text too short
	tmpText := etree.ExtractionText(resultBody)
	tmpTextLength := utf8.RuneCountInString(tmpText)

	if len(dom.Children(resultBody)) == 0 || tmpTextLength < opts.Config.MinExtractedSize {
		recoverWildText(backupDoc, resultBody, potentialTags, cache, opts)
		tmpText = etree.ExtractionText(resultBody)
	}
	previous := ""
	for _, element := range dom.Children(resultBody) {
		current := trim(dom.TextContent(element))
		if current != "" && current == previous && utf8.RuneCountInString(current) > minDuplicateLength {
			etree.Remove(element)
		} else {
			previous = current
		}
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
	potentialTags := maps.Clone(tagCatalog)

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

	tmpComments := etree.ExtractionText(commentsBody)
	if tmpComments != "" {
		return commentsBody, tmpComments
	}

	return nil, ""
}
