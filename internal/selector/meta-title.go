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
	case strings.Contains(class, "post-title"),
		strings.Contains(class, "entry-title"),
		strings.Contains(class, "headline"),
		strings.Contains(id, "headline"),
		strings.Contains(itemProp, "headline"),
		strings.Contains(class, "post__title"):
	default:
		return false
	}

	return true
}

func metaTitleRule2(n *html.Node) bool {
	switch dom.ClassName(n) {
	case "entry-title", "post-title":
		return true
	default:
		return false
	}
}

func metaTitleRule3(n *html.Node) bool {
	id := dom.ID(n)
	class := dom.ClassName(n)
	tagName := dom.TagName(n)

	switch tagName {
	case "h1":
	default:
		return false
	}

	switch {
	case strings.Contains(class, "title"),
		strings.Contains(id, "title"):
	default:
		return false
	}

	return true
}
