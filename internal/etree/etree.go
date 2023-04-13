// This file is part of go-trafilatura, Go package for extracting readable
// content, comments and metadata from a web page. Source available in
// <https://github.com/markusmobius/go-trafilatura>.
// Copyright (C) 2021 Markus Mobius
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by the
// Free Software Foundation, either version 3 of the License, or (at your
// option) any later version.
//
// This program is distributed in the hope that it will be useful, but
// WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY
// or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License
// for more details.
//
// You should have received a copy of the GNU General Public License along
// with this program. If not, see <https://www.gnu.org/licenses/>.

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
	for _, tag := range tags {
		elements := dom.GetElementsByTagName(tree, tag)
		for i := len(elements) - 1; i >= 0; i-- {
			Strip(elements[i])
		}
	}
}

// StripElements deletes all elements with the provided tag names from a tree or subtree.
// This will remove the elements and their entire subtree, including all their attributes,
// text content and descendants. It will also remove the tail text of the element unless
// you explicitly set the keepTail argument to true.
func StripElements(tree *html.Node, keepTail bool, tags ...string) {
	// Fetch all suitable elements
	for _, tag := range tags {
		elements := dom.GetElementsByTagName(tree, tag)
		for i := len(elements) - 1; i >= 0; i-- {
			Remove(elements[i], keepTail)
		}
	}
}

// Remove will removes the element and its entire subtree, including all of its attributes,
// text content and descendants. It will also remove the tail text of the element unless
// you explicitly set the keepTail argument to true.
func Remove(element *html.Node, keepTail ...bool) {
	if element == nil || element.Parent == nil {
		return
	}

	if len(keepTail) == 0 || !keepTail[0] {
		for _, tailNode := range TailNodes(element) {
			tailNode.Parent.RemoveChild(tailNode)
		}
	}

	element.Parent.RemoveChild(element)
}

// Strip will removes the element but not their text/tail content or descendants.
// Instead, it will merge the text content and children of the element into its parent.
func Strip(element *html.Node) {
	if element == nil || element.Parent == nil {
		return
	}

	// Move children to parent
	for _, child := range dom.ChildNodes(element) {
		clone := dom.Clone(child, true)
		element.Parent.InsertBefore(clone, element)
	}

	// Remove the element itself
	element.Parent.RemoveChild(element)
}

// ToString encode an element to string representation of its structure.
func ToString(tree *html.Node, prettify ...bool) string {
	if tree == nil {
		return ""
	}

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
	if len(prettify) > 0 && prettify[0] {
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
