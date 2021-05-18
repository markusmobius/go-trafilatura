package main

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	nurl "net/url"
	"os"
	fp "path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/markusmobius/go-trafilatura"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/sync/semaphore"
)

var rxCdata = regexp.MustCompile(`(?i)^<!--\[cdata\[(.*)\]\]-->$`)

func sitemapCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sitemap [flags] [url]",
		Short: "Download and extract pages from a sitemap",
		Long: "Download and extract pages from list of urls that specified in a sitemap.\n" +
			"Trafilatura supports simple sitemap finder, so you can point the url into\n" +
			"an ordinary web page then Trafilatura will attempt to find the sitemap.",
		Args: cobra.ExactArgs(1),
		Run:  sitemapCmdHandler,
	}

	flags := cmd.Flags()
	flags.StringP("output", "o", ".", "output directory for the result (default current work dir)")
	flags.Int("parallel", 10, "number of concurrent download at a time (default 10)")
	flags.Int("delay", 0, "delay between each url download in seconds (default 0)")
	flags.String("filter", "", "regular expression for allowed url")
	flags.String("exclude", "", "regular expression for excluded url")
	flags.StringArray("domains", nil, "list of allowed domains")
	flags.StringArray("no-domains", nil, "list of excluded domains")

	return cmd
}

func sitemapCmdHandler(cmd *cobra.Command, args []string) {
	// Parse flags
	flags := cmd.Flags()
	delay, _ := flags.GetInt("delay")
	nThread, _ := flags.GetInt("parallel")
	allowedPattern, _ := flags.GetString("filter")
	excludedPattern, _ := flags.GetString("exclude")
	allowedDomains, _ := flags.GetStringArray("domains")
	excludedDomains, _ := flags.GetStringArray("no-domains")
	outputDir, _ := flags.GetString("output")

	// Find sitemap URL
	httpClient := createHttpClient(cmd)
	sitemapURLs, err := findSitemapURLs(httpClient, args[0])
	if err != nil {
		logrus.Fatalf("failed to find sitemap: %v", err)
	}

	// Download all sitemaps recursively, concurrently
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

	pageURLs := (&sitemapDownloader{
		cache:      make(map[string]struct{}),
		httpClient: httpClient,
		filter:     fnFilter,
		delay:      time.Duration(delay) * time.Second,
		semaphore:  semaphore.NewWeighted(int64(nThread)),
	}).downloadURLs(context.Background(), sitemapURLs)

	logrus.Printf("found %d page URLs", len(pageURLs))

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

func findSitemapURLs(client *http.Client, baseURL string) ([]*nurl.URL, error) {
	// Make sure URL valid
	if !isValidURL(baseURL) {
		return nil, fmt.Errorf("url is not valid")
	}

	// Parse URL
	parsedURL, _ := nurl.ParseRequestURI(baseURL)

	// If it already looks like a sitemap URL, return
	if strings.HasSuffix(baseURL, ".xml") ||
		strings.HasSuffix(baseURL, ".xml.gz") ||
		strings.HasSuffix(baseURL, "sitemap") {
		return []*nurl.URL{parsedURL}, nil
	}

	// If not found, try to check in robots.txt.
	// Here we'll ignore error since it's possible that a site doesn't have robots.txt
	parsedURL.Path = "/robots.txt"
	parsedURL.RawQuery = nurl.Values{}.Encode()
	parsedURL.Fragment = ""

	sitemapURLs, err := findSitemapURLsInRobots(client, parsedURL.String())
	if err != nil {
		logrus.Warnln("failed to look in robots.txt:", err)
	}

	// If there are no sitemap found, just add the default path.
	if len(sitemapURLs) == 0 {
		parsedURL.Path = "/sitemap.xml"
		sitemapURLs = append(sitemapURLs, parsedURL)
	}

	return sitemapURLs, nil
}

func findSitemapURLsInRobots(client *http.Client, robotsURL string) ([]*nurl.URL, error) {
	// Download URL
	logrus.Println("downloading robots.txt:", robotsURL)
	resp, err := client.Get(robotsURL)
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

			if isValidURL(line) {
				parsedURL, _ := nurl.ParseRequestURI(line)
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
