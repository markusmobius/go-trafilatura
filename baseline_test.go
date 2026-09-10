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
