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
// or contains(@class, "related-stories") or contains(@class, "most-popular")
// or contains(@class, "mol-factbox") or starts-with(@class, "ZendeskForm")
// or contains(@class, "message-container") or contains(@id, "message_container")
// or contains(@class, "yin") or contains(@class, "zlylin") or
// contains(@class, "xg1") or contains(@id, "bmdh")
// or @data-lp-replacement-content or @data-testid]`,
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
	case contains(lower(id), "footer"),
		contains(lower(class), "footer"),
		contains(id, "related"),
		contains(lower(class), "related"),
		contains(id, "viral"),
		contains(class, "viral"),
		startsWith(id, "shar"),
		startsWith(class, "shar"),
		contains(class, "share-"),
		contains(id, "social"),
		contains(class, "social"),
		contains(class, "sociable"),
		contains(id, "syndication"),
		contains(class, "syndication"),
		startsWith(id, "jp-"),
		startsWith(id, "dpsp-content"),
		contains(class, "embedded"),
		contains(class, "embed"),
		contains(id, "newsletter"),
		contains(class, "newsletter"),
		contains(class, "subnav"),
		contains(id, "cookie"),
		contains(class, "cookie"),
		contains(id, "tags"),
		contains(class, "tags"),
		contains(id, "sidebar"),
		contains(class, "sidebar"),
		contains(id, "banner"),
		contains(class, "banner"),
		contains(class, "meta"),
		contains(id, "menu"),
		contains(class, "menu"),
		contains(lower(id), "nav"),
		contains(lower(role), "nav"),
		startsWith(class, "nav"),
		contains(lower(class), "navigation"),
		contains(class, "navbar"),
		contains(class, "navbox"),
		startsWith(class, "post-nav"),
		contains(id, "breadcrumb"),
		contains(class, "breadcrumb"),
		contains(id, "bread-crumb"),
		contains(class, "bread-crumb"),
		contains(id, "author"),
		contains(class, "author"),
		contains(id, "button"),
		contains(class, "button"),
		contains(lower(class), "byline"),
		contains(class, "rating"),
		startsWith(class, "widget"),
		contains(class, "attachment"),
		contains(class, "timestamp"),
		contains(class, "user-info"),
		contains(class, "user-profile"),
		contains(class, "-ad-"),
		contains(class, "-icon"),
		contains(class, "article-infos"),
		contains(lower(class), "infoline"),
		contains(dataComponent, "MostPopularStories"),
		contains(class, "outbrain"),
		contains(class, "taboola"),
		contains(class, "criteo"),
		contains(class, "options"),
		contains(class, "consent"),
		contains(class, "modal-content"),
		contains(class, "paid-content"),
		contains(class, "paidcontent"),
		contains(id, "premium-"),
		contains(id, "paywall"),
		contains(class, "obfuscated"),
		contains(class, "blurred"),
		contains(class, " ad "),
		contains(class, "next-post"),
		contains(class, "related-stories"),
		contains(class, "most-popular"),
		contains(class, "mol-factbox"),
		startsWith(class, "ZendeskForm"),
		contains(class, "message-container"),
		contains(id, "message_container"),
		contains(class, "yin"),
		contains(class, "zlylin"),
		contains(class, "xg1"),
		contains(id, "bmdh"),
		dom.HasAttribute(n, "data-lp-replacement-content"),
		dom.HasAttribute(n, "data-testid"):
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
// or contains(@style, "display:none") or contains(@style, "display: none") or contains(@class, " hidden") or @aria-hidden="true"
// or contains(@class, "notloaded")]`,
func overallDiscardedContentRule2(n *html.Node) bool {
	id := dom.ID(n)
	class := dom.ClassName(n)
	style := dom.GetAttribute(n, "style")
	hidden := dom.GetAttribute(n, "hidden")
	ariaHidden := dom.GetAttribute(n, "aria-hidden")

	switch {
	case class == "comments-title",
		contains(class, "comments-title"),
		contains(class, "nocomments"),
		startsWith(id, "reply-"),
		startsWith(class, "reply-"),
		contains(class, "-reply-"),
		contains(class, "message"),
		contains(id, "akismet"),
		contains(class, "akismet"),
		startsWith(class, "hide-"),
		contains(class, "hide-print"),
		contains(id, "hidden"),
		contains(style, "hidden"),
		contains(hidden, "hidden"),
		contains(class, "noprint"),
		contains(style, "display:none"),
		contains(style, "display: none"),
		contains(class, " hidden"),
		ariaHidden == "true",
		contains(class, "notloaded"):
	default:
		return false
	}

	return true

}
