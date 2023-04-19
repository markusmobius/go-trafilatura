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
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	nurl "net/url"
	"os"
	fp "path/filepath"
	"strings"
	"time"

	betree "github.com/beevik/etree"
	"github.com/markusmobius/go-trafilatura"
	gonanoid "github.com/matoous/go-nanoid/v2"

	"github.com/go-shiori/dom"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/net/html"
	"golang.org/x/sync/semaphore"
)

var feedTypes = sliceToMap(
	"application/atom+xml",
	"application/rdf+xml",
	"application/rss+xml",
	"application/x.atom+xml",
	"application/x-atom+xml",
	"text/atom+xml",
	"text/rdf+xml",
	"text/rss+xml",
	"text/xml",
)

func feedCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "feed [flags] [url]",
		Short: "Download and extract pages from a feed",
		Long: "Download and extract pages from list of urls that specified in a feed.\n" +
			"It supports both RSS and Atom feed. Trafilatura supports simple feed\n" +
			"finder, so you can point the url into an ordinary web page then\n" +
			"Trafilatura will attempt to find the feed.",
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			newFeedCmdHandler(cmd).run(args)
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

type feedCmdHandler struct {
	userAgent       string
	httpClient      *http.Client
	pagesDownloader *batchDownloader
	filterFunc      func(url *nurl.URL) bool
	urlOnly         bool
}

func newFeedCmdHandler(cmd *cobra.Command) *feedCmdHandler {
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

	// Prepare filter
	mapAllowedDomains := sliceToMap(allowedDomains...)
	mapExcludedDomains := sliceToMap(excludedDomains...)

	rxAllow, err := rxFromString(allowedPattern)
	if err != nil {
		logrus.Fatalf("filter pattern is not valid: %v", err)
	}

	rxExclude, err := rxFromString(excludedPattern)
	if err != nil {
		logrus.Fatalf("exclude pattern is not valid: %v", err)
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
	return &feedCmdHandler{
		userAgent:       userAgent,
		httpClient:      httpClient,
		pagesDownloader: pagesDownloader,
		filterFunc:      fnFilter,
		urlOnly:         urlOnly,
	}
}

func (fch *feedCmdHandler) run(args []string) {
	// Find feed page
	feedPage, err := fch.findFeedPage(args[0])
	if err != nil {
		logrus.Fatalf("failed to find feed: %v", err)
	}

	// Parse feed page
	pageURLs, err := fch.parseFeedPage(feedPage)
	if err != nil {
		logrus.Fatalf("failed to parse feed: %v", err)
	}
	logrus.Printf("found %d page URLs", len(pageURLs))

	// If user only want to print URLs, stop
	if fch.urlOnly {
		for _, url := range pageURLs {
			fmt.Println(url.String())
		}
		return
	}

	// Download and process pages concurrently
	err = fch.pagesDownloader.downloadURLs(context.Background(), pageURLs)
	if err != nil {
		logrus.Fatalf("download pages failed: %v", err)
	}
}

func (fch *feedCmdHandler) findFeedPage(baseURL string) (io.Reader, error) {
	// Make sure URL valid
	parsedBaseURL, valid := validateURL(baseURL)
	if !valid {
		return nil, fmt.Errorf("url is not valid")
	}

	// Prepare buffer for containing result
	buffer := bytes.NewBuffer(nil)
	var feedURL string

	// Process the URL
	err := func() error {
		// Downloading base URL
		logrus.Println("downloading", baseURL)
		resp, err := download(fch.httpClient, fch.userAgent, baseURL)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		// If it's XML, we got the feed so return it
		contentType := resp.Header.Get("Content-Type")
		if fch.contentIsFeed(contentType) {
			_, err = io.Copy(buffer, resp.Body)
			feedURL = baseURL
			return err
		}

		// If it's HTML, look for feed URL in document
		if !strings.Contains(contentType, "text/html") {
			return fmt.Errorf("page is not html: \"%s\"", contentType)
		}

		feedURL, err = fch.findFeedUrlInHtml(resp.Body, parsedBaseURL)
		return err
	}()

	if err != nil {
		return nil, fmt.Errorf("failed looking for feed: %v", err)
	}

	// If feed URL not found, give up
	if feedURL == "" {
		return nil, fmt.Errorf("feed not found")
	}

	// If feed URL found and buffer not empty, we are finished
	if feedURL != "" && buffer.Len() != 0 {
		return buffer, nil
	}

	// At this point, buffer is empty but feed URL is found, so download it.
	err = func() error {
		// Downloading feed URL
		logrus.Println("downloading feed", feedURL)
		resp, err := download(fch.httpClient, fch.userAgent, feedURL)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		// Fail if it's not XML
		contentType := resp.Header.Get("Content-Type")
		if !fch.contentIsFeed(contentType) {
			return fmt.Errorf("page is not feed: \"%s\"", contentType)
		}

		_, err = io.Copy(buffer, resp.Body)
		return err
	}()

	if err != nil {
		return nil, fmt.Errorf("download feed failed: %v", err)
	}

	if buffer.Len() == 0 {
		return nil, fmt.Errorf("feed not found")
	}

	return buffer, nil
}

func (fch *feedCmdHandler) parseFeedPage(r io.Reader) ([]*nurl.URL, error) {
	var pageURLs []*nurl.URL

	// Parse doc
	doc := betree.NewDocument()
	_, err := doc.ReadFrom(r)
	if err != nil {
		return nil, err
	}

	// Get feed items (for RSS)
	for _, item := range doc.FindElements("//item") {
		for _, link := range item.FindElements("./link") {
			parsedURL, valid := validateURL(link.Text())
			if valid {
				pageURLs = append(pageURLs, parsedURL)
				break
			}
		}
	}

	// Get feed entries (for Atom)
	for _, entry := range doc.FindElements("//entry") {
		for _, link := range entry.FindElements("./link") {
			href := link.SelectAttrValue("href", "")
			if href == "" {
				continue
			}

			parsedURL, valid := validateURL(href)
			if valid {
				pageURLs = append(pageURLs, parsedURL)
				break
			}
		}
	}

	// Make sure page URLs are unique
	uniquePageURLs := []*nurl.URL{}
	uniqueTracker := make(map[string]struct{})

	for _, url := range pageURLs {
		if fch.filterFunc != nil && !fch.filterFunc(url) {
			continue
		}

		strURL := url.String()
		if _, exist := uniqueTracker[strURL]; exist {
			continue
		}

		uniqueTracker[strURL] = struct{}{}
		uniquePageURLs = append(uniquePageURLs, url)
	}

	return uniquePageURLs, nil
}

func (fch *feedCmdHandler) findFeedUrlInHtml(r io.Reader, baseURL *nurl.URL) (string, error) {
	// Parse document
	doc, err := html.Parse(r)
	if err != nil {
		return "", err
	}

	// Look for feed URL in <link> and <a>
	for _, node := range dom.QuerySelectorAll(doc, `link, a`) {
		rel := dom.GetAttribute(node, "rel")
		if rel != "alternate" {
			continue
		}

		href := dom.GetAttribute(node, "href")
		if href == "" {
			continue
		}

		nodeType := dom.GetAttribute(node, "type")
		if fch.contentIsFeed(nodeType) {
			return createAbsoluteURL(href, baseURL), nil
		}
	}

	return "", nil
}

func (fch *feedCmdHandler) contentIsFeed(contentType string) bool {
	_, exist := feedTypes[contentType]
	return exist
}
