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

var MetaAuthorDiscard = []Rule{
	metaAuthorDiscardRule1,
	metaAuthorDiscardRule2,
}

// `.//*[(self::a or self::div or self::section or self::span)][@id='comments' or @class='comments' or @class='title' or @class='date' or
// contains(@id, 'commentlist') or contains(@class, 'commentlist') or contains(@class, 'sidebar') or contains(@class, 'is-hidden') or contains(@class, 'quote')
// or contains(@id, 'comment-list') or contains(@class, 'comments-list') or contains(@class, 'embedly-instagram') or contains(@id, 'ProductReviews') or
// starts-with(@id, 'comments') or contains(@data-component, "Figure") or contains(@class, "article-share") or contains(@class, "article-support") or contains(@class, "print") or contains(@class, "category") or contains(@class, "meta-date") or contains(@class, "meta-reviewer")
// or starts-with(@class, 'comments') or starts-with(@class, 'Comments')]`,
func metaAuthorDiscardRule1(n *html.Node) bool {
	id := dom.ID(n)
	class := dom.ClassName(n)
	dataComponent := dom.GetAttribute(n, "data-component")
	tagName := dom.TagName(n)

	switch tagName {
	case "a", "div", "section", "span":
	default:
		return false
	}

	switch {
	case id == "comments",
		class == "comments",
		class == "title",
		class == "date",
		strings.Contains(id, "commentlist"),
		strings.Contains(class, "commentlist"),
		strings.Contains(class, "sidebar"),
		strings.Contains(class, "is-hidden"),
		strings.Contains(class, "quote"),
		strings.Contains(id, "comment-list"),
		strings.Contains(class, "comment-list"),
		strings.Contains(class, "embedly-instagram"),
		strings.Contains(id, "ProductReviews"),
		strings.HasPrefix(id, "comments"),
		strings.Contains(dataComponent, "Figure"),
		strings.Contains(class, "article-share"),
		strings.Contains(class, "article-support"),
		strings.Contains(class, "print"),
		strings.Contains(class, "category"),
		strings.Contains(class, "meta-date"),
		strings.Contains(class, "meta-reviewer"),
		strings.HasPrefix(class, "comments"),
		strings.HasPrefix(class, "Comments"):
	default:
		return false
	}

	return true
}

// "//time|//figure",
func metaAuthorDiscardRule2(n *html.Node) bool {
	tagName := dom.TagName(n)

	switch tagName {
	case "time", "figure":
	default:
		return false
	}

	return true
}
