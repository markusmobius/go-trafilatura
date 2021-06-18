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
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/gogs/chardet"
	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding/unicode"
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

// ====================== INFORMATION ======================
// Methods below these point are used for making sure that
// a HTML document is parsed using UTF-8 encoder, so these
// are not exist in original Trafilatura.
// =========================================================

// normalizeTextEncoding convert text encoding from NFD to NFC.
// It also remove soft hyphen since apparently it's useless in web.
// See: https://web.archive.org/web/19990117011731/http://www.hut.fi/~jkorpela/shy.html
func normalizeTextEncoding(r io.Reader) io.Reader {
	fnSoftHyphen := func(r rune) bool { return r == '\u00AD' }
	softHyphenSet := runes.Predicate(fnSoftHyphen)
	transformer := transform.Chain(norm.NFD, runes.Remove(softHyphenSet), norm.NFC)
	return transform.NewReader(r, transformer)
}

// parseHTMLDocument parses a reader and try to convert the character encoding into UTF-8.
func parseHTMLDocument(r io.Reader) (*html.Node, error) {
	// Split the reader using tee
	content, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	// Detect page encoding
	res, err := chardet.NewHtmlDetector().DetectBest(content)
	if err != nil {
		return nil, err
	}

	pageEncoding, _ := charset.Lookup(res.Charset)
	if pageEncoding == nil {
		pageEncoding = unicode.UTF8
	}

	// Parse HTML using the page encoding
	r = bytes.NewReader(content)
	r = transform.NewReader(r, pageEncoding.NewDecoder())
	r = normalizeTextEncoding(r)
	return html.Parse(r)
}
