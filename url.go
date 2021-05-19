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

func extractDomainURL(url string) string {
	isAbsolute, parsedURL := isAbsoluteURL(url)
	if !isAbsolute {
		return ""
	}

	return parsedURL.Hostname()
}
