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

var DiscardedPaywall = []Rule{
	discardedPaywallRule1,
}

// `.//*[(self::div or self::p)][
// contains(@id, "paywall") or contains(@id, "premium") or
// contains(@class, "paid-content") or contains(@class, "paidcontent") or
// contains(@class, "obfuscated") or contains(@class, "blurred") or
// contains(@class, "restricted") or contains(@class, "overlay")
// ]`,
func discardedPaywallRule1(n *html.Node) bool {
	id := dom.ID(n)
	class := dom.ClassName(n)
	tagName := dom.TagName(n)

	switch tagName {
	case "div", "p":
	default:
		return false
	}

	switch {
	case contains(id, "paywall"),
		contains(id, "premium"),
		contains(class, "paid-content"),
		contains(class, "paidcontent"),
		contains(class, "obfuscated"),
		contains(class, "blurred"),
		contains(class, "restricted"),
		contains(class, "overlay"):
	default:
		return false
	}

	return true
}
