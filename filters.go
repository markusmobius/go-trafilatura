package trafilatura

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/go-shiori/dom"
	"github.com/markusmobius/go-trafilatura/etree"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/html"
)

var (
	rxHtmlLang   = regexp.MustCompile(`(?i)[a-z]{2}`)
	rxTextFilter = regexp.MustCompile(`(?i)\W*(Drucken|E-?Mail|Facebook|Flipboard|Google|Instagram|Linkedin|Mail|PDF|Pinterest|Pocket|Print|Reddit|Twitter|Whatsapp|Xing)$`)
)

// checkHtmlLanguage checks HTML meta-elements for language information and
// split the result in case there are several language.
func checkHtmlLanguage(doc *html.Node, targetLanguage string) bool {
	htmlNode := doc
	if dom.TagName(htmlNode) != "html" {
		htmlNode = dom.QuerySelector(doc, "html")
	}

	if htmlNode != nil {
		langAttr := dom.GetAttribute(htmlNode, "lang")
		for _, lang := range rxHtmlLang.FindAllString(langAttr, -1) {
			if lang == targetLanguage {
				return true
			}
		}
	}

	metaNodes := dom.QuerySelectorAll(doc, `meta[http-equiv="content-language"]`)
	for _, metaNode := range metaNodes {
		metaContent := dom.GetAttribute(metaNode, "content")
		for _, lang := range rxHtmlLang.FindAllString(metaContent, -1) {
			if lang == targetLanguage {
				return true
			}
		}
	}

	logrus.Warnln("language detection failed")
	return false
}

// textFilter filters out unwanted text
func textFilter(n *html.Node) bool {
	var testText string
	text, tail := etree.Text(n), etree.Tail(n)
	if text == "" && tail != "" {
		testText = tail
	} else {
		testText = text
	}

	if !textCharsTest(testText) {
		return true
	}

	for _, line := range strings.Split(testText, "\n") {
		if rxTextFilter.MatchString(line) {
			return true
		}
	}

	return false
}

// textCharsTest determine if a string is only composed of spaces and/or control characters.
func textCharsTest(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	return true
}

// duplicateTest checks for duplicate text within cache
func duplicateTest(element *html.Node, cache *Cache, opts Options) bool {
	var isDuplicate bool
	testString := trim(etree.IterText(element, " "))

	if utf8.RuneCountInString(testString) > opts.Config.MinDuplicateCheckSize {
		cacheVal, _ := cache.Get(testString)
		if cacheVal > opts.Config.MaxDuplicateCount {
			isDuplicate = true
		}
		cache.Put(testString, cacheVal+1)
	}

	return isDuplicate
}
