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
	"unicode/utf8"

	"github.com/go-shiori/dom"
	"github.com/go-shiori/go-readability"
	distiller "github.com/markusmobius/go-domdistiller"
	"github.com/markusmobius/go-trafilatura/internal/etree"
	"github.com/markusmobius/go-trafilatura/internal/selector"
	"golang.org/x/net/html"
)

type _FallbackGenerator func() (string, *html.Node)

var tagsToSanitize = sliceToMap(
	"aside", "audio", "button", "fieldset", "figure", "footer", "iframe",
	"input", "label", "link", "nav", "noindex", "noscript",
	"object", "option", "select", "source", "svg", "time",
)

// compareExternalExtraction decide whether to choose own or external extraction based on
// a series of heuristics. In original Trafilatura, they use python-readability and justext,
// while here we use go-readability and go-domdistiller. Since there are difference in
// implementation between them, here we do it a bit differently compared to the original code.
//
// In original Trafilatura, this function is named `compare_extraction`.
func compareExternalExtraction(originalDoc, extractedDoc *html.Node, opts Options) (*html.Node, string) {
	// Bypass for favor recall
	extractedText := trim(etree.IterText(extractedDoc, " "))
	lenExtracted := utf8.RuneCountInString(extractedText)
	if opts.Focus == FavorRecall && lenExtracted > opts.Config.MinExtractedSize*10 {
		return extractedDoc, extractedText
	}

	// Convert url to string for logging
	var originalUrl string
	if opts.OriginalURL != nil {
		originalUrl = opts.OriginalURL.String()
	}
	logInfo(opts, "trying external extractor for url %q", originalUrl)

	// Prior cleaning
	cleanedDoc := dom.Clone(originalDoc, true)
	if opts.Focus == FavorPrecision {
		cleanedDoc = pruneUnwantedNodes(cleanedDoc, selector.OverallDiscardedContent)
	}

	// Process each candidate
	for _, generator := range createFallbackGenerators(cleanedDoc, opts) {
		// Generate candidate, skip if empty
		candidateTitle, candidateDoc := generator()
		if candidateDoc == nil {
			continue
		}

		// Extract text from candidate
		candidateText := trim(etree.IterText(candidateDoc, " "))
		lenCandidate := utf8.RuneCountInString(candidateText)
		logInfo(opts, "comparison for %q: candidate %d vs extracted %d",
			candidateTitle, lenCandidate, lenExtracted)

		// Check if candidate is usable
		if candidateIsUsable(candidateDoc, extractedDoc, lenCandidate, lenExtracted, opts) {
			extractedDoc, lenExtracted = candidateDoc, lenCandidate
			logDebug(opts, "candidate %s is usable", candidateTitle)
		}

		if lenExtracted >= opts.Config.MinExtractedSize {
			logDebug(opts, "candidate %s is used", candidateTitle)
			break
		}
	}

	// Final cleaning
	sanitizeTree(extractedDoc, opts)
	extractedText = trim(etree.IterText(extractedDoc, " "))
	return extractedDoc, extractedText
}

func createFallbackGenerators(doc *html.Node, opts Options) []_FallbackGenerator {
	// Initial variables
	var generators []_FallbackGenerator
	var customCandidates []*html.Node
	var readabilityCandidate, distillerCandidate *html.Node

	if opts.FallbackCandidates != nil {
		customCandidates = opts.FallbackCandidates.Others
		distillerCandidate = opts.FallbackCandidates.Distiller
		readabilityCandidate = opts.FallbackCandidates.Readability
	}

	// First is the user specified custom candidates.
	for i, candidate := range customCandidates {
		if candidate == nil {
			continue
		}

		generators = append(generators, func() (string, *html.Node) {
			return fmt.Sprintf("Candidate-%d", i), candidate
		})
	}

	// Next is Readability
	readabilityTitle := "Readability"

	if readabilityCandidate != nil {
		generators = append(generators, func() (string, *html.Node) {
			return readabilityTitle, readabilityCandidate
		})
	} else {
		generators = append(generators, func() (string, *html.Node) {
			result, _ := readability.FromDocument(doc, opts.OriginalURL)
			return readabilityTitle, result.Node
		})
	}

	// Last is Dom Distiller
	distillerTitle := "Dom Distiller"

	if distillerCandidate != nil {
		generators = append(generators, func() (string, *html.Node) {
			return distillerTitle, distillerCandidate
		})
	} else {
		generators = append(generators, func() (string, *html.Node) {
			clone := dom.Clone(doc, true)
			result, _ := distiller.Apply(clone, &distiller.Options{
				OriginalURL:    opts.OriginalURL,
				SkipPagination: true})
			if result == nil {
				return "", nil
			}
			return distillerTitle, result.Node
		})
	}

	return generators
}

// candidateIsUsable check if the fallback candidate is good enough to use as extraction result.
func candidateIsUsable(candidateDoc, extractedDoc *html.Node, lenCandidate, lenExtracted int, opts Options) bool {
	var candidateUsable bool

	if lenCandidate == 0 || lenCandidate == lenExtracted {
		candidateUsable = false
	} else if lenExtracted == 0 && lenCandidate > 0 {
		candidateUsable = true
	} else if lenExtracted > 2*lenCandidate {
		candidateUsable = false
	} else if lenCandidate > 2*lenExtracted {
		candidateUsable = true
	} else {
		// Borderline case
		extractedHeads := dom.GetElementsByTagName(extractedDoc, "head")
		extractedTables := dom.GetElementsByTagName(extractedDoc, "table")
		extractedParagraphs := dom.GetElementsByTagName(extractedDoc, "p")
		candidateHeadings := dom.QuerySelectorAll(candidateDoc, "h2,h3,h4")

		var pTextLength int
		for _, p := range extractedParagraphs {
			pText := trim(etree.IterText(p, " "))
			pTextLength += utf8.RuneCountInString(pText)
		}

		if pTextLength == 0 && lenCandidate > opts.Config.MinExtractedSize*2 {
			candidateUsable = true
		} else if len(extractedTables) > len(extractedParagraphs) && lenCandidate > opts.Config.MinExtractedSize*2 {
			candidateUsable = true
		} else if opts.Focus == FavorRecall && len(extractedHeads) == 0 &&
			len(candidateHeadings) > 0 && lenCandidate > lenExtracted {
			candidateUsable = true
		} else {
			candidateUsable = false
		}
	}

	mustFavorRecall := lenExtracted < opts.Config.MinExtractedSize && opts.Focus == FavorRecall
	return candidateUsable || mustFavorRecall
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
