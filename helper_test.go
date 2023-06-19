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
	"io"
	nurl "net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-shiori/dom"
	"github.com/sirupsen/logrus"
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
		logrus.Panicln(err)
	}

	return f
}

// parseMockFile open then convert a mock file into html.Node.
func parseMockFile(mockFiles map[string]string, url string) *html.Node {
	f := openMockFile(mockFiles, url)
	defer f.Close()

	doc, err := dom.Parse(f)
	if err != nil {
		logrus.Panicln(err)
	}

	return doc
}

// extractMockFile open then extract content from a mock file.
func extractMockFile(mockFiles map[string]string, url string) *ExtractResult {
	// Open mock file
	f := openMockFile(mockFiles, url)
	defer f.Close()

	// Parse URL
	parsedURL, err := nurl.ParseRequestURI(url)
	if err != nil {
		logrus.Panicln(err)
	}

	// Extract
	opts := Options{OriginalURL: parsedURL, FallbackCandidates: &FallbackConfig{}}
	result, err := Extract(f, opts)
	if err != nil {
		logrus.Panicln(err)
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
