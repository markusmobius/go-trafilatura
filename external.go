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
	"fmt"
	"strings"

	"github.com/go-shiori/dom"
	"github.com/go-shiori/go-readability"
	distiller "github.com/markusmobius/go-domdistiller"
	"github.com/markusmobius/go-trafilatura/internal/etree"
	"golang.org/x/net/html"
)

func tryReadability(originalExtract, doc *html.Node, opts Options) (*html.Node, error) {
	// Extract using go-readability
	docHtml := dom.OuterHTML(doc)
	r := strings.NewReader(docHtml)
	article, err := readability.FromReader(r, opts.OriginalURL)
	if err != nil {
		return nil, err
	}

	return article.Node, nil
}

func tryDomDistiller(originalExtract, doc *html.Node, opts Options) (*html.Node, error) {
	// Extract using go-domdistiller
	docHtml := dom.OuterHTML(doc)
	r := strings.NewReader(docHtml)

	res, err := distiller.ApplyForReader(r, &distiller.Options{
		OriginalURL:    opts.OriginalURL,
		SkipPagination: true})
	if err != nil {
		return nil, err
	}

	// Since dom-distiller return extract in HTML string, parse it back into node
	distillerExtract := dom.CreateElement("div")
	dom.SetInnerHTML(distillerExtract, res.HTML)
	if len(dom.Children(distillerExtract)) == 0 {
		return nil, fmt.Errorf("can't find any content")
	}

	return distillerExtract, nil
}

// sanitizeTree converts and sanitize the output from the generic
// fallback algorithm (post-processing).
func sanitizeTree(tree *html.Node, opts Options) {
	// Get list of tags to sanitize
	sanitizeList := duplicateMap(tagsToSanitize)
	if opts.IncludeImages {
		delete(sanitizeList, "img")
		delete(sanitizeList, "image")
	}

	// Delete unnecessary elements
	for _, elem := range dom.GetElementsByTagName(tree, "*") {
		elemTag := dom.TagName(elem)
		if _, exist := sanitizeList[elemTag]; exist {
			etree.Remove(elem, true)
		}
	}

	// Handle links
	strippingList := duplicateMap(tagsToStrip)
	strippingList["span"] = struct{}{}

	if !opts.IncludeLinks {
		strippingList["a"] = struct{}{}
	}

	if opts.IncludeImages {
		delete(strippingList, "img")
	}

	for tagName := range strippingList {
		etree.StripTags(tree, tagName)
	}

	pruneHTML(tree)

	// Sanitize
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
