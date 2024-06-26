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

var PrecisionDiscardedContent = []Rule{
	precisionDiscardedContentRule1,
	precisionDiscardedContentRule2,
}

// `.//header`,
func precisionDiscardedContentRule1(n *html.Node) bool {
	return dom.TagName(n) == "header"
}

// `.//*[(self::div or self::dd or self::dt or self::li or self::ul or self::ol or self::dl or self::p or self::section or self::span)][
// contains(@id, "bottom") or contains(@class, "bottom") or
// contains(@id, "link") or contains(@class, "link")
// or contains(@style, "border")]`,
func precisionDiscardedContentRule2(n *html.Node) bool {
	id := dom.ID(n)
	class := dom.ClassName(n)
	style := dom.GetAttribute(n, "style")
	tagName := dom.TagName(n)

	switch tagName {
	case "div", "dd", "dt", "li", "ul", "ol", "dl", "p", "section", "span":
	default:
		return false
	}

	switch {
	case contains(id, "bottom"),
		contains(class, "bottom"),
		contains(id, "link"),
		contains(class, "link"),
		contains(style, "border"):
	default:
		return false
	}

	return true
}
