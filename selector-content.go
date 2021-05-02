package trafilatura

import (
	"strings"

	"github.com/go-shiori/dom"
	"golang.org/x/net/html"
)

func contentSelectorRule1(n *html.Node) bool {
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

func contentSelectorRule2(n *html.Node) bool {
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

func contentSelectorRule3(n *html.Node) bool {
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

func contentSelectorRule4(n *html.Node) bool {
	tagName := dom.TagName(n)
	return tagName == "article"
}

func contentSelectorRule5(n *html.Node) bool {
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

func contentSelectorRule6(n *html.Node) bool {
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

func contentSelectorRule7(n *html.Node) bool {
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

func discardedContentSelectorRule1(n *html.Node) bool {
	id := dom.ID(n)
	class := dom.ClassName(n)

	switch {
	case strings.Contains(id, "footer"),
		strings.Contains(class, "footer"),
		strings.Contains(id, "bottom"),
		strings.Contains(class, "bottom"):
		return true
	default:
		return false
	}
}

func discardedContentSelectorRule2(n *html.Node) bool {
	id := dom.ID(n)
	class := dom.ClassName(n)
	tagName := dom.TagName(n)
	role := dom.GetAttribute(n, "role")

	switch tagName {
	case "div", "section", "span",
		"ul", "ol", "dl", // list
		"dd", "dt", "li": // item
	default:
		return false
	}

	switch {
	case strings.Contains(id, "related"),
		strings.Contains(strings.ToLower(class), "related"),
		strings.Contains(id, "viral"),
		strings.Contains(class, "viral"),
		strings.HasPrefix(id, "shar"),
		strings.HasPrefix(class, "shar"),
		strings.Contains(class, "share-"),
		strings.Contains(id, "social"),
		strings.Contains(class, "social"),
		strings.Contains(class, "sociable"),
		strings.Contains(id, "syndication"),
		strings.Contains(class, "syndication"),
		strings.HasPrefix(id, "jp-"),
		strings.HasPrefix(id, "dpsp-content"),
		strings.Contains(id, "teaser"),
		strings.Contains(strings.ToLower(class), "teaser"),
		strings.Contains(id, "newsletter"),
		strings.Contains(class, "newsletter"),
		strings.Contains(id, "cookie"),
		strings.Contains(class, "cookie"),
		strings.Contains(id, "tags"),
		strings.Contains(class, "tags"),
		strings.Contains(id, "sidebar"),
		strings.Contains(class, "sidebar"),
		strings.Contains(id, "banner"),
		strings.Contains(class, "banner"),
		strings.Contains(class, "meta"),
		strings.Contains(id, "menu"),
		strings.Contains(class, "menu"),
		strings.HasPrefix(id, "nav"),
		strings.HasPrefix(class, "nav"),
		strings.Contains(id, "navigation"),
		strings.Contains(strings.ToLower(class), "navigation"),
		strings.Contains(role, "navigation"),
		strings.Contains(class, "navbox"),
		strings.HasPrefix(class, "post-nav"),
		strings.Contains(id, "breadcrumb"),
		strings.Contains(class, "breadcrumb"),
		strings.Contains(id, "bread-crumb"),
		strings.Contains(class, "bread-crumb"),
		strings.Contains(id, "author"),
		strings.Contains(class, "author"),
		strings.Contains(id, "button"),
		strings.Contains(class, "button"),
		strings.Contains(id, "caption"),
		strings.Contains(class, "caption"),
		strings.Contains(strings.ToLower(class), "byline"),
		strings.Contains(class, "rating"),
		strings.HasPrefix(class, "widget"),
		strings.Contains(class, "attachment"),
		strings.Contains(class, "timestamp"),
		strings.Contains(class, "user-info"),
		strings.Contains(class, "user-profile"),
		strings.Contains(class, "-ad-"),
		strings.Contains(class, "-icon"),
		strings.Contains(class, "article-infos"),
		strings.Contains(strings.ToLower(class), "infoline"):
	default:
		return false
	}

	return true
}

func discardedContentSelectorRule3(n *html.Node) bool {
	// handle comment debris
	id := dom.ID(n)
	class := dom.ClassName(n)

	switch {
	case class == "comments-title",
		strings.Contains(class, "comments-title"),
		strings.Contains(class, "nocomments"),
		strings.HasPrefix(id, "reply-"),
		strings.HasPrefix(class, "reply-"),
		strings.Contains(class, "-reply-"),
		strings.Contains(class, "message"),
		strings.Contains(id, "akismet"),
		strings.Contains(class, "akismet"):
		return true
	default:
		return false
	}
}

func discardedContentSelectorRule4(n *html.Node) bool {
	// handle hidden nodes
	id := dom.ID(n)
	class := dom.ClassName(n)
	style := dom.GetAttribute(n, "style")

	switch {
	case strings.HasPrefix(class, "hide-"),
		strings.Contains(class, "hide-print"),
		strings.Contains(id, "hidden"),
		strings.Contains(style, "hidden"),
		dom.HasAttribute(n, "hidden"),
		strings.Contains(class, "noprint"),
		strings.Contains(style, "display:none"),
		strings.Contains(class, " hidden"):
		return true
	default:
		return false
	}
}
