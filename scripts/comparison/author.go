package main

import (
	"fmt"
	nurl "net/url"
	"strings"
	"time"

	"github.com/markusmobius/go-trafilatura"
)

func compareAuthorExtraction() {
	var nDocument int
	var nCorrect int
	start := time.Now()

	for strURL, entry := range comparisonData {
		// Make sure entry valid
		if entry.File == "" || len(entry.Authors) == 0 {
			continue
		}

		// Make sure URL is valid
		url, err := nurl.ParseRequestURI(strURL)
		if err != nil {
			log.Error().Msgf("failed to parse %s: %v", strURL, err)
			continue
		}

		// Open file
		f, err := openDataFile(entry.File)
		if err != nil {
			log.Error().Msgf("%v", err)
			continue
		}

		// Run trafilatura
		result, _ := trafilatura.Extract(f, trafilatura.Options{
			OriginalURL: url,
		})

		// Compare result
		nDocument++
		if result != nil {
			if strings.Join(entry.Authors, "; ") == result.Metadata.Author {
				nCorrect++
			}
		}
	}

	// Print result
	fmt.Printf("Duration:   %v\n", time.Since(start))
	fmt.Printf("N document: %d\n", nDocument)
	fmt.Printf("N correct:  %d\n", nCorrect)
	fmt.Printf("Percentage: %f\n", float64(nCorrect)/float64(nDocument)*100)
}
