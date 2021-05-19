package selector

import (
	"strings"

	"github.com/go-shiori/dom"
	"golang.org/x/net/html"
)

func metaAuthorRule1(n *html.Node) bool {
	class := dom.ClassName(n)
	rel := dom.GetAttribute(n, "rel")
	tagName := dom.TagName(n)

	switch tagName {
	case "a", "address", "link", "p", "span":
	case "author":
		return true
	default:
		return false
	}

	switch {
	case rel == "me",
		rel == "author",
		class == "author":
	default:
		return false
	}

	return true
}

func metaAuthorRule2(n *html.Node) bool {
	class := dom.ClassName(n)
	itemProp := dom.GetAttribute(n, "itemprop")
	tagName := dom.TagName(n)

	switch tagName {
	case "a", "span":
	default:
		return false
	}

	switch {
	case strings.Contains(class, "author"),
		strings.Contains(class, "authors"),
		strings.Contains(class, "posted-by"),
		strings.Contains(itemProp, "author"):
	default:
		return false
	}

	return true
}

func metaAuthorRule3(n *html.Node) bool {
	class := dom.ClassName(n)
	tagName := dom.TagName(n)

	switch tagName {
	case "a", "div", "p", "span":
	default:
		return false
	}

	switch {
	case strings.Contains(class, "byline"):
	default:
		return false
	}

	return true
}

func metaAuthorRule4(n *html.Node) bool {
	class := dom.ClassName(n)

	switch {
	case strings.Contains(class, "author"),
		strings.Contains(class, "screenname"):
		return true
	default:
		return false
	}
}
