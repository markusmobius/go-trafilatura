package trafilatura

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_MetadataJson(t *testing.T) {
	var metadata Metadata

	metadata = testGetMetadataFromFile("simple/json-metadata-1.html")
	assert.Equal(t, "Maggie Haberman; Shane Goldmacher; Michael Crowley", metadata.Author)

	metadata = testGetMetadataFromFile("simple/json-metadata-2.html")
	assert.Equal(t, "Jenny Smith", metadata.Author)

	metadata = testGetMetadataFromFile("simple/json-metadata-3.html")
	assert.Equal(t, "Jean Sévillia", metadata.Author)

	metadata = testGetMetadataFromFile("simple/json-metadata-4.html")
	assert.Equal(t, "Apple Spring Forward Event Live Blog", metadata.Title)

	metadata = testGetMetadataFromFile("simple/json-metadata-5.html")
	assert.Equal(t, "Apple Spring Forward Event Live Blog", metadata.Title)

	metadata = testGetMetadataFromFile("simple/json-metadata-5-b.html")
	assert.Equal(t, "Apple Spring Forward Event Live Blog", metadata.Title)

	metadata = testGetMetadataFromFile("simple/json-metadata-6.html")
	assert.Equal(t, "Douglas Noel Adams", metadata.Author)

	metadata = testGetMetadataFromFile("simple/json-metadata-7.html")
	assert.Empty(t, metadata.Categories)

	metadata = testGetMetadataFromFile("simple/json-metadata-8.html")
	assert.Equal(t, "Mickelson comments hurt new league: Norman", metadata.Title)
	assert.Equal(t, "7News", metadata.Sitename)
	assert.Equal(t, "Digital Staff", metadata.Author)
	assert.Contains(t, metadata.Categories, "Golf")

	metadata = testGetMetadataFromFile("simple/json-metadata-9.html")
	assert.Equal(t, "Australians stuck in Shanghai's COVID lockdown beg consular officials to help them flee", metadata.Title)
	assert.Equal(t, "ABC News", metadata.Sitename)
	assert.Equal(t, "Bill Birtles", metadata.Author)

	metadata = testGetMetadataFromFile("simple/json-metadata-10.html")
	assert.Equal(t, "New York City Enters Higher Coronavirus Risk Level as Case Numbers Rise", metadata.Title)
	assert.Equal(t, "The New York Times", metadata.Sitename)
	assert.Equal(t, "Sharon Otterman; Emma G Fitzsimmons", metadata.Author)

	metadata = testGetMetadataFromFile("simple/json-metadata-11.html")
	assert.Equal(t, "Decreto permite que consumidor cancele serviços de empresas via WhatsApp", metadata.Title)
	assert.Equal(t, "UOL", metadata.Sitename)
	assert.Equal(t, "Caio Mello", metadata.Author)

	metadata = testGetMetadataFromFile("simple/json-metadata-12.html")
	assert.Equal(t, "12 words and phrases you need to survive in Hamburg", metadata.Title)
	assert.Equal(t, "The Local", metadata.Sitename)
	assert.Equal(t, "Alexander Johnstone", metadata.Author)

	metadata = testGetMetadataFromFile("simple/json-metadata-13.html")
	assert.Equal(t, "Andreessen Horowitz", metadata.Sitename)
	assert.Equal(t, "", metadata.Author)

	metadata = testGetMetadataFromFile("simple/json-metadata-14.html")
	assert.Equal(t, "", metadata.Sitename)
	assert.Equal(t, "", metadata.Author)

	metadata = testGetMetadataFromFile("simple/json-metadata-15.html")
	assert.Equal(t, "", metadata.Sitename)
	assert.Equal(t, "", metadata.Author)

	metadata = testGetMetadataFromFile("simple/json-metadata-16.html")
	assert.Equal(t, "", metadata.Sitename)
	assert.Equal(t, "", metadata.Author)

	metadata = testGetMetadataFromFile("simple/json-metadata-17.html")
	assert.Equal(t, "", metadata.Sitename)
	assert.Equal(t, "Apple Spring Forward Event Live Blog", metadata.Title)

	metadata = testGetMetadataFromFile("simple/json-metadata-17-b.html")
	assert.Equal(t, "", metadata.Sitename)
	assert.Equal(t, "", metadata.Title)

	metadata = testGetMetadataFromFile("simple/json-metadata-18.html")
	assert.Equal(t, "EastEnders' June Brown leaves soap 'for good'", metadata.Title)
	assert.Equal(t, "BBC News", metadata.Sitename)

	metadata = testGetMetadataFromFile("simple/json-metadata-19.html")
	assert.Equal(t, "BBC News", metadata.Sitename)

	metadata = testGetMetadataFromFile("simple/json-metadata-20.html")
	assert.Equal(t, "John Doe", metadata.Author)
	assert.Equal(t, "How to Tie a Reef Knot", metadata.Title)

	metadata = testGetMetadataFromFile("simple/json-metadata-21.html")
	assert.Equal(t, "Bill Birtles; John Smith", metadata.Author)

	metadata = testGetMetadataFromFile("simple/json-metadata-22.html")
	assert.Equal(t, "Find perfection in these places where land meets water.", metadata.Title)
	assert.Equal(t, "National Geographic", metadata.Sitename)
	assert.Equal(t, "Kimberley Lovato", metadata.Author)
}
