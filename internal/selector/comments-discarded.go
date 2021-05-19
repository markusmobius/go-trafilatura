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

func discardedCommentsRule1(n *html.Node) bool {
	id := dom.ID(n)
	tagName := dom.TagName(n)

	switch tagName {
	case "div", "section":
	default:
		return false
	}

	if !strings.HasPrefix(id, "respond") {
		return false
	}

	return true
}

func discardedCommentsRule2(n *html.Node) bool {
	tagName := dom.TagName(n)

	switch tagName {
	case "cite", "blockquote", "pre", "q":
		return true
	default:
		return false
	}
}

func discardedCommentsRule3(n *html.Node) bool {
	id := dom.ID(n)
	class := dom.ClassName(n)
	style := dom.GetAttribute(n, "style")

	switch {
	case class == "comments-title",
		strings.Contains(class, "comments-title"),
		strings.Contains(class, "nocomments"),
		strings.HasPrefix(id, "reply-"),
		strings.HasPrefix(class, "reply-"),
		strings.Contains(class, "-reply-"),
		strings.Contains(class, "message"),
		strings.Contains(id, "akismet"),
		strings.Contains(class, "akismet"),
		strings.Contains(style, "display:none"):
	default:
		return false
	}

	return true
}
