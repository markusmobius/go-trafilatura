package trafilatura

import (
	"fmt"
	"io"
	"unicode/utf8"

	"github.com/go-shiori/dom"
	"github.com/markusmobius/go-trafilatura/etree"
	"github.com/markusmobius/go-trafilatura/selector"
	"golang.org/x/net/html"
)

func Extract(r io.Reader, opts Options) error {
	// Prepare cache for detecting text duplicate
	cache := NewCache(cacheSize)

	// Parse HTML
	doc, err := html.Parse(r)
	if err != nil {
		return err
	}

	// HTML language check
	if opts.TargetLanguage != "" && !checkHtmlLanguage(doc, opts.TargetLanguage) {
		return fmt.Errorf("web page language is not %s", opts.TargetLanguage)
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

	// TODO: Here in original Trafilatura, we are supposed to convert HTML tags
	// into the one that suitable for XML. However, since we prefer the results
	// to be XML, we won't do it here.

	// Extract comments first, then remove
	var tmpComments string
	var lenComments int
	var commentsBody *html.Node
	doNothing(lenComments, commentsBody)

	if opts.IncludeComments {
		commentsBody, tmpComments = extractComments(doc, cache, opts.Deduplicate)
		lenComments = utf8.RuneCountInString(tmpComments)
	}

	return nil
}

// extractComments try and extract comments out of potential sections in the HTML.
func extractComments(doc *html.Node, cache *Cache, deduplicate bool) (*html.Node, string) {
	// Prepare final container
	commentsBody := etree.Element("body")

	// Prepare potential tags
	potentialTags := duplicateMap(tagCatalog)

	// Process each selector rules
	for _, rule := range selector.CommentsRule {
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

	tmpComments := etree.IterText(commentsBody)
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
