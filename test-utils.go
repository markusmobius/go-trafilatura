package trafilatura

import (
	"io"
	nurl "net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"golang.org/x/net/html"
)

// openMockFile is used to open HTML document from specified mock file.
// Make sure to close the reader later.
func openMockFile(mockFiles map[string]string, url string) io.ReadCloser {
	// Open file
	path := mockFiles[url]
	path = filepath.Join("test-files", path)

	f, err := os.Open(path)
	if err != nil {
		logrus.Panicln(err)
	}

	return f
}

// parseMockFile open then convert a mock file into html.Node.
func parseMockFile(mockFiles map[string]string, url string) *html.Node {
	f := openMockFile(mockFiles, url)
	defer f.Close()

	doc, err := html.Parse(f)
	if err != nil {
		logrus.Panicln(err)
	}

	return doc
}

// extractMockFile open then extract content from a mock file.
func extractMockFile(mockFiles map[string]string, url string) *ExtractResult {
	// Open mock file
	f := openMockFile(mockFiles, url)
	defer f.Close()

	// Parse URL
	parsedURL, err := nurl.ParseRequestURI(url)
	if err != nil {
		logrus.Panicln(err)
	}

	// Extract
	opts := Options{OriginalURL: parsedURL, NoFallback: true}
	result, err := Extract(f, opts)
	if err != nil {
		logrus.Panicln(err)
	}

	return result
}

// docFromStr create document from raw HTML string. Used in tests.
func docFromStr(str string) *html.Node {
	doc, _ := html.Parse(strings.NewReader(str))
	return doc
}
