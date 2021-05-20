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
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/go-shiori/dom"
	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding"
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

func isSoftHyphen(r rune) bool {
	return r == '\u00AD'
}

func containsErrorRune(bt []byte) bool {
	return bytes.ContainsRune(bt, utf8.RuneError)
}

// normalizeReaderEncoding convert text encoding from NFD to NFC.
// It also remove soft hyphen since apparently it's useless in web.
// See: https://web.archive.org/web/19990117011731/http://www.hut.fi/~jkorpela/shy.html
func normalizeReaderEncoding(r io.Reader) io.Reader {
	transformer := transform.Chain(norm.NFD, runes.Remove(runes.Predicate(isSoftHyphen)), norm.NFC)
	return transform.NewReader(r, transformer)
}

// parseHTMLDocument parses a reader and try to convert the character encoding into UTF-8.
func parseHTMLDocument(r io.Reader) (*html.Node, error) {
	// Check if there is invalid character.
	r, valid := validateRunes(r)

	// If already valid, we can parse and return it.
	if valid {
		r = normalizeReaderEncoding(r)
		return html.Parse(r)
	}

	// Find the decoder and parse HTML.
	r, charset := findHtmlCharset(r)
	r = transform.NewReader(r, charset.NewDecoder())
	r = normalizeReaderEncoding(r)
	return html.Parse(r)
}

// validateRunes check a reader to make sure all runes inside it is
// valid UTF-8 character. Since a reader only usable once, here
// we also return a mirror of the reader.
func validateRunes(r io.Reader) (io.Reader, bool) {
	buffer := bytes.NewBuffer(nil)
	tee := io.TeeReader(r, buffer)

	allValid := true
	scanner := bufio.NewScanner(tee)
	for scanner.Scan() {
		line := scanner.Bytes()
		if !utf8.Valid(line) || containsErrorRune(line) {
			allValid = false
			break
		}
	}

	ioutil.ReadAll(tee)
	return buffer, allValid
}

// validateRunes check HTML page in the reader to find charset
// that used in the HTML page. If none found, we assume it's
// UTF-8. Since a reader only usable once, here we also return
// a mirror of the reader.
func findHtmlCharset(r io.Reader) (io.Reader, encoding.Encoding) {
	// Prepare mirror
	buffer := bytes.NewBuffer(nil)
	tee := io.TeeReader(r, buffer)

	// Parse HTML normally
	doc, err := html.Parse(tee)
	if err != nil {
		return buffer, unicode.UTF8
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
		return buffer, unicode.UTF8
	}

	// Find the encoder
	e, _ := charset.Lookup(customCharset)
	if e == nil {
		e = unicode.UTF8
	}

	return buffer, e
}
