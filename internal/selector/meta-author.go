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

func metaAuthorRule1(n *html.Node) bool {
	class := dom.ClassName(n)
	rel := dom.GetAttribute(n, "rel")
	tagName := dom.TagName(n)

	switch tagName {
	case "a", "address", "link", "p", "span":
	case "author":
		return true
	default:
		return false
	}

	switch {
	case rel == "me",
		rel == "author",
		class == "author":
	default:
		return false
	}

	return true
}

func metaAuthorRule2(n *html.Node) bool {
	class := dom.ClassName(n)
	itemProp := dom.GetAttribute(n, "itemprop")
	tagName := dom.TagName(n)

	switch tagName {
	case "a", "span":
	default:
		return false
	}

	switch {
	case strings.Contains(class, "author"),
		strings.Contains(class, "authors"),
		strings.Contains(class, "posted-by"),
		strings.Contains(itemProp, "author"):
	default:
		return false
	}

	return true
}

func metaAuthorRule3(n *html.Node) bool {
	class := dom.ClassName(n)
	tagName := dom.TagName(n)

	switch tagName {
	case "a", "div", "p", "span":
	default:
		return false
	}

	switch {
	case strings.Contains(class, "byline"):
	default:
		return false
	}

	return true
}

func metaAuthorRule4(n *html.Node) bool {
	class := dom.ClassName(n)

	switch {
	case strings.Contains(class, "author"),
		strings.Contains(class, "screenname"):
		return true
	default:
		return false
	}
}
