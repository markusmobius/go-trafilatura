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
