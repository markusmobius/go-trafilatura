package main

import (
	"fmt"
	"io/fs"
	"os"
	fp "path/filepath"
	"time"

	"github.com/go-shiori/dom"
	"github.com/go-shiori/go-readability"
	distiller "github.com/markusmobius/go-domdistiller"
	"github.com/markusmobius/go-htmldate"
	"github.com/markusmobius/go-trafilatura"
	"golang.org/x/net/html"
)

func main() {
	// Find file names
	filePaths, err := getFileList()
	checkError(err)

	// Process each path. All error handling is skipped here for brevity,
	// in real world we should check and handle each error.
	var nDocument int
	var parseTime time.Duration
	var readabilityTime time.Duration
	var domDistillerTime time.Duration
	var trafilaturaTime time.Duration
	var dateTime time.Duration

	for _, path := range filePaths {
		nDocument++

		// Parse file
		start := time.Now()
		doc, err := parseFile(path)
		checkError(err)
		parseTime += time.Now().Sub(start)

		// Use readability
		start = time.Now()
		readabilityResult, _ := readability.FromDocument(doc, nil)
		readabilityTime += time.Now().Sub(start)

		// Use dom distiller
		start = time.Now()
		distillerOpts := &distiller.Options{SkipPagination: true}
		distillerResult, _ := distiller.Apply(doc, distillerOpts)
		domDistillerTime += time.Now().Sub(start)

		// Use trafilatura
		start = time.Now()
		trafilaturaOpts := trafilatura.Options{
			FallbackCandidates: []*html.Node{
				readabilityResult.Node,
				distillerResult.Node,
			},
		}

		trafilatura.ExtractDocument(doc, trafilaturaOpts)
		trafilaturaTime += time.Now().Sub(start)

		// Use html date
		start = time.Now()

		// Last modified date
		dateOpts := htmldate.Options{}
		htmldate.FromDocument(doc, dateOpts)

		// Publish date
		dateOpts.UseOriginalDate = true
		htmldate.FromDocument(doc, dateOpts)
		dateTime += time.Now().Sub(start)
	}

	// Print message
	parseDuration := parseTime.Seconds()
	readabilityDuration := readabilityTime.Seconds()
	domDistillerDuration := domDistillerTime.Seconds()
	trafilaturaDuration := trafilaturaTime.Seconds()
	dateDuration := dateTime.Seconds()
	totalDuration := parseDuration + readabilityDuration +
		domDistillerDuration + trafilaturaDuration + dateDuration

	parseSpeed := float64(nDocument) / parseDuration
	readabilitySpeed := float64(nDocument) / readabilityDuration
	domDistillerSpeed := float64(nDocument) / domDistillerDuration
	trafilaturaSpeed := float64(nDocument) / trafilaturaDuration
	dateSpeed := float64(nDocument) / dateDuration
	avgSpeed := float64(nDocument) / totalDuration

	fmt.Printf("N document : %d\n", nDocument)
	fmt.Printf("Parsing    : %.3f s (%.3f doc/s)\n", parseDuration, parseSpeed)
	fmt.Printf("Readability: %.3f s (%.3f doc/s)\n", readabilityDuration, readabilitySpeed)
	fmt.Printf("Distiller  : %.3f s (%.3f doc/s)\n", domDistillerDuration, domDistillerSpeed)
	fmt.Printf("Trafilatura: %.3f s (%.3f doc/s)\n", trafilaturaDuration, trafilaturaSpeed)
	fmt.Printf("HtmlDate   : %.3f s (%.3f doc/s)\n", dateDuration, dateSpeed)
	fmt.Printf("Total      : %.3f s (%.3f doc/s)\n", totalDuration, avgSpeed)

}

func getFileList() ([]string, error) {
	var filePaths []string
	err := fp.Walk("test-files", func(path string, info fs.FileInfo, err error) error {
		if !info.IsDir() && fp.Ext(path) == ".html" {
			filePaths = append(filePaths, path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return filePaths, nil
}

func parseFile(path string) (*html.Node, error) {
	// Open file
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return dom.Parse(f)
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}
