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

var MetaTitle = []Rule{
	metaTitleRule1,
	metaTitleRule2,
	metaTitleRule3,
}

// `//*[(self::h1 or self::h2)][contains(@class, "post-title") or contains(@class, "entry-title") or contains(@class, "headline") or contains(@id, "headline") or contains(@itemprop, "headline") or contains(@class, "post__title") or contains(@class, "article-title")]`,
func metaTitleRule1(n *html.Node) bool {
	id := dom.ID(n)
	class := dom.ClassName(n)
	itemProp := dom.GetAttribute(n, "itemprop")
	tagName := dom.TagName(n)

	switch tagName {
	case "h1", "h2":
	default:
		return false
	}

	switch {
	case contains(class, "post-title"),
		contains(class, "entry-title"),
		contains(class, "headline"),
		contains(id, "headline"),
		contains(itemProp, "headline"),
		contains(class, "post__title"),
		contains(class, "article-title"):
	default:
		return false
	}

	return true
}

// `//*[@class="entry-title" or @class="post-title"]`,
func metaTitleRule2(n *html.Node) bool {
	switch dom.ClassName(n) {
	case "entry-title", "post-title":
		return true
	default:
		return false
	}
}

// `//*[(self::h1 or self::h2 or self::h3)][contains(@class, "title") or contains(@id, "title")]`,
func metaTitleRule3(n *html.Node) bool {
	id := dom.ID(n)
	class := dom.ClassName(n)
	tagName := dom.TagName(n)

	switch tagName {
	case "h1", "h2", "h3":
	default:
		return false
	}

	switch {
	case contains(class, "title"),
		contains(id, "title"):
	default:
		return false
	}

	return true
}
