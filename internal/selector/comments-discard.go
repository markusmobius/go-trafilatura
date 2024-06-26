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

var DiscardedComments = []Rule{
	discardedCommentsRule1,
	discardedCommentsRule2,
	discardedCommentsRule3,
}

// `.//*[(self::div or self::section)][starts-with(@id, "respond")]`,
func discardedCommentsRule1(n *html.Node) bool {
	id := dom.ID(n)
	tagName := dom.TagName(n)

	switch tagName {
	case "div", "section":
	default:
		return false
	}

	switch {
	case startsWith(id, "respond"):
	default:
		return false
	}

	return true
}

// `.//cite|.//quote`,
func discardedCommentsRule2(n *html.Node) bool {
	tagName := dom.TagName(n)
	return tagName == "cite" || tagName == "quote"
}

// `.//*[@class="comments-title" or contains(@class, "comments-title") or
// contains(@class, "nocomments") or starts-with(@id, "reply-") or
// starts-with(@class, "reply-") or contains(@class, "-reply-") or contains(@class, "message")
// or contains(@class, "signin") or
// contains(@id, "akismet") or contains(@class, "akismet") or contains(@style, "display:none")]`,
func discardedCommentsRule3(n *html.Node) bool {
	id := dom.ID(n)
	class := dom.ClassName(n)
	style := dom.GetAttribute(n, "style")

	switch {
	case class == "comments-title",
		contains(class, "comments-title"),
		contains(class, "nocomments"),
		startsWith(id, "reply-"),
		startsWith(class, "reply-"),
		contains(class, "-reply-"),
		contains(class, "message"),
		contains(class, "signin"),
		contains(id, "akismet"),
		contains(class, "akismet"),
		contains(style, "display:none"):
	default:
		return false
	}

	return true
}
