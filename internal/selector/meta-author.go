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
	"strings"

	"github.com/go-shiori/dom"
	"golang.org/x/net/html"
)

var MetaAuthor = []Rule{
	metaAuthorRule1,
	metaAuthorRule2,
	metaAuthorRule3,
}

// specific and almost specific
// `//*[(self::a or self::address or self::div or self::link or self::p or self::span or self::strong)][@rel="author" or @id="author" or @class="author" or @itemprop="author name" or rel="me" or contains(@class, "author-name") or contains(@class, "AuthorName") or contains(@class, "authorName") or contains(@class, "author name")]|//author`,
func metaAuthorRule1(n *html.Node) bool {
	id := dom.ID(n)
	class := dom.ClassName(n)
	rel := dom.GetAttribute(n, "rel")
	itemProp := dom.GetAttribute(n, "itemprop")
	tagName := dom.TagName(n)

	switch tagName {
	case "a", "address", "div", "link", "p", "span", "strong":
	case "author":
		return true
	default:
		return false
	}

	switch {
	case rel == "author",
		id == "author",
		class == "author",
		itemProp == "author name",
		rel == "me",
		strings.Contains(class, "author-name"),
		strings.Contains(class, "AuthorName"),
		strings.Contains(class, "authorName"),
		strings.Contains(class, "author name"):
	default:
		return false
	}

	return true
}

// almost generic and generic, last ones not common
// `//*[(self::a or self::div or self::h3 or self::h4 or self::p or self::span)][contains(@class, "author") or contains(@id, "author") or contains(@itemprop, "author") or @class="byline" or contains(@id, "zuozhe") or contains(@class, "zuozhe") or contains(@id, "bianji") or contains(@class, "bianji") or contains(@id, "xiaobian") or contains(@class, "xiaobian") or contains(@class, "submitted-by") or contains(@class, "posted-by") or @class="username" or @class="BBL" or contains(@class, "journalist-name")]`,
func metaAuthorRule2(n *html.Node) bool {
	id := dom.ID(n)
	class := dom.ClassName(n)
	itemProp := dom.GetAttribute(n, "itemprop")
	tagName := dom.TagName(n)

	switch tagName {
	case "a", "div", "h3", "h4", "p", "span":
	default:
		return false
	}

	switch {
	case strings.Contains(class, "author"),
		strings.Contains(id, "author"),
		strings.Contains(itemProp, "author"),
		class == "byline",
		strings.Contains(id, "zuozhe"),
		strings.Contains(class, "zuozhe"),
		strings.Contains(id, "bianji"),
		strings.Contains(class, "bianji"),
		strings.Contains(id, "xiaobian"),
		strings.Contains(class, "xiaobian"),
		strings.Contains(class, "submitted-by"),
		strings.Contains(class, "posted-by"),
		class == "username",
		class == "BBL",
		strings.Contains(class, "journalist-name"):
	default:
		return false
	}

	return true
}

// last resort: any element
// `//*[contains(translate(@id, "A", "a"), "author") or contains(translate(@class, "A", "a"), "author") or contains(@class, "screenname") or contains(@data-component, "Byline") or contains(@itemprop, "author") or contains(@class, "writer") or contains(translate(@class, "B", "b"), "byline")]`,
func metaAuthorRule3(n *html.Node) bool {
	id := dom.ID(n)
	class := dom.ClassName(n)
	dataComponent := dom.GetAttribute(n, "data-component")
	itemProp := dom.GetAttribute(n, "itemprop")

	lId := strings.ToLower(id)
	lClass := strings.ToLower(class)
	lDataComponent := strings.ToLower(dataComponent)

	switch {
	case strings.Contains(lId, "author"),
		strings.Contains(lClass, "author"),
		strings.Contains(class, "screenname"),
		strings.Contains(lDataComponent, "byline"),
		strings.Contains(itemProp, "author"),
		strings.Contains(class, "writer"),
		strings.Contains(lClass, "byline"):
	default:
		return false
	}

	return true
}
