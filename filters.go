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

func nodeHasHighLinkDensity(n *html.Node) ([]*html.Node, bool) {
	// Fetch links in node
	links := dom.GetElementsByTagName(n, "a")
	if len(links) == 0 {
		return nil, false
	}

	// Prepare limit and threshold
	var limitLength int
	var threshold float64

	switch {
	case dom.TagName(n) == "p":
		limitLength, threshold = 25, 0.9
	case n.NextSibling == nil:
		limitLength, threshold = 200, 0.66
	default:
		limitLength, threshold = 100, 0.66
	}

	// Check if text of this node is within limit
	text := dom.TextContent(n)
	text = strNormalize(text)
	if utf8.RuneCountInString(text) >= limitLength {
		return nil, false
	}

	// Collect link info
	linkLength, nShortLinks, nonEmptyLinks := collectLinkInfo(links)
	nNonEmptyLinks := len(nonEmptyLinks)
	if nNonEmptyLinks == 0 {
		// In this case, there are only empty links which obviously means
		// that this node doesn't have high link density. However, since it's
		// empty might as well delete it, so here we return true.
		return nil, true
	}

	// Check if links data surpass threshold
	if float64(linkLength) >= threshold*float64(nNonEmptyLinks) ||
		float64(nShortLinks)/float64(nNonEmptyLinks) >= threshold {
		return nonEmptyLinks, true
	}

	return nil, false
}

func tableHasHighLinkDensity(table *html.Node) bool {
	// Fetch links in table
	links := dom.GetElementsByTagName(table, "a")
	if len(links) == 0 {
		return false
	}

	// Check text length
	text := dom.TextContent(table)
	text = strNormalize(text)
	textLength := utf8.RuneCountInString(text)
	if textLength <= 250 {
		return false
	}

	// Collect link info
	linkTextLength, nShortLinks, nonEmptyLinks := collectLinkInfo(links)
	nNonEmptyLinks := len(nonEmptyLinks)
	if nNonEmptyLinks == 0 {
		// In this case, there are only empty links which obviously means
		// that this node doesn't have high link density. However, since it's
		// empty might as well delete it, so here we return true.
		return true
	}

	if (textLength <= 1000 && float64(linkTextLength) > float64(textLength)*0.8) ||
		(textLength > 1000 && float64(linkTextLength) > float64(textLength)*0.5) {
		return true
	}

	if float64(nShortLinks) > float64(len(links))*0.66 {
		return true
	}

	return false
}

func collectLinkInfo(links []*html.Node) (linkLength, nShortLinks int, nonEmptyLinks []*html.Node) {
	for _, link := range links {
		text := dom.TextContent(link)
		text = strNormalize(text)

		textLength := utf8.RuneCountInString(text)
		if textLength == 0 {
			continue
		}

		linkLength += textLength
		if textLength < 100 {
			nShortLinks++
		}

		nonEmptyLinks = append(nonEmptyLinks, link)
	}

	return
}
