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

const (
	mimeRSS  = "application/rss+xml"
	mimeAtom = "application/atom+xml"
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
		Run:  feedCmdHandler,
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

func feedCmdHandler(cmd *cobra.Command, args []string) {
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

	// Find feed page
	httpClient := createHttpClient(cmd)
	feedPage, err := findFeedPage(httpClient, args[0])
	if err != nil {
		logrus.Fatalf("failed to find feed: %v", err)
	}

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

	// Parse feed page
	pageURLs, err := parseFeedPage(feedPage, fnFilter)
	if err != nil {
		logrus.Fatalf("failed to parse feed: %v", err)
	}

	logrus.Printf("found %d page URLs", len(pageURLs))

	// If user only want to print URLs, stop
	if urlOnly {
		for _, url := range pageURLs {
			fmt.Println(url.String())
		}
		return
	}

	// Make sure output dir exist
	os.MkdirAll(outputDir, os.ModePerm)

	// Download and process pages concurrently
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

	err = (&batchDownloader{
		httpClient:     createHttpClient(cmd),
		extractOptions: createExtractorOptions(cmd),
		semaphore:      semaphore.NewWeighted(int64(nThread)),
		delay:          time.Duration(delay) * time.Second,
		cancelOnError:  false,
		writeFunc:      fnWrite,
	}).downloadURLs(context.Background(), pageURLs)

	if err != nil {
		logrus.Fatalf("download pages failed: %v", err)
	}
}

func findFeedPage(client *http.Client, baseURL string) (io.Reader, error) {
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
		resp, err := client.Get(baseURL)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		// If it's XML, we got the feed so return it
		contentType := resp.Header.Get("Content-Type")
		if contentIsFeed(contentType) {
			_, err = io.Copy(buffer, resp.Body)
			feedURL = baseURL
			return err
		}

		// If it's HTML, look for feed URL in document
		if !strings.Contains(contentType, "text/html") {
			return fmt.Errorf("page is not html: \"%s\"", contentType)
		}

		feedURL, err = findFeedUrlInHtml(resp.Body, parsedBaseURL)
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
		resp, err := client.Get(feedURL)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		// Fail if it's not XML
		contentType := resp.Header.Get("Content-Type")
		if !contentIsFeed(contentType) {
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

func parseFeedPage(r io.Reader, filterFunc func(*nurl.URL) bool) ([]*nurl.URL, error) {
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
		if filterFunc != nil && !filterFunc(url) {
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

func findFeedUrlInHtml(r io.Reader, baseURL *nurl.URL) (string, error) {
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
		if strings.Contains(nodeType, mimeRSS) || strings.Contains(nodeType, mimeAtom) {
			return createAbsoluteURL(href, baseURL), nil
		}
	}

	return "", nil
}

func contentIsFeed(contentType string) bool {
	switch {
	case strings.Contains(contentType, "text/xml"),
		strings.Contains(contentType, "application/xml"),
		strings.Contains(contentType, mimeRSS),
		strings.Contains(contentType, mimeAtom):
		return true

	default:
		return false
	}
}