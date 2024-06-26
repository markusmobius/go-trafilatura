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
	nurl "net/url"
	"path"
	"strings"
)

// isAbsoluteURL checks if URL is valid and absolute.
func isAbsoluteURL(s string) (bool, *nurl.URL) {
	// Make sure URL is absolute
	url, err := nurl.ParseRequestURI(s)
	if err != nil {
		return false, nil
	}

	// We only want HTTP(s)
	if url.Scheme != "http" && url.Scheme != "https" {
		return false, nil
	}

	return true, url
}

// CreateAbsoluteURL convert url to absolute path based on base.
// However, if url is prefixed with hash (#), the url won't be changed.
func createAbsoluteURL(url string, base *nurl.URL) string {
	if url == "" || base == nil {
		return url
	}

	// If it is hash tag, return as it is
	if strings.HasPrefix(url, "#") {
		return url
	}

	// If it is data URI, return as it is
	if strings.HasPrefix(url, "data:") {
		return url
	}

	// If it is javascript URI, return as it is
	if strings.HasPrefix(url, "javascript:") {
		return url
	}

	// If it is already an absolute URL, return as it is
	tmp, err := nurl.ParseRequestURI(url)
	if err == nil && tmp.Scheme != "" && tmp.Hostname() != "" {
		return url
	}

	// Otherwise, resolve against base URI.
	// Normalize URL first.
	if !strings.HasPrefix(url, "/") {
		url = path.Join(base.Path, url)
	}

	tmp, err = nurl.Parse(url)
	if err != nil {
		return url
	}

	return base.ResolveReference(tmp).String()
}

func getDomainURL(url string) string {
	isAbsolute, parsedURL := isAbsoluteURL(url)
	if !isAbsolute {
		return ""
	}

	return parsedURL.Hostname()
}

func getBaseURL(url string) string {
	isAbsolute, parsedURL := isAbsoluteURL(url)
	if !isAbsolute {
		return ""
	}

	return parsedURL.Scheme + "://" + parsedURL.Hostname()
}

func validateURL(url string, baseURL *nurl.URL) (string, bool) {
	// If it's already an absolute URL, return it
	if isAbs, _ := isAbsoluteURL(url); isAbs {
		return url, true
	}

	// If not, try to convert it into absolute URL using base URL
	// instead of using domain name
	newURL := createAbsoluteURL(url, baseURL)
	if isAbs, _ := isAbsoluteURL(newURL); isAbs {
		return newURL, true
	}

	return url, false
}
