package selector

import (
	"strings"

	"github.com/go-shiori/dom"
	"golang.org/x/net/html"
)

func discardedCommentsRule1(n *html.Node) bool {
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

func discardedCommentsRule2(n *html.Node) bool {
	tagName := dom.TagName(n)

	switch tagName {
	case "cite", "blockquote", "pre", "q":
		return true
	default:
		return false
	}
}

func discardedCommentsRule3(n *html.Node) bool {
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
		strings.Contains(class, "message"),
		strings.Contains(id, "akismet"),
		strings.Contains(class, "akismet"),
		strings.Contains(style, "display:none"):
	default:
		return false
	}

	return true
}
