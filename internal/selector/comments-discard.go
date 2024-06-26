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

var DiscardedComments = []Rule{
	discardedCommentsRule1,
	discardedCommentsRule2,
	discardedCommentsRule3,
}

// `.//*[(self::div or self::section)][starts-with(@id, "respond")]`,
func discardedCommentsRule1(n *html.Node) bool {
	id := dom.ID(n)
	tagName := dom.TagName(n)

	switch tagName {
	case "div", "section":
	default:
		return false
	}

	switch {
	case startsWith(id, "respond"):
	default:
		return false
	}

	return true
}

// `.//cite|.//quote`,
func discardedCommentsRule2(n *html.Node) bool {
	tagName := dom.TagName(n)
	return tagName == "cite" || tagName == "quote"
}

// `.//*[@class="comments-title" or contains(@class, "comments-title") or
// contains(@class, "nocomments") or starts-with(@id, "reply-") or
// starts-with(@class, "reply-") or contains(@class, "-reply-") or contains(@class, "message")
// or contains(@class, "signin") or
// contains(@id, "akismet") or contains(@class, "akismet") or contains(@style, "display:none")]`,
func discardedCommentsRule3(n *html.Node) bool {
	id := dom.ID(n)
	class := dom.ClassName(n)
	style := dom.GetAttribute(n, "style")

	switch {
	case class == "comments-title",
		contains(class, "comments-title"),
		contains(class, "nocomments"),
		startsWith(id, "reply-"),
		startsWith(class, "reply-"),
		contains(class, "-reply-"),
		contains(class, "message"),
		contains(class, "signin"),
		contains(id, "akismet"),
		contains(class, "akismet"),
		contains(style, "display:none"):
	default:
		return false
	}

	return true
}
