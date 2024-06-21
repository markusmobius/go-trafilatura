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

package main

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	nurl "net/url"
	"os"
	fp "path/filepath"
	"strings"
	"time"

	"github.com/markusmobius/go-trafilatura"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/spf13/cobra"
	"golang.org/x/sync/semaphore"
)

func sitemapCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sitemap [flags] [url]",
		Short: "Download and extract pages from a sitemap",
		Long: "Download and extract pages from list of urls that specified in a sitemap.\n" +
			"Trafilatura supports simple sitemap finder, so you can point the url into\n" +
			"an ordinary web page then Trafilatura will attempt to find the sitemap.",
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			newSitemapCmdHandler(cmd).run(args)
		},
	}

	flags := cmd.Flags()
	flags.StringP("output", "o", ".", "output directory for the result (default current work dir)")
	flags.Int("parallel", 10, "number of concurrent download at a time (default 10)")
	flags.Int("delay", 0, "delay between each url download in seconds (default 0)")
	flags.String("filter", "", "regular expression for allowed url")
	flags.String("exclude", "", "regular expression for excluded url")
	flags.StringArray("domains", nil, "list of allowed domains")
	flags.StringArray("no-domains", nil, "list of excluded domains")
	flags.Bool("url-only", false, "only print page urls without downloading or processing them")

	return cmd
}

type sitemapCmdHandler struct {
	userAgent         string
	httpClient        *http.Client
	sitemapDownloader *sitemapDownloader
	pagesDownloader   *batchDownloader
	urlOnly           bool
}

func newSitemapCmdHandler(cmd *cobra.Command) *sitemapCmdHandler {
	// Parse flags
	flags := cmd.Flags()
	delay, _ := flags.GetInt("delay")
	nThread, _ := flags.GetInt("parallel")
	allowedPattern, _ := flags.GetString("filter")
	excludedPattern, _ := flags.GetString("exclude")
	allowedDomains, _ := flags.GetStringArray("domains")
	excludedDomains, _ := flags.GetStringArray("no-domains")
	outputDir, _ := flags.GetString("output")
	urlOnly, _ := flags.GetBool("url-only")
	userAgent, _ := cmd.Flags().GetString("user-agent")

	// Prepare http client
	httpClient := createHttpClient(cmd)

	// Prepare sitemap downloader
	mapAllowedDomains := sliceToMap(allowedDomains...)
	mapExcludedDomains := sliceToMap(excludedDomains...)

	rxAllow, err := rxFromString(allowedPattern)
	if err != nil {
		log.Fatal().Msgf("filter pattern is not valid: %v", err)
	}

	rxExclude, err := rxFromString(excludedPattern)
	if err != nil {
		log.Fatal().Msgf("exclude pattern is not valid: %v", err)
	}

	fnFilter := func(url *nurl.URL) bool {
		strURL := url.String()
		domainName := url.Hostname()
		_, allowed := mapAllowedDomains[domainName]
		_, excluded := mapExcludedDomains[domainName]

		switch {
		case len(mapExcludedDomains) > 0 && excluded,
			len(mapAllowedDomains) > 0 && !allowed,
			rxExclude != nil && rxExclude.MatchString(strURL),
			rxAllow != nil && !rxAllow.MatchString(strURL):
			return false
		}

		return true
	}

	sDownloader := &sitemapDownloader{
		cache:      make(map[string]struct{}),
		httpClient: httpClient,
		userAgent:  userAgent,
		filterFunc: fnFilter,
		delay:      time.Duration(delay) * time.Second,
		semaphore:  semaphore.NewWeighted(int64(nThread)),
	}

	// Prepare pages downloader
	nameExt := outputExt(cmd)
	fnWrite := func(result *trafilatura.ExtractResult, url *nurl.URL, idx int) error {
		name := nameFromURL(url)
		timestamp := time.Now().Format("150405")
		id, err := gonanoid.New(6)
		if err != nil {
			return err
		}

		name = timestamp + "-" + name + "-" + id + nameExt
		dst, err := os.Create(fp.Join(outputDir, name))
		if err != nil {
			return err
		}
		defer dst.Close()

		return writeOutput(dst, result, cmd)
	}

	pagesDownloader := &batchDownloader{
		userAgent:      userAgent,
		httpClient:     httpClient,
		extractOptions: createExtractorOptions(cmd),
		semaphore:      semaphore.NewWeighted(int64(nThread)),
		delay:          time.Duration(delay) * time.Second,
		cancelOnError:  false,
		writeFunc:      fnWrite,
	}

	// Make sure output dir exist
	os.MkdirAll(outputDir, os.ModePerm)

	// Return handler
	return &sitemapCmdHandler{
		userAgent:         userAgent,
		httpClient:        httpClient,
		sitemapDownloader: sDownloader,
		pagesDownloader:   pagesDownloader,
		urlOnly:           urlOnly,
	}
}

func (sch *sitemapCmdHandler) run(args []string) {
	// Find sitemap URL
	sitemapURLs, err := sch.findSitemapURLs(args[0])
	if err != nil {
		log.Fatal().Msgf("failed to find sitemap: %v", err)
	}

	// Download all sitemaps recursively, concurrently
	ctx := context.Background()
	pageURLs := sch.sitemapDownloader.downloadURLs(ctx, sitemapURLs)
	log.Info().Msgf("found %d page URLs", len(pageURLs))

	// If user only want to print URLs, stop
	if sch.urlOnly {
		for _, url := range pageURLs {
			fmt.Println(url.String())
		}
		return
	}

	// Download and process pages concurrently
	err = sch.pagesDownloader.downloadURLs(ctx, pageURLs)
	if err != nil {
		log.Fatal().Msgf("download pages failed: %v", err)
	}
}

func (sch *sitemapCmdHandler) findSitemapURLs(baseURL string) ([]*nurl.URL, error) {
	// Make sure URL valid
	if !isValidURL(baseURL) {
		return nil, fmt.Errorf("url is not valid")
	}

	// Parse URL
	parsedURL, _ := nurl.ParseRequestURI(baseURL)

	// If it already looks like a sitemap URL, return
	if strings.HasSuffix(baseURL, ".xml") ||
		strings.HasSuffix(baseURL, "sitemap") {
		return []*nurl.URL{parsedURL}, nil
	}

	// If not found, try to check in robots.txt.
	// Here we'll ignore error since it's possible that a site doesn't have robots.txt
	parsedURL.Path = "/robots.txt"
	parsedURL.RawQuery = nurl.Values{}.Encode()
	parsedURL.Fragment = ""

	sitemapURLs, err := sch.findSitemapURLsInRobots(parsedURL.String())
	if err != nil {
		log.Warn().Msgf("failed to look in robots.txt: %v", err)
	}

	// If there are no sitemap found, just add the default path.
	if len(sitemapURLs) == 0 {
		parsedURL.Path = "/sitemap.xml"
		sitemapURLs = append(sitemapURLs, parsedURL)
	}

	return sitemapURLs, nil
}

func (sch *sitemapCmdHandler) findSitemapURLsInRobots(robotsURL string) ([]*nurl.URL, error) {
	// Download URL
	log.Info().Msgf("downloading robots.txt: %q", robotsURL)
	resp, err := download(sch.httpClient, sch.userAgent, robotsURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Make sure it's text file
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/plain") {
		return nil, fmt.Errorf("%s is not plain text", robotsURL)
	}

	// Scan and find sitemap
	sitemapURLs := []*nurl.URL{}
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		line = strings.ToLower(line)

		if strings.HasPrefix(line, "sitemap:") {
			line = strings.TrimPrefix(line, "sitemap:")
			line = strings.TrimSpace(line)

			parsedURL, valid := validateURL(line)
			if valid {
				sitemapURLs = append(sitemapURLs, parsedURL)
			}
		}
	}

	// Return
	if len(sitemapURLs) == 0 {
		return nil, fmt.Errorf("sitemap url not found")
	}

	return sitemapURLs, nil
}
