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
