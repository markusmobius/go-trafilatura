package main

import (
	"fmt"
	nurl "net/url"
	"strings"
	"time"

	"github.com/go-shiori/dom"
	"github.com/go-shiori/go-readability"
	distiller "github.com/markusmobius/go-domdistiller"
	"github.com/markusmobius/go-trafilatura"
	"golang.org/x/net/html"
)

type ExtractorParameter struct {
	URL      *nurl.URL
	Document *html.Node
	Entry    ComparisonEntry
}

type EvaluationResult struct {
	TruePositives  int
	FalseNegatives int
	FalsePositives int
	TrueNegatives  int
	Duration       time.Duration
}

func compareContentExtraction() {
	params := prepareExtractorParameter()

	var errors []error
	errors = append(errors, runReadability(params)...)                      // Readability
	errors = append(errors, runDomDistiller(params)...)                     // DOM Distiller
	errors = append(errors, runTrafilatura(params, false, false, false)...) // Standard Trafilatura
	errors = append(errors, runTrafilatura(params, true, false, false)...)  // Trafilatura + Fallback
	errors = append(errors, runTrafilatura(params, true, true, false)...)   // Trafilatura + Precision
	errors = append(errors, runTrafilatura(params, true, false, true)...)   // Trafilatura + Recall

	// Print errors
	for _, err := range errors {
		log.Warn().Err(err)
	}
}

func prepareExtractorParameter() []ExtractorParameter {
	var params []ExtractorParameter

	for strURL, entry := range comparisonData {
		// Make sure URL is valid
		url, err := nurl.ParseRequestURI(strURL)
		if err != nil {
			log.Error().Msgf("failed to parse %s: %v", strURL, err)
			continue
		}

		// Open file
		f, err := openDataFile(entry.File)
		if err != nil {
			log.Error().Err(err)
			continue
		}

		// Create document
		doc, err := dom.Parse(f)
		if err != nil {
			log.Error().Msgf("failed to parse %s: %v", entry.File, err)
			continue
		}

		// Save parameters
		params = append(params, ExtractorParameter{
			URL:      url,
			Document: doc,
			Entry:    entry,
		})
	}

	return params
}

func runTrafilatura(params []ExtractorParameter, useFallback, favorPrecision, favorRecall bool) []error {
	title := "trafilatura"
	if useFallback {
		title += "+fallback"
	}

	if favorPrecision {
		title += "+precision"
	} else if favorRecall {
		title += "+recall"
	}

	start := time.Now()
	var errors []error
	var evaluation EvaluationResult
	var opts = trafilatura.Options{
		ExcludeComments: true,
		ExcludeTables:   false,
		FavorPrecision:  favorPrecision,
		FavorRecall:     favorRecall,
	}
	if useFallback {
		opts.FallbackCandidates = &trafilatura.FallbackConfig{}
	}

	for _, param := range params {
		var textResult string
		opts.OriginalURL = param.URL
		result, err := trafilatura.ExtractDocument(param.Document, opts)
		if err != nil {
			errors = append(errors, fmt.Errorf("%s error for %q: %v", title, param.URL, err))
		} else {
			textResult = result.ContentText
		}

		evaluation = evaluateResult(evaluation, textResult, param.Entry)
	}

	evaluation.Duration = time.Since(start)
	printEvaluationResult(title, evaluation)
	fmt.Println()

	return errors
}

func runReadability(params []ExtractorParameter) []error {
	title := "readability"
	start := time.Now()

	var errors []error
	var evaluation EvaluationResult
	for _, param := range params {
		article, err := readability.FromDocument(param.Document, param.URL)
		if err != nil {
			errors = append(errors, fmt.Errorf("%s error for %q: %v", title, param.URL, err))
		}

		evaluation = evaluateResult(evaluation, article.TextContent, param.Entry)
	}

	evaluation.Duration = time.Since(start)
	printEvaluationResult(title, evaluation)
	fmt.Println()

	return errors
}

func runDomDistiller(params []ExtractorParameter) []error {
	title := "dom distiller"
	start := time.Now()

	var errors []error
	var evaluation EvaluationResult
	for _, param := range params {
		var textResult string
		res, err := distiller.Apply(param.Document, &distiller.Options{
			OriginalURL:    param.URL,
			SkipPagination: true})
		if err != nil {
			errors = append(errors, fmt.Errorf("%s error for %q: %v", title, param.URL, err))
		} else {
			textResult = res.Text
		}

		evaluation = evaluateResult(evaluation, textResult, param.Entry)
	}

	evaluation.Duration = time.Since(start)
	printEvaluationResult(title, evaluation)
	fmt.Println()

	return errors
}

func evaluateResult(current EvaluationResult, result string, entry ComparisonEntry) EvaluationResult {
	var ev EvaluationResult

	// Report problematic entry
	if nWith := len(entry.With); nWith == 0 || nWith > 6 {
		log.Warn().Msgf("entry %s has %d with", entry.File, nWith)
	}

	if nWithout := len(entry.Without); nWithout == 0 || nWithout > 6 {
		log.Warn().Msgf("entry %s has %d without", entry.File, nWithout)
	}

	// Examine
	if result == "" {
		ev.FalseNegatives = len(entry.With)
		ev.TrueNegatives = len(entry.Without)
	} else {
		// Expected output
		for _, str := range entry.With {
			if strings.Contains(result, str) {
				ev.TruePositives++
			} else {
				ev.FalseNegatives++
			}
		}

		// Unwanted output
		for _, str := range entry.Without {
			if strings.Contains(result, str) {
				ev.FalsePositives++
			} else {
				ev.TrueNegatives++
			}
		}
	}

	// Merge with the current
	return EvaluationResult{
		TruePositives:  current.TruePositives + ev.TruePositives,
		FalseNegatives: current.FalseNegatives + ev.FalseNegatives,
		FalsePositives: current.FalsePositives + ev.FalsePositives,
		TrueNegatives:  current.TrueNegatives + ev.TrueNegatives,
	}
}

func printEvaluationResult(title string, ev EvaluationResult) {
	// Calculate performance
	tp := float64(ev.TruePositives)
	fn := float64(ev.FalseNegatives)
	fp := float64(ev.FalsePositives)
	tn := float64(ev.TrueNegatives)
	precision := tp / (tp + fp)
	recall := tp / (tp + fn)
	accuracy := (tp + tn) / (tp + tn + fp + fn)
	fScore := (2 * tp) / (2*tp + fp + fn)

	// Print data
	fmt.Println(strings.ToUpper(title))
	fmt.Printf("Duration  = %.3f second(s)\n", ev.Duration.Seconds())
	fmt.Printf("Precision = %.3f\n", precision)
	fmt.Printf("Recall    = %.3f\n", recall)
	fmt.Printf("Accuracy  = %.3f\n", accuracy)
	fmt.Printf("F Score   = %.3f\n", fScore)
}
