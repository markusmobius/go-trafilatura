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

func commentsRule1(n *html.Node) bool {
	id := dom.ID(n)
	class := dom.ClassName(n)
	tagName := dom.TagName(n)

	switch tagName {
	case "div", "section", "ol", "ul", "dl":
	default:
		return false
	}

	switch {
	case strings.Contains(id, "commentlist"),
		strings.Contains(class, "commentlist"),
		strings.Contains(class, "comment-page"),
		strings.Contains(class, "comment-list"),
		strings.Contains(class, "comments-list"),
		strings.Contains(class, "comments-content"):
	default:
		return false
	}

	return true
}

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
	case strings.HasPrefix(id, "comments"),
		strings.HasPrefix(class, "comments"),
		strings.HasPrefix(class, "Comments"),
		strings.HasPrefix(id, "comment-"),
		strings.HasPrefix(class, "comment-"),
		strings.Contains(class, "article-comments"):
	default:
		return false
	}

	return true
}

func commentsRule3(n *html.Node) bool {
	id := dom.ID(n)
	tagName := dom.TagName(n)

	switch tagName {
	case "div", "section", "ol", "ul", "dl":
	default:
		return false
	}

	switch {
	case strings.HasPrefix(id, "comol"),
		strings.HasPrefix(id, "disqus_thread"),
		strings.HasPrefix(id, "dsq_comments"):
	default:
		return false
	}

	return true
}

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
	case strings.HasPrefix(id, "social"),
		strings.Contains(class, "comment"):
	default:
		return false
	}

	return true
}
