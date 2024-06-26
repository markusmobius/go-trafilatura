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

package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"mime"
	"net/http"
	nurl "net/url"
	"os"
	fp "path/filepath"
	"strings"
	"time"

	"github.com/markusmobius/go-trafilatura"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"golang.org/x/net/html"
)

var log zerolog.Logger

func init() {
	log = zerolog.New(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: "2006-01-02 15:04",
	}).With().Timestamp().Logger().Level(zerolog.Disabled)
}

// defaultUserAgent is the default user agent to use, which is Firefox's.
const defaultUserAgent = "Mozilla/5.0 (X11; Linux x86_64; rv:88.0) Gecko/20100101 Firefox/88.0"

func main() {
	// Create root command
	rootCmd := &cobra.Command{
		Use:   "go-trafilatura [flags] [source]",
		Run:   rootCmdHandler,
		Short: "extract readable content from a HTML file or url",
		Long: "Extract readable content from a specified source which can be either a HTML file or url.\n" +
			"It also has supports for batch download url either from a file which contains list of url,\n" +
			"RSS feeds and sitemap.",
		Args: cobra.ExactArgs(1),
	}

	// Register persistent flags
	flags := rootCmd.PersistentFlags()
	flags.StringP("format", "f", "", "output format for the extract result, either 'html' (default), 'txt' or 'json'")
	flags.StringP("language", "l", "", "target language (ISO 639-1 codes)")
	flags.Bool("no-fallback", false, "disable fallback extraction using readability and dom-distiller")
	flags.Bool("no-comments", false, "exclude comments  extraction result")
	flags.Bool("no-tables", false, "include tables in extraction result")
	flags.Bool("images", false, "include images in extraction result (experimental)")
	flags.Bool("links", false, "keep links in extraction result (experimental)")
	flags.Bool("deduplicate", false, "filter out duplicate segments and sections")
	flags.Bool("has-metadata", false, "only output documents with title, URL and date")
	flags.BoolP("verbose", "v", false, "enable log message")
	flags.IntP("timeout", "t", 30, "timeout for downloading web page in seconds")
	flags.Bool("skip-tls", false, "skip X.509 (TLS) certificate verification")
	flags.StringP("user-agent", "u", defaultUserAgent, "set custom user agent")

	// Add sub commands
	rootCmd.AddCommand(batchCmd(), sitemapCmd(), feedCmd())

	// Execute
	err := rootCmd.Execute()
	if err != nil {
		log.Fatal().Err(err)
	}
}

func rootCmdHandler(cmd *cobra.Command, args []string) {
	// Process source
	source := args[0]
	opts := createExtractorOptions(cmd)
	httpClient := createHttpClient(cmd)
	userAgent, _ := cmd.Flags().GetString("user-agent")

	var err error
	var result *trafilatura.ExtractResult

	switch {
	case fileExists(source):
		result, err = processFile(source, opts)
	case isValidURL(source):
		parsedURL, _ := nurl.ParseRequestURI(source)
		result, err = processURL(httpClient, userAgent, parsedURL, opts)
	}

	if err != nil {
		log.Fatal().Msgf("failed to extract %s: %v", source, err)
	}

	if result == nil {
		log.Fatal().Msgf("failed to extract %s: no readable content", source)
	}

	// Print result
	err = writeOutput(os.Stdout, result, cmd)
	if err != nil {
		log.Fatal().Msgf("failed to write output: %v", err)
	}
}

func processFile(path string, opts trafilatura.Options) (*trafilatura.ExtractResult, error) {
	// Open file
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Make sure it's html
	var fReader io.Reader
	mimeType := mime.TypeByExtension(fp.Ext(path))
	if strings.Contains(mimeType, "text/html") {
		fReader = f
	} else {
		buffer := bytes.NewBuffer(nil)
		tee := io.TeeReader(f, buffer)

		_, err := html.Parse(tee)
		if err != nil {
			return nil, fmt.Errorf("%s is not a valid html file: %v", path, err)
		}

		fReader = buffer
	}

	// Extract
	result, err := trafilatura.Extract(fReader, opts)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func processURL(client *http.Client, userAgent string, url *nurl.URL, opts trafilatura.Options) (*trafilatura.ExtractResult, error) {
	// Download URL
	strURL := url.String()
	log.Info().Msgf("downloading %q", strURL)

	resp, err := download(client, userAgent, strURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Make sure it's html
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		return nil, fmt.Errorf("page is not html: \"%s\"", contentType)
	}

	// Extract
	opts.OriginalURL = url
	result, err := trafilatura.Extract(resp.Body, opts)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func createExtractorOptions(cmd *cobra.Command) trafilatura.Options {
	var opts trafilatura.Options

	flags := cmd.Flags()
	NoFallback, _ := flags.GetBool("no-fallback")
	if !NoFallback {
		opts.FallbackCandidates = &trafilatura.FallbackConfig{}
	}
	opts.TargetLanguage, _ = flags.GetString("language")
	opts.ExcludeComments, _ = flags.GetBool("no-comments")
	opts.ExcludeTables, _ = flags.GetBool("no-tables")
	opts.IncludeImages, _ = flags.GetBool("images")
	opts.IncludeLinks, _ = flags.GetBool("links")
	opts.Deduplicate, _ = flags.GetBool("deduplicate")
	opts.HasEssentialMetadata, _ = flags.GetBool("has-metadata")
	opts.EnableLog, _ = flags.GetBool("verbose")
	return opts
}

func createHttpClient(cmd *cobra.Command) *http.Client {
	flags := cmd.Flags()
	timeout, _ := flags.GetInt("timeout")
	skipTls, _ := flags.GetBool("skip-tls")

	return &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: skipTls,
			},
		},
	}
}

func download(client *http.Client, userAgent string, url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
