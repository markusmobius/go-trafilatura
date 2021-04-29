package trafilatura

import (
	"github.com/go-shiori/dom"
	"golang.org/x/net/html"
)

// cleanHTML cleans the tree by discarding unwanted elements
func cleanHTML(doc *html.Node, includeTables, includeImages bool) {
	// Prepare list to be cleaned
	killList := duplicateMap(tagsToKill)
	removeList := duplicateMap(tagsToRemove)

	if !includeTables {
		killList["table"] = struct{}{}
	}

	if includeImages {
		// Many websites have <img> inside <figure> or <picture> or <source> tag
		delete(killList, "figure")
		delete(killList, "picture")
		delete(killList, "source")
		delete(removeList, "img")
	}

	// Remove comment nodes
	removeCommentNodes(doc)

	// Remove nodes in kill list including its children
	for tagName := range killList {
		nodes := dom.GetElementsByTagName(doc, tagName)
		removeNodes(nodes)
	}

	// Remove nodes in remove list but keep its children
	for tagName := range removeList {
		nodes := dom.GetElementsByTagName(doc, tagName)
		removeNodesKeepChildren(nodes)
	}
}

// pruneHTML deletes selected empty elements
func pruneHTML(doc *html.Node) {
	// Find all empty nodes
	var emptyNodes []*html.Node
	for _, node := range dom.GetElementsByTagName(doc, "*") {
		tagName := dom.TagName(node)
		if _, exist := emptyTagsToRemove[tagName]; !exist {
			continue
		}

		if len(dom.ChildNodes(node)) != 0 {
			continue
		}

		emptyNodes = append(emptyNodes, node)
	}

	removeNodes(emptyNodes)
}

// removeCommentNodes find all comment nodes in document then remove it.
func removeCommentNodes(doc *html.Node) {
	// Find all comment nodes
	var finder func(*html.Node)
	var commentNodes []*html.Node

	finder = func(node *html.Node) {
		if node.Type == html.CommentNode {
			commentNodes = append(commentNodes, node)
		}

		for child := node.FirstChild; child != nil; child = child.NextSibling {
			finder(child)
		}
	}

	for child := doc.FirstChild; child != nil; child = child.NextSibling {
		finder(child)
	}

	// Remove it
	removeNodes(commentNodes)
}

// removeNodes iterates over a nodeList and remove each of them.
func removeNodes(nodeList []*html.Node) {
	for i := len(nodeList) - 1; i >= 0; i-- {
		node := nodeList[i]
		parentNode := node.Parent
		if parentNode != nil {
			parentNode.RemoveChild(node)
		}
	}
}

// removeNodesKeepChildren iterates over a nodeList and remove each of them
// while still keeping the children.
func removeNodesKeepChildren(nodeList []*html.Node) {
	for i := len(nodeList) - 1; i >= 0; i-- {
		node := nodeList[i]

		// Make sure node has parent
		if node.Parent == nil {
			continue
		}

		// Make sure node has children
		childNodes := dom.ChildNodes(node)
		if len(childNodes) == 0 {
			continue
		}

		// Move children to parent
		for _, child := range childNodes {
			clone := dom.Clone(child, true)
			node.Parent.InsertBefore(clone, node)
		}

		// Remove the node itself
		node.Parent.RemoveChild(node)
	}
}
