package trafilatura

import (
	"regexp"

	"github.com/go-shiori/dom"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/html"
)

var (
	rxPageLang = regexp.MustCompile(`(?i)[a-z]{2}`)
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
