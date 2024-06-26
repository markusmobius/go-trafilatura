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
