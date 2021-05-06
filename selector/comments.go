package selector

import (
	"strings"

	"github.com/go-shiori/dom"
	"golang.org/x/net/html"
)

func commentsRule1(n *html.Node) bool {
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

func commentsRule2(n *html.Node) bool {
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

func commentsRule3(n *html.Node) bool {
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

func commentsRule4(n *html.Node) bool {
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
