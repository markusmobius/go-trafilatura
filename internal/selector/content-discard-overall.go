// This file is part of go-trafilatura, Go package for extracting readable
// content, comments and metadata from a web page. Source available in
// <https://github.com/markusmobius/go-trafilatura>.
//
// Copyright (C) 2021 Markus Mobius
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code in this file is ported from <https://github.com/adbar/trafilatura>
// which available under Apache 2.0 license.

package selector

import (
	"github.com/go-shiori/dom"
	"golang.org/x/net/html"
)

var OverallDiscardedContent = []Rule{
	overallDiscardedContentRule1,
	overallDiscardedContentRule2,
	discardedLegalRules,
}

// navigation + footers, news outlets related posts, sharing, jp-post-flair jp-relatedposts
// paywalls
// `.//*[self::div or self::item or self::ol or self::ul or self::dl or self::p or self::section or self::span][
// contains(translate(@id, "F","f"), "footer") or contains(translate(@class, "F","f"), "footer")
// or contains(@id, "related") or contains(@class, "elated") or
// contains(@id|@class, "viral") or
// starts-with(@id|@class, "shar") or
// contains(@class, "share-") or
// contains(translate(@id, "S", "s"), "share") or
// contains(@id|@class, "social") or contains(@class, "sociable") or
// contains(@id|@class, "syndication") or
// starts-with(@id, "jp-") or starts-with(@id, "dpsp-content") or
// contains(@class, "embedded") or contains(@class, "embed") or
// contains(@id|@class, "newsletter") or
// contains(@class, "subnav") or
// contains(@id|@class, "cookie") or
// contains(@id|@class, "tags") or contains(@class, "tag-list") or
// contains(@id|@class, "sidebar") or
// contains(@id|@class, "banner") or contains(@class, "bar") or
// contains(@class, "meta") or contains(@id, "menu") or contains(@class, "menu") or
// contains(translate(@id, "N", "n"), "nav") or contains(translate(@role, "N", "n"), "nav")
// or starts-with(@class, "nav") or contains(@class, "avigation") or
// contains(@class, "navbar") or contains(@class, "navbox") or starts-with(@class, "post-nav")
// or contains(@id|@class, "breadcrumb") or
// contains(@id|@class, "bread-crumb") or
// contains(@id|@class, "author") or
// contains(@id|@class, "button")
// or contains(translate(@class, "B", "b"), "byline")
// or contains(@class, "rating") or contains(@class, "widget") or
// contains(@class, "attachment") or contains(@class, "timestamp") or
// contains(@class, "user-info") or contains(@class, "user-profile") or
// contains(@class, "-ad-") or contains(@class, "-icon")
// or contains(@class, "article-infos") or
// contains(@class, "nfoline")
// or contains(@data-component, "MostPopularStories")
// or contains(@class, "outbrain") or contains(@class, "taboola")
// or contains(@class, "criteo") or contains(@class, "options") or contains(@class, "expand")
// or contains(@class, "consent") or contains(@class, "modal-content")
// or contains(@class, " ad ") or contains(@class, "permission")
// or contains(@class, "next-") or contains(@class, "-stories")
// or contains(@class, "most-popular") or contains(@class, "mol-factbox")
// or starts-with(@class, "ZendeskForm") or contains(@id|@class, "message-container")
// or contains(@class, "yin") or contains(@class, "zlylin")
// or contains(@class, "xg1") or contains(@id, "bmdh")
// or contains(@class, "slide") or contains(@class, "viewport")
// or @data-lp-replacement-content
// or contains(@id, "premium") or contains(@class, "overlay")
// or contains(@class, "paid-content") or contains(@class, "paidcontent")
// or contains(@class, "obfuscated") or contains(@class, "blurred")]`,
func overallDiscardedContentRule1(n *html.Node) bool {
	id := dom.ID(n)
	class := dom.ClassName(n)
	role := dom.GetAttribute(n, "role")
	dataComponent := dom.GetAttribute(n, "data-component")
	tagName := dom.TagName(n)
	idClass := id + class

	switch tagName {
	case "div", "dd", "dt", "li", "ul", "ol", "dl", "p", "section", "span":
	default:
		return false
	}

	switch {
	case contains(lower(id), "footer"),
		contains(lower(class), "footer"),
		contains(id, "related"),
		contains(class, "elated"),
		contains(idClass, "viral"),
		startsWith(idClass, "shar"),
		contains(class, "share-"),
		contains(lower(id), "share"),
		contains(idClass, "social"),
		contains(class, "sociable"),
		contains(idClass, "syndication"),
		startsWith(id, "jp-"),
		startsWith(id, "dpsp-content"),
		contains(class, "embedded"),
		contains(class, "embed"),
		contains(idClass, "newsletter"),
		contains(class, "subnav"),
		contains(idClass, "cookie"),
		contains(idClass, "tags"),
		contains(class, "tag-list"),
		contains(idClass, "sidebar"),
		contains(idClass, "banner"),
		contains(class, "bar"),
		contains(class, "meta"),
		contains(id, "menu"),
		contains(class, "menu"),
		contains(lower(id), "nav"),
		contains(lower(role), "nav"),
		startsWith(class, "nav"),
		contains(class, "avigation"),
		contains(class, "navbar"),
		contains(class, "navbox"),
		startsWith(class, "post-nav"),
		contains(idClass, "breadcrumb"),
		contains(idClass, "bread-crumb"),
		contains(idClass, "author"),
		contains(idClass, "button"),
		contains(lower(class), "byline"),
		contains(class, "rating"),
		contains(class, "widget"),
		contains(class, "attachment"),
		contains(class, "timestamp"),
		contains(class, "user-info"),
		contains(class, "user-profile"),
		contains(class, "-ad-"),
		contains(class, "-icon"),
		contains(class, "article-infos"),
		contains(class, "nfoline"),
		contains(dataComponent, "MostPopularStories"),
		contains(class, "outbrain"),
		contains(class, "taboola"),
		contains(class, "criteo"),
		contains(class, "options"),
		contains(class, "expand"),
		contains(class, "consent"),
		contains(class, "modal-content"),
		contains(class, " ad "),
		contains(class, "permission"),
		contains(class, "next-"),
		contains(class, "-stories"),
		contains(class, "most-popular"),
		contains(class, "mol-factbox"),
		startsWith(class, "ZendeskForm"),
		contains(idClass, "message-container"),
		contains(class, "yin"),
		contains(class, "zlylin"),
		contains(class, "xg1"),
		contains(id, "bmdh"),
		contains(class, "slide"),
		contains(class, "viewport"),
		dom.HasAttribute(n, "data-lp-replacement-content"),
		contains(id, "premium"),
		contains(class, "overlay"),
		contains(class, "paid-content"),
		contains(class, "paidcontent"),
		contains(class, "obfuscated"),
		contains(class, "blurred"):
	default:
		return false
	}

	return true
}

// comment debris + hidden parts
// `.//*[@class="comments-title" or contains(@class, "comments-title") or
// contains(@class, "nocomments") or starts-with(@id|@class, "reply-") or
// contains(@class, "-reply-") or contains(@class, "message") or contains(@id, "reader-comments")
// or contains(@id, "akismet") or contains(@class, "akismet") or contains(@class, "suggest-links") or
// starts-with(@class, "hide-") or contains(@class, "-hide-") or contains(@class, "hide-print") or
// contains(@id|@style, "hidden") or contains(@class, " hidden") or contains(@class, " hide")
// or contains(@class, "noprint") or contains(@style, "display:none") or contains(@style, "display: none")
// or @aria-hidden="true" or contains(@class, "notloaded")]`,
func overallDiscardedContentRule2(n *html.Node) bool {
	id := dom.ID(n)
	class := dom.ClassName(n)
	style := dom.GetAttribute(n, "style")
	ariaHidden := dom.GetAttribute(n, "aria-hidden")
	idClass := id + class
	idStyle := id + style

	switch {
	case class == "comments-title",
		contains(class, "comments-title"),
		contains(class, "nocomments"),
		startsWith(idClass, "reply-"),
		contains(class, "-reply-"),
		contains(class, "message"),
		contains(id, "reader-comments"),
		contains(id, "akismet"),
		contains(class, "akismet"),
		contains(class, "suggest-links"),
		startsWith(class, "hide-"),
		contains(class, "-hide-"),
		contains(class, "hide-print"),
		contains(idStyle, "hidden"),
		contains(class, " hidden"),
		contains(class, " hide"),
		contains(class, "noprint"),
		contains(style, "display:none"),
		contains(style, "display: none"),
		ariaHidden == "true",
		contains(class, "notloaded"):
	default:
		return false
	}

	return true

}

// discardedLegalRules filters out cookie-consent banners, privacy footers
// and similar legal boiler-plate.
//
// It returns true when the node (or its attributes) match any of the
// heuristics; selector.PruneUnwantedSections will then remove the node.
func discardedLegalRules(n *html.Node) bool {
	tag := dom.TagName(n)
	id := dom.ID(n)
	class := dom.ClassName(n)
	src := dom.GetAttribute(n, "src")
	idClass := id + class
	idClassLower := lower(idClass) // helper already defined in selector utils

	// ------------------------------------------------------------------
	// 1. Script tags that load cookielaw / consent libraries
	// ------------------------------------------------------------------
	if tag == "script" && contains(lower(src), "cookielaw") {
		return true
	}

	// ------------------------------------------------------------------
	// 2. Plain footer element (very often just legal text)
	// ------------------------------------------------------------------
	if tag == "footer" {
		return true
	}

	// ------------------------------------------------------------------
	// 3. Known cookie / consent SDK containers
	// ------------------------------------------------------------------
	if id == "onetrust-consent-sdk" || startsWith(id, "ot-sdk") {
		return true
	}

	// ------------------------------------------------------------------
	// 4. Heuristic keywords in id / class
	// ------------------------------------------------------------------
	switch {
	case contains(idClassLower, "cookie"),
		contains(idClassLower, "consent"),
		contains(idClassLower, "privacy"),
		contains(idClassLower, "gdpr"),
		contains(idClassLower, "legal"),
		contains(idClassLower, "optanon"),
		contains(idClassLower, "cmp"), // IAB / TCF CMPs
		contains(idClassLower, "truste"),
		contains(idClassLower, "evidon"),
		contains(idClassLower, "onetrust"):
		return true
	}

	return false
}
