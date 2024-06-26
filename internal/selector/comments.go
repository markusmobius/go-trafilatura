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

var Comments = []Rule{
	commentsRule1,
	commentsRule2,
	commentsRule3,
	commentsRule4,
}

// `.//*[(self::div or self::ol or self::ul or self::dl or self::section)][contains(@id, 'commentlist')
// or contains(@class, 'commentlist') or contains(@class, 'comment-page') or
// contains(@id, 'comment-list') or contains(@class, 'comments-list') or
// contains(@class, 'comments-content') or contains(@class, 'post-comments')]`,
func commentsRule1(n *html.Node) bool {
	id := dom.ID(n)
	class := dom.ClassName(n)
	tagName := dom.TagName(n)

	switch tagName {
	case "div", "ol", "ul", "dl", "section":
	default:
		return false
	}

	switch {
	case contains(id, "commentlist"),
		contains(class, "commentlist"),
		contains(class, "comment-page"),
		contains(id, "comment-list"),
		contains(class, "comment-list"), // additional
		contains(class, "comments-list"),
		contains(class, "comments-content"),
		contains(class, "post-comments"):
	default:
		return false
	}

	return true
}

// `.//*[(self::div or self::section or self::ol or self::ul or self::dl)][starts-with(@id, 'comments')
// or starts-with(@class, 'comments') or starts-with(@class, 'Comments') or
// starts-with(@id, 'comment-') or starts-with(@class, 'comment-') or
// contains(@class, 'article-comments')]`,
func commentsRule2(n *html.Node) bool {
	id := dom.ID(n)
	class := dom.ClassName(n)
	tagName := dom.TagName(n)

	switch tagName {
	case "div", "section", "ol", "ul", "dl":
	default:
		return false
	}

	switch {
	case startsWith(id, "comments"),
		startsWith(lower(class), "comments"),
		startsWith(id, "comment-"),
		startsWith(class, "comment-"),
		contains(class, "article-comments"):
	default:
		return false
	}

	return true
}

// `.//*[(self::div or self::section or self::ol or self::ul or self::dl)][starts-with(@id, 'comol') or
// starts-with(@id, 'disqus_thread') or starts-with(@id, 'dsq-comments')]`,
func commentsRule3(n *html.Node) bool {
	id := dom.ID(n)
	tagName := dom.TagName(n)

	switch tagName {
	case "div", "section", "ol", "ul", "dl":
	default:
		return false
	}

	switch {
	case startsWith(id, "comol"),
		startsWith(id, "disqus_thread"),
		startsWith(id, "dsq_comments"):
	default:
		return false
	}

	return true
}

// `.//*[(self::div or self::section)][starts-with(@id, 'social') or contains(@class, 'comment')]`,
func commentsRule4(n *html.Node) bool {
	id := dom.ID(n)
	class := dom.ClassName(n)
	tagName := dom.TagName(n)

	switch tagName {
	case "div", "section":
	default:
		return false
	}

	switch {
	case startsWith(id, "social"),
		contains(class, "comment"):
	default:
		return false
	}

	return true
}
