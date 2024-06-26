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

package selector

import (
	"github.com/go-shiori/dom"
	"golang.org/x/net/html"
)

var MetaTags = []Rule{
	metaTagsRule1,
	metaTagsRule2,
	metaTagsRule3,
	metaTagsRule4,
}

// `//div[@class="tags"]//a[@href]`,
func metaTagsRule1(n *html.Node) bool {
	if dom.TagName(n) != "a" || !dom.HasAttribute(n, "href") {
		return false
	}

	ancestors := getNodeAncestors(n, "div")
	if len(ancestors) == 0 {
		return false
	}

	for _, ancestor := range ancestors {
		if dom.ClassName(ancestor) == "tags" {
			return true
		}
	}

	return false
}

// `//p[starts-with(@class, 'entry-tags')]//a[@href]`,
func metaTagsRule2(n *html.Node) bool {
	if dom.TagName(n) != "a" || !dom.HasAttribute(n, "href") {
		return false
	}

	ancestors := getNodeAncestors(n, "p")
	if len(ancestors) == 0 {
		return false
	}

	for _, ancestor := range ancestors {
		class := dom.ClassName(ancestor)
		if startsWith(class, "entry-tags") {
			return true
		}
	}

	return false
}

// `//div[@class="row" or @class="jp-relatedposts" or
// @class="entry-utility" or starts-with(@class, 'tag') or
// starts-with(@class, 'postmeta') or starts-with(@class, 'meta')]//a[@href]`,
func metaTagsRule3(n *html.Node) bool {
	if dom.TagName(n) != "a" || !dom.HasAttribute(n, "href") {
		return false
	}

	ancestors := getNodeAncestors(n, "div")
	if len(ancestors) == 0 {
		return false
	}

	for _, ancestor := range ancestors {
		class := dom.ClassName(ancestor)

		switch {
		case class == "row",
			class == "jp-relatedposts",
			class == "entry-utility",
			startsWith(class, "tag"),
			startsWith(class, "postmeta"),
			startsWith(class, "meta"):
			return true
		}
	}

	return false
}

// `//*[@class="entry-meta" or contains(@class, "topics") or contains(@class, "tags-links")]//a[@href]`,
func metaTagsRule4(n *html.Node) bool {
	if dom.TagName(n) != "a" || !dom.HasAttribute(n, "href") {
		return false
	}

	for parent := n.Parent; parent != nil; parent = parent.Parent {
		class := dom.ClassName(parent)

		switch {
		case class == "entry-meta",
			contains(class, "topics"),
			contains(class, "tags-links"):
			return true
		}
	}

	return false
}
