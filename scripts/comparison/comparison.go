package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	nurl "net/url"
	"os"
	fp "path/filepath"
	"strings"
	"time"

	"github.com/go-shiori/dom"
	"github.com/go-shiori/go-readability"
	distiller "github.com/markusmobius/go-domdistiller"
	"github.com/markusmobius/go-trafilatura"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/html"
)

func main() {
	var (
		nDocument             int
		evNothing             evaluationResult
		evEverything          evaluationResult
		evTrafilatura         evaluationResult
		evTrafilaturaFallback evaluationResult
		evReadability         evaluationResult
		evDomDistiller        evaluationResult
	)

	for strURL, entry := range comparisonData {
		// Make sure URL is valid
		url, err := nurl.ParseRequestURI(strURL)
		if err != nil {
			logrus.Errorf("failed to parse %s: %v", strURL, err)
			continue
		}

		// Open file
		f, err := os.Open(fp.Join("test-files", "comparison", entry.File))
		if err != nil {
			f, err = os.Open(fp.Join("test-files", "mock", entry.File))
			if err != nil {
				logrus.Errorf("failed to open %s: %v", entry.File, err)
				continue
			}
		}

		// Read all content of the file so it can be used multiple times
		fContent, err := ioutil.ReadAll(f)
		f.Close()
		if err != nil {
			logrus.Errorf("failed to read %s: %v", entry.File, err)
			continue
		}

		if len(fContent) == 0 {
			continue
		}

		// Create document
		doc, err := dom.Parse(bytes.NewReader(fContent))
		if err != nil {
			logrus.Errorf("failed to parse %s: %v", entry.File, err)
			continue
		}

		// Null hypotheses
		ev := evaluateResult("", entry)
		evNothing = mergeEvaluationResult(evNothing, ev)

		ev = evaluateResult(string(fContent), entry)
		evEverything = mergeEvaluationResult(evEverything, ev)

		// Readability
		start := time.Now()
		docReadability, result, err := runReadability(url, doc)
		if err != nil {
			logrus.Warnf("readability error in %s: %v", strURL, err)
		}

		duration := time.Now().Sub(start)
		ev = evaluateResult(result, entry)
		evReadability = mergeEvaluationResult(evReadability, ev)
		evReadability.Duration += duration

		// Dom Distiller
		start = time.Now()
		docDistiller, result, err := runDomDistiller(url, doc)
		if err != nil {
			logrus.Warnf("dom-distiller error in %s: %v", strURL, err)
		}

		duration = time.Now().Sub(start)
		ev = evaluateResult(result, entry)
		evDomDistiller = mergeEvaluationResult(evDomDistiller, ev)
		evDomDistiller.Duration += duration

		// Trafilatura
		start = time.Now()
		result, err = runTrafilatura(url, doc)
		if err != nil {
			logrus.Warnf("trafilatura error in %s: %v", strURL, err)
		}

		duration = time.Now().Sub(start)
		ev = evaluateResult(result, entry)
		evTrafilatura = mergeEvaluationResult(evTrafilatura, ev)
		evTrafilatura.Duration += duration

		// Trafilatura + fallback
		start = time.Now()
		result, err = runTrafilaturaFallback(url, doc, docReadability, docDistiller)
		if err != nil {
			logrus.Warnf("trafilatura+x error in %s: %v", strURL, err)
		}

		duration = time.Now().Sub(start)
		ev = evaluateResult(result, entry)
		evTrafilaturaFallback = mergeEvaluationResult(evTrafilaturaFallback, ev)
		evTrafilaturaFallback.Duration += duration

		// Counter
		nDocument++
	}

	fmt.Printf("N Documents: %d\n\n", nDocument)
	fmt.Printf("Nothing: %s\n\n", evNothing.info())

	fmt.Printf("Everything: %s\n", evEverything.info())
	fmt.Printf("\t%s\n\n", evEverything.scoreInfo())

	fmt.Printf("Readability: %s\n", evReadability.info())
	fmt.Printf("\t%s\n\n", evReadability.scoreInfo())

	fmt.Printf("DOM Distiller: %s\n", evDomDistiller.info())
	fmt.Printf("\t%s\n\n", evDomDistiller.scoreInfo())

	fmt.Printf("Trafilatura: %s\n", evTrafilatura.info())
	fmt.Printf("\t%s\n\n", evTrafilatura.scoreInfo())

	fmt.Printf("Trafilatura+Fallback: %s\n", evTrafilaturaFallback.info())
	fmt.Printf("\t%s\n\n", evTrafilaturaFallback.scoreInfo())
}

func evaluateResult(result string, entry comparisonEntry) evaluationResult {
	var ev evaluationResult

	// Report problematic entry
	if nWith := len(entry.With); nWith == 0 || nWith > 6 {
		logrus.Warnf("entry %s has %d with", entry.File, nWith)
	}

	if nWithout := len(entry.Without); nWithout == 0 || nWithout > 6 {
		logrus.Warnf("entry %s has %d without", entry.File, nWithout)
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

	return ev
}

func runTrafilatura(url *nurl.URL, doc *html.Node) (string, error) {
	result, err := trafilatura.ExtractDocument(doc, trafilatura.Options{
		OriginalURL:     url,
		NoFallback:      true,
		ExcludeComments: true,
	})

	if err != nil {
		return "", err
	}

	return result.ContentText, nil
}

func runTrafilaturaFallback(url *nurl.URL, doc *html.Node, fallbackCandidates ...*html.Node) (string, error) {
	result, err := trafilatura.ExtractDocument(doc, trafilatura.Options{
		OriginalURL:        url,
		NoFallback:         false,
		ExcludeComments:    true,
		FallbackCandidates: fallbackCandidates,
	})

	if err != nil {
		return "", err
	}

	return result.ContentText, nil
}

func runReadability(url *nurl.URL, doc *html.Node) (*html.Node, string, error) {
	article, err := readability.FromDocument(doc, url)
	if err != nil {
		return nil, "", err
	}

	return article.Node, article.TextContent, nil
}

func runDomDistiller(url *nurl.URL, doc *html.Node) (*html.Node, string, error) {
	res, err := distiller.Apply(doc, &distiller.Options{
		OriginalURL:    url,
		SkipPagination: true})
	if err != nil {
		return nil, "", err
	}

	return res.Node, res.Text, nil
}

type evaluationResult struct {
	TruePositives  int
	FalseNegatives int
	FalsePositives int
	TrueNegatives  int
	Duration       time.Duration
}

func mergeEvaluationResult(old, new evaluationResult) evaluationResult {
	old.TruePositives += new.TruePositives
	old.FalseNegatives += new.FalseNegatives
	old.FalsePositives += new.FalsePositives
	old.TrueNegatives += new.TrueNegatives

	return old
}

func (ev evaluationResult) info() string {
	str := fmt.Sprintf("TP=%d FN=%d FP=%d TN=%d",
		ev.TruePositives, ev.FalseNegatives,
		ev.FalsePositives, ev.TrueNegatives)

	if ev.Duration != 0 {
		str += fmt.Sprintf(" duration=%.3f s", ev.Duration.Seconds())
	}

	return str
}

func (ev evaluationResult) scoreInfo() string {
	precision, recall, accuracy, fScore := ev.score()
	return fmt.Sprintf("precision=%.3f recall=%.3f acc=%.3f f-score=%.3f",
		precision, recall, accuracy, fScore)
}

func (ev evaluationResult) score() (precision, recall, accuracy, fScore float64) {
	tp := float64(ev.TruePositives)
	fn := float64(ev.FalseNegatives)
	fp := float64(ev.FalsePositives)
	tn := float64(ev.TrueNegatives)

	precision = tp / (tp + fp)
	recall = tp / (tp + fn)
	accuracy = (tp + tn) / (tp + tn + fp + fn)
	fScore = (2 * tp) / (2*tp + fp + fn)
	return
}
