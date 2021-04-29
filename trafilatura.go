package trafilatura

import (
	"fmt"
	"io"
	nurl "net/url"
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

	return nil
}
