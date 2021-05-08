package trafilatura

import (
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

// loadMockFile is used to load HTML document from specified mock file.
func loadMockFile(mockFiles map[string]string, url string) *html.Node {
	// Open file
	path := mockFiles[url]
	path = filepath.Join("test-files", path)
	f, err := os.Open(path)
	if err != nil {
		logrus.Panicln(err)
	}

	// Parse HTML
	doc, err := html.Parse(f)
	if err != nil {
		logrus.Panicln(err)
	}

	return doc
}
