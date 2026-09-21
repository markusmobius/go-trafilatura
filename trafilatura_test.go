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
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	nurl "net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/go-shiori/dom"
	"github.com/markusmobius/go-htmldate"
	"github.com/markusmobius/go-trafilatura/v2/internal/etree"
	"github.com/markusmobius/go-trafilatura/v2/internal/lru"
	"github.com/markusmobius/go-trafilatura/v2/internal/selector"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/html"
)

var (
	exampleURL, _ = nurl.ParseRequestURI("https://example.org")

	trafilaturaMockFiles = map[string]string{
		"http://exotic_tags": "exotic_tags.html",
	}

	zeroOpts = Options{
		EnableFallback: true,
		OriginalURL:    exampleURL,
		Config:         zeroConfig,
	}

	zeroConfig = &Config{
		MinOutputSize:    0,
		MinExtractedSize: 0,
	}

	defaultOpts = Options{
		Config: DefaultConfig(),
	}
)

func Test_Python220_CoreMatrix(test *testing.T) {
	type sample struct {
		HTML             string          `json:"html"`
		Focus            ExtractionFocus `json:"focus"`
		Flags            int             `json:"flags"`
		Variant          int             `json:"variant"`
		Accepted         bool            `json:"accepted"`
		Snapshot         string          `json:"snapshot"`
		Content          string          `json:"content"`
		Comments         string          `json:"comments"`
		CommentsSnapshot string          `json:"comments_snapshot"`
	}
	var reference struct {
		Core struct {
			Sequences  []*sample `json:"sequences"`
			Extraction []*sample `json:"extraction"`
			Edges      []*sample `json:"edge_sequences"`
		} `json:"native_core"`
		Forums []struct {
			HTML  string `json:"html"`
			Forum bool   `json:"forum"`
		} `json:"forums"`
	}
	data, err := os.ReadFile("test-files/python-2.2.0-reference.json")
	if err != nil {
		test.Fatal(err)
	}
	if err := json.Unmarshal(data, &reference); err != nil {
		test.Fatal(err)
	}
	assert.Len(test, reference.Core.Sequences, 4992)
	assert.Len(test, reference.Core.Extraction, 6177)
	assert.Len(test, reference.Core.Edges, 162)
	reference.Core.Sequences = append(reference.Core.Sequences, reference.Core.Edges...)
	assert.Len(test, reference.Forums, 30)
	for _, current := range reference.Forums {
		assert.Equal(test, current.Forum, forumThreadPage(docFromStr(current.HTML)), "%s", current.HTML)
	}
	options := func(current *sample) Options {
		address, _ := nurl.Parse("https://example.com/news/page")
		return Options{Config: DefaultConfig(), Focus: current.Focus, IncludeImages: current.Flags&1 != 0, IncludeLinks: current.Flags&2 != 0,
			Deduplicate: current.Flags&4 != 0, ExcludeComments: current.Flags&16 != 0, ExcludeTables: current.Flags&32 != 0, OriginalURL: address, HtmlDateMode: Disabled}
	}
	compact := func(value string) string { return strings.Join(strings.Fields(value), "") }
	nodeText := func(node *html.Node) string {
		if node == nil {
			return ""
		}
		return dom.TextContent(node)
	}
	duplicateParagraph := strings.Repeat("Duplicate long paragraph text. ", 4)
	duplicateInput := "<main><p>" + duplicateParagraph + "</p><p>" + duplicateParagraph + `</p><figure><img src="image.jpg" alt="photo"></figure><table><tr><td>data</td></tr></table></main>`
	retainedGoExpectation := func(current *sample) sample {
		expected := *current
		if current.HTML == duplicateInput && current.Focus != FavorPrecision {
			expected.Content = strings.TrimSpace(duplicateParagraph)
			if current.Variant == 4 {
				expected.Accepted = true
			}
		}
		if current.Flags&1 == 0 && strings.Contains(current.HTML, "<figure><figure>") {
			expected.Content = strings.TrimPrefix(expected.Content, "Later image caption.")
		}
		return expected
	}
	for index, current := range reference.Core.Sequences {
		if current == nil {
			continue
		}
		test.Run(fmt.Sprintf("sequence/%d", index), func(test *testing.T) {
			opts := options(current)
			expected := retainedGoExpectation(current)
			body, snapshot, comments, commentsSnapshot := extractionSequence(docFromStr(current.HTML), lru.NewCache(opts.Config.CacheSize), opts)
			assert.Equal(test, compact(expected.Content), compact(dom.TextContent(body)))
			if compact(current.Snapshot) != compact(current.Content) || expected.Content != current.Content {
				assert.Equal(test, compact(expected.Content), compact(snapshot))
			} else {
				assert.Equal(test, trim(current.Snapshot), trim(snapshot))
			}
			assert.Equal(test, compact(current.Comments), compact(nodeText(comments)))
			assert.Equal(test, trim(current.CommentsSnapshot), trim(commentsSnapshot))
		})
	}
	for index, current := range reference.Core.Extraction {
		if current == nil {
			continue
		}
		test.Run(fmt.Sprintf("extraction/%d", index), func(test *testing.T) {
			opts := options(current)
			switch current.Variant {
			case 3:
				opts.TargetLanguage = "en"
			case 4:
				opts.MaxTreeSize = 1
			case 5:
				opts.Config.MinOutputSize, opts.Config.MinOutputCommentSize = 50000, 50000
			case 6:
				opts.PruneSelector = "p.bar, aside"
			case 7:
				opts.PruneSelector = "p:not("
			}
			expected := retainedGoExpectation(current)
			result, err := ExtractDocument(docFromStr(current.HTML), opts)
			if !assert.Equal(test, expected.Accepted, err == nil, "%v", err) || err != nil {
				return
			}
			assert.Equal(test, compact(expected.Content), compact(dom.TextContent(result.ContentNode)))
			assert.Equal(test, compact(current.Comments), compact(nodeText(result.CommentsNode)))
		})
	}
}

func Test_Python220_TextFilter(test *testing.T) {
	var reference struct {
		Cases []struct {
			Text     string `json:"text"`
			Filtered bool   `json:"filtered"`
		} `json:"text_filters"`
	}
	data, err := os.ReadFile("test-files/python-2.2.0-reference.json")
	if err != nil {
		test.Fatal(err)
	}
	if err := json.Unmarshal(data, &reference); err != nil {
		test.Fatal(err)
	}
	assert.Len(test, reference.Cases, 349)
	for _, sample := range reference.Cases {
		element := etree.Element("p")
		etree.SetText(element, sample.Text)
		assert.Equal(test, sample.Filtered, textFilter(element), "Python text filter %q", sample.Text)
	}
}

func Test_Python220_ContentSnapshots(test *testing.T) {
	var reference struct {
		Cases []struct {
			HTML             string          `json:"html"`
			Focus            ExtractionFocus `json:"focus"`
			Snapshot         string          `json:"snapshot"`
			Length           int             `json:"length"`
			Cleaned          string          `json:"cleaned"`
			SequenceSnapshot string          `json:"sequence_snapshot"`
			SequenceCleaned  string          `json:"sequence_cleaned"`
		} `json:"content_snapshots"`
	}
	data, err := os.ReadFile("test-files/python-2.2.0-reference.json")
	if err != nil {
		test.Fatal(err)
	}
	if err := json.Unmarshal(data, &reference); err != nil {
		test.Fatal(err)
	}
	assert.Len(test, reference.Cases, 27)
	shortParagraph := strings.TrimSpace(strings.Repeat("A substantial repeated article paragraph. ", 2))
	mediumParagraph := strings.TrimSpace(strings.Repeat("A substantial repeated article paragraph. ", 4))
	shortBaseline := strings.Join([]string{shortParagraph, shortParagraph, shortParagraph, "data"}, "\n")
	retainedGoRecovery := map[int]string{
		6: shortBaseline, 7: shortBaseline,
		12: mediumParagraph, 13: mediumParagraph,
		15: mediumParagraph, 16: mediumParagraph,
	}
	for index, sample := range reference.Cases {
		test.Run(fmt.Sprint(index), func(test *testing.T) {
			opts := Options{Config: DefaultConfig(), Focus: sample.Focus, ExcludeComments: true}
			body, snapshot := extractContent(prepareTree(docFromStr(sample.HTML), opts), lru.NewCache(opts.Config.CacheSize), opts)
			assert.Equal(test, etree.ExtractionText(body), snapshot)
			assert.Equal(test, strings.Fields(sample.Cleaned), strings.Fields(snapshot))
			assert.LessOrEqual(test, len([]rune(snapshot)), sample.Length)
			assert.Equal(test, sample.Cleaned, trim(etree.IterText(body, " ")))
			expectedSequence := sample.SequenceCleaned
			if recovered, exists := retainedGoRecovery[index]; exists {
				expectedSequence = recovered
			}
			body, snapshot, _, _ = extractionSequence(docFromStr(sample.HTML), lru.NewCache(opts.Config.CacheSize), opts)
			assert.Equal(test, strings.Fields(expectedSequence), strings.Fields(snapshot))
			assert.Equal(test, strings.Fields(expectedSequence), strings.Fields(etree.IterText(body, " ")))
		})
	}
}

func Test_ExtractContent_PostCleanupText(test *testing.T) {
	paragraph := strings.TrimSpace(strings.Repeat("Retained article prose has enough detail to describe the subject. ", 3))
	for _, copies := range []int{1, 2, 4} {
		for _, focus := range []ExtractionFocus{Balanced, FavorRecall, FavorPrecision} {
			test.Run(fmt.Sprintf("copies-%d/focus-%d", copies, focus), func(test *testing.T) {
				opts := Options{Config: DefaultConfig(), Focus: focus, ExcludeComments: true}
				opts.Config.MinExtractedSize = 0
				input := "<html><body><article>" + strings.Repeat("<p>"+paragraph+"</p>", copies) + "</article></body></html>"
				body, text := extractContent(prepareTree(docFromStr(input), opts), lru.NewCache(opts.Config.CacheSize), opts)
				assert.Equal(test, paragraph, trim(etree.IterText(body, " ")))
				assert.Equal(test, etree.ExtractionText(body), text)
			})
		}
	}
}

func Test_RecoveryIgnoresRemovedDuplicates(test *testing.T) {
	caption := strings.TrimSpace(strings.Repeat("A descriptive caption identifies the subject and the setting. ", 3))
	article := strings.TrimSpace(strings.Repeat("The complete article explains the evidence, observations and conclusions in detail. ", 12))
	data, err := json.Marshal(map[string]string{"@context": "https://schema.org", "@type": "NewsArticle", "articleBody": article})
	if !assert.NoError(test, err) {
		return
	}
	for _, copies := range []int{1, 2, 4} {
		for _, focus := range []ExtractionFocus{Balanced, FavorRecall} {
			test.Run(fmt.Sprintf("copies-%d/focus-%d", copies, focus), func(test *testing.T) {
				opts := Options{Config: DefaultConfig(), Focus: focus, ExcludeComments: true}
				assert.Less(test, len([]rune(caption)), opts.Config.MinExtractedSize)
				input := `<html><head><script type="application/ld+json">` + string(data) + `</script></head><body><article>` + strings.Repeat("<p>"+caption+"</p>", copies) + "</article></body></html>"
				body, text, _, _ := extractionSequence(docFromStr(input), lru.NewCache(opts.Config.CacheSize), opts)
				assert.Equal(test, article, trim(etree.IterText(body, " ")))
				assert.Equal(test, article, trim(text))
			})
		}
	}
}

func Test_InputSafety(test *testing.T) {
	result, err := ExtractDocument(nil, Options{})
	assert.Error(test, err)
	assert.Nil(test, result)
	failure := errors.New("reader failure")
	_, err = Extract(iotest.ErrReader(failure), Options{})
	assert.ErrorIs(test, err, failure)
	for _, input := range [][]byte{{0x1f, 0x8b}, {0x1f, 0x8b, 0x00, 0x00}} {
		result, err := Extract(bytes.NewReader(input), Options{})
		assert.Error(test, err)
		assert.Nil(test, result)
	}
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	_, err = writer.Write([]byte("<html><body><p>Caf\u00e9.</p></body></html>"))
	assert.NoError(test, err)
	assert.NoError(test, writer.Close())
	result, err = Extract(iotest.OneByteReader(bytes.NewReader(compressed.Bytes())), Options{InputEncoding: "utf-8"})
	if assert.NoError(test, err) && assert.NotNil(test, result) {
		assert.Equal(test, "Caf\u00e9.", result.ContentText)
	}
	_, err = Extract(bytes.NewReader(compressed.Bytes()[:compressed.Len()-4]), Options{})
	assert.Error(test, err)
}

func Test_FallbackPreparation_PreservesInput(test *testing.T) {
	article := strings.Repeat(`<p>Article content with <span>inline formatting</span> and enough text to exercise the external extractor.</p>`, 12)
	for _, focus := range []ExtractionFocus{Balanced, FavorRecall, FavorPrecision} {
		for _, frame := range []string{"", `<fencedframe>Frame-only content that must not be extracted.</fencedframe>`} {
			test.Run(fmt.Sprintf("focus-%d/frame-%t", focus, frame != ""), func(test *testing.T) {
				doc := docFromStr(`<html><head><title>Example article</title></head><body><nav>Navigation menu</nav>` + frame + `<article><h1>Example article</h1>` + article + `</article></body></html>`)
				before := dom.OuterHTML(doc)
				opts := Options{Config: DefaultConfig(), EnableFallback: true, Focus: focus}
				body, text := compareExternalExtraction(doc, etree.Element("body"), opts)
				assert.NotNil(test, body)
				assert.Contains(test, text, "Article content with inline formatting")
				assert.NotContains(test, text, "Frame-only content")
				assert.Equal(test, before, dom.OuterHTML(doc))
			})
		}
	}
}

func Test_Python220_InputsAndOptions(test *testing.T) {
	test.Run("test_input", func(test *testing.T) {
		test.Run("nil", func(test *testing.T) {
			assert.NotPanics(test, func() {
				result, _ := Extract(nil, Options{EnableFallback: true})
				assert.Nil(test, result)
			})
		})
		for _, input := range []string{
			"<html><body>\u00c4\u00d6\u00dc</body></html>",
			"<html><body>\x2f\x2e\x9f</body></html>",
		} {
			doc, err := dom.Parse(strings.NewReader(input))
			assert.NoError(test, err)
			assert.NotNil(test, doc)
		}
		page := "<html><body><article>" + strings.Repeat("<p>Long enough article paragraph\x1d for baseline\uffff to trigger.</p>", 3) + "</article></body></html>"
		_, text := baseline(docFromStr(page))
		assert.NotEmpty(test, text)
		result, err := Extract(strings.NewReader(page), Options{})
		assert.NoError(test, err)
		assert.NotNil(test, result)
		result, err = Extract(strings.NewReader("<html><body><p>A\u0308ffin</p></body></html>"), zeroOpts)
		if assert.NoError(test, err) {
			assert.Equal(test, "\u00c4ffin", result.ContentText)
		}
		for _, sample := range []struct {
			tag      string
			accepted bool
		}{{"p", true}, {"unexpected", false}} {
			node := etree.Element(sample.tag)
			etree.SetText(node, "text")
			processed := handleTextElem(node, nil, nil, defaultOpts)
			assert.Equal(test, sample.accepted, processed != nil)
			if processed != nil {
				assert.Equal(test, "text", etree.Text(processed))
			}
		}
		result, err = Extract(strings.NewReader("<html><body><p>ABC</p></body></html>"), Options{EnableFallback: true})
		if assert.NoError(test, err) && assert.NotNil(test, result) {
			assert.Equal(test, "ABC", result.ContentText)
		}
		test.Run("gzip", func(test *testing.T) {
			input, err := os.Open("test-files/mock/webpage.html.gz")
			if !assert.NoError(test, err) {
				return
			}
			defer input.Close()
			result, err := Extract(input, Options{EnableFallback: true})
			if assert.NoError(test, err) && assert.NotNil(test, result) {
				assert.True(test, strings.Contains(result.ContentText, "Long story short,"), "unit_tests.py:224: gzip input must be decompressed before extraction")
			}
		})
	})
	test.Run("test_extraction_options", func(test *testing.T) {
		input := `<html><head><meta http-equiv="content-language" content="EN"/></head><body><div="article-body"><p>Text.<!-- comment --><?php echo "This is a PHP processing instruction"; ?></p></div></body></html>`
		highMinimum := DefaultConfig()
		highMinimum.MinExtractedSize, highMinimum.MinExtractedCommentSize = 10000, 10000
		highMinimum.MinOutputSize, highMinimum.MinOutputCommentSize = 10000, 10000
		for _, sample := range []struct {
			name     string
			opts     Options
			accepted bool
		}{
			{"configured_minimum", Options{Config: highMinimum, EnableFallback: true}, false},
			{"zero_minimum", zeroOpts, true},
			{"optional_metadata", Options{Config: zeroConfig, EnableFallback: true}, true},
			{"essential_metadata", Options{Config: zeroConfig, EnableFallback: true, HasEssentialMetadata: true}, false},
			{"wrong_language", Options{Config: zeroConfig, EnableFallback: true, TargetLanguage: "de"}, false},
			{"wrong_language_fast", Options{Config: zeroConfig, TargetLanguage: "de"}, false},
		} {
			test.Run(sample.name, func(test *testing.T) {
				result, _ := Extract(strings.NewReader(input), sample.opts)
				assert.Equal(test, sample.accepted, result != nil)
			})
		}
		large := "<html><head/><body>" + strings.Repeat("<p>ABC def ghi jkl.</p>", 1000) + "<p>Posted on 1st Dec 2019<.</p></body></html>"
		result, err := Extract(strings.NewReader(large), zeroOpts)
		if assert.NoError(test, err) {
			assert.False(test, result.Metadata.Date.IsZero())
		}
		result, err = Extract(strings.NewReader(large), Options{Config: highMinimum, EnableFallback: true, HtmlDateMode: Fast})
		if assert.NoError(test, err) && assert.NotNil(test, result) {
			assert.True(test, result.Metadata.Date.IsZero())
		}
		test.Run("with_metadata_false", func(test *testing.T) {
			test.Skip("Go always returns metadata; Python's with_metadata=False option has no equivalent")
		})
	})
	test.Run("test_extract_with_metadata", func(test *testing.T) {
		address, err := nurl.Parse("http://aa.bb/cc.html")
		if !assert.NoError(test, err) {
			return
		}
		for _, sample := range []struct{ input, title, date string }{
			{"<html>\n        <head></head>\n        <body>\n        <article>\n        <p>AAA, <p>BBB</p>, CCC.</p>\n        </article>\n        </body>\n        </html>\n    ", "", ""},
			{"<html>\n        <head><title>title</title></head>\n        <body>\n        <article>\n        <div>May 24, 2021</div>\n        <p>AAA, <p>BBB</p>, CCC.</p>\n        </article>\n        </body>\n        </html>\n    ", "title", "2021-05-24"},
		} {
			result, err := Extract(strings.NewReader(sample.input), Options{OriginalURL: address, HtmlDateMode: Extensive})
			if !assert.NoError(test, err) {
				continue
			}
			for _, text := range []string{"AAA", "BBB", "CCC"} {
				assert.Contains(test, result.ContentText, text)
			}
			assert.Equal(test, address.String(), result.Metadata.URL)
			assert.Equal(test, sample.title, result.Metadata.Title)
			if sample.date == "" {
				assert.True(test, result.Metadata.Date.IsZero())
			} else {
				assert.Equal(test, sample.date, result.Metadata.Date.Format("2006-01-02"))
			}
		}
		result, _ := Extract(strings.NewReader(`<html><head><meta http-equiv="content-language" content="es"></head><body><article><p>AAA, <p>BBB</p>, CCC.</p></article></body></html>`), Options{TargetLanguage: "en"})
		assert.Nil(test, result)
	})
	test.Run("test_large_doc_performance", func(test *testing.T) {
		input := "<html><body>" + strings.Repeat("<p>Sample text</p>", 10000) + "</body></html>"
		start := time.Now()
		_, _ = Extract(strings.NewReader(input), zeroOpts)
		assert.Less(test, time.Since(start), 5*time.Second)
	})
	test.Run("test_wrong_language_discarded", func(test *testing.T) {
		result, _ := Extract(strings.NewReader("<html><body>"+strings.Repeat("<p>Questo testo non \u00e8 affatto in lingua inglese.</p>", 20)+"</body></html>"), Options{TargetLanguage: "en", Config: zeroConfig, EnableFallback: true})
		assert.Nil(test, result)
	})
	test.Run("test_lang_detection", func(test *testing.T) {
		for _, sample := range []struct{ input, expected string }{{"<html><body><p>Texto en espa\u00f1ol</p></body></html>", "es"}, {"<html><body><p>Texte en fran\u00e7ais</p></body></html>", "fr"}} {
			result, err := Extract(strings.NewReader(sample.input), zeroOpts)
			if assert.NoError(test, err) {
				assert.Equal(test, sample.expected, languageClassifier(result.ContentText, ""))
			}
		}
	})
	test.Run("test_html_conversion", func(test *testing.T) {
		for _, title := range []string{"Title", "Title 1"} {
			input := "<html><body><article><h1>" + title + "</h1><p>Text.</p></article></body></html>"
			result, err := Extract(strings.NewReader(input), Options{Config: zeroConfig, EnableFallback: true})
			if assert.NoError(test, err) && assert.NotNil(test, result) {
				assert.Equal(test, "<body><h1>"+title+"</h1><p>Text.</p></body>", etree.ToString(result.ContentNode))
				if title == "Title 1" {
					assert.Equal(test, title, result.Metadata.Title)
				}
			}
		}
		input := `<html><body><article><p>Body text here.</p><img src="pic.jpg" alt="a"/></article></body></html>`
		result, err := Extract(strings.NewReader(input), Options{Config: zeroConfig, EnableFallback: true, IncludeImages: true})
		if assert.NoError(test, err) && assert.NotNil(test, result) {
			assert.Contains(test, etree.ToString(result.ContentNode), `<img src="pic.jpg" alt="a"/>`)
			assert.Nil(test, dom.QuerySelector(result.ContentNode, "graphic"))
		}
	})
}

func Test_Python220_ImagesAndLinks(test *testing.T) {
	extract := func(test *testing.T, input string, options Options) *ExtractResult {
		test.Helper()
		result, err := Extract(strings.NewReader(input), options)
		if !assert.NoError(test, err) || !assert.NotNil(test, result) {
			test.FailNow()
		}
		return result
	}
	test.Run("test_images", func(test *testing.T) {
		input, err := os.ReadFile("test-files/simple/http_sample.html")
		if !assert.NoError(test, err) {
			return
		}
		result := extract(test, string(input), Options{EnableFallback: true})
		assert.Nil(test, dom.QuerySelector(result.ContentNode, `img[src="test.jpg"]`), "unit_tests.py:886")
		result = extract(test, string(input), Options{IncludeImages: true})
		assert.NotNil(test, dom.QuerySelector(result.ContentNode, `img[src="test.jpg"][title="Example image"]`), "unit_tests.py:887")
		for _, sample := range []struct {
			line                   int
			input, address, source string
		}{
			{895, `<img data-src="test.jpg" alt="text" title="a title"/>`, "", "test.jpg"},
			{896, `<p><img data-src="test.jpg" alt="text" title="a title"/></p>`, "", "test.jpg"},
			{897, `<p><img other="test.jpg" alt="text" title="a title"/></p>`, "", ""},
			{898, `<div><p><img data-src="test.jpg" alt="text" title="a title"/></p></div>`, "", "test.jpg"},
			{899, `<div><p><img data-src-small="test.jpg" alt="text" title="a title"/></p></div>`, "", "test.jpg"},
			{900, `<div><p><img src="https://a.b/test.jpg" alt="text" title="a title"/></p></div>`, "", "https://a.b/test.jpg"},
			{906, `<div><p><img src="//a.b/test.jpg" alt="text" title="a title"/></p></div>`, "http://a.b/c/d.html", "http://a.b/test.jpg"},
			{910, `<div><p><img src="/a.b/test.jpg" alt="text" title="a title"/></p></div>`, "http://a.b/c/d.html", "http://a.b/a.b/test.jpg"},
			{914, `<div><p><img src="./a.b/test.jpg" alt="text" title="a title"/></p></div>`, "http://a.b/c/d.html", "http://a.b/c/a.b/test.jpg"},
			{918, `<div><p><img src="../a.b/test.jpg" alt="text" title="a title"/></p></div>`, "http://a.b/c/d.html", "http://a.b/a.b/test.jpg"},
		} {
			test.Run(fmt.Sprintf("line_%d", sample.line), func(test *testing.T) {
				options := Options{Config: zeroConfig, IncludeImages: true}
				if sample.address != "" {
					options.OriginalURL, err = nurl.Parse(sample.address)
					if !assert.NoError(test, err) {
						return
					}
				}
				result := extract(test, "<html><body><article>"+sample.input+"</article></body></html>", options)
				images := dom.GetElementsByTagName(result.ContentNode, "img")
				if sample.source == "" {
					assert.Empty(test, images)
					assert.Empty(test, result.ContentText)
					return
				}
				if assert.Len(test, images, 1) {
					assert.Equal(test, sample.source, dom.GetAttribute(images[0], "src"))
					assert.Equal(test, "text", dom.GetAttribute(images[0], "alt"))
					assert.Equal(test, "a title", dom.GetAttribute(images[0], "title"))
				}
			})
		}
	})
	test.Run("test_links", func(test *testing.T) {
		for _, sample := range []struct {
			line                   int
			input, address, target string
		}{
			{961, `<html><body><p><a href="testlink.html">Test link text.</a> This part of the text has to be long enough.</p></body></html>`, "", "testlink.html"},
			{965, `<html><body><p><a href="testlink.html">Test link text.</a> This part of the text has to be long enough.</p></body></html>`, "https://www.example.com/", "https://www.example.com/testlink.html"},
			{972, `<html><body><p><a>Test link text.</a> This part of the text has to be long enough.</p></body></html>`, "", ""},
		} {
			test.Run(fmt.Sprintf("line_%d", sample.line), func(test *testing.T) {
				options := Options{IncludeLinks: true, Config: zeroConfig}
				if sample.address != "" {
					address, err := nurl.Parse(sample.address)
					if !assert.NoError(test, err) {
						return
					}
					options.OriginalURL = address
				}
				result := extract(test, sample.input, options)
				links := dom.GetElementsByTagName(result.ContentNode, "a")
				if assert.Len(test, links, 1) {
					assert.Equal(test, sample.target, dom.GetAttribute(links[0], "href"))
					assert.Equal(test, "Test link text.", dom.TextContent(links[0]))
					assert.Equal(test, " This part of the text has to be long enough.", etree.Tail(links[0]))
				}
			})
		}
		input, err := os.ReadFile("test-files/simple/http_sample.html")
		if !assert.NoError(test, err) {
			return
		}
		result := extract(test, string(input), Options{IncludeLinks: true, Config: zeroConfig})
		link := dom.QuerySelector(result.ContentNode, `a[href="testlink.html"]`)
		if assert.NotNil(test, link, "unit_tests.py:983") {
			assert.Equal(test, "link", dom.TextContent(link))
		}
		result = extract(test, `<html><body><p>Test text under <a rel="license" href="">CC BY-SA license</a>.</p></body></html>`, Options{IncludeLinks: true, Config: zeroConfig})
		assert.Equal(test, "CC BY-SA license", result.Metadata.License, "unit_tests.py:989")
	})
}

func Test_Python220_Structures(test *testing.T) {
	extractBody := func(test *testing.T, input string, options Options) *html.Node {
		test.Helper()
		options.Config = zeroConfig
		result, err := Extract(strings.NewReader("<html><body><article><p>enough intro text here for extraction</p>"+input+"</article></body></html>"), options)
		if !assert.NoError(test, err) {
			test.FailNow()
		}
		return result.ContentNode
	}
	for _, sample := range []struct{ name, input, selector, expected string }{
		{"test_blockquote_inline_content/bold", "<blockquote><p>A <b>bold</b> word</p></blockquote>", "blockquote", "<blockquote><p>A <b>bold</b> word</p></blockquote>"},
		{"test_blockquote_inline_content/link", "<blockquote><p>see <a href='http://x.com'>link</a></p></blockquote>", "blockquote", `<blockquote><p>see <a href="http://x.com">link</a></p></blockquote>`},
		{"test_blockquote_inline_content/image", "<blockquote><p>text</p><img src='x.jpg' alt='img'/></blockquote>", "blockquote", `<blockquote><p>text</p><img src="x.jpg" alt="img"/></blockquote>`},
		{"test_list_item_block_child_single_bullet", "<ul><li><p>x <b>bold</b> y</p></li></ul>", "ul", "<ul><li><p>x</p><b>bold</b> y</li></ul>"},
		{"test_list_item_image_gets_bullet", "<ul><li><img src='/i.jpg' alt='a'></li><li>plain</li></ul>", "ul", `<ul><li><img src="/i.jpg" alt="a"/></li><li>plain</li></ul>`},
		{"test_ordered_list_numbering/three", "<ol><li>one</li><li>two</li><li>three</li></ol>", "ol", "<ol><li>one</li><li>two</li><li>three</li></ol>"},
		{"test_ordered_list_numbering/one", "<ol><li>only</li></ol>", "ol", "<ol><li>only</li></ol>"},
		{"test_ordered_list_numbering/unordered", "<ul><li>a</li><li>b</li></ul>", "ul", "<ul><li>a</li><li>b</li></ul>"},
		{"test_nested_list_indentation/unordered", "<ul><li>a<ul><li>b</li><li>c</li></ul></li><li>d</li></ul>", "ul", "<ul><li>a<ul><li>b</li><li>c</li></ul></li><li>d</li></ul>"},
		{"test_nested_list_indentation/ordered", "<ul><li>a<ol><li>b</li></ol></li></ul>", "ul", "<ul><li>a<ol><li>b</li></ol></li></ul>"},
		{"test_list_item_link_with_inline_formatting/bold", "<ul><li>see <a href='http://x.com'><b>bold link</b></a> here</li></ul>", "ul", `<ul><li>see <a href="http://x.com"><b>bold link</b></a> here</li></ul>`},
		{"test_list_item_link_with_inline_formatting/mixed", "<ul><li>see <a href='http://x.com'>link <b>bold</b></a> here</li></ul>", "ul", `<ul><li>see <a href="http://x.com">link <b>bold</b></a> here</li></ul>`},
		{"test_paragraph_link_with_inline_formatting", "<p>see <a href='http://x.com'><b>bold</b></a> more</p>", "p:last-child", `<p>see <a href="http://x.com"><b>bold</b></a> more</p>`},
		{"test_nested_inline_formatting/paragraph", "<p>text <b><i>nested</i></b> end</p>", "p:last-child", "<p>text <b><i>nested</i></b> end</p>"},
		{"test_nested_inline_formatting/prefix", "<p><b>prefix <i>italic</i></b></p>", "p:last-child", "<p><b>prefix <i>italic</i></b></p>"},
		{"test_nested_inline_formatting/list", "<ul><li>text <b><i>nested</i></b> end</li></ul>", "ul", "<ul><li>text <b><i>nested</i></b> end</li></ul>"},
		{"test_blockquote_bare_inline", "<blockquote><b>bold</b> text here</blockquote>", "blockquote", "<blockquote><b>bold</b> text here</blockquote>"},
		{"test_del_and_code_in_non_paragraph_contexts/top", "<del>gone</del>", "del", "<del>gone</del>"},
		{"test_del_and_code_in_non_paragraph_contexts/list", "<ul><li>text <del>struck</del> more</li></ul>", "ul", "<ul><li>text <del>struck</del> more</li></ul>"},
		{"test_del_and_code_in_non_paragraph_contexts/quote", "<blockquote>text <del>struck</del> more</blockquote>", "blockquote", "<blockquote>text <del>struck</del> more</blockquote>"},
		{"test_del_and_code_in_non_paragraph_contexts/code", "<ul><li>use <code>func()</code> here</li></ul>", "ul", "<ul><li>use <code>func()</code> here</li></ul>"},
		{"test_hi_del_nesting_with_direct_text", "<p>before <b>bold <del>struck</del></b></p>", "p:last-child", "<p>before <b>bold <del>struck</del></b></p>"},
		{"test_image_tail_not_duplicated", "<ul><li>a <img src='i.jpg' alt='A'/> b</li></ul>", "ul", `<ul><li>a <img src="i.jpg" alt="A"/> b</li></ul>`},
	} {
		test.Run(sample.name, func(test *testing.T) {
			body := extractBody(test, sample.input, Options{IncludeLinks: true, IncludeImages: true, EnableFallback: true})
			element := dom.QuerySelector(body, sample.selector)
			if assert.NotNil(test, element) {
				assert.Equal(test, etree.ToString(python220Element(test, sample.expected)), etree.ToString(element))
			}
		})
	}
	test.Run("test_list_item_attr_whitelist", func(test *testing.T) {
		body := extractBody(test, `<ul><li>x <img src="p.jpg" class="c" width="9" alt="a"/> <a href="http://x.io" class="q">lnk</a> y</li></ul>`, Options{IncludeImages: true, IncludeLinks: true, Focus: FavorRecall, EnableFallback: true})
		image := dom.QuerySelector(body, "img")
		if assert.NotNil(test, image) {
			assert.Equal(test, "p.jpg", dom.GetAttribute(image, "src"))
			assert.Equal(test, "a", dom.GetAttribute(image, "alt"))
		}
		assert.Empty(test, dom.QuerySelectorAll(body, "[class], [width]"))
		assert.NotNil(test, dom.QuerySelector(body, `a[href="http://x.io"]`))
	})
	test.Run("test_include_images_does_not_truncate", func(test *testing.T) {
		lead := strings.Repeat("This single lead paragraph is deliberately long enough to exceed the minimum extracted size. ", 4)
		input := "<html><body><article><img src='/lead.jpg' alt='lead'><p>" + lead + "</p></article><div id='content'>"
		for index := 1; index < 5; index++ {
			input += fmt.Sprintf("<p>Continuation paragraph %d that must also survive extraction in full here.</p>", index)
		}
		input += "</div></body></html>"
		result, err := Extract(strings.NewReader(input), Options{IncludeImages: true, EnableFallback: true})
		if !assert.NoError(test, err) {
			return
		}
		assert.NotNil(test, dom.QuerySelector(result.ContentNode, `img[src="/lead.jpg"]`))
		for index := 1; index < 5; index++ {
			assert.Contains(test, result.ContentText, fmt.Sprintf("Continuation paragraph %d", index))
		}
	})
	test.Run("test_table_cell_keeps_nested_formatting", func(test *testing.T) {
		for _, sample := range []struct{ input, selector, expected string }{{"<p><b>bold</b></p>", "b", "bold"}, {"<p><img src='/i.jpg' alt='a'></p>", "img", ""}, {"<p>pre <b>mid</b> post</p>", "b", "mid"}, {"<p>x <del>gone</del> y</p>", "del", "gone"}, {"<p>x <code>c</code> y</p>", "code", "c"}} {
			body := extractBody(test, "<table><tr><td>"+sample.input+"</td><td>x</td></tr></table>", Options{IncludeImages: true, EnableFallback: true})
			element := dom.QuerySelector(body, "td "+sample.selector)
			if assert.NotNil(test, element) {
				assert.Equal(test, sample.expected, dom.TextContent(element))
				if sample.selector == "img" {
					assert.Equal(test, "/i.jpg", dom.GetAttribute(element, "src"))
					assert.Equal(test, "a", dom.GetAttribute(element, "alt"))
				}
			}
		}
	})
	test.Run("test_table_image_in_cell", func(test *testing.T) {
		address := "http://aa.bb/c.jpg"
		for _, sample := range []struct {
			input string
			alts  []string
			text  string
		}{
			{`<td>a<img src="` + address + `" alt="img"/><span>a</span></td>`, []string{"img"}, "aa"},
			{`<td><a href="` + address + `"><img src="` + address + `" alt="img"/><span>a</span></a></td>`, []string{"img"}, "a"},
			{`<td><img src="` + address + `" alt="img"/><span>a</span></td>`, []string{"img"}, "a"},
			{`<td><img src="` + address + `" alt="img1"/><span>a</span><img src="` + address + `" alt="img2"/></td>`, []string{"img1", "img2"}, "a"},
		} {
			body := extractBody(test, "<table><tr><td>a</td><td>b</td><td>c</td></tr><tr>"+sample.input+"<td><p>b</p><p>c</p></td><td>d</td></tr></table>", Options{IncludeImages: true})
			rows := dom.QuerySelectorAll(body, "tr")
			if !assert.Len(test, rows, 2) {
				continue
			}
			cells := dom.Children(rows[1])
			if !assert.Len(test, cells, 3) {
				continue
			}
			assert.Equal(test, sample.text, noSpace(dom.TextContent(cells[0])))
			assert.Equal(test, "b c", trim(etree.IterText(cells[1], " ")))
			assert.Equal(test, "d", dom.TextContent(cells[2]))
			var alts []string
			for _, image := range dom.QuerySelectorAll(cells[0], "img") {
				assert.Equal(test, address, dom.GetAttribute(image, "src"))
				alts = append(alts, dom.GetAttribute(image, "alt"))
			}
			assert.Equal(test, sample.alts, alts)
		}
	})
	combo := `<p>Intro with a <a href="http://x.io/p">link</a> and <b>bold</b> word.</p><table><tr><td>h1</td><td>h2</td></tr><tr><td><a href="http://x.io/c"><b>bold link</b></a></td><td><img src="http://x.io/i.jpg" alt="pic"/></td></tr></table>`
	for _, disabled := range []string{"", "include_links", "include_images", "include_tables"} {
		name := "test_combined_links_formatting_images_tables"
		if disabled != "" {
			name = "test_combined_flags_toggle_off/" + disabled
		}
		test.Run(name, func(test *testing.T) {
			body := extractBody(test, combo, Options{IncludeImages: disabled != "include_images", IncludeLinks: disabled != "include_links", ExcludeTables: disabled == "include_tables", EnableFallback: true})
			assert.Contains(test, dom.TextContent(body), "Intro with a link and bold word.")
			assert.NotNil(test, dom.QuerySelector(body, "p b"))
			assert.Equal(test, disabled != "include_links", dom.QuerySelector(body, `a[href="http://x.io/p"]`) != nil)
			assert.Equal(test, disabled != "include_tables", dom.QuerySelector(body, "table") != nil)
			if disabled != "include_tables" {
				assert.NotNil(test, dom.QuerySelector(body, "td b"))
				assert.Equal(test, disabled != "include_links", dom.QuerySelector(body, `td a[href="http://x.io/c"]`) != nil)
				assert.Equal(test, disabled != "include_images", dom.QuerySelector(body, `td img[src="http://x.io/i.jpg"][alt="pic"]`) != nil)
			}
		})
	}
	test.Run("test_combined_flags_toggle_off/include_formatting", func(test *testing.T) {
		test.Skip("Go always preserves HTML formatting; Python's Markdown-formatting toggle has no equivalent")
	})
}

func Test_Python220_Filters(test *testing.T) {
	test.Run("test_filters/language_filter", func(test *testing.T) {
		assert.Equal(test, "de", languageClassifier("Hier ist ein Text auf Deutsch", ""))
		assert.NotEqual(test, "en", languageClassifier("Hier ist ein Text auf Deutsch", ""))
		assert.Equal(test, "de", languageClassifier("Hier ist ein Text.", "Die Kommentare sind aber etwas l\u00e4nger."))
		input := "<html><body><article><p>How many ages hence/Shall this our lofty scene be acted over,/In states unborn and accents yet unknown!</p></article></body></html>"
		for _, language := range []string{"de", "en"} {
			result, _ := Extract(strings.NewReader(input), Options{Config: zeroConfig, EnableFallback: true, TargetLanguage: language})
			assert.Equal(test, language == "en", result != nil, language)
		}
		paragraph := "<p>In sleep a king, but waking no such matter.</p>"
		for _, sample := range []struct {
			htmlLanguage, target string
			fallback, accepted   bool
		}{{"en-US", "en", false, true}, {"en-US", "de", false, false}, {"de-DE", "de", true, false}} {
			result, _ := Extract(strings.NewReader(`<html lang="`+sample.htmlLanguage+`"><body>`+strings.Repeat(paragraph, 50)+`</body></html>`), Options{TargetLanguage: sample.target, EnableFallback: sample.fallback})
			assert.Equal(test, sample.accepted, result != nil, sample)
		}
	})
	test.Run("test_filters/max_tree_size", func(test *testing.T) {
		for _, sample := range []struct {
			paragraph string
			count     int
			accepted  bool
		}{{"<p>abc</p>", 50, true}, {"<p>abc</p>", 501, false}, {`<p><hi rend="#i">abc</hi></p>`, 501, false}, {`<p><hi rend="#i">abc</hi></p>`, 499, true}} {
			doc := python220Element(test, "<html><body>"+strings.Repeat(sample.paragraph, sample.count)+"</body></html>")
			result, _ := ExtractDocument(doc, Options{MaxTreeSize: 500, EnableFallback: true})
			assert.Equal(test, sample.accepted, result != nil, sample)
		}
		assert.Zero(test, Options{}.MaxTreeSize)
	})
	test.Run("test_filters/check_html_lang", func(test *testing.T) {
		for _, sample := range []struct {
			input, language  string
			strict, expected bool
		}{
			{`<html><body></body></html>`, "en", false, true},
			{`<html lang="de_DE, en_US"><body></body></html>`, "de", false, true},
			{`<html lang="de_DE, en_US"><body></body></html>`, "en", false, true},
			{`<html lang="de_DE, en_US"><body></body></html>`, "de", true, true},
			{`<html lang="de_DE, en_US"><body></body></html>`, "en", true, true},
			{`<html><head><meta http-equiv="content-language" content="en"></head><body></body></html>`, "en", false, true},
			{`<html><head><meta http-equiv="content-language" content="en"></head><body></body></html>`, "de", false, false},
			{`<html><head><meta http-equiv="content-language" content="DE"></head><body></body></html>`, "de", false, true},
			{`<html lang="en-US"><head><meta property="og:locale" content="de_DE" /></head><body></body></html>`, "de", false, true},
			{`<html lang="en-US"><head><meta property="og:locale" content="de_DE" /></head><body></body></html>`, "en", false, false},
			{`<html lang="en"><body></body></html>`, "it", true, false},
			{`<html lang="en"><body></body></html>`, "it", false, true},
			{`<html lang="en-US"><head><meta property="og:locale" content="de_DE" /></head><body></body></html>`, "de", true, true},
		} {
			assert.Equal(test, sample.expected, checkHtmlLanguage(docFromStr(sample.input), Options{TargetLanguage: sample.language}, sample.strict), sample)
		}
	})
	test.Run("test_filters/url_blacklist", func(test *testing.T) {
		test.Skip("Go Options has no URL blacklist; this Python option has no Go API equivalent")
	})
	test.Run("test_filters/config_file", func(test *testing.T) {
		test.Skip("Python configuration-file loading is excluded; MaxTreeSize is tested through Go Options")
	})
	test.Run("test_prune_xpath", func(test *testing.T) {
		body := strings.Repeat("<p>abc</p>", 50)
		for _, sample := range []struct{ body, selector, expected string }{{body, "p", ""}, {"<h1>ABC</h1>" + body, "p", "ABC"}, {"<h1>ABC</h1>" + body, "p, h1", ""}, {"<h1>ABC</h1><h2>42</h2>" + body, "p, h1", "42"}} {
			input := "<html><body>" + sample.body + "</body></html>"
			result, err := Extract(strings.NewReader(input), Options{PruneSelector: sample.selector, EnableFallback: true, Config: zeroConfig})
			if assert.NoError(test, err) {
				assert.Equal(test, sample.expected, result.ContentText)
			}
			result, err = Extract(strings.NewReader(input), Options{EnableFallback: true, Config: zeroConfig})
			if assert.NoError(test, err) {
				assert.NotEmpty(test, result.ContentText)
			}
		}
		result, err := Extract(strings.NewReader("<html><body><p>abc</p></body></html><!-- comment -->"), Options{EnableFallback: true, Config: zeroConfig})
		if assert.NoError(test, err) {
			assert.Equal(test, "abc", result.ContentText)
		}
		test.Run("comment_xpath", func(test *testing.T) {
			test.Skip("CSS PruneSelector cannot select comment nodes; normal comment removal is checked above")
		})
	})
}

func Test_Python220_Deduplication(test *testing.T) {
	test.Run("test_lrucache", func(test *testing.T) {
		cache := lru.NewCache(2)
		first := python220Element(test, "<body><p>AAAA BBBB AAAA BBBB AAAA BBBB AAAA BBBB AAAA BBBB AAAA BBBB AAAA BBBB AAAA BBBB AAAA BBBB AAAA BBBB AAAA BBBB AAAA BBBB AAAA BBBB</p></body>")
		second := python220Element(test, "<body><p>CCCC DDDD CCCC DDDD CCCC DDDD CCCC DDDD CCCC DDDD CCCC DDDD CCCC DDDD CCCC DDDD CCCC DDDD CCCC DDDD CCCC DDDD</p></body>")
		third := python220Element(test, "<body><p>EEEE FFFF EEEE FFFF EEEE FFFF EEEE FFFF EEEE FFFF EEEE FFFF EEEE FFFF EEEE FFFF EEEE FFFF EEEE FFFF EEEE FFFF EEEE FFFF EEEE FFFF</p></body>")
		firstParagraph, secondParagraph, thirdParagraph := dom.Children(first)[0], dom.Children(second)[0], dom.Children(third)[0]
		for _, sample := range []struct {
			node      *html.Node
			duplicate bool
		}{{firstParagraph, false}, {firstParagraph, false}, {first, false}, {firstParagraph, true}, {second, false}, {secondParagraph, false}, {second, false}, {secondParagraph, true}, {third, false}, {third, false}, {third, false}, {secondParagraph, true}, {thirdParagraph, true}, {firstParagraph, false}} {
			assert.Equal(test, sample.duplicate, duplicateTest(sample.node, cache, defaultOpts))
		}
		cache.Clear()
		assert.False(test, duplicateTest(secondParagraph, cache, defaultOpts))
		_, found := cache.Get("tralala")
		assert.False(test, found)
	})
	test.Run("test_dedup/paragraph", func(test *testing.T) {
		cache := lru.NewCache(2)
		paragraph := python220Element(test, "<p>"+strings.Repeat("abc", 50)+"</p>")
		opts := defaultOpts
		opts.Deduplicate = true
		for index := range 4 {
			assert.Equal(test, index < 3, processNode(paragraph, cache, opts) != nil)
		}
	})
	test.Run("test_dedup/cross_document", func(test *testing.T) {
		test.Skip("Go creates an extraction-local cache; there is no Python-style process-global LRU_TEST")
	})
	test.Run("test_dedup_reset_caches", func(test *testing.T) { test.Skip("Go has no process-global extraction cache or reset_caches API") })
	for _, name := range []string{"test_hashes", "test_content_fingerprint", "test_simhash", "test_sample_tokens"} {
		test.Run(name, func(test *testing.T) {
			test.Skip("Standalone hashing, Simhash, and token APIs are excluded by the port scope")
		})
	}
}

func Test_Python220_Tables(test *testing.T) {
	tableCells := func(test *testing.T, input string, opts Options) [][]string {
		test.Helper()
		opts.Config = zeroConfig
		result, err := Extract(strings.NewReader("<html><body><article><p>enough intro text here for extraction</p>"+input+"</article></body></html>"), opts)
		if !assert.NoError(test, err) {
			return nil
		}
		table := dom.QuerySelector(result.ContentNode, "table")
		if !assert.NotNil(test, table) {
			return nil
		}
		var rows [][]string
		for _, row := range dom.QuerySelectorAll(table, "tr") {
			var cells []string
			for _, cell := range dom.Children(row) {
				cells = append(cells, trim(etree.IterText(cell, " ")))
			}
			rows = append(rows, cells)
		}
		return rows
	}
	for _, sample := range []struct {
		name, input string
		expected    [][]string
		recall      bool
	}{
		{"test_table_colspan_padding", "<table><tr><td colspan='2'>a</td><td>b</td></tr><tr><td>c</td><td>d</td><td>e</td></tr></table>", [][]string{{"a", "", "b"}, {"c", "d", "e"}}, false},
		{"test_table_rowspan_aligned", "<table><tr><td rowspan='2'>x</td><td>a</td></tr><tr><td>b</td></tr></table>", [][]string{{"x", "a"}, {"", "b"}}, false},
		{"test_table_rowspan_colspan_combined", "<table><tr><td rowspan='2' colspan='2'>big</td><td>c</td></tr><tr><td>x</td></tr></table>", [][]string{{"big", "", "c"}, {"", "", "x"}}, false},
		{"test_table_rowspan_decrement_on_padding", "<table><tr><td>a</td><td rowspan='2'>b</td><td>c</td></tr><tr><td>x</td></tr><tr><td>d</td><td>e</td><td>f</td></tr></table>", [][]string{{"a", "b", "c"}, {"x", "", ""}, {"d", "e", "f"}}, false},
		{"test_table_empty_cells_and_rows/leading-empty", "<table><tr><td></td><td>b</td></tr></table>", [][]string{{"", "b"}}, false},
		{"test_table_empty_cells_and_rows/trailing-empty", "<table><tr><td>a</td><td></td></tr></table>", [][]string{{"a", ""}}, false},
		{"test_table_empty_cells_and_rows/all-empty-row-dropped", "<table><tr><td>a</td><td>b</td></tr><tr><td></td><td></td></tr></table>", [][]string{{"a", "b"}}, false},
		{"test_table_empty_cells_and_rows/empty-row-middle", "<table><tr><td>a</td><td>c</td></tr><tr><td></td><td></td></tr><tr><td>d</td><td>e</td></tr></table>", [][]string{{"a", "c"}, {"d", "e"}}, false},
		{"test_table_empty_cells_and_rows/empty-tr", "<table><tr><td>a</td><td>c</td></tr><tr></tr><tr><td>d</td><td>e</td></tr></table>", [][]string{{"a", "c"}, {"d", "e"}}, false},
		{"test_table_cell_block_elements_flattened/heading", "<table><tr><td><h2>Title</h2></td><td>b</td></tr></table>", [][]string{{"Title", "b"}}, false},
		{"test_table_cell_block_elements_flattened/paragraph", "<table><tr><td><p>para</p></td><td>b</td></tr></table>", [][]string{{"para", "b"}}, false},
		{"test_table_cell_block_elements_flattened/heading-plus-tail", "<table><tr><td><h2>Title</h2>txt</td><td>b</td></tr></table>", [][]string{{"Title txt", "b"}}, false},
		{"test_table_cell_block_elements_flattened/list-recall", "<table><tr><td><ul><li>i1</li><li>i2</li></ul></td><td>b</td></tr></table>", [][]string{{"i1 i2", "b"}}, true},
	} {
		test.Run(sample.name, func(test *testing.T) {
			opts := Options{EnableFallback: true}
			if sample.recall {
				opts.Focus = FavorRecall
			}
			assert.Equal(test, sample.expected, tableCells(test, sample.input, opts))
		})
	}
	test.Run("test_table_colspan_content", func(test *testing.T) {
		for _, count := range []int{1, 2} {
			input := "<table><tr><td>a</td><td>b</td><td>c</td></tr>" + strings.Repeat("<tr><td>a</td><td colspan='2'><p>b</p><p>c</p></td></tr>", count) + "</table>"
			rows := tableCells(test, input, Options{})
			if assert.Len(test, rows, count+1) {
				assert.Equal(test, []string{"a", "b", "c"}, rows[0])
				for _, row := range rows[1:] {
					assert.Equal(test, []string{"a", "b c", ""}, row)
				}
			}
		}
	})
	test.Run("test_table_bad_span_attr_treated_as_colspan1", func(test *testing.T) {
		for _, attribute := range []string{`span="2"`, `span="2.1"`, `span="-1"`, `span="abc"`} {
			rows := tableCells(test, "<table><tr><td "+attribute+">a</td><td>b</td></tr><tr><td>c</td><td>d</td><td>e</td></tr></table>", Options{})
			if assert.NotEmpty(test, rows) {
				assert.Equal(test, []string{"a", "b", ""}, rows[0])
			}
		}
	})
	test.Run("test_colspan_zero_trust", func(test *testing.T) {
		for _, sample := range []struct {
			span     string
			expected int
		}{{"1", 1}, {"3", 3}, {"12", 12}, {"\u00b2", 1}, {"1\u00b2", 1}, {"2x", 1}, {"", 1}, {"-2", 1}} {
			assert.Equal(test, sample.expected, tableSpan(python220Element(test, `<td colspan="`+sample.span+`">x</td>`), "colspan"), sample.span)
		}
	})
	test.Run("test_table_huge_or_bad_colspan_no_crash", func(test *testing.T) {
		for _, first := range []string{`<td colspan="9007199254740991">a</td>`, `<th colspan="9007199254740991">a</th>`, `<td colspan="2x">a</td>`} {
			assert.NotNil(test, tableCells(test, "<table><tr>"+first+"<td>b</td></tr><tr><td>c</td><td>d</td><td>e</td></tr></table>", Options{}))
		}
	})
	for _, sample := range []struct {
		name, input   string
		with, without []string
	}{
		{"test_table_nested_in_cell", "<table><tr><td>A</td></tr><tr><td><table><tr><td>inner</td></tr></table></td></tr><tr><td>AFTER</td></tr></table>", []string{"A", "AFTER"}, []string{"inner"}},
		{"test_table_nested_tail_preserved", "<table><tr><td>before<table><tr><td>inner</td></tr></table>after-tail</td></tr></table>", []string{"before", "after-tail"}, []string{"inner"}},
		{"test_table_nested_tail_with_prior_child", "<table><tr><td><del>struck</del><table><tr><td>inner</td></tr></table>after-tail</td></tr></table>", []string{"after-tail"}, []string{"inner"}},
		{"test_table_comment_in_row", "<table><tr><!-- ignored --><td>visible</td></tr></table>", []string{"visible"}, nil},
	} {
		test.Run(sample.name, func(test *testing.T) {
			table := handleTable(python220Element(test, sample.input), maps.Clone(tagCatalog), nil, defaultOpts)
			if !assert.NotNil(test, table) {
				return
			}
			text := dom.TextContent(table)
			for _, expected := range sample.with {
				assert.Contains(test, text, expected)
			}
			for _, unwanted := range sample.without {
				assert.NotContains(test, text, unwanted)
			}
		})
	}
	test.Run("test_table_orphan_cells_no_tr", func(test *testing.T) {
		table := handleTable(python220Element(test, "<table><td>a</td><td>b</td></table>"), maps.Clone(tagCatalog), nil, defaultOpts)
		var texts []string
		for _, cell := range dom.QuerySelectorAll(table, "td, th") {
			texts = append(texts, etree.Text(cell))
		}
		assert.Equal(test, []string{"a", "b"}, texts)
	})
	test.Run("test_table_stray_cell_descendant", func(test *testing.T) {
		table := handleTable(python220Element(test, "<table><tr><td><div><td>inner</td></div></td></tr></table>"), maps.Clone(tagCatalog), nil, defaultOpts)
		var texts []string
		for _, cell := range dom.QuerySelectorAll(table, "td, th") {
			texts = append(texts, etree.Text(cell))
		}
		assert.Contains(test, texts, "inner")
	})
	test.Run("test_table_caption", func(test *testing.T) {
		rows := tableCells(test, "<table><caption>My Caption</caption><tr><td>a</td><td>b</td></tr></table>", Options{EnableFallback: true})
		if assert.GreaterOrEqual(test, len(rows), 2) {
			assert.Contains(test, rows[0], "My Caption")
			assert.Equal(test, []string{"a", "b"}, rows[1])
		}
		table := handleTable(python220Element(test, "<table><caption>  </caption><tr><td>x</td></tr></table>"), maps.Clone(tagCatalog), nil, defaultOpts)
		if assert.NotNil(test, table) {
			cell := dom.QuerySelector(table, "td, th")
			if assert.NotNil(test, cell) {
				assert.Equal(test, "x", etree.Text(cell))
			}
			assert.Empty(test, dom.QuerySelectorAll(table, "th"))
		}
	})
	test.Run("test_table_nested_in_cell_pipeline", func(test *testing.T) {
		outer := strings.Repeat("This is the outer row with plenty of text to survive readability. ", 2)
		inner := strings.Repeat("Inner nested table cell with sufficient content for extraction. ", 2)
		input := "<html><body><article><p>enough intro text here for extraction</p><table><tr><td>" + outer + "</td></tr><tr><td><table><tr><td>" + inner + "</td></tr></table></td></tr></table></article></body></html>"
		result, err := Extract(strings.NewReader(input), Options{Config: zeroConfig, EnableFallback: true})
		if assert.NoError(test, err) {
			assert.Contains(test, result.ContentText, strings.TrimSpace(outer)[:20])
			assert.Contains(test, result.ContentText, strings.TrimSpace(inner)[:20])
			assert.Equal(test, 1, strings.Count(result.ContentText, strings.TrimSpace(inner)))
		}
	})
}

func Test_Python220_TableAndListLegacy(test *testing.T) {
	extract := func(test *testing.T, input string, options Options) *ExtractResult {
		test.Helper()
		options.Config = zeroConfig
		result, err := Extract(strings.NewReader(input), options)
		if !assert.NoError(test, err) || !assert.NotNil(test, result) {
			test.FailNow()
		}
		return result
	}
	test.Run("test_table_processing/comments", func(test *testing.T) {
		node := python220Element(test, "<table><!-- c1 --><tr><td>cell text<!-- c2 --></td></tr></table>")
		processed := handleTable(node, maps.Clone(tagCatalog), nil, defaultOpts)
		if assert.NotNil(test, processed) {
			assert.Contains(test, dom.TextContent(processed), "cell text")
		}
	})
	test.Run("test_table_processing/multi_row_links", func(test *testing.T) {
		input := `<html><body><article>
        <p>Enough intro text to satisfy trafilatura's minimum extraction length requirements for this test.</p>
        <table>
            <tr><th>Key</th><th>Value</th></tr>
            <tr><td><a href="/k1">Coord</a>:</td><td><a href="/v1">48 N</a></td></tr>
            <tr><td><a href="/k2">State</a>:</td><td><a href="/v2">BW</a></td></tr>
            <tr><td><a href="/k3">Region</a>:</td><td><a href="/v3">Stuttgart</a></td></tr>
        </table>
    </article></body></html>`
		result := extract(test, input, Options{IncludeLinks: true, EnableFallback: true})
		for _, sample := range []struct{ label, href, value string }{{"Coord", "/k1", "48 N"}, {"State", "/k2", "BW"}, {"Region", "/k3", "Stuttgart"}} {
			link := dom.QuerySelector(result.ContentNode, `a[href="`+sample.href+`"]`)
			if assert.NotNil(test, link) {
				assert.Equal(test, sample.label, dom.TextContent(link))
			}
			assert.Contains(test, result.ContentText, sample.value)
		}
	})
	test.Run("test_table_processing/headers_and_cells", func(test *testing.T) {
		for _, sample := range []struct {
			input   string
			headers int
			cells   []string
		}{
			{"<table><tr><th>head 1</th><th>head 2</th></tr><tr><td>1</td><td>2</td></tr></table>", 2, []string{"head 1", "head 2", "1", "2"}},
			{"<table><tr><th>a</th><th>b</th><th>c</th></tr></table>", 3, []string{"a", "b", "c"}},
			{"<table><tr><td>cell<br>1</td><td>cell<p>2</p></td></tr></table>", 0, []string{"cell 1", "cell 2"}},
			{`<table><tr><td><a href="link.html">a</a></td></tr></table>`, 0, []string{"a"}},
		} {
			result := extract(test, "<html><body><article>"+sample.input+"</article></body></html>", Options{})
			assert.Len(test, dom.GetElementsByTagName(result.ContentNode, "th"), sample.headers)
			var values []string
			for _, cell := range dom.QuerySelectorAll(result.ContentNode, "td, th") {
				values = append(values, trim(etree.IterText(cell, " ")))
			}
			assert.Equal(test, sample.cells, values)
		}
		for _, label := range []string{strings.Repeat("abc", 100), strings.Repeat(" ", 100)} {
			input := `<html><body><article><table><tr><td><a href="link.html">` + label + `</a></td></tr></table></article></body></html>`
			assert.Empty(test, extract(test, input, Options{}).ContentText)
		}
	})
	test.Run("test_table_cell_list_no_row_break", func(test *testing.T) {
		input := "<html><body><article><p>enough intro text here for extraction</p><table><tr><td><ul><li>i1</li><li>i2</li></ul></td><td>b</td></tr></table></article></body></html>"
		result := extract(test, input, Options{EnableFallback: true})
		rows := dom.GetElementsByTagName(result.ContentNode, "tr")
		if assert.Len(test, rows, 1) {
			cells := dom.Children(rows[0])
			if assert.Len(test, cells, 2) {
				assert.Equal(test, "b", dom.TextContent(cells[1]))
			}
		}
	})
	test.Run("test_list_processing/basic_order", func(test *testing.T) {
		input := "<html><body><article><p>P 1</p><ul><li>Item 1</li><li>Item 2</li></ul><p>P 2</p></article></body></html>"
		result := extract(test, input, Options{})
		assert.Equal(test, "<body><p>P 1</p><ul><li>Item 1</li><li>Item 2</li></ul><p>P 2</p></body>", etree.ToString(result.ContentNode))
	})
	test.Run("test_list_processing/link_only_items", func(test *testing.T) {
		input := `<html><body><article>
<p>If your eye is twitching and you do not have other symptoms, here are some common everyday causes worth knowing about.</p>
<ul>
<li><p><a href="https://example.org/stress">Stress</a></p></li>
<li><p>Fatigue</p></li>
<li><p><a href="https://example.org/strain">Eye strain</a></p></li>
</ul>
</article></body></html>`
		result := extract(test, input, Options{IncludeLinks: true})
		for _, sample := range []struct{ href, label string }{{"https://example.org/stress", "Stress"}, {"https://example.org/strain", "Eye strain"}} {
			link := dom.QuerySelector(result.ContentNode, `a[href="`+sample.href+`"]`)
			if assert.NotNil(test, link) {
				assert.Equal(test, sample.label, dom.TextContent(link))
			}
		}
		assert.Contains(test, result.ContentText, "Fatigue")
	})
	test.Run("test_recover_wild_text_dedup_scan_cap/default", func(test *testing.T) {
		container := "The quick brown fox jumps over the lazy dog while the sun was setting slowly over the meadow today."
		substring := "quick brown fox jumps over the lazy dog while the sun was setting slowly"
		filler := strings.Repeat("Zebra quokka platypus wallaby echidna kookaburra numbat bilby quoll dingo marsupial. ", 4)
		input := "<html><body><div>" + filler + "</div><div>" + container + "</div><div>" + substring + "</div></body></html>"
		body := etree.Element("body")
		recoverWildText(docFromStr(input), body, maps.Clone(tagCatalog), nil, Options{Config: DefaultConfig(), Focus: FavorRecall})
		var values []string
		for _, child := range dom.Children(body) {
			values = append(values, trim(dom.TextContent(child)))
		}
		assert.NotContains(test, values, substring)
	})
	test.Run("test_recover_wild_text_dedup_scan_cap/monkeypatch", func(test *testing.T) {
		test.Skip("Go's dedupeScanCap is constant; the Python-only monkeypatch branch cannot be invoked with the same input")
	})
}

func Test_Python220_Recovery(test *testing.T) {
	extractText := func(test *testing.T, input string, opts Options) string {
		test.Helper()
		result, err := Extract(strings.NewReader(input), opts)
		if !assert.NoError(test, err) {
			return ""
		}
		return result.ContentText + "\n" + result.CommentsText
	}
	test.Run("test_no_duplicate_content", func(test *testing.T) {
		first := "<!doctype html><body><main><article><div><br>Line that has to have at least 125 characters for the bug to appear so here is some filler text text text text text text text</div></article></main></body></html>"
		assert.Equal(test, 1, strings.Count(extractText(test, first, Options{EnableFallback: true}), "Line that has to have"))
		second := "<html><body><div id='content'><p>Authoritative taxonomy of but let us leave it as it is 1 2 3</p></div><p>some text long enough not to skip and printed twice on this line some text long enough not to skip and printed twice on this line</p></body></html>"
		assert.Equal(test, 1, strings.Count(extractText(test, second, Options{EnableFallback: true}), "Authoritative taxonomy"))
		third := "<html><body><nav>menu chrome</nav><article><h1>The Example Chronicle</h1><p>First synthetic paragraph of adequate length for extraction to engage properly.</p><p>Second synthetic paragraph, also long enough to matter for the extractor.</p></article><footer>footer chrome</footer></body></html>"
		for _, input := range []string{third, strings.ReplaceAll(third, "article>", "main>")} {
			text := extractText(test, input, Options{EnableFallback: true})
			assert.Equal(test, 1, strings.Count(text, "First synthetic paragraph"))
			assert.Equal(test, 1, strings.Count(text, "Second synthetic paragraph"))
		}
	})
	test.Run("test_no_duplicate_content_list_item", func(test *testing.T) {
		paragraph := "This is a moderately long description paragraph exceeding the fifty character dedup threshold here."
		input := "<html><body><article><dl><dt>Term</dt><dd><p>" + paragraph + "</p></dd></dl></article></body></html>"
		assert.Equal(test, 1, strings.Count(extractText(test, input, Options{}), paragraph))
	})
	test.Run("test_no_duplicate_content_nonadjacent", func(test *testing.T) {
		duplicate := strings.Repeat("X", 30) + " short duplicate description text for the list item here right now please."
		wild := strings.Repeat("Y", 30) + " this is genuinely separate wild text living outside the article container elsewhere in the page body content over here, quite far removed from it."
		input := "<html><body><p>" + wild + "</p><article><dl><dt>Term</dt><dd><p>" + duplicate + "</p></dd></dl></article></body></html>"
		text := extractText(test, input, Options{})
		assert.Equal(test, 1, strings.Count(text, duplicate))
		assert.Equal(test, 1, strings.Count(text, wild))
		assert.Contains(test, text, "Term")
	})
	test.Run("test_recover_wild_text_inline_formatting_dedup", func(test *testing.T) {
		paragraph := "This paragraph has Hyper<b>link</b>ed formatting inside and needs to be comfortably longer than the fifty character dedup gate to be caught by the substring check."
		input := "<html><body><article><dl><dt>Term one</dt><dd><p>" + paragraph + "</p></dd></dl></article></body></html>"
		assert.Equal(test, 1, strings.Count(extractText(test, input, Options{}), "formatting inside"))
	})
	test.Run("test_recall_escalation", func(test *testing.T) {
		var input strings.Builder
		input.WriteString("<html><body>")
		for index := range 3 {
			fmt.Fprintf(&input, "<p>Wild paragraph number %d directly under body, with enough words to pass the paragraph checks in place.</p>", index)
		}
		for index := range 30 {
			fmt.Fprintf(&input, "<div>Main content block %d living in a bare div element with plenty of meaningful words to matter here today.</div>", index)
		}
		input.WriteString("</body></html>")
		text := extractText(test, input.String(), Options{})
		assert.Equal(test, 3, strings.Count(text, "Wild paragraph"))
		assert.Equal(test, 30, strings.Count(text, "Main content block"))
	})
	intro := "<p>" + strings.Repeat("Introductory paragraph with regular prose content that is moderately long and clearly meaningful, providing enough text to exceed the minimum extraction size threshold used by the extractor. ", 2) + "</p>"
	forumJSON := `<script type="application/ld+json">{"@context":"https://schema.org","@type":"DiscussionForumPosting","headline":"Test thread"}</script>`
	thread := func(jsonld, tag, introduction string, wrap bool) string {
		var replies strings.Builder
		for index := range 8 {
			fmt.Fprintf(&replies, "<%s class='comment-body'>Reply number %d contains substantial discussion content with plenty of genuine words and enough length to be recognized as real text by a paragraph density classifier, not boilerplate at all today, said the commenter.</%s>", tag, index, tag)
		}
		wrapper := "div"
		if wrap {
			wrapper = "article"
		}
		return fmt.Sprintf("<html><head>%s</head><body><%s>%s</%s><div id='comments' class='comments-area'>%s</div></body></html>", jsonld, wrapper, introduction, wrapper, replies.String())
	}
	test.Run("test_recall_escalation_justext_comment_scoping", func(test *testing.T) {
		for _, sample := range []struct {
			name, jsonld string
			fast         bool
			replies      int
		}{{"blog_excludes_comments", "", false, 0}, {"forum_keeps_posts", forumJSON, false, 8}, {"blog_excludes_comments_fast", "", true, 0}} {
			test.Run(sample.name, func(test *testing.T) {
				text := extractText(test, thread(sample.jsonld, "div", intro, true), Options{ExcludeComments: true, EnableFallback: !sample.fast})
				assert.Contains(test, text, "Introductory paragraph")
				assert.Equal(test, sample.replies, strings.Count(text, "Reply number"))
			})
		}
	})
	test.Run("test_recall_escalation_no_comment_doubling", func(test *testing.T) {
		for _, tag := range []string{"p", "div"} {
			test.Run(tag, func(test *testing.T) {
				text := extractText(test, thread(forumJSON, tag, intro, true), Options{EnableFallback: true})
				assert.Contains(test, text, "Introductory paragraph")
				assert.Equal(test, 8, strings.Count(text, "Reply number"))
			})
		}
	})
	test.Run("test_dfp_long_opening_post_keeps_replies", func(test *testing.T) {
		var longIntro strings.Builder
		for index := range 16 {
			fmt.Fprintf(&longIntro, "<p>Opening post paragraph %d lays out the question in full detail, with context, history and several worked examples that make the thread starter alone longer than the escalation length gate, so no rescue pass can be relied upon for the replies.</p>", index)
		}
		text := extractText(test, thread(forumJSON, "p", longIntro.String(), true), Options{EnableFallback: true})
		assert.Contains(test, text, "Opening post paragraph")
		assert.Equal(test, 8, strings.Count(text, "Reply number"))
	})
	test.Run("test_dfp_precision_keeps_posts", func(test *testing.T) {
		text := extractText(test, thread(forumJSON, "p", intro, true), Options{EnableFallback: true, Focus: FavorPrecision})
		assert.Equal(test, 8, strings.Count(text, "Reply number"))
	})
	test.Run("test_recall_escalation_blog_comment_leak", func(test *testing.T) {
		for _, fast := range []bool{false, true} {
			text := extractText(test, thread("", "div", intro, false), Options{EnableFallback: !fast})
			assert.Contains(test, text, "Introductory paragraph")
			assert.Equal(test, 0, strings.Count(text, "Reply number"))
		}
	})
	test.Run("test_main_pass_excludes_comments_when_disabled", func(test *testing.T) {
		var replies strings.Builder
		for index := range 8 {
			fmt.Fprintf(&replies, "<div>Reader comment number %d that must never appear when comments are excluded, long enough to matter.</div>", index)
		}
		input := "<html><body><article><p>Short intro under the escalation and rescue thresholds here.</p></article><div id='comments' class='comments-area'>" + replies.String() + "</div></body></html>"
		for _, fast := range []bool{false, true} {
			text := extractText(test, input, Options{EnableFallback: !fast, ExcludeComments: true})
			assert.Contains(test, text, "Short intro")
			assert.Equal(test, 0, strings.Count(text, "Reader comment number"))
		}
	})
	test.Run("test_main_pass_excludes_details_wrapped_comments", func(test *testing.T) {
		body := "<article>" + strings.Repeat("<p>Real article paragraph with enough content to be extracted normally here.</p>", 3) + "</article>"
		var comments strings.Builder
		comments.WriteString("<details id='comments'><summary>Comments</summary>")
		for index := range 6 {
			fmt.Fprintf(&comments, "<p>Reader comment number %d that must never leak into the body text.</p>", index)
		}
		comments.WriteString("</details>")
		for _, fast := range []bool{false, true} {
			text := extractText(test, "<html><body>"+body+comments.String()+"</body></html>", Options{EnableFallback: !fast, ExcludeComments: true})
			assert.Contains(test, text, "Real article paragraph")
			assert.Equal(test, 0, strings.Count(text, "Reader comment number"))
		}
		faq := "<details class='faq'><summary>More</summary><p>Kept expandable content paragraph that is genuine.</p></details>"
		assert.Contains(test, extractText(test, "<html><body>"+body+faq+"</body></html>", Options{EnableFallback: true}), "Kept expandable content")
	})
}

func Test_InputEncoding(test *testing.T) {
	cases := []struct {
		name     string
		encoding string
		text     string
		want     string
	}{
		{"utf8", "utf-8", "Cafe\u0301 co\u00adoperate", "Caf\u00e9 cooperate"},
		{"utf8 alias", " \tUtF8 ", "Cafe\u0301 co\u00adoperate", "Caf\u00e9 cooperate"},
		{"windows1252", "windows-1252", "Caf\xe9 \x93co\xadoperate\x94", "Caf\u00e9 \u201ccooperate\u201d"},
		{"html latin1 alias", "iso-8859-1", "Caf\xe9 \x93co\xadoperate\x94", "Caf\u00e9 \u201ccooperate\u201d"},
		{"shift jis", "shift_jis", "\x93\xfa\x96\x7b\x8c\xea", "\u65e5\u672c\u8a9e"},
	}
	for _, item := range cases {
		test.Run(item.name, func(test *testing.T) {
			input := `<html><head><meta charset="utf-8"><title>` + item.text + `</title></head><body><article><p>` + item.text + `</p></article></body></html>`
			result, err := Extract(iotest.OneByteReader(strings.NewReader(input)), Options{
				InputEncoding: item.encoding,
				Config:        zeroConfig,
				HtmlDateMode:  Disabled,
			})
			if !assert.NoError(test, err) {
				return
			}
			assert.Equal(test, item.want, result.ContentText)
			assert.Equal(test, item.want, result.Metadata.Title)
		})
	}

	test.Run("default and parsed document", func(test *testing.T) {
		input := `<html><head><title>Cafe` + "\u0301" + ` article</title></head><body><article><p>` + strings.Repeat("Cafe\u0301 co\u00adoperate with the original parser. ", 20) + `</p></article><div id="comments"><p>A reader comment.</p></div></body></html>`
		options := Options{Config: zeroConfig, HtmlDateMode: Disabled}
		doc, err := dom.Parse(strings.NewReader(input))
		if !assert.NoError(test, err) {
			return
		}
		legacy, err := ExtractDocument(doc, options)
		if !assert.NoError(test, err) {
			return
		}
		for _, label := range []string{"", "utf-8"} {
			options.InputEncoding = label
			result, err := Extract(strings.NewReader(input), options)
			if !assert.NoError(test, err) {
				continue
			}
			assert.Equal(test, legacy.ContentText, result.ContentText)
			assert.Equal(test, legacy.CommentsText, result.CommentsText)
			assert.Equal(test, legacy.Metadata, result.Metadata)
			assert.Equal(test, dom.OuterHTML(legacy.ContentNode), dom.OuterHTML(result.ContentNode))
		}
		options.InputEncoding = "not-a-charset"
		result, err := ExtractDocument(doc, options)
		if assert.NoError(test, err) {
			assert.Equal(test, legacy.ContentText, result.ContentText)
			assert.Equal(test, legacy.Metadata, result.Metadata)
		}
	})

	test.Run("reader errors", func(test *testing.T) {
		readErr := errors.New("input read failed")
		for _, label := range []string{"", "utf-8", "windows-1252"} {
			result, err := Extract(iotest.ErrReader(readErr), Options{InputEncoding: label})
			assert.ErrorIs(test, err, readErr)
			assert.Nil(test, result)
		}
	})

	test.Run("unknown charset", func(test *testing.T) {
		readErr := errors.New("input should not be read")
		result, err := Extract(iotest.ErrReader(readErr), Options{InputEncoding: "not-a-charset"})
		assert.ErrorContains(test, err, "unsupported charset")
		assert.NotErrorIs(test, err, readErr)
		assert.Nil(test, result)
	})
}

func Test_Trim(t *testing.T) {
	// Test string trimming
	assert.Equal(t, "Test", trim("	Test  "))
	assert.Equal(t, "Test Test", trim("\t\tTest  Test\r\n"))

	elem := etree.Element("body")
	etree.SetText(elem, "Test Text")
	assert.False(t, textFilter(elem))

	etree.SetText(elem, "Instagram")
	assert.True(t, textFilter(elem))

	etree.SetText(elem, "\t\t")
	assert.True(t, textFilter(elem))
}

func Test_ExoticTags(t *testing.T) {
	var result *ExtractResult
	var htmlString string
	var opts Options

	// Cover some edge cases with a specially crafted file
	result = extractMockFile(trafilaturaMockFiles, "http://exotic_tags")
	assert.Contains(t, result.ContentText, "Teletype text")
	assert.Contains(t, result.ContentText, "My new car is silver.")

	// Misformed HTML declaration
	htmlString = `<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01 Transitional//EN" 2012"http://www.w3.org/TR/html4/loose.dtd"><html><head></head><body><p>ABC</p></body></html>`
	result, err := Extract(strings.NewReader(htmlString), zeroOpts)
	assert.Nil(t, err)
	assert.Contains(t, result.ContentText, "ABC")

	// Quotes
	potentialTags := maps.Clone(tagCatalog)
	assert.Nil(t, handleQuotes(etree.Element("blockquote"), nil, zeroOpts))
	assert.Nil(t, handleTable(etree.Element("table"), potentialTags, nil, zeroOpts))

	// Nested <p> with trailing line break
	element, second := etree.Element("p"), etree.Element("p")
	etree.SetText(element, "1st part.")
	etree.SetText(second, "2nd part.")
	etree.Append(element, second)
	etree.SubElement(element, "br")

	converted := handleParagraphs(element, map[string]struct{}{"p": {}}, nil, zeroOpts)
	assert.Equal(t, "<p>1st part. 2nd part.</p>", etree.ToString(converted))

	// Naked div with <br>
	opts = Options{Config: zeroConfig}
	htmlString = `<html><body><main><div>1.<br/>2.<br/>3.<br/></div></main></body></html>`
	result, _ = Extract(strings.NewReader(htmlString), opts)
	assert.Contains(t, result.ContentText, "1.\n2.\n3.")

	// HTML5: <details>
	opts = Options{Config: zeroConfig}
	htmlString = `<html><body><article><details><summary>Epcot Center</summary><p>Epcot is a theme park at Walt Disney World Resort featuring exciting attractions, international pavilions, award-winning fireworks and seasonal special events.</p></details></article></body></html>`
	result, _ = Extract(strings.NewReader(htmlString), opts)
	assert.Contains(t, result.ContentText, "Epcot Center")
	assert.Contains(t, result.ContentText, "award-winning fireworks")

	htmlString = `<html><body><article><details><summary>Epcot Center</summary><p>Epcot is a theme park at Walt Disney World Resort featuring exciting attractions, international pavilions, award-winning fireworks and seasonal special events.</p></details></article></body></html>`
	result, _ = Extract(strings.NewReader(htmlString), opts)
	assert.Contains(t, result.ContentText, "Epcot Center")
	assert.Contains(t, result.ContentText, "award-winning fireworks")

	// Edge cases
	htmlString = `
	<!DOCTYPE html>
	<html>
	<head>
		<meta charset="UTF-8">
		<title>A weird bug</title>
	</head>
	<body>
		<div>
			<h1>Lorem ipsum dolor sit amet, consectetur adipiscing elit.</h1>
			<h2>Sed et interdum lectus.</h2>
			<p>Quisque molestie nunc eu arcu condimentum fringilla.</p>
			<!-- strong can be changed to b, em, i, u, or kbd -->
			<strong><a></a></strong>
			<h2>Aliquam eget interdum elit, id posuere ipsum.</h2>
			<p>Phasellus lectus erat, hendrerit sed tortor ac, dignissim vehicula metus.<br/></p>
		</div>
	</body>
	</html>`
	opts = Options{IncludeLinks: true, IncludeImages: true}
	result, _ = Extract(strings.NewReader(htmlString), opts)
	assert.NotEmpty(t, result.ContentText)

	htmlString = `
	<html>
	<head>
		<meta charset="UTF-8">
		<title>A weird bug</title>
	</head>
	<body>
		<div id="content">
			<h1>A header</h1>
			<h2>Very specific bug so odd</h2>
			<h3>Nested header</h3>
			<p>Some "hyphenated-word quote" followed by a bit more text line.</p>
			<em>
				<p>em improperly wrapping p here</p>
			</em>
			<p>Text here<br/></p>
			<h3>More articles</h3>
		</div>
	</body>
	</html>`

	opts = Options{IncludeLinks: true, IncludeImages: true}
	for _, focus := range []ExtractionFocus{Balanced, FavorRecall, FavorPrecision} {
		opts.Focus = focus
		result, _ = Extract(strings.NewReader(htmlString), opts)
		assert.Contains(t, result.ContentText, "em improperly wrapping p here")
		assert.True(t, strings.HasSuffix(result.ContentText, "Text here"))
	}
}

func Test_HtmlProcessing(t *testing.T) {
	var opts Options
	var node *html.Node
	var htmlString string
	var result *ExtractResult

	strToNode := func(s string) *html.Node {
		n, _ := dom.FastParse(strings.NewReader(s))
		return n
	}

	// Paywalls
	opts = Options{Config: zeroConfig}
	htmlString = `<html><body><main><p>1</p><p id="premium">2</p><p>3</p></main></body></html>`
	result, _ = Extract(strings.NewReader(htmlString), opts)
	assert.Equal(t, "1\n3", result.ContentText)

	// Test tail of node deleted if set as text
	node = strToNode(`<div><p></p>tail</div>`)
	node = processNode(dom.QuerySelector(node, "p"), nil, defaultOpts)
	assert.Equal(t, "tail", etree.Text(node))
	assert.Equal(t, "", etree.Tail(node))

	node = strToNode(`<ol><li></li>text in tail</ol>`)
	node = processNode(dom.QuerySelector(node, "li"), nil, defaultOpts)
	assert.Equal(t, "text in tail", etree.Text(node))
	assert.Equal(t, "", etree.Tail(node))

	node = strToNode(`<p><br/>tail</p>`)
	node = processNode(dom.QuerySelector(node, "br"), nil, defaultOpts)
	assert.Equal(t, "", etree.Text(node))
	assert.Equal(t, "tail", etree.Tail(node))

	node = strToNode(`<div><p>some text</p>tail</div>`)
	node = processNode(dom.QuerySelector(node, "p"), nil, defaultOpts)
	assert.Equal(t, "some text", etree.Text(node))
	assert.Equal(t, "tail", etree.Tail(node))

	// Text node handler
	node = strToNode(`<p><a href="url"><strong>bold</strong>inner</a>outer</p>`)
	node = handleTextNode(dom.QuerySelector(node, "a"), nil, false, false, defaultOpts)
	assert.Equal(t, "outer", etree.Tail(node))

	node = strToNode(`<p><a href="url">text</a>tail</p>`)
	node = handleTextNode(dom.QuerySelector(node, "a"), nil, false, false, defaultOpts)
	assert.Equal(t, "text", etree.Text(node))
	assert.Equal(t, "tail", etree.Tail(node))

	node = strToNode(`<p><a href="url"></a>tail</p>`)
	node = handleTextNode(dom.QuerySelector(node, "a"), nil, false, false, defaultOpts)
	assert.Equal(t, "tail", etree.Text(node))
	assert.Equal(t, "", etree.Tail(node))

	node = strToNode(`<p><a href="url">text<strong>bold</strong></a>tail</p>`)
	node = handleTextNode(dom.QuerySelector(node, "a"), nil, false, false, defaultOpts)
	assert.Equal(t, "text", etree.Text(node))
	assert.Equal(t, "tail", etree.Tail(node))
}

func Test_LanguageClassifier(t *testing.T) {
	var lang string
	var htmlInput string
	var result *ExtractResult

	// Content text only
	lang = languageClassifier("Hier ist ein Text auf Deutsch", "")
	assert.Equal(t, "de", lang)

	lang = languageClassifier("Hier ist ein Text auf Deutsch", "")
	assert.NotEqual(t, "en", lang)

	// Comments text
	lang = languageClassifier("Hier ist ein Text auf Deutsch", "Die Kommentare sind aber etwas länger.")
	assert.Equal(t, "de", lang)

	lang = languageClassifier("This is English.", "Die Kommentare sind aber etwas länger.")
	assert.Equal(t, "de", lang)

	// Extraction result
	htmlInput = `<html><body><p>Texto en español</p></body></html>`
	result, _ = Extract(strings.NewReader(htmlInput), zeroOpts)
	assert.Equal(t, "es", result.Metadata.Language)

	htmlInput = `<html><body><p>Après la pluie, le beau temps.</p></body></html>`
	result, _ = Extract(strings.NewReader(htmlInput), zeroOpts)
	assert.Equal(t, "fr", result.Metadata.Language)
}

func Test_LanguageClassifier_Compatibility(test *testing.T) {
	for _, sample := range []struct {
		name, content, comments, language string
	}{
		{"empty", "", "", "af"},
		{"whitespace", " \t\n", "", "af"},
		{"nonlinguistic", "12345 !?", "", "zxx"},
		{"french_phrase", "Texte en français", "", "fr"},
		{"english_sentence", "In sleep a king, but waking no such matter.", "", "en"},
		{"spanish_phrase", "Texto en español", "", "es"},
		{"german_sentence", "Hier ist ein Text auf Deutsch", "", "de"},
		{"italian_sentence", "Questo testo non è affatto in lingua inglese.", "", "it"},
		{"russian_sentence", "В этой статье рассказывается о новой городской библиотеке и мероприятиях для читателей.", "", "ru"},
		{"japanese_sentence", "これは日本語で書かれた文章です。今日は新しい図書館について紹介します。", "", "ja"},
		{"chinese_sentence", "这篇文章介绍了城市的新图书馆，以及读者可以参加的活动。", "", "zh"},
		{"arabic_sentence", "تشرح هذه المقالة كيف يجمع الباحثون البيانات ويقارنون النتائج قبل نشر الدراسة.", "", "ar"},
		{"longer_comments", "This is English.", "Die Kommentare sind aber etwas länger.", "de"},
		{"longer_content", "Die Kommentare sind aber etwas länger.", "This is English.", "de"},
		{"equal_lengths_choose_comments", "Texte en français", "Texto en español ", "es"},
	} {
		test.Run(sample.name, func(test *testing.T) {
			test.Parallel()
			assert.Equal(test, sample.language, languageClassifier(sample.content, sample.comments))
		})
	}
}

type pythonLanguageReference struct {
	Commit    string
	Python    string
	Packages  map[string]string
	Languages []struct {
		Content, Comments, Language string
	}
	LanguageExtraction []struct {
		HTML, Target, Language, Content, Comments string
		Fast, Accepted                            bool
	} `json:"language_extraction"`
	MetadataAttributes []struct {
		HTML, Author string
	} `json:"metadata_attributes"`
	Selectors []struct {
		Tag        string
		Attributes [][2]string
		Matches    []bool
	} `json:"selectors"`
	Pruning []struct {
		HTML      string
		Group     int
		Backup    bool
		Tree      json.RawMessage
		InputTree json.RawMessage `json:"input_tree"`
	} `json:"pruning"`
}

func loadPythonLanguageReference(test *testing.T) pythonLanguageReference {
	test.Helper()
	data, err := os.ReadFile("test-files/python-2.2.0-reference.json")
	if err != nil {
		test.Fatal(err)
	}
	var reference pythonLanguageReference
	if err := json.Unmarshal(data, &reference); err != nil {
		test.Fatal(err)
	}
	assert.Equal(test, "c1bc9531a2a978326112ca9987e1382745116136", reference.Commit)
	assert.Equal(test, "3.12.13", reference.Python)
	assert.Equal(test, "0.4.0", reference.Packages["py3langid"])
	return reference
}

func Test_Python220_LanguageClassifier(test *testing.T) {
	reference := loadPythonLanguageReference(test)
	assert.Len(test, reference.Languages, 324)
	for index, sample := range reference.Languages {
		test.Run(fmt.Sprintf("case_%03d", index), func(test *testing.T) {
			test.Parallel()
			assert.Equal(test, sample.Language, languageClassifier(sample.Content, sample.Comments),
				"content=%q comments=%q", sample.Content, sample.Comments)
		})
	}
}

func Test_Python220_LanguageExtraction(test *testing.T) {
	reference := loadPythonLanguageReference(test)
	assert.Len(test, reference.LanguageExtraction, 36)
	retainedGoLanguages := map[int]string{
		0: "en", 1: "en",
		4: "en", 5: "en",
		8: "en", 9: "en",
		12: "fr", 13: "fr",
		16: "fr", 17: "fr",
		20: "fr", 21: "fr",
		24: "es", 25: "es",
		28: "es", 29: "es",
		32: "es", 33: "es",
	}
	for index, sample := range reference.LanguageExtraction {
		test.Run(fmt.Sprintf("case_%03d", index), func(test *testing.T) {
			test.Parallel()
			document, err := html.Parse(strings.NewReader(sample.HTML))
			if err != nil {
				test.Fatal(err)
			}
			before := dom.OuterHTML(document)
			result, err := ExtractDocument(document, Options{
				TargetLanguage: sample.Target,
				EnableFallback: !sample.Fast,
				HtmlDateMode:   Disabled,
			})
			assert.Equal(test, before, dom.OuterHTML(document))
			if !sample.Accepted {
				assert.Error(test, err)
				assert.Nil(test, result)
				return
			}
			if err != nil {
				test.Fatal(err)
			}
			expectedLanguage := sample.Language
			if sample.Target == "" {
				assert.Empty(test, sample.Language)
				expectedLanguage = retainedGoLanguages[index]
				assert.NotEmpty(test, expectedLanguage)
			}
			assert.Equal(test, expectedLanguage, result.Metadata.Language)
			assert.Equal(test, sample.Content, result.ContentText)
			assert.Equal(test, sample.Comments, result.CommentsText)
		})
	}
}

func Test_Python220_MetaAttributeSelection(test *testing.T) {
	reference := loadPythonLanguageReference(test)
	assert.Len(test, reference.MetadataAttributes, 60)
	retainedGoAuthors := map[int]string{
		5: "Maria Example", 6: "Maria Example",
		25: "Maria Example", 26: "Maria Example",
		45: "Maria Example", 46: "Maria Example",
	}
	for index, sample := range reference.MetadataAttributes {
		test.Run(fmt.Sprintf("case_%03d", index), func(test *testing.T) {
			document, err := html.Parse(strings.NewReader(sample.HTML))
			if err != nil {
				test.Fatal(err)
			}
			result := extractMetadata(document, Options{HtmlDateMode: Disabled})
			expectedAuthor := sample.Author
			if retainedAuthor, retained := retainedGoAuthors[index]; retained {
				assert.Empty(test, sample.Author)
				expectedAuthor = retainedAuthor
			}
			assert.Equal(test, expectedAuthor, result.Author, sample.HTML)
		})
	}
}

func Test_Metadata_AuthorAttributeWhitespace(test *testing.T) {
	cases := []struct {
		name      string
		rule      selector.Rule
		tag       string
		attribute string
		value     string
		matches   bool
	}{
		{"specific-id", selector.MetaAuthor[0], "span", "id", " \tauthor\n", true},
		{"specific-class", selector.MetaAuthor[0], "a", "class", " \tauthor\n", true},
		{"generic-username", selector.MetaAuthor[1], "div", "class", " \tusername\n", true},
		{"generic-case-preserved", selector.MetaAuthor[1], "div", "class", " USERNAME ", false},
		{"generic-extra-token", selector.MetaAuthor[1], "div", "class", " username \t other ", false},
		{"raw-rel", selector.MetaAuthor[0], "a", "rel", " author ", false},
		{"raw-itemprop", selector.MetaAuthor[0], "span", "itemprop", " author name ", false},
		{"discard-id-prefix", selector.MetaAuthorDiscard[0], "section", "id", " \tcomments-thread\n", true},
		{"discard-title", selector.MetaAuthorDiscard[0], "span", "class", " \ttitle\n", true},
		{"discard-date", selector.MetaAuthorDiscard[0], "div", "class", " \tdate\n", true},
		{"discard-class-prefix", selector.MetaAuthorDiscard[0], "section", "class", " \tComments \n thread ", true},
		{"discard-case-preserved", selector.MetaAuthorDiscard[0], "div", "class", " DATE ", false},
		{"discard-extra-token", selector.MetaAuthorDiscard[0], "div", "class", " date \t other ", false},
	}
	for _, sample := range cases {
		test.Run(sample.name, func(test *testing.T) {
			node := &html.Node{
				Type: html.ElementNode,
				Data: sample.tag,
				Attr: []html.Attribute{{Key: sample.attribute, Val: sample.value}},
			}
			assert.Equal(test, sample.matches, sample.rule(node))
			assert.Equal(test, sample.value, dom.GetAttribute(node, sample.attribute))
		})
	}
	test.Run("selection-after-discard", func(test *testing.T) {
		document, err := html.Parse(strings.NewReader(`<html><body><div id=" author "><span class=" title ">Article title</span><span class=" date ">Date label</span><section id=" comments-thread ">Other Writer</section><a class=" username ">Jane Example</a></div></body></html>`))
		if err != nil {
			test.Fatal(err)
		}
		before := dom.OuterHTML(document)
		result := extractMetadata(document, Options{HtmlDateMode: Disabled})
		assert.Equal(test, "Jane Example", result.Author)
		assert.Equal(test, before, dom.OuterHTML(document))
	})
}

func Test_Python220_ContentSelectors(test *testing.T) {
	reference := loadPythonLanguageReference(test)
	assert.Greater(test, len(reference.Selectors), 10000)
	groups := [][]selector.Rule{selector.Content, selector.OverallDiscardedContent, selector.PrecisionDiscardedContent, selector.Comments, selector.DiscardedComments, selector.RemovedComments, selector.DiscardedImage, selector.DiscardedTeaser}
	for index, sample := range reference.Selectors {
		node := &html.Node{Type: html.ElementNode, Data: sample.Tag}
		for _, attribute := range sample.Attributes {
			node.Attr = append(node.Attr, html.Attribute{Key: attribute[0], Val: attribute[1]})
		}
		ruleIndex := 0
		for _, group := range groups {
			for _, rule := range group {
				assert.Equal(test, sample.Matches[ruleIndex], rule(node), "selector %d rule %d: %+v", index, ruleIndex, sample)
				ruleIndex++
			}
		}
	}
}

func pythonTreeSnapshot(node *html.Node) any {
	if node.Type == html.DocumentNode {
		return pythonTreeSnapshot(dom.QuerySelector(node, "html"))
	}
	if node.Type == html.CommentNode {
		return map[string]any{"comment": node.Data}
	}
	tag := node.Data
	switch tag {
	case "li", "dd", "dt":
		tag = "item"
	case "ol", "ul", "dl":
		tag = "list"
	case "blockquote", "pre", "q":
		tag = "quote"
	}
	attributes := [][2]string{}
	for _, attribute := range node.Attr {
		attributes = append(attributes, [2]string{attribute.Key, attribute.Val})
	}
	children := []any{}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.TextNode {
			if child.Data == "" {
				continue
			}
			if len(children) > 0 {
				if previous, ok := children[len(children)-1].(string); ok {
					children[len(children)-1] = previous + child.Data
					continue
				}
			}
			children = append(children, child.Data)
		} else if child.Type == html.ElementNode || child.Type == html.CommentNode {
			children = append(children, pythonTreeSnapshot(child))
		}
	}
	return map[string]any{"tag": tag, "attributes": attributes, "children": children}
}

func pythonTreeImport(test *testing.T, data json.RawMessage) *html.Node {
	test.Helper()
	var text string
	if json.Unmarshal(data, &text) == nil {
		return &html.Node{Type: html.TextNode, Data: text}
	}
	var item struct {
		Tag        string
		Comment    *string
		Attributes [][2]string
		Children   []json.RawMessage
	}
	if err := json.Unmarshal(data, &item); err != nil {
		test.Fatal(err)
	}
	if item.Comment != nil {
		return &html.Node{Type: html.CommentNode, Data: *item.Comment}
	}
	node := &html.Node{Type: html.ElementNode, Data: item.Tag}
	for _, attribute := range item.Attributes {
		node.Attr = append(node.Attr, html.Attribute{Key: attribute[0], Val: attribute[1]})
	}
	for _, child := range item.Children {
		node.AppendChild(pythonTreeImport(test, child))
	}
	return node
}

func Test_Python220_Pruning(test *testing.T) {
	reference := loadPythonLanguageReference(test)
	assert.Len(test, reference.Pruning, 288)
	groups := [][]selector.Rule{selector.Content, selector.OverallDiscardedContent, selector.PrecisionDiscardedContent, selector.Comments, selector.DiscardedComments, selector.RemovedComments, selector.DiscardedImage, selector.DiscardedTeaser}
	for index, sample := range reference.Pruning {
		document := pythonTreeImport(test, sample.InputTree)
		result := pruneUnwantedNodes(document, groups[sample.Group], sample.Backup)
		actual, err := json.Marshal(pythonTreeSnapshot(result))
		if err != nil {
			test.Fatal(err)
		}
		assert.JSONEq(test, string(sample.Tree), string(actual), "pruning %d: %s", index, sample.HTML)
	}
}

func Test_PythonCoreTrace(test *testing.T) {
	corpus := os.Getenv("TRAFILATURA_TRACE_CORPUS")
	if corpus == "" {
		test.Skip("temporary corpus diagnostic")
	}
	data, err := os.ReadFile(corpus)
	if err != nil {
		test.Fatal(err)
	}
	var pages []struct{ File, URL, HTML string }
	if err := json.Unmarshal(data, &pages); err != nil {
		test.Fatal(err)
	}
	for _, page := range pages {
		if page.File != "scmp.com.playbook.html" && page.File != "ebrosia.de.zinfandel.html" {
			continue
		}
		document, err := html.Parse(strings.NewReader(strings.TrimPrefix(page.HTML, "\ufeff")))
		if err != nil {
			test.Fatal(err)
		}
		originalURL, _ := nurl.ParseRequestURI(page.URL)
		opts := Options{OriginalURL: originalURL, ExcludeComments: true, Config: DefaultConfig()}
		cleaned := prepareTree(document, opts)
		potential := maps.Clone(tagCatalog)
		for _, tag := range []string{"table", "tr", "td", "th"} {
			potential[tag] = struct{}{}
		}
		for ruleIndex, rule := range selector.Content {
			if node := selector.Query(cleaned, rule); node != nil {
				pruned := pruneUnwantedSections(dom.Clone(node, true), potential, opts)
				test.Logf("%s rule %d: %s id=%q class=%q before=%d after=%d", page.File, ruleIndex, node.Data, dom.GetAttribute(node, "id"), dom.GetAttribute(node, "class"), len([]rune(dom.TextContent(node))), len([]rune(dom.TextContent(pruned))))
			}
		}
		_, mainText := extractContent(cleaned, lru.NewCache(4096), opts)
		_, sequenceText, _, _ := extractionSequence(dom.Clone(document, true), lru.NewCache(4096), opts)
		_, baselineText := baseline(dom.Clone(document, true))
		_, recallText := recallRetry(dom.Clone(document, true), opts)
		test.Logf("%s MAIN=%d SEQUENCE=%d BASELINE=%d RECALL=%d PAGE=%d", page.File, len([]rune(mainText)), len([]rune(sequenceText)), len([]rune(baselineText)), len([]rune(recallText)), len([]rune(html2txt(document))))
	}
}

func Benchmark_LanguageClassifier(bench *testing.B) {
	input, err := os.Open("test-files/mock/webpage.html.gz")
	if err != nil {
		bench.Fatal(err)
	}
	defer input.Close()
	result, err := Extract(input, Options{EnableFallback: true})
	if err != nil {
		bench.Fatal(err)
	}

	for _, sample := range []struct{ name, text string }{
		{"short_phrase", "Texte en français"},
		{"saved_article", result.ContentText},
	} {
		bench.Run(sample.name, func(bench *testing.B) {
			languageClassifier(sample.text, "")
			bench.ReportAllocs()
			bench.SetBytes(int64(len(sample.text)))
			for bench.Loop() {
				languageClassifier(sample.text, "")
			}
		})
	}
}

func Test_Cache(t *testing.T) {
	cache := lru.NewCache(2)

	div1 := etree.Element("div")
	p1 := etree.SubElement(div1, "p")
	etree.SetText(p1, "AAAA BBBB AAAA BBBB AAAA BBBB AAAA BBBB AAAA BBBB AAAA BBBB AAAA BBBB AAAA BBBB AAAA BBBB AAAA BBBB AAAA BBBB AAAA BBBB AAAA BBBB")

	assert.False(t, duplicateTest(p1, cache, defaultOpts))
	assert.False(t, duplicateTest(p1, cache, defaultOpts))
	assert.False(t, duplicateTest(div1, cache, defaultOpts))
	assert.True(t, duplicateTest(p1, cache, defaultOpts))

	div2 := etree.Element("div")
	p2 := etree.SubElement(div2, "p")
	etree.SetText(p2, "CCCC DDDD CCCC DDDD CCCC DDDD CCCC DDDD CCCC DDDD CCCC DDDD CCCC DDDD CCCC DDDD CCCC DDDD CCCC DDDD CCCC DDDD")

	assert.False(t, duplicateTest(div2, cache, defaultOpts))
	assert.False(t, duplicateTest(p2, cache, defaultOpts))
	assert.False(t, duplicateTest(div2, cache, defaultOpts))
	assert.True(t, duplicateTest(p2, cache, defaultOpts))

	div3 := etree.Element("div")
	p3 := etree.SubElement(div3, "p")
	etree.SetText(p3, "EEEE FFFF EEEE FFFF EEEE FFFF EEEE FFFF EEEE FFFF EEEE FFFF EEEE FFFF EEEE FFFF EEEE FFFF EEEE FFFF EEEE FFFF EEEE FFFF EEEE FFFF")

	assert.False(t, duplicateTest(div3, cache, defaultOpts))
	assert.False(t, duplicateTest(div3, cache, defaultOpts))
	assert.False(t, duplicateTest(div3, cache, defaultOpts))

	// Since cache haven't been cleared, try the old nodes
	assert.True(t, duplicateTest(p2, cache, defaultOpts))
	assert.True(t, duplicateTest(p3, cache, defaultOpts))
	assert.False(t, duplicateTest(p1, cache, defaultOpts))

	// Clear the cache then try again
	cache.Clear()
	assert.False(t, duplicateTest(p2, cache, defaultOpts))

	// Get wrong key
	val, exist := cache.Get("tralala")
	assert.Zero(t, val)
	assert.False(t, exist)
}

func Test_Formatting(t *testing.T) {
	var r io.Reader
	var opts Options
	var result *ExtractResult
	fnHtml := func(r *ExtractResult) string {
		return etree.ToString(r.ContentNode)
	}

	// Trailing line break
	r = strings.NewReader("<html><body><p>This here is the text.<br/></p></body></html>")
	result, _ = Extract(r, zeroOpts)
	assert.NotContains(t, fnHtml(result), "<br/>")

	// Simple
	r = strings.NewReader("<html><body><p><b>This here is in bold font.</b></p></body></html>")
	result, _ = Extract(r, zeroOpts)
	assert.Contains(t, fnHtml(result), "<p><b>This here is in bold font.</b></p>")

	// Title
	r = strings.NewReader("<html><body><article><h3>Title</h3><p><b>This here is in bold font.</b></p></article></body></html>")
	result, _ = Extract(r, zeroOpts)
	assert.Contains(t, fnHtml(result), "<h3>Title</h3>")
	assert.Contains(t, fnHtml(result), "<p><b>This here is in bold font.</b></p>")

	// Nested
	r = strings.NewReader("<html><body><p><b>This here is in bold and <i>italic</i> font.</b></p></body></html>")
	result, _ = Extract(r, zeroOpts)
	assert.Contains(t, fnHtml(result), "<p><b>This here is in bold and <i>italic</i> font.</b></p>")

	// Empty
	r = strings.NewReader("<html><body><p><b><i></i></b></p></body></html>")
	result, _ = Extract(r, zeroOpts)
	assert.Contains(t, fnHtml(result), "<body></body>")

	// Wild div
	r = strings.NewReader("<html><body><article><div><strong>Wild text</strong></div></article></body></html>")
	result, _ = Extract(r, zeroOpts)
	assert.Contains(t, fnHtml(result), "<p>")
	assert.Contains(t, fnHtml(result), "<strong>Wild text</strong>")
	assert.Equal(t, "Wild text", result.ContentText)

	// Links
	r = strings.NewReader(`<html><body><p><a href="">Link text</a></p></body></html>`)
	result, _ = Extract(r, zeroOpts)
	assert.Equal(t, "Link text", dom.TextContent(result.ContentNode))

	// Line breaks
	r = strings.NewReader(`<html><body><p><br/></p></body></html>`)
	result, _ = Extract(r, zeroOpts)
	assert.Equal(t, "", dom.TextContent(result.ContentNode))

	r = strings.NewReader(`<html><body><p><br/>Here is the text.</p></body></html>`)
	result, _ = Extract(r, zeroOpts)
	assert.Equal(t, "Here is the text.", dom.TextContent(result.ContentNode))

	// Handle formatting tails
	body := etree.Element("body")
	element := etree.SubElement(body, "b")
	etree.SetText(element, "Here is the text.")
	etree.SetTail(element, "And a tail.")

	converted := handleFormatting(element, nil, zeroOpts)
	assert.Equal(t, "<p><b>Here is the text.</b>And a tail.</p>", etree.ToString(converted))

	// Empty elements
	r = strings.NewReader("<html><body><div>\t\n</div><div>There is text here.</div></body></html>")
	result, _ = Extract(r, zeroOpts)
	assert.Equal(t, "<div><p>There is text here.</p></div>", fnHtml(result))

	// List with links
	opts = Options{IncludeLinks: true, Config: zeroConfig}
	r = strings.NewReader(`<html><body><article><ul><li>Number 1</li><li>Number <a href="test.html">2</a></li><li>Number 3</li><p>Test</p></article></body></html>`)
	result, _ = Extract(r, opts)
	assert.Contains(t, fnHtml(result), `<li>Number <a href="test.html">2</a></li>`)

	// (Markdown) formatting within <p>-tag
	rawHTML := `<html><body><p><b>bold</b>, <i>italics</i>, <tt>tt</tt>, <strike>deleted</strike>, <u>underlined</u>, <a href="test.html">link</a> and additional text to bypass detection.</p></body></html>`

	opts = Options{IncludeLinks: false, Config: zeroConfig}
	result, _ = Extract(strings.NewReader(rawHTML), opts)
	assert.Equal(t, "bold, italics, tt, deleted, underlined, link and additional text to bypass detection.", dom.TextContent(result.ContentNode))
	assert.Contains(t, dom.OuterHTML(result.ContentNode), `<p><b>bold</b>, <i>italics</i>, <tt>tt</tt>, <strike>deleted</strike>, <u>underlined</u>, link and additional text to bypass detection.</p>`)

	opts = Options{IncludeLinks: true, Config: zeroConfig}
	result, _ = Extract(strings.NewReader(rawHTML), opts)
	assert.Contains(t, dom.OuterHTML(result.ContentNode), `<p><b>bold</b>, <i>italics</i>, <tt>tt</tt>, <strike>deleted</strike>, <u>underlined</u>, <a href="test.html">link</a> and additional text to bypass detection.</p>`)

	// Line break following formatting
	r = strings.NewReader("<html><body><article><p><strong>Staff Review of the Financial Situation</strong><br>Domestic financial conditions remained accommodative over the intermeeting period.</p></article></body></html>")
	result, _ = Extract(r, zeroOpts)
	assert.Equal(t, "Staff Review of the Financial Situation\nDomestic financial conditions remained accommodative over the intermeeting period.", dom.InnerText(result.ContentNode))

	// Title with formatting
	r = strings.NewReader(`
		<html><body>
			<article>
				<h4 id="1theinoperator">1) The <code>in</code> Operator</h4>
				<p>The easiest way to check if a Python string contains a substring is to use the <code>in</code> operator. The <code>in</code> operator is used to check data structures for membership in Python. It returns a Boolean (either <code>True</code> or <code>False</code>) and can be used as follows:</p>
			</article>
		</body></html>`)
	result, _ = Extract(r, zeroOpts)
	assert.Contains(t, fnHtml(result), `<h4>1) The <code>in</code> Operator</h4>`)
	assert.Contains(t, fnHtml(result), `<p>The easiest way to check if a Python string contains a substring is to use the <code>in</code> operator.`)
	assert.Contains(t, fnHtml(result), `The <code>in</code> operator is used to check data structures for membership in Python.`)
	assert.Contains(t, fnHtml(result), `It returns a Boolean (either <code>True</code> or <code>False</code>) and can be used as follows:`)

	// Double <p> elems
	r = strings.NewReader("<html><body><p>AAA, <p>BBB</p>, CCC.</p></body></html>")
	result, _ = Extract(r, Options{IncludeLinks: true, Config: zeroConfig})
	assert.Contains(t, result.ContentText, "AAA")
	assert.Contains(t, result.ContentText, "BBB")
	assert.Contains(t, result.ContentText, "CCC")
}

func Test_Filters(t *testing.T) {
	// Helper function
	rRepeatElement := func(element string, repeat int) io.Reader {
		str := fmt.Sprintf("<html><body>%s</body></html>", strings.Repeat(element, repeat))
		return strings.NewReader(str)
	}

	// Recursion limit
	p1 := "<p>abc</p>"
	p2 := "<p><i>abc</i></p>"
	opts := Options{MaxTreeSize: 500}

	result, _ := Extract(rRepeatElement(p1, 50), opts)
	assert.NotNil(t, result)

	result, _ = Extract(rRepeatElement(p1, 501), opts)
	assert.Nil(t, result)

	result, _ = Extract(rRepeatElement(p2, 501), opts)
	assert.Nil(t, result)

	result, _ = Extract(rRepeatElement(p2, 499), opts)
	assert.NotNil(t, result)

	// HTML lang filter
	// No lang
	opts = Options{TargetLanguage: "en"}
	doc := docFromStr(`<html><body></body></html>`)
	assert.True(t, checkHtmlLanguage(doc, opts, false))

	// Lang detection on content
	str := `html><body><article><p>How many ages hence/Shall this our lofty scene be acted over,/In states unborn and accents yet unknown!</p></article></body></html>`

	opts = Options{TargetLanguage: "de"}
	result, _ = Extract(strings.NewReader(str), opts)
	assert.Nil(t, result)

	opts = Options{TargetLanguage: "en"}
	result, _ = Extract(strings.NewReader(str), opts)
	assert.NotNil(t, result)

	p3 := "<p>Thus have I had thee as a dream doth flatter, In sleep a king, but waking no such matter.</p>"
	str = `<html lang="en-US"><body>` + strings.Repeat(p3, 50) + `</body></html>`

	opts = Options{TargetLanguage: "en"}
	result, _ = Extract(strings.NewReader(str), opts)
	assert.NotNil(t, result)

	opts = Options{TargetLanguage: "de"}
	result, _ = Extract(strings.NewReader(str), opts)
	assert.Nil(t, result)

	str = `<html lang="de-DE"><body>` + strings.Repeat(p3, 50) + `</body></html>`
	opts = Options{TargetLanguage: "de"}
	result, _ = Extract(strings.NewReader(str), opts)
	assert.Nil(t, result)

	// http-equiv="content-language"
	opts.TargetLanguage = "en"
	doc = docFromStr(`<html><head><meta http-equiv="content-language" content="en"></head><body></body></html>`)
	assert.True(t, checkHtmlLanguage(doc, opts, false))

	opts.TargetLanguage = "de"
	doc = docFromStr(`<html><head><meta http-equiv="content-language" content="en"></head><body></body></html>`)
	assert.False(t, checkHtmlLanguage(doc, opts, false))

	opts.TargetLanguage = "de"
	doc = docFromStr(`<html><head><meta http-equiv="content-language" content="DE"></head><body></body></html>`)
	assert.True(t, checkHtmlLanguage(doc, opts, false))

	// HTML lang attribute superseded by og:locale
	doc = docFromStr(`<html lang="en-US"><head><meta property="og:locale" content="de_DE" /></head><body></body></html>`)

	opts.TargetLanguage = "de"
	assert.True(t, checkHtmlLanguage(doc, opts, false))

	opts.TargetLanguage = "en"
	assert.False(t, checkHtmlLanguage(doc, opts, false))

	// Last choice: HTML lang attribute
	doc = docFromStr(`<html lang="de_DE, en_US"><body></body></html>`)

	opts.TargetLanguage = "de"
	assert.True(t, checkHtmlLanguage(doc, opts, false))
	assert.True(t, checkHtmlLanguage(doc, opts, true))

	opts.TargetLanguage = "en"
	assert.True(t, checkHtmlLanguage(doc, opts, false))
	assert.True(t, checkHtmlLanguage(doc, opts, true))

	// If strict, lang attribute in <html> should be checked
	opts.TargetLanguage = "it"
	doc = docFromStr(`<html lang="en"><body></body></html>`)
	assert.False(t, checkHtmlLanguage(doc, opts, true)) // is strict
	assert.True(t, checkHtmlLanguage(doc, opts, false)) // not strict

	// Even if in strict mode, lang in <html> attribute is the last choice
	opts.TargetLanguage = "de"
	doc = docFromStr(`<html lang="en-US"><head><meta property="og:locale" content="de_DE" /></head><body></body></html>`)
	assert.True(t, checkHtmlLanguage(doc, opts, true))  // is strict
	assert.True(t, checkHtmlLanguage(doc, opts, false)) // not strict
}

func Test_External(t *testing.T) {
	var f io.Reader
	var opts Options
	var doc *html.Node
	var result *ExtractResult

	// Remove unwanted elements
	doc = docFromStr(`<html><body><footer>Test text</footer></body></html>`)
	sanitizeTree(doc, defaultOpts)
	assert.Empty(t, etree.IterText(doc, " "))

	doc = docFromStr(`<html><body><table><th>Test text</th><tr><td>Test</td></tr></table></body></html>`)
	sanitizeTree(doc, defaultOpts)
	assert.NotEmpty(t, etree.IterText(doc, " "))

	// Strip fancy tags while excluding links and images
	doc = docFromStr(`<html><body><p>Text here <fancy>Test text</fancy><a href="">with a link</a>.</p><img src="test.jpg"/></body></html>`)
	sanitizeTree(doc, defaultOpts)

	mainTree := dom.QuerySelector(doc, "body")
	assert.Len(t, dom.Children(mainTree), 1)

	// Strip fancy tags while including links and images
	opts = Options{IncludeLinks: true, IncludeImages: true}
	doc = docFromStr(`<html><body><p>Text here <fancy>Test text</fancy><a href="">with a link</a>.</p><img src="test.jpg"/></body></html>`)
	sanitizeTree(doc, opts)

	mainTree = dom.QuerySelector(doc, "body")
	aNodes := dom.GetElementsByTagName(mainTree, "a")
	imgNodes := dom.GetElementsByTagName(mainTree, "img")

	assert.Len(t, dom.Children(mainTree), 2)
	assert.NotZero(t, len(aNodes))
	assert.NotZero(t, len(imgNodes))

	// Test language
	opts = Options{TargetLanguage: "en"}
	str := `<html><body>` + strings.Repeat("<p>Non è inglese.</p>", 20) + `</body></html>`
	result, _ = Extract(strings.NewReader(str), opts)
	assert.Nil(t, result)

	// No tables
	f, _ = os.Open(filepath.Join("test-files", "simple", "apache.html"))
	doc, _ = html.Parse(f)

	opts = Options{ExcludeTables: false}
	result, _ = ExtractDocument(doc, opts)
	assert.Contains(t, result.ContentText, "localhost:80")

	opts = Options{ExcludeTables: true}
	result, _ = ExtractDocument(doc, opts)
	assert.NotContains(t, result.ContentText, "localhost:80")

	// Table sub elements
	f, _ = os.Open(filepath.Join("test-files", "simple", "scam.html"))
	doc, _ = html.Parse(f)

	opts = Options{ExcludeTables: true, Config: zeroConfig}
	result, _ = ExtractDocument(doc, opts)
	assert.Empty(t, result.ContentText)

	opts = Options{ExcludeTables: true, EnableFallback: true, Config: zeroConfig}
	result, _ = ExtractDocument(doc, opts)
	assert.Empty(t, result.ContentText)
	assert.NotContains(t, result.ContentText, "Uncensored Hosting")
	assert.NotContains(t, result.ContentText, "ChooseBetter")
}

func Test_Images(t *testing.T) {
	// File type
	assert.True(t, isImageFile("test.jpg"))
	assert.False(t, isImageFile("test.txt"))

	for _, source := range []string{"PHOTO.JPG", "photo.JpEg?width=400", "image.PNG#view", "photo.AVIF", "photo.HEIC"} {
		assert.True(t, isImageFile(source), source)
	}
	assert.False(t, isImageFile("photo.jpgx"))
	assert.False(t, isImageFile(strings.Repeat("x", 8192)+".jpg"))

	// Image handler
	img := handleImage(nil)
	assert.Nil(t, img)

	img = handleImage(etree.FromString(`<img src="test.jpg"/>`))
	assert.NotNil(t, img)

	img = handleImage(etree.FromString(`<img data-src="test.jpg" alt="text" title="a title"/>`))
	assert.NotNil(t, img)

	img = handleImage(etree.FromString(`<img other="test.jpg"/>`))
	assert.Nil(t, img)

	// Extension checker
	assert.True(t, isImageFile("test.jpg"))
	assert.False(t, isImageFile("test.txt"))

	// Text element handler
	assert.Nil(t, handleTextElem(etree.Element("img"), nil, nil, defaultOpts))

	// From file
	f, _ := os.Open(filepath.Join("test-files", "simple", "http_sample.html"))
	bt, _ := io.ReadAll(f)

	// Comparing between includeImages = true and false
	opts := defaultOpts
	opts.Config = zeroConfig

	result, _ := Extract(bytes.NewReader(bt), opts)
	contentHtml := dom.OuterHTML(result.ContentNode)
	assert.NotContains(t, contentHtml, `<img src="test.jpg" title="Example image"/>`)

	opts.IncludeImages = true
	result, _ = Extract(bytes.NewReader(bt), opts)
	contentHtml = dom.OuterHTML(result.ContentNode)
	assert.Contains(t, contentHtml, `<img src="test.jpg" title="Example image"/>`)

	// From string
	var str string
	str = `<html><body><article><p><img data-src="test.jpg" alt="text" title="a title"/></p></article></body></html>`
	result, _ = Extract(strings.NewReader(str), opts)
	contentHtml = dom.OuterHTML(result.ContentNode)
	assert.Contains(t, contentHtml, `<img src="test.jpg" alt="text" title="a title"/>`)

	str = `<html><body><article><p><img other="test.jpg" alt="text" title="a title"/></p></article></body></html>`
	result, _ = Extract(strings.NewReader(str), opts)
	contentHtml = dom.OuterHTML(result.ContentNode)
	assert.Equal(t, `<body></body>`, contentHtml)

	str = `<html><body><article><div><p><img data-src="test.jpg" alt="text" title="a title"/></p></div></article></body></html>`
	result, _ = Extract(strings.NewReader(str), opts)
	contentHtml = dom.OuterHTML(result.ContentNode)
	assert.Contains(t, contentHtml, `<img src="test.jpg" alt="text" title="a title"/>`)

	str = `<html><body><article><div><p><img data-src-small="test.jpg" alt="text" title="a title"/></p></div></article></body></html>`
	result, _ = Extract(strings.NewReader(str), opts)
	contentHtml = dom.OuterHTML(result.ContentNode)
	assert.Contains(t, contentHtml, `<img src="test.jpg" alt="text" title="a title"/>`)

	str = `<img src="data:image/jpeg;base64,iVBORw0KGgoAAAANSUhEUgAAAAUAAAAFCAYAAACNbyblAAAAHElEQVQI12P4//8/w38GIAXDIBKE0DHxgljNBAAO9TXL0Y4OHwAAAABJRU5ErkJggg==" alt="text"></img>`
	result, _ = Extract(strings.NewReader(str), opts)
	contentHtml = dom.OuterHTML(result.ContentNode)
	assert.Equal(t, `<body></body>`, contentHtml)

	// CNN example
	f, _ = os.Open(filepath.Join("test-files", "simple", "cnn-image.html"))
	doc, _ := html.Parse(f)
	img = handleImage(dom.QuerySelector(doc, "img"))
	assert.NotNil(t, img)
	assert.True(t, dom.HasAttribute(img, "alt"))
	assert.True(t, dom.HasAttribute(img, "src"))

	// Modified CNN example
	f, _ = os.Open(filepath.Join("test-files", "simple", "cnn-image-modified.html"))
	doc, _ = html.Parse(f)
	img = handleImage(dom.QuerySelector(doc, "img"))
	assert.NotNil(t, img)
	assert.True(t, dom.HasAttribute(img, "alt"))
	assert.True(t, dom.HasAttribute(img, "src"))
	assert.True(t, strings.HasPrefix(dom.GetAttribute(img, "src"), "http"))
}

func Test_Images_RelativeSourcesAndTails(test *testing.T) {
	baseURL, err := nurl.Parse("https://example.org/articles/page.html")
	assert.NoError(test, err)
	testCases := []struct {
		source   string
		expected string
	}{
		{"photo.jpg", "https://example.org/articles/photo.jpg"},
		{"../photo.jpg", "https://example.org/photo.jpg"},
		{"/photo.jpg", "https://example.org/photo.jpg"},
		{"//cdn.example.org/photo.jpg", "https://cdn.example.org/photo.jpg"},
		{"https://cdn.example.org/photo.jpg", "https://cdn.example.org/photo.jpg"},
	}
	for _, testCase := range testCases {
		test.Run(testCase.source, func(test *testing.T) {
			node := etree.FromString(`<div><img src="` + testCase.source + `"> after image</div>`)
			image := handleImage(dom.QuerySelector(node, "img"), Options{OriginalURL: baseURL})
			assert.Equal(test, testCase.expected, dom.GetAttribute(image, "src"))
			assert.Equal(test, " after image", etree.Tail(image))
		})
	}
}

func Test_NestedInline_Upstream22(test *testing.T) {
	testCases := []string{
		`<p>before <a href="/page"><strong>linked <em>words</em></strong></a> after</p>`,
		`<ul><li>before <strong>bold <code>code</code></strong> after</li></ul>`,
		`<blockquote><p>before <a href="/page"><strong>linked words</strong></a> after</p></blockquote>`,
		`<p>Hyper<b>link</b>ed</p>`,
	}
	for _, input := range testCases {
		test.Run(input, func(test *testing.T) {
			opts := Options{Config: zeroConfig, IncludeLinks: true}
			result, err := Extract(strings.NewReader(`<html><body><article>`+input+`</article></body></html>`), opts)
			if assert.NoError(test, err) && assert.NotNil(test, result) {
				assert.Contains(test, etree.ToString(result.ContentNode), input)
				assert.Equal(test, dom.TextContent(etree.FromString(input)), dom.TextContent(result.ContentNode))
			}
		})
	}
}

func Test_Links(t *testing.T) {
	// Prepare options
	linkOpts := Options{
		IncludeLinks: true,
		Config:       zeroConfig,
	}

	// Test handleTextElem
	processed := handleTextElem(etree.Element("a"), nil, nil, defaultOpts)
	assert.Nil(t, processed)

	// Formatting link
	element := etree.FromString(`<a href="testlink.html">Test link text.</a>`)
	processed = handleFormatting(element, nil, zeroOpts)
	assert.NotNil(t, processed)

	// Extracting links with target
	htmlStr := `<html><body><p><a href="testlink.html">Test link text.</a>This part of the text has to be long enough.</p></body></html>`
	result, _ := Extract(strings.NewReader(htmlStr), zeroOpts)
	assert.NotContains(t, dom.OuterHTML(result.ContentNode), "testlink.html")

	result, _ = Extract(strings.NewReader(htmlStr), linkOpts)
	assert.Contains(t, dom.OuterHTML(result.ContentNode), `<a href="testlink.html">Test link text.</a>This part of the text has to be long enough.`)

	// Relative link conversion
	originalURL, _ := nurl.ParseRequestURI("https://www.example.com")
	result, _ = Extract(strings.NewReader(htmlStr), Options{
		IncludeLinks: true,
		Config:       zeroConfig,
		OriginalURL:  originalURL})
	assert.Contains(t, dom.OuterHTML(result.ContentNode), `<a href="https://www.example.com/testlink.html">Test link text.</a>This part of the text has to be long enough.`)

	// Extracting links without target
	htmlStr = `<html><body><p><a>Test link text.</a>This part of the text has to be long enough.</p></body></html>`
	result, _ = Extract(strings.NewReader(htmlStr), linkOpts)
	assert.Contains(t, dom.OuterHTML(result.ContentNode), `<a>Test link text.</a>This part of the text has to be long enough.`)

	htmlStr = `<html><body><article><a>Segment 1</a><h1><a>Segment 2</a></h1><p>Segment 3</p></article></body></html>`
	result, _ = Extract(strings.NewReader(htmlStr), linkOpts)
	assert.Contains(t, result.ContentText, "1")
	assert.Contains(t, result.ContentText, "2")
	assert.Contains(t, result.ContentText, "3")

	// Extracting document with links, from file
	f, _ := os.Open(filepath.Join("test-files", "simple", "http_sample.html"))
	bt, _ := io.ReadAll(f)

	result, _ = Extract(bytes.NewReader(bt), zeroOpts)
	assert.NotContains(t, dom.OuterHTML(result.ContentNode), "testlink.html")

	result, _ = Extract(bytes.NewReader(bt), linkOpts)
	assert.Contains(t, dom.OuterHTML(result.ContentNode), "testlink.html")

	// Test license link
	htmlStr = `<html><body><p>Test text under <a rel="license" href="">CC BY-SA license</a>.</p></body></html>`
	result, _ = Extract(strings.NewReader(htmlStr), linkOpts)
	assert.Contains(t, dom.OuterHTML(result.ContentNode), "<a>CC BY-SA license</a>")

	// Link in p, length threshold
	// var opts Options
	htmlStr = `<html><body><article><p><a>` + strings.Repeat("abcd", 20) + `</a></p></article></body></html>`

	opts := Options{Config: zeroConfig, Focus: Balanced}
	result, _ = Extract(strings.NewReader(htmlStr), opts)
	assert.Contains(t, dom.TextContent(result.ContentNode), "abcd")

	opts = Options{Config: zeroConfig, Focus: FavorPrecision}
	result, _ = Extract(strings.NewReader(htmlStr), opts)
	assert.Empty(t, dom.TextContent(result.ContentNode))
}

func Test_ExtractionOptions(t *testing.T) {
	var opts Options
	var result *ExtractResult
	htmlStr := `<html>
		<head>
			<meta http-equiv="content-language" content="EN" />
		</head>
		<body>
			<div="article-body">
				<p>Text.<!-- comment --></p>
			</div>
		</body>
	</html>`

	opts = Options{Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlStr), opts)
	assert.NotNil(t, result)

	opts = Options{HasEssentialMetadata: true, Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlStr), opts)
	assert.Nil(t, result)

	opts = Options{TargetLanguage: "de", Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlStr), opts)
	assert.Nil(t, result)

	// Try HtmlDate config
	htmlStr = `<html><head/><body>` +
		strings.Repeat(`<p>ABC def ghi jkl.</p>`, 1000) +
		`<p>Posted on 1st Dec 2019<.</p></body></html>`
	doc, _ := dom.FastParse(strings.NewReader(htmlStr))

	dateOpts := htmldate.Options{UseOriginalDate: true}
	opts = Options{Config: zeroConfig, HtmlDateOptions: &dateOpts}

	meta := extractMetadata(doc, opts)
	assert.NotZero(t, meta.Date)

	dateOpts.SkipExtensiveSearch = true
	meta = extractMetadata(doc, opts)
	assert.Zero(t, meta.Date)
}

func Test_PrecisionRecall(t *testing.T) {
	var opts Options
	var result *ExtractResult
	var htmlStr string

	// Basic test
	htmlStr = `<html><body><p>This here is the text.</p></body></html>`

	opts = Options{Focus: FavorPrecision, Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlStr), opts)
	assert.NotNil(t, result)

	opts = Options{Focus: FavorRecall, Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlStr), opts)
	assert.NotNil(t, result)

	// Teaser text
	htmlStr = `<html><body>
		<div class="article-body">
			<div class="teaser-content">
				<p>This here is a teaser text.</p>
			</div>
			<p>This here is the text.</p>
		</div>
	</body></html>`

	opts = Options{Focus: FavorRecall, Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlStr), opts)
	assert.Contains(t, result.ContentText, "teaser text")

	opts = Options{Focus: Balanced, Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlStr), opts)
	assert.NotContains(t, result.ContentText, "teaser text")

	opts = Options{Focus: FavorPrecision, Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlStr), opts)
	assert.NotContains(t, result.ContentText, "teaser text")

	// Never extracted
	htmlStr = `<html><body><article><div><p>
		<a href="test.html">1.</a>
		<br />
		<a href="test2.html">2.</a>
	</p></div></article></body></html>`

	opts = Options{Focus: FavorRecall, Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlStr), opts)
	assert.NotContains(t, result.ContentText, "1")

	opts = Options{Focus: FavorPrecision, Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlStr), opts)
	assert.NotContains(t, result.ContentText, "1")

	// Only found when favor recall
	htmlStr = `<html><body>
		<div class="article-body">
			<p>content</p>
			<p class="link">Test</p>
		</div>
	</body></html>`

	opts = Options{Focus: FavorRecall, Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlStr), opts)
	assert.Contains(t, result.ContentText, "content")
	assert.Contains(t, result.ContentText, "Test")

	opts = Options{Focus: FavorPrecision, Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlStr), opts)
	assert.Contains(t, result.ContentText, "content")
	assert.NotContains(t, result.ContentText, "Test")

	htmlStr = `<html><body><article>
		<aside><p>Here is the text.</p></aside>
	</article></body></html>`

	opts = Options{Focus: Balanced, Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlStr), opts)
	assert.NotEqual(t, "Here is the text.", result.ContentText)

	opts = Options{Focus: FavorRecall, Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlStr), opts)
	assert.Equal(t, "Here is the text.", result.ContentText)

	htmlStr = `<html><body><div>
		<h2>Title</h2>
		<small>Text.</small>
	</div></body></html>`
	opts = Options{Focus: FavorRecall, Config: zeroConfig, EnableFallback: true}
	result, _ = Extract(strings.NewReader(htmlStr), opts)
	assert.NotEmpty(t, result.ContentText)

	htmlStr = `<html><body><div>
		<span>Text.</span>
	</div></body></html>`

	opts = Options{Focus: FavorPrecision, Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlStr), opts)
	assert.Empty(t, result.ContentText)

	opts = Options{Focus: FavorRecall, Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlStr), opts)
	assert.Equal(t, "Text.", result.ContentText)
}

func Test_Fallback_Upstream22(test *testing.T) {
	extracted := etree.FromString(`<div><p>` + strings.Repeat("Article text. ", 12) + `</p></div>`)
	rawJSON := etree.FromString(`<div><p>{` + strings.Repeat(`"data": 123,`, 50) + `}</p></div>`)
	assert.False(test, candidateIsUsable(rawJSON, extracted, len(dom.TextContent(rawJSON)), len(dom.TextContent(extracted)), defaultOpts))

	candidate := etree.FromString(`<div><p>` + strings.Repeat("Recovered article text. ", 12) + `</p></div>`)
	opts := Options{Focus: FavorRecall, Config: DefaultConfig()}
	assert.True(test, candidateIsUsable(candidate, extracted, len(dom.TextContent(candidate)), len(dom.TextContent(extracted)), opts))
	assert.False(test, candidateIsUsable(etree.Element("div"), extracted, 0, 150, opts))

	opts = Options{Config: DefaultConfig(), EnableFallback: true, FallbackCandidates: &FallbackCandidates{Readability: candidate}}
	original := etree.ToString(candidate)
	_, _ = compareExternalExtraction(docFromStr(`<html><body></body></html>`), etree.Element("body"), opts)
	assert.Equal(test, original, etree.ToString(candidate))

	opts.IncludeLinks = true
	opts.OriginalURL = exampleURL
	candidate = etree.FromString(`<div><p><a href="/article">Linked text</a></p></div>`)
	sanitizeTree(candidate, opts)
	assert.Equal(test, "https://example.org/article", dom.GetAttribute(dom.QuerySelector(candidate, "a"), "href"))
}

func Test_ExtractionSequence_Upstream22(test *testing.T) {
	mainText := strings.Repeat("Primary article content. ", 20)
	commentText := strings.Repeat("Reader discussion text. ", 20)
	for _, fallback := range []bool{false, true} {
		for _, focus := range []ExtractionFocus{Balanced, FavorRecall, FavorPrecision} {
			input := `<html><body><article><p>` + mainText + `</p></article><details id="comments"><p>` + commentText + `</p></details></body></html>`
			result, err := Extract(strings.NewReader(input), Options{ExcludeComments: true, Focus: focus, EnableFallback: fallback})
			if assert.NoError(test, err) {
				assert.Contains(test, result.ContentText, "Primary article content.")
				assert.NotContains(test, result.ContentText, "Reader discussion text.")
				assert.Empty(test, result.CommentsText)
			}
		}
	}

	for _, schema := range []string{
		`{"@type":"DiscussionForumPosting"}`,
		`{"@type":["Article","DiscussionForumPosting"]}`,
		`{"@graph":[{"@type":"DiscussionForumPosting"}]}`,
	} {
		for _, exclude := range []bool{false, true} {
			input := `<html><head><script type="application/ld+json">` + schema + `</script></head><body><p>` + mainText + `</p><div id="comments"><p>` + commentText + `</p></div></body></html>`
			result, err := Extract(strings.NewReader(input), Options{ExcludeComments: exclude})
			if assert.NoError(test, err) {
				assert.Contains(test, result.ContentText, "Reader discussion text.")
				assert.Empty(test, result.CommentsText)
			}
		}
	}
	for _, schema := range []string{`{"@type":"QAPage"}`, `{"description":"DiscussionForumPosting"}`, `{"@type":7}`, `{invalid}`} {
		assert.False(test, forumThreadPage(docFromStr(`<html><script type="application/ld+json">`+schema+`</script></html>`)))
	}

	additional := strings.Repeat("Additional substantive material in this article. ", 80)
	input := `<html><body><article><p>` + mainText + `</p><div class="teaser-content"><p>` + additional + `</p></div></article></body></html>`
	opts := Options{Focus: Balanced, Config: DefaultConfig()}
	result, err := Extract(strings.NewReader(input), opts)
	if assert.NoError(test, err) {
		assert.Contains(test, result.ContentText, "Additional substantive material")
		assert.Equal(test, 1, strings.Count(result.ContentText, trim(mainText)))
	}
	assert.Equal(test, Balanced, opts.Focus)
	assert.Equal(test, DefaultConfig(), opts.Config)
	opts.Focus = FavorPrecision
	result, err = Extract(strings.NewReader(input), opts)
	if assert.NoError(test, err) {
		assert.NotContains(test, result.ContentText, "Additional substantive material")
	}
}

func Test_TableProcessing(t *testing.T) {
	var opts Options
	var processedTable *html.Node
	var nodeValues []string
	potentialTags := maps.Clone(tagCatalog)
	iterNodeValues := func(root *html.Node) []string {
		var nodeValues []string
		for _, node := range etree.Iter(root) {
			nodeTag := dom.TagName(node)
			nodeText := trim(etree.Text(node))
			if nodeText == "" {
				nodeValues = append(nodeValues, nodeTag)
			} else {
				nodeValues = append(nodeValues, nodeTag+"-"+nodeText)
			}
		}
		return nodeValues
	}

	// Simple table
	tableSimpleCell := etree.FromString(`<table><tr><td>cell1</td><td>cell2</td></tr><tr><td>cell3</td><td>cell4</td></tr></table>`)
	processedTable = handleTable(tableSimpleCell, potentialTags, nil, defaultOpts)
	assert.Equal(t, []string{"table", "tr", "td-cell1", "td-cell2", "tr", "td-cell3", "td-cell4"}, iterNodeValues(processedTable))

	// If a cell contains 'exotic' tags, they are cleaned during the extraction
	// Process and the content is merged with the parent e.g. <td>
	tableCellWithChildren := etree.FromString(`<table><tr><td><p>text</p><p>more text</p></td></tr></table>`)
	processedTable = handleTable(tableCellWithChildren, potentialTags, nil, defaultOpts)
	assert.Equal(t, `<table><tr><td><p>text</p><p>more text</p></td></tr></table>`, dom.OuterHTML(processedTable))

	// Complex table that hasn't been cleaned yet
	complexPage := docFromStr(`
	<html><body>
		<article>
			<table>
			<tbody>
				<tr>
				<td><small>text<br></small>
					<h4>more_text</h4>
				</td>
				<td><a href='link'>linktext</a></td>
				</tr>
			</tbody>
			</table>
		</article>
	</body></html>`)
	opts = Options{IncludeLinks: true, Config: zeroConfig}
	result, _ := ExtractDocument(complexPage, opts)
	assert.Contains(t, dom.OuterHTML(result.ContentNode), `<table><tr><td>text<h4>more_text</h4></td>`+
		"\n\t\t\t\t"+`<td><a href="link">linktext</a></td>`+"\n\t\t\t\t"+`</tr></table>`)

	// Table cell with text and child
	tableCellWithTextAndChild := etree.FromString(`<table><tr><td>text<lb/><p>more text</p></td></tr></table>`)
	processedTable = handleTable(tableCellWithTextAndChild, potentialTags, nil, defaultOpts)
	assert.Equal(t, `<table><tr><td>text<p>more text</p></td></tr></table>`, dom.OuterHTML(processedTable))

	// Table cell with link
	tableCellWithLink := etree.FromString(`<table><tr><td><a href='test'>link</a></td></tr></table>`)
	processedTable = handleTable(tableCellWithLink, potentialTags, nil, defaultOpts)
	nodeValues = iterNodeValues(dom.QuerySelector(processedTable, "td"))
	assert.Equal(t, []string{"td", "a-link"}, nodeValues)
	assert.Equal(t, "test", dom.GetAttribute(dom.QuerySelector(processedTable, "a"), "href"))

	// Table with head
	tableWithHead := etree.FromString(`
	<table>
		<tr><th>Month</th><th>Days</th></tr>
		<tr><td>January</td><td>31</td></tr>
		<tr><td>February</td><td>28</td></tr>
	</table>`)
	processedTable = handleTable(tableWithHead, potentialTags, nil, defaultOpts)
	assert.Equal(t, 3, len(dom.Children(processedTable)))

	firstRow := dom.Children(processedTable)[0]
	firstRowCells := dom.Children(firstRow)
	assert.Equal(t, 2, len(firstRowCells))
	assert.Equal(t, "th", dom.TagName(firstRowCells[0]))
	assert.Equal(t, "th", dom.TagName(firstRowCells[1]))
	assert.Equal(t, "Month", dom.TextContent(firstRowCells[0]))
	assert.Equal(t, "Days", dom.TextContent(firstRowCells[1]))

	// Table with head span
	tableWithHeadSpan := etree.FromString(`
	<table>
		<tr>
			<th>Name</th>
			<th>Adress</th>
			<th colspan="2">Phone</th>
		</tr>
		<tr>
			<td>Jane Doe</td>
			<td>test@example.com</td>
			<td>phone 1</td>
			<td>phone 2</td>
		</tr>
	</table>`)
	processedTable = handleTable(tableWithHeadSpan, potentialTags, nil, defaultOpts)
	assert.Equal(t, 2, len(dom.Children(processedTable)))

	firstRow = dom.Children(processedTable)[0]
	firstRowCells = dom.Children(firstRow)
	assert.Equal(t, 4, len(firstRowCells))
	assert.Equal(t, "th", dom.TagName(firstRowCells[0]))
	assert.Equal(t, "th", dom.TagName(firstRowCells[1]))
	assert.Equal(t, "th", dom.TagName(firstRowCells[2]))

	// Table cell with formatting
	tableCellWithFormatting := etree.FromString(`<table><tr><td><mark>highlighted text</mark></td></tr></table>`)
	processedTable = handleTable(tableCellWithFormatting, potentialTags, nil, defaultOpts)
	firstCell := dom.QuerySelector(processedTable, "td")
	assert.NotNil(t, firstCell)
	assert.Equal(t, dom.OuterHTML(firstCell), `<td><mark>highlighted text</mark></td>`)

	// Table cell with span
	tableCellWithSpan := etree.FromString(`<table><tr><td><span style='sth'>span text</span></td></tr></table>`)
	processedTable = handleTable(tableCellWithSpan, potentialTags, nil, defaultOpts)
	firstCell = dom.QuerySelector(processedTable, "td")
	assert.NotNil(t, firstCell)
	assert.Equal(t, dom.OuterHTML(firstCell), `<td><p></p></td>`)

	// Table with nested elements
	tableNestedElements := docFromStr(`
	<html><body>
		<article>
			<table>
				<tr>
					<td><b>Present Tense</b></td>
					<td>I buy</td>
					<td>you buy</td>
					<td>he/she/it buys</td>
					<td>we buy</td>
					<td>you buy</td>
					<td>they buy</td>
				</tr>
			</table>
		</article>
	</body></html>`)
	opts = Options{IncludeLinks: true, Config: zeroConfig}
	result, _ = ExtractDocument(tableNestedElements, opts)
	assert.Contains(t, dom.OuterHTML(result.ContentNode), ``+
		`<tr>`+
		`<td><b>Present Tense</b></td>`+"\n\t\t\t\t\t"+
		`<td>I buy</td>`+
		`<td>you buy</td>`+
		`<td>he/she/it buys</td>`+
		`<td>we buy</td>`+
		`<td>you buy</td>`+
		`<td>they buy</td>`+
		`</tr>`)

	// Table with links
	// TODO: further tests and adjustsments
	tableWithLinks := docFromStr(`` +
		`<html><body><article><table><tr><td><a href="test.html">` +
		strings.Repeat("ABCD", 100) +
		`</a></td></tr></table></article></body></html>`)
	opts = Options{IncludeLinks: true, Config: zeroConfig}
	result, _ = ExtractDocument(tableWithLinks, opts)
	assert.NotContains(t, result.ContentText, "ABCD")

	// Nested table 1
	tableNested1 := docFromStr(`
	<html><body><article>
		<table><th>1</th><table><tr><td>2</td></tr></table></table>
	</article></body></html>`)
	opts = Options{IncludeLinks: true, Config: zeroConfig}
	result, _ = ExtractDocument(tableNested1, opts)
	// TODO: all elements are there, but output not nested
	// TODO: th conversion
	assert.Contains(t, dom.OuterHTML(result.ContentNode), "<th>1</th>")
	assert.Contains(t, dom.OuterHTML(result.ContentNode), "<td>2</td>")

	// Nested table 2
	tableNested2 := etree.FromString(`
	<table><tr><td>
		<table><tr><td>1</td></tr></table>
	</td></tr></table>`)
	processedTable = handleTable(tableNested2, potentialTags, nil, defaultOpts)
	assert.NotContains(t, dom.TextContent(processedTable), "1")
	assert.Equal(t, "1", dom.TextContent(dom.QuerySelector(tableNested2, "table")))

	// Nested table - complex
	tableNestedComplex := etree.FromString(`
	<table>
		<tr>
			<td>
				<table><tr><td>1</td></tr></table>
			</td>
			<td>text1</td>
		</tr>
		<tr>
			<td>text2</td>
		</tr>
	</table>`)
	processedTable = handleTable(tableNestedComplex, potentialTags, nil, defaultOpts)
	assert.Equal(t, []string{"table", "tr", "td", "td-text1", "tr", "td-text2", "td"}, iterNodeValues(processedTable))

	// Table with list
	tableWithList := etree.FromString(`
	<table>
		<tr><td>
			<p>a list</p>
			<ul>
				<li>one</li>
				<li>two</li>
			</ul>
		</td></tr>
	</table>`)
	processedTable = handleTable(dom.Clone(tableWithList, true), potentialTags, nil, defaultOpts)
	assert.Equal(t, []string{"table", "tr", "td", "p-a list", "ul"}, iterNodeValues(processedTable))

	recallOpts := Options{Config: DefaultConfig(), Focus: FavorRecall}
	processedTable = handleTable(dom.Clone(tableWithList, true), potentialTags, nil, recallOpts)
	assert.Equal(t, []string{"table", "tr", "td", "p-a list", "ul", "li-one", "li-two"}, iterNodeValues(processedTable))

	// Broken table 1 (broken as in uncommon structure)
	tableBroken1 := etree.FromString(`<table><td>cell1</td><tr><td>cell2</td></tr></table>`)
	processedTable = handleTable(tableBroken1, potentialTags, nil, defaultOpts)
	assert.Equal(t, []string{"table", "tr", "td-cell1", "tr", "td-cell2"}, iterNodeValues(processedTable))

	// Broken table 2
	tableBroken2 := docFromStr(`<table><tr><p>text</p></tr><tr><td>cell</td></tr></table>`)
	tableBroken2 = dom.QuerySelector(tableBroken2, "table")
	processedTable = handleTable(tableBroken2, potentialTags, nil, defaultOpts)
	assert.Equal(t, []string{"table", "tr", "td-cell"}, iterNodeValues(processedTable))

	// Table nested in figure https://github.com/adbar/trafilatura/issues/301
	tableInFigure := docFromStr(`<html><body><article><figure><table><th>1</th><tr><td>2</td></tr></table></figure></article></body></html>`)
	result, _ = ExtractDocument(tableInFigure, zeroOpts)
	assert.Contains(t, dom.OuterHTML(result.ContentNode), "<th>1</th>")
	assert.Contains(t, dom.OuterHTML(result.ContentNode), "<td>2</td>")
}

func Test_TableProcessing_Upstream22(test *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty cells", `<table><tr><td>A</td><td></td><td>C</td></tr><tr><td>D</td></tr></table>`, `<table><tr><td>A</td><td></td><td>C</td></tr><tr><td>D</td><td></td><td></td></tr></table>`},
		{"colspan", `<table><tr><th colspan="2">Header</th><th>Last</th></tr><tr><td>A</td><td>B</td><td>C</td></tr></table>`, `<table><tr><th>Header</th><th></th><th>Last</th></tr><tr><td>A</td><td>B</td><td>C</td></tr></table>`},
		{"rowspan", `<table><tr><td rowspan="2">A</td><td>B</td></tr><tr><td>C</td></tr></table>`, `<table><tr><td>A</td><td>B</td></tr><tr><td></td><td>C</td></tr></table>`},
		{"combined spans", `<table><tr><td rowspan="2" colspan="2">A</td><td>B</td></tr><tr><td>C</td></tr></table>`, `<table><tr><td>A</td><td></td><td>B</td></tr><tr><td></td><td></td><td>C</td></tr></table>`},
		{"rowspan expires on padding", `<table><tr><td>A</td><td rowspan="2">B</td><td>C</td></tr><tr><td>D</td></tr><tr><td>E</td><td>F</td><td>G</td></tr></table>`, `<table><tr><td>A</td><td>B</td><td>C</td></tr><tr><td>D</td><td></td><td></td></tr><tr><td>E</td><td>F</td><td>G</td></tr></table>`},
		{"caption", `<table><caption>Table caption</caption><tr><td>A</td><td>B</td></tr></table>`, `<table><tr><th>Table caption</th><td></td></tr><tr><td>A</td><td>B</td></tr></table>`},
		{"nested inline link", `<table><tr><td><a href="/page"><b>linked words</b></a> tail</td></tr></table>`, `<table><tr><td><a href="/page"><b>linked words</b></a> tail</td></tr></table>`},
		{"nested table tail", `<table><tr><td>before<table><tr><td>nested</td></tr></table>after</td><td>last</td></tr></table>`, `<table><tr><td>beforeafter</td><td>last</td></tr></table>`},
		{"empty rows", `<table><tr><td></td><td></td></tr><tr><td>A</td><td>B</td></tr></table>`, `<table><tr><td>A</td><td>B</td></tr></table>`},
	}
	for _, testCase := range testCases {
		test.Run(testCase.name, func(test *testing.T) {
			node := etree.FromString(testCase.input)
			result := handleTable(node, maps.Clone(tagCatalog), nil, zeroOpts)
			postCleaning(result)
			assert.Equal(test, testCase.expected, etree.ToString(result))
		})
	}
	for _, span := range []string{"n/a", "-1", "1.5", "999999999999999999999999999999999"} {
		node := etree.FromString(`<table><tr><td colspan="` + span + `">bounded</td></tr></table>`)
		result := handleTable(node, maps.Clone(tagCatalog), nil, zeroOpts)
		assert.LessOrEqual(test, len(dom.QuerySelectorAll(result, "td")), maxTableSpan)
		assert.Equal(test, "bounded", dom.TextContent(result))
	}
	for _, testCase := range []struct {
		value string
		span  int
	}{
		{"2", 2}, {"\u0662", 2}, {"\uff12", 2}, {"\U0001d7da", 2},
		{"\u00b2", 1}, {"1\u00b2", 1}, {"2x", 1}, {"", 1}, {"0", 0},
		{strings.Repeat("9", 5000), maxTableSpan}, {strings.Repeat("9", 100) + "x", 1},
	} {
		cell := etree.Element("td")
		dom.SetAttribute(cell, "colspan", testCase.value)
		assert.Equal(test, testCase.span, tableSpan(cell, "colspan"))
	}
	input := `<html><body><article><p>Introduction to the tables.</p><table><tr><td>Outer cell text.</td></tr><tr><td><table><tr><td>Nested cell text.</td></tr></table></td></tr></table></article></body></html>`
	result, err := Extract(strings.NewReader(input), Options{Config: zeroConfig})
	if assert.NoError(test, err) {
		assert.Contains(test, result.ContentText, "Outer cell text.")
		assert.Equal(test, 1, strings.Count(result.ContentText, "Nested cell text."))
		assert.Equal(test, 2, len(dom.QuerySelectorAll(result.ContentNode, "table")))
	}
}

func Test_Recovery_Upstream22(test *testing.T) {
	opts := Options{Config: DefaultConfig()}
	doc := docFromStr(`<html><body><article><h1>Retained heading</h1><p>Initial text.</p></article><p>Recovered text.</p></body></html>`)
	body, text := extractContent(doc, nil, opts)
	assert.Contains(test, text, "Retained heading")
	assert.Contains(test, text, "Recovered text.")
	assert.Equal(test, 1, strings.Count(text, "Initial text."))
	assert.NotNil(test, dom.QuerySelector(body, "h1"))

	longText := strings.Repeat("Substantial repeated content. ", 10)
	doc = docFromStr(`<html><body><article><p>` + longText + `</p><p>` + longText + `</p></article></body></html>`)
	body, _ = extractContent(doc, nil, opts)
	assert.Equal(test, 1, len(dom.QuerySelectorAll(body, "p")))

	doc = docFromStr(`<html><body><article><p>Short repeat</p><p>Short repeat</p></article></body></html>`)
	body, _ = extractContent(doc, nil, zeroOpts)
	assert.Equal(test, 2, len(dom.QuerySelectorAll(body, "p")))

	body = etree.FromString(`<body><p>Hyper<b>link</b>ed text.</p></body>`)
	if dom.TagName(body) != "body" {
		container := etree.Element("body")
		etree.Append(container, body)
		body = container
	}
	doc = docFromStr(`<html><body><p>Hyper<b>link</b>ed text.</p><p>New text.</p></body></html>`)
	recoverWildText(doc, body, tagCatalog, nil, opts)
	assert.Equal(test, 1, strings.Count(dom.TextContent(body), "Hyperlinked text."))
	assert.Contains(test, dom.TextContent(body), "New text.")

	unicodeText := strings.Repeat("\u4e2d", 80_000)
	body = etree.Element("body")
	etree.SetText(etree.SubElement(body, "p"), "prefix"+unicodeText+"suffix")
	doc = docFromStr(`<html><body><p>` + unicodeText + `</p></body></html>`)
	recoverWildText(doc, body, tagCatalog, nil, opts)
	assert.Len(test, dom.Children(body), 1)
}

func Test_Extraction_PreservesInput(test *testing.T) {
	doc := docFromStr(`<html><body><article><p>Keep this text.</p><p class="remove">Remove this text.</p></article></body></html>`)
	original := dom.OuterHTML(doc)
	_, err := ExtractDocument(doc, Options{Config: zeroConfig, PruneSelector: ".remove"})
	assert.NoError(test, err)
	assert.Equal(test, original, dom.OuterHTML(doc))
}

func Test_ListProcessing(t *testing.T) {
	var opts Options
	var processedList *html.Node
	var result *ExtractResult
	var strResult string
	iterNodeValues := func(root *html.Node) []string {
		var nodeValues []string
		for _, node := range etree.Iter(root) {
			nodeTag := dom.TagName(node)
			nodeText := trim(etree.Text(node))
			if nodeText == "" {
				nodeValues = append(nodeValues, nodeTag)
			} else {
				nodeValues = append(nodeValues, nodeTag+"-"+nodeText)
			}
		}
		return nodeValues
	}

	// Malformed lists (common error)
	listMalformed := etree.FromString(`
	<ul>Description of the list:
		<li>List item 1</li>
		<li>List item 2</li>
		<li>List item 3</li>
	</ul>`)
	opts = Options{Config: zeroConfig}
	processedList = handleLists(listMalformed, nil, opts)
	strResult = etree.ToString(processedList)
	assert.Equal(t, 3, strings.Count(strResult, "List item"))
	assert.Contains(t, strResult, "Description")

	// Nested list
	listNested := docFromStr(`
	<html><body><article>
		<ul>
			<li>Coffee</li>
			<li>Tea
				<ul>
					<li>Black tea</li>
					<li>Green tea</li>
				</ul>
			</li>
			<li>Milk</li>
		</ul>
	</article></body></html>`)
	opts = Options{Config: zeroConfig}
	result, _ = ExtractDocument(listNested, opts)
	assert.Contains(t, noSpace(dom.OuterHTML(result.ContentNode)), noSpace(`
	<ul>
		<li>Coffee</li>
		<li>Tea
			<ul>
				<li>Black tea</li>
				<li>Green tea</li>
			</ul>
		</li>
		<li>Milk</li>
	</ul>`))

	// Description list
	listDescription := docFromStr(`
	<html><body><article>
		<dl>
			<dt>Coffee</dt>
			<dd>Black hot drink</dd>
			<dt>Milk</dt>
			<dd>White cold drink</dd>
		</dl>
	</article></body></html>`)
	opts = Options{Config: zeroConfig}
	result, _ = ExtractDocument(listDescription, opts)
	assert.Contains(t, noSpace(dom.OuterHTML(result.ContentNode)), noSpace(`
	<dl>
		<dt>Coffee</dt>
		<dd>Black hot drink</dd>
		<dt>Milk</dt>
		<dd>White cold drink</dd>
	</dl>`))

	// Item with child
	listItemWithChild := etree.FromString(`<ul><li><p>text</p></li></ul>`)
	processedList = handleLists(listItemWithChild, nil, defaultOpts)
	assert.Equal(t, []string{"ul", "li", "p-text"}, iterNodeValues(processedList))

	listItemWithTextAndChild := etree.FromString(`<ul><li>text1<p>text2</p></li></ul>`)
	processedList = handleLists(listItemWithTextAndChild, nil, defaultOpts)
	assert.Equal(t, []string{"ul", "li-text1", "p-text2"}, iterNodeValues(processedList))

	listItemWithBr := etree.FromString(`<ul><li>text<br/>more text</li></ul>`)
	processedList = handleLists(listItemWithBr, nil, defaultOpts)
	assert.Equal(t, []string{"ul", "li-text", "br"}, iterNodeValues(processedList))

	// List with text outside item
	listWithTextOutside := etree.FromString(`<ul>header<li>text</li></ul>`)
	processedList = handleLists(listWithTextOutside, nil, defaultOpts)
	assert.Equal(t, []string{"ul", "li-header", "li-text"}, iterNodeValues(processedList))

	// Simple list
	listSimple := etree.FromString(`<ul>   <li>text</li></ul>`)
	processedList = handleLists(listSimple, nil, defaultOpts)
	assert.Len(t, dom.Children(processedList), 1)

	// List item with tail
	listItemWithTail := etree.FromString(`<ul><li>text</li>tail</ul>`)
	processedList = handleLists(listItemWithTail, nil, defaultOpts)
	children := dom.Children(processedList)
	assert.Len(t, children, 1)
	assert.Equal(t, "text tail", dom.TextContent(children[0]))

	// List item with child and tail #1
	listItemWithChildAndTail := etree.FromString(`<ul><li><p>text</p></li>tail</ul>`)
	processedList = handleLists(listItemWithChildAndTail, nil, defaultOpts)
	children = dom.Children(processedList)
	assert.Len(t, children, 1)

	firstItem := children[0]
	assert.Empty(t, etree.Tail(firstItem))
	assert.Equal(t, "tail", etree.Tail(dom.Children(firstItem)[0]))

	// List item with child and tail #2
	listItemWithChildAndTail = etree.FromString(`<ul><li><p>text</p>tail1</li>tail</ul>`)
	processedList = handleLists(listItemWithChildAndTail, nil, defaultOpts)
	children = dom.Children(processedList)
	assert.Len(t, children, 1)

	firstItem = children[0]
	assert.Empty(t, etree.Tail(firstItem))
	assert.Equal(t, "tail1 tail", etree.Tail(dom.Children(firstItem)[0]))

	// List item with child and tail #3
	listItemWithChildAndTail = etree.FromString("<ul><li><p>text</p>\n</li>tail</ul>")
	processedList = handleLists(listItemWithChildAndTail, nil, defaultOpts)
	children = dom.Children(processedList)
	assert.Len(t, children, 1)

	firstItem = children[0]
	assert.Empty(t, etree.Tail(firstItem))
	assert.Equal(t, "tail", etree.Tail(dom.Children(firstItem)[0])) // TODO: TIFU

	// List item with tail and nested list
	listItemWithTailAndNestedList := etree.FromString(`` +
		`<ul>` +
		`<li><ul><li>text</li></ul></li>` +
		`tail` +
		`</ul>`)
	processedList = handleLists(listItemWithTailAndNestedList, nil, defaultOpts)
	assert.Equal(t, "tail", etree.Tail(dom.QuerySelector(processedList, "li ul")))
}

func Test_CodeBlocks(t *testing.T) {
	var opts Options
	var result *ExtractResult
	var htmlInput string
	var htmlOutput string
	var expected string

	// Highlight js
	htmlInput = `` +
		`<div class="s-prose js-post-body" itemprop="text">` +
		`<p>Code:</p>` +
		`<pre class="lang-sql s-code-block"><code class="hljs language-sql">code\n` +
		`<span class="hljs-keyword">highlighted</span> more <span class="hljs-keyword">code</span>` +
		`</code></pre>` +
		`</div>`

	opts = Options{Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlInput), opts)

	htmlOutput = etree.ToString(result.ContentNode)
	assert.Contains(t, htmlOutput, `<code>code\nhighlighted more code</code>`)
	assert.NotContains(t, htmlOutput, `<q>`)

	// Github
	htmlInput = `` +
		`<div class="highlight highlight-source-shell notranslate position-relative overflow-auto" dir="auto"><pre>$ pip install PyGithub</pre><div class="zeroclipboard-container position-absolute right-0 top-0">` +
		`<clipboard-copy aria-label="Copy" class="ClipboardButton btn js-clipboard-copy m-2 p-0 tooltipped-no-delay" data-copy-feedback="Copied!" data-tooltip-direction="w" value="$ pip install PyGithub" tabindex="0" role="button" style="display: inherit;">` +
		`<svg aria-hidden="true" height="16" viewBox="0 0 16 16" version="1.1" width="16" data-view-component="true" class="octicon octicon-copy js-clipboard-copy-icon m-2">` +
		`<path d="M0 6.75C0 5.784.784 5 1.75 5h1.5a.75.75 0 0 1 0 1.5h-1.5a.25.25 0 0 0-.25.25v7.5c0 .138.112.25.25.25h7.5a.25.25 0 0 0 .25-.25v-1.5a.75.75 0 0 1 1.5 0v1.5A1.75 1.75 0 0 1 9.25 16h-7.5A1.75 1.75 0 0 1 0 14.25Z"></path><path d="M5 1.75C5 .784 5.784 0 6.75 0h7.5C15.216 0 16 .784 16 1.75v7.5A1.75 1.75 0 0 1 14.25 11h-7.5A1.75 1.75 0 0 1 5 9.25Zm1.75-.25a.25.25 0 0 0-.25.25v7.5c0 .138.112.25.25.25h7.5a.25.25 0 0 0 .25-.25v-7.5a.25.25 0 0 0-.25-.25Z"></path>` +
		`</svg>` +
		`<svg aria-hidden="true" height="16" viewBox="0 0 16 16" version="1.1" width="16" data-view-component="true" class="octicon octicon-check js-clipboard-check-icon color-fg-success d-none m-2">` +
		`<path d="M13.78 4.22a.75.75 0 0 1 0 1.06l-7.25 7.25a.75.75 0 0 1-1.06 0L2.22 9.28a.751.751 0 0 1 .018-1.042.751.751 0 0 1 1.042-.018L6 10.94l6.72-6.72a.75.75 0 0 1 1.06 0Z"></path>` +
		`</svg>` +
		`</clipboard-copy>` +
		`</div></div>`

	opts = Options{Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlInput), opts)

	htmlOutput = etree.ToString(result.ContentNode)
	assert.Contains(t, htmlOutput, `<code>$ pip install PyGithub</code>`)
	assert.NotContains(t, htmlOutput, `<q>`)

	// Inline code
	htmlInput = `<div><p>paragraph</p><p>here is <code>some</code> code</p></div>`

	opts = Options{Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlInput), opts)

	htmlOutput = etree.ToString(result.ContentNode)
	assert.Contains(t, htmlOutput, `<code>some</code>`)
	assert.NotContains(t, htmlOutput, `<q>`)

	// W3 Schools
	htmlInput = `` +
		`<div class="w3-example"><h3>Example</h3>` +
		`<p>Create a class named Person, use the __init__() function to assign values ` +
		`for name and age:</p>` +
		`<div class="w3-code notranslate pythonHigh"><span class="pythoncolor" style="color:black"><span class="pythonnumbercolor" style="color:red">` +
		`</span>  <span class="pythonkeywordcolor" style="color:mediumblue">class</span> Person:<br>&nbsp; <span class="pythonkeywordcolor" style="color:mediumblue">def</span> __init__(self, name, age):<br>&nbsp;&nbsp;&nbsp; <span class="pythonnumbercolor" style="color:red">` +
		`</span>  self.name = name<br>&nbsp;&nbsp;&nbsp; self.age = age<br><br>p1 = Person(<span class="pythonstringcolor" style="color:brown">"John"</span>, <span class="pythonnumbercolor" style="color:red">` +
		`</span>  <span class="pythonnumbercolor" style="color:red">36</span>)<br><span class="pythonnumbercolor" style="color:red">` +
		`</span>  <br><span class="pythonkeywordcolor" style="color:mediumblue">print</span>(p1.name)<br><span class="pythonkeywordcolor" style="color:mediumblue">print</span>(p1.age) </span></div>` +
		`</div>`

	opts = Options{Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlInput), opts)

	htmlOutput = trim(dom.OuterHTML(result.ContentNode))
	expected = `` +
		`<code> ` +
		`class Person:<br/> ` +
		`def __init__(self, name, age):<br/> ` +
		`self.name = name<br/> ` +
		`self.age = age<br/>` +
		`<br/>p1 = Person(&#34;John&#34;, 36)<br/> ` +
		`<br/>print(p1.name)<br/>print(p1.age) ` +
		`</code>`
	assert.Contains(t, htmlOutput, expected)
	assert.NotContains(t, htmlOutput, `<q>`)

	// Pip
	htmlInput = `
	<div>
		<p>Code:</p>
		<pre lang="python3">
			<span class="kn">import</span>
			<span class="nn">openai</span>
			<span class="kn">from</span>
			<span class="nn">openai_function_call</span>
			<span class="kn">import</span>
			<span class="n">openai_function</span>
		</pre>
	</div>`

	opts = Options{Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlInput), opts)

	htmlOutput = trim(dom.OuterHTML(result.ContentNode))
	expected = `<code> import openai from openai_function_call import openai_function </code>`
	assert.Contains(t, htmlOutput, expected)
	assert.NotContains(t, htmlOutput, `<q>`)

	// Medium JS
	htmlInput = `
	<div>
		<p>Code:</p>
		<pre class="lw lx ly lz ma nq nr ns bo nt ba bj">
			<span id="fe48" class="nu mo ev nr b bf nv nw l nx ny" data-selectable-paragraph="">
				<span class="hljs-keyword">import</span> openai_function<br><br>
				<span class="hljs-meta">@openai_function</span>
			</span>
		</pre>
	</div>`

	opts = Options{Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlInput), opts)

	htmlOutput = trim(dom.OuterHTML(result.ContentNode))
	expected = `<code> import openai_function<br/><br/> @openai_function </code>`
	assert.Contains(t, htmlOutput, expected)
	assert.NotContains(t, htmlOutput, `<q>`)

	// Medium SSR
	htmlInput = `
	<div>
		<p>Code:</p>
		<pre class="lw lx ly lz ma nq nr ns bo nt ba bj">
			<span id="fe48" class="nu mo ev nr b bf nv nw l nx ny">
				import openai_function<br><br>
				@openai_functiondef sum(a:int, b:int):<br/>
				&quot;&quot;&quot;Sum description adds a + b&quot;&quot;&quot;
			</span>
		</pre>
	</div>`

	opts = Options{Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlInput), opts)

	htmlOutput = trim(dom.OuterHTML(result.ContentNode))
	expected = `<code> import openai_function<br/><br/> @openai_functiondef sum(a:int, b:int):<br/> &#34;&#34;&#34;Sum description adds a + b&#34;&#34;&#34; </code>`
	assert.Contains(t, htmlOutput, expected)
	assert.NotContains(t, htmlOutput, `<q>`)

	// Code element
	htmlInput = `<div><p>Code:</p><pre><code><span>my code</span></code></pre>`

	opts = Options{Config: zeroConfig}
	result, _ = Extract(strings.NewReader(htmlInput), opts)

	htmlOutput = trim(dom.OuterHTML(result.ContentNode))
	assert.Contains(t, htmlOutput, `<code>my code</code>`)
	assert.NotContains(t, htmlOutput, `<q>`)
}

func Test_PruneSelector(t *testing.T) {
	// Helper function
	createDoc := func(strContent string) *html.Node {
		str := fmt.Sprintf(`<html><body>%s</body></html>`, strContent)
		doc, _ := dom.FastParse(strings.NewReader(str))
		return doc
	}

	// Variable helper
	var result *ExtractResult
	opts := Options{
		Config:         zeroConfig,
		EnableFallback: true,
	}

	// Example HTML
	p := `<p>abc</p>`
	h1 := `<h1>ABC</h1>`
	h2 := `<h2>42</h2>`
	doc1 := createDoc(strings.Repeat(p, 50))
	doc2 := createDoc(h1 + strings.Repeat(p, 50))
	doc3 := createDoc(h1 + h2 + strings.Repeat(p, 50))

	// Sanity check
	result, _ = ExtractDocument(doc1, opts)
	assert.NotEmpty(t, dom.OuterHTML(result.ContentNode))

	result, _ = ExtractDocument(doc2, opts)
	assert.NotEmpty(t, dom.OuterHTML(result.ContentNode))

	result, _ = ExtractDocument(doc3, opts)
	assert.NotEmpty(t, dom.OuterHTML(result.ContentNode))

	// With prune selector
	opts.PruneSelector = "p"
	result, _ = ExtractDocument(doc1, opts)
	assert.Equal(t, "", result.ContentText)

	opts.PruneSelector = "p"
	result, _ = ExtractDocument(doc2, opts)
	assert.Equal(t, "ABC", result.ContentText)

	opts.PruneSelector = "p, h1"
	result, _ = ExtractDocument(doc2, opts)
	assert.Equal(t, "", result.ContentText)

	opts.PruneSelector = "p, h1"
	result, _ = ExtractDocument(doc3, opts)
	assert.Equal(t, "42", result.ContentText)
}

func Test_MixedContentExtraction(t *testing.T) {
	htmlContent := `<html><body><p>Text here</p><img src="img.jpg"/><video src="video.mp4"/></body></html>`
	result, _ := Extract(strings.NewReader(htmlContent), zeroOpts)
	assert.Equal(t, "Text here", result.ContentText)
}

func Test_NonStdHtmlEntities(t *testing.T) {
	htmlContent := `<html><body><p>Text &customentity; more text</p></body></html>`
	result, _ := Extract(strings.NewReader(htmlContent), zeroOpts)
	assert.Equal(t, "Text &customentity; more text", result.ContentText)
}

func Test_LargeDocPerformance(t *testing.T) {
	htmlContent := `<html><body>` + strings.Repeat(`<p>Sample text</p>`, 1000) + `</body></html>`
	start := time.Now()
	Extract(strings.NewReader(htmlContent), zeroOpts)
	assert.LessOrEqual(t, time.Since(start), 5*time.Second)
}
