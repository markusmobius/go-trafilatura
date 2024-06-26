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

	"github.com/stretchr/testify/assert"
)

func Test_isAbsoluteURL(t *testing.T) {
	assertURL := func(url string, expected bool) {
		isAbs, _ := isAbsoluteURL(url)
		assert.Equal(t, expected, isAbs)
	}

	assertURL("http://www.test.org:7ERT/test", false)
	assertURL("ntp://www.test.org/test", false)
	assertURL("ftps://www.test.org/test", false)
	assertURL("http://t.g/test", true)
	assertURL("http://test.org/test", true)
}
