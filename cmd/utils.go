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

package main

import (
	"io"
	"net/http"
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

func getFileContentType(r io.Reader) (string, error) {
	// Only the first 512 bytes are used to sniff the content type.
	buffer := make([]byte, 512)
	_, err := r.Read(buffer)
	if err != nil {
		return "", err
	}

	contentType := http.DetectContentType(buffer)
	return contentType, nil
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
