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

// Code in this file is ported from <https://github.com/adbar/trafilatura>
// which available under Apache 2.0 license. It mirrors the Markdown output
// produced by `xmltotxt(tree, include_formatting=True)` in the original
// `trafilatura/xml.py`, adapted to the standard HTML tags that
// go-trafilatura keeps in its content tree.

package trafilatura

import (
	"strconv"
	"strings"

	"github.com/go-shiori/dom"
	"github.com/markusmobius/go-trafilatura/internal/etree"
	"golang.org/x/net/html"
)

const maxTableWidth = 1000

// newlineSentinel mirrors Python's "\u2424" spacing hack: it marks a hard
// block separation that survives line filtering and is later turned into a
// blank line (i.e. a paragraph break in Markdown).
const newlineSentinel = "\u2424"

// mdHiFormatting maps inline emphasis HTML tags to their Markdown delimiter.
// This mirrors `HI_FORMATTING` combined with the HTML->XML `rend` mapping used
// by the original Trafilatura.
var mdHiFormatting = map[string]string{
	"em":     "*",
	"i":      "*",
	"b":      "**",
	"strong": "**",
	"u":      "__",
	"kbd":    "`",
	"samp":   "`",
	"tt":     "`",
	"var":    "`",
}

var (
	// mdDelTags are rendered as strikethrough (`~~text~~`).
	mdDelTags = sliceToMap("del", "s", "strike")

	// mdCodeTags are rendered as inline code or fenced code blocks.
	mdCodeTags = sliceToMap("code", "pre")

	// mdStripLeadingTags have their leading text trimmed (matches Python's
	// handling of "article"/"list"/"table").
	mdStripLeadingTags = mergeStringSets(sliceToMap("article", "table"), mapXmlListTags)

	// mdNewlineElems are block-level elements that introduce a line break,
	// mirroring Python's NEWLINE_ELEMS.
	mdNewlineElems = mergeStringSets(
		mapXmlGraphicTags, // img
		mapXmlHeadTags,    // h1-h6, summary
		mapXmlLbTags,      // br, hr, lb
		mapXmlListTags,    // ul, ol, dl
		sliceToMap("p"),
		sliceToMap("blockquote", "q"),
		sliceToMap("tr"),
		sliceToMap("table"),
	)

	// mdSpecialFormatting elements handle their own spacing, mirroring
	// Python's SPECIAL_FORMATTING.
	mdSpecialFormatting = mergeStringSets(
		mdCodeTags,
		mdDelTags,
		mapXmlHeadTags,
		mapXmlHiTags,
		mapXmlRefTags,
		mapXmlItemTags,
		mapXmlCellTags,
	)
)

// htmlToMarkdown converts a content/comments node tree into Markdown text.
// It clones the tree first so that the source node is left untouched.
func htmlToMarkdown(node *html.Node) string {
	if node == nil {
		return ""
	}

	clone := dom.Clone(node, true)

	var parts []string
	mdProcessElement(clone, &parts, true)

	return mdSanitize(strings.Join(parts, ""))
}

// mdProcessElement recursively flattens an element and its children into a
// list of Markdown string fragments. Ported from `process_element`.
func mdProcessElement(element *html.Node, parts *[]string, includeFormatting bool) {
	tagName := dom.TagName(element)

	if inMap(tagName, mapXmlCellTags) && dom.PreviousElementSibling(element) == nil {
		*parts = append(*parts, "| ")
	}

	elemText := etree.Text(element)
	if elemText != "" {
		*parts = append(*parts, mdReplaceElementText(element, includeFormatting))
	}

	// When inside a table cell, the tail is appended right after the element
	// text since cells are flattened into a single line.
	if etree.Tail(element) != "" && tagName != "img" && mdIsInTableCell(element) {
		tail := trim(etree.Tail(element))
		if tail != "" {
			switch mdLastChar(*parts) {
			case " ", "|", "":
			default:
				tail = " " + tail
			}
		}
		*parts = append(*parts, tail)
	}

	for _, child := range dom.Children(element) {
		mdProcessElement(child, parts, includeFormatting)
	}

	if elemText == "" {
		if tagName == "img" {
			title := dom.GetAttribute(element, "title")
			alt := dom.GetAttribute(element, "alt")
			src := dom.GetAttribute(element, "src")
			if src == "" {
				src = dom.GetAttribute(element, "data-src")
			}
			*parts = append(*parts, "!["+trim(title+" "+alt)+"]("+src+")")
		}

		if tail := etree.Tail(element); tail != "" {
			*parts = append(*parts, " "+trim(tail))
		} else if inMap(tagName, mdNewlineElems) {
			if tagName == "tr" {
				mdProcessRow(element, parts)
			} else if !mdHasAncestorCell(element) {
				*parts = append(*parts, "\n")
			}
		} else if !inMap(tagName, mapXmlCellTags) && !inMap(tagName, mapXmlItemTags) {
			return
		}
	}

	// End-tag spacing.
	inCell := mdIsInTableCell(element)
	switch {
	case inMap(tagName, mdNewlineElems) && !mdHasAncestorCell(element) && !mdIsElementInItem(element):
		if includeFormatting && tagName != "tr" {
			*parts = append(*parts, "\n"+newlineSentinel+"\n")
		} else {
			*parts = append(*parts, "\n")
		}
	case inMap(tagName, mapXmlCellTags):
		*parts = append(*parts, " | ")
	case (inMap(tagName, mapXmlHeadTags) || inMap(tagName, mapXmlItemTags)) && inCell && !mdIsLastElementInCell(element):
		*parts = append(*parts, " ")
	case !inMap(tagName, mdSpecialFormatting) && !mdIsLastElementInCell(element):
		*parts = append(*parts, " ")
	}

	// Tail that comes after the closing tag.
	if etree.Tail(element) != "" && !inCell && tagName != "img" {
		inItem := mdIsElementInItem(element)
		var tail string
		if inItem || inMap(tagName, mapXmlListTags) {
			tail = trim(etree.Tail(element))
		} else {
			tail = etree.Tail(element)
		}

		if tail != "" && inItem {
			switch mdLastChar(*parts) {
			case " ", "\n", "|", "":
			default:
				tail = " " + tail
			}
		}
		*parts = append(*parts, tail)
	}

	if mdIsLastElementInItem(element) && !inCell {
		*parts = append(*parts, "\n")
	}
}

// mdProcessRow handles the Markdown table row padding and the header separator.
// Ported from the `row` branch of `process_element`.
func mdProcessRow(row *html.Node, parts *[]string) {
	cells := mdDirectCells(row)
	cellCount := len(cells)

	spanInfo := dom.GetAttribute(row, "colspan")
	if spanInfo == "" {
		spanInfo = dom.GetAttribute(row, "span")
	}

	span := 0
	if n, err := strconv.Atoi(spanInfo); err == nil {
		span = n
	}

	maxSpan := min(max(span, cellCount), maxTableWidth)
	if cellCount < maxSpan {
		*parts = append(*parts, strings.Repeat("|", maxSpan-cellCount)+"\n")
	}

	if mdRowHasHead(row) {
		*parts = append(*parts, "\n|"+strings.Repeat("---|", maxSpan)+"\n")
	}
}

// mdReplaceElementText computes the Markdown rendering of an element's own text
// (the text before its first child). Ported from `replace_element_text`.
func mdReplaceElementText(element *html.Node, includeFormatting bool) string {
	tagName := dom.TagName(element)
	elemText := etree.Text(element)

	if includeFormatting && elemText != "" {
		switch {
		case inMap(tagName, mdStripLeadingTags):
			elemText = strings.TrimSpace(elemText)

		case inMap(tagName, mapXmlHeadTags) && !mdIsInTableCell(element):
			elemText = strings.Repeat("#", mdHeadLevel(tagName)) + " " + elemText

		case inMap(tagName, mdDelTags):
			elemText = "~~" + elemText + "~~"

		case inMap(tagName, mapXmlHiTags):
			if delim, ok := mdHiFormatting[tagName]; ok {
				elemText = delim + elemText + delim
			}

		case inMap(tagName, mdCodeTags):
			lbs := mdLineBreakDescendants(element)
			if strings.Contains(elemText, "\n") || len(lbs) > 0 {
				for _, lb := range lbs {
					elemText = elemText + "\n" + etree.Tail(lb)
					etree.Remove(lb)
				}
				elemText = "```\n" + elemText + "\n```\n"
			} else {
				elemText = "`" + elemText + "`"
			}
		}
	}

	// Links.
	if inMap(tagName, mapXmlRefTags) && elemText != "" {
		linkText := "[" + elemText + "]"
		if target := trim(dom.GetAttribute(element, "href")); target != "" {
			elemText = linkText + "(" + target + ")"
		} else {
			elemText = linkText
		}
	}

	// Table cells.
	if inMap(tagName, mapXmlCellTags) {
		elemText = strings.TrimSpace(elemText)
		if elemText != "" && !mdIsLastElementInCell(element) {
			elemText += " "
		}
	}

	// List items.
	if mdIsFirstElementInItem(element) && !mdIsInTableCell(element) {
		elemText = "- " + elemText
	}

	return elemText
}

// mdSanitize joins fragments into the final Markdown string. It drops
// whitespace-only lines and turns the newline sentinel into blank lines,
// mirroring `sanitize(text, preserve_space=True)`.
func mdSanitize(text string) string {
	lines := strings.Split(text, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		kept = append(kept, line)
	}

	result := strings.Join(kept, "\n")
	result = strings.ReplaceAll(result, newlineSentinel, "")
	return strings.TrimSpace(result)
}

// mdHeadLevel returns the heading level for a head tag (h1-h6), defaulting to 2.
func mdHeadLevel(tagName string) int {
	if len(tagName) == 2 && tagName[0] == 'h' && tagName[1] >= '1' && tagName[1] <= '6' {
		return int(tagName[1] - '0')
	}
	return 2
}

// mdLineBreakDescendants returns all line-break elements (br/hr/lb) within el.
func mdLineBreakDescendants(el *html.Node) []*html.Node {
	var result []*html.Node
	for _, node := range dom.GetElementsByTagName(el, "*") {
		if inMap(dom.TagName(node), mapXmlLbTags) {
			result = append(result, node)
		}
	}
	return result
}

// mdDirectCells returns the direct child cell elements (th/td) of a row.
func mdDirectCells(row *html.Node) []*html.Node {
	var result []*html.Node
	for _, child := range dom.Children(row) {
		if inMap(dom.TagName(child), mapXmlCellTags) {
			result = append(result, child)
		}
	}
	return result
}

// mdRowHasHead reports whether a row contains a header cell (th).
func mdRowHasHead(row *html.Node) bool {
	for _, child := range dom.Children(row) {
		if dom.TagName(child) == "th" {
			return true
		}
	}
	return false
}

// mdIsInTableCell reports whether the element or one of its ancestors is a cell.
func mdIsInTableCell(el *html.Node) bool {
	if el.Parent == nil {
		return false
	}
	for cur := el; cur != nil; cur = cur.Parent {
		if cur.Type == html.ElementNode && inMap(dom.TagName(cur), mapXmlCellTags) {
			return true
		}
	}
	return false
}

// mdHasAncestorCell reports whether one of the element's ancestors is a cell.
func mdHasAncestorCell(el *html.Node) bool {
	for cur := el.Parent; cur != nil; cur = cur.Parent {
		if cur.Type == html.ElementNode && inMap(dom.TagName(cur), mapXmlCellTags) {
			return true
		}
	}
	return false
}

// mdIsLastElementInCell reports whether the element is the last element of its cell.
func mdIsLastElementInCell(el *html.Node) bool {
	if !mdIsInTableCell(el) {
		return false
	}

	container := el
	if !inMap(dom.TagName(el), mapXmlCellTags) {
		container = mdParentElement(el)
	}
	if container == nil {
		return true
	}

	children := dom.Children(container)
	return len(children) == 0 || children[len(children)-1] == el
}

// mdIsElementInItem reports whether the element is, or is contained within, a
// list item.
func mdIsElementInItem(el *html.Node) bool {
	for cur := el; cur != nil; cur = cur.Parent {
		if cur.Type == html.ElementNode && inMap(dom.TagName(cur), mapXmlItemTags) {
			return true
		}
	}
	return false
}

// mdIsFirstElementInItem reports whether the element is the first element in a
// list item.
func mdIsFirstElementInItem(el *html.Node) bool {
	if inMap(dom.TagName(el), mapXmlItemTags) && etree.Text(el) != "" {
		return true
	}

	var itemAncestor *html.Node
	for cur := el; cur != nil; cur = cur.Parent {
		if cur.Type == html.ElementNode && inMap(dom.TagName(cur), mapXmlItemTags) {
			itemAncestor = cur
			break
		}
	}

	if itemAncestor == nil {
		return false
	}
	return etree.Text(itemAncestor) == ""
}

// mdIsLastElementInItem reports whether the element is the last element in a
// list item.
func mdIsLastElementInItem(el *html.Node) bool {
	if !mdIsElementInItem(el) {
		return false
	}

	if inMap(dom.TagName(el), mapXmlItemTags) {
		return len(dom.Children(el)) == 0
	}

	next := dom.NextElementSibling(el)
	if next == nil {
		return true
	}
	return inMap(dom.TagName(next), mapXmlItemTags)
}

// mdParentElement returns the nearest element-node parent of el.
func mdParentElement(el *html.Node) *html.Node {
	for cur := el.Parent; cur != nil; cur = cur.Parent {
		if cur.Type == html.ElementNode {
			return cur
		}
	}
	return nil
}

// mdLastChar returns the last character of the last fragment, or "" if empty.
func mdLastChar(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	last := parts[len(parts)-1]
	if last == "" {
		return ""
	}
	runes := []rune(last)
	return string(runes[len(runes)-1])
}

// mergeStringSets merges several string sets into a new one.
func mergeStringSets(sets ...map[string]struct{}) map[string]struct{} {
	result := make(map[string]struct{})
	for _, set := range sets {
		for key := range set {
			result[key] = struct{}{}
		}
	}
	return result
}
