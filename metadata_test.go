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
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/go-shiori/dom"
	"github.com/markusmobius/go-htmldate"
	"github.com/markusmobius/go-trafilatura/v2/internal/selector"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/html"
)

func Test_Python220_Metadata(test *testing.T) {
	runPython220Assertions(test, "test-files/python-2.2.0-metadata.json")
}

func Test_Python220_Extraction(test *testing.T) {
	runPython220Assertions(test, "test-files/python-2.2.0-extraction.json")
}

func Test_Python220_CodeAndFAQ(test *testing.T) {
	runPython220Assertions(test, "test-files/python-2.2.0-structures.json")
}

func runPython220Assertions(test *testing.T, fixture string) {
	var suite struct {
		Commit    string
		Inventory []struct {
			Test       string
			Line       int
			Assertions int
			Exclusion  string
			GoTests    []string `json:"go_tests"`
			Native     []struct {
				Line           int
				Source, Reason string
				GoTests        []string `json:"go_tests"`
			}
		}
		Operations []struct {
			Test      string
			Operation string
			Arguments []any
			Keywords  map[string]any
			Initial   map[string]any
		}
		Assertions []struct {
			Test       string
			Line       int
			Source     string
			Expression any
		}
		Unmapped []struct {
			Test      string
			Line      int
			Assertion string
		}
	}
	data, err := os.ReadFile(fixture)
	if !assert.NoError(test, err) || !assert.NoError(test, json.Unmarshal(data, &suite)) {
		return
	}
	assert.Equal(test, "c1bc9531a2a978326112ca9987e1382745116136", suite.Commit)
	results := make(map[int]any)
	var evaluate func(*testing.T, any) any
	stringsOf := func(value any) []string {
		var result []string
		if values, ok := value.([]any); ok {
			for _, item := range values {
				result = append(result, fmt.Sprint(item))
			}
		}
		return result
	}
	stringOf := func(value any) string {
		if value == nil {
			return ""
		}
		if node, ok := value.(*html.Node); ok {
			return dom.TextContent(node)
		}
		return fmt.Sprint(value)
	}
	truthOf := func(value any) bool {
		switch value := value.(type) {
		case nil:
			return false
		case bool:
			return value
		case string:
			return value != ""
		case float64:
			return value != 0
		case []any:
			return len(value) != 0
		case map[string]any:
			return len(value) != 0
		default:
			return true
		}
	}
	textOf := func(test *testing.T, value any) string {
		test.Helper()
		switch value.(type) {
		case string, *html.Node:
			return stringOf(value)
		default:
			test.Fatalf("Upstream expects a string result; Go returned %T (%v)", value, value)
		}
		return ""
	}
	nullable := func(value string) any {
		if value == "" {
			return nil
		}
		return value
	}
	metadataValue := func(metadata Metadata) any {
		encoded, err := json.Marshal(metadata)
		assert.NoError(test, err)
		var fields map[string]any
		assert.NoError(test, json.Unmarshal(encoded, &fields))
		result := make(map[string]any)
		for field, value := range fields {
			if text, ok := value.(string); ok && text == "" {
				value = nil
			}
			result[strings.ToLower(field)] = value
		}
		result["date"] = nil
		if !metadata.Date.IsZero() {
			result["date"] = metadata.Date.Format("2006-01-02")
		}
		return result
	}
	evaluate = func(test *testing.T, expression any) any {
		if values, ok := expression.([]any); ok {
			result := make([]any, len(values))
			for index, value := range values {
				result[index] = evaluate(test, value)
			}
			return result
		}
		object, ok := expression.(map[string]any)
		if !ok {
			return expression
		}
		if literal, ok := object["literal"]; ok {
			return literal
		}
		if identity, ok := object["result"].(float64); ok {
			index := int(identity)
			if index < 0 || index >= len(suite.Operations) {
				test.Fatalf("Invalid upstream operation index: %v", identity)
			}
			if result, exists := results[index]; exists {
				return result
			}
			request := suite.Operations[index]
			arguments := evaluate(test, request.Arguments).([]any)
			var result any
			var initial Metadata
			if base, exists := request.Initial["base"]; exists {
				fields, _ := evaluate(test, base).(map[string]any)
				combined := make(map[string]any)
				for name, value := range fields {
					if name != "date" {
						combined[name] = value
					}
				}
				for name, value := range request.Initial["fields"].(map[string]any) {
					combined[name] = evaluate(test, value)
				}
				encoded, err := json.Marshal(combined)
				assert.NoError(test, err)
				assert.NoError(test, json.Unmarshal(encoded, &initial))
			}
			switch request.Operation {
			case "extract", "extract_dom":
				opts := Options{EnableFallback: request.Keywords["fast"] != true, ExcludeComments: request.Keywords["include_comments"] == false, ExcludeTables: request.Keywords["include_tables"] == false, IncludeLinks: request.Keywords["include_links"] == true, IncludeImages: request.Keywords["include_images"] == true}
				for key := range request.Keywords {
					switch key {
					case "config", "fast", "favor_precision", "favor_recall", "include_comments", "include_formatting", "include_images", "include_links", "include_tables", "output_format", "target_language", "with_metadata", "deduplicate", "url":
					default:
						test.Fatalf("No Go option adapter for %q", key)
					}
				}
				opts.Deduplicate = request.Keywords["deduplicate"] == true
				opts.HtmlDateMode = Extensive
				opts.TargetLanguage = stringOf(request.Keywords["target_language"])
				if address, ok := request.Keywords["url"].(string); ok {
					opts.OriginalURL, err = url.Parse(address)
					assert.NoError(test, err)
				}
				if request.Keywords["favor_recall"] == true {
					opts.Focus = FavorRecall
				}
				if request.Keywords["favor_precision"] == true {
					opts.Focus = FavorPrecision
				}
				if config, ok := request.Keywords["config"].(map[string]any); ok {
					opts.Config = DefaultConfig()
					if extensive, exists := config["extensive_date_search"]; exists && strings.EqualFold(stringOf(extensive), "off") {
						opts.HtmlDateMode = Fast
					}
					for key, target := range map[string]*int{"min_output_size": &opts.Config.MinOutputSize, "min_extracted_size": &opts.Config.MinExtractedSize, "min_output_comm_size": &opts.Config.MinOutputCommentSize, "min_extracted_comm_size": &opts.Config.MinExtractedCommentSize, "min_duplicate_check_size": &opts.Config.MinDuplicateCheckSize, "max_repetitions": &opts.Config.MaxDuplicateCount} {
						if value, exists := config[key]; exists {
							parsed, err := strconv.Atoi(stringOf(value))
							if !assert.NoError(test, err) {
								test.FailNow()
							}
							*target = parsed
						}
					}
				}
				extracted, err := Extract(strings.NewReader(stringOf(arguments[0])), opts)
				if err == nil {
					result = strings.TrimSpace(extracted.ContentText + "\n" + extracted.CommentsText)
					if request.Operation == "extract_dom" && (request.Keywords["output_format"] == "xml" || request.Keywords["output_format"] == "markdown" || request.Keywords["include_formatting"] == true) {
						result = extracted.ContentNode
					}
				}
			case "is_image_file":
				result = isImageFile(stringOf(arguments[0]))
			case "handle_image":
				var node *html.Node
				if arguments[0] != nil {
					node = dom.QuerySelector(docFromStr(stringOf(arguments[0])), "img")
				}
				if image := handleImage(node); image != nil {
					result = image
				}
			case "handle_textelem":
				if node := handleTextElem(python220Element(test, stringOf(arguments[0])), nil, nil, defaultOpts); node != nil {
					result = node
				}
			case "load_fixture":
				data, err := os.ReadFile(filepath.Join("test-files", "mock", stringOf(arguments[0])))
				if !assert.NoError(test, err) {
					test.FailNow()
				}
				result = string(data)
			case "load_mock_page":
				address := stringOf(arguments[0])
				filename, exists := rwMockFiles[address]
				if !exists {
					test.Fatalf("Upstream saved page is not mapped: %s", address)
				}
				data, err := os.ReadFile(filepath.Join("test-files", "mock", filename))
				if !assert.NoError(test, err) {
					test.FailNow()
				}
				originalURL, err := url.Parse(address)
				if !assert.NoError(test, err) {
					test.FailNow()
				}
				extracted, err := Extract(strings.NewReader(string(data)), Options{OriginalURL: originalURL, EnableFallback: true, IncludeLinks: request.Keywords["links"] == true, TargetLanguage: stringOf(request.Keywords["langcheck"])})
				if err == nil {
					result = strings.TrimSpace(extracted.ContentText + "\n" + extracted.CommentsText)
					if request.Keywords["xml_flag"] == true {
						result = dom.OuterHTML(extracted.ContentNode)
					}
				}
			case "extract_metadata":
				opts := Options{HtmlDateMode: Extensive}
				if extensive, exists := request.Keywords["extensive"].(bool); exists && !extensive {
					opts.HtmlDateMode = Fast
				}
				if len(arguments) > 1 && arguments[1] != nil {
					opts.OriginalURL, err = url.Parse(stringOf(arguments[1]))
					assert.NoError(test, err)
				}
				if address, ok := request.Keywords["default_url"].(string); ok {
					opts.OriginalURL, err = url.Parse(address)
					assert.NoError(test, err)
				}
				opts.BlacklistedAuthors = stringsOf(request.Keywords["author_blacklist"])
				if config, ok := request.Keywords["date_config"].(map[string]any); ok {
					opts.HtmlDateOptions = &htmldate.Options{UseOriginalDate: config["original_date"] == true, SkipExtensiveSearch: config["extensive_search"] != true}
				}
				var previous htmldate.Options
				if opts.HtmlDateOptions != nil {
					previous = *opts.HtmlDateOptions
				}
				var document *html.Node
				if input := stringOf(arguments[0]); strings.TrimSpace(input) != "" {
					document = docFromStr(input)
				}
				result = metadataValue(extractMetadata(document, opts))
				if opts.HtmlDateOptions != nil {
					assert.Equal(test, previous, *opts.HtmlDateOptions, "Python metadata_tests.py:406: date options must not be mutated")
				}
			case "extract_meta_json", "extract_json", "process_parent", "extract_json_parse_error", "extract_json_author":
				switch request.Operation {
				case "extract_meta_json":
					result = metadataValue(extractJsonLd(Options{}, docFromStr(stringOf(arguments[0])), initial))
				case "extract_json":
					result = metadataValue(extractJSONMetadata(arguments[0], initial))
				case "process_parent":
					result = metadataValue(processJSONMetadata(arguments[0], initial))
				case "extract_json_parse_error":
					result = metadataValue(recoverJSONMetadata(stringOf(arguments[0]), initial))
				case "extract_json_author":
					pattern := rxJSONAuthor
					if strings.Contains(stringOf(arguments[1]), "[Pp]erson") {
						pattern = rxJSONPerson
					}
					result = nullable(extractJSONAuthors(stringOf(arguments[0]), pattern))
				}
			case "extract_title":
				result = nullable(extractDomTitle(docFromStr(stringOf(arguments[0]))))
			case "extract_url":
				result = nullable(extractDomURL(docFromStr(stringOf(arguments[0]))))
			case "extract_metainfo":
				attribute := "class"
				if strings.Contains(fmt.Sprint(arguments[1]), "@type") {
					attribute = "type"
				}
				rule := func(node *html.Node) bool { return dom.TagName(node) == "p" && dom.HasAttribute(node, attribute) }
				result = nullable(extractDomMetaSelectors(docFromStr(stringOf(arguments[0])), 200, []selector.Rule{rule}))
			case "normalize_tags":
				result = normalizeTags(stringOf(arguments[0]))
			case "normalize_json":
				result = normalizeJSONText(stringOf(arguments[0]))
			case "normalize_authors":
				result = nullable(normalizeAuthors(stringOf(arguments[0]), stringOf(arguments[1])))
			case "check_authors":
				result = nullable(removeBlacklistedAuthors(stringOf(arguments[0]), Options{BlacklistedAuthors: stringsOf(arguments[1])}))
			default:
				test.Fatalf("No Go adapter for upstream operation %q", request.Operation)
			}
			results[index] = result
			return result
		}
		name, ok := object["op"].(string)
		if !ok {
			result := make(map[string]any)
			for key, value := range object {
				result[key] = evaluate(test, value)
			}
			return result
		}
		if name == "And" || name == "Or" {
			for _, argument := range object["args"].([]any) {
				value := truthOf(evaluate(test, argument))
				if name == "And" && !value {
					return false
				}
				if name == "Or" && value {
					return true
				}
			}
			return name == "And"
		}
		arguments := evaluate(test, object["args"]).([]any)
		switch name {
		case "field":
			if fields, ok := arguments[0].(map[string]any); ok {
				return fields[stringOf(arguments[1])]
			}
			if node, ok := arguments[0].(*html.Node); ok && arguments[1] == "attrib" {
				attributes := make(map[string]any)
				for _, attribute := range node.Attr {
					attributes[attribute.Key] = attribute.Val
				}
				return attributes
			}
			if values, ok := arguments[0].([]any); ok {
				index := int(arguments[1].(float64))
				if index < 0 || index >= len(values) {
					test.Fatalf("Upstream indexed element %d but Go returned %v", index, values)
				}
				return values[index]
			}
			test.Fatalf("Cannot project %v from %T", arguments[1], arguments[0])
		case "Eq", "Is":
			return reflect.DeepEqual(arguments[0], arguments[1])
		case "NotEq", "IsNot":
			return !reflect.DeepEqual(arguments[0], arguments[1])
		case "Gt", "GtE", "Lt", "LtE":
			left, leftOK := arguments[0].(float64)
			right, rightOK := arguments[1].(float64)
			if !leftOK || !rightOK {
				test.Fatalf("Unsupported upstream ordered comparison: %T %s %T", arguments[0], name, arguments[1])
			}
			switch name {
			case "Gt":
				return left > right
			case "GtE":
				return left >= right
			case "Lt":
				return left < right
			case "LtE":
				return left <= right
			}
		case "In", "NotIn":
			contains := false
			switch container := arguments[1].(type) {
			case string:
				contains = strings.Contains(container, stringOf(arguments[0]))
			case *html.Node:
				expected := stringOf(arguments[0])
				switch {
				case expected == "quote" || expected == "<quote>":
					contains = dom.QuerySelector(container, "blockquote") != nil
				case expected == "lb":
					contains = dom.QuerySelector(container, "br") != nil
				case expected == "<p>":
					contains = dom.QuerySelector(container, "p") != nil
				case expected == "<main/>":
					contains = len(dom.Children(container)) == 0 && dom.TextContent(container) == ""
				case expected == `rend="#b"` || expected == `rend="#i"` || expected == `rend="#t"` || expected == `rend="#u"` || expected == "<del>":
					query := map[string]string{`rend="#b"`: "b, strong", `rend="#i"`: "i, em", `rend="#t"`: "tt, kbd", `rend="#u"`: "u", "<del>": "del, s, strike"}[expected]
					contains = dom.QuerySelector(container, query) != nil
				case strings.HasPrefix(expected, "<"):
					fragment := python220Element(test, expected)
					contains = strings.Contains(python220CanonicalHTML(container), python220CanonicalHTML(fragment))
				case strings.HasPrefix(expected, "### "):
					for _, heading := range dom.QuerySelectorAll(container, "h3") {
						contains = contains || dom.TextContent(heading) == strings.TrimPrefix(expected, "### ")
					}
				default:
					contains = strings.Contains(dom.TextContent(container), expected)
				}
			case []any:
				for _, value := range container {
					contains = contains || reflect.DeepEqual(value, arguments[0])
				}
			case map[string]any:
				_, contains = container[stringOf(arguments[0])]
			}
			return contains == (name == "In")
		case "Not":
			return !truthOf(arguments[0])
		case "len":
			switch value := arguments[0].(type) {
			case string:
				return float64(utf8.RuneCountInString(value))
			case []any:
				return float64(len(value))
			case map[string]any:
				return float64(len(value))
			default:
				test.Fatalf("Upstream expects a value with a length; Go returned %T (%v)", value, value)
			}
		case "startswith":
			return strings.HasPrefix(textOf(test, arguments[0]), textOf(test, arguments[1]))
		case "endswith":
			return strings.HasSuffix(textOf(test, arguments[0]), textOf(test, arguments[1]))
		case "count":
			return float64(strings.Count(textOf(test, arguments[0]), textOf(test, arguments[1])))
		case "str":
			if arguments[0] == nil {
				return "None"
			}
			if value, ok := arguments[0].(bool); ok {
				if value {
					return "True"
				}
				return "False"
			}
			return stringOf(arguments[0])
		case "strip":
			return strings.TrimSpace(textOf(test, arguments[0]))
		case "replace":
			return strings.ReplaceAll(textOf(test, arguments[0]), textOf(test, arguments[1]), textOf(test, arguments[2]))
		case "get":
			if node, ok := arguments[0].(*html.Node); ok {
				return nullable(dom.GetAttribute(node, stringOf(arguments[1])))
			}
			test.Fatalf("Cannot get an attribute from %T", arguments[0])
		default:
			test.Fatalf("No Go assertion adapter for %q", name)
		}
		return nil
	}
	for _, entry := range suite.Inventory {
		if len(entry.GoTests) > 0 {
			test.Logf("Native mapping: %s -> %s", entry.Test, strings.Join(entry.GoTests, ", "))
			continue
		}
		test.Run(entry.Test, func(test *testing.T) {
			if entry.Exclusion != "" {
				test.Skip(entry.Exclusion)
			}
			for _, check := range entry.Native {
				if len(check.GoTests) > 0 {
					test.Logf("Native mapping: %s:line_%d -> %s", entry.Test, check.Line, strings.Join(check.GoTests, ", "))
					continue
				}
				test.Run(fmt.Sprintf("line_%d", check.Line), func(test *testing.T) { test.Skip(check.Reason) })
			}
			for _, check := range suite.Assertions {
				if check.Test != entry.Test {
					continue
				}
				test.Run(fmt.Sprintf("line_%d", check.Line), func(test *testing.T) {
					actual := evaluate(test, check.Expression)
					if !assert.True(test, truthOf(actual), "%s:%d: %s", check.Test, check.Line, check.Source) {
						if expression, ok := check.Expression.(map[string]any); ok {
							if name := expression["op"]; name == "Eq" || name == "Is" {
								operands := evaluate(test, expression["args"]).([]any)
								test.Logf("Go value: %#v; upstream expected: %#v", operands[0], operands[1])
							}
						}
					}
				})
			}
			for index, request := range suite.Operations {
				if request.Test == entry.Test {
					if _, evaluated := results[index]; !evaluated {
						test.Run(fmt.Sprintf("operation_%d", index), func(test *testing.T) { evaluate(test, map[string]any{"result": float64(index)}) })
					}
				}
			}
		})
	}
	for _, entry := range suite.Unmapped {
		if entry.Test == "metadata_tests.py/test_date_config" && entry.Line == 406 {
			continue
		}
		test.Run(fmt.Sprintf("%s/line_%d", entry.Test, entry.Line), func(test *testing.T) {
			if entry.Test == "metadata_tests.py/test_author_blacklist" && (entry.Line == 101 || entry.Line == 102) {
				test.Skip("Go always returns metadata and has no with_metadata toggle")
			}
			test.Fatalf("Unmapped upstream assertion: %s", entry.Assertion)
		})
	}
}

func Test_Metadata(t *testing.T) {
	rawHTML := `
	<html>

	<head>
		<title>Test Title</title>
		<meta itemprop="author" content="Jenny Smith" />
		<meta property="og:url" content="https://example.org" />
		<meta itemprop="description" content="Description" />
		<meta property="og:published_time" content="2017-09-01" />
		<meta name="article:publisher" content="The Newspaper" />
		<meta property="image" content="https://example.org/example.jpg" />
	</head>

	<body>
		<p class="entry-categories">
			<a href="https://example.org/category/cat1/">Cat1</a>,
			<a href="https://example.org/category/cat2/">Cat2</a>
		</p>
		<p>
			<a href="https://creativecommons.org/licenses/by-sa/4.0/" rel="license">CC BY-SA</a>
		</p>
	</body>

	</html>`

	metadata := testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Test Title", metadata.Title)
	assert.Equal(t, "Jenny Smith", metadata.Author)
	assert.Equal(t, "https://example.org", metadata.URL)
	assert.Equal(t, "Description", metadata.Description)
	assert.Equal(t, "The Newspaper", metadata.Sitename)
	assert.Equal(t, "2017-09-01", metadata.Date.Format("2006-01-02"))
	assert.Equal(t, []string{"Cat1", "Cat2"}, metadata.Categories)
	assert.Equal(t, "CC BY-SA 4.0", metadata.License)
	assert.Equal(t, "https://example.org/example.jpg", metadata.Image)
}

func Test_Metadata_Titles(t *testing.T) {
	var rawHTML string
	isEqual := func(rawHTML string, expected any) {
		metadata := testGetMetadataFromHTML(rawHTML)
		assert.Equal(t, expected, metadata.Title)
	}

	rawHTML = `<html><body><h3 class="title">T</h3><h3 id="title"></h3></body></html>`
	isEqual(rawHTML, "")

	rawHTML = `<html><head><title>Test Title</title><meta property="og:title" content=" " /></head><body><h1>First</h1></body></html>`
	isEqual(rawHTML, "First")

	rawHTML = `<html><head><title>Test Title</title><meta name="title" content=" " /></head><body><h1>First</h1></body></html>`
	isEqual(rawHTML, "First")

	rawHTML = `<html><head><title>Test Title</title></head><body></body></html>`
	isEqual(rawHTML, "Test Title")

	rawHTML = `<html><body><h1>First</h1><h1>Second</h1></body></html>`
	isEqual(rawHTML, "First")

	rawHTML = `<html><body><h1> </h1><h1> First non-empty title </h1><h1>Second</h1></body></html>`
	isEqual(rawHTML, "First non-empty title")

	rawHTML = `<html><head><title>example.org</title></head><body><h2>Article heading</h2></body></html>`
	isEqual(rawHTML, "Article heading")

	rawHTML = `<html><body><h1>   </h1><div class="post-title">Test Title</div></body></html>`
	isEqual(rawHTML, "Test Title")

	rawHTML = `<html><body><h2 class="block-title">Main menu</h2><h1 class="article-title">Test Title</h1></body></html>`
	isEqual(rawHTML, "Test Title")

	rawHTML = `<html><body><h2>First</h2><h1>Second</h1></body></html>`
	isEqual(rawHTML, "Second")

	rawHTML = `<html><body><h2>First</h2><h2>Second</h2></body></html>`
	isEqual(rawHTML, "First")

	rawHTML = `<html><body><title></title></body></html>`
	isEqual(rawHTML, "")

	rawHTML = `<html><head><title> - Home</title></head><body/></html>`
	isEqual(rawHTML, "- Home")

	rawHTML = `<html><head><title>My Title » My Website</title></head><body/></html>`
	isEqual(rawHTML, "My Title") // TODO: and metadata.sitename == "My Website"

	// Try from file
	metadata := testGetMetadataFromFile("simple/metadata-title.html")
	assert.Equal(t, "Semantic satiation", metadata.Title)
}

func Test_Metadata_normalizeAuthors(t *testing.T) {
	// Alias for shorter test
	na := normalizeAuthors
	isEqual := assert.Equal

	isEqual(t, "Abc", na("", "abc"))
	isEqual(t, "Steve Steve", na("", "Steve Steve 123"))
	isEqual(t, "Steve Steve", na("", "By Steve Steve"))
	isEqual(t, "Seán Federico O'Murchú", na("", "Seán Federico O'Murchú"))
	isEqual(t, "John Doe", na("", "John Doe"))
	isEqual(t, "Alice; Bob; John Doe", na("Alice; Bob", "John Doe"))
	isEqual(t, "Alice; Bob", na("Alice; Bob", "john.doe@example.com"))
	isEqual(t, "Étienne", na("", "\u00e9tienne"))
	isEqual(t, "Étienne", na("", "&#233;tienne"))
	isEqual(t, "Alice; Bob", na("", "Alice &amp; Bob"))
	isEqual(t, "John Doe", na("", "<b>John Doe</b>"))
	isEqual(t, "John Doe", na("", "John 😊 Doe"))
	isEqual(t, "John Doe", na("", "words by John Doe"))
	isEqual(t, "John Doe", na("", "John Doe123"))
	isEqual(t, "John Doe", na("", "John_Doe"))
	isEqual(t, "John Doe", na("", "John Doe* "))
	isEqual(t, "John Doe", na("", "John Doe of John Doe"))
	isEqual(t, "John Doe", na("", "John Doe — John Doe"))
	isEqual(t, "John  Doe", na("", `John "The King" Doe`))
}

func Test_Metadata_Authors(t *testing.T) {
	var opts Options
	var rawHTML string
	var metadata Metadata

	isEqual := func(rawHTML string, expected string) {
		metadata := testGetMetadataFromHTML(rawHTML)
		assert.Equal(t, expected, metadata.Author)
	}

	headHTML := func(s string) string {
		return `<html><head>` + s + `</head><body></body></html>`
	}

	bodyHTML := func(s string) string {
		return `<html><body>` + s + `</body></html>`
	}

	// Extraction from head
	rawHTML = headHTML(`<meta itemprop="author" content="Jenny Smith"/>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = headHTML(`<meta itemprop="author" content="Jenny Smith"/><meta itemprop="author" content="John Smith"/>`)
	isEqual(rawHTML, "Jenny Smith; John Smith")

	rawHTML = headHTML(`<meta itemprop="author" content="Jenny Smith und John Smith"/>`)
	isEqual(rawHTML, "Jenny Smith; John Smith")

	rawHTML = headHTML(`<meta name="author" content="Jenny Smith"/><meta name="author" content="John Smith"/>`)
	isEqual(rawHTML, "Jenny Smith; John Smith")

	rawHTML = headHTML(`<meta name="author" content="Jenny Smith and John Smith"/>`)
	isEqual(rawHTML, "Jenny Smith; John Smith")

	rawHTML = headHTML(`<meta name="author" content="Jenny Smith"/>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = headHTML(`<meta name="author" content="Hank O&#39;Hop"/>`)
	isEqual(rawHTML, "Hank O'Hop")

	rawHTML = headHTML(`<meta name="author" content="Jenny Smith ❤️"/>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = headHTML(`<meta name="citation_author" content="Jenny Smith and John Smith"/>`)
	isEqual(rawHTML, "Jenny Smith; John Smith")

	rawHTML = headHTML(`<meta property="author" content="Jenny Smith"/><meta property="author" content="John Smith"/>`)
	isEqual(rawHTML, "Jenny Smith; John Smith")

	rawHTML = headHTML(`<meta itemprop="author" content="Jenny Smith and John Smith"/>`)
	isEqual(rawHTML, "Jenny Smith; John Smith")

	rawHTML = headHTML(`<meta name="article:author" content="Jenny Smith"/>`)
	isEqual(rawHTML, "Jenny Smith")

	// Extraction from body
	rawHTML = bodyHTML(`<a href="" rel="author">Jenny Smith</a>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<a href="" rel="author">Jenny "The Author" Smith</a>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<span class="author">Jenny Smith</span>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<h4 class="author">Jenny Smith</h4>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<h4 class="author">Jenny Smith — Trafilatura</h4>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<span class="wrapper--detail__writer">Jenny Smith</span>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<span id="author-name">Jenny Smith</span>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<figure data-component="Figure"><div class="author">Jenny Smith</div></figure>`)
	isEqual(rawHTML, "")

	rawHTML = bodyHTML(`<div class="sidebar"><div class="author">Jenny Smith</div></figure>`)
	isEqual(rawHTML, "")

	rawHTML = bodyHTML(`<div class="quote"><p>My quote here</p><p class="quote-author"><span>—</span> Jenny Smith</p></div>`)
	isEqual(rawHTML, "")

	rawHTML = bodyHTML(`<span class="author">Jenny Smith and John Smith</span>`)
	isEqual(rawHTML, "Jenny Smith; John Smith")

	rawHTML = bodyHTML(`<a class="author">Jenny Smith</a>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<a class="author">Jenny Smith <div class="title">Editor</div></a>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<a class="author">Jenny Smith from Trafilatura</a>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<meta itemprop="author" content="Fake Author"/><a class="author">Jenny Smith from Trafilatura</a>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<a class="username">Jenny Smith</a>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<div class="submitted-by"><a>Jenny Smith</a></div>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<div class="byline-content"><div class="byline"><a>Jenny Smith</a></div><time>July 12, 2021 08:05</time></div>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<h3 itemprop="author">Jenny Smith</h3>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<div class="article-meta article-meta-byline article-meta-with-photo article-meta-author-and-reviewer" itemprop="author" itemscope="" itemtype="http://schema.org/Person"><span class="article-meta-photo-wrap"><img src="" alt="Jenny Smith" itemprop="image" class="article-meta-photo"></span><span class="article-meta-contents"><span class="article-meta-author">By <a href="" itemprop="url"><span itemprop="name">Jenny Smith</span></a></span><span class="article-meta-date">May 18 2022</span><span class="article-meta-reviewer">Reviewed by <a href="">Robert Smith</a></span></span></div>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<div data-component="Byline">Jenny Smith</div>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<span id="author">Jenny Smith</span>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<span id="author">Jenny Smith – The Moon</span>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<span id="author">Jenny_Smith</span>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<span itemprop="author name">Shannon Deery, Mitch Clarke, Susie O’Brien, Laura Placella, Kara Irving, Jordy Atkinson, Suzan Delibasic</span>`)
	isEqual(rawHTML, "Shannon Deery; Mitch Clarke; Susie O’Brien; Laura Placella; Kara Irving; Jordy Atkinson; Suzan Delibasic")

	rawHTML = bodyHTML(`<address class="author">Jenny Smith</address>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<author>Jenny Smith</author>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<div class="author"><span class="profile__name"> Jenny Smith </span> <a href="https://twitter.com/jenny_smith" class="profile__social" target="_blank"> @jenny_smith </a> <span class="profile__extra lg:hidden"> 11:57AM </span> </div>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<p class="author-section byline-plain">By <a class="author" rel="nofollow">Jenny Smith For Daily Mail Australia</a></p>`)
	isEqual(rawHTML, "Jenny Smith")

	rawHTML = bodyHTML(`<div class="o-Attribution__a-Author"><span class="o-Attribution__a-Author--Label">By:</span><span class="o-Attribution__a-Author--Prefix"><span class="o-Attribution__a-Name"><a href="//web.archive.org/web/20210707074846/https://www.discovery.com/profiles/ian-shive">Ian Shive</a></span></span></div>`)
	isEqual(rawHTML, "Ian Shive")

	rawHTML = bodyHTML(`<div class="ArticlePage-authors"><div class="ArticlePage-authorName" itemprop="name"><span class="ArticlePage-authorBy">By&nbsp;</span><a aria-label="Ben Coxworth" href="https://newatlas.com/author/ben-coxworth/"><span>Ben Coxworth</span></a></div></div>`)
	isEqual(rawHTML, "Ben Coxworth")

	rawHTML = bodyHTML(`<div><strong><a class="d1dba0c3091a3c30ebd6" data-testid="AuthorURL" href="/by/p535y1">AUTHOR NAME</a></strong></div`)
	isEqual(rawHTML, "AUTHOR NAME")

	rawHTML = `<html><head><meta data-rh="true" property="og:author" content="By &lt;a href=&quot;/profiles/amir-vera&quot;&gt;Amir Vera&lt;/a&gt;, Seán Federico O&#x27;Murchú, &lt;a href=&quot;/profiles/tara-subramaniam&quot;&gt;Tara Subramaniam&lt;/a&gt; and Adam Renton, CNN"/></head><body>f{end}`
	isEqual(rawHTML, "Amir Vera; Seán Federico O'Murchú; Tara Subramaniam; Adam Renton; CNN")

	// Blacklist
	opts = Options{BlacklistedAuthors: []string{"Jenny Smith"}}
	rawHTML = `<html><head><meta itemprop="author" content="Jenny Smith"/></head><body></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML, opts)
	assert.Equal(t, "", metadata.Author)

	opts = Options{BlacklistedAuthors: []string{"A", "b"}}
	assert.Equal(t, "c; d", removeBlacklistedAuthors("a; B; c; d", opts))
	assert.Equal(t, "c; d", removeBlacklistedAuthors("a;B;c;d", opts))
}

func Test_Metadata_URLs(t *testing.T) {
	var rawHTML string
	expected := "https://example.org"
	isEqual := func(rawHTML string, expected string, customOpts ...Options) {
		metadata := testGetMetadataFromHTML(rawHTML, customOpts...)
		assert.Equal(t, expected, metadata.URL)
	}

	rawHTML = `<html><head><meta property="og:url" content="https://example.org"/></head><body></body></html>`
	isEqual(rawHTML, expected)

	rawHTML = `<html><head><link rel="canonical" href="https://example.org"/></head><body></body></html>`
	isEqual(rawHTML, expected)

	rawHTML = `<html><head><meta name="twitter:url" content="https://example.org"/></head><body></body></html>`
	isEqual(rawHTML, expected)

	rawHTML = `<html><head><link rel="alternate" hreflang="x-default" href="https://example.org"/></head><body></body></html>`
	isEqual(rawHTML, expected)

	// Test on partial URLs
	rawHTML = `<html><head><link rel="canonical" href="/article/medical-record"/><meta name="twitter:url" content="https://example.org"/></head><body></body></html>`
	assert.Equal(t, "https://example.org/article/medical-record", extractDomURL(docFromStr(rawHTML)))

	rawHTML = `<html><head><base href="https://example.org" target="_blank"/></head><body></body></html>`
	isEqual(rawHTML, expected)
}

func Test_Metadata_Descriptions(t *testing.T) {
	rawHTML := `<html><head><meta itemprop="description" content="Description"/></head><body></body></html>`
	metadata := testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Description", metadata.Description)

	rawHTML = `<html><head><meta property="og:description" content="&amp;#13; A Northern Territory action plan, which includes plans to support development and employment on Aboriginal land, has received an update. &amp;#13..." /></head><body></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "A Northern Territory action plan, which includes plans to support development and employment on Aboriginal land, has received an update. ...", metadata.Description)
}

func Test_Metadata_DescriptionEntities(test *testing.T) {
	testCases := []struct {
		name     string
		content  string
		expected string
	}{
		{"decimal control", "before&amp;#8;after", "beforeafter"},
		{"unterminated control", "before&amp;#8...after", "before...after"},
		{"hex control", "before&amp;#x8;after", "beforeafter"},
		{"vertical tab", "before&amp;#11;after", "beforeafter"},
		{"form feed", "before&amp;#12;after", "before after"},
		{"whitespace", "before&amp;#9;&amp;#10;&amp;#13;after", "before after"},
		{"valid entities", "price &amp;#x20AC;5 &amp;amp; tax", "price \u20ac5 & tax"},
		{"nonprinting Unicode", "a&amp;#x200d;b", "ab"},
		{"null replacement", "before&amp;#0;after", "before\ufffdafter"},
		{"noncharacter", "before&amp;#xFDD0;after", "beforeafter"},
	}
	for _, attribute := range []string{`name="description"`, `property="og:description"`, `itemprop="description"`} {
		for _, testCase := range testCases {
			test.Run(attribute+"/"+testCase.name, func(test *testing.T) {
				input := `<html><head><meta ` + attribute + ` content="` + testCase.content + `"></head><body></body></html>`
				metadata := testGetMetadataFromHTML(input)
				assert.Equal(test, testCase.expected, metadata.Description)
			})
		}
	}

	metadata := testGetMetadataFromFile("comparison/zahlenzauberin.wordpress.com.ferien.html")
	assert.Equal(test, "Dank Kabelanschluss kann ich, auch in Sachsen ,Bayern 2 hören. Da läuft gerade ein spannendes nah dran zum Thema: Freude, Falle, Frust: Der Mutterliebe zarte Sorgen Raben- versus Gluckenmütter …", metadata.Description)

	for _, attribute := range []string{`name="keywords"`, `property="article:tag"`, `property="og:article:tag"`} {
		test.Run(attribute, func(test *testing.T) {
			input := `<html><head><meta ` + attribute + ` content="zero&#x200d;width"></head><body></body></html>`
			metadata := testGetMetadataFromHTML(input)
			assert.Equal(test, []string{"zero\u200dwidth"}, metadata.Tags)
		})
	}
}

func Test_Metadata_Dates(t *testing.T) {
	var rawHTML string
	isEqual := func(rawHTML string, expected string, customOpts ...Options) {
		metadata := testGetMetadataFromHTML(rawHTML, customOpts...)
		assert.Equal(t, expected, metadata.Date.Format("2006-01-02"))
	}

	rawHTML = `<html><head><meta property="og:published_time" content="2017-09-01"/></head><body></body></html>`
	isEqual(rawHTML, "2017-09-01")

	rawHTML = `<html><head><meta property="og:url" content="https://example.org/2017/09/01/content.html"/></head><body></body></html>`
	isEqual(rawHTML, "2017-09-01")

	// Compare extensive mode
	opts := defaultOpts
	rawHTML = `<html><body><p>Veröffentlicht am 1.9.17</p></body></html>`

	opts.EnableFallback = false // fast mode
	isEqual(rawHTML, "2017-09-01", opts)

	opts.EnableFallback = true // extensive mode
	isEqual(rawHTML, "2017-09-01", opts)

}

func Test_Metadata_Categories(t *testing.T) {
	var rawHTML string
	isEqual := func(rawHTML string, expected ...string) {
		metadata := testGetMetadataFromHTML(rawHTML)
		assert.Equal(t, expected, metadata.Categories)
	}

	rawHTML = `<html><body>
		<p class="entry-categories">
			<a href="https://example.org/category/cat1/">Cat1</a>,
			<a href="https://example.org/category/cat2/">Cat2</a>
		</p></body></html>`
	isEqual(rawHTML, "Cat1", "Cat2")

	rawHTML = `<html><body>
		<div class="postmeta"><a href="https://example.org/category/cat1/">Cat1</a></div>
	</body></html>`
	isEqual(rawHTML, "Cat1")
}

func Test_Metadata_Tags(t *testing.T) {
	var rawHTML string
	isEqual := func(rawHTML string, expected ...string) {
		metadata := testGetMetadataFromHTML(rawHTML)
		assert.Equal(t, expected, metadata.Tags)
	}

	rawHTML = `<html><body>
		<p class="entry-tags">
			<a href="https://example.org/tags/tag1/">Tag1</a>,
			<a href="https://example.org/tags/tag2/">Tag2</a>
		</p></body></html>`
	isEqual(rawHTML, "Tag1", "Tag2")

	rawHTML = `<html><body>
		<p class="entry-tags">
			<a href="https://example.org/tags/tag1/">    Tag1   </a>,
			<a href="https://example.org/tags/tag2/"> 1 &amp; 2 </a>
		</p></body></html>`
	isEqual(rawHTML, "Tag1", "1 & 2")

	rawHTML = `<html><head>
		<meta name="keywords" content="sodium, salt, paracetamol, blood, pressure, high, heart, &amp;quot, intake, warning, study, &amp;quot, medicine, dissolvable, cardiovascular" />
	</head></html>`
	isEqual(rawHTML, "sodium, salt, paracetamol, blood, pressure, high, heart, intake, warning, study, medicine, dissolvable, cardiovascular")
}

func Test_Metadata_Sitename(t *testing.T) {
	var rawHTML string
	isEqual := func(rawHTML string, expected string) {
		metadata := testGetMetadataFromHTML(rawHTML)
		assert.Equal(t, expected, metadata.Sitename)
	}

	rawHTML = `<html><head><meta name="article:publisher" content="@"/></head><body/></html>`
	isEqual(rawHTML, "")

	rawHTML = `<html><head><meta name="article:publisher" content="The Newspaper"/></head><body/></html>`
	isEqual(rawHTML, "The Newspaper")

	rawHTML = `<html><head><meta property="article:publisher" content="The Newspaper"/></head><body/></html>`
	isEqual(rawHTML, "The Newspaper")

	rawHTML = `<html><head><title>sitemaps.org - Home</title></head><body/></html>`
	isEqual(rawHTML, "sitemaps.org")
}

func Test_Metadata_License(t *testing.T) {
	// From <a> rel
	rawHTML := `<html><body><p><a href="https://creativecommons.org/licenses/by-sa/4.0/" rel="license">CC BY-SA</a></p></body></html>`
	metadata := testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "CC BY-SA 4.0", metadata.License)

	rawHTML = `<html><body><p><a href="https://licenses.org/unknown" rel="license">Unknown</a></p></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Unknown", metadata.License)

	// Footer
	rawHTML = `<html><body><footer><a href="https://creativecommons.org/licenses/by-sa/4.0/">CC BY-SA</a></footer></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "CC BY-SA 4.0", metadata.License)

	// Real world footer test: netzpolitik.org
	rawHTML = `<html><body>
	<div class="footer__navigation">
		<p class="footer__licence">
			<strong>Lizenz: </strong>
			Die von uns verfassten Inhalte stehen, soweit nicht anders vermerkt, unter der Lizenz
			<a href="http://creativecommons.org/licenses/by-nc-sa/4.0/">Creative Commons BY-NC-SA 4.0.</a>
		</p>
	</div></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "CC BY-NC-SA 4.0", metadata.License)

	// This is not a license
	rawHTML = `<html><body><footer class="entry-footer">
		<span class="cat-links">Posted in <a href="https://sallysbakingaddiction.com/category/seasonal/birthday/" rel="category tag">Birthday</a></span>
	</footer></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Empty(t, metadata.License)

	// This is a license
	rawHTML = `<html><body><footer class="entry-footer">
		<span>The license is <a href="https://example.org/1">CC BY-NC</a></span>
	</footer></body></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "CC BY-NC", metadata.License)
}

func Test_Metadata_AuthorDeduplication(test *testing.T) {
	assert.Equal(test, "John Doe", normalizeAuthors("John", "John Doe"))
	assert.Equal(test, "John Doe", normalizeAuthors("John Doe", "John"))
	assert.Equal(test, "Jane Smith; John Doe", normalizeAuthors("John; Jane Smith", "John Doe"))
	assert.Equal(test, "John Doe", normalizeAuthors("", "John; John Doe"))
}

func Test_Metadata_LicenseNestedText(test *testing.T) {
	for _, input := range []string{
		`<html><body><a rel="license" href="/license"><span>CC BY-SA 4.0</span></a></body></html>`,
		`<html><body><footer><a href="/license"><span>CC BY-SA 4.0</span></a></footer></body></html>`,
	} {
		metadata := testGetMetadataFromHTML(input)
		assert.Equal(test, "CC BY-SA 4.0", metadata.License)
	}
}

func Test_Metadata_MetaImages(t *testing.T) {
	var rawHTML string
	exampleURL, _ := url.ParseRequestURI("http://example.org")
	isEqual := func(rawHTML string, expected string) {
		metadata := testGetMetadataFromHTML(rawHTML, Options{
			Config:      DefaultConfig(),
			OriginalURL: exampleURL,
		})
		assert.Equal(t, expected, metadata.Image)
	}

	// Image extraction from meta SEO tags
	rawHTML = `<html><head><meta property="image" content="https://example.org/example.jpg"></html>`
	isEqual(rawHTML, "https://example.org/example.jpg")

	rawHTML = `<html><head><meta property="og:image:url" content="example.jpg"></html>`
	isEqual(rawHTML, "example.jpg")

	rawHTML = `<html><head><meta property="og:image" content="https://example.org/example-opengraph.jpg" /><body/></html>`
	isEqual(rawHTML, "https://example.org/example-opengraph.jpg")

	rawHTML = `<html><head><meta property="twitter:image" content="https://example.org/example-twitter.jpg"></html>`
	isEqual(rawHTML, "https://example.org/example-twitter.jpg")

	rawHTML = `<html><head><meta property="twitter:image:src" content="example-twitter.jpg"></html>`
	isEqual(rawHTML, "example-twitter.jpg")

	for _, name := range []string{"image", "og:image", "twitter:image", "twitter:image:src"} {
		rawHTML = `<html><head><meta name="` + name + `" content="example.jpg"></head></html>`
		isEqual(rawHTML, "example.jpg")
	}

	rawHTML = `<html><head><meta name="image" content="other.jpg"><meta property="og:image" content="preferred.jpg"></head></html>`
	isEqual(rawHTML, "preferred.jpg")

	// Without image
	rawHTML = `<html><head><meta name="robots" content="index, follow, max-image-preview:large, max-snippet:-1, max-video-preview:-1" /></html>`
	isEqual(rawHTML, "")
}

func Test_Metadata_MetaTags(t *testing.T) {
	rawHTML := `<html>
		<head>
			<meta property="og:title" content="Open Graph Title" />
			<meta property="og:author" content="Jenny Smith" />
			<meta property="og:description" content="This is an Open Graph description" />
			<meta property="og:site_name" content="My first site" />
			<meta property="og:url" content="https://example.org/test" />
			<meta property="og:type" content="Open Graph Type" />
		</head>
		<body><a rel="license" href="https://creativecommons.org/">Creative Commons</a></body>
	</html>`
	metadata := testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Open Graph Title", metadata.Title)
	assert.Equal(t, "Jenny Smith", metadata.Author)
	assert.Equal(t, "This is an Open Graph description", metadata.Description)
	assert.Equal(t, "My first site", metadata.Sitename)
	assert.Equal(t, "https://example.org/test", metadata.URL)
	assert.Equal(t, "Creative Commons", metadata.License)
	assert.Equal(t, "Open Graph Type", metadata.PageType)

	rawHTML = `<html><head>
			<meta name="dc.title" content="Open Graph Title" />
			<meta name="dc.creator" content="Jenny Smith" />
			<meta name="dc.description" content="This is an Open Graph description" />
		</head></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Open Graph Title", metadata.Title)
	assert.Equal(t, "Jenny Smith", metadata.Author)
	assert.Equal(t, "This is an Open Graph description", metadata.Description)

	rawHTML = `<html><head>
			<meta itemprop="headline" content="Title" />
		</head></html>`
	metadata = testGetMetadataFromHTML(rawHTML)
	assert.Equal(t, "Title", metadata.Title)

	// Test error
	isEmpty := func(meta Metadata) bool {
		return meta.Title == "" &&
			meta.Author == "" &&
			meta.URL == "" &&
			meta.Hostname == "" &&
			meta.Description == "" &&
			meta.Sitename == "" &&
			meta.Date.IsZero() &&
			len(meta.Categories) == 0 &&
			len(meta.Tags) == 0
	}

	metadata = testGetMetadataFromHTML("")
	assert.True(t, isEmpty(metadata))

	metadata = testGetMetadataFromHTML("<html><title></title></html>")
	assert.True(t, isEmpty(metadata))
}

func testGetMetadataFromHTML(rawHTML string, customOpts ...Options) Metadata {
	// Parse raw html
	doc, err := html.Parse(strings.NewReader(rawHTML))
	if err != nil {
		panic(err)
	}

	if len(customOpts) > 0 {
		return extractMetadata(doc, customOpts[0])
	}

	return extractMetadata(doc, defaultOpts)
}

func testGetMetadataFromURL(url string, customOpts ...Options) Metadata {
	doc := parseMockFile(metadataMockFiles, url)
	if len(customOpts) > 0 {
		return extractMetadata(doc, customOpts[0])
	}
	return extractMetadata(doc, defaultOpts)
}

func testGetMetadataFromFile(path string) Metadata {
	// Open file
	path = filepath.Join("test-files", path)
	f, err := os.Open(path)
	if err != nil {
		log.Panic().Err(err)
	}

	// Parse HTML
	doc, err := html.Parse(f)
	if err != nil {
		log.Panic().Err(err)
	}

	return extractMetadata(doc, defaultOpts)
}
