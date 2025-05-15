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
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/RadhiFadlillah/whatlanggo"
	"github.com/go-shiori/dom"
	"github.com/markusmobius/go-trafilatura/internal/etree"
	"github.com/markusmobius/go-trafilatura/internal/lru"
	"github.com/markusmobius/go-trafilatura/internal/re2go"
	"golang.org/x/net/html"
)

var (
	rxHtmlLang = regexp.MustCompile(`(?i)[a-z]{2}`)
)

// checkHtmlLanguage checks HTML meta-elements for language information and
// split the result in case there are several language.
func checkHtmlLanguage(doc *html.Node, opts Options, strict bool) bool {
	htmlNode := doc
	if dom.TagName(htmlNode) != "html" {
		htmlNodes := dom.GetElementsByTagName(doc, "html")
		if len(htmlNodes) > 0 {
			htmlNode = htmlNodes[0]
		}
	}

	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Content-Language
	selectors := []string{
		`meta[http-equiv="content-language"][content]`,
		`meta[property="og:locale"][content]`,
	}

	for _, selector := range selectors {
		metaNodes := dom.QuerySelectorAll(doc, selector)
		if len(metaNodes) == 0 {
			continue
		}

		for _, metaNode := range metaNodes {
			metaContent := dom.GetAttribute(metaNode, "content")
			for _, lang := range rxHtmlLang.FindAllString(metaContent, -1) {
				if strings.ToLower(lang) == opts.TargetLanguage {
					return true
				}
			}
		}

		logWarn(opts, "html language detection in meta failed")
		return false
	}

	// HTML lang attribute: sometimes a wrong indication
	if strict && htmlNode != nil && dom.HasAttribute(htmlNode, "lang") {
		langAttr := dom.GetAttribute(htmlNode, "lang")
		for _, lang := range rxHtmlLang.FindAllString(langAttr, -1) {
			if strings.ToLower(lang) == opts.TargetLanguage {
				return true
			}
		}

		logWarn(opts, "html language detection failed")
		return false
	}

	logWarn(opts, "no html language elements found")
	return true
}

// languageClassifier returns the language of the text.
func languageClassifier(contentText, commentsText string) string {
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

// textFilter filters out unwanted text
func textFilter(n *html.Node) bool {
	var testText string
	text, tail := etree.Text(n), etree.Tail(n)
	if text == "" {
		testText = tail
	} else {
		testText = text
	}

	if !textCharsTest(testText) {
		return true
	}

	lines := strings.Split(testText, "\n")
	return slices.ContainsFunc(lines, re2go.IsTextFilter)
}

// textCharsTest determine if a string is only composed of spaces and/or control characters.
func textCharsTest(s string) bool {
	s = trim(s)
	return s != ""
}

// duplicateTest checks for duplicate text within cache
func duplicateTest(element *html.Node, cache *lru.Cache, opts Options) bool {
	var isDuplicate bool
	testString := trim(etree.IterText(element, " "))

	if utf8.RuneCountInString(testString) > opts.Config.MinDuplicateCheckSize {
		cacheVal, _ := cache.Get(testString)
		if cacheVal > opts.Config.MaxDuplicateCount {
			isDuplicate = true
		}
		cache.Put(testString, cacheVal+1)
	}

	return isDuplicate
}
