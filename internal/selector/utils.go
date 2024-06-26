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
