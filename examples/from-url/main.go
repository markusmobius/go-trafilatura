package main

import (
	"fmt"
	"log"
	"net/http"
	nurl "net/url"
	"time"

	"github.com/go-shiori/dom"
	"github.com/markusmobius/go-trafilatura"
)

var (
	httpClient = &http.Client{Timeout: 30 * time.Second}
)

func main() {
	// Prepare URL
	url := "https://www.federalreserve.gov/monetarypolicy/fomcminutes20160727.htm"
	parsedURL, err := nurl.ParseRequestURI(url)
	if err != nil {
		log.Fatalf("failed to parse url: %v", err)
	}

	// Fetch article
	resp, err := httpClient.Get(url)
	if err != nil {
		log.Fatalf("failed to fetch the page: %v", err)
	}
	defer resp.Body.Close()

	// Extract content
	opts := trafilatura.Options{
		IncludeImages: true,
		OriginalURL:   parsedURL,
	}

	result, err := trafilatura.Extract(resp.Body, opts)
	if err != nil {
		log.Fatalf("failed to extract: %v", err)
	}

	// Print result
	doc := trafilatura.CreateReadableDocument(result)
	fmt.Println(dom.OuterHTML(doc))
}
