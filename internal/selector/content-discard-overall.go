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

var OverallDiscardedContent = []Rule{
	overallDiscardedContentRule1,
	overallDiscardedContentRule2,
}

// navigation + footers, news outlets related posts, sharing, jp-post-flair jp-relatedposts
// `.//*[(self::div or self::item or self::ol or self::ul or self::dl
// or self::p or self::section or self::span)][
// contains(translate(@id, "F", "f"), "footer") or contains(translate(@class, "F", "f"), "footer")
// or contains(@id, "related") or contains(translate(@class, "R", "r"), "related") or
// contains(@id, "viral") or contains(@class, "viral") or
// starts-with(@id, "shar") or starts-with(@class, "shar") or
// contains(@class, "share-") or
// contains(@id, "social") or contains(@class, "social") or contains(@class, "sociable") or
// contains(@id, "syndication") or contains(@class, "syndication") or
// starts-with(@id, "jp-") or starts-with(@id, "dpsp-content") or
// contains(@class, "embedded") or contains(@class, "embed")
// or contains(@id, "newsletter") or contains(@class, "newsletter")
// or contains(@class, "subnav") or
// contains(@id, "cookie") or contains(@class, "cookie") or contains(@id, "tags")
// or contains(@class, "tags")  or contains(@id, "sidebar") or
// contains(@class, "sidebar") or contains(@id, "banner") or contains(@class, "banner")
// or contains(@class, "meta") or
// contains(@id, "menu") or contains(@class, "menu") or
// contains(translate(@id, "N", "n"), "nav") or contains(translate(@role, "N", "n"), "nav")
// or starts-with(@class, "nav") or contains(translate(@class, "N", "n"), "navigation") or
// contains(@class, "navbar") or contains(@class, "navbox") or starts-with(@class, "post-nav")
// or contains(@id, "breadcrumb") or contains(@class, "breadcrumb") or
// contains(@id, "bread-crumb") or contains(@class, "bread-crumb") or
// contains(@id, "author") or contains(@class, "author") or
// contains(@id, "button") or contains(@class, "button")
// or contains(translate(@class, "B", "b"), "byline")
// or contains(@class, "rating") or starts-with(@class, "widget") or
// contains(@class, "attachment") or contains(@class, "timestamp") or
// contains(@class, "user-info") or contains(@class, "user-profile") or
// contains(@class, "-ad-") or contains(@class, "-icon")
// or contains(@class, "article-infos") or
// contains(translate(@class, "I", "i"), "infoline")
// or contains(@data-component, "MostPopularStories")
// or contains(@class, "outbrain") or contains(@class, "taboola")
// or contains(@class, "criteo") or contains(@class, "options")
// or contains(@class, "consent") or contains(@class, "modal-content")
// or contains(@class, "paid-content") or contains(@class, "paidcontent")
// or contains(@id, "premium-") or contains(@id, "paywall")
// or contains(@class, "obfuscated") or contains(@class, "blurred")
// or contains(@class, " ad ")
// or contains(@class, "next-post")
// or contains(@class, "message-container") or contains(@id, "message_container")
// or contains(@class, "yin") or contains(@class, "zlylin") or
// contains(@class, "xg1") or contains(@id, "bmdh")
// or @data-lp-replacement-content]`,
func overallDiscardedContentRule1(n *html.Node) bool {
	id := dom.ID(n)
	class := dom.ClassName(n)
	role := dom.GetAttribute(n, "role")
	dataComponent := dom.GetAttribute(n, "data-component")
	tagName := dom.TagName(n)

	switch tagName {
	case "div", "dd", "dt", "li", "ul", "ol", "dl", "p", "section", "span":
	default:
		return false
	}

	switch {
	case strings.Contains(strings.ToLower(id), "footer"),
		strings.Contains(strings.ToLower(class), "footer"),
		strings.Contains(id, "related"),
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
		strings.Contains(class, "embedded"),
		strings.Contains(class, "embed"),
		strings.Contains(id, "newsletter"),
		strings.Contains(class, "newsletter"),
		strings.Contains(class, "subnav"),
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
		strings.Contains(strings.ToLower(id), "nav"),
		strings.Contains(strings.ToLower(role), "nav"),
		strings.HasPrefix(class, "nav"),
		strings.Contains(strings.ToLower(class), "navigation"),
		strings.Contains(class, "navbar"),
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
		strings.Contains(strings.ToLower(class), "infoline"),
		strings.Contains(dataComponent, "MostPopularStories"),
		strings.Contains(class, "outbrain"),
		strings.Contains(class, "taboola"),
		strings.Contains(class, "criteo"),
		strings.Contains(class, "options"),
		strings.Contains(class, "consent"),
		strings.Contains(class, "modal-content"),
		strings.Contains(class, "paid-content"),
		strings.Contains(class, "paidcontent"),
		strings.Contains(id, "premium-"),
		strings.Contains(id, "paywall"),
		strings.Contains(class, "obfuscated"),
		strings.Contains(class, "blurred"),
		strings.Contains(class, " ad "),
		strings.Contains(class, "next-post"),
		strings.Contains(class, "message-container"),
		strings.Contains(id, "message_container"),
		strings.Contains(class, "yin"),
		strings.Contains(class, "zlylin"),
		strings.Contains(class, "xg1"),
		strings.Contains(id, "bmdh"),
		dom.HasAttribute(n, "data-lp-replacement-content"):
	default:
		return false
	}

	return true
}

// comment debris + hidden parts
// `.//*[@class="comments-title" or contains(@class, "comments-title") or
// contains(@class, "nocomments") or starts-with(@id, "reply-") or starts-with(@class, "reply-") or
// contains(@class, "-reply-") or contains(@class, "message")
// or contains(@id, "akismet") or contains(@class, "akismet") or
// starts-with(@class, "hide-") or contains(@class, "hide-print") or contains(@id, "hidden")
// or contains(@style, "hidden") or contains(@hidden, "hidden") or contains(@class, "noprint")
// or contains(@style, "display:none") or contains(@class, " hidden") or @aria-hidden="true"
// or contains(@class, "notloaded")]`,
func overallDiscardedContentRule2(n *html.Node) bool {
	id := dom.ID(n)
	class := dom.ClassName(n)
	style := dom.GetAttribute(n, "style")
	hidden := dom.GetAttribute(n, "hidden")
	ariaHidden := dom.GetAttribute(n, "aria-hidden")

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
		strings.HasPrefix(class, "hide-"),
		strings.Contains(class, "hide-print"),
		strings.Contains(id, "hidden"),
		strings.Contains(style, "hidden"),
		strings.Contains(hidden, "hidden"),
		strings.Contains(class, "noprint"),
		strings.Contains(style, "display:none"),
		strings.Contains(class, " hidden"),
		ariaHidden == "true",
		strings.Contains(class, "notloaded"):
	default:
		return false
	}

	return true

}
