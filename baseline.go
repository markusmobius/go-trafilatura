package trafilatura

import (
	"encoding/json"
	"strings"
	"unicode/utf8"

	"github.com/go-shiori/dom"
	"github.com/markusmobius/go-trafilatura/internal/etree"
	"golang.org/x/net/html"
)

func basicCleaning(doc *html.Node) *html.Node {
	discardedElements := dom.QuerySelectorAll(doc, "aside,footer,script,style")
	for i := len(discardedElements) - 1; i >= 0; i-- {
		discardedElements[i].Parent.RemoveChild(discardedElements[i])
	}
	return doc
}

// baseline uses baseline extraction function targeting text paragraphs and/or JSON metadata.
func baseline(doc *html.Node) (*html.Node, string) {
	var tmpText string
	postBody := etree.Element("body")
	if doc == nil {
		return postBody, ""
	}

	// Scrape JSON+LD for article body
	for _, script := range dom.QuerySelectorAll(doc, `script[type="application/ld+json"]`) {
		// Get the json text inside the script
		jsonLdText := dom.TextContent(script)
		jsonLdText = strings.TrimSpace(jsonLdText)
		jsonLdText = html.UnescapeString(jsonLdText)
		if jsonLdText == "" {
			continue
		}

		// Decode JSON text, assuming it is an object
		data := map[string]any{}
		err := json.Unmarshal([]byte(jsonLdText), &data)
		if err != nil {
			continue
		}

		// Find article body recursively
		var articleBody string
		var findArticleBody func(obj map[string]any)

		findArticleBody = func(obj map[string]any) {
			for key, value := range obj {
				switch v := value.(type) {
				case string:
					v = trim(v)
					if strings.ToLower(key) == "articlebody" && v != "" {
						if strings.Contains(v, "<p>") {
							tmp := dom.CreateElement("div")
							dom.SetInnerHTML(tmp, v)
							articleBody = trim(dom.TextContent(tmp))
						} else {
							articleBody = v
						}
						return
					}

				case map[string]any:
					findArticleBody(v)

				case []any:
					for _, item := range v {
						if obj, isObject := item.(map[string]any); isObject {
							findArticleBody(obj)
						}
					}
				}
			}
		}

		findArticleBody(data)
		if articleBody != "" {
			p := etree.SubElement(postBody, "p")
			etree.SetText(p, articleBody)
			tmpText += " " + articleBody
		}
	}

	tmpText = trim(tmpText)
	if utf8.RuneCountInString(tmpText) > 100 {
		return postBody, tmpText
	}

	// Basic tree cleaning
	doc = basicCleaning(doc)

	// Scrape from article tag
	articleElement := dom.QuerySelector(doc, "article")
	if articleElement != nil {
		articleText := trim(dom.TextContent(articleElement))
		if utf8.RuneCountInString(articleText) > 100 {
			p := etree.SubElement(postBody, "p")
			etree.SetText(p, articleText)
			tmpText += " " + articleText
		}
	}

	if len(dom.Children(postBody)) > 0 {
		tmpText = trim(tmpText)
		return postBody, tmpText
	}

	// Scrape from text paragraphs
	results := make(map[string]struct{})
	for _, element := range etree.Iter(doc, "blockquote", "pre", "q", "code", "p") {
		entry := trim(dom.TextContent(element))
		if _, exist := results[entry]; !exist {
			p := etree.SubElement(postBody, "p")
			etree.SetText(p, entry)
			tmpText += " " + entry
			results[entry] = struct{}{}
		}
	}

	tmpText = trim(tmpText)
	if utf8.RuneCountInString(tmpText) > 100 {
		return postBody, tmpText
	}

	// Default strategy: clean the tree and take everything
	if body := dom.QuerySelector(doc, "body"); body != nil {
		text := trim(etree.IterText(body, "\n"))
		if utf8.RuneCountInString(text) > 100 {
			elem := etree.SubElement(postBody, "p")
			etree.SetText(elem, text)
			return postBody, text
		}
	}

	// New fallback
	text := trim(dom.TextContent(doc))
	elem := etree.SubElement(postBody, "p")
	etree.SetText(elem, text)
	return postBody, text
}
