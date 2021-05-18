package main

import (
	"context"
	"net/http"
	nurl "net/url"
	"time"

	"github.com/markusmobius/go-trafilatura"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

type batchDownloader struct {
	httpClient     *http.Client
	extractOptions trafilatura.Options
	semaphore      *semaphore.Weighted
	delay          time.Duration
	cancelOnError  bool
	writeFunc      func(*trafilatura.ExtractResult, *nurl.URL, int) error
}

func (bd *batchDownloader) downloadURLs(ctx context.Context, urls []*nurl.URL) error {
	g, ctx := errgroup.WithContext(context.Background())

	for i, url := range urls {
		i, url := i, url

		g.Go(func() error {
			// Acquire semaphore to limit concurrent download
			err := bd.semaphore.Acquire(ctx, 1)
			if err != nil {
				return nil
			}

			// Process URL
			result, err := processURL(bd.httpClient, url, bd.extractOptions)
			bd.semaphore.Release(1)
			if err != nil {
				if bd.cancelOnError {
					return err
				}

				logrus.Warnf("failed to process %s: %v", url.String(), err)
				return nil
			}

			// Write to file
			err = bd.writeFunc(result, url, i)
			if err != nil {
				if bd.cancelOnError {
					return err
				}

				logrus.Warnf("failed to write %s: %v", url.String(), err)
				return nil
			}

			// Add delay (to prevent too many request to target server)
			time.Sleep(bd.delay)
			return nil
		})
	}

	return g.Wait()
}
