package etree

import (
	"bytes"

	"github.com/go-shiori/dom"
	"golang.org/x/net/html"
)

// Iter loops over this element and all subelements in document order,
// and returns all elements with a matching tag.
func Iter(element *html.Node, tags ...string) []*html.Node {
	// Prepare list of element
	var elementList []*html.Node

	// Check if current element need to be included
	if len(tags) == 0 {
		elementList = []*html.Node{element}
		tags = append(tags, "*")
	} else {
		elementTag := dom.TagName(element)
		for _, tag := range tags {
			if tag == elementTag {
				elementList = []*html.Node{element}
				break
			}
		}
	}

	// Populate list with matching children
	matchingChildren := dom.GetAllNodesWithTag(element, tags...)
	elementList = append(elementList, matchingChildren...)

	return elementList
}

// Text returns texts before first subelement. If there was no text,
// this function will returns an empty string.
func Text(element *html.Node) string {
	buffer := bytes.NewBuffer(nil)
	for child := element.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode {
			break
		} else if child.Type == html.TextNode {
			buffer.WriteString(child.Data)
		}
	}

	return buffer.String()
}

// SetText sets the value for element's text.
func SetText(element *html.Node, text string) {
	// Remove the old text
	var oldTexts []*html.Node
	for child := element.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode {
			break
		} else if child.Type == html.TextNode {
			oldTexts = append(oldTexts, child)
		}
	}
	dom.RemoveNodes(oldTexts, nil)

	// Insert the new text
	newText := dom.CreateTextNode(text)
	element.InsertBefore(newText, element.FirstChild)
}

// Tail returns text after this element's end tag, but before the
// next sibling element's start tag. If there was no text, this
// function will returns an empty string.
func Tail(element *html.Node) string {
	buffer := bytes.NewBuffer(nil)
	for _, tailNode := range TailNodes(element) {
		buffer.WriteString(tailNode.Data)
	}

	return buffer.String()
}

// SetTail sets the value for element's tail.
func SetTail(element *html.Node, tail string) {
	// Remove the old tails
	dom.RemoveNodes(TailNodes(element), nil)

	// Insert the new tail
	newTail := dom.CreateTextNode(tail)
	if element.NextSibling != nil {
		element.Parent.InsertBefore(newTail, element.NextSibling)
	} else {
		element.Parent.AppendChild(newTail)
	}
}

// TailNodes returns the list of tail nodes for the element.
func TailNodes(element *html.Node) []*html.Node {
	var nodes []*html.Node
	for next := element.NextSibling; next != nil; next = next.NextSibling {
		if next.Type == html.ElementNode {
			break
		} else if next.Type == html.TextNode {
			nodes = append(nodes, next)
		}
	}

	return nodes
}

// Append appends single subelement into the node. Don't use it
// inside loop because the behavior will be different compared
// to Python lxml. For these case, use Extend instead.
func Append(node, subelement *html.Node) {
	dom.AppendChild(node, subelement)
}

// Extend appends subelements into the node.
func Extend(node *html.Node, subelements ...*html.Node) {
	// Find the node's last child as reference
	referenceNode := node.LastChild

	// Prepare  function for appending element
	var append func(*html.Node)
	if referenceNode == nil {
		append = func(child *html.Node) {
			dom.PrependChild(node, child)
		}
	} else {
		append = func(child *html.Node) {
			// Detach child from its family
			var detachedChild *html.Node

			if child.Parent != nil {
				detachedChild = dom.Clone(child, true)
				child.Parent.RemoveChild(child)
			} else {
				detachedChild = child
			}

			// Put the child after reference node
			if referenceNode.NextSibling == nil {
				dom.AppendChild(node, detachedChild)
			} else {
				node.InsertBefore(detachedChild, referenceNode.NextSibling)
			}
		}
	}

	// Process new elements backward
	for i := len(subelements) - 1; i >= 0; i-- {
		tailNodes := TailNodes(subelements[i])
		for j := len(tailNodes) - 1; j >= 0; j-- {
			append(tailNodes[j])
		}
		append(subelements[i])
	}
}
