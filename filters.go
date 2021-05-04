package trafilatura

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/go-shiori/dom"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/html"
)

var (
	rxPageLang   = regexp.MustCompile(`(?i)[a-z]{2}`)
	rxTextFilter = regexp.MustCompile(`(?i)\W*(Drucken|E-?Mail|Facebook|Flipboard|Google|Instagram|Linkedin|Mail|PDF|Pinterest|Pocket|Print|Reddit|Twitter|Whatsapp|Xing)$`)
)

func checkHtmlLanguage(doc *html.Node, targetLanguage string) bool {
	if htmlNode := dom.QuerySelector(doc, "html"); htmlNode != nil {
		langAttr := dom.GetAttribute(htmlNode, "lang")
		for _, lang := range rxPageLang.FindAllString(langAttr, -1) {
			if lang == targetLanguage {
				return true
			}
		}
	}

	metaNodes := dom.QuerySelectorAll(doc, `meta[http-equiv="content-language"]`)
	for _, metaNode := range metaNodes {
		metaContent := dom.GetAttribute(metaNode, "content")
		for _, lang := range rxPageLang.FindAllString(metaContent, -1) {
			if lang == targetLanguage {
				return true
			}
		}
	}

	logrus.Warnln("language detection failed")
	return false
}

func textFilter(n *html.Node) bool {
	text := dom.TextContent(n)
	if !textCharsTest(text) {
		return true
	}

	for _, line := range strings.Split(text, "\n") {
		if rxTextFilter.MatchString(line) {
			return true
		}
	}

	return false
}

func textCharsTest(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	return true
}

func duplicateTest(n *html.Node, cache *Cache) bool {
	text := dom.TextContent(n)
	text = strNormalize(text)

	isDuplicate := false
	if utf8.RuneCountInString(text) > minDuplicateCheckSize {
		duplicateCount, _ := cache.Get(text)
		if duplicateCount > maxDuplicateCount {
			isDuplicate = true
		}

		cache.Put(text, duplicateCount+1)
	}

	return isDuplicate
}
