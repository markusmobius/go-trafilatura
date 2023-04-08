package trafilatura

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_MetadataJson_Authors(t *testing.T) {
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

	metadata = testGetMetadataFromFile("simple/json-metadata-6.html")
	assert.Equal(t, "Douglas Noel Adams", metadata.Author)

	metadata = testGetMetadataFromFile("simple/json-metadata-7.html")
	assert.Empty(t, metadata.Categories)
}
