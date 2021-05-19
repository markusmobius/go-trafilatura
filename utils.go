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

package trafilatura

import (
	"bytes"
	"io"
	"regexp"
	"strings"

	"github.com/go-shiori/dom"
	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var (
	rxImageExtension = regexp.MustCompile(`(?i)([^\s]+(\.(jpe?g|png|gif|bmp)))`)
	rxCharset        = regexp.MustCompile(`(?i)charset\s*=\s*([^;\s"]+)`)
)

// trim removes unnecessary spaces within a text string.
func trim(s string) string {
	s = strings.TrimSpace(s)
	return strings.Join(strings.Fields(s), " ")
}

func strWordCount(s string) int {
	return len(strings.Fields(s))
}

func strOr(args ...string) string {
	for i := 0; i < len(args); i++ {
		if args[i] != "" {
			return args[i]
		}
	}
	return ""
}

func strIn(s string, args ...string) bool {
	for i := 0; i < len(args); i++ {
		if args[i] == s {
			return true
		}
	}
	return false
}

func getRune(s string, idx int) rune {
	for i, r := range s {
		if i == idx {
			return r
		}
	}

	return -1
}

func isImageFile(imageSrc string) bool {
	return imageSrc != "" && rxImageExtension.MatchString(imageSrc)
}

// doNothing is placeholder function to store unused variables
// so Go formatter doesn't complain.
func doNothing(i ...interface{}) {}

func isSoftHyphen(r rune) bool {
	return r == '\u00AD'
}

// parseHTML parses a reader and try to convert the character encoding into UTF-8.
func parseHTML(r io.Reader) (*html.Node, error) {
	// Prepare tee for reusing reader
	buffer := bytes.NewBuffer(nil)
	tee := io.TeeReader(r, buffer)

	// Parse HTML normally
	doc, err := html.Parse(tee)
	if err != nil {
		return nil, err
	}

	// Look for charset in <meta> elements
	var customCharset string
	for _, meta := range dom.GetElementsByTagName(doc, "meta") {
		// Look in charset
		charsetAttr := dom.GetAttribute(meta, "charset")
		if charsetAttr != "" {
			customCharset = strings.TrimSpace(charsetAttr)
			break
		}

		// Look in http-equiv="Content-Type"
		content := dom.GetAttribute(meta, "content")
		httpEquiv := dom.GetAttribute(meta, "http-equiv")
		if httpEquiv == "Content-Type" && content != "" {
			matches := rxCharset.FindStringSubmatch(content)
			if len(matches) > 0 {
				customCharset = matches[1]
				break
			}
		}
	}

	// If there are no custom charset specified, assume it's utf-8
	if customCharset == "" {
		customCharset = "utf-8"
	}

	// Encode HTML in UTF-8 encoding.
	// While on it, remove soft hyphen that might be found in German articles.
	e, _ := charset.Lookup(customCharset)
	transformer := transform.Chain(e.NewDecoder(),
		norm.NFD,
		runes.Remove(runes.Predicate(isSoftHyphen)),
		norm.NFC)

	r = transform.NewReader(buffer, transformer)
	doc, err = html.Parse(r)
	if err != nil {
		return nil, err
	}

	return doc, nil
}
