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
	"context"
	"net/http"
	nurl "net/url"
	"time"

	"github.com/markusmobius/go-trafilatura"
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

				log.Warn().Msgf("failed to process %s: %v", url.String(), err)
				return nil
			}

			// Write to file
			err = bd.writeFunc(result, url, i)
			if err != nil {
				if bd.cancelOnError {
					return err
				}

				log.Warn().Msgf("failed to write %s: %v", url.String(), err)
				return nil
			}

			// Add delay (to prevent too many request to target server)
			time.Sleep(bd.delay)
			return nil
		})
	}

	return g.Wait()
}
