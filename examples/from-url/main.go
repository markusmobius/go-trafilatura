package main

import (
	"fmt"
	"net/http"
	nurl "net/url"
	"regexp"
	"time"

	"github.com/go-shiori/dom"
	"github.com/markusmobius/go-trafilatura"
	"github.com/sirupsen/logrus"
)

var (
	httpClient = &http.Client{Timeout: 30 * time.Second}
	rxCharset  = regexp.MustCompile(`(?i)charset\s*=\s*([^;\s"]+)`)
)

func main() {
	// Prepare URL
	url := "https://www.finanzen.net/nachricht/trading/anzeige-value-stars-mit-ausgewaehlten-aktien-den-dax-schlagen-5873873"
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

	// Print result
	fmt.Println(dom.OuterHTML(result.ContentNode))
}
