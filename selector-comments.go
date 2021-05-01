package trafilatura

import (
	"strings"

	"github.com/go-shiori/dom"
	"golang.org/x/net/html"
)

type selectorRule func(*html.Node) bool

var commentSelectorRules = []selectorRule{
	commentSelectorRule1,
	commentSelectorRule2,
	commentSelectorRule3,
	commentSelectorRule4,
}

var discardedCommentSelectorRules = []selectorRule{
	discardedCommentSelectorRule1,
	discardedCommentSelectorRule2,
	discardedCommentSelectorRule3,
}

func commentSelectorRule1(n *html.Node) bool {
	id := dom.ID(n)
	class := dom.ClassName(n)
	tagName := dom.TagName(n)

	switch tagName {
	case "div", "section", "ol", "ul", "dl":
	default:
		return false
	}

	switch {
	case strings.Contains(id, "commentlist"),
		strings.Contains(class, "commentlist"),
		strings.Contains(class, "comment-page"),
		strings.Contains(class, "comment-list"),
		strings.Contains(class, "comments-list"),
		strings.Contains(class, "comments-content"):
	default:
		return false
	}

	return true
}

func commentSelectorRule2(n *html.Node) bool {
	id := dom.ID(n)
	class := dom.ClassName(n)
	tagName := dom.TagName(n)

	switch tagName {
	case "div", "section", "ol", "ul", "dl":
	default:
		return false
	}

	switch {
	case strings.HasPrefix(id, "comments"),
		strings.HasPrefix(class, "comments"),
		strings.HasPrefix(class, "Comments"),
		strings.HasPrefix(id, "comment-"),
		strings.HasPrefix(class, "comment-"),
		strings.Contains(class, "article-comments"):
	default:
		return false
	}

	return true
}

func commentSelectorRule3(n *html.Node) bool {
	id := dom.ID(n)
	tagName := dom.TagName(n)

	switch tagName {
	case "div", "section", "ol", "ul", "dl":
	default:
		return false
	}

	switch {
	case strings.HasPrefix(id, "comol"),
		strings.HasPrefix(id, "disqus_thread"),
		strings.HasPrefix(id, "dsq_comments"):
	default:
		return false
	}

	return true
}

func commentSelectorRule4(n *html.Node) bool {
	id := dom.ID(n)
	class := dom.ClassName(n)
	tagName := dom.TagName(n)

	switch tagName {
	case "div", "section":
	default:
		return false
	}

	switch {
	case strings.HasPrefix(id, "social"),
		strings.Contains(class, "comment"):
	default:
		return false
	}

	return true
}

func discardedCommentSelectorRule1(n *html.Node) bool {
	id := dom.ID(n)
	tagName := dom.TagName(n)

	switch tagName {
	case "div", "section":
	default:
		return false
	}

	if !strings.HasPrefix(id, "respond") {
		return false
	}

	return true
}

func discardedCommentSelectorRule2(n *html.Node) bool {
	tagName := dom.TagName(n)

	switch tagName {
	case "cite", "blockquote", "pre", "q":
		return true
	default:
		return false
	}
}

func discardedCommentSelectorRule3(n *html.Node) bool {
	id := dom.ID(n)
	class := dom.ClassName(n)
	style := dom.GetAttribute(n, "style")

	switch {
	case class == "comments-title",
		strings.Contains(class, "comments-title"),
		strings.Contains(class, "nocomments"),
		strings.HasPrefix(id, "reply-"),
		strings.HasPrefix(class, "reply-"),
		strings.Contains(class, "-reply-"),
		strings.Contains(class, "akismet"),
		strings.Contains(style, "display:none"):
	default:
		return false
	}

	return true
}
