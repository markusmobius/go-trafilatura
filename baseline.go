package trafilatura

import (
	"encoding/json"
	"maps"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/go-shiori/dom"
	"github.com/markusmobius/go-trafilatura/v2/internal/etree"
	"golang.org/x/net/html"
)

var basicCleaningSelector = strings.Join([]string{
	`aside`,
	`footer`,
	`div[id*="footer"]`,
	`div[class*="footer"]`,
	`script`,
	`style`,
	`fencedframe`,
	`svg`,
	`template`,
}, ", ")

var rxCookieConsent = regexp.MustCompile(`(?i)cookie[-_]?(?:banner|bar|consent|law|notice|policy|description)|notice[-_]{0,2}cookie|consent[-_]?(?:banner|manager|sdk)|borlabs|cookiebot|cmplz|onetrust|moove[-_]?gdpr`)
var rxJSONContentHook = regexp.MustCompile(`articleBody|reviewBody|recipeInstructions|acceptedAnswer|"(?:Product|VideoObject|HowTo)"`)
var rxEmbeddedHTML = regexp.MustCompile(`(?i)</(?:a|abbr|address|article|aside|b|blockquote|body|br|caption|cite|code|dd|del|div|dl|dt|em|figcaption|figure|footer|h[1-6]|head|header|hr|html|i|img|ins|kbd|li|main|mark|nav|ol|p|pre|q|quote|s|section|small|span|strong|sub|summary|sup|table|tbody|td|tfoot|th|thead|time|title|tr|u|ul)>|<(?:a|abbr|address|article|aside|b|blockquote|body|br|caption|cite|code|dd|del|div|dl|dt|em|figcaption|figure|footer|h[1-6]|head|header|hr|html|i|img|ins|kbd|li|main|mark|nav|ol|p|pre|q|quote|s|section|small|span|strong|sub|summary|sup|table|tbody|td|tfoot|th|thead|time|title|tr|u|ul)(?:\s[^<>]*=[^<>]*)?/?>`)

var textBlockTags = sliceToMap(
	"address", "article", "aside", "blockquote", "br", "dd", "div", "dl", "dt",
	"figcaption", "figure", "footer", "form", "h1", "h2", "h3", "h4", "h5", "h6",
	"header", "hr", "li", "main", "nav", "ol", "p", "pre", "section", "summary",
	"table", "td", "th", "tr", "ul",
)

func basicCleaning(doc *html.Node) *html.Node {
	discardedElements := dom.QuerySelectorAll(doc, basicCleaningSelector)
	for _, element := range etree.IterDescendants(doc) {
		if rxCookieConsent.MatchString(dom.ClassName(element)) || rxCookieConsent.MatchString(dom.ID(element)) {
			discardedElements = append(discardedElements, element)
		}
	}
	for i := len(discardedElements) - 1; i >= 0; i-- {
		etree.Remove(discardedElements[i], true)
	}
	return doc
}

func jsonItems(value any) []any {
	if value == nil {
		return nil
	}
	if items, ok := value.([]any); ok {
		return items
	}
	return []any{value}
}

func decodeJSON(input string, output any) error {
	err := json.Unmarshal([]byte(input), output)
	if err == nil {
		return nil
	}
	var escaped strings.Builder
	quoted, backslash, changed := false, false, false
	for index := 0; index < len(input); index++ {
		character := input[index]
		if quoted && !backslash && character < 0x20 {
			escaped.WriteString(`\u00`)
			escaped.WriteByte("0123456789abcdef"[character>>4])
			escaped.WriteByte("0123456789abcdef"[character&0xf])
			changed = true
		} else {
			escaped.WriteByte(character)
		}
		if backslash {
			backslash = false
		} else if quoted && character == '\\' {
			backslash = true
		} else if character == '"' {
			quoted = !quoted
		}
	}
	if !changed {
		return err
	}
	return json.Unmarshal([]byte(escaped.String()), output)
}

func walkJSONContent(value any, bodies, teasers *[]string) {
	for _, item := range jsonItems(value) {
		object, ok := item.(map[string]any)
		if !ok {
			continue
		}
		for _, key := range []string{"articleBody", "reviewBody"} {
			if text, ok := object[key].(string); ok && text != "" {
				*bodies = append(*bodies, text)
			}
		}
		for _, key := range []string{"recipeInstructions", "step"} {
			for _, step := range jsonItems(object[key]) {
				if text, ok := step.(string); ok {
					*bodies = append(*bodies, text)
				} else if step, ok := step.(map[string]any); ok {
					for _, sub := range append([]any{step}, jsonItems(step["itemListElement"])...) {
						if sub, ok := sub.(map[string]any); ok {
							if text, ok := sub["text"].(string); ok {
								*bodies = append(*bodies, text)
							}
						}
					}
				}
			}
		}
		if answer, ok := object["acceptedAnswer"].(map[string]any); ok {
			if text, ok := answer["text"].(string); ok {
				*bodies = append(*bodies, text)
			}
		}
		for _, schemaType := range getSchemaTypes(object, true) {
			if strIn(schemaType, "product", "videoobject") {
				if text, ok := object["description"].(string); ok {
					*teasers = append(*teasers, text)
				}
				break
			}
		}
		for _, key := range []string{"@graph", "mainEntity"} {
			walkJSONContent(object[key], bodies, teasers)
		}
	}
}

func collectJSONContent(doc *html.Node) (bodies, teasers []string) {
	for _, script := range dom.QuerySelectorAll(doc, `script[type="application/ld+json"]`) {
		text := dom.TextContent(script)
		if !rxJSONContentHook.MatchString(text) {
			continue
		}
		var value any
		if decodeJSON(text, &value) == nil {
			walkJSONContent(value, &bodies, &teasers)
		}
	}
	if preload := dom.QuerySelector(doc, `div[id="data-preloaded"]`); preload != nil {
		var topics map[string]json.RawMessage
		if json.Unmarshal([]byte(dom.GetAttribute(preload, "data-preloaded")), &topics) == nil {
			for _, key := range slices.Sorted(maps.Keys(topics)) {
				if !strings.HasPrefix(key, "topic_") {
					continue
				}
				var encoded string
				var topic map[string]any
				if json.Unmarshal(topics[key], &encoded) != nil || json.Unmarshal([]byte(encoded), &topic) != nil {
					continue
				}
				stream, _ := topic["post_stream"].(map[string]any)
				for _, post := range jsonItems(stream["posts"]) {
					if post, ok := post.(map[string]any); ok {
						if cooked, ok := post["cooked"].(string); ok {
							bodies = append(bodies, cooked)
						}
					}
				}
			}
		}
	}
	return
}

func renderBaselineText(raw string) string {
	raw = removeControlCharacters(html.UnescapeString(raw))
	if rxEmbeddedHTML.MatchString(raw) {
		container := etree.Element("div")
		dom.SetInnerHTML(container, raw)
		return trim(dom.TextContent(container))
	}
	return trim(raw)
}

func buildBaselineBody(texts []string, deduplicate bool) (*html.Node, string) {
	body := etree.Element("body")
	var content strings.Builder
	contentLength := 0
	for _, text := range texts {
		text = removeControlCharacters(text)
		textLength := utf8.RuneCountInString(text)
		if text == "" || (deduplicate && textLength > minDuplicateLength && contentLength <= dedupeScanCap && strings.Contains(content.String(), text)) {
			continue
		}
		etree.SetText(etree.SubElement(body, "p"), text)
		if content.Len() > 0 {
			content.WriteByte('\n')
			contentLength++
		}
		content.WriteString(text)
		contentLength += textLength
	}
	return body, content.String()
}

func baseline(doc *html.Node) (*html.Node, string) {
	if doc == nil {
		return etree.Element("body"), ""
	}
	doc = dom.Clone(doc, true)
	bodies, teasers := collectJSONContent(doc)
	for index := range bodies {
		bodies[index] = renderBaselineText(bodies[index])
	}
	if body, text := buildBaselineBody(bodies, true); utf8.RuneCountInString(text) > 100 {
		return body, text
	}
	doc = basicCleaning(doc)
	var articles []string
	maxLength := 0
	for _, article := range dom.QuerySelectorAll(doc, "article") {
		nested := false
		for parent := article.Parent; parent != nil; parent = parent.Parent {
			if dom.TagName(parent) == "article" {
				nested = true
				break
			}
		}
		if !nested {
			text := trim(dom.TextContent(article))
			if length := utf8.RuneCountInString(text); length > 100 {
				articles = append(articles, text)
				maxLength = max(maxLength, length)
			}
		}
	}
	if len(articles) > 0 {
		articles = slices.DeleteFunc(articles, func(text string) bool {
			return utf8.RuneCountInString(text)*5 < maxLength
		})
		return buildBaselineBody(articles, false)
	}
	var paragraphs []string
	for _, element := range etree.Iter(doc, "blockquote", "code", "p", "pre", "q", "quote") {
		paragraphs = append(paragraphs, trim(dom.TextContent(element)))
	}
	if body, text := buildBaselineBody(paragraphs, true); utf8.RuneCountInString(text) > 100 {
		return body, text
	}
	for index := range teasers {
		teasers[index] = renderBaselineText(teasers[index])
	}
	teaserBody, teaserText := buildBaselineBody(teasers, true)
	var parts []string
	var collectText func(*html.Node)
	collectText = func(node *html.Node) {
		if node.Type == html.TextNode {
			if text := trim(node.Data); text != "" {
				parts = append(parts, text)
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			collectText(child)
		}
	}
	if body := dom.QuerySelector(doc, "body"); body != nil {
		collectText(body)
	}
	body, text := buildBaselineBody([]string{strings.Join(parts, "\n")}, false)
	if utf8.RuneCountInString(teaserText) > 100 && utf8.RuneCountInString(teaserText) > utf8.RuneCountInString(text) {
		return teaserBody, teaserText
	}
	return body, text
}

func html2txt(input any) string {
	var doc *html.Node
	switch value := input.(type) {
	case *html.Node:
		doc = value
	case string:
		tokenizer := html.NewTokenizer(strings.NewReader(value))
		for {
			tokenType := tokenizer.Next()
			if tokenType == html.ErrorToken {
				return ""
			}
			if tokenType == html.StartTagToken || tokenType == html.SelfClosingTagToken {
				tag, _ := tokenizer.TagName()
				if !strIn(string(tag), "html", "head", "body") {
					return ""
				}
				break
			}
		}
		var err error
		doc, err = html.Parse(strings.NewReader(value))
		if err != nil {
			return ""
		}
	}
	if doc == nil {
		return ""
	}
	doc = dom.Clone(doc, true)
	if body := dom.QuerySelector(doc, "body"); body != nil {
		doc = body
	}
	doc = basicCleaning(doc)
	var text strings.Builder
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		block := inMap(dom.TagName(node), textBlockTags)
		if block {
			text.WriteByte(' ')
		}
		if node.Type == html.TextNode {
			text.WriteString(removeControlCharacters(node.Data))
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
		if block {
			text.WriteByte(' ')
		}
	}
	visit(doc)
	return trim(text.String())
}
