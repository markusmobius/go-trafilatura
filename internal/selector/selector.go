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
	"github.com/go-shiori/dom"
	"golang.org/x/net/html"
)

type Rule func(*html.Node) bool

func Query(root *html.Node, selector Rule) *html.Node {
	for _, e := range dom.GetElementsByTagName(root, "*") {
		if selector(e) {
			return e
		}
	}
	return nil
}

func QueryAll(root *html.Node, selector Rule) []*html.Node {
	var matches []*html.Node
	for _, e := range dom.GetElementsByTagName(root, "*") {
		if selector(e) {
			matches = append(matches, e)
		}
	}
	return matches
}
