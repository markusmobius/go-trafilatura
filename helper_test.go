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

	"github.com/go-shiori/dom"
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
		OriginalURL:        parsedURL,
		FallbackCandidates: &FallbackConfig{},
		IncludeLinks:       includeLinks}
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
