package trafilatura

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-shiori/dom"
	"github.com/markusmobius/go-trafilatura/etree"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/html"
)

var (
	trafilaturaMockFiles = map[string]string{
		"http://exotic_tags": "exotic_tags.html",
	}

	zeroOpts = Options{
		NoFallback: true,
		Config: &Config{
			MinOutputSize:    0,
			MinExtractedSize: 0,
		},
	}

	defaultOpts = Options{
		Config: DefaultConfig(),
	}
)

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
	// Cover some edge cases with a specially crafted file
	result := extractMockFile(trafilaturaMockFiles, "http://exotic_tags")
	assert.Contains(t, result.ContentText, "Teletype text")
	assert.Contains(t, result.ContentText, "My new car is silver.")

	// Misformed HTML declaration
	htmlString := `<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01 Transitional//EN" 2012"http://www.w3.org/TR/html4/loose.dtd"><html><head></head><body><p>ABC</p></body></html>`
	result, err := Extract(strings.NewReader(htmlString), zeroOpts)
	assert.Nil(t, err)
	assert.Contains(t, result.ContentText, "ABC")

	// Quotes
	assert.Nil(t, handleQuotes(etree.Element("blockquote"), nil, zeroOpts))
	assert.Nil(t, handleTable(etree.Element("table"), nil, zeroOpts))

	// Nested <p>
	element, second := etree.Element("p"), etree.Element("p")
	etree.SetText(element, "1st part.")
	etree.SetText(second, "2nd part.")
	etree.Append(element, second)

	converted := handleParagraphs(element, map[string]struct{}{"p": {}}, nil, zeroOpts)
	assert.Equal(t, "<p>1st part. 2nd part.</p>", etree.ToString(converted))

	// Trailing line break
	etree.SubElement(element, "br")
	converted = handleParagraphs(element, map[string]struct{}{"p": {}}, nil, zeroOpts)
	assert.Equal(t, "<p>1st part. 2nd part.</p>", etree.ToString(converted))

	// Malformed lists (common error)
	lists := etree.FromString(`
	<ul>Description of the list:
		<li>List item 1</li>
		<li>List item 2</li>
		<li>List item 3</li>
	</ul>`)

	handledLists := handleLists(lists, nil, zeroOpts)
	strResult := etree.ToString(handledLists)
	assert.Equal(t, 3, strings.Count(strResult, "List item"))
	assert.Contains(t, strResult, "Description")
}

func Test_Cache(t *testing.T) {
	cache := NewCache(2)

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
	fnHtml := func(r *ExtractResult) string {
		return etree.ToString(r.ContentNode)
	}

	// Simple
	r := strings.NewReader("<html><body><p><b>This here is in bold font.</b></p></body></html>")
	result, _ := Extract(r, zeroOpts)
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
	assert.Equal(t, "Wild text", dom.TextContent(result.ContentNode))

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

	converted := handleFormatting(element)
	assert.Equal(t, etree.ToString(converted), "<p><b>Here is the text.</b>And a tail.</p>")
}

func Test_Baseline(t *testing.T) {
	// Blank document
	doc := docFromStr("")
	_, result := baseline(doc)
	assert.Equal(t, "", result)

	// Extract articleBody in JSON+LD
	doc = docFromStr(`<html><body><script type="application/ld+json">{"description":"In letzter Zeit kam man am Begriff \"Hygge\", was so viel wie \"angenehm\" oder \"gemütlich\" bedeutet, ja nicht vorbei. Jetzt macht ihm ein neuer Glücks-Trend ...","image":[{"name":"Mit der Ikigai-Methode wirst du glücklicher","url":"https:\/\/image.brigitte.de\/10973004\/uncropped-0-0\/7d00b2658fd0a3b19e1b161f4657cc20\/Xw\/ikigai--1-.jpg","width":"2048","height":"1366","@type":"ImageObject"},{"name":"Mit der Ikigai-Methode wirst du glücklicher","url":"https:\/\/image.brigitte.de\/10973004\/16x9-1280-720\/bf947c7c24167d7c0adae0be10942d57\/Uf\/ikigai--1-.jpg","width":"1280","height":"720","@type":"ImageObject"},{"name":"Mit der Ikigai-Methode wirst du glücklicher","url":"https:\/\/image.brigitte.de\/10973004\/16x9-938-528\/bf947c7c24167d7c0adae0be10942d57\/JK\/ikigai--1-.jpg","width":"938","height":"528","@type":"ImageObject"},{"name":"Mit der Ikigai-Methode wirst du glücklicher","url":"https:\/\/image.brigitte.de\/10973004\/large1x1-622-622\/f5544b7d67e1be04f7729b130e7e0485\/KN\/ikigai--1-.jpg","width":"622","height":"622","@type":"ImageObject"}],"mainEntityOfPage":{"@id":"https:\/\/www.brigitte.de\/liebe\/persoenlichkeit\/ikigai-macht-dich-sofort-gluecklicher--10972896.html","@type":"WebPage"},"headline":"Ikigai macht dich sofort glücklicher!","datePublished":"2019-06-19T14:29:08+0000","dateModified":"2019-06-19T14:29:10+0000","author":{"name":"BRIGITTE.de","@type":"Organization"},"publisher":{"name":"BRIGITTE.de","logo":{"url":"https:\/\/image.brigitte.de\/11476842\/uncropped-0-0\/f19537e97b9189bf0f25ce924168bedb\/kK\/bri-logo-schema-org.png","width":"167","height":"60","@type":"ImageObject"},"@type":"Organization"},"articleBody":"In letzter Zeit kam man am Begriff \"Hygge\" (\"gemütlich\" oder \"angenehm\") nicht vorbei. Jetzt macht ihm ein neuer Glücks-Trend Konkurrenz: \"Ikigai\". Bist du glücklich? Schwierige Frage, nicht wahr? Viele von uns müssen da erst mal überlegen.","@type":"NewsArticle"}</script></body></html>`)
	_, result = baseline(doc)
	assert.True(t, strings.HasPrefix(result, "In letzter Zeit kam man"))
	assert.True(t, strings.HasSuffix(result, "erst mal überlegen."))

	// Extract from <article> tag
	doc = docFromStr("<html><body><article><b>The article consists of this text.</b></article></body></html>")
	_, result = baseline(doc)
	assert.NotEmpty(t, result)
	assert.Equal(t, "The article consists of this text.", result)

	// Extract from quote
	doc = docFromStr("<html><body><blockquote>This is only a quote but it is better than nothing.</blockquote></body></html>")
	_, result = baseline(doc)
	assert.NotEmpty(t, result)
	assert.Equal(t, "This is only a quote but it is better than nothing.", result)
}

func Test_Language(t *testing.T) {
	// Main text
	assert.Equal(t, "de", getLanguage("Hier ist ein Text auf Deutsch", ""))

	// Comments text
	assert.Equal(t, "de", getLanguage("This is English.", "Die Kommentare sind aber etwas länger."))
}

func Test_Filters(t *testing.T) {
	// Helper function
	rRepeatElement := func(element string, repeat int) io.Reader {
		str := fmt.Sprintf("<html><body>%s</body></html>", strings.Repeat(element, repeat))
		return strings.NewReader(str)
	}

	// URL blacklist
	opts := Options{URLBlacklist: []string{"https://example.org"}}
	r := strings.NewReader(`<html><head><link rel="canonical" href="https://example.org"/></head><body></body></html>`)
	result, _ := Extract(r, opts)
	assert.Nil(t, result)

	// Recursion limit
	p1 := "<p>abc</p>"
	p2 := "<p><i>abc</i></p>"
	opts = Options{MaxTreeSize: 500}

	result, _ = Extract(rRepeatElement(p1, 50), opts)
	assert.NotNil(t, result)

	result, _ = Extract(rRepeatElement(p1, 501), opts)
	assert.Nil(t, result)

	result, _ = Extract(rRepeatElement(p2, 501), opts)
	assert.Nil(t, result)

	result, _ = Extract(rRepeatElement(p2, 499), opts)
	assert.NotNil(t, result)

	// HTML lang filter
	// TODO: In original Trafilatura, the value of p3 is set to "In sleep a king,
	// but waking no such matter." which is part of Sonnet 87, classic English poem
	// by Shakespear. Unfortunately, whatlanggo struggle to detect its language.
	// However, when I added the entire closure of Sonnet 87, it works. Need to
	// investigate later.
	p3 := "<p>Thus have I had thee as a dream doth flatter, In sleep a king, but waking no such matter.</p>"
	str := `<html lang="en-US"><body>` + strings.Repeat(p3, 50) + `</body></html>`

	opts.TargetLanguage = "en"
	result, _ = Extract(strings.NewReader(str), opts)
	assert.NotNil(t, result)

	opts.TargetLanguage = "de"
	result, _ = Extract(strings.NewReader(str), opts)
	assert.Nil(t, result)

	doc := docFromStr(`<html lang="de_DE, en_US"><body></body></html>`)
	assert.True(t, checkHtmlLanguage(doc, "de"))

	doc = docFromStr(`<html lang="en"><body></body></html>`)
	assert.False(t, checkHtmlLanguage(doc, "it"))

	doc = docFromStr(`<html><head><meta http-equiv="content-language" content="en"></head><body></body></html>`)
	assert.True(t, checkHtmlLanguage(doc, "en"))

	doc = docFromStr(`<html><head><meta http-equiv="content-language" content="en"></head><body></body></html>`)
	assert.False(t, checkHtmlLanguage(doc, "de"))
}

func Test_External(t *testing.T) {
	// Remove unwanted elements
	doc := docFromStr(`<html><body><footer>Test text</footer></body></html>`)
	sanitizeTree(doc, defaultOpts)
	assert.Empty(t, etree.IterText(doc, " "))

	doc = docFromStr(`<html><body><table><th>Test text</th><tr><td>Test</td></tr></table></body></html>`)
	sanitizeTree(doc, defaultOpts)
	assert.NotEmpty(t, etree.IterText(doc, " "))

	// Strip fancy tags
	doc = docFromStr(`<html><body><p>Text here <fancy>Test text</fancy></p></body></html>`)
	sanitizeTree(doc, defaultOpts)
	assert.Contains(t, dom.OuterHTML(doc), "<p>Text here Test text</p>")

	// Test language
	opts := Options{TargetLanguage: "en"}
	str := `<html><body>` + strings.Repeat("<p>Non è inglese.</p>", 20) + `</body></html>`
	result, _ := Extract(strings.NewReader(str), opts)
	assert.Nil(t, result)
}

func Test_Images(t *testing.T) {
	// Image handler
	img := handleImage(etree.FromString(`<img src="test.jpg"/>`))
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
	f, _ := os.Open(filepath.Join("test-files", "mock", "http_sample.html"))
	bt, _ := ioutil.ReadAll(f)

	opts := defaultOpts
	result, _ := Extract(bytes.NewReader(bt), opts)
	contentHtml := dom.OuterHTML(result.ContentNode)
	assert.NotContains(t, contentHtml, `<img src="test.jpg" title="Example image"/>`)

	opts.NoFallback = true
	opts.IncludeImages = true
	result, _ = Extract(bytes.NewReader(bt), opts)
	contentHtml = dom.OuterHTML(result.ContentNode)
	assert.Contains(t, contentHtml, `<img src="test.jpg" title="Example image"/>`)

	// CNN example
	f, _ = os.Open(filepath.Join("test-files", "mock", "cnn-image.html"))
	doc, _ := html.Parse(f)
	img = handleImage(dom.QuerySelector(doc, "img"))
	assert.NotNil(t, img)
	assert.True(t, dom.HasAttribute(img, "alt"))
	assert.True(t, dom.HasAttribute(img, "src"))

	// Modified CNN example
	f, _ = os.Open(filepath.Join("test-files", "mock", "cnn-image-modified.html"))
	doc, _ = html.Parse(f)
	img = handleImage(dom.QuerySelector(doc, "img"))
	assert.NotNil(t, img)
	assert.True(t, dom.HasAttribute(img, "alt"))
	assert.True(t, dom.HasAttribute(img, "src"))
	assert.True(t, strings.HasPrefix(dom.GetAttribute(img, "src"), "http"))
}

func Test_Links(t *testing.T) {
	// Prepare options
	linkOpts := Options{
		IncludeLinks: true, NoFallback: true,
		Config: &Config{
			MinOutputSize:    0,
			MinExtractedSize: 0,
		},
	}

	// Test handleTextElem
	processed := handleTextElem(etree.Element("a"), nil, nil, defaultOpts)
	assert.Nil(t, processed)

	// Formatting link
	element := etree.FromString(`<a href="testlink.html">Test link text.</a>`)
	processed = handleFormatting(element)
	assert.NotNil(t, processed)

	// Extracting document with links
	htmlStr := `<html><body><p><a href="testlink.html">Test link text.</a></p></body></html>`
	result, _ := Extract(strings.NewReader(htmlStr), zeroOpts)
	assert.NotContains(t, dom.OuterHTML(result.ContentNode), "testlink.html")

	result, _ = Extract(strings.NewReader(htmlStr), linkOpts)
	assert.Contains(t, dom.OuterHTML(result.ContentNode), "testlink.html")

	// Extracting document  with links, from file
	f, _ := os.Open(filepath.Join("test-files", "mock", "http_sample.html"))
	bt, _ := ioutil.ReadAll(f)

	result, _ = Extract(bytes.NewReader(bt), zeroOpts)
	assert.NotContains(t, dom.OuterHTML(result.ContentNode), "testlink.html")

	result, _ = Extract(bytes.NewReader(bt), linkOpts)
	assert.Contains(t, dom.OuterHTML(result.ContentNode), "testlink.html")
}
