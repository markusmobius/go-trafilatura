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

func discardedContentRule1(n *html.Node) bool {
	id := dom.ID(n)
	class := dom.ClassName(n)

	switch {
	case strings.Contains(id, "footer"),
		strings.Contains(class, "footer"),
		strings.Contains(id, "bottom"),
		strings.Contains(class, "bottom"):
		return true
	default:
		return false
	}
}

func discardedContentRule2(n *html.Node) bool {
	id := dom.ID(n)
	class := dom.ClassName(n)
	tagName := dom.TagName(n)
	role := dom.GetAttribute(n, "role")

	switch tagName {
	case "div", "section", "span", "p",
		"ul", "ol", "dl", // list
		"dd", "dt", "li": // item
	default:
		return false
	}

	switch {
	case strings.Contains(id, "related"),
		strings.Contains(strings.ToLower(class), "related"),
		strings.Contains(id, "viral"),
		strings.Contains(class, "viral"),
		strings.HasPrefix(id, "shar"),
		strings.HasPrefix(class, "shar"),
		strings.Contains(class, "share-"),
		strings.Contains(id, "social"),
		strings.Contains(class, "social"),
		strings.Contains(class, "sociable"),
		strings.Contains(id, "syndication"),
		strings.Contains(class, "syndication"),
		strings.HasPrefix(id, "jp-"),
		strings.HasPrefix(id, "dpsp-content"),
		strings.Contains(id, "teaser"),
		strings.Contains(strings.ToLower(class), "teaser"),
		strings.Contains(id, "newsletter"),
		strings.Contains(class, "newsletter"),
		strings.Contains(id, "cookie"),
		strings.Contains(class, "cookie"),
		strings.Contains(id, "tags"),
		strings.Contains(class, "tags"),
		strings.Contains(id, "sidebar"),
		strings.Contains(class, "sidebar"),
		strings.Contains(id, "banner"),
		strings.Contains(class, "banner"),
		strings.Contains(class, "meta"),
		strings.Contains(id, "menu"),
		strings.Contains(class, "menu"),
		strings.HasPrefix(id, "nav"),
		strings.HasPrefix(class, "nav"),
		strings.Contains(id, "navigation"),
		strings.Contains(strings.ToLower(class), "navigation"),
		strings.Contains(role, "navigation"),
		strings.Contains(class, "navbox"),
		strings.HasPrefix(class, "post-nav"),
		strings.Contains(id, "breadcrumb"),
		strings.Contains(class, "breadcrumb"),
		strings.Contains(id, "bread-crumb"),
		strings.Contains(class, "bread-crumb"),
		strings.Contains(id, "author"),
		strings.Contains(class, "author"),
		strings.Contains(id, "button"),
		strings.Contains(class, "button"),
		strings.Contains(id, "caption"),
		strings.Contains(class, "caption"),
		strings.Contains(strings.ToLower(class), "byline"),
		strings.Contains(class, "rating"),
		strings.HasPrefix(class, "widget"),
		strings.Contains(class, "attachment"),
		strings.Contains(class, "timestamp"),
		strings.Contains(class, "user-info"),
		strings.Contains(class, "user-profile"),
		strings.Contains(class, "-ad-"),
		strings.Contains(class, "-icon"),
		strings.Contains(class, "article-infos"),
		strings.Contains(strings.ToLower(class), "infoline"):
	default:
		return false
	}

	return true
}

// discardedContentRule3 handle comment debris
func discardedContentRule3(n *html.Node) bool {
	id := dom.ID(n)
	class := dom.ClassName(n)

	switch {
	case class == "comments-title",
		strings.Contains(class, "comments-title"),
		strings.Contains(class, "nocomments"),
		strings.HasPrefix(id, "reply-"),
		strings.HasPrefix(class, "reply-"),
		strings.Contains(class, "-reply-"),
		strings.Contains(class, "message"),
		strings.Contains(id, "akismet"),
		strings.Contains(class, "akismet"):
		return true
	default:
		return false
	}
}

// discardedContentRule4 handle hidden nodes
func discardedContentRule4(n *html.Node) bool {
	id := dom.ID(n)
	class := dom.ClassName(n)
	style := dom.GetAttribute(n, "style")

	switch {
	case strings.HasPrefix(class, "hide-"),
		strings.Contains(class, "hide-print"),
		strings.Contains(id, "hidden"),
		strings.Contains(style, "hidden"),
		dom.HasAttribute(n, "hidden"),
		strings.Contains(class, "noprint"),
		strings.Contains(style, "display:none"),
		strings.Contains(class, " hidden"):
		return true
	default:
		return false
	}
}
