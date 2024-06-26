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

var Content = []Rule{
	contentRule1,
	contentRule2,
	contentRule3,
	contentRule4,
	contentRule5,
}

// `.//*[(self::article or self::div or self::main or self::section)][
// @class="post" or @class="entry" or
// contains(@class, "post-text") or contains(@class, "post_text") or
// contains(@class, "post-body") or contains(@class, "post-entry") or contains(@class, "postentry") or
// contains(@class, "post-content") or contains(@class, "post_content") or
// contains(@class, "postcontent") or contains(@class, "postContent") or contains(@class, "post_inner_wrapper") or
// contains(@class, "article-text") or contains(@class, "articletext") or contains(@class, "articleText")
// or contains(@id, "entry-content") or
// contains(@class, "entry-content") or contains(@id, "article-content") or
// contains(@class, "article-content") or contains(@id, "article__content") or
// contains(@class, "article__content") or contains(@id, "article-body") or
// contains(@class, "article-body") or contains(@id, "article__body") or
// contains(@class, "article__body") or @itemprop="articleBody" or
// contains(translate(@id, "B", "b"), "articlebody") or contains(translate(@class, "B", "b"), "articleBody")
// or @id="articleContent" or contains(@class, "ArticleContent") or
// contains(@class, "page-content") or contains(@class, "text-content") or
// contains(@id, "body-text") or contains(@class, "body-text") or
// contains(@class, "article__container") or contains(@id, "art-content") or contains(@class, "art-content")][1]`,
func contentRule1(n *html.Node) bool {
	id := dom.ID(n)
	class := dom.ClassName(n)
	itemProp := dom.GetAttribute(n, "itemprop")
	tagName := dom.TagName(n)

	switch tagName {
	case "article", "div", "main", "section":
	default:
		return false
	}

	switch {
	case
		class == "post",
		class == "entry",
		contains(class, "post-text"),
		contains(class, "post_text"),
		contains(class, "post-body"),
		contains(class, "post-entry"),
		contains(class, "postentry"),
		contains(class, "post-content"),
		contains(class, "post_content"),
		contains(lower(class), "postcontent"),
		contains(class, "post_inner_wrapper"),
		contains(class, "article-text"),
		contains(lower(class), "articletext"),
		contains(id, "entry-content"),
		contains(class, "entry-content"),
		contains(id, "article-content"),
		contains(class, "article-content"),
		contains(id, "article__content"),
		contains(class, "article__content"),
		contains(id, "article-body"),
		contains(class, "article-body"),
		contains(id, "article__body"),
		contains(class, "article__body"),
		itemProp == "articleBody",
		contains(lower(id), "articlebody"),
		contains(lower(class), "articlebody"),
		id == "articleContent",
		contains(class, "ArticleContent"),
		contains(class, "page-content"),
		contains(class, "text-content"),
		contains(id, "body-text"),
		contains(class, "body-text"),
		contains(class, "article__container"),
		contains(id, "art-content"),
		contains(class, "art-content"):
	default:
		return false
	}

	return true
}

// `(.//article)[1]`,
func contentRule2(n *html.Node) bool {
	return dom.TagName(n) == "article"
}

// `(.//*[(self::article or self::div or self::main or self::section)][
// contains(@class, 'post-bodycopy') or
// contains(@class, 'storycontent') or contains(@class, 'story-content') or
// @class='postarea' or @class='art-postcontent' or
// contains(@class, 'theme-content') or contains(@class, 'blog-content') or
// contains(@class, 'section-content') or contains(@class, 'single-content') or
// contains(@class, 'single-post') or
// contains(@class, 'main-column') or contains(@class, 'wpb_text_column') or
// starts-with(@id, 'primary') or starts-with(@class, 'article ') or @class="text" or
// @id="article" or @class="cell" or @id="story" or @class="story" or
// contains(@class, "story-body") or contains(@id, "story-body") or contains(@class, "field-body") or
// contains(translate(@class, "FULTEX","fultex"), "fulltext")]) or
// @role='article'])[1]
func contentRule3(n *html.Node) bool {
	id := dom.ID(n)
	class := dom.ClassName(n)
	tagName := dom.TagName(n)
	role := dom.GetAttribute(n, "role")

	switch tagName {
	case "article", "div", "main", "section":
	default:
		return false
	}

	switch {
	case contains(class, "post-bodycopy"),
		contains(class, "storycontent"),
		contains(class, "story-content"),
		class == "postarea",
		class == "art-postcontent",
		contains(class, "theme-content"),
		contains(class, "blog-content"),
		contains(class, "section-content"),
		contains(class, "single-content"),
		contains(class, "single-post"),
		contains(class, "main-column"),
		contains(class, "wpb_text_column"),
		startsWith(id, "primary"),
		startsWith(class, "article"),
		class == "text",
		id == "article",
		class == "cell",
		id == "story",
		class == "story",
		contains(class, "story-body"),
		contains(id, "story-body"),
		contains(class, "field-body"),
		contains(lower(class), "fulltext"),
		role == "article":
	default:
		return false
	}

	return true
}

// `(.//*[(self::article or self::div or self::main or self::section)][
// contains(@id, "content-main") or contains(@class, "content-main") or contains(@class, "content_main") or
// contains(@id, "content-body") or contains(@class, "content-body") or contains(@id, "contentBody")
// or contains(@class, "content__body") or contains(translate(@id, "CM","cm"), "main-content") or contains(translate(@class, "CM","cm"), "main-content")
// or contains(translate(@class, "CP","cp"), "page-content") or
// @id="content" or @class="content"])[1]`,
func contentRule4(n *html.Node) bool {
	id := dom.ID(n)
	class := dom.ClassName(n)
	tagName := dom.TagName(n)

	switch tagName {
	case "article", "div", "main", "section":
	default:
		return false
	}

	switch {
	case contains(id, "content-main"),
		contains(class, "content-main"),
		contains(class, "content_main"),
		contains(id, "content-body"),
		contains(class, "content-body"),
		contains(id, "contentBody"),
		contains(class, "content__body"),
		contains(lower(id), "main-content"),
		contains(lower(class), "main-content"),
		contains(lower(class), "page-content"),
		id == "content",
		class == "content":
	default:
		return false
	}

	return true
}

// `(.//*[(self::article or self::div or self::section)][starts-with(@class, "main") or starts-with(@id, "main") or starts-with(@role, "main")])[1]|(.//main)[1]`,
func contentRule5(n *html.Node) bool {
	id := dom.ID(n)
	class := dom.ClassName(n)
	tagName := dom.TagName(n)
	role := dom.GetAttribute(n, "role")

	switch tagName {
	case "article", "div", "section":
	case "main":
		return true
	default:
		return false
	}

	switch {
	case startsWith(class, "main"),
		startsWith(id, "main"),
		startsWith(role, "main"):
	default:
		return false
	}

	return true
}
