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

// Code in this file is ported from <https://github.com/adbar/trafilatura>
// which available under GNU GPL v3 license.

package selector

import (
	"strings"

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

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func startsWith(s, prefix string) bool {
	return strings.HasPrefix(s, prefix)
}

func lower(s string) string {
	return strings.ToLower(s)
}
