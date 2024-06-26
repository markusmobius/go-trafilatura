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

package main

import (
	nurl "net/url"
	"os"
	"path"
	"regexp"
	"strings"
)

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func validateURL(url string) (*nurl.URL, bool) {
	parsedURL, err := nurl.ParseRequestURI(url)
	if err != nil {
		return nil, false
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, false
	}

	return parsedURL, true
}

func isValidURL(url string) bool {
	_, valid := validateURL(url)
	return valid
}

func sliceToMap(strings ...string) map[string]struct{} {
	result := make(map[string]struct{})
	for _, s := range strings {
		result[s] = struct{}{}
	}
	return result
}

func rxFromString(str string) (*regexp.Regexp, error) {
	if str == "" {
		return nil, nil
	}

	return regexp.Compile(str)
}

func nameFromURL(url *nurl.URL) string {
	urlPath := strings.Trim(url.Path, "/")
	domain := strings.TrimPrefix(url.Hostname(), "www.")

	newName := strings.ReplaceAll(domain, ".", "-")
	if urlPath != "" {
		urlPath = path.Base(urlPath)
		urlPath = strings.ReplaceAll(urlPath, "/", "-")
		urlPath = strings.ReplaceAll(urlPath, ".", "-")
		newName += "-" + urlPath
	}

	return newName
}

// createAbsoluteURL convert url to absolute path based on base.
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
