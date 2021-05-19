package main

import (
	"context"
	"fmt"
	"net/http"
	nurl "net/url"
	"strings"
	"sync"
	"time"

	betree "github.com/beevik/etree"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

type sitemapDownloader struct {
	sync.RWMutex

	cache      map[string]struct{}
	httpClient *http.Client
	semaphore  *semaphore.Weighted
	delay      time.Duration
	filterFunc func(*nurl.URL) bool
}

func (sd *sitemapDownloader) downloadURLs(ctx context.Context, urls []*nurl.URL) []*nurl.URL {
	pageURLs := []*nurl.URL{}
	g, ctx := errgroup.WithContext(context.Background())

	for _, url := range urls {
		url := url

		g.Go(func() error {
			// Make sure this url hasn't been downloaded
			if sd.isDownloaded(url) {
				return nil
			}

			// Acquire semaphore to limit concurrent download
			err := sd.semaphore.Acquire(ctx, 1)
			if err != nil {
				return nil
			}

			// Download and parse url
			newSitemapURLs, newPageURLs, err := sd.downloadURL(url)
			sd.markAsDownloaded(url)
			sd.semaphore.Release(1)

			if err != nil {
				logrus.Warnf("failed to parse sitemap: %v", err)
				return nil
			}

			// Process the additional urls
			additionalPageURLs := sd.downloadURLs(ctx, newSitemapURLs)

			// Save all page URLs
			sd.Lock()
			pageURLs = append(pageURLs, newPageURLs...)
			pageURLs = append(pageURLs, additionalPageURLs...)
			sd.Unlock()

			return nil
		})
	}

	g.Wait()

	// Make sure page URLs are unique
	uniquePageURLs := []*nurl.URL{}
	uniqueTracker := make(map[string]struct{})

	for _, url := range pageURLs {
		if sd.filterFunc != nil && !sd.filterFunc(url) {
			continue
		}

		strURL := url.String()
		if _, exist := uniqueTracker[strURL]; exist {
			continue
		}

		uniqueTracker[strURL] = struct{}{}
		uniquePageURLs = append(uniquePageURLs, url)
	}

	return uniquePageURLs
}

func (sd *sitemapDownloader) downloadURL(url *nurl.URL) ([]*nurl.URL, []*nurl.URL, error) {
	// Download URL
	strURL := url.String()
	logrus.Println("downloading sitemap", strURL)

	resp, err := sd.httpClient.Get(strURL)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	// Make sure it's XML
	contentType := resp.Header.Get("Content-Type")
	if !sd.contentIsXML(contentType) {
		return nil, nil, fmt.Errorf("%s is not xml: \"%s\"", strURL, contentType)
	}

	// Parse
	doc := betree.NewDocument()
	if _, err := doc.ReadFrom(resp.Body); err != nil {
		return nil, nil, err
	}

	pageURLs := []*nurl.URL{}
	sitemapURLs := []*nurl.URL{}
	for _, loc := range doc.FindElements("//loc") {
		parsedURL, valid := validateURL(loc.Text())
		if !valid {
			continue
		}

		parent := loc.Parent()
		if parent == nil {
			continue
		}

		switch parent.Tag {
		case "url":
			pageURLs = append(pageURLs, parsedURL)
		case "sitemap":
			sitemapURLs = append(sitemapURLs, parsedURL)
		}
	}

	// Add delay (to prevent too many request to target server)
	time.Sleep(sd.delay)

	return sitemapURLs, pageURLs, nil
}

func (sd *sitemapDownloader) isDownloaded(url *nurl.URL) bool {
	sd.RLock()
	defer sd.RUnlock()

	_, processed := sd.cache[url.String()]
	return processed
}

func (sd *sitemapDownloader) markAsDownloaded(url *nurl.URL) {
	sd.Lock()
	sd.cache[url.String()] = struct{}{}
	sd.Unlock()
}

func (sd *sitemapDownloader) contentIsXML(contentType string) bool {
	switch {
	case strings.Contains(contentType, "text/xml"),
		strings.Contains(contentType, "application/xml"):
		return true

	default:
		return false
	}
}
