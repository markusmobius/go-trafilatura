package etree

import (
	"bytes"
	"strings"

	"github.com/go-shiori/dom"
	"golang.org/x/net/html"
)

// Iter loops over this element and all subelements in document order,
// and returns all elements with a matching tag.
func Iter(element *html.Node, tags ...string) []*html.Node {
	// Convert tags to map
	mapTags := make(map[string]struct{})
	for _, tag := range tags {
		mapTags[tag] = struct{}{}
	}

	// If there are no tags specified, return element and all of its children
	if len(mapTags) == 0 {
		return append(
			[]*html.Node{element},
			dom.GetElementsByTagName(element, "*")...)
	}

	// At this point there are list of tags defined, so first prepare list of element.
	var elementList []*html.Node

	// First, check if element should be included in list
	if _, requested := mapTags[dom.TagName(element)]; requested {
		elementList = append(elementList, element)
	}

	// Next look in children recursively
	var finder func(*html.Node)
	finder = func(node *html.Node) {
		if node.Type == html.ElementNode {
			if _, requested := mapTags[node.Data]; requested {
				elementList = append(elementList, node)
			}
		}

		for child := node.FirstChild; child != nil; child = child.NextSibling {
			finder(child)
		}
	}

	for child := element.FirstChild; child != nil; child = child.NextSibling {
		finder(child)
	}

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
	// Make sure element is not void
	if dom.IsVoidElement(element) {
		return
	}

	// Remove the old text
	child := element.FirstChild
	for child != nil && child.Type != html.ElementNode {
		nextSibling := child.NextSibling
		if child.Type == html.TextNode {
			element.RemoveChild(child)
		}
		child = nextSibling
	}

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
	// Make sure parent exist and not void
	if element.Parent == nil || dom.IsVoidElement(element.Parent) {
		return
	}

	// Remove the old tails
	dom.RemoveNodes(TailNodes(element), nil)

	// If the new tail is blank, stop
	if tail == "" {
		return
	}

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

// Append appends single subelement into the node.
func Append(node, subelement *html.Node) {
	// Fetch the tail nodes of the subelement
	tailNodes := TailNodes(subelement)

	dom.AppendChild(node, subelement)
	for _, tail := range tailNodes {
		dom.AppendChild(node, tail)
	}
}

// Extend appends subelements into the node.
func Extend(node *html.Node, subelements ...*html.Node) {
	for _, subelement := range subelements {
		Append(node, subelement)
	}
}

// IterText loops over this element and all subelements in document order,
// and returns all inner text. Similar with dom.TextContent, except here we
// add whitespaces when element level changed.
func IterText(node *html.Node, separator string) string {
	var buffer bytes.Buffer
	var finder func(*html.Node, int)
	var lastLevel int

	finder = func(n *html.Node, level int) {
		if n.Type == html.ElementNode && dom.IsVoidElement(n) {
			buffer.WriteString(separator)
		} else if n.Type == html.TextNode {
			if level != lastLevel {
				buffer.WriteString(separator)
			}
			buffer.WriteString(n.Data)
		}

		lastLevel = level
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			finder(child, level+1)
		}
	}

	finder(node, 0)
	result := buffer.String()
	return strings.TrimSpace(result)
}
