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

var RemovedComments = []Rule{
	removedCommentsRule1,
}

// `.//*[self::div or self::ol or self::ul or self::dl or self::section][
// starts-with(translate(@id, "C","c"), 'comment') or
// starts-with(translate(@class, "C","c"), 'comment') or
// contains(@class, 'article-comments') or contains(@class, 'post-comments')
// or starts-with(@id, 'comol') or starts-with(@id, 'disqus_thread')
// or starts-with(@id, 'dsq-comments')]`,
func removedCommentsRule1(n *html.Node) bool {
	id := dom.ID(n)
	class := dom.ClassName(n)
	tagName := dom.TagName(n)

	switch tagName {
	case "div", "ol", "ul", "dl", "section":
	default:
		return false
	}

	switch {
	case startsWith(lower(id), "comment"),
		startsWith(lower(class), "comment"),
		contains(class, "article-comments"),
		contains(class, "post-comments"),
		startsWith(id, "comol"),
		startsWith(id, "disqus_thread"),
		startsWith(id, "dsq-comments"):
	default:
		return false
	}

	return true
}
