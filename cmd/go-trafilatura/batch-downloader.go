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
	extractOptions trafilatura.Options
	semaphore      *semaphore.Weighted
	httpClient     *http.Client
	userAgent      string
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
			result, err := processURL(bd.httpClient, bd.userAgent, url, bd.extractOptions)
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
