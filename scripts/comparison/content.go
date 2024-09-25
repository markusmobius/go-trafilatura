package main

import (
	"context"
	"fmt"
	nurl "net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-shiori/dom"
	"github.com/go-shiori/go-readability"
	distiller "github.com/markusmobius/go-domdistiller"
	gt "github.com/markusmobius/go-trafilatura"
	"golang.org/x/net/html"
	"golang.org/x/sync/semaphore"
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

func compareContentExtraction(nWorker int) {
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
		prepareReadability(nWorker),                             // Readability
		prepareDomDistiller(nWorker, -1),                        // DOM Distiller
		prepareDomDistiller(nWorker, int(distiller.PrevNext)),   // DOM Distiller + Pagination PrevNext
		prepareDomDistiller(nWorker, int(distiller.PageNumber)), // DOM Distiller + Pagination PageNumber
		prepareTrafilatura(nWorker, false, gt.Balanced),         // Standard Trafilatura
		prepareTrafilatura(nWorker, true, gt.Balanced),          // Trafilatura + Fallback
		prepareTrafilatura(nWorker, true, gt.FavorPrecision),    // Trafilatura + Precision
		prepareTrafilatura(nWorker, true, gt.FavorRecall),       // Trafilatura + Recall
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

func prepareTrafilatura(nWorker int, useFallback bool, focus gt.ExtractionFocus) ExtractorRunner {
	return func(params []ExtractorParameter) (ExtractionPerformance, []error) {
		// Prepare title
		title := "Trafilatura"

		if useFallback {
			title += " + Fallback"
		}

		switch focus {
		case gt.Balanced:
			title += " + Balanced"
		case gt.FavorPrecision:
			title += " + Favor Precision"
		case gt.FavorRecall:
			title += " + Favor Recall"
		}

		// Print log
		log.Info().Msgf("running %s", title)

		// Initiate extraction result
		var mu sync.Mutex
		var errors []error
		var evaluation EvaluationResult

		// Prepare wait group
		var wg sync.WaitGroup
		ctx := context.TODO()
		sem := semaphore.NewWeighted(int64(nWorker))

		// Process each parameter
		start := time.Now()
		for _, param := range params {
			// Acquire semaphore and wait group
			if err := sem.Acquire(ctx, 1); err != nil {
				log.Fatal().Msgf("failed to acquire semaphore: %v", err)
			}
			wg.Add(1)

			go func(param ExtractorParameter) {
				// Make sure to release
				defer func() {
					wg.Done()
					sem.Release(1)
				}()

				// Extract document
				result, err := gt.ExtractDocument(param.Document, gt.Options{
					EnableFallback:  useFallback,
					ExcludeComments: true,
					ExcludeTables:   false,
					Focus:           focus,
					OriginalURL:     param.URL,
				})

				// Evaluate the result
				var ev EvaluationResult
				if err == nil {
					ev = evaluateEntry(param.Entry, result.ContentText)
				}

				// Temporarily lock resource
				mu.Lock()
				defer mu.Unlock()

				evaluation = applyEvaluation(evaluation, ev)
				if err != nil {
					errors = append(errors, fmt.Errorf("%s error for %q: %v", title, param.URL, err))
				}
			}(param)
		}

		// Return the performance
		wg.Wait()
		perf := calculatePerformance(title, time.Since(start), evaluation)
		return perf, errors
	}
}

func prepareReadability(nWorker int) ExtractorRunner {
	return func(params []ExtractorParameter) (ExtractionPerformance, []error) {
		title := "Readability"
		log.Info().Msgf("running %s", title)

		// Initiate extraction result
		var mu sync.Mutex
		var errors []error
		var evaluation EvaluationResult

		// Prepare wait group
		var wg sync.WaitGroup
		ctx := context.TODO()
		sem := semaphore.NewWeighted(int64(nWorker))

		// Process each parameter
		start := time.Now()
		for _, param := range params {
			// Acquire semaphore and wait group
			if err := sem.Acquire(ctx, 1); err != nil {
				log.Fatal().Msgf("failed to acquire semaphore: %v", err)
			}
			wg.Add(1)

			go func(param ExtractorParameter) {
				// Make sure to release
				defer func() {
					wg.Done()
					sem.Release(1)
				}()

				// Extract document
				article, err := readability.FromDocument(param.Document, param.URL)

				// Evaluate the result
				var ev EvaluationResult
				if err == nil {
					ev = evaluateEntry(param.Entry, article.TextContent)
				}

				// Temporarily lock resource
				mu.Lock()
				defer mu.Unlock()

				evaluation = applyEvaluation(evaluation, ev)
				if err != nil {
					errors = append(errors, fmt.Errorf("%s error for %q: %v", title, param.URL, err))
				}
			}(param)
		}

		// Return the performance
		wg.Wait()
		perf := calculatePerformance(title, time.Since(start), evaluation)
		return perf, errors
	}
}

func prepareDomDistiller(nWorker int, paginationAlgo int) ExtractorRunner {
	return func(params []ExtractorParameter) (ExtractionPerformance, []error) {
		title := "Dom Distiller"
		if paginationAlgo == int(distiller.PrevNext) {
			title += " + Pagination PrevNext"
		} else if paginationAlgo == int(distiller.PageNumber) {
			title += " + Pagination PageNumber"
		}

		log.Info().Msgf("running %s", title)

		// Initiate extraction result
		var mu sync.Mutex
		var errors []error
		var evaluation EvaluationResult

		// Prepare wait group
		var wg sync.WaitGroup
		ctx := context.TODO()
		sem := semaphore.NewWeighted(int64(nWorker))

		// Process each parameter
		start := time.Now()
		for _, param := range params {
			// Acquire semaphore and wait group
			if err := sem.Acquire(ctx, 1); err != nil {
				log.Fatal().Msgf("failed to acquire semaphore: %v", err)
			}
			wg.Add(1)

			go func(param ExtractorParameter) {
				// Make sure to release
				defer func() {
					wg.Done()
					sem.Release(1)
				}()

				// Extract document
				res, err := distiller.Apply(param.Document, &distiller.Options{
					OriginalURL:    param.URL,
					SkipPagination: paginationAlgo < 0,
					PaginationAlgo: distiller.PaginationAlgo(paginationAlgo),
				})

				// Evaluate the result
				var ev EvaluationResult
				if err == nil {
					ev = evaluateEntry(param.Entry, res.Text)
				}

				// Temporarily lock resource
				mu.Lock()
				defer mu.Unlock()

				evaluation = applyEvaluation(evaluation, ev)
				if err != nil {
					errors = append(errors, fmt.Errorf("%s error for %q: %v", title, param.URL, err))
				}
			}(param)
		}

		// Return the performance
		wg.Wait()
		perf := calculatePerformance(title, time.Since(start), evaluation)
		return perf, errors
	}
}

func evaluateEntry(entry ComparisonEntry, result string) EvaluationResult {
	// Report problematic entry
	if nWith := len(entry.With); nWith == 0 || nWith > 6 {
		log.Warn().Msgf("entry %s has %d with", entry.File, nWith)
	}

	if nWithout := len(entry.Without); nWithout == 0 || nWithout > 6 {
		log.Warn().Msgf("entry %s has %d without", entry.File, nWithout)
	}

	// If result empty, return early
	var ev EvaluationResult

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

func applyEvaluation(current, new EvaluationResult) EvaluationResult {
	current.TruePositives += new.TruePositives
	current.FalseNegatives += new.FalseNegatives
	current.FalsePositives += new.FalsePositives
	current.TrueNegatives += new.TrueNegatives
	return current
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
