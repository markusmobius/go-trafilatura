package selector

import (
	"strings"

	"github.com/go-shiori/dom"
	"golang.org/x/net/html"
)

func contentRule1(n *html.Node) bool {
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
		strings.Contains(class, "story-body"),
		id == "article",
		class == "post",
		class == "entry":
	default:
		return false
	}

	return true
}

func contentRule2(n *html.Node) bool {
	class := dom.ClassName(n)
	tagName := dom.TagName(n)

	switch tagName {
	case "article", "div", "main", "section":
	default:
		return false
	}

	switch {
	case strings.Contains(class, "post-text"),
		strings.Contains(class, "post_text"),
		strings.Contains(class, "post-body"),
		strings.Contains(class, "post-entry"),
		strings.Contains(class, "postentry"),
		strings.Contains(class, "post-content"),
		strings.Contains(class, "post_content"),
		strings.Contains(class, "postcontent"),
		strings.Contains(class, "postContent"),
		strings.Contains(class, "article-text"),
		strings.Contains(class, "articletext"),
		strings.Contains(class, "articleText"):
	default:
		return false
	}

	return true
}

func contentRule3(n *html.Node) bool {
	id := dom.ID(n)
	class := dom.ClassName(n)
	tagName := dom.TagName(n)
	itemProp := dom.GetAttribute(n, "itemprop")

	switch tagName {
	case "article", "div", "main", "section":
	default:
		return false
	}

	switch {
	case strings.Contains(id, "entry-content"),
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
		id == "articleContent",
		strings.Contains(class, "ArticleContent"),
		strings.Contains(class, "page-content"),
		strings.Contains(class, "text-content"),
		strings.Contains(class, "content__body"),
		strings.Contains(id, "body-text"),
		strings.Contains(class, "body-text"):
	default:
		return false
	}

	return true
}

func contentRule4(n *html.Node) bool {
	tagName := dom.TagName(n)
	return tagName == "article"
}

func contentRule5(n *html.Node) bool {
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
		class == "text",
		class == "cell",
		id == "story",
		class == "story",
		strings.Contains(strings.ToLower(class), "fulltext"):
	default:
		return false
	}

	return true
}

func contentRule6(n *html.Node) bool {
	id := dom.ID(n)
	class := dom.ClassName(n)
	tagName := dom.TagName(n)

	switch tagName {
	case "article", "div", "main", "section":
	default:
		return false
	}

	switch {
	case strings.Contains(id, "main-content"),
		strings.Contains(class, "main-content"),
		strings.Contains(strings.ToLower(class), "page-content"):
	default:
		return false
	}

	return true
}

func contentRule7(n *html.Node) bool {
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
	case strings.HasPrefix(id, "main"),
		strings.HasPrefix(class, "main"),
		strings.HasPrefix(role, "main"):
	default:
		return false
	}

	return true
}
