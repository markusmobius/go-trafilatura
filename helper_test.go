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
// which available under Apache 2.0 license.

package trafilatura

import (
	"io"
	nurl "net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	xmltree "github.com/beevik/etree"
	"github.com/go-shiori/dom"
	"github.com/markusmobius/go-trafilatura/internal/etree"
	"golang.org/x/net/html"
)

// openMockFile is used to open HTML document from specified mock file.
// Make sure to close the reader later.
func openMockFile(mockFiles map[string]string, url string) io.ReadCloser {
	// Open file
	path := mockFiles[url]
	path = filepath.Join("test-files", "mock", path)

	f, err := os.Open(path)
	if err != nil {
		log.Panic().Err(err)
	}

	return f
}

// parseMockFile open then convert a mock file into html.Node.
func parseMockFile(mockFiles map[string]string, url string) *html.Node {
	f := openMockFile(mockFiles, url)
	defer f.Close()

	doc, err := dom.Parse(f)
	if err != nil {
		log.Panic().Err(err)
	}

	return doc
}

// extractMockFile open then extract content from a mock file.
func extractMockFile(mockFiles map[string]string, url string, enableLink ...bool) *ExtractResult {
	// Open mock file
	f := openMockFile(mockFiles, url)
	defer f.Close()

	// Parse URL
	parsedURL, err := nurl.ParseRequestURI(url)
	if err != nil {
		log.Panic().Err(err)
	}

	// Extract
	var includeLinks bool
	if len(enableLink) > 0 {
		includeLinks = enableLink[0]
	}

	opts := Options{
		OriginalURL:    parsedURL,
		EnableFallback: true,
		IncludeLinks:   includeLinks}
	result, err := Extract(f, opts)
	if err != nil {
		log.Panic().Err(err)
	}

	return result
}

// docFromStr create document from raw HTML string. Used in tests.
func docFromStr(str string) *html.Node {
	doc, _ := html.Parse(strings.NewReader(str))
	return doc
}

func noSpace(s string) string {
	s = strings.Join(strings.Fields(s), "")
	return strings.TrimSpace(s)
}

func python220CanonicalHTML(element *html.Node) string {
	cloned := dom.Clone(element, true)
	for _, node := range etree.Iter(cloned) {
		switch dom.TagName(node) {
		case "strong":
			node.Data = "b"
		case "em":
			node.Data = "i"
		case "s", "strike":
			node.Data = "del"
		case "kbd":
			node.Data = "tt"
		}
	}
	return etree.ToString(cloned)
}

func python220Element(test testing.TB, input string) *html.Node {
	test.Helper()
	document := xmltree.NewDocument()
	if err := document.ReadFromString(input); err != nil {
		test.Fatalf("Invalid upstream internal-tree fixture: %v", err)
	}
	var convert func(*xmltree.Element) *html.Node
	convert = func(source *xmltree.Element) *html.Node {
		tag := source.Tag
		switch tag {
		case "ref":
			tag = "a"
		case "graphic":
			tag = "img"
		case "lb":
			tag = "br"
		case "quote":
			tag = "blockquote"
		case "row":
			tag = "tr"
		case "cell":
			tag = "td"
			if source.SelectAttrValue("role", "") == "head" {
				tag = "th"
			}
		case "list":
			tag = source.SelectAttrValue("rend", "ul")
		case "item":
			tag = "li"
		case "head":
			tag = source.SelectAttrValue("rend", "h2")
		case "hi":
			tag = strings.TrimPrefix(source.SelectAttrValue("rend", "#i"), "#")
			if tag == "t" {
				tag = "tt"
			}
		}
		target := &html.Node{Type: html.ElementNode, Data: tag}
		for _, attribute := range source.Attr {
			key := attribute.Key
			if key == "rend" || (source.Tag == "cell" && key == "role") {
				continue
			}
			if source.Tag == "ref" && key == "target" {
				key = "href"
			}
			target.Attr = append(target.Attr, html.Attribute{Key: key, Val: attribute.Value})
		}
		for _, token := range source.Child {
			switch child := token.(type) {
			case *xmltree.Element:
				target.AppendChild(convert(child))
			case *xmltree.CharData:
				target.AppendChild(&html.Node{Type: html.TextNode, Data: child.Data})
			case *xmltree.Comment:
				target.AppendChild(&html.Node{Type: html.CommentNode, Data: child.Data})
			}
		}
		return target
	}
	if document.Root() == nil {
		test.Fatal("Upstream internal-tree fixture has no root")
	}
	return convert(document.Root())
}
