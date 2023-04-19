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
// contains(@class, "postcontent") or contains(@class, "postContent") or
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
		strings.Contains(class, "post-text"),
		strings.Contains(class, "post_text"),
		strings.Contains(class, "post-body"),
		strings.Contains(class, "post-entry"),
		strings.Contains(class, "postentry"),
		strings.Contains(class, "post-content"),
		strings.Contains(class, "post_content"),
		strings.Contains(strings.ToLower(class), "postcontent"),
		strings.Contains(class, "article-text"),
		strings.Contains(strings.ToLower(class), "articletext"),
		strings.Contains(id, "entry-content"),
		strings.Contains(class, "entry-content"),
		strings.Contains(id, "article-content"),
		strings.Contains(class, "article-content"),
		strings.Contains(id, "article__content"),
		strings.Contains(class, "article__content"),
		strings.Contains(id, "article-body"),
		strings.Contains(class, "article-body"),
		strings.Contains(id, "article__body"),
		strings.Contains(class, "article__body"),
		itemProp == "articleBody",
		strings.Contains(strings.ToLower(id), "articlebody"),
		strings.Contains(strings.ToLower(class), "articlebody"),
		id == "articleContent",
		strings.Contains(class, "ArticleContent"),
		strings.Contains(class, "page-content"),
		strings.Contains(class, "text-content"),
		strings.Contains(id, "body-text"),
		strings.Contains(class, "body-text"),
		strings.Contains(class, "article__container"),
		strings.Contains(id, "art-content"),
		strings.Contains(class, "art-content"):
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
// contains(@class, "story-body") or contains(@class, "field-body") or
// contains(translate(@class, "FULTEX","fultex"), "fulltext")])[1]`,
func contentRule3(n *html.Node) bool {
	id := dom.ID(n)
	class := dom.ClassName(n)
	tagName := dom.TagName(n)

	switch tagName {
	case "article", "div", "main", "section":
	default:
		return false
	}

	switch {
	case strings.Contains(class, "post-bodycopy"),
		strings.Contains(class, "storycontent"),
		strings.Contains(class, "story-content"),
		class == "postarea",
		class == "art-postcontent",
		strings.Contains(class, "theme-content"),
		strings.Contains(class, "blog-content"),
		strings.Contains(class, "section-content"),
		strings.Contains(class, "single-content"),
		strings.Contains(class, "single-post"),
		strings.Contains(class, "main-column"),
		strings.Contains(class, "wpb_text_column"),
		strings.HasPrefix(id, "primary"),
		strings.HasPrefix(class, "article"),
		class == "text",
		id == "article",
		class == "cell",
		id == "story",
		class == "story",
		strings.Contains(class, "story-body"),
		strings.Contains(class, "field-body"),
		strings.Contains(strings.ToLower(class), "fulltext"):
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
	case strings.Contains(id, "content-main"),
		strings.Contains(class, "content-main"),
		strings.Contains(class, "content_main"),
		strings.Contains(id, "content-body"),
		strings.Contains(class, "content-body"),
		strings.Contains(id, "contentBody"),
		strings.Contains(class, "content__body"),
		strings.Contains(strings.ToLower(id), "main-content"),
		strings.Contains(strings.ToLower(class), "main-content"),
		strings.Contains(strings.ToLower(class), "page-content"),
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
	case strings.HasPrefix(class, "main"),
		strings.HasPrefix(id, "main"),
		strings.HasPrefix(role, "main"):
	default:
		return false
	}

	return true
}
