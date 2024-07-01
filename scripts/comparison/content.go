package main

import (
	"fmt"
	nurl "net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-shiori/dom"
	"github.com/go-shiori/go-readability"
	distiller "github.com/markusmobius/go-domdistiller"
	"github.com/markusmobius/go-trafilatura"
	"golang.org/x/net/html"
)

type ExtractorRunner func([]ExtractorParameter) (ExtractionPerformance, []error)

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
}

type ExtractionPerformance struct {
	Title     string
	Duration  time.Duration
	Precision float64
	Recall    float64
	Accuracy  float64
	FScore    float64
}

func compareContentExtraction() {
	// Prepare table
	tab := NewTable()
	tab.AddHeaders(
		"Extractor",
		"Duration (s)",
		"Precision",
		"Recall",
		"Accuracy",
		"F Score",
	)

	// Prepare extractor parameter
	params := prepareExtractorParameter()

	// Prepare extractors
	runners := []ExtractorRunner{
		prepareReadability(),                                 // Readability
		prepareDomDistiller(),                                // DOM Distiller
		prepareTrafilatura(false, trafilatura.Balanced),      // Standard Trafilatura
		prepareTrafilatura(true, trafilatura.Balanced),       // Trafilatura + Fallback
		prepareTrafilatura(true, trafilatura.FavorPrecision), // Trafilatura + Precision
		prepareTrafilatura(true, trafilatura.FavorRecall),    // Trafilatura + Recall
	}

	// Run extractors
	var errors []error
	for _, runner := range runners {
		// Run the runner
		perf, runnerErrors := runner(params)
		errors = append(errors, runnerErrors...)

		// Put performance to table
		tab.AddRow(
			perf.Title,
			str(perf.Duration.Seconds()),
			str(perf.Precision),
			str(perf.Recall),
			str(perf.Accuracy),
			str(perf.FScore),
		)
	}

	// Print errors
	for _, err := range errors {
		log.Warn().Err(err)
	}

	// Print documents count
	fmt.Printf("Number of documents: %d\n", len(params))

	// Print table
	tab.Print()
}

func prepareExtractorParameter() []ExtractorParameter {
	log.Info().Msg("prepare parameters")

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
			log.Error().Msgf("%v", err)
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

func prepareTrafilatura(useFallback bool, focus trafilatura.ExtractionFocus) ExtractorRunner {
	return func(params []ExtractorParameter) (ExtractionPerformance, []error) {
		// Prepare title
		titles := []string{"Trafilatura"}
		if useFallback {
			titles = append(titles, "Fallback")
		}

		if focus == trafilatura.FavorPrecision {
			titles = append(titles, "Precision")
		} else if focus == trafilatura.FavorRecall {
			titles = append(titles, "Recall")
		}

		title := strings.Join(titles, " + ")

		// Print log
		log.Info().Msgf("running %s", title)

		// Prepare Trafilatura options
		opts := trafilatura.Options{
			EnableFallback:  useFallback,
			ExcludeComments: true,
			ExcludeTables:   false,
			Focus:           focus,
		}

		// Initiate extraction result
		start := time.Now()
		var errors []error
		var evaluation EvaluationResult

		// Process each parameter
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

		// Return the performance
		perf := calculatePerformance(title, time.Since(start), evaluation)
		return perf, errors
	}
}

func prepareReadability() ExtractorRunner {
	return func(params []ExtractorParameter) (ExtractionPerformance, []error) {
		start := time.Now()
		title := "Readability"
		log.Info().Msgf("running %s", title)

		var errors []error
		var evaluation EvaluationResult
		for _, param := range params {
			article, err := readability.FromDocument(param.Document, param.URL)
			if err != nil {
				errors = append(errors, fmt.Errorf("%s error for %q: %v", title, param.URL, err))
			}

			evaluation = evaluateResult(evaluation, article.TextContent, param.Entry)
		}

		perf := calculatePerformance(title, time.Since(start), evaluation)
		return perf, errors
	}
}

func prepareDomDistiller() ExtractorRunner {
	return func(params []ExtractorParameter) (ExtractionPerformance, []error) {
		start := time.Now()
		title := "Dom Distiller"
		log.Info().Msgf("running %s", title)

		var errors []error
		var evaluation EvaluationResult
		for _, param := range params {
			var textResult string
			res, err := distiller.Apply(param.Document, &distiller.Options{
				OriginalURL:    param.URL,
				SkipPagination: true,
			})

			if err != nil {
				errors = append(errors, fmt.Errorf("%s error for %q: %v", title, param.URL, err))
			} else {
				textResult = res.Text
			}

			evaluation = evaluateResult(evaluation, textResult, param.Entry)
		}

		perf := calculatePerformance(title, time.Since(start), evaluation)
		return perf, errors
	}
}

func evaluateResult(ev EvaluationResult, result string, entry ComparisonEntry) EvaluationResult {
	// Report problematic entry
	if nWith := len(entry.With); nWith == 0 || nWith > 6 {
		log.Warn().Msgf("entry %s has %d with", entry.File, nWith)
	}

	if nWithout := len(entry.Without); nWithout == 0 || nWithout > 6 {
		log.Warn().Msgf("entry %s has %d without", entry.File, nWithout)
	}

	// If result empty, return early
	if result == "" {
		ev.FalseNegatives += len(entry.With)
		ev.TrueNegatives += len(entry.Without)
		return ev
	}

	// Check expected output
	for _, str := range entry.With {
		if strings.Contains(result, str) {
			ev.TruePositives++
		} else {
			ev.FalseNegatives++
		}
	}

	// Check unwanted output
	for _, str := range entry.Without {
		if strings.Contains(result, str) {
			ev.FalsePositives++
		} else {
			ev.TrueNegatives++
		}
	}

	return ev
}

func calculatePerformance(title string, duration time.Duration, ev EvaluationResult) ExtractionPerformance {
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
	return ExtractionPerformance{
		Title:     title,
		Duration:  duration,
		Precision: precision,
		Recall:    recall,
		Accuracy:  accuracy,
		FScore:    fScore,
	}
}

func str(val float64) string {
	return strconv.FormatFloat(val, 'f', 3, 64)
}
