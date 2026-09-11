package trafilatura

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_MetadataJson(t *testing.T) {
	var metadata Metadata

	metadata = testGetMetadataFromFile("simple/json-metadata-1-a.html")
	assert.Equal(t, "Maggie Haberman; Shane Goldmacher; Michael Crowley", metadata.Author)
	assert.Equal(t, "newsarticle", metadata.PageType)

	metadata = testGetMetadataFromFile("simple/json-metadata-1-b.html")
	assert.Equal(t, "Safety Insurance Group, Inc.", metadata.Sitename)

	metadata = testGetMetadataFromFile("simple/json-metadata-2-a.html")
	assert.Equal(t, "Jenny Smith", metadata.Author)
	assert.Equal(t, "newsarticle", metadata.PageType)

	metadata = testGetMetadataFromFile("simple/json-metadata-2-b.html")
	assert.Equal(t, "Amir Vera; Seán Federico O'Murchú; Tara Subramaniam; Adam Renton; CNN", metadata.Author)
	assert.Equal(t, "newsarticle", metadata.PageType)

	metadata = testGetMetadataFromFile("simple/json-metadata-2-c.html")
	assert.Equal(t, "Deborah O'Donoghue", metadata.Author)
	assert.Equal(t, "website", metadata.PageType)

	metadata = testGetMetadataFromFile("simple/json-metadata-2-d.html")
	assert.Equal(t, "Sam McPhee; Tara Cosoleto", metadata.Author)
	assert.Equal(t, "newsarticle", metadata.PageType)

	metadata = testGetMetadataFromFile("simple/json-metadata-3-a.html")
	assert.Empty(t, metadata.Author)
	assert.Empty(t, metadata.PageType)

	metadata = testGetMetadataFromFile("simple/json-metadata-3-b.html")
	assert.Equal(t, "John Doe", metadata.Author)
	assert.Equal(t, "article", metadata.PageType)
	assert.Equal(t, "Example Article", metadata.Title)

	metadata = testGetMetadataFromFile("simple/json-metadata-3-c.html")
	assert.Equal(t, "John Doe", metadata.Author)
	assert.Equal(t, "blogposting", metadata.PageType)
	assert.Equal(t, "Breaking News: Example Article", metadata.Title)

	metadata = testGetMetadataFromFile("simple/json-metadata-3-d.html")
	assert.Equal(t, "Example Webpage", metadata.Sitename)

	metadata = testGetMetadataFromFile("simple/json-metadata-4.html")
	assert.Equal(t, "Coming this April, HBO NOW will be available exclusively in the U.S. on Apple TV and the App Store.", metadata.Title)
	assert.Equal(t, "blogposting", metadata.PageType)

	metadata = testGetMetadataFromFile("simple/json-metadata-5.html")
	assert.Equal(t, "iPhone is growing at nearly twice the rate of the rest of the smartphone market.", metadata.Title)
	assert.Equal(t, "blogposting", metadata.PageType)

	metadata = testGetMetadataFromFile("simple/json-metadata-5-b.html")
	assert.Equal(t, "Apple Spring Forward Event Live Blog", metadata.Title)
	assert.Equal(t, "liveblogposting", metadata.PageType)

	metadata = testGetMetadataFromFile("simple/json-metadata-6.html")
	assert.Equal(t, "Douglas Noel Adams", metadata.Author)
	assert.Equal(t, "socialmediaposting", metadata.PageType)

	metadata = testGetMetadataFromFile("simple/json-metadata-7.html")
	assert.Empty(t, metadata.Categories)
	assert.Equal(t, "article", metadata.PageType)

	metadata = testGetMetadataFromFile("simple/json-metadata-8.html")
	assert.Equal(t, "Mickelson comments hurt new league: Norman", metadata.Title)
	assert.Equal(t, "7News", metadata.Sitename)
	assert.Equal(t, "Digital Staff", metadata.Author)
	assert.Contains(t, metadata.Categories, "Golf")
	assert.Equal(t, "webpage", metadata.PageType)

	metadata = testGetMetadataFromFile("simple/json-metadata-9.html")
	assert.Equal(t, "Australians stuck in Shanghai's COVID lockdown beg consular officials to help them flee", metadata.Title)
	assert.Equal(t, "ABC News", metadata.Sitename)
	assert.Equal(t, "Bill Birtles", metadata.Author)
	assert.Equal(t, "newsarticle", metadata.PageType)

	metadata = testGetMetadataFromFile("simple/json-metadata-10.html")
	assert.Equal(t, "New York City Enters Higher Coronavirus Risk Level as Case Numbers Rise", metadata.Title)
	assert.Equal(t, "The New York Times", metadata.Sitename)
	assert.Equal(t, "Sharon Otterman; Emma G Fitzsimmons", metadata.Author)
	assert.Equal(t, "newsarticle", metadata.PageType)

	metadata = testGetMetadataFromFile("simple/json-metadata-11.html")
	assert.Equal(t, "Decreto permite que consumidor cancele serviços de empresas via WhatsApp", metadata.Title)
	assert.Equal(t, "UOL", metadata.Sitename)
	assert.Equal(t, "Caio Mello", metadata.Author)
	assert.Equal(t, "newsarticle", metadata.PageType)

	metadata = testGetMetadataFromFile("simple/json-metadata-12.html")
	assert.Equal(t, "12 words and phrases you need to survive in Hamburg", metadata.Title)
	assert.Equal(t, "The Local", metadata.Sitename)
	assert.Equal(t, "Alexander Johnstone", metadata.Author)
	assert.Equal(t, "newsarticle", metadata.PageType)

	metadata = testGetMetadataFromFile("simple/json-metadata-13.html")
	assert.Equal(t, "Andreessen Horowitz", metadata.Sitename)
	assert.Equal(t, "", metadata.Author)
	assert.Equal(t, "website", metadata.PageType)

	metadata = testGetMetadataFromFile("simple/json-metadata-14.html")
	assert.Equal(t, "", metadata.Sitename)
	assert.Equal(t, "", metadata.Author)
	assert.Equal(t, "", metadata.PageType)

	metadata = testGetMetadataFromFile("simple/json-metadata-15.html")
	assert.Equal(t, "", metadata.Sitename)
	assert.Equal(t, "", metadata.Author)
	assert.Empty(t, metadata.PageType)

	metadata = testGetMetadataFromFile("simple/json-metadata-16.html")
	assert.Equal(t, "", metadata.Sitename)
	assert.Equal(t, "", metadata.Author)
	assert.Empty(t, metadata.PageType)

	metadata = testGetMetadataFromFile("simple/json-metadata-17.html")
	assert.Equal(t, "", metadata.Sitename)
	assert.Equal(t, "Coming this April, HBO NOW will be available exclusively in the U.S. on Apple TV and the App Store.", metadata.Title)
	assert.Equal(t, "blogposting", metadata.PageType)

	metadata = testGetMetadataFromFile("simple/json-metadata-17-b.html")
	assert.Equal(t, "", metadata.Sitename)
	assert.Equal(t, "", metadata.Title)
	assert.Equal(t, "", metadata.PageType)

	metadata = testGetMetadataFromFile("simple/json-metadata-18.html")
	assert.Equal(t, "EastEnders' June Brown leaves soap 'for good'", metadata.Title)
	assert.Equal(t, "BBC News", metadata.Sitename)
	assert.Equal(t, "reportagenewsarticle", metadata.PageType)

	metadata = testGetMetadataFromFile("simple/json-metadata-19.html")
	assert.Equal(t, "BBC News", metadata.Sitename)
	assert.Equal(t, "reportagenewsarticle", metadata.PageType)

	metadata = testGetMetadataFromFile("simple/json-metadata-20.html")
	assert.Equal(t, "John Doe", metadata.Author)
	assert.Equal(t, "How to Tie a Reef Knot", metadata.Title)
	assert.Equal(t, "article", metadata.PageType)

	metadata = testGetMetadataFromFile("simple/json-metadata-21.html")
	assert.Equal(t, "Bill Birtles; John Smith", metadata.Author)
	assert.Equal(t, "newsarticle", metadata.PageType)

	metadata = testGetMetadataFromFile("simple/json-metadata-22.html")
	assert.Equal(t, "Find perfection in these places where land meets water.", metadata.Title)
	assert.Equal(t, "National Geographic", metadata.Sitename)
	assert.Equal(t, "Kimberley Lovato", metadata.Author)
	assert.Equal(t, "article", metadata.PageType)

	metadata = testGetMetadataFromFile("simple/json-metadata-23.html")
	assert.Empty(t, metadata.Title)
	assert.Empty(t, metadata.Author)
}

func Test_MetadataJson_PublisherGuard(test *testing.T) {
	for _, testCase := range []struct {
		publisher string
		original  string
		expected  string
	}{
		{`{"name":"BBC"}`, "https://www.bbc.com", "BBC"},
		{`{"name":"BBC"}`, "BBC News", "BBC News"},
		{`{"name":"BBC News"}`, "BBC", "BBC News"},
		{`{"name":""}`, "BBC News", "BBC News"},
		{`{"name":123}`, "BBC News", "BBC News"},
		{`null`, "BBC News", "BBC News"},
	} {
		doc := docFromStr(`<html><head><script type="application/ld+json">{"@context":"https://schema.org","@type":"NewsArticle","publisher":` + testCase.publisher + `}</script></head></html>`)
		metadata := extractJsonLd(defaultOpts, doc, Metadata{Sitename: testCase.original})
		assert.Equal(test, testCase.expected, metadata.Sitename)
	}
}

func Test_MetadataJson_AuthorCandidates(test *testing.T) {
	testCases := []struct {
		name     string
		schema   string
		original string
		expected string
	}{
		{
			name:     "publisher people",
			schema:   `{"@context":"https://schema.org","@type":"NewsArticle","publisher":{"@type":"Organization","employee":{"@type":"Person","name":"Employee Person"},"founder":{"@type":"Person","name":"Founder Person"}}}`,
			original: "Meta Writer",
			expected: "Meta Writer",
		},
		{
			name:   "untyped publisher people",
			schema: `{"@context":"https://schema.org","@type":"NewsArticle","publisher":{"employee":{"@type":"Person","name":"Employee Person"},"founder":{"@type":"Person","name":"Founder Person"}}}`,
		},
		{
			name:     "explicit author order",
			schema:   `{"@context":"https://schema.org","@type":"NewsArticle","author":[{"@type":"Person","name":"Second Writer"},{"@type":"Person","name":"First Writer"}],"image":{"@type":"ImageObject","author":{"@type":"Person","name":"Photographer Person"}},"publisher":{"@type":"Organization","founder":{"@type":"Person","name":"Founder Person"}}}`,
			expected: "Second Writer; First Writer",
		},
		{
			name:     "standalone person",
			schema:   `{"@context":"https://schema.org","@type":"Person","name":"Standalone Writer"}`,
			expected: "Standalone Writer",
		},
		{
			name:     "graph people",
			schema:   `{"@context":"https://schema.org","@graph":[{"@type":"Person","name":"Second Writer"},{"@type":"Person","name":"First Writer"}]}`,
			expected: "Second Writer; First Writer",
		},
	}
	for _, testCase := range testCases {
		test.Run(testCase.name, func(test *testing.T) {
			input := `<html><head><meta name="author" content="` + testCase.original + `"><script type="application/ld+json">` + testCase.schema + `</script></head><body></body></html>`
			metadata := testGetMetadataFromHTML(input)
			assert.Equal(test, testCase.expected, metadata.Author)
		})
	}

	for _, fixture := range []struct {
		file   string
		author string
	}{
		{"haufe.de-ordnungsgeld.html", "Www Haufe De; Copyright Haufe-Lexware GmbH; Co KG"},
		{"fischundfang.de-pop-ups.html", "Fisch; Fang Seminare"},
		{"wildundhund.de-bonn.html", "WILD; HUND aktiv"},
	} {
		test.Run(fixture.file, func(test *testing.T) {
			metadata := testGetMetadataFromFile("comparison/" + fixture.file)
			assert.Equal(test, fixture.author, metadata.Author)
		})
	}
}

func Test_MetadataJson_AuthorShapes(test *testing.T) {
	testCases := []struct {
		authors  string
		expected string
	}{
		{`"John Doe"`, "John Doe"},
		{`{"name":"John Doe"}`, "John Doe"},
		{`{"@type":"Person","name":{"@type":"Person","name":"John Doe"}}`, "John Doe"},
		{`["John Doe", {"name":"Jane Smith"}, null, false, 42, {"@type":"Organization","name":"Not An Author"}]`, "John Doe; Jane Smith"},
		{`{"@type":"Person","name":42}`, ""},
	}
	for _, testCase := range testCases {
		test.Run(testCase.authors, func(test *testing.T) {
			input := `<html><head><script type="application/ld+json">{"@context":"https://schema.org","@type":"NewsArticle","headline":"Example Article","publisher":"Bare Publisher","author":` + testCase.authors + `}</script></head></html>`
			metadata := testGetMetadataFromHTML(input)
			assert.Equal(test, testCase.expected, metadata.Author)
			assert.Empty(test, metadata.Sitename)
		})
	}
}
