package selector

import (
	"strings"

	"github.com/go-shiori/dom"
	"golang.org/x/net/html"
)

func metaCategoriesRule1(n *html.Node) bool {
	if dom.TagName(n) != "a" {
		return false
	}

	ancestors := getNodeAncestors(n, "div")
	if len(ancestors) == 0 {
		return false
	}

	for _, ancestor := range ancestors {
		id := dom.ID(ancestor)
		class := dom.ClassName(ancestor)

		switch {
		case strings.HasPrefix(class, "post-info"),
			strings.HasPrefix(class, "postinfo"),
			strings.HasPrefix(class, "post-meta"),
			strings.HasPrefix(class, "postmeta"),
			strings.HasPrefix(class, "meta"),
			strings.HasPrefix(class, "entry-meta"),
			strings.HasPrefix(class, "entry-info"),
			strings.HasPrefix(class, "entry-utility"),
			strings.HasPrefix(id, "postpath"):
			return true
		}
	}

	return false
}

func metaCategoriesRule2(n *html.Node) bool {
	if dom.TagName(n) != "a" {
		return false
	}

	ancestors := getNodeAncestors(n, "p")
	if len(ancestors) == 0 {
		return false
	}

	for _, ancestor := range ancestors {
		id := dom.ID(ancestor)
		class := dom.ClassName(ancestor)

		switch {
		case strings.HasPrefix(class, "postmeta"),
			strings.HasPrefix(class, "entry-categories"),
			class == "postinfo",
			id == "filedunder":
			return true
		}
	}

	return false
}

func metaCategoriesRule3(n *html.Node) bool {
	if dom.TagName(n) != "a" {
		return false
	}

	ancestors := getNodeAncestors(n, "footer")
	if len(ancestors) == 0 {
		return false
	}

	for _, ancestor := range ancestors {
		class := dom.ClassName(ancestor)

		switch {
		case strings.HasPrefix(class, "entry-meta"),
			strings.HasPrefix(class, "entry-footer"):
			return true
		}
	}

	return false
}

func metaCategoriesRule4(n *html.Node) bool {
	if dom.TagName(n) != "a" {
		return false
	}

	ancestors := getNodeAncestors(n, "li")
	ancestors = append(ancestors, getNodeAncestors(n, "span")...)
	if len(ancestors) == 0 {
		return false
	}

	for _, ancestor := range ancestors {
		class := dom.ClassName(ancestor)

		switch {
		case class == "post-category",
			class == "postcategory",
			class == "entry-category":
			return true
		}
	}

	return false
}

func metaCategoriesRule5(n *html.Node) bool {
	if dom.TagName(n) != "a" {
		return false
	}

	ancestors := getNodeAncestors(n, "header")
	if len(ancestors) == 0 {
		return false
	}

	for _, ancestor := range ancestors {
		class := dom.ClassName(ancestor)

		if class == "entry-header" {
			return true
		}
	}

	return false
}

func metaCategoriesRule6(n *html.Node) bool {
	if dom.TagName(n) != "a" {
		return false
	}

	ancestors := getNodeAncestors(n, "div")
	if len(ancestors) == 0 {
		return false
	}

	for _, ancestor := range ancestors {
		switch dom.ClassName(ancestor) {
		case "row", "tags":
			return true
		}
	}

	return false
}
