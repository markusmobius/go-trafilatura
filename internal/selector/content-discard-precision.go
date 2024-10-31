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

var PrecisionDiscardedContent = []Rule{
	precisionDiscardedContentRule1,
	precisionDiscardedContentRule2,
}

// `.//header`,
func precisionDiscardedContentRule1(n *html.Node) bool {
	return dom.TagName(n) == "header"
}

// `.//*[self::div or self::dd or self::dt or self::li or self::ul or self::ol or self::dl or self::p or self::section or self::span][
// contains(@id|@class, "bottom") or
// contains(@id|@class, "link") or
// contains(@style, "border")`,
func precisionDiscardedContentRule2(n *html.Node) bool {
	id := dom.ID(n)
	class := dom.ClassName(n)
	style := dom.GetAttribute(n, "style")
	tagName := dom.TagName(n)
	idClass := id + class

	switch tagName {
	case "div", "dd", "dt", "li", "ul", "ol", "dl", "p", "section", "span":
	default:
		return false
	}

	switch {
	case contains(idClass, "bottom"),
		contains(idClass, "link"),
		contains(style, "border"):
	default:
		return false
	}

	return true
}
