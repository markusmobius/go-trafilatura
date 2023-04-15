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

package trafilatura

import (
	"testing"

	"github.com/go-shiori/dom"
	"github.com/markusmobius/go-trafilatura/internal/etree"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/html"
)

func Test_processNode(t *testing.T) {
	var node *html.Node

	node = nodeFromStr(`<div><p></p>tail</div>`)
	node = dom.QuerySelector(node, "p")
	node = processNode(node, nil, defaultOpts)
	assert.Equal(t, "tail", etree.Text(node))
	assert.Equal(t, "", etree.Tail(node))

	node = nodeFromStr(`<ul><li></li>text in tail</ul>`)
	node = dom.QuerySelector(node, "li")
	node = processNode(node, nil, defaultOpts)
	assert.Equal(t, "text in tail", etree.Text(node))
	assert.Equal(t, "", etree.Tail(node))

	node = nodeFromStr(`<p><br/>tail</p>`)
	node = dom.QuerySelector(node, "br")
	node = processNode(node, nil, defaultOpts)
	assert.Equal(t, "", etree.Text(node))
	assert.Equal(t, "tail", etree.Tail(node))

	node = nodeFromStr(`<div><p>some text</p>tail</div>`)
	node = dom.QuerySelector(node, "p")
	node = processNode(node, nil, defaultOpts)
	assert.Equal(t, "some text", etree.Text(node))
	assert.Equal(t, "tail", etree.Tail(node))
}
