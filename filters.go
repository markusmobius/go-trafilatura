package trafilatura

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/go-shiori/dom"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/html"
)

const (
	minDuplicateCheckSize = 100
	maxDuplicateCount     = 2
)

var (
	rxPageLang   = regexp.MustCompile(`(?i)[a-z]{2}`)
	rxTextFilter = regexp.MustCompile(`(?i)\W*(Drucken|E-?Mail|Facebook|Flipboard|Google|Instagram|Linkedin|Mail|PDF|Pinterest|Pocket|Print|Reddit|Twitter|Whatsapp|Xing)$`)
)

func checkPageLanguage(doc *html.Node, targetLanguage string) bool {
	if htmlNode := dom.QuerySelector(doc, "html"); htmlNode != nil {
		langAttr := dom.GetAttribute(htmlNode, "lang")
		for _, lang := range rxPageLang.FindAllString(langAttr, -1) {
			if lang == targetLanguage {
				return true
			}
		}
	}

	metaNodes := dom.QuerySelectorAll(doc, `[http-equiv="content-language"]`)
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
	text = strings.TrimSpace(text)
	if text == "" {
		return true
	}

	for _, line := range strings.Split(text, "\n") {
		if rxTextFilter.MatchString(line) {
			return true
		}
	}

	return false
}

func duplicateFilter(cache *Cache, n *html.Node) bool {
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

func nodeTextFilter(node *html.Node, deduplicate bool, cache *Cache) bool {
	// Make sure text is not empty
	text := dom.TextContent(node)
	text = strings.TrimSpace(text)
	if text == "" {
		return true
	}

	// If text doesn't contain any word, stop
	if !rxWords.MatchString(text) {
		return true
	}

	// Check filter
	if textFilter(node) {
		return true
	}

	if deduplicate && cache != nil && duplicateFilter(cache, node) {
		return true
	}

	return false
}

func commentsNodeFilter(n *html.Node, cache *Cache, deduplicate bool, potentialTags map[string]struct{}) bool {
	// Make sure node is one of the potential comments
	if _, isPotential := potentialTags[dom.TagName(n)]; !isPotential {
		return false
	}

	// Make sure node is not empty and not duplicated
	if nodeTextFilter(n, deduplicate, cache) {
		return false
	}

	return true
}
