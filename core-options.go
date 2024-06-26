// This file is part of go-trafilatura, Go package for extracting readable
// content, comments and metadata from a web page. Source available in
// <https://github.com/markusmobius/go-trafilatura>.
//
// Copyright (C) 2021 Markus Mobius
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code in this file is ported from <https://github.com/adbar/trafilatura>
// which available under Apache 2.0 license.

package trafilatura

import (
	nurl "net/url"

	"github.com/markusmobius/go-htmldate"
	"golang.org/x/net/html"
)

// Config is advanced setting to fine tune the extraction result.
// You can use it to specify the minimal size of the extracted content
// and how many duplicate text allowed. However, for most of the time
// the default config should be good enough.
type Config struct {
	// Deduplication config
	CacheSize             int
	MaxDuplicateCount     int
	MinDuplicateCheckSize int

	// Extraction size setting
	MinExtractedSize        int
	MinExtractedCommentSize int
	MinOutputSize           int
	MinOutputCommentSize    int
}

// DefaultConfig returns the default configuration value.
func DefaultConfig() *Config {
	return &Config{
		CacheSize:             4096,
		MinDuplicateCheckSize: 100,
		MaxDuplicateCount:     2,

		MinExtractedSize:        250,
		MinExtractedCommentSize: 1,
		MinOutputSize:           1,
		MinOutputCommentSize:    1,
	}
}

// FallbackCandidates allows to specify a list of fallback candidates
// in particular: readability and domdistiller
type FallbackConfig struct {
	//readability
	HasReadability      bool
	ReadabilityFallback *html.Node
	HasDistiller        bool
	DistillerFallback   *html.Node
	//other fallbacks are possible as well: if set the above four settings are ignored
	OtherFallbacks []*html.Node
}

// Options is configuration for the extractor.
type Options struct {
	// Config is the advanced configuration to fine tune the
	// extraction result. Keep it as nil to use default config.
	Config *Config

	// OriginalURL is the original URL of the page. Might be overwritten by URL in metadata.
	OriginalURL *nurl.URL

	// TargetLanguage is ISO 639-1 language code to make the extractor only process web page that
	// uses the specified language.
	TargetLanguage string

	// If FallbackCandidates is nil then no fallback will be performed`
	// Otherwise: readability and domdistiller fallbacks will be used if precalculated
	// OtherFallbacks!=nil will ensure that this list is used (rather than readability/distiller)
	FallbackCandidates *FallbackConfig

	// FavorPrecision specify whether to prefer less text but correct extraction.
	FavorPrecision bool

	// FavorRecall specify whether to prefer more text even when unsure.
	FavorRecall bool

	// ExcludeComments specify whether to exclude comments from the extraction result.
	ExcludeComments bool

	// ExcludeTables specify whether to exclude information within the HTML <table> element.
	ExcludeTables bool

	// IncludeImages specify whether the extraction result will include images (experimental).
	IncludeImages bool

	// IncludeLinks specify whether the extraction result will include links along with their
	// targets (experimental).
	IncludeLinks bool

	// BlacklistedAuthors is list of author names to be excluded from extraction result.
	BlacklistedAuthors []string

	// Deduplicate specify whether to remove duplicate segments and sections.
	Deduplicate bool

	// HasEssentialMetadata make the extractor only keep documents featuring all essential
	// metadata (date, title, url).
	HasEssentialMetadata bool

	// MaxTreeSize specify max number of elements inside a document.
	// Document that surpass this value will be discarded.
	MaxTreeSize int

	// EnableLog specify whether log should be enabled or not.
	EnableLog bool

	// HtmlDateOverride is user provided extracted date from `go-htmldate` package.
	HtmlDateOverride *htmldate.Result

	// HtmlDateOptions is user provided configuration for the external `go-htmldate`
	// package that used to look for publish date of a web page.
	HtmlDateOptions *htmldate.Options

	// PruneSelector is the CSS selector to select nodes to be pruned before extraction.
	PruneSelector string
}
