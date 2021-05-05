package selector

import (
	"strings"

	"github.com/go-shiori/dom"
	"golang.org/x/net/html"
)

func metaTitleRule1(n *html.Node) bool {
	switch dom.ClassName(n) {
	case "entry-title", "post-title":
		return true
	default:
		return false
	}
}

func metaTitleRule2(n *html.Node) bool {
	id := dom.ID(n)
	class := dom.ClassName(n)
	itemProp := dom.GetAttribute(n, "itemprop")
	tagName := dom.TagName(n)

	switch tagName {
	case "h1", "h2":
	default:
		return false
	}

	switch {
	case strings.Contains(class, "post-title"),
		strings.Contains(class, "entry-title"),
		strings.Contains(class, "headline"),
		strings.Contains(id, "headline"),
		strings.Contains(itemProp, "headline"),
		strings.Contains(class, "post__title"):
	default:
		return false
	}

	return true
}

func metaTitleRule3(n *html.Node) bool {
	id := dom.ID(n)
	class := dom.ClassName(n)
	tagName := dom.TagName(n)

	switch tagName {
	case "h1":
	default:
		return false
	}

	switch {
	case strings.Contains(class, "title"),
		strings.Contains(id, "title"):
	default:
		return false
	}

	return true
}

func metaTitleRule4(n *html.Node) bool {
	tagName := dom.TagName(n)
	if tagName != "h1" {
		return false
	}

	if n.Parent == nil || dom.TagName(n.Parent) != "header" {
		return false
	}

	return true
}
