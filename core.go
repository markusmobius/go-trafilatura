package trafilatura

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/go-shiori/dom"
	"golang.org/x/net/html"
)

func Extract(r io.Reader, opts Options) error {
	// Prepare cache for detecting text duplicate
	cache := NewCache(128)

	// Parse HTML
	doc, err := html.Parse(r)
	if err != nil {
		return err
	}

	// Check for the web page language
	if opts.TargetLanguage != "" && !checkHtmlLanguage(doc, opts.TargetLanguage) {
		return fmt.Errorf("web page language is not %s", opts.TargetLanguage)
	}

	// If fallback extractor is enabled, backup the doc first
	var docBackup *html.Node
	if !opts.NoFallback {
		docBackup = dom.Clone(doc, true)
	}
	fmt.Println(docBackup)

	// Extract metadata if necessary
	var metadata Metadata
	if opts.OutputFormat != Text {
		// Fetch metadata
		metadata = extractMetadata(doc, opts.OriginalURL)

		// Stop content extraction if URL included in blacklist
		if metadata.URL != "" && strIn(metadata.URL, opts.URLBlacklist...) {
			return fmt.Errorf("%s is in blacklist", metadata.URL)
		}

		// Check if essential metadata is missing
		if opts.HasEssentialMetadata {
			if metadata.Title == "" {
				return fmt.Errorf("title is required")
			}

			if metadata.URL == "" {
				return fmt.Errorf("url is required")
			}

			// TODO: need to port htmldate
			// if metadata.Date == "" {
			// 	return fmt.Errorf("date is required")
			// }
		}
	}

	// Clean document
	docCleaning(doc, opts.IncludeTables, opts.IncludeImages)

	// Extract comments
	var commentsContainer *html.Node
	if opts.IncludeComments {
		commentsContainer = extractComments(doc, cache, opts.Deduplicate)
	}

	fmt.Println(commentsContainer)
	return nil
}

// extractComments try and extract comments out of potential sections in the HTML.
func extractComments(doc *html.Node, cache *Cache, deduplicate bool) *html.Node {
	// Prepare final container
	commentsContainer := dom.CreateElement("body")

	// Prepare potential tags
	potentialTags := duplicateMap(tagCatalog)

	// Process each selector
	for _, rule := range commentSelectorRules {
		// Capture first node that matched with the rule
		var commentNode *html.Node
		for _, n := range dom.GetElementsByTagName(doc, "*") {
			if rule(n) {
				commentNode = n
				break
			}
		}

		// If no nodes matched, try next selector rule
		if commentNode == nil {
			continue
		}

		// Discard unwanted comments
		var discardedNodes []*html.Node
		for _, n := range dom.GetElementsByTagName(commentNode, "*") {
			for _, discardRule := range discardedCommentSelectorRules {
				if discardRule(n) {
					discardedNodes = append(discardedNodes, n)
					break
				}
			}
		}

		if len(discardedNodes) > 0 {
			removeNodes(discardedNodes)
		}

		// Strip unwanted tags
		unwantedNodes := dom.QuerySelectorAll(commentNode, "a, span")
		if len(unwantedNodes) > 0 {
			stripNodes(unwantedNodes)
		}

		// Extract comments
		for _, childNode := range dom.GetElementsByTagName(commentNode, "*") {
			processedElement := processCommentsNode(childNode, cache, deduplicate, potentialTags)
			if processedElement != nil {
				extendNode(commentsContainer, processedElement)
			}
		}

		// Make sure comment container is not empty
		if len(dom.Children(commentsContainer)) > 0 {
			commentNode.Parent.RemoveChild(commentNode)
			break
		}
	}

	return commentsContainer
}

// extractContent find the main content of a page using a set of selectors, then
// extract relevant elements, strip them of unwanted subparts and convert them.
func extractContent(doc *html.Node, cache *Cache, opts Options) (*html.Node, bool) {
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

	// Process each selector
	for _, rule := range contentSelectorRules {
		// Capture first node that matched with the rule
		var contentNode *html.Node
		for _, n := range dom.GetElementsByTagName(doc, "*") {
			if rule(n) {
				contentNode = n
				break
			}
		}

		// If no nodes matched, try next selector rule
		if contentNode == nil {
			continue
		}

		// Discard unwanted sections
		discardUnwanted(contentNode)

		// Remove elements by link density
		deleteByLinkDensity(contentNode, "div", true)
		deleteByLinkDensity(contentNode, "ul", false)
		deleteByLinkDensity(contentNode, "ol", false)
		deleteByLinkDensity(contentNode, "dl", false)
		deleteByLinkDensity(contentNode, "p", false)

		if _, exist := potentialTags["table"]; exist {
			var tablesToRemove []*html.Node
			for _, table := range iterTags(contentNode, "table") {
				if linkDensityTestTables(table) {
					tablesToRemove = append(tablesToRemove, table)
				}
			}

			if len(tablesToRemove) > 0 {
				removeNodes(tablesToRemove)
			}
		}

		// If content node now empty, try other selector
		if len(dom.Children(contentNode)) == 0 {
			continue
		}

		// Check if there are enough <p> with text
		var paragraphText string
		for _, p := range dom.GetElementsByTagName(contentNode, "p") {
			for _, node := range dom.ChildNodes(p) {
				if node.Type != html.TextNode {
					continue
				}

				text := dom.TextContent(node)
				paragraphText += strings.TrimSpace(text)
			}
		}

		if utf8.RuneCountInString(paragraphText) < minExtractedSize*2 {
			potentialTags["div"] = struct{}{}
		}

		if _, exist := potentialTags["a"]; !exist {
			stripNodes(dom.GetElementsByTagName(contentNode, "a"))
		}

		if _, exist := potentialTags["span"]; !exist {
			stripNodes(dom.GetElementsByTagName(contentNode, "span"))
		}

		// Populate result container
		for _, e := range dom.GetElementsByTagName(contentNode, "*") {
			processed := handleTextElem(e, potentialTags, cache, opts.Deduplicate)
			if processed != nil {
				extendNode(resultBody, processed)
			}
		}

		// Remove trailing titles
		finalChildren := dom.Children(resultBody)
		if len(finalChildren) > 0 {
			lastElement := finalChildren[len(finalChildren)-1]
			switch dom.TagName(lastElement) {
			case "h1", "h2", "h3", "h4", "h5", "h6":
				lastElement.Parent.RemoveChild(lastElement)
			}
		}

		// Exit the loop if the result has children
		if len(dom.Children(resultBody)) > 1 {
			break
		}
	}

	// Try parsing wild <p> elements if nothing found or text too short
	var sureThing bool
	tmpText := strNormalize(dom.TextContent(resultBody))

	if len(dom.Children(resultBody)) > 0 || utf8.RuneCountInString(tmpText) < minExtractedSize {
		recoverWildText(doc, resultBody, potentialTags, cache, opts.Deduplicate)
		tmpText = strNormalize(dom.TextContent(resultBody))
	} else {
		sureThing = true
	}

	// Filter output
	removeNodes(dom.GetElementsByTagName(resultBody, "done"))
	stripNodes(dom.GetElementsByTagName(resultBody, "div"))

	return resultBody, sureThing
}

func processCommentsNode(n *html.Node, cache *Cache, deduplicate bool, potentialTags map[string]struct{}) *html.Node {
	// Make sure node is one of the potential tags
	if _, isPotential := potentialTags[dom.TagName(n)]; !isPotential {
		return nil
	}

	// Make sure node is not empty and not duplicated
	processedNode := handleTextNode(n, deduplicate, cache)
	if processedNode != nil {
		processedNode.Attr = nil
		return processedNode
	}

	return nil
}

func deleteByLinkDensity(node *html.Node, tagName string, backtracking bool) {
	var nodesToDelete []*html.Node
	textNodes := make(map[string][]*html.Node)

	for _, n := range iterTags(node, tagName) {
		nonEmptyLinks, isHighDensity := linkDensityTest(n)

		if isHighDensity {
			nodesToDelete = append(nodesToDelete, n)
			continue
		}

		if backtracking && len(nonEmptyLinks) > 0 {
			text := dom.TextContent(n)
			text = strNormalize(text)
			if _, exist := textNodes[text]; !exist {
				textNodes[text] = []*html.Node{n}
			} else {
				textNodes[text] = append(textNodes[text], n)
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

	if len(nodesToDelete) > 0 {
		removeNodes(nodesToDelete)
	}
}

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

func handleLists(element *html.Node, cache *Cache, deduplicate bool) *html.Node {
	processedElement := dom.CreateElement(dom.TagName(element))

	text := dom.TextContent(element)
	if text != "" {
		dom.SetTextContent(processedElement, text)
	}

	for _, child := range iterTags(element, "dd", "dt", "li") {
		newChild := dom.CreateElement(dom.TagName(child))

		if len(dom.Children(child)) == 0 {
			processedChild := processNode(child, cache, deduplicate)
			if processedChild != nil {
				dom.SetTextContent(newChild, dom.TextContent(processedChild))
				dom.AppendChild(processedElement, newChild)
			}
		} else {
			for _, subElement := range iterTags(child) {
				processedSubChild := handleTextNode(subElement, deduplicate, cache)
				if processedSubChild != nil {
					subChildElement := dom.CreateElement(dom.TagName(processedSubChild))
					dom.SetTextContent(subChildElement, dom.TextContent(processedSubChild))
					dom.AppendChild(newChild, subChildElement)
				}

				if subElement.Type == html.ElementNode {
					subElement.Data = "done"
				}
			}

			stripNodes(dom.GetElementsByTagName(newChild, "dd"))
			stripNodes(dom.GetElementsByTagName(newChild, "dt"))
			stripNodes(dom.GetElementsByTagName(newChild, "li"))
		}

		text := dom.TextContent(newChild)
		text = strings.TrimSpace(text)
		if text != "" || len(dom.Children(newChild)) > 0 {
			dom.AppendChild(processedElement, newChild)
		}

		child.Data = "done"
	}

	if len(dom.Children(processedElement)) > 0 && textCharsTest(dom.TextContent(processedElement)) {
		return processedElement
	}

	return nil
}

func handleQuotes(element *html.Node, cache *Cache, deduplicate bool) *html.Node {
	processedElement := dom.CreateElement(dom.TagName(element))

	for _, child := range iterTags(element) {
		processedChild := processNode(child, cache, deduplicate)
		if processedChild != nil {
			newChild := dom.CreateElement(dom.TagName(child))
			dom.SetTextContent(newChild, dom.TextContent(processedChild))
			dom.AppendChild(processedElement, newChild)
		}
		child.Data = "done"
	}

	if len(dom.Children(processedElement)) > 0 && textCharsTest(dom.TextContent(processedElement)) {
		stripNodes(dom.GetElementsByTagName(processedElement, "blockquote"))
		stripNodes(dom.GetElementsByTagName(processedElement, "pre"))
		stripNodes(dom.GetElementsByTagName(processedElement, "q"))
		stripNodes(dom.GetElementsByTagName(processedElement, "code"))
		return processedElement
	}

	return nil
}

func handleTitles(element *html.Node, cache *Cache, deduplicate bool) *html.Node {
	title := processNode(element, cache, deduplicate)
	if title != nil && textCharsTest(dom.TextContent(title)) {
		return title
	}

	return nil
}

func handleParagraphs(element *html.Node, potentialTags map[string]struct{}, cache *Cache, deduplicate bool) *html.Node {
	// Clear element attribute
	element.Attr = nil

	// Handle paragraph without children
	if len(dom.Children(element)) == 0 {
		return processNode(element, cache, deduplicate)
	}

	// Handle with children
	processedElement := dom.Clone(element, false)

	// Fix nested <p>
	for _, p := range dom.GetElementsByTagName(element, "p") {
		text := strings.TrimSpace(dom.TextContent(p))
		if text != "" && p.PrevSibling != nil {
			if p.PrevSibling.Type == html.TextNode && !strings.HasSuffix(p.PrevSibling.Data, " ") {
				p.PrevSibling.Data += " "
			} else {
				space := dom.CreateTextNode(" ")
				p.Parent.InsertBefore(space, p)
			}
		}

		stripNodes([]*html.Node{p})
	}

	for _, child := range iterTags(element) {
		childTag := dom.TagName(child)
		if _, exist := potentialTags[childTag]; !exist {
			continue
		}

		processedChild := handleTextNode(child, deduplicate, cache)
		if processedChild != nil {
			if childTag == "p" {
				childText := dom.TextContent(child)
				childText = strings.TrimSpace(childText)
				processedElementText := dom.TextContent(processedElement)
				processedElementText = strings.TrimSpace(processedElementText)

				if processedElementText != "" {
					processedElementText += " " + childText
				} else {
					processedElementText = childText
				}

				dom.SetTextContent(processedElement, processedElementText)
				continue
			}

			newSub := dom.CreateElement(childTag)

			// Handle formatting
			if strIn(childTag, "em", "i", "b", "strong", "u", "kbd", "samp", "tt", "var", "sub", "sup", "a") {
				children := dom.Children(child)
				if len(children) > 0 {
					for _, item := range children {
						itemTag := dom.TagName(item)
						itemText := dom.TextContent(item)

						if textCharsTest(itemText) {
							dom.SetTextContent(item, " "+itemText)
						}

						stripNodes(dom.GetElementsByTagName(child, itemTag))
					}
				}

				if childTag == "a" {
					dom.SetTextContent(newSub, dom.TextContent(processedChild))

					childTarget := dom.GetAttribute(child, "target")
					childTarget = strings.TrimSpace(childTarget)
					if childTarget != "" {
						dom.SetAttribute(newSub, "target", childTarget)
					}

					childHref := dom.GetAttribute(child, "href")
					childHref = strings.TrimSpace(childHref)
					if childHref != "" {
						dom.SetAttribute(newSub, "href", childHref)
					}
				}
			}

			// Prepare text
			processedChildText := dom.TextContent(processedChild)
			if !textCharsTest(processedChildText) {
				dom.SetTextContent(processedChild, "")
				processedChildText = ""
			}

			// Handle if there are already children
			dom.SetTextContent(newSub, processedChildText)
			dom.AppendChild(processedElement, newSub)
			child.Data = "done"
		}
	}

	// Finish
	processedElementText := dom.TextContent(processedElement)
	processedElementText = strings.TrimSpace(processedElementText)
	processedElementChildren := dom.Children(processedElement)

	if len(processedElementChildren) > 0 || processedElementText != "" {
		// Clean trailing line break
		if len(processedElementChildren) > 0 {
			lastChild := processedElementChildren[len(processedElementChildren)-1]
			switch dom.TagName(lastChild) {
			case "br", "hr":
				lastChild.Parent.RemoveChild(lastChild)
			}
		}

		return processedElement
	}

	return nil
}

func handleFormatting(element *html.Node) *html.Node {
	var processedElement *html.Node

	if text := dom.TextContent(element); text != "" {
		processedChild := dom.CreateElement(dom.TagName(element))
		if textCharsTest(text) {
			dom.SetTextContent(processedChild, strings.TrimSpace(text))
		}

		processedElement = dom.CreateElement("p")
		dom.AppendChild(processedElement, processedChild)
	}

	return processedElement
}

func handleTable(element *html.Node, cache *Cache, deduplicate bool) *html.Node {
	// TODO: I'm not sure we should match the original code here.
	// For now I just make sure the table has no nested table.
	nestedTables := dom.GetElementsByTagName(element, "table")
	if len(nestedTables) > 0 {
		return nil
	}

	return element
}

func handleImage(element *html.Node) *html.Node {
	processedElement := dom.CreateElement(dom.TagName(element))

	// Handle image source
	elementSrc := strings.TrimSpace(dom.GetAttribute(element, "src"))
	elementDataSrc := strings.TrimSpace(dom.GetAttribute(element, "data-src"))

	if isImageFile(elementDataSrc) {
		dom.SetAttribute(processedElement, "src", elementDataSrc)
	} else if isImageFile(elementSrc) {
		dom.SetAttribute(processedElement, "src", elementSrc)
	} else {
		for _, attr := range element.Attr {
			attrVal := strings.TrimSpace(attr.Val)
			if strings.HasPrefix(attr.Key, "data-src") && isImageFile(attrVal) {
				dom.SetAttribute(processedElement, "src", attrVal)
				break
			}
		}
	}

	// Handle additional data
	elementAlt := strings.TrimSpace(dom.GetAttribute(element, "alt"))
	if elementAlt != "" {
		dom.SetAttribute(processedElement, "alt", elementAlt)
	}

	elementTitle := strings.TrimSpace(dom.GetAttribute(element, "title"))
	if elementTitle != "" {
		dom.SetAttribute(processedElement, "title", elementTitle)
	}

	// If image doesn't have any attributes, return nil
	if len(processedElement.Attr) == 0 {
		return nil
	}

	// Post process the URL
	finalSrc := dom.GetAttribute(processedElement, "src")
	if finalSrc != "" && strings.HasPrefix(finalSrc, "//") {
		finalSrc = "http://" + strings.TrimPrefix(finalSrc, "//")
		dom.SetAttribute(processedElement, "src", finalSrc)
	}

	return processedElement
}

func handleOtherElement(element *html.Node, potentialTags map[string]struct{}, cache *Cache, deduplicate bool) *html.Node {
	// Delete non potential element
	tagName := dom.TagName(element)
	if _, exist := potentialTags[tagName]; !exist {
		return nil
	}

	if tagName == "div" {
		processedElement := handleTextNode(element, deduplicate, cache)
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

func recoverWildText(doc *html.Node, contentContainer *html.Node, potentialTags map[string]struct{}, cache *Cache, deduplicate bool) {
	// Prune
	discardUnwanted(doc)

	// Decide if links are preserved
	if _, exist := potentialTags["a"]; !exist {
		stripNodes(dom.GetElementsByTagName(doc, "a"))
		stripNodes(dom.GetElementsByTagName(doc, "span"))
	} else {
		stripNodes(dom.GetElementsByTagName(doc, "span"))
	}

	tagsToProcess := []string{"blockquote", "code", "div", "p", "pre", "q", "table"}
	for _, element := range iterTags(doc, tagsToProcess...) {
		processedElement := handleTextElem(element, potentialTags, cache, deduplicate)
		if processedElement != nil {
			extendNode(contentContainer, processedElement)
		}
	}
}
