package trafilatura

import (
	"io"
	nurl "net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
	"golang.org/x/net/html"
)

var (
	rxImageExtension = regexp.MustCompile(`(?i)([^\s]+(\.(jpe?g|png|gif|bmp)))`)
)

// trim removes unnecessary spaces within a text string.
func trim(s string) string {
	s = strings.TrimSpace(s)
	return strings.Join(strings.Fields(s), " ")
}

func strWordCount(s string) int {
	return len(strings.Fields(s))
}

func strOr(args ...string) string {
	for i := 0; i < len(args); i++ {
		if args[i] != "" {
			return args[i]
		}
	}
	return ""
}

func strIn(s string, args ...string) bool {
	for i := 0; i < len(args); i++ {
		if args[i] == s {
			return true
		}
	}
	return false
}

func getRune(s string, idx int) rune {
	for i, r := range s {
		if i == idx {
			return r
		}
	}

	return -1
}

func isImageFile(imageSrc string) bool {
	return imageSrc != "" && rxImageExtension.MatchString(imageSrc)
}

// doNothing is placeholder function to store unused variables
// so Go formatter doesn't complain.
func doNothing(i ...interface{}) {}

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
	opts := Options{OriginalURL: parsedURL}
	result, err := Extract(f, opts)
	if err != nil {
		logrus.Panicln(err)
	}

	return result
}

func docFromStr(str string) *html.Node {
	doc, _ := html.Parse(strings.NewReader(str))
	return doc
}
