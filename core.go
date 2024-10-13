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
	"fmt"
	"io"
	nurl "net/url"
	"os"
	"unicode/utf8"

	"github.com/andybalholm/cascadia"
	"github.com/go-shiori/dom"
	"github.com/markusmobius/go-trafilatura/internal/etree"
	"github.com/markusmobius/go-trafilatura/internal/lru"
	"github.com/markusmobius/go-trafilatura/internal/selector"
	"github.com/rs/zerolog"
	"golang.org/x/net/html"
)

var log zerolog.Logger

func init() {
	log = zerolog.New(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: "2006-01-02 15:04",
	}).With().Timestamp().Logger()
}

// ExtractResult is the result of content extraction.
type ExtractResult struct {
	// ContentNode is the extracted content as a `html.Node`.
	ContentNode *html.Node

	// CommentsNode is the extracted comments as a `html.Node`.
	// Will be nil if `ExcludeComments` in `Options` is set to true.
	CommentsNode *html.Node

	// ContentText is the extracted content as a plain text.
	ContentText string

	// CommentsText is the extracted comments as a plain text.
	// Will be empty if `ExcludeComments` in `Options` is set to true.
	CommentsText string

	// Metadata is the extracted metadata which taken from several sources i.e.
	// <meta> tags, JSON+LD and OpenGraph scheme.
	Metadata Metadata
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

	// Prune using selectors that user specified.
	// No backup as this is completely full control of the user.
	if opts.PruneSelector != "" {
		cssSelector, err := cascadia.ParseGroup(opts.PruneSelector)
		if err == nil {
			doc = pruneUnwantedNodes(doc, []selector.Rule{cssSelector.Match})
		}
	}

	// Backup document to make sure the original kept untouched
	doc = dom.Clone(doc, true)
	docBackup1 := dom.Clone(doc, true)
	docBackup2 := dom.Clone(doc, true)

	// Clean document
	docCleaning(doc, opts)
	simplifyTags(doc, opts)

	// Convert HTML tags
	convertTags(doc, opts)

	// Extract comments first, then remove
	var tmpComments string
	var lenComments int
	var commentsBody *html.Node

	if !opts.ExcludeComments { // Comment is included
		commentsBody, tmpComments = extractComments(doc, cache, opts)
		lenComments = utf8.RuneCountInString(tmpComments)
	} else if opts.Focus == FavorPrecision {
		doc = pruneUnwantedNodes(doc, selector.RemovedComments)
	}

	// Extract content
	postBody, tmpBodyText := extractContent(doc, cache, opts)

	// Use fallback if necessary
	if opts.EnableFallback {
		postBody, tmpBodyText = compareExternalExtraction(docBackup1, postBody, opts)
	}

	// Rescue: try to use original/dirty tree
	lenText := utf8.RuneCountInString(tmpBodyText)
	if lenText < opts.Config.MinExtractedSize && opts.Focus != FavorPrecision {
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
		logDebug(opts, "not enough comments: %s", opts.OriginalURL)
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
