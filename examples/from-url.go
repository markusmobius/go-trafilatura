// +build ignore

package main

import (
	"fmt"
	"net/http"
	nurl "net/url"
	"time"

	"github.com/go-shiori/dom"
	"github.com/markusmobius/go-trafilatura"
	"github.com/sirupsen/logrus"
)

var (
	httpClient = &http.Client{Timeout: 30 * time.Second}
)

func main() {
	// Prepare URL
	url := "https://arstechnica.com/science/2021/05/rare-flesh-eating-black-fungus-rides-covids-coattails-in-india/"
	parsedURL, err := nurl.ParseRequestURI(url)
	if err != nil {
		logrus.Fatalf("failed to parse url: %v", err)
	}

	// Fetch article
	resp, err := httpClient.Get(url)
	if err != nil {
		logrus.Fatalf("failed to fetch the page: %v", err)
	}
	defer resp.Body.Close()

	// Extract content
	opts := trafilatura.Options{
		IncludeImages: true,
		OriginalURL:   parsedURL,
	}

	result, err := trafilatura.Extract(resp.Body, opts)
	if err != nil {
		logrus.Fatalf("failed to extract: %v", err)
	}

	fmt.Println(dom.OuterHTML(result.ContentNode))
}
