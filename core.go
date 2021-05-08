package trafilatura

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/abadojack/whatlanggo"
	"github.com/go-shiori/dom"
	"github.com/markusmobius/go-trafilatura/etree"
	"github.com/markusmobius/go-trafilatura/selector"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/html"
)

type ExtractResult struct {
	ContentNode  *html.Node
	CommentsNode *html.Node
	ContentText  string
	CommentsText string
	Metadata     Metadata
}

func Extract(r io.Reader, opts Options) (*ExtractResult, error) {
	// Prepare cache for detecting text duplicate
	cache := NewCache(cacheSize)

	// Parse HTML
	doc, err := html.Parse(r)
	if err != nil {
		return nil, err
	}

	// HTML language check
	if opts.TargetLanguage != "" && !checkHtmlLanguage(doc, opts.TargetLanguage) {
		return nil, fmt.Errorf("web page language is not %s", opts.TargetLanguage)
	}

	// If fallback extractor is enabled, backup the doc first
	var docBackup *html.Node
	if !opts.NoFallback {
		docBackup = dom.Clone(doc, true)
	}
	doNothing(docBackup)

	// Extract metadata if necessary
	var metadata Metadata
	if opts.OutputFormat != Text {
		// Fetch metadata
		metadata = extractMetadata(doc, opts.OriginalURL)

		// Stop extraction if URL is in blacklist
		if metadata.URL != "" && strIn(metadata.URL, opts.URLBlacklist...) {
			return nil, fmt.Errorf("%s is in blacklist", metadata.URL)
		}

		// Check if essential metadata is missing
		if opts.HasEssentialMetadata {
			if metadata.Title == "" {
				return nil, fmt.Errorf("title is required")
			}

			if metadata.URL == "" {
				return nil, fmt.Errorf("url is required")
			}

			// TODO: need to port htmldate
			// if metadata.Date == "" {
			// 	return nil, fmt.Errorf("date is required")
			// }
		}
	}

	// Clean document
	docCleaning(doc, opts.IncludeTables, opts.IncludeImages)

	// TODO: Here in original Trafilatura, we are supposed to convert HTML tags
	// into the one that suitable for XML. However, since we prefer the results
	// to be XML, we won't do it here.

	// Extract comments first, then remove
	var tmpComments string
	var lenComments int
	var commentsBody *html.Node
	doNothing(commentsBody)

	if opts.IncludeComments {
		commentsBody, tmpComments = extractComments(doc, cache, opts.Deduplicate)
		lenComments = utf8.RuneCountInString(tmpComments)
	}

	// Extract content
	postBody, tmpBodyText, sureThing := extractContent(doc, cache, opts)
	doNothing(postBody, tmpBodyText, sureThing)

	// Use fallback if necessary
	if !opts.NoFallback {
		postBody, tmpBodyText = compareExtraction(docBackup, postBody, opts)
		// Add baseline as additional fallback
		if len(dom.Children(postBody)) == 0 {
			postBody, tmpBodyText = baseline(docBackup)
		}
	} else {
		// Rescue: try to use original/dirty tree
		lenText := utf8.RuneCountInString(tmpBodyText)
		if !sureThing && lenText < minExtractedSize {
			postBody, tmpBodyText = baseline(docBackup)
		}
	}

	// Tree size sanity check
	if opts.MaxTreeSize > 0 {
		if len(dom.Children(postBody)) > opts.MaxTreeSize {
			etree.StripTags(postBody, "h1", "h2", "h3", "h4", "h5", "h6")
			if nChildren := len(dom.Children(postBody)); nChildren > opts.MaxTreeSize {
				return nil, fmt.Errorf("output tree to long, discarding file : %d", nChildren)
			}
		}
	}

	// Size checks
	if lenComments < minExtractedCommentSize {
		logrus.Warnf("not enough comments: %s", opts.OriginalURL)
	}

	lenText := utf8.RuneCountInString(tmpBodyText)
	if lenText < minOutputSize && lenComments < minOutputCommentSize {
		return nil, fmt.Errorf("text and comments are not long enough: %d %d", lenText, lenComments)
	}

	// Check duplicates at body level
	if opts.Deduplicate && duplicateTest(postBody, cache) {
		return nil, fmt.Errorf("extracted body has been duplicated")
	}

	// Sanity check on language
	if opts.TargetLanguage != "" {
		lang := getLanguage(tmpBodyText, tmpComments)
		if lang != opts.TargetLanguage {
			return nil, fmt.Errorf("wrong language, want %s got %s", opts.TargetLanguage, lang)
		}
	}

	return &ExtractResult{
		ContentNode:  postBody,
		ContentText:  tmpBodyText,
		CommentsNode: commentsBody,
		CommentsText: tmpComments,
		Metadata:     metadata,
	}, nil
}

// extractComments try and extract comments out of potential sections in the HTML.
func extractComments(doc *html.Node, cache *Cache, deduplicate bool) (*html.Node, string) {
	// Prepare final container
	commentsBody := etree.Element("body")

	// Prepare potential tags
	potentialTags := duplicateMap(tagCatalog)

	// Process each selector rules
	for _, rule := range selector.CommentsRules {
		// Capture first node that matched with the rule
		var subTree *html.Node
		for _, n := range dom.GetElementsByTagName(doc, "*") {
			if rule(n) {
				subTree = n
				break
			}
		}

		// If no nodes matched, try next selector rule
		if subTree == nil {
			continue
		}

		// Prune
		discardUnwantedComments(subTree)
		etree.StripTags(subTree, "a", "span")

		// Extract comments
		var processedElems []*html.Node
		for _, elem := range dom.GetElementsByTagName(subTree, "*") {
			processed := processCommentsNode(elem, potentialTags, cache, deduplicate)
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
	return commentsBody, tmpComments
}

func processCommentsNode(elem *html.Node, potentialTags map[string]struct{}, cache *Cache, deduplicate bool) *html.Node {
	// Make sure node is one of the potential tags
	if _, isPotential := potentialTags[dom.TagName(elem)]; !isPotential {
		return nil
	}

	// Make sure node is not empty and not duplicated
	processedNode := handleTextNode(elem, cache, true, deduplicate)
	if processedNode != nil {
		processedNode.Attr = nil
		return processedNode
	}

	return nil
}

// extractContent find the main content of a page using a set of selectors, then
// extract relevant elements, strip them of unwanted subparts and convert them.
func extractContent(doc *html.Node, cache *Cache, opts Options) (*html.Node, string, bool) {
	var sureThing bool
	resultBody := dom.CreateElement("body")

	// Prepare potential tags
	potentialTags := duplicateMap(tagCatalog)

	if opts.IncludeTables {
		potentialTags["table"] = struct{}{}
	}

	if opts.IncludeImages {
		potentialTags["img"] = struct{}{}
	}

	if opts.IncludeLinks {
		potentialTags["a"] = struct{}{}
	}

	// Iterate each selector rule
	for _, rule := range selector.ContentRules {
		// Capture first node that matched with the rule
		var subTree *html.Node
		for _, n := range dom.GetElementsByTagName(doc, "*") {
			if rule(n) {
				subTree = n
				break
			}
		}

		// If no nodes matched, try next selector rule
		if subTree == nil {
			continue
		}

		// Prune
		discardUnwanted(subTree)

		// Remove elements by link density
		deleteByLinkDensity(subTree, "div", true)
		deleteByLinkDensity(subTree, "ul", false)
		deleteByLinkDensity(subTree, "ol", false)
		deleteByLinkDensity(subTree, "dl", false)
		deleteByLinkDensity(subTree, "p", false)

		// Define iteration strategy
		if _, exist := potentialTags["table"]; exist {
			for _, table := range etree.Iter(subTree, "table") {
				if linkDensityTestTables(table) {
					etree.Remove(table)
				}
			}
		}

		// If sub tree now empty, try other selector
		if len(dom.Children(subTree)) == 0 {
			continue
		}

		// Check if there are enough <p> with text
		var paragraphText string
		for _, p := range dom.GetElementsByTagName(subTree, "p") {
			for _, node := range dom.ChildNodes(p) {
				if node.Type == html.TextNode {
					paragraphText += trim(node.Data)
				}
			}
		}

		if utf8.RuneCountInString(paragraphText) < minExtractedSize*2 {
			potentialTags["div"] = struct{}{}
		}

		if _, exist := potentialTags["a"]; !exist {
			etree.StripTags(subTree, "a")
		}

		if _, exist := potentialTags["span"]; !exist {
			etree.StripTags(subTree, "span")
		}

		// Populate result body
		var processedElems []*html.Node
		for _, elem := range dom.GetElementsByTagName(subTree, "*") {
			processed := handleTextElem(elem, potentialTags, cache, opts.Deduplicate)
			if processed != nil {
				processedElems = append(processedElems, processed)
			}
		}
		etree.Extend(resultBody, processedElems...)

		// Remove trailing titles
		finalChildren := dom.Children(resultBody)
		for i := len(finalChildren) - 1; i >= 0; i-- {
			switch dom.TagName(finalChildren[i]) {
			case "h1", "h2", "h3", "h4", "h5", "h6":
				etree.Remove(finalChildren[i])
			}
		}

		// Exit the loop if the result has children
		if len(dom.Children(resultBody)) > 1 {
			break
		}
	}

	// Try parsing wild <p> elements if nothing found or text too short
	tmpText := trim(etree.IterText(resultBody, " "))
	tmpTextLength := utf8.RuneCountInString(tmpText)

	if len(dom.Children(resultBody)) > 0 || tmpTextLength < minExtractedSize {
		recoverWildText(doc, resultBody, potentialTags, cache, opts.Deduplicate)
		tmpText = trim(etree.IterText(resultBody, " "))
	} else {
		sureThing = true
	}

	// Filter output
	etree.StripElements(resultBody, false, "done")
	etree.StripTags(resultBody, "div")

	return resultBody, tmpText, sureThing
}

// deleteByLinkDensity determines the link density of elements with respect to
// their length, and remove the elements identified as boilerplate.
func deleteByLinkDensity(subTree *html.Node, tagName string, backtracking bool) {
	var nodesToDelete []*html.Node
	textNodes := make(map[string][]*html.Node)

	for _, elem := range etree.Iter(subTree, tagName) {
		nonEmptyLinks, isHighDensity := linkDensityTest(elem)

		if isHighDensity {
			nodesToDelete = append(nodesToDelete, elem)
			continue
		}

		if backtracking && len(nonEmptyLinks) > 0 {
			text := trim(dom.TextContent(elem))
			if _, exist := textNodes[text]; !exist {
				textNodes[text] = []*html.Node{elem}
			} else {
				textNodes[text] = append(textNodes[text], elem)
			}
		}
	}

	if backtracking {
		for text, nodes := range textNodes {
			textLength := utf8.RuneCountInString(text)
			if textLength > 0 && textLength < 1000 && len(nodes) >= 3 {
				nodesToDelete = append(nodesToDelete, nodes...)
			}
		}
	}

	for _, elem := range nodesToDelete {
		etree.Remove(elem)
	}
}

// handleTextElem process text element and determine how to deal with its content.
func handleTextElem(element *html.Node, potentialTags map[string]struct{}, cache *Cache, deduplicate bool) *html.Node {
	switch dom.TagName(element) {
	case "ul", "ol", "dl":
		return handleLists(element, cache, deduplicate)
	case "blockquote", "pre", "q", "code":
		return handleQuotes(element, cache, deduplicate)
	case "h1", "h2", "h3", "h4", "h5", "h6":
		return handleTitles(element, cache, deduplicate)
	case "p":
		return handleParagraphs(element, potentialTags, cache, deduplicate)
	case "br", "hr":
		if textCharsTest(etree.Tail(element)) {
			element = processNode(element, cache, deduplicate)
			if element != nil {
				newElement := etree.Element("p")
				etree.SetText(newElement, etree.Tail(element))
				return newElement
			}
		}
	case "em", "i", "b", "strong", "u", "kbd", "samp", "tt", "var", "sub", "sup", "a", "span":
		return handleFormatting(element)
	case "table":
		if _, exist := potentialTags["table"]; exist {
			return handleTable(element, cache, deduplicate)
		}
	case "img":
		if _, exist := potentialTags["img"]; exist {
			return handleImage(element)
		}
	default:
		return handleOtherElement(element, potentialTags, cache, deduplicate)
	}

	return nil
}

// handleLists process lists elements
func handleLists(element *html.Node, cache *Cache, deduplicate bool) *html.Node {
	processedElement := etree.Element(dom.TagName(element))

	if text := etree.Text(element); text != "" {
		etree.SetText(processedElement, text)
	}

	for _, child := range etree.Iter(element, "dd", "dt", "li") {
		newChild := dom.CreateElement(dom.TagName(child))

		if len(dom.Children(child)) == 0 {
			processedChild := processNode(child, cache, deduplicate)
			if processedChild != nil {
				etree.SetText(newChild, etree.Text(processedChild))
				etree.SetTail(newChild, etree.Tail(processedChild))
				etree.Append(processedElement, newChild)
			}
		} else {
			for _, subElement := range etree.Iter(child) {
				processedSubChild := handleTextNode(subElement, cache, false, deduplicate)
				if processedSubChild != nil {
					subChildElement := etree.SubElement(newChild, dom.TagName(processedSubChild))
					etree.SetText(subChildElement, etree.Text(processedSubChild))
					etree.SetTail(subChildElement, etree.Tail(processedSubChild))
				}

				if subElement.Type == html.ElementNode {
					subElement.Data = "done"
				}
			}

			etree.StripTags(newChild, "dd", "dt", "li")
		}

		if etree.Text(newChild) != "" || len(dom.Children(newChild)) > 0 {
			etree.Append(processedElement, newChild)
		}

		child.Data = "done"
	}

	// Test if it has children and text. Avoid double tags??
	if len(dom.Children(processedElement)) > 0 && textCharsTest(etree.IterText(processedElement, "")) {
		return processedElement
	}

	return nil
}

// handleQuotes process quotes elements.
func handleQuotes(element *html.Node, cache *Cache, deduplicate bool) *html.Node {
	processedElement := etree.Element(dom.TagName(element))

	for _, child := range etree.Iter(element) {
		processedChild := processNode(child, cache, deduplicate)
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
func handleTitles(element *html.Node, cache *Cache, deduplicate bool) *html.Node {
	tail := etree.Tail(element)
	if tail != "" && rxWords.MatchString(tail) {
		logrus.Warnf("tail in title, stripping: %s", tail)
	}

	etree.SetTail(element, "")
	title := processNode(element, cache, deduplicate)
	if title != nil && textCharsTest(etree.Text(element)) {
		return title
	}

	return nil
}

// handleParagraphs process paragraphs (p) elements along with their children, trim and clean the content.
func handleParagraphs(element *html.Node, potentialTags map[string]struct{}, cache *Cache, deduplicate bool) *html.Node {
	// Clear attributes
	element.Attr = nil

	// Handle paragraph without children
	if len(dom.Children(element)) == 0 {
		return processNode(element, cache, deduplicate)
	}

	// Prepare tag maps
	formattingTags := duplicateMap(formatTagCatalog)
	formattingTags["a"] = struct{}{}

	// Handle with children
	processedElement := etree.Element(dom.TagName(element))

	for _, child := range etree.Iter(element) {
		childTag := dom.TagName(child)
		if _, exist := potentialTags[childTag]; !exist {
			continue
		}

		processedChild := handleTextNode(child, cache, false, deduplicate)
		if processedChild != nil {
			// Needing attention for nested <p>
			if childTag == "p" {
				childText := trim(etree.Text(child))
				if elemText := etree.Text(processedElement); elemText != "" {
					etree.SetText(processedElement, elemText+" "+childText)
				} else {
					etree.SetText(processedElement, childText)
				}
				continue
			}

			newSub := etree.Element(dom.TagName(child))

			// Handle formatting
			if _, exist := formattingTags[childTag]; exist {
				if children := dom.Children(child); len(children) > 0 {
					for _, item := range children {
						itemText := etree.Text(item)
						if textCharsTest(itemText) {
							etree.SetText(item, " "+itemText)
						}

						etree.StripTags(child, dom.TagName(item))
					}
				}

				if childTag == "a" {
					etree.SetText(newSub, etree.Text(processedChild))
					etree.SetTail(newSub, etree.Tail(processedChild))

					childTarget := trim(dom.GetAttribute(child, "target"))
					if childTarget != "" {
						dom.SetAttribute(newSub, "target", childTarget)
					}

					childHref := trim(dom.GetAttribute(child, "href"))
					if childHref != "" { // it shouldn't be here
						dom.SetAttribute(newSub, "href", childHref)
					}
				}
			} else if childTag == "br" || childTag == "hr" { // handle line breaks
				if tmp := processNode(child, cache, deduplicate); tmp != nil {
					etree.SetTail(processedChild, etree.Tail(tmp))
				}
			}

			// Prepare text
			processedChildText := etree.Text(processedChild)
			processedChildTail := etree.Tail(processedChild)
			if !textCharsTest(processedChildText) {
				etree.SetText(processedChild, "")
				processedChildText = ""
			}

			// Handle if there are already children
			if len(dom.Children(processedElement)) > 0 {
				if textCharsTest(processedChildTail) {
					etree.SetTail(newSub, processedChildText+processedChildTail)
				} else {
					etree.SetTail(newSub, processedChildText)
				}
			} else {
				etree.SetText(newSub, processedChildText)
				etree.SetTail(newSub, processedChildTail)
			}

			etree.Append(processedElement, newSub)
			child.Data = "done"
		}
	}

	// Finish
	processedElementText := etree.Text(processedElement)
	processedElementChildren := dom.Children(processedElement)

	if len(processedElementChildren) > 0 || processedElementText != "" {
		// Clean trailing line break
		if len(processedElementChildren) > 0 {
			lastChild := processedElementChildren[len(processedElementChildren)-1]
			lastChildTag := dom.TagName(lastChild)
			lastChildTail := etree.Tail(lastChild)

			if (lastChildTag == "br" || lastChildTag == "hr") && lastChildTail == "" {
				etree.Remove(lastChild)
			}
		}

		return processedElement
	}

	return nil
}

// handleFormatting process formatting elements (b, i, etc) found
// outside of paragraphs.
func handleFormatting(element *html.Node) *html.Node {
	var processedElement *html.Node
	text, tail := etree.Text(element), etree.Tail(element)

	if text != "" || tail != "" {
		processedElement = etree.Element("p")
		processedChild := etree.SubElement(processedElement, dom.TagName(element))

		if textCharsTest(text) {
			etree.SetText(processedChild, trim(text))
		}

		if textCharsTest(tail) {
			etree.SetTail(processedChild, trim(tail))
		}
	}

	return processedElement
}

// handleTable process single table element.
func handleTable(tableElement *html.Node, cache *Cache, deduplicate bool) *html.Node {
	newTable := etree.Element(("table"))
	newRow := etree.Element("tr")
	i := 0

	// TODO: we are supposed to strip structural elements here, but I'm not so sure.
	// Check it again later, I guess.
	etree.StripTags(tableElement, "thead", "tbody", "tfoot")

	// Explore sub-elements
	for _, subElement := range etree.Iter(tableElement) {
		i++

		subElementTag := dom.TagName(subElement)
		if subElementTag == "tr" {
			if len(dom.Children(newRow)) > 0 {
				etree.Append(newTable, newRow)
				newRow = etree.Element("tr")
			}
		} else if subElementTag == "td" || subElementTag == "th" {
			processedCell := processNode(subElement, cache, deduplicate)
			if processedCell == nil || !textCharsTest(etree.Text(processedCell)) {
				continue
			}

			newSub := etree.SubElement(newRow, subElementTag)
			etree.SetText(newSub, etree.Text(processedCell))
		} else if subElementTag == "table" && i > 1 {
			// beware of nested tables
			break
		}
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

	// If image doesn't have any attributes, return nil
	if len(processedElement.Attr) == 0 {
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
func handleOtherElement(element *html.Node, potentialTags map[string]struct{}, cache *Cache, deduplicate bool) *html.Node {
	// Delete non potential element
	tagName := dom.TagName(element)
	if _, exist := potentialTags[tagName]; !exist {
		return nil
	}

	if tagName == "div" {
		processedElement := handleTextNode(element, cache, false, deduplicate)
		if processedElement != nil {
			processedElement.Attr = nil
			if dom.TagName(processedElement) == "div" {
				processedElement.Data = "p"
			}

			return processedElement
		}
	}

	return nil
}

func recoverWildText(doc, resultBody *html.Node, potentialTags map[string]struct{}, cache *Cache, deduplicate bool) {
	// Prune
	discardUnwanted(doc)

	// Decide if links are preserved
	if _, exist := potentialTags["a"]; !exist {
		etree.StripTags(doc, "a", "ref", "span")
	} else {
		etree.StripTags(doc, "span")
	}

	var processedElems []*html.Node
	tagsToProcess := []string{"blockquote", "code", "div", "p", "pre", "q", "table"}

	for _, element := range etree.Iter(doc, tagsToProcess...) {
		processedElement := handleTextElem(element, potentialTags, cache, deduplicate)
		if processedElement != nil {
			processedElems = append(processedElems, processedElement)
		}
	}

	etree.Extend(resultBody, processedElems...)
}

// compareExtraction decide whether to choose own or external extraction
// based on a series of heuristics. In original Trafilatura, they use
// python-readability and justext, while here we use go-readability and
// go-domdistiller. Since there are difference in implementation between
// them, here we do it a bit differently compared to the original code.
func compareExtraction(doc, originalExtract *html.Node, opts Options) (*html.Node, string) {
	// Convert url to string
	var originalUrl string
	if opts.OriginalURL != nil {
		originalUrl = opts.OriginalURL.String()
	}

	// Try readability and dom-distiller
	readabilityExtract, err := tryReadability(originalExtract, doc, originalUrl)
	if err != nil {
		logrus.Warnf("readability failed: %v", err)
	}

	distillerExtract, err := tryDomDistiller(originalExtract, doc, originalUrl)
	if err != nil {
		logrus.Warnf("dom-distiller failed: %v", err)
	}

	// Pick the final extract
	var finalExtract *html.Node
	switch {
	case readabilityExtract != nil && distillerExtract == nil:
		finalExtract = readabilityExtract
	case readabilityExtract == nil && distillerExtract != nil:
		finalExtract = distillerExtract
	case readabilityExtract != nil && distillerExtract != nil:
		distillerText := trim(etree.IterText(distillerExtract, " "))
		readabilityText := trim(etree.IterText(readabilityExtract, " "))

		lenDistillerText := utf8.RuneCountInString(distillerText)
		lenReadabilityText := utf8.RuneCountInString(readabilityText)
		if lenReadabilityText >= lenDistillerText {
			finalExtract = readabilityExtract
		} else {
			finalExtract = distillerExtract
		}
	default:
		finalExtract = originalExtract
	}

	// Sanitize the tree
	sanitizeTree(finalExtract, opts)

	// Return data
	finalText := trim(etree.IterText(finalExtract, " "))
	return finalExtract, finalText
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
		if jsonLdText == "" {
			continue
		}

		// Decode JSON text, assuming it is an object
		data := map[string]interface{}{}
		err := json.Unmarshal([]byte(jsonLdText), &data)
		if err != nil {
			continue
		}

		// Find article body recursively
		var articleBody string
		var findArticleBody func(obj map[string]interface{})

		findArticleBody = func(obj map[string]interface{}) {
			for key, value := range obj {
				switch v := value.(type) {
				case string:
					v = trim(v)
					if strings.ToLower(key) == "articlebody" && v != "" {
						articleBody = v
						return
					}

				case map[string]interface{}:
					findArticleBody(v)

				case []interface{}:
					for _, item := range v {
						itemObject, isObject := item.(map[string]interface{})
						if isObject {
							findArticleBody(itemObject)
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

	// Scrape from article tag
	articleElement := dom.QuerySelector(doc, "article")
	if articleElement != nil {
		tmpText := trim(dom.TextContent(articleElement))
		lenText := utf8.RuneCountInString(tmpText)
		if lenText > 0 {
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
	return postBody, tmpText
}

// getLanguage returns the language of the text.
func getLanguage(contentText, commentsText string) string {
	lenContent := utf8.RuneCountInString(contentText)
	lenComments := utf8.RuneCountInString(commentsText)

	var langTest string
	if lenComments > lenContent {
		langTest = commentsText
	} else {
		langTest = contentText
	}

	lang := whatlanggo.DetectLang(langTest)
	return lang.Iso6391()
}
