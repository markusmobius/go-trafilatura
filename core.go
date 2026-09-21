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
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	nurl "net/url"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/andybalholm/cascadia"
	"github.com/go-shiori/dom"
	"github.com/markusmobius/go-trafilatura/v2/internal/etree"
	"github.com/markusmobius/go-trafilatura/v2/internal/lru"
	"github.com/markusmobius/go-trafilatura/v2/internal/selector"
	"github.com/rs/zerolog"
	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
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
	if r == nil {
		return nil, fmt.Errorf("HTML reader is nil")
	}
	if opts.InputEncoding != "" {
		if encoding, _ := charset.Lookup(opts.InputEncoding); encoding == nil {
			return nil, fmt.Errorf("unsupported charset: %q", opts.InputEncoding)
		}
	}
	buffered := bufio.NewReader(r)
	header, err := buffered.Peek(2)
	if err != nil && err != io.EOF {
		return nil, err
	}
	r = buffered
	if len(header) == 2 && header[0] == 0x1f && header[1] == 0x8b {
		compressed, err := gzip.NewReader(buffered)
		if err != nil {
			return nil, err
		}
		defer compressed.Close()
		r = compressed
	}

	// Parse HTML
	var doc *html.Node
	if opts.InputEncoding == "" {
		doc, err = dom.Parse(r)
	} else {
		decoded, decodeErr := charset.NewReaderLabel(opts.InputEncoding, r)
		if decodeErr != nil {
			return nil, decodeErr
		}
		normalized := transform.NewReader(decoded, transform.Chain(
			norm.NFD,
			runes.Remove(runes.Predicate(func(character rune) bool { return character == '\u00ad' })),
			norm.NFC,
		))
		doc, err = html.Parse(normalized)
	}
	if err != nil {
		return nil, err
	}

	return ExtractDocument(doc, opts)
}

// ExtractDocument parses the specified document and find the main readable content.
func ExtractDocument(doc *html.Node, opts Options) (*ExtractResult, error) {
	if doc == nil {
		return nil, fmt.Errorf("HTML document is nil")
	}
	doc = dom.Clone(doc, true)

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

	postBody, tmpBodyText, commentsBody, tmpComments := extractionSequence(doc, cache, opts)
	lenComments := utf8.RuneCountInString(tmpComments)
	lenText := utf8.RuneCountInString(tmpBodyText)

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
	if opts.TargetLanguage != "" && lang != opts.TargetLanguage {
		return nil, fmt.Errorf("wrong language, want %s got %s", opts.TargetLanguage, lang)
	}
	metadata.Language = lang

	// Post cleaning
	postCleaning(postBody)
	postCleaning(commentsBody)

	return &ExtractResult{
		ContentNode:  postBody,
		ContentText:  plainText(postBody),
		CommentsNode: commentsBody,
		CommentsText: plainText(commentsBody),
		Metadata:     metadata,
	}, nil
}

func plainText(root *html.Node) string {
	var text strings.Builder
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		if node == nil {
			return
		}
		block := inMap(dom.TagName(node), textBlockTags)
		if block {
			text.WriteByte('\n')
		}
		if node.Type == html.TextNode {
			text.WriteString(node.Data)
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
		if block {
			text.WriteByte('\n')
		}
	}
	visit(root)
	var lines []string
	for _, line := range strings.Split(text.String(), "\n") {
		if line = trim(line); line != "" {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}

var discussionForumPosting = regexp.MustCompile(`"@type"[\s\v\x{1c}-\x{1f}\x{85}\p{Z}]*:[\s\v\x{1c}-\x{1f}\x{85}\p{Z}]*(?:"DiscussionForumPosting"|\[[^\]]*"DiscussionForumPosting")`)

func forumThreadPage(doc *html.Node) bool {
	for _, script := range dom.QuerySelectorAll(doc, `script[type="application/ld+json"]`) {
		if discussionForumPosting.MatchString(etree.Text(script)) {
			return true
		}
	}
	return false
}

func prepareTree(doc *html.Node, opts Options) *html.Node {
	cleaned := dom.Clone(doc, true)
	prepareCore(cleaned, opts)
	return cleaned
}

func recallRetry(doc *html.Node, opts Options) (*html.Node, string) {
	opts.Focus = FavorRecall
	cache := lru.NewCache(opts.Config.CacheSize)
	body, text := extractContent(prepareTree(doc, opts), cache, opts)
	if opts.EnableFallback {
		body, text = compareExternalExtraction(doc, body, opts)
	}
	return body, text
}

func extractionSequence(doc *html.Node, cache *lru.Cache, opts Options) (*html.Node, string, *html.Node, string) {
	isForum := forumThreadPage(doc)
	if opts.ExcludeComments && (opts.Focus == FavorPrecision || !isForum) {
		doc = pruneUnwantedNodes(doc, selector.RemovedComments)
	}
	cleaned := prepareTree(doc, opts)
	var commentsBody, forumPosts *html.Node
	var commentsText string
	if !opts.ExcludeComments {
		commentsBody, commentsText = extractComments(cleaned, cache, opts)
		if commentsText != "" && isForum {
			forumPosts = commentsBody
			commentsBody, commentsText = nil, ""
			cleaned = prepareTree(doc, opts)
		}
	}
	if opts.Focus == FavorPrecision && !isForum {
		cleaned = pruneUnwantedNodes(cleaned, selector.RemovedComments)
	}
	body, text := extractContent(cleaned, cache, opts)
	if opts.EnableFallback {
		body, text = compareExternalExtraction(doc, body, opts)
	}
	length := utf8.RuneCountInString(text)
	if length < opts.Config.MinExtractedSize && opts.Focus != FavorPrecision {
		body, text = baseline(doc)
		length = utf8.RuneCountInString(text)
		forumPosts = nil
	}
	if opts.Focus == Balanced && length > 0 && length < 3000 && float64(length) < 0.2*float64(utf8.RuneCountInString(html2txt(doc))) {
		retryDoc := doc
		if !isForum {
			retryDoc = pruneUnwantedNodes(dom.Clone(doc, true), selector.RemovedComments)
		}
		retryBody, retryText := recallRetry(retryDoc, opts)
		retryLength := utf8.RuneCountInString(retryText)
		var distillerBody *html.Node
		var distillerText string
		if opts.EnableFallback {
			distillerBody, distillerText = distillerRescue(retryDoc, opts)
		}
		distillerLength := utf8.RuneCountInString(distillerText)
		if distillerLength > retryLength && distillerLength > 2*length {
			body, text, forumPosts = distillerBody, distillerText, nil
		} else if retryLength >= opts.Config.MinExtractedSize && float64(retryLength) > 1.5*float64(length) {
			body, text, forumPosts = retryBody, retryText, nil
		}
	}
	if forumPosts != nil {
		var existing []string
		for _, element := range dom.Children(body) {
			existing = append(existing, trim(dom.TextContent(element)))
		}
		bodyText := strings.Join(existing, "\n")
		changed := false
		for _, post := range dom.Children(forumPosts) {
			if postText := trim(dom.TextContent(post)); postText != "" && !strings.Contains(bodyText, postText) {
				etree.Append(body, post)
				changed = true
			}
		}
		if changed {
			text = etree.ExtractionText(body)
		}
	}
	return body, text, commentsBody, commentsText
}
