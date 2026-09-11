// This file is part of go-trafilatura, Go package for extracting readable
// content, comments and metadata from a web page. Source available in
// <https://github.com/markusmobius/go-trafilatura>.
//
// Copyright (C) 2021 Markus Mobius
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code in this file is ported from <https://github.com/adbar/trafilatura>
// which available under Apache 2.0 license.

package trafilatura

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/go-shiori/dom"
	"github.com/markusmobius/go-trafilatura/internal/etree"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/html"
)

func Test_Baseline(t *testing.T) {
	var doc *html.Node
	var result string

	// Blank document
	doc = docFromStr("")
	_, result = baseline(doc)
	assert.Empty(t, result)

	// Invalid HTML
	doc = docFromStr(`<invalid html>`)
	_, result = baseline(doc)
	assert.Empty(t, result)

	// Extract from <article> tag
	doc = docFromStr(`<html><body><article>` +
		strings.Repeat(`The article consists of this text.`, 10) +
		`</article></body></html>`)
	_, result = baseline(doc)
	assert.NotEmpty(t, result)

	doc = docFromStr("<html><body><article><b>The article consists of this text.</b></article></body></html>")
	_, result = baseline(doc)
	assert.NotEmpty(t, result)

	// Extract from quote
	doc = docFromStr("<html><body><blockquote>This is only a quote but it is better than nothing.</blockquote></body></html>")
	_, result = baseline(doc)
	assert.NotEmpty(t, result)

	// Invalid JSON
	doc = docFromStr(`
		<html><body>
			<script type="application/ld+json">
				{"articleBody": "This is the article body, it has to be long enough to fool the length threshold which is set at len 100."  # invalid JSON
			</script>
		</body></html>`)
	_, result = baseline(doc)
	assert.Empty(t, result)

	// JSON OK
	doc = docFromStr(`
		<html><body>
			<script type="application/ld+json">
				{
					"@type": "Article",
					"articleBody": "This is the article body, it has to be long enough to fool the length threshold which is set at len 100."
				}
			</script>
		</body></html>`)
	_, result = baseline(doc)
	assert.Equal(t, "This is the article body, it has to be long enough to fool the length threshold which is set at len 100.", result)

	// JSON malformed
	doc = docFromStr(`
		<html><body>
			<script type="application/ld+json">
				{
					"@type": "Article",
					"articleBody": "<p>This is the article body, it has to be long enough to fool the length threshold which is set at len 100.</p>"
				}
			</script>
		</body></html>`)
	_, result = baseline(doc)
	assert.Equal(t, "This is the article body, it has to be long enough to fool the length threshold which is set at len 100.", result)

	// Real-world examples
	doc = docFromStr(`<html>
		<body>
			<script type="application/ld+json">
				{
					"description": "In letzter Zeit kam man am Begriff \"Hygge\", was so viel wie \"angenehm\" oder \"gemütlich\" bedeutet, ja nicht vorbei. Jetzt macht ihm ein neuer Glücks-Trend ...",
					"image": [
						{
							"name": "Mit der Ikigai-Methode wirst du glücklicher",
							"url": "https:\/\/image.brigitte.de\/10973004\/uncropped-0-0\/7d00b2658fd0a3b19e1b161f4657cc20\/Xw\/ikigai--1-.jpg",
							"width": "2048",
							"height": "1366",
							"@type": "ImageObject"
						},
						{
							"name": "Mit der Ikigai-Methode wirst du glücklicher",
							"url": "https:\/\/image.brigitte.de\/10973004\/16x9-1280-720\/bf947c7c24167d7c0adae0be10942d57\/Uf\/ikigai--1-.jpg",
							"width": "1280",
							"height": "720",
							"@type": "ImageObject"
						},
						{
							"name": "Mit der Ikigai-Methode wirst du glücklicher",
							"url": "https:\/\/image.brigitte.de\/10973004\/16x9-938-528\/bf947c7c24167d7c0adae0be10942d57\/JK\/ikigai--1-.jpg",
							"width": "938",
							"height": "528",
							"@type": "ImageObject"
						},
						{
							"name": "Mit der Ikigai-Methode wirst du glücklicher",
							"url": "https:\/\/image.brigitte.de\/10973004\/large1x1-622-622\/f5544b7d67e1be04f7729b130e7e0485\/KN\/ikigai--1-.jpg",
							"width": "622",
							"height": "622",
							"@type": "ImageObject"
						}
					],
					"mainEntityOfPage": {
						"@id": "https:\/\/www.brigitte.de\/liebe\/persoenlichkeit\/ikigai-macht-dich-sofort-gluecklicher--10972896.html",
						"@type": "WebPage"
					},
					"headline": "Ikigai macht dich sofort glücklicher!",
					"datePublished": "2019-06-19T14:29:08+0000",
					"dateModified": "2019-06-19T14:29:10+0000",
					"author": { "name": "BRIGITTE.de", "@type": "Organization" },
					"publisher": {
						"name": "BRIGITTE.de",
						"logo": {
							"url": "https:\/\/image.brigitte.de\/11476842\/uncropped-0-0\/f19537e97b9189bf0f25ce924168bedb\/kK\/bri-logo-schema-org.png",
							"width": "167",
							"height": "60",
							"@type": "ImageObject"
						},
						"@type": "Organization"
					},
					"articleBody": "In letzter Zeit kam man am Begriff \"Hygge\" (\"gemütlich\" oder \"angenehm\") nicht vorbei. Jetzt macht ihm ein neuer Glücks-Trend Konkurrenz: \"Ikigai\". Bist du glücklich? Schwierige Frage, nicht wahr? Viele von uns müssen da erst mal überlegen.",
					"@type": "NewsArticle"
				}
			</script>
		</body>
	</html>`)
	_, result = baseline(doc)
	assert.True(t, strings.HasPrefix(result, "In letzter Zeit kam man"))
	assert.True(t, strings.HasSuffix(result, "erst mal überlegen."))

	doc = docFromStr("<html><body><div>   Document body...   </div><script> console.log('Hello world') </script></body></html>")
	_, result = baseline(doc)
	assert.Equal(t, "Document body...", result)
}

func Test_Baseline_Upstream22(test *testing.T) {
	fullText := strings.Repeat("Complete article content. ", 8)
	for _, schema := range []any{
		[]any{map[string]any{"articleBody": fullText}},
		map[string]any{"@graph": []any{map[string]any{"reviewBody": fullText}}},
		map[string]any{"recipeInstructions": []any{map[string]any{"text": fullText}}},
		map[string]any{"@type": "HowTo", "step": []any{map[string]any{"itemListElement": []any{map[string]any{"text": fullText}}}}},
		map[string]any{"mainEntity": map[string]any{"acceptedAnswer": map[string]any{"text": fullText}}},
	} {
		encoded, err := json.Marshal(schema)
		assert.NoError(test, err)
		doc := docFromStr(`<html><body><script type="application/ld+json">` + string(encoded) + `</script></body></html>`)
		original := dom.OuterHTML(doc)
		_, text := baseline(doc)
		assert.Equal(test, trim(fullText), text)
		assert.Equal(test, original, dom.OuterHTML(doc))
	}
	assert.Equal(test, "i<b and c>d", renderBaselineText("i<b and c>d"))
	assert.Equal(test, "Embedded text", renderBaselineText("&lt;p&gt;Embedded &lt;b&gt;text&lt;/b&gt;&lt;/p&gt;"))
	unrelated := docFromStr(`<html><body><script type="application/ld+json">{"step":"` + fullText + `"}</script></body></html>`)
	_, unrelatedText := baseline(unrelated)
	assert.Empty(test, unrelatedText)
	unicodeText := strings.Repeat("\u4e2d", 80_000)
	unicodeBody, unicodeResult := buildBaselineBody([]string{unicodeText, unicodeText}, true)
	assert.Equal(test, unicodeText, unicodeResult)
	assert.Len(test, dom.Children(unicodeBody), 1)

	doc := docFromStr(`<html><body><script type="application/ld+json">{"articleBody":"Rejected short JSON"}</script><article>` + fullText + `</article></body></html>`)
	body, text := baseline(doc)
	assert.Equal(test, trim(fullText), text)
	assert.Equal(test, 1, len(dom.Children(body)))

	doc = docFromStr(`<html><body><blockquote><p>` + fullText + `</p></blockquote></body></html>`)
	body, text = baseline(doc)
	assert.Equal(test, trim(fullText), text)
	assert.Equal(test, 1, len(dom.Children(body)))

	doc = docFromStr(`<html><body><article>` + fullText + `</article><article>` + fullText + `</article></body></html>`)
	body, _ = baseline(doc)
	assert.Equal(test, 2, len(dom.Children(body)))

	topic, err := json.Marshal(map[string]any{"post_stream": map[string]any{"posts": []any{map[string]any{"cooked": "<p>" + fullText + "</p>"}, false}}})
	assert.NoError(test, err)
	preload, err := json.Marshal(map[string]string{"topic_1": string(topic)})
	assert.NoError(test, err)
	doc = docFromStr(`<html><body><div id="data-preloaded"></div></body></html>`)
	dom.SetAttribute(dom.QuerySelector(doc, "div"), "data-preloaded", string(preload))
	_, text = baseline(doc)
	assert.Equal(test, trim(fullText), text)
}

func Test_HTML2Text_Upstream22(test *testing.T) {
	doc := docFromStr(`<html><body class="cookies-not-set"><div>A<b>B</b>C</div><p>Second<br>line</p><div class="CookieConsentBanner">Discard consent text</div><svg>discard</svg><template>discard</template></body></html>`)
	original := dom.OuterHTML(doc)
	assert.Equal(test, "ABC Second line", html2txt(doc))
	assert.Equal(test, original, dom.OuterHTML(doc))
	assert.Equal(test, "", html2txt(nil))
	body, text := buildBaselineBody([]string{"repeat", "repeat"}, true)
	assert.Equal(test, "repeat\nrepeat", text)
	assert.Equal(test, 2, len(dom.Children(body)))
	assert.NotEmpty(test, etree.ToString(body))
}

func Test_Python220_Baseline(test *testing.T) {
	jsonDocument := func(payload, body string) string {
		return `<html><head><script type="application/ld+json">` + payload + `</script></head><body>` + body + `</body></html>`
	}
	textOf := func(input string) string {
		_, text := baseline(docFromStr(input))
		return text
	}
	encode := func(value any) string {
		encoded, err := json.Marshal(value)
		assert.NoError(test, err)
		return string(encoded)
	}
	paragraph := "Real paragraph content that should be extracted by the paragraph strategy, comfortably long enough for the gate."

	test.Run("test_baseline", func(test *testing.T) {
		body, text := baseline(nil)
		assert.Equal(test, "body", body.Data)
		assert.Empty(test, text)
		body, text = baseline(docFromStr(""))
		assert.Equal(test, "body", body.Data)
		assert.Empty(test, text)
		assert.Empty(test, textOf("<invalid html>"))
		for _, sample := range []struct{ input, expected string }{
			{"<html><body><article>" + strings.Repeat("The article consists of this text.", 10) + "</article></body></html>", "The article consists of this text."},
			{"<html><body><article><b>The article consists of this text.</b></article></body></html>", "The article consists of this text."},
			{"<html><body><quote>This is only a quote but it is better than nothing.</quote></body></html>", "This is only a quote but it is better than nothing."},
		} {
			assert.Contains(test, textOf(sample.input), sample.expected)
		}
		article := "This is the article body, it has to be long enough to fool the length threshold which is set at len 100."
		assert.Empty(test, textOf(jsonDocument(`{"articleBody": "`+article+`"  # invalid JSON`, "")))
		assert.Greater(test, len(textOf(jsonDocument(`{"@type":"Article","articleBody":"`+article+`"}`, ""))), 100)
		text = textOf(jsonDocument(`{"@type":"Article","articleBody":"<p>`+article+`<\/p>"}`, ""))
		assert.True(test, strings.HasPrefix(text, "This is the article body"))
		assert.NotContains(test, text, "<p>")
		assert.Equal(test, "Document body...", textOf("<html><body><div>   Document body...   </div><script> console.log('Hello world') </script></body></html>"))
	})
	test.Run("test_baseline_strategy_fallthrough", func(test *testing.T) {
		text := textOf(jsonDocument(`{"articleBody": "Too short."}`, "<p>"+paragraph+"</p>"))
		assert.Contains(test, text, paragraph)
		assert.Greater(test, len(text), 100)
	})
	test.Run("test_baseline_jsonld_shapes", func(test *testing.T) {
		body := "Body text from structured data, made comfortably long enough to pass the one hundred character length threshold."
		for _, payload := range []string{
			`{"articleBody": "` + body + `"}`,
			`{"@context": "https://schema.org", "@graph": [{"@type": "Article", "articleBody": "` + body + `"}]}`,
			`[{"@type": "Article", "articleBody": "` + body + `"}]`,
			`[42, {"@type": "Article", "articleBody": "` + body + `"}]`,
		} {
			assert.Equal(test, 1, strings.Count(textOf(jsonDocument(payload, "")), body))
		}
	})
	test.Run("test_baseline_jsonld_content", func(test *testing.T) {
		for _, sample := range []struct{ name, body, expected, forbidden string }{
			{"raw controls", "First line of the body text\n\twith a raw newline and a tab inside the JSON string, comfortably long enough to pass the gate.", "First line of the body text with a raw newline", ""},
			{"markup", "Opening text<br/>more after a break element<div>and inside a div</div> comfortably long enough to pass the one hundred character gate.", "Opening text", "<"},
			{"escaped markup", "&lt;p&gt;Escaped paragraph body text that should come out clean, comfortably long enough to pass the one hundred character gate.&lt;/p&gt;", "Escaped paragraph body text", "&lt;"},
			{"comparison", "Comparing a < b in plain prose should stay intact, with padding to exceed the one hundred character gate.", "a < b", ""},
			{"expression", "For x<y and z>0 the series converges absolutely, with plenty of extra padding to exceed the length gate here.", "x<y and z>0", ""},
			{"tag letters", "If p<q then u<s or i<b and c>d holds, with extra padding to comfortably exceed the one hundred character gate.", "i<b and c>d holds", ""},
		} {
			test.Run(sample.name, func(test *testing.T) {
				text := textOf(jsonDocument(`{"articleBody": "`+sample.body+`"}`, ""))
				assert.Contains(test, text, sample.expected)
				if sample.forbidden != "" {
					assert.NotContains(test, text, sample.forbidden)
				}
				if sample.name == "tag letters" {
					assert.Contains(test, text, "p<q")
				}
			})
		}
	})
	test.Run("test_baseline_teaser_does_not_shadow_longer_body", func(test *testing.T) {
		description := "Short marketing product blurb standing in as a JSON-LD teaser, just past the one hundred character content gate here."
		var paragraphs strings.Builder
		for index := range 5 {
			paragraphs.WriteString(fmt.Sprintf("<div>Real article paragraph number %d with genuine sentence content that only the whole-body dump captures from these bare div elements on the page.</div>", index))
		}
		assert.Contains(test, textOf(jsonDocument(`{"@type": "Product", "description": "`+description+`"}`, paragraphs.String())), "Real article paragraph number 3")
	})
	test.Run("test_baseline_teaser_used_when_body_shorter", func(test *testing.T) {
		description := "Full product description carrying the actual page content, comfortably longer than the sparse body below and past the gate."
		assert.Contains(test, textOf(jsonDocument(`{"@type": "Product", "description": "`+description+`"}`, "<div>x</div>")), "Full product description")
	})
	test.Run("test_baseline_description_teaser_tier", func(test *testing.T) {
		teaser := "Description teaser standing in for page content, comfortably longer than the one hundred character gate here."
		fulltext := "The full body text which must take precedence over the description teaser and clears the gate comfortably."
		for _, schemaType := range []string{"Product", "VideoObject"} {
			test.Run(schemaType, func(test *testing.T) {
				payload := encode(map[string]any{"@type": schemaType, "description": teaser})
				assert.Contains(test, textOf(jsonDocument(payload, "")), teaser)
				both := encode([]any{map[string]any{"@type": schemaType, "description": teaser}, map[string]any{"@type": "Article", "articleBody": fulltext}})
				for _, input := range []string{jsonDocument(both, ""), jsonDocument(payload, "<p>"+fulltext+"</p>")} {
					text := textOf(input)
					assert.Contains(test, text, fulltext)
					assert.NotContains(test, text, teaser)
				}
			})
		}
	})
	test.Run("test_baseline_jsonld_double_embed", func(test *testing.T) {
		for _, sample := range []struct{ text, payload string }{
			{"Article body text embedded twice via duplicated JSON-LD scripts, comfortably longer than the one hundred character strategy gate.", `{"articleBody": "%s"}`},
			{"Video description standing in for page content here, comfortably longer than the one hundred character gate.", `{"@type": "VideoObject", "description": "%s"}`},
		} {
			script := `<script type="application/ld+json">` + fmt.Sprintf(sample.payload, sample.text) + `</script>`
			assert.Equal(test, 1, strings.Count(textOf("<html><head>"+script+script+"</head><body></body></html>"), sample.text))
		}
	})
	test.Run("test_baseline_control_chars_tree_input", func(test *testing.T) {
		control := "before\x01after, with plenty of padding words to pass the one hundred character strategy gate comfortably."
		for _, input := range []string{jsonDocument(`{"articleBody": "`+control+`"}`, ""), "<html><body><p>" + control + "</p></body></html>", "<html><body><div>" + control + "</div></body></html>"} {
			text := textOf(input)
			assert.Greater(test, len(text), 100)
			assert.NotContains(test, text, "\x01")
			assert.Contains(test, text, "before")
			assert.Contains(test, text, "after")
		}
	})
	test.Run("test_baseline_nested_paragraph_not_duplicated", func(test *testing.T) {
		paragraph := "Quoted paragraph text repeated by nesting here, made comfortably longer than the one hundred character strategy gate threshold."
		assert.Equal(test, 1, strings.Count(textOf("<html><body><blockquote>Attribution line: <p>"+paragraph+"</p></blockquote></body></html>"), paragraph))
	})
	test.Run("test_baseline_paragraph_dedup_keeps_short_repeats", func(test *testing.T) {
		long := "The annual conference will be held in Berlin this coming September, organizers said today."
		short := "held in Berlin"
		assert.Equal(test, 2, strings.Count(textOf("<html><body><p>"+long+"</p><p>"+short+"</p></body></html>"), short))
	})
	test.Run("test_baseline_dedup_keeps_cross_boundary_paragraph", func(test *testing.T) {
		paragraphs := []string{"The meeting concluded with remarks about the annual budget", "planning process that begins next week according to officials.", "annual budget planning process that begins next week"}
		body, _ := buildBaselineBody(paragraphs, true)
		children := dom.Children(body)
		if assert.Len(test, children, 3) {
			assert.Equal(test, paragraphs[2], etree.Text(children[2]))
		}
	})
	test.Run("test_baseline_article_nested_not_duplicated", func(test *testing.T) {
		inner := strings.Repeat("Inner article content sentence repeated to build a properly long body text here. ", 3)
		assert.Equal(test, 1, strings.Count(textOf("<html><body><article>Outer wrapper. <article>"+inner+"</article></article></body></html>"), strings.TrimSpace(inner)))
	})
	test.Run("test_baseline_schema_properties", func(test *testing.T) {
		for _, sample := range []struct {
			payload  string
			expected []string
		}{
			{`{"@type":"Recipe","recipeInstructions":[{"@type":"HowToStep","text":"Mix the flour with sugar and butter until the dough is smooth and pliable in texture."},{"@type":"HowToStep","text":"Bake for thirty five minutes at one hundred eighty degrees until golden brown on top."}]}`, []string{"Mix the flour", "Bake for thirty five minutes"}},
			{`{"@type":"Recipe","recipeInstructions":["Combine all the dry ingredients thoroughly in a large mixing bowl before adding any liquids.","Pour the batter into the tin and bake until a skewer inserted in the centre comes out clean."]}`, []string{"Combine all the dry ingredients", "Pour the batter into the tin"}},
			{`{"@type":"FAQPage","mainEntity":[{"@type":"Question","name":"How?","acceptedAnswer":{"@type":"Answer","text":"This is the accepted answer to the question, written with a comfortable amount of words to pass the one hundred character gate."}}]}`, []string{"This is the accepted answer"}},
		} {
			text := textOf(jsonDocument(sample.payload, ""))
			for _, expected := range sample.expected {
				assert.Contains(test, text, expected)
			}
		}
	})
	test.Run("test_baseline_discourse_preload", func(test *testing.T) {
		topic := `{"post_stream":{"posts":[{"cooked":"<p>First forum post with a good amount of substantive discussion content in it.</p>"},{"cooked":"<p>Second forum post continuing the thread with more useful information for everyone.</p>"}]}}`
		preloaded := html.EscapeString(encode(map[string]string{"site_settings": "ignored", "topic_12": topic}))
		text := textOf(`<html><body><div id="data-preloaded" data-preloaded="` + preloaded + `"></div></body></html>`)
		assert.Contains(test, text, "First forum post")
		assert.Contains(test, text, "Second forum post")
		assert.NotContains(test, text, "<p>")
	})
	test.Run("test_baseline_discourse_preload_malformed", func(test *testing.T) {
		for _, preloaded := range []string{"[]", `"just a string"`, html.EscapeString(encode(map[string]string{"topic_12": `["not", "a", "dict"]`}))} {
			assert.Contains(test, textOf(`<html><body><div id="data-preloaded" data-preloaded="`+preloaded+`"></div><p>`+paragraph+`</p></body></html>`), paragraph)
		}
	})
	test.Run("test_build_body_dedupe_cap", func(test *testing.T) {
		filler := "Opening paragraph content long enough to push the accumulated text past a tiny cap."
		duplicate := "This exact paragraph repeats and is well above the duplicate length gate for sure."
		_, text := buildBaselineBody([]string{filler, duplicate, duplicate}, true)
		assert.Equal(test, 1, strings.Count(text, duplicate))
		test.Run("patched_cap_50", func(test *testing.T) {
			test.Skip("Python monkeypatches a module global; Go's dedupeScanCap is a compile-time constant. The default-cap assertion above is portable.")
		})
	})
	test.Run("test_baseline_howto", func(test *testing.T) {
		step := "Measure the table and add the overhang you want on every side before cutting any fabric at all."
		payload := `{"@type":"HowTo","step":[{"@type":"HowToSection","itemListElement":{"@type":"HowToDirection","text":"` + step + ` Then hem the edges neatly all around the cloth."}}]}`
		assert.Contains(test, textOf(jsonDocument(payload, "")), step)
		payload = `{"@type":"HowTo","step":[{"@type":"HowToStep","text":"Own step text: preheat the oven to a moderate temperature first.","itemListElement":[{"@type":"HowToDirection","text":"Set the dial to one hundred and eighty degrees."}]}]}`
		text := textOf(jsonDocument(payload, ""))
		assert.Contains(test, text, "preheat the oven")
		assert.Contains(test, text, "one hundred and eighty")
	})
	test.Run("test_baseline_element_input_not_mutated", func(test *testing.T) {
		doc := docFromStr("<html><body><aside>side text</aside><p>" + strings.Repeat("Real paragraph content on the page. ", 4) + "</p></body></html>")
		_, text := baseline(doc)
		assert.Contains(test, text, "Real paragraph content")
		assert.NotContains(test, text, "side text")
		assert.NotNil(test, dom.QuerySelector(doc, "aside"))
	})
	test.Run("test_baseline_article_dominance", func(test *testing.T) {
		main := strings.Repeat("Main article content sentence repeated to build a properly dominant body. ", 8)
		teaser := "Related article teaser text that comfortably exceeds the length gate...."
		text := textOf("<html><body><article>" + main + "</article><article>" + teaser + teaser[:40] + "</article></body></html>")
		assert.Contains(test, text, "Main article content")
		assert.NotContains(test, text, "Related article teaser")
		post := "Forum post number %d with a comparable amount of discussion text in every single post here."
		var body strings.Builder
		for index := range 3 {
			body.WriteString("<article>" + fmt.Sprintf(post, index) + "</article>")
		}
		text = textOf("<html><body>" + body.String() + "</body></html>")
		for index := range 3 {
			assert.Contains(test, text, fmt.Sprintf(post, index))
		}
	})
	test.Run("test_html2txt", func(test *testing.T) {
		for _, sample := range []struct {
			input    any
			expected string
		}{
			{"<html><body>Here is the body text</body></html>", "Here is the body text"},
			{"", ""}, {"123", ""}, {"<html></html>", ""}, {"<html><body/></html>", ""},
			{"<html><body><style>font-size: 8pt</style><p>ABC</p></body></html>", "ABC"},
			{"<html><body><p>First block.</p><p>Second block.</p><h2>Heading</h2>line<br>break</body></html>", "First block. Second block. Heading line break"},
			{"<html><body><div>a<p>b</p></div></body></html>", "a b"},
			{"<html><body><p>Hyper<b>link</b></p></body></html>", "Hyperlink"},
			{docFromStr("<div><p>bare element text</p></div>"), "bare element text"},
			{`<?xml version="1.0"?><feed xmlns="http://www.w3.org/2005/Atom"><title>Feed title</title><entry><title>Entry one</title><summary>Summary one</summary></entry><entry><title>Entry two</title><summary>Summary two</summary></entry></feed>`, ""},
			{"<html><body><svg><title>Icon name</title></svg><template><p>markup</p></template><p>visible</p></body></html>", "visible"},
			{docFromStr("<html><body><p>before\x01after visible text</p></body></html>"), "beforeafter visible text"},
		} {
			assert.Equal(test, sample.expected, html2txt(sample.input), sample.input)
		}
		doc := docFromStr("<html><body><aside>side</aside><p>content</p></body></html>")
		assert.Equal(test, "content", html2txt(doc))
		assert.Contains(test, dom.TextContent(doc), "side")
	})
}
