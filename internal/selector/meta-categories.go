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

var MetaCategories = []Rule{
	metaCategoriesRule1,
	metaCategoriesRule2,
	metaCategoriesRule3,
	metaCategoriesRule4,
	metaCategoriesRule5,
	metaCategoriesRule6,
}

// `//div[starts-with(@class, 'post-info') or starts-with(@class, 'postinfo') or
// starts-with(@class, 'post-meta') or starts-with(@class, 'postmeta') or
// starts-with(@class, 'meta') or starts-with(@class, 'entry-meta') or starts-with(@class, 'entry-info') or
// starts-with(@class, 'entry-utility') or starts-with(@id, 'postpath')]//a[@href]`,
func metaCategoriesRule1(n *html.Node) bool {
	if dom.TagName(n) != "a" || !dom.HasAttribute(n, "href") {
		return false
	}

	ancestors := getNodeAncestors(n, "div")
	if len(ancestors) == 0 {
		return false
	}

	for _, ancestor := range ancestors {
		id := dom.ID(ancestor)
		class := dom.ClassName(ancestor)

		switch {
		case startsWith(class, "post-info"),
			startsWith(class, "postinfo"),
			startsWith(class, "post-meta"),
			startsWith(class, "postmeta"),
			startsWith(class, "meta"),
			startsWith(class, "entry-meta"),
			startsWith(class, "entry-info"),
			startsWith(class, "entry-utility"),
			startsWith(id, "postpath"):
			return true
		}
	}

	return false
}

// `//p[starts-with(@class, 'postmeta') or starts-with(@class, 'entry-categories') or @class='postinfo' or @id='filedunder']//a[@href]`,
func metaCategoriesRule2(n *html.Node) bool {
	if dom.TagName(n) != "a" || !dom.HasAttribute(n, "href") {
		return false
	}

	ancestors := getNodeAncestors(n, "p")
	if len(ancestors) == 0 {
		return false
	}

	for _, ancestor := range ancestors {
		id := dom.ID(ancestor)
		class := dom.ClassName(ancestor)

		switch {
		case startsWith(class, "postmeta"),
			startsWith(class, "entry-categories"),
			class == "postinfo",
			id == "filedunder":
			return true
		}
	}

	return false
}

// `//footer[starts-with(@class, 'entry-meta') or starts-with(@class, 'entry-footer')]//a[@href]`,
func metaCategoriesRule3(n *html.Node) bool {
	if dom.TagName(n) != "a" || !dom.HasAttribute(n, "href") {
		return false
	}

	ancestors := getNodeAncestors(n, "footer")
	if len(ancestors) == 0 {
		return false
	}

	for _, ancestor := range ancestors {
		class := dom.ClassName(ancestor)

		switch {
		case startsWith(class, "entry-meta"),
			startsWith(class, "entry-footer"):
			return true
		}
	}

	return false
}

// `//*[(self::li or self::span)][@class="post-category" or @class="postcategory" or @class="entry-category" or contains(@class, "cat-links")]//a[@href]`,
func metaCategoriesRule4(n *html.Node) bool {
	if dom.TagName(n) != "a" || !dom.HasAttribute(n, "href") {
		return false
	}

	ancestors := getNodeAncestors(n, "li")
	ancestors = append(ancestors, getNodeAncestors(n, "span")...)
	if len(ancestors) == 0 {
		return false
	}

	for _, ancestor := range ancestors {
		class := dom.ClassName(ancestor)

		switch {
		case class == "post-category",
			class == "postcategory",
			class == "entry-category",
			contains(class, "cat-links"):
			return true
		}
	}

	return false
}

// `//header[@class="entry-header"]//a[@href]`,
func metaCategoriesRule5(n *html.Node) bool {
	if dom.TagName(n) != "a" || !dom.HasAttribute(n, "href") {
		return false
	}

	ancestors := getNodeAncestors(n, "header")
	if len(ancestors) == 0 {
		return false
	}

	for _, ancestor := range ancestors {
		class := dom.ClassName(ancestor)

		if class == "entry-header" {
			return true
		}
	}

	return false
}

// `//div[@class="row" or @class="tags"]//a[@href]`,
func metaCategoriesRule6(n *html.Node) bool {
	if dom.TagName(n) != "a" || !dom.HasAttribute(n, "href") {
		return false
	}

	ancestors := getNodeAncestors(n, "div")
	if len(ancestors) == 0 {
		return false
	}

	for _, ancestor := range ancestors {
		switch dom.ClassName(ancestor) {
		case "row", "tags":
			return true
		}
	}

	return false
}
