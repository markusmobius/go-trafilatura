package main

import (
	"fmt"
	nurl "net/url"
	"strings"
	"time"

	"github.com/markusmobius/go-trafilatura"
	"github.com/sirupsen/logrus"
)

func compareAuthorExtraction() {
	var nDocument int
	var nCorrect int
	start := time.Now()

	comparisonData = map[string]comparisonEntry{
		"https://aoc.media/opinion/2019/12/09/pour-le-neoliberalisme-la-retraite-est-un-archaisme/": {
			File:        "aoc.media.archaisme.html",
			Title:       "Pour le néolibéralisme, la retraite est un archaïsme",
			Date:        "2019-12-10",
			Description: "Pour le néolibéralisme, la retraite ne peut être qu’un archaïsme, une sorte de déviance inadaptée, qui nous fait prendre du retard dans la compétition mondiale, et dont l’État lui-même doit programmer la disparition progressive. L’affrontement qui se met en place ces jours-ci dépasse donc les questions techniques de « réforme systémique » ou d’« ajustement paramétrique » dont nous parle le jargon des experts. Il oppose, bien plus profondément, deux visions incompatibles de l’avenir du vivant et de nos rythmes de vie.",
			Authors:     []string{"Barbara Stiegler"},
			With:        []string{"Pour le néolibéralisme, la retraite", "les grandes grèves de 1995 furent", "Pour réaliser ce programme, il impose"},
			Without:     []string{"Pour lire la suite", "Pour accéder en illimité", "Pour rester informé inscrivez-vous à la newsletter"},
		},
	}

	for strURL, entry := range comparisonData {
		// Make sure entry valid
		if entry.File == "" || len(entry.Authors) == 0 {
			continue
		}

		// Make sure URL is valid
		url, err := nurl.ParseRequestURI(strURL)
		if err != nil {
			logrus.Errorf("failed to parse %s: %v", strURL, err)
			continue
		}

		// Open file
		f, err := openDataFile(entry.File)
		if err != nil {
			logrus.Error(err)
			continue
		}

		// Run trafilatura
		result, _ := trafilatura.Extract(f, trafilatura.Options{
			OriginalURL: url,
			NoFallback:  true,
			EnableLog:   true,
		})

		// Compare result
		nDocument++
		if result != nil {
			if strings.Join(entry.Authors, "; ") == result.Metadata.Author {
				nCorrect++
				fmt.Printf("%s\t%s\n", entry.File, result.Metadata.Author)
			} else {
			}
		}
	}

	// Print result
	fmt.Printf("Duration:   %v\n", time.Since(start))
	fmt.Printf("N document: %d\n", nDocument)
	fmt.Printf("N correct:  %d\n", nCorrect)
}
