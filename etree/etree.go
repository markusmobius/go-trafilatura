package etree

import (
	"strings"

	"github.com/go-shiori/dom"
	"github.com/yosssi/gohtml"
	"golang.org/x/net/html"
)

// Element returns a html.Node with specified tag name.
func Element(tagName string) *html.Node {
	return dom.CreateElement(tagName)
}

// SubElement creates a html.Node with specified tag name, and
// append it to an existing element.
func SubElement(parent *html.Node, tagName string) *html.Node {
	element := dom.CreateElement(tagName)
	dom.AppendChild(parent, element)
	return element
}

// StripTags deletes all elements with the provided tag names from
// a tree or subtree. This will remove the elements and their attributes,
// but not their text/tail content or descendants. Instead, it will merge
// the text content and children of the element into its parent.
func StripTags(tree *html.Node, tags ...string) {
	// Fetch all suitable elements
	var elements []*html.Node
	for _, tag := range tags {
		elements = append(elements, dom.GetElementsByTagName(tree, tag)...)
	}

	for _, element := range elements {
		// Make sure element has parent
		if element.Parent == nil {
			continue
		}

		// Move children to parent
		for _, child := range dom.ChildNodes(element) {
			clone := dom.Clone(child, true)
			element.Parent.InsertBefore(clone, element)
		}

		// Remove the element itself
		element.Parent.RemoveChild(element)
	}
}

// StripElements deletes all elements with the provided tag names from a tree or subtree.
// This will remove the elements and their entire subtree, including all their attributes,
// text content and descendants. It will also remove the tail text of the element unless
// you explicitly set the keepTail argument to true.
func StripElements(tree *html.Node, keepTail bool, tags ...string) {
	// Fetch all suitable elements
	var elements []*html.Node
	for _, tag := range tags {
		elements = append(elements, dom.GetElementsByTagName(tree, tag)...)
	}

	// Remove each element
	for _, element := range elements {
		Remove(element, keepTail)
	}
}

// Remove will removes the element and its entire subtree, including all of its attributes,
// text content and descendants. It will also remove the tail text of the element unless
// you explicitly set the keepTail argument to true.
func Remove(element *html.Node, keepTail ...bool) {
	if element.Parent == nil {
		return
	}

	if len(keepTail) == 0 || !keepTail[0] {
		for _, tailNode := range TailNodes(element) {
			tailNode.Parent.RemoveChild(tailNode)
		}
	}

	element.Parent.RemoveChild(element)
}

// ToString encode an element to string representation of its structure.
func ToString(tree *html.Node, prettify bool) string {
	// Create temporary container
	container := dom.CreateElement("tmp")

	// Put clone of tree inside
	dom.AppendChild(container, dom.Clone(tree, true))

	// Put tails of tree inside container
	for next := tree.NextSibling; next != nil; next = next.NextSibling {
		if next.Type == html.ElementNode {
			break
		} else if next.Type == html.TextNode {
			clone := dom.Clone(next, false)
			dom.AppendChild(container, clone)
		}
	}

	// Convert to string
	str := dom.InnerHTML(container)
	if prettify {
		str = gohtml.Format(str)
	}

	return str
}

// FromString parses an HTML document or element from a string.
func FromString(str string) *html.Node {
	doc, err := html.Parse(strings.NewReader(str))
	if err != nil {
		return nil
	}

	root := dom.QuerySelector(doc, "body > *")
	return dom.Clone(root, true)
}
