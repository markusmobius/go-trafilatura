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
	"unicode/utf8"

	"github.com/go-shiori/dom"
	"github.com/go-shiori/go-readability"
	distiller "github.com/markusmobius/go-domdistiller"
	"github.com/markusmobius/go-trafilatura/internal/etree"
	"golang.org/x/net/html"
)

var tagsToSanitize = sliceToMap(
	"aside", "audio", "button", "fieldset", "figure", "footer", "iframe",
	"input", "label", "link", "nav", "noindex", "noscript",
	"object", "option", "select", "source", "svg", "time",
)

// compareExtraction decide whether to choose own or external extraction based on a series
// of heuristics. In original Trafilatura, they use python-readability and justext, while
// here we use go-readability and go-domdistiller. Since there are difference in
// implementation between them, here we do it a bit differently compared to the original code.
func compareExtraction(doc, originalExtract *html.Node, opts Options) (*html.Node, string) {
	// Bypass for favor recall
	originalText := trim(etree.IterText(originalExtract, " "))
	lenOriginal := utf8.RuneCountInString(originalText)
	if opts.FavorRecall && lenOriginal > opts.Config.MinExtractedSize*10 {
		return originalExtract, originalText
	}

	// Clean doc to be used for Readability and Dom Distiller
	cleanedDoc := dom.Clone(doc, true)
	cleanedDoc = pruneUnwantedSections(cleanedDoc, opts)

	fallbackCandidates := []*html.Node{}
	//if there are other fallback candidates we use those
	if opts.FallbackCandidates.OtherFallbacks != nil && len(opts.FallbackCandidates.OtherFallbacks) > 0 {
		fallbackCandidates = opts.FallbackCandidates.OtherFallbacks
	} else {
		if opts.FallbackCandidates.HasReadability {
			if opts.FallbackCandidates.ReadabilityFallback != nil {
				fallbackCandidates = append(fallbackCandidates, opts.FallbackCandidates.ReadabilityFallback)
			}
		} else {
			//we run readability
			readabilityExtract, err := tryReadability(cleanedDoc, opts)
			if err == nil {
				fallbackCandidates = append(fallbackCandidates, readabilityExtract)
			} else {
				logWarn(opts, "readability failed: %v", err)
			}
		}
		//now we append domdistiller if it was alreadyrun
		if opts.FallbackCandidates.HasDistiller {
			if opts.FallbackCandidates.DistillerFallback != nil {
				fallbackCandidates = append(fallbackCandidates, opts.FallbackCandidates.DistillerFallback)
			}
		} else {
			// Here we append nil to fallback candidates. This nil value is used to
			// notify Trafilatura to run Go-DomDistiller for that candidate. We do it
			// this way to make sure that dom-distiller will only be run if readability
			// result is still not good enough to use.
			fallbackCandidates = append(fallbackCandidates, nil)
		}
	}

	// Convert url to string for logging
	var originalUrl string
	if opts.OriginalURL != nil {
		originalUrl = opts.OriginalURL.String()
	}

	// Compare
	for i, candidate := range fallbackCandidates {
		// Use dom-distiller if necessary
		if candidate == nil {
			var err error
			candidate, err = tryDomDistiller(cleanedDoc, opts)
			if err != nil {
				logWarn(opts, "dom-distiller failed: %v", err)
				continue
			}
		}

		// Extract text from candidate
		candidateText := trim(dom.TextContent(candidate))
		lenCandidate := utf8.RuneCountInString(candidateText)
		logInfo(opts, "extracted length: %d (candidate-%d) %d (original)", lenCandidate, i+1, lenOriginal)

		// TODO: This part is pretty different compared to the original.
		// Check if this candidate can be used, either because it pass length check
		// or because we need to favor recall.
		var candidateUsable bool

		if lenCandidate == 0 || lenCandidate == lenOriginal {
			candidateUsable = false
		} else if lenOriginal == 0 && lenCandidate > 0 {
			candidateUsable = true
		} else if lenOriginal > 2*lenCandidate {
			candidateUsable = false
		} else if lenCandidate > 2*lenOriginal {
			candidateUsable = true
		} else {
			// Borderline case
			heads := dom.GetElementsByTagName(doc, "head")
			tables := dom.GetElementsByTagName(doc, "table")
			paragraphs := dom.GetElementsByTagName(doc, "p")
			candidateHeadings := dom.QuerySelectorAll(candidate, "h2,h3,h4")

			var pTextLength int
			for _, p := range paragraphs {
				pText := trim(etree.IterText(p, " "))
				pTextLength += utf8.RuneCountInString(pText)
			}

			if pTextLength == 0 && lenCandidate > opts.Config.MinExtractedSize*2 {
				candidateUsable = true
			} else if len(tables) > len(paragraphs) && lenCandidate > opts.Config.MinExtractedSize*2 {
				candidateUsable = true
			} else if opts.FavorRecall && len(heads) == 0 &&
				len(candidateHeadings) > 0 && lenCandidate > lenOriginal {
				candidateUsable = true
			} else {
				logDebug(opts, "extraction values: %d %d for %s", lenOriginal, lenCandidate, originalUrl)
				candidateUsable = false
			}
		}

		mustFavorRecall := lenOriginal < opts.Config.MinExtractedSize && opts.FavorRecall
		if candidateUsable || mustFavorRecall {
			originalExtract = candidate
			lenOriginal = lenCandidate
			logDebug(opts, "candidate-%d usable: %s", i+1, originalUrl)
		}

		if lenOriginal >= opts.Config.MinExtractedSize {
			logDebug(opts, "candidate-%d used: %s", i+1, originalUrl)
			break
		}
	}

	// Sanitize the tree
	sanitizeTree(originalExtract, opts)

	// Return data
	finalText := trim(etree.IterText(originalExtract, " "))
	return originalExtract, finalText
}

func tryReadability(doc *html.Node, opts Options) (*html.Node, error) {
	// Extract using go-readability
	article, err := readability.FromDocument(doc, opts.OriginalURL)
	if err != nil {
		return nil, err
	}

	return article.Node, nil
}

func tryDomDistiller(doc *html.Node, opts Options) (*html.Node, error) {
	// Extract using go-domdistiller
	distillerOpts := &distiller.Options{
		OriginalURL:    opts.OriginalURL,
		SkipPagination: true,
	}

	doc = dom.Clone(doc, true)
	res, err := distiller.Apply(doc, distillerOpts)
	if err != nil {
		return nil, err
	}

	return res.Node, nil
}

// sanitizeTree converts and sanitize the output from the generic
// fallback algorithm (post-processing).
func sanitizeTree(tree *html.Node, opts Options) {
	// 1. Clean
	docCleaning(tree, opts)

	subElements := dom.GetElementsByTagName(tree, "*")
	for i := len(subElements) - 1; i >= 0; i-- {
		elemTag := dom.TagName(subElements[i])
		if _, exist := tagsToSanitize[elemTag]; exist {
			subElements[i].Parent.RemoveChild(subElements[i])
		}
	}

	if !opts.IncludeLinks {
		etree.StripTags(tree, "a")
	}

	etree.StripTags(tree, "span")

	// 2. Sanitize
	var sanitizationList []string
	uniqueTags := make(map[string]struct{})
	for _, node := range dom.GetElementsByTagName(tree, "*") {
		tagName := dom.TagName(node)
		if _, exist := uniqueTags[tagName]; exist {
			continue
		}

		uniqueTags[tagName] = struct{}{}
		if _, exist := validTagCatalog[tagName]; !exist {
			sanitizationList = append(sanitizationList, tagName)
		}
	}

	if len(sanitizationList) > 0 {
		etree.StripTags(tree, sanitizationList...)
	}
}
