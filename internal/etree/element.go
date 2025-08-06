// This file is part of go-trafilatura, Go package for extracting readable
// content, comments and metadata from a web page. Source available in
// <https://github.com/markusmobius/go-trafilatura>.
//
// Copyright (C) 2021 Markus Mobius
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package etree

import (
	"bytes"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/go-shiori/dom"
	"golang.org/x/net/html"
)

// Iter loops over this element and all subelements in document order,
// and returns all elements with a matching tag.
func Iter(element *html.Node, tags ...string) []*html.Node {
	// Make sure element is exist
	if element == nil {
		return nil
	}

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

// IterDescendants here is similar with Iter, except it excludes itself.
func IterDescendants(element *html.Node, tags ...string) []*html.Node {
	elementList := Iter(element, tags...)
	if len(elementList) == 0 {
		return nil
	}

	// If the first element in the list is itself, exclude it.
	if elementList[0] == element {
		return elementList[1:]
	}

	return elementList
}

// Text returns texts before first subelement. If there was no text,
// this function will returns an empty string.
func Text(element *html.Node) string {
	if element == nil {
		return ""
	}

	var sb strings.Builder
	for child := element.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode {
			break
		} else if child.Type == html.TextNode {
			sb.WriteString(child.Data)
		}
	}

	return sb.String()
}

// SetText sets the value for element's text.
func SetText(element *html.Node, text string) {
	// Make sure element is not void
	if element == nil || dom.IsVoidElement(element) {
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
	if element == nil {
		return ""
	}

	var sb strings.Builder
	for _, tailNode := range TailNodes(element) {
		sb.WriteString(tailNode.Data)
	}

	return sb.String()
}

// SetTail sets the value for element's tail.
func SetTail(element *html.Node, tail string) {
	// Make sure parent exist and not void
	if element == nil || element.Parent == nil || dom.IsVoidElement(element.Parent) {
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
	// Make sure element is exist
	if element == nil {
		return nil
	}

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
	if node == nil || subelement == nil {
		return
	}

	tails := TailNodes(subelement)
	dom.AppendChild(node, subelement)
	for _, tail := range tails {
		dom.AppendChild(node, tail)
	}
}

// Extend appends subelements into the node.
func Extend(node *html.Node, subelements ...*html.Node) {
	if node == nil || len(subelements) == 0 {
		return
	}

	for _, subelement := range subelements {
		Append(node, subelement)
	}
}

// IterText loops over this element and all subelements in document order,
// and returns all inner text. Similar with dom.TextContent, except here we
// add whitespaces when element level changed.
func IterText(node *html.Node, separator string) string {
	if node == nil {
		return ""
	}

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

func IterTextWithSpacing(n *html.Node) string {
	var b strings.Builder

	var walk func(*html.Node)
	walk = func(node *html.Node) {
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.TextNode {
				text := strings.TrimSpace(c.Data)
				if b.Len() > 0 && needsSpace(b.String(), text) {
					b.WriteString(" ")
				}
				b.WriteString(text)
			} else {
				walk(c)
			}
			if tail := strings.TrimSpace(c.Data); tail != "" {
				b.WriteString(" ")
			}
		}
	}

	walk(n)
	return strings.TrimSpace(b.String())
}

func needsSpace(prev, next string) bool {
	if prev == "" || next == "" {
		return false
	}
	r1, _ := utf8.DecodeLastRuneInString(prev)
	r2, _ := utf8.DecodeRuneInString(next)

	// Insert space only if both are alphanumeric (i.e., words)
	return unicode.IsLetter(r1) && unicode.IsLetter(r2)
}
