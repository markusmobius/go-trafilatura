package main

import (
	"fmt"
	"log"
	"net/http"
	nurl "net/url"
	"os"
	"time"

	"github.com/go-shiori/dom"
	"github.com/markusmobius/go-trafilatura"
)

var (
	httpClient = &http.Client{Timeout: 30 * time.Second}
)

func main() {
	// Open file
	f, err := os.Open("input3.html")
	if err != nil {
		log.Fatalf("failed to open: %v", err)
	}
	defer f.Close()

	// Extract content
	parsedURL, _ := nurl.ParseRequestURI("https://www.faz.net/aktuell/rhein-main/frankfurt/hoch-gepokert-krachend-gescheitert-frankfurter-fdp-17361555.html")
	opts := trafilatura.Options{
		ExcludeComments: true,
		IncludeImages:   true,
		IncludeLinks:    true,
		EnableLog:       true,
		OriginalURL:     parsedURL,
	}

	result, err := trafilatura.Extract(f, opts)
	if err != nil {
		log.Fatalf("failed to extract: %v", err)
	}

	// Print result
	fmt.Println(dom.OuterHTML(result.ContentNode))
}
