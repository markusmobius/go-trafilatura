package trafilatura

import (
	"fmt"
	"io"
	nurl "net/url"
	"strings"
	"time"

	"github.com/go-shiori/dom"
	"golang.org/x/net/html"
)

// ExtractFormat is enum to specify the format for extraction result.
type ExtractFormat uint8

const (
	Json ExtractFormat = 1 << iota
	Text
	CSV
	HTML
)

// Options is configuration for the extractor.
type Options struct {
	// Optional ID for the extracted metadata.
	RecordID int64

	// Optional time for the extracted metadata.
	ExtractionTime time.Time

	// Original URL of the page.
	OriginalURL *nurl.URL

	// Specify whether to skip alternative extraction using go-readability
	// and go-domdistiller.
	NoFallback bool

	// Specify whether to extract comments along with the main text.
	OutputFormat ExtractFormat

	// Only process web page that uses the specified language (ISO 639-1 format).
	TargetLanguage string

	// Specify whether to extract comments along with the main text.
	IncludeComments bool

	// Take into account information within the HTML <table> element.
	IncludeTables bool

	// Take images into account (experimental).
	IncludeImages bool

	// Keep structural elements related to formatting, which will be present in HTML
	// format and will be converted to markdown in others.
	IncludeFormatting bool

	// Keep links along with their targets (experimental).
	IncludeLinks bool

	// Specify whether to remove duplicate segments and documents.
	Deduplicate bool

	// Only keep documents featuring all essential metadata (date, title, url).
	HasEssentialMetadata bool

	// Discard documents with too many elements.
	MaxTreeSize bool

	// Provide a blacklist of URLs to filter out documents.
	URLBlacklist []string
}

func Extract(r io.Reader, opts Options) error {
	// Prepare cache for detecting text duplicate
	cache := NewCache(128)

	// Parse HTML
	doc, err := html.Parse(r)
	if err != nil {
		return err
	}

	// Check for the web page language
	if opts.TargetLanguage != "" && !checkPageLanguage(doc, opts.TargetLanguage) {
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

	// Clean HTML
	cleanHTML(doc, opts.IncludeTables, opts.IncludeImages)

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
	// Prepare potential tags
	potentialTags := duplicateMap(tagCatalog)

	// Fetch all nodes in document
	nodes := dom.GetElementsByTagName(doc, "*")

	// Process each selector
	for _, rule := range commentSelectorRules {
		// Capture first node that matched with the rule
		var commentNode *html.Node
		for i := 0; i < len(nodes); i++ {
			if rule(nodes[i]) {
				commentNode = nodes[i]
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

		// Process comment node
		var removableNodes []*html.Node
		for _, childNode := range dom.GetElementsByTagName(commentNode, "*") {
			isUseful := commentsNodeFilter(childNode, cache, deduplicate, potentialTags)
			if !isUseful {
				removableNodes = append(removableNodes, childNode)
			} else {
				childNode.Attr = nil
			}
		}

		if len(removableNodes) > 0 {
			removeNodes(removableNodes)
		}

		// If comment node is not empty, separate from its parent then return it
		text := dom.TextContent(commentNode)
		text = strings.TrimSpace(text)
		if text != "" {
			clone := dom.Clone(commentNode, true)
			commentNode.Parent.RemoveChild(commentNode)
			return clone
		}
	}

	return nil
}
