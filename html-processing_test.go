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
	"testing"

	"github.com/go-shiori/dom"
	"github.com/markusmobius/go-trafilatura/internal/etree"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/html"
)

func Test_processNode(t *testing.T) {
	var node *html.Node

	node = etree.FromString(`<div><p></p>tail</div>`)
	node = dom.QuerySelector(node, "p")
	node = processNode(node, nil, defaultOpts)
	assert.Equal(t, "tail", etree.Text(node))
	assert.Equal(t, "", etree.Tail(node))

	node = etree.FromString(`<ul><li></li>text in tail</ul>`)
	node = dom.QuerySelector(node, "li")
	node = processNode(node, nil, defaultOpts)
	assert.Equal(t, "text in tail", etree.Text(node))
	assert.Equal(t, "", etree.Tail(node))

	node = etree.FromString(`<p><br/>tail</p>`)
	node = dom.QuerySelector(node, "br")
	node = processNode(node, nil, defaultOpts)
	assert.Equal(t, "", etree.Text(node))
	assert.Equal(t, "tail", etree.Tail(node))

	node = etree.FromString(`<div><p>some text</p>tail</div>`)
	node = dom.QuerySelector(node, "p")
	node = processNode(node, nil, defaultOpts)
	assert.Equal(t, "some text", etree.Text(node))
	assert.Equal(t, "tail", etree.Tail(node))
}
