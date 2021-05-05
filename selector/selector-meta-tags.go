package selector

import (
	"strings"

	"github.com/go-shiori/dom"
	"golang.org/x/net/html"
)

func metaTagsRule1(n *html.Node) bool {
	if dom.TagName(n) != "a" {
		return false
	}

	ancestors := getNodeAncestors(n, "div")
	if len(ancestors) == 0 {
		return false
	}

	for _, ancestor := range ancestors {
		if dom.ClassName(ancestor) == "tags" {
			return true
		}
	}

	return false
}

func metaTagsRule2(n *html.Node) bool {
	if dom.TagName(n) != "a" {
		return false
	}

	ancestors := getNodeAncestors(n, "p")
	if len(ancestors) == 0 {
		return false
	}

	for _, ancestor := range ancestors {
		class := dom.ClassName(ancestor)
		if strings.HasPrefix(class, "entry-tags") {
			return true
		}
	}

	return false
}

func metaTagsRule3(n *html.Node) bool {
	if dom.TagName(n) != "a" {
		return false
	}

	ancestors := getNodeAncestors(n, "div")
	if len(ancestors) == 0 {
		return false
	}

	for _, ancestor := range ancestors {
		class := dom.ClassName(ancestor)

		switch {
		case class == "row",
			class == "jp-relatedposts",
			class == "entry-utility",
			strings.HasPrefix(class, "tag"),
			strings.HasPrefix(class, "postmeta"),
			strings.HasPrefix(class, "meta"):
			return true
		}
	}

	return false
}

func metaTagsRule4(n *html.Node) bool {
	if dom.TagName(n) != "a" {
		return false
	}

	for parent := n.Parent; parent != nil; parent = parent.Parent {
		class := dom.ClassName(parent)

		switch {
		case class == "entry-meta",
			strings.Contains(class, "topics"):
			return true
		}
	}

	return false
}
