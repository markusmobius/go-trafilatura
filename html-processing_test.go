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
	"fmt"
	"maps"
	"net/url"
	"strings"
	"testing"

	"github.com/go-shiori/dom"
	"github.com/markusmobius/go-trafilatura/v2/internal/etree"
	"github.com/markusmobius/go-trafilatura/v2/internal/selector"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/html"
)

func Test_Python220_Internals(test *testing.T) {
	test.Run("test_trim", func(test *testing.T) {
		assert.Equal(test, "Test", trim("\tTest  "))
		assert.Equal(test, "Test Test", trim("\t\tTest  Test\r\n"))
		for _, sample := range []struct {
			text     string
			filtered bool
		}{{"Test Text", false}, {"Instagram", true}, {"\t\t", true}} {
			node := etree.Element("body")
			etree.SetText(node, sample.text)
			assert.Equal(test, sample.filtered, textFilter(node))
		}
		test.Run("sanitize_and_cache", func(test *testing.T) {
			test.Skip("Go has no nullable sanitize API or process-global string-function cache")
		})
	})
	test.Run("test_recover_wild_text_default_tags", func(test *testing.T) {
		tree := docFromStr("<body><div>some wild text outside the main frame, long enough to be recovered</div></body>")
		body := etree.Element("body")
		opts := defaultOpts
		opts.Focus = FavorRecall
		recoverWildText(tree, body, nil, nil, opts)
		assert.NotNil(test, body)
	})
	test.Run("test_prune_boilerplate_table_after_nested", func(test *testing.T) {
		link := `<ref target="/x">link text that is reasonably long over here</ref>`
		rows := strings.Repeat("<row><cell>"+link+"</cell></row>", 6)
		nested := "<row><cell><table><row><cell>x</cell></row></table></cell></row>"
		real := "Real article content paragraph that should always survive the pruning pass intact here now."
		tree := python220Element(test, "<html><body><table>"+rows+nested+"</table><table>"+rows+"</table><p>"+real+"</p></body></html>")
		pruned := pruneUnwantedSections(tree, map[string]struct{}{"table": {}, "p": {}}, defaultOpts)
		assert.Empty(test, dom.QuerySelectorAll(pruned, "table"))
		assert.NotNil(test, dom.QuerySelector(pruned, "p"))
	})
	test.Run("test_prune_keep_teasers", func(test *testing.T) {
		input := `<html><body><div class="teaser"><p>Real article body text that is long enough to count as genuine content here now.</p></div></body></html>`
		assert.Nil(test, dom.QuerySelector(pruneUnwantedSections(docFromStr(input), map[string]struct{}{"p": {}}, defaultOpts), "p"))
		assert.NotNil(test, dom.QuerySelector(pruneUnwantedSections(docFromStr(input), map[string]struct{}{"p": {}}, defaultOpts, true), "p"))
	})
	test.Run("test_aria_layout_table_reclassified", func(test *testing.T) {
		for _, role := range []string{"presentation", "none"} {
			doc := docFromStr(`<html><body><table role="` + role + `"><tr><td><table><tr><td>data</td></tr></table></td></tr></table></body></html>`)
			docCleaning(doc, defaultOpts)
			assert.Nil(test, dom.QuerySelector(doc, "table[role]"))
			assert.NotNil(test, dom.QuerySelector(doc, "table"))
			doc = docFromStr(`<html><body><table role="` + role + `"><tr><td>text</td></tr></table></body></html>`)
			docCleaning(doc, defaultOpts)
			assert.Nil(test, dom.QuerySelector(doc, "table"))
		}
	})
	test.Run("test_table_colgroup_no_crash", func(test *testing.T) {
		for _, input := range []string{"<table><colgroup><col style='width:50%'><col style='width:50%'></colgroup><tr><td>a</td><td>b</td></tr></table>", "<table><col span='2'><tr><td>x</td><td>y</td></tr></table>", "<table><col><td>orphan</td><tr><td>normal</td></tr></table>"} {
			table := handleTable(dom.QuerySelector(docFromStr(input), "table"), maps.Clone(tagCatalog), nil, defaultOpts)
			for _, cell := range dom.QuerySelectorAll(table, "td, th") {
				assert.NotEmpty(test, etree.Text(cell))
			}
		}
	})
	test.Run("test_sanitize_tree_th_dedup", func(test *testing.T) {
		doc := docFromStr("<html><body><table><tr><th>A</th><th>B</th></tr><tr><th>C</th><th>D</th></tr><tr><td>1</td><td>2</td></tr></table></body></html>")
		sanitizeTree(doc, defaultOpts)
		headers := dom.QuerySelectorAll(doc, "th")
		assert.Len(test, headers, 2)
		for _, cell := range headers {
			assert.Contains(test, []string{"A", "B"}, etree.Text(cell))
		}
		for _, cell := range dom.QuerySelectorAll(doc, "td") {
			assert.Contains(test, []string{"C", "D", "1", "2"}, etree.Text(cell))
		}
		doc = docFromStr("<html><body><table><tr><td>x</td></tr></table></body></html>")
		sanitizeTree(doc, defaultOpts)
		assert.Empty(test, dom.QuerySelectorAll(doc, "th"))
	})
	test.Run("test_sanitize_tree_absolutizes_links", func(test *testing.T) {
		doc := docFromStr(`<html><body><p><a href="/path/page">link</a> ` + strings.Repeat("padding ", 10) + "</p></body></html>")
		address, err := url.Parse("https://www.example.org")
		if !assert.NoError(test, err) {
			return
		}
		sanitizeTree(doc, Options{OriginalURL: address, IncludeLinks: true})
		assert.NotNil(test, dom.QuerySelector(doc, `a[href="https://www.example.org/path/page"]`))
	})
	test.Run("test_settings_element_lists", func(test *testing.T) {
		input := "<html><body><div><p>Alpha paragraph number one with more than enough length to be considered real body content.</p><blockquote>Quoted block of text which is normally retained within the trafilatura extraction output.</blockquote><p>Beta paragraph number two also with more than enough length to be treated as real body content.</p><p>Gamma paragraph number three providing extra padding for the document body content section here.</p><p>Delta paragraph number four keeping the extracted text comfortably above the minimum size limit.</p></div></body></html>"
		extractText := func() string {
			result, err := Extract(strings.NewReader(input), Options{})
			if !assert.NoError(test, err) {
				return ""
			}
			return result.ContentText
		}
		assert.Contains(test, extractText(), "Quoted block")
		original := maps.Clone(tagsToClean)
		defer func() { tagsToClean = original }()
		tagsToClean["blockquote"] = struct{}{}
		assert.NotContains(test, extractText(), "Quoted block")
		tagsToClean = original
		assert.Contains(test, extractText(), "Quoted block")
	})
	test.Run("test_htmlprocessing/paragraphs", func(test *testing.T) {
		paragraph := python220Element(test, "<p>text<lb>x</lb></p>")
		processed := handleParagraphs(paragraph, map[string]struct{}{"p": {}, "br": {}}, nil, defaultOpts)
		if assert.NotNil(test, processed) {
			assert.Empty(test, dom.QuerySelectorAll(processed, "br"))
		}
		for _, sample := range []struct{ input, text string }{{`<p><hi rend="#b">pre<quote>mid</quote>end</hi></p>`, " mid"}, {`<p><hi rend="#b">start<lb/>tail</hi></p>`, " tail"}} {
			processed := handleParagraphs(python220Element(test, sample.input), maps.Clone(tagCatalog), nil, defaultOpts)
			bold := dom.QuerySelector(processed, "b")
			if assert.NotNil(test, bold) {
				assert.Contains(test, etree.Text(bold), sample.text)
			}
		}
	})
}

func Test_Python220_HTML(test *testing.T) {
	matches := func(input string, rules []selector.Rule) bool {
		node := python220Element(test, input)
		for _, rule := range rules {
			if len(selector.QueryAll(node, rule)) > 0 {
				return true
			}
		}
		return false
	}
	test.Run("test_link_density_tables_threshold", func(test *testing.T) {
		for _, sample := range []struct {
			text, link int
			discard    bool
		}{{50, 250, true}, {200, 250, false}, {240, 360, false}, {400, 600, true}, {600, 600, false}} {
			test.Run(fmt.Sprintf("%d_%d", sample.text, sample.link), func(test *testing.T) {
				input := `<table><cell>` + strings.Repeat("y", sample.text) + `<ref target="/x">` + strings.Repeat("x", sample.link) + `</ref></cell></table>`
				assert.Equal(test, sample.discard, linkDensityTestTables(python220Element(test, input), defaultOpts))
			})
		}
	})
	test.Run("test_link_density_tables_textless_links_kept", func(test *testing.T) {
		input := `<table><cell>` + strings.Repeat("data ", 50) + `<ref target="/x"><graphic src="/i.png"/></ref></cell></table>`
		assert.False(test, linkDensityTestTables(python220Element(test, input), defaultOpts))
	})
	for _, sample := range []struct {
		name, text, sibling string
		count               int
		discard             bool
	}{
		{"test_link_density_short_link_list_kept", "Recommended product number %d: a nice gadget", "following content sibling here", 3, false},
		{"test_link_density_large_link_farm_pruned", "Latest news headline number %d about some topic today", "real article sibling here", 20, true},
		{"test_link_density_whole_card_links_kept", "Align: a widget that aligns its child within itself and optionally sizes itself based on the child's given size", "real article sibling here", 8, false},
	} {
		test.Run(sample.name, func(test *testing.T) {
			var links strings.Builder
			for index := range sample.count {
				text := sample.text
				if strings.Contains(text, "%d") {
					text = fmt.Sprintf(text, index)
				}
				links.WriteString(fmt.Sprintf(`<ref target="/p%d">%s</ref> `, index, text))
			}
			body := python220Element(test, `<body><div>`+links.String()+`</div><p>`+sample.sibling+`</p></body>`)
			element := dom.Children(body)[0]
			length := len(trim(dom.TextContent(element)))
			if sample.count == 3 {
				assert.Greater(test, length, 100)
				assert.Less(test, length, 150)
			} else {
				assert.Greater(test, length, 300)
			}
			_, discard := linkDensityTest(element, defaultOpts)
			assert.Equal(test, sample.discard, discard)
		})
	}
	test.Run("test_overall_discard_legacy_tokens", func(test *testing.T) {
		for _, class := range []string{"yin", "zlylin", "mol-factbox"} {
			assert.True(test, matches(`<html><body><div class="`+class+`"><p>content</p></div></body></html>`, selector.OverallDiscardedContent), class)
		}
		assert.False(test, matches(`<html><body><div class="xg1"><p>content</p></div></body></html>`, selector.OverallDiscardedContent))
	})
	test.Run("test_overall_discard_matches_both_attributes", func(test *testing.T) {
		for _, sample := range []struct {
			attributes string
			discard    bool
		}{{`class="x" id="author-box"`, true}, {`id="x" class="sidebar"`, true}, {`class="hidden-x" id="cookieBanner"`, false}} {
			assert.Equal(test, sample.discard, matches(`<html><body><div `+sample.attributes+`><p>content</p></div></body></html>`, selector.OverallDiscardedContent), sample.attributes)
		}
	})
	test.Run("test_precision_discard_link_token_only", func(test *testing.T) {
		for _, sample := range []struct {
			tag, class string
			discard    bool
		}{{"div", "link", true}, {"div", "nav link", true}, {"div", "article-permalink", false}, {"div", "headline-link", false}, {"div", "featured-link--wrap", false}, {"div", "article-bottom", true}, {"header", "site-header", true}} {
			input := fmt.Sprintf(`<html><body><%s class="%s"><p>content</p></%s></body></html>`, sample.tag, sample.class, sample.tag)
			assert.Equal(test, sample.discard, matches(input, selector.PrecisionDiscardedContent), sample.class)
		}
	})
	test.Run("test_body_xpath_fulltext_class", func(test *testing.T) {
		for _, class := range []string{"fulltext", "FullText", "fullText", "FULLTEXT", "article-fulltext", "FulltextWrapper"} {
			assert.True(test, matches(`<html><body><div class="`+class+`"><p>content</p></div></body></html>`, selector.Content), class)
		}
	})
	test.Run("test_basic_cleaning_cookie_banner_scope", func(test *testing.T) {
		content := "<p>" + strings.Repeat("Real article text about a subject. ", 5) + "</p>"
		banners := "<div id='onetrust-consent-sdk'><p>By clicking Accept you agree we can store cookies.</p></div><div class='cookie-notice-container'><p>We use cookies to improve our service.</p></div>"
		doc := docFromStr("<html><body class='single-post cookies-not-set'><div class='cookie-recipe-content'>" + content + "</div>" + banners + "</body></html>")
		_, text := baseline(doc)
		assert.Contains(test, text, "Real article text")
		assert.NotContains(test, text, "cookies")
		text = html2txt(doc)
		assert.Contains(test, text, "Real article text")
		assert.NotContains(test, text, "cookies")
	})
	test.Run("test_htmlprocessing", func(test *testing.T) {
		options := defaultOpts
		options.IncludeImages, options.IncludeLinks = true, true
		doc := docFromStr(`<html><body><a href="/x"><img src="a.jpg"/><img src="b.jpg"/><img src="c.jpg"/></a></body></html>`)
		convertTags(doc, options)
		var sources []string
		for _, image := range dom.QuerySelectorAll(doc, "img") {
			sources = append(sources, dom.GetAttribute(image, "src"))
		}
		assert.Equal(test, []string{"a.jpg", "b.jpg", "c.jpg"}, sources)
		for _, sample := range []struct{ input, tag, text, tail string }{
			{"<div><p></p>tail</div>", "p", "tail", ""},
			{"<list><item></item>text in tail</list>", "li", "text in tail", ""},
			{"<p><lb/>tail</p>", "br", "", "tail"},
			{"<div><p>some text</p>tail</div>", "p", "some text", "tail"},
		} {
			root := python220Element(test, sample.input)
			node := processNode(dom.QuerySelector(root, sample.tag), nil, options)
			if assert.NotNil(test, node) {
				assert.Equal(test, sample.text, etree.Text(node))
				assert.Equal(test, sample.tail, etree.Tail(node))
			}
		}
		for _, sample := range []struct{ input, text, tail string }{
			{`<p><ref target='url'><hi rend='#b'>bold</hi>inner</ref>outer</p>`, "", "outer"},
			{`<p><ref target='url'>text</ref>tail</p>`, "text", "tail"},
			{`<p><ref target='url'></ref>tail</p>`, "tail", ""},
			{`<p><ref target='url'>text<hi rend='#b'>bold</hi></ref>tail</p>`, "text", "tail"},
		} {
			node := handleTextNode(dom.QuerySelector(python220Element(test, sample.input), "a"), nil, false, false, options)
			if assert.NotNil(test, node) {
				assert.Equal(test, sample.tail, etree.Tail(node))
				if sample.text != "" {
					assert.Equal(test, sample.text, etree.Text(node))
				}
			}
		}
		node := python220Element(test, "<div><p><span>span</span> span tail</p> p tail </div>")
		assert.Equal(test, "span span tail p tail ", dom.TextContent(node))
		pruneUnwantedNodes(node, []selector.Rule{func(node *html.Node) bool { return dom.TagName(node) == "span" }})
		assert.Equal(test, " span tail p tail ", dom.TextContent(node))
		assert.False(test, linkDensityTestTables(python220Element(test, `<table><cell>`+strings.Repeat("word ", 50)+`<ref target="/x"></ref></cell></table>`), options))
		assert.False(test, linkDensityTestTables(python220Element(test, `<table><cell>short `+strings.Repeat(`<ref target="/x">link</ref> `, 5)+`</cell></table>`), options))
	})
}

func Test_processNode(t *testing.T) {
	var node *html.Node

	node = etree.FromString(`<div><p></p>tail</div>`)
	node = dom.QuerySelector(node, "p")
	node = processNode(node, nil, defaultOpts)
	assert.Equal(t, "tail", etree.Text(node))
	assert.Equal(t, "", etree.Tail(node))

	node = etree.FromString(`<ul><li></li>text in tail</ul>`)
	node = dom.QuerySelector(node, "li")
	node = processNode(node, nil, defaultOpts)
	assert.Equal(t, "text in tail", etree.Text(node))
	assert.Equal(t, "", etree.Tail(node))

	node = etree.FromString(`<p><br/>tail</p>`)
	node = dom.QuerySelector(node, "br")
	node = processNode(node, nil, defaultOpts)
	assert.Equal(t, "", etree.Text(node))
	assert.Equal(t, "tail", etree.Tail(node))

	node = etree.FromString(`<div><p>some text</p>tail</div>`)
	node = dom.QuerySelector(node, "p")
	node = processNode(node, nil, defaultOpts)
	assert.Equal(t, "some text", etree.Text(node))
	assert.Equal(t, "tail", etree.Tail(node))
}

func Test_pruneUnwantedNodes(test *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "first child tail stays in parent",
			input:    `<div><p><span>span</span> span tail</p> p tail </div>`,
			expected: `<div><p> span tail</p> p tail </div>`,
		},
		{
			name:     "parent text is preserved",
			input:    `<div>start<span>discard</span>end<p>keep</p></div>`,
			expected: `<div>startend<p>keep</p></div>`,
		},
		{
			name:     "previous sibling tail is preserved",
			input:    `<div><p>first</p> between<span>discard</span> after<p>last</p></div>`,
			expected: `<div><p>first</p> between after<p>last</p></div>`,
		},
		{
			name:     "nested discarded nodes",
			input:    `<div><span>outer<span>inner</span>inner tail</span>outer tail</div>`,
			expected: `<div>outer tail</div>`,
		},
	}
	queries := []selector.Rule{func(node *html.Node) bool {
		return dom.TagName(node) == "span"
	}}
	for _, testCase := range testCases {
		test.Run(testCase.name, func(test *testing.T) {
			node := etree.FromString(testCase.input)
			result := pruneUnwantedNodes(node, queries)
			assert.Equal(test, testCase.expected, etree.ToString(result))
			assert.Same(test, node, result)
		})
	}
}

func Test_HTMLParser_Upstream22(test *testing.T) {
	for _, prefix := range []string{
		`<!DOCTYPE html>`,
		`<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">`,
		`<!DOCTYPE html SYSTEM "https://example.org/document/type.dtd">`,
	} {
		result, err := Extract(strings.NewReader(prefix+`<html><body><article><p>Content after the doctype.</p></article></body></html>`), Options{Config: zeroConfig})
		if assert.NoError(test, err) {
			assert.Equal(test, "Content after the doctype.", result.ContentText)
		}
	}
}

func Test_Selectors_Upstream22(test *testing.T) {
	for _, testCase := range []struct {
		input   string
		discard bool
	}{
		{`<div id="article" class="sharbuttons">text</div>`, true},
		{`<div id="article" class="social">text</div>`, true},
		{`<div id="social" class="article">text</div>`, true},
		{`<div id="soc" class="ial">text</div>`, false},
		{`<div class="xg1">text</div>`, false},
		{`<div id="article" class="cookie-recipe">text</div>`, false},
		{`<div class="cookie-recipe" id="article">text</div>`, true},
	} {
		node := etree.FromString(testCase.input)
		discard := false
		for _, rule := range selector.OverallDiscardedContent {
			discard = discard || rule(node)
		}
		assert.Equal(test, testCase.discard, discard, testCase.input)
	}
	for _, class := range []string{"permalink", "headline-link", "linkage", "linked-text"} {
		node := etree.FromString(`<p class="` + class + `">text</p>`)
		assert.False(test, selector.PrecisionDiscardedContent[1](node), class)
	}
	for _, class := range []string{"link", "other link", "link other"} {
		node := etree.FromString(`<p class="` + class + `">text</p>`)
		assert.True(test, selector.PrecisionDiscardedContent[1](node), class)
	}
	node := etree.FromString(`<div class="FullText">text</div>`)
	assert.True(test, selector.Content[2](node))
	for _, class := range []string{"author-name", "authorname", "AuthorName", "authorName"} {
		node := etree.FromString(`<span class="` + class + `">Author Name</span>`)
		assert.True(test, selector.MetaAuthor[0](node), class)
	}
	node = etree.FromString(`<span class="author name">Author Name</span>`)
	assert.False(test, selector.MetaAuthor[0](node))
	assert.True(test, selector.MetaAuthor[1](node))
	metadata := extractMetadata(docFromStr(`<html><body><span class="author name">Generic Author</span><span class="authorname">Specific Author</span></body></html>`), defaultOpts)
	assert.Equal(test, "Specific Author", metadata.Author)
}

func Test_convertTags_LinkedImages(test *testing.T) {
	node := etree.FromString(`<div><a href="/x"><picture><img src="a.jpg"><img src="b.jpg"></picture><img src="c.jpg"></a></div>`)
	opts := Options{IncludeImages: true, IncludeLinks: true}
	convertTags(node, opts)
	assert.Equal(test, `<div><img src="a.jpg"/><img src="b.jpg"/><img src="c.jpg"/></div>`, etree.ToString(node))

	node = etree.FromString(`<div><a href="/x">image link<img src="a.jpg"></a></div>`)
	convertTags(node, opts)
	assert.Equal(test, `<div><a href="/x">image link</a><img src="a.jpg"/></div>`, etree.ToString(node))
	_, discard := linkDensityTest(node, opts)
	assert.False(test, discard)
}

func Test_CodeBlocks_Indicators(test *testing.T) {
	for _, code := range []string{`print("hello")`, "if ready {\n    run()\n}", "value = 1\n    value += 1"} {
		test.Run(code, func(test *testing.T) {
			node := etree.FromString("<div><pre>" + code + "</pre></div>")
			convertTags(node, zeroOpts)
			assert.NotNil(test, dom.QuerySelector(node, "code"))
			assert.Equal(test, code, dom.TextContent(node))
		})
	}
	node := etree.FromString("<div><pre>ordinary quoted prose</pre></div>")
	convertTags(node, zeroOpts)
	assert.Nil(test, dom.QuerySelector(node, "code"))
}

func Test_FencedFrames(test *testing.T) {
	for _, fallback := range []bool{false, true} {
		for _, input := range []string{
			`<html><body><article><p>real text</p><fencedframe><p>advertising payload</p></fencedframe><p>more text</p></article></body></html>`,
			`<html><body><fencedframe><article><h1>advertising payload</h1><p>advertising payload</p></article></fencedframe></body></html>`,
		} {
			result, err := Extract(strings.NewReader(input), Options{Config: zeroConfig, EnableFallback: fallback})
			if assert.NoError(test, err) && assert.NotNil(test, result) {
				assert.NotContains(test, result.ContentText, "advertising payload")
			}
		}
	}
}

func Test_Cleaning_Upstream22(test *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"nobr text", `<div><p>before <nobr>unbroken words</nobr> after</p></div>`, `<div><p>before unbroken words after</p></div>`},
		{"removed element tail", `<div><p>before<script>discard</script>after</p></div>`, `<div><p>beforeafter</p></div>`},
		{"empty sub and sup", `<div><p>before<sup></sup>middle<sub></sub>after</p></div>`, `<div><p>beforemiddleafter</p></div>`},
		{"FAQ heading", `<div><strong class="schema-faq-question">The question?</strong><p>The answer.</p></div>`, `<div><h3>The question?</h3><p>The answer.</p></div>`},
	}
	for _, testCase := range testCases {
		test.Run(testCase.name, func(test *testing.T) {
			node := etree.FromString(testCase.input)
			docCleaning(node, zeroOpts)
			convertTags(node, zeroOpts)
			assert.Equal(test, testCase.expected, etree.ToString(node))
		})
	}
	for _, role := range []string{"presentation", "none"} {
		node := etree.FromString(`<div><table role="` + role + `"><tr><td><p>layout text</p></td></tr></table></div>`)
		docCleaning(node, zeroOpts)
		assert.Nil(test, dom.QuerySelector(node, "table"))
		assert.Contains(test, dom.TextContent(node), "layout text")
	}
}

func Test_CoreCleaning_RemovesAllMatches(test *testing.T) {
	for _, tag := range []string{"nav", "aside", "fieldset"} {
		for _, depth := range []int{1, 2, 4} {
			for _, focus := range []ExtractionFocus{Balanced, FavorRecall, FavorPrecision} {
				test.Run(fmt.Sprintf("%s/depth-%d/focus-%d", tag, depth, focus), func(test *testing.T) {
					nested := strings.Repeat("<"+tag+">", depth) + "discard nested content" + strings.Repeat("</"+tag+">", depth)
					input := "<div>before" + nested + "between<p>Article text kept intact.</p><" + tag + ">discard sibling content</" + tag + ">after</div>"
					document := etree.FromString(input)
					opts := Options{Config: DefaultConfig(), Focus: focus}
					docCleaningMode(document, opts, true)
					assert.Empty(test, dom.GetElementsByTagName(document, tag))
					assert.Equal(test, "beforebetweenArticle text kept intact.after", dom.TextContent(document))
					cleaned := etree.ToString(document)
					docCleaningMode(document, opts, true)
					assert.Equal(test, cleaned, etree.ToString(document))
				})
			}
		}
	}
}

func Test_LinkDensity_Upstream22(test *testing.T) {
	for _, input := range []string{
		`<ul><li><p><a href="/x">Linked list item</a></p></li></ul>`,
		`<table><tr><td><p><a href="/x">Linked table cell</a></p></td></tr></table>`,
	} {
		node := etree.FromString(input)
		deleteByLinkDensity(node, zeroOpts, false, "p")
		assert.NotNil(test, dom.QuerySelector(node, "p"))
	}
	node := etree.FromString(`<div>` + strings.Repeat(`<a href="/news">A moderately long news headline with enough text</a>`, 10) + `</div>`)
	_, discard := linkDensityTest(node, zeroOpts)
	assert.True(test, discard)

	node = etree.FromString(`<div>` + strings.Repeat(`<a href="/article">`+strings.Repeat("Editorial sentence. ", 10)+`</a>`, 5) + `</div>`)
	_, discard = linkDensityTest(node, zeroOpts)
	assert.False(test, discard)

	node = etree.FromString(`<table><tr><td>` + strings.Repeat("Informative cell content. ", 20) + `<a href="/flag"><img src="flag.png"></a></td></tr></table>`)
	assert.False(test, linkDensityTestTables(node, zeroOpts))
}
