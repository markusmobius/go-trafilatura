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

var MetaAuthor = []Rule{
	metaAuthorRule1,
	metaAuthorRule2,
	metaAuthorRule3,
}

// specific and almost specific
// `//*[self::a or self::address or self::div or self::link or self::p or self::span or self::strong][@rel="author" or @id="author" or @class="author" or @itemprop="author name" or rel="me" or contains(@class, "author-name") or contains(@class, "AuthorName") or contains(@class, "authorName") or contains(@class, "author name") or @data-testid="AuthorCard" or @data-testid="AuthorURL"]|//author`,
func metaAuthorRule1(n *html.Node) bool {
	id := dom.ID(n)
	class := dom.ClassName(n)
	rel := dom.GetAttribute(n, "rel")
	itemProp := dom.GetAttribute(n, "itemprop")
	dataTestID := dom.GetAttribute(n, "data-testid")
	tagName := dom.TagName(n)

	switch tagName {
	case "a", "address", "div", "link", "p", "span", "strong":
	case "author":
		return true
	default:
		return false
	}

	switch {
	case rel == "author",
		id == "author",
		class == "author",
		itemProp == "author name",
		rel == "me",
		contains(class, "author-name"),
		contains(class, "AuthorName"),
		contains(class, "authorName"),
		contains(class, "author name"),
		dataTestID == "AuthorCard",
		dataTestID == "AuthorURL":
	default:
		return false
	}

	return true
}

// almost generic and generic, last ones not common
// `//*[self::a or self::div or self::h3 or self::h4 or self::p or self::span][contains(@class, "author") or contains(@id, "author") or contains(@itemprop, "author") or @class="byline" or contains(@class, "channel-name") or contains(@id, "zuozhe") or contains(@class, "zuozhe") or contains(@id, "bianji") or contains(@class, "bianji") or contains(@id, "xiaobian") or contains(@class, "xiaobian") or contains(@class, "submitted-by") or contains(@class, "posted-by") or @class="username" or @class="byl" or @class="BBL" or contains(@class, "journalist-name")]`
func metaAuthorRule2(n *html.Node) bool {
	id := dom.ID(n)
	class := dom.ClassName(n)
	itemProp := dom.GetAttribute(n, "itemprop")
	tagName := dom.TagName(n)

	switch tagName {
	case "a", "div", "h3", "h4", "p", "span":
	default:
		return false
	}

	switch {
	case contains(class, "author"),
		contains(id, "author"),
		contains(itemProp, "author"),
		class == "byline",
		contains(class, "channel-name"),
		contains(id, "zuozhe"),
		contains(class, "zuozhe"),
		contains(id, "bianji"),
		contains(class, "bianji"),
		contains(id, "xiaobian"),
		contains(class, "xiaobian"),
		contains(class, "submitted-by"),
		contains(class, "posted-by"),
		class == "username",
		class == "byl",
		class == "BBL",
		contains(class, "journalist-name"):
	default:
		return false
	}

	return true
}

// last resort: any element
// `//*[contains(translate(@id, "A", "a"), "author") or contains(translate(@class, "A", "a"), "author") or contains(@class, "screenname") or contains(@data-component, "Byline") or contains(@itemprop, "author") or contains(@class, "writer") or contains(translate(@class, "B", "b"), "byline")]`
func metaAuthorRule3(n *html.Node) bool {
	id := dom.ID(n)
	class := dom.ClassName(n)
	dataComponent := dom.GetAttribute(n, "data-component")
	itemProp := dom.GetAttribute(n, "itemprop")

	switch {
	case contains(lower(id), "author"),
		contains(lower(class), "author"),
		contains(class, "screenname"),
		contains(lower(dataComponent), "byline"),
		contains(itemProp, "author"),
		contains(class, "writer"),
		contains(lower(class), "byline"):
	default:
		return false
	}

	return true
}
