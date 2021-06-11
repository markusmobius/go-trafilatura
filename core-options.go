// This file is part of go-trafilatura, Go package for extracting readable
// content, comments and metadata from a web page. Source available in
// <https://github.com/markusmobius/go-trafilatura>.
// Copyright (C) 2021 Markus Mobius
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by the
// Free Software Foundation, either version 3 of the License, or (at your
// option) any later version.
//
// This program is distributed in the hope that it will be useful, but
// WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY
// or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License
// for more details.
//
// You should have received a copy of the GNU General Public License along
// with this program. If not, see <https://www.gnu.org/licenses/>.

// Code in this file is ported from <https://github.com/adbar/trafilatura>
// which available under GNU GPL v3 license.

package trafilatura

import (
	nurl "net/url"
	"time"
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

		MinExtractedSize:        200,
		MinExtractedCommentSize: 10,
		MinOutputSize:           10,
		MinOutputCommentSize:    10,
	}
}

// Options is configuration for the extractor.
type Options struct {
	// Config is the advanced configuration to fine tune the
	// extraction result. Keep it as nil to use default config.
	Config *Config

	// RecordID is optional ID for the extracted metadata.
	RecordID int64

	// ExtractionTime is optional data for the extracted metadata.
	ExtractionTime time.Time

	// OriginalURL is the original URL of the page. Might be overwritten by URL in metadata.
	OriginalURL *nurl.URL

	// PageCharset is the charset that used by the web page. If not specified, Trafilatura
	// will try to guess it by using charset value in <meta> tags.
	PageCharset string

	// TargetLanguage is ISO 639-1 language code to make the extractor only process web page that
	// uses the specified language.
	TargetLanguage string

	// NoFallback specify whether to skip fallback extractor using readability and dom-distiller.
	NoFallback bool

	// ExcludeComments specify whether to exclude comments from the extraction result.
	ExcludeComments bool

	// ExcludeTables specify whether to exclude information within the HTML <table> element.
	ExcludeTables bool

	// IncludeImages specify whether the extraction result will include images (experimental).
	IncludeImages bool

	// IncludeLinks specify whether the extraction result will include links along with their
	// targets (experimental).
	IncludeLinks bool

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
}
