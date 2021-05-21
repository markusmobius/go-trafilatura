// This file is part of go-trafilatura, Go package for extracting readable
// content, comments and metadata from a web page. Source available in
// <https://github.com/markusmobius/go-trafilatura>.
// Copyright (C) 2021 Markus Mobius
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by the
// Free Software Foundation, either version 3 of the License, or (at your
// option) any later version.
//
// This program is distributed in the hope that it will be useful, but
// WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY
// or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License
// for more details.
//
// You should have received a copy of the GNU General Public License along
// with this program. If not, see <https://www.gnu.org/licenses/>.

// Code in this file is ported from <https://github.com/adbar/trafilatura>
// which available under GNU GPL v3 license.

package trafilatura

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/go-shiori/dom"
	"github.com/markusmobius/go-trafilatura/internal/etree"
	"golang.org/x/net/html"
)

var (
	rxHtmlLang   = regexp.MustCompile(`(?i)[a-z]{2}`)
	rxTextFilter = regexp.MustCompile(`(?i)\W*(Drucken|E-?Mail|Facebook|Flipboard|Google|Instagram|Linkedin|Mail|PDF|Pinterest|Pocket|Print|Reddit|Twitter|Whatsapp|Xing)$`)
)

// checkHtmlLanguage checks HTML meta-elements for language information and
// split the result in case there are several language.
func checkHtmlLanguage(doc *html.Node, opts Options) bool {
	htmlNode := doc
	if dom.TagName(htmlNode) != "html" {
		htmlNodes := dom.GetElementsByTagName(doc, "html")
		if len(htmlNodes) > 0 {
			htmlNode = htmlNodes[0]
		}
	}

	if htmlNode != nil && dom.HasAttribute(htmlNode, "lang") {
		langAttr := dom.GetAttribute(htmlNode, "lang")
		for _, lang := range rxHtmlLang.FindAllString(langAttr, -1) {
			if lang == opts.TargetLanguage {
				return true
			}
		}

		logWarn(opts, "html language detection failed")
		return false
	}

	metaNodes := dom.QuerySelectorAll(doc, `meta[http-equiv="content-language"]`)
	if len(metaNodes) > 0 {
		for _, metaNode := range metaNodes {
			metaContent := dom.GetAttribute(metaNode, "content")
			for _, lang := range rxHtmlLang.FindAllString(metaContent, -1) {
				if lang == opts.TargetLanguage {
					return true
				}
			}
		}

		logWarn(opts, "html language detection in meta failed")
		return false
	}

	logWarn(opts, "no html language elements found")
	return true
}

// textFilter filters out unwanted text
func textFilter(n *html.Node) bool {
	var testText string
	text, tail := etree.Text(n), etree.Tail(n)
	if text == "" && tail != "" {
		testText = tail
	} else {
		testText = text
	}

	if !textCharsTest(testText) {
		return true
	}

	for _, line := range strings.Split(testText, "\n") {
		if rxTextFilter.MatchString(line) {
			return true
		}
	}

	return false
}

// textCharsTest determine if a string is only composed of spaces and/or control characters.
func textCharsTest(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	return true
}

// duplicateTest checks for duplicate text within cache
func duplicateTest(element *html.Node, cache *Cache, opts Options) bool {
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
