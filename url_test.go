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
