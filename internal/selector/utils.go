package selector

import (
	"github.com/go-shiori/dom"
	"golang.org/x/net/html"
)

func getNodeAncestors(node *html.Node, ancestorTag string) []*html.Node {
	var ancestors []*html.Node
	for parent := node.Parent; parent != nil; parent = parent.Parent {
		if dom.TagName(parent) == ancestorTag {
			ancestors = append(ancestors, parent)
		}
	}

	return ancestors
}
