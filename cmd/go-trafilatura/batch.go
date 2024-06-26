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
	"bufio"
	"context"
	"fmt"
	nurl "net/url"
	"os"
	fp "path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/markusmobius/go-trafilatura"
	"github.com/spf13/cobra"
	"golang.org/x/sync/semaphore"
)

func batchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "batch [flags] [file]",
		Short: "Download and extract pages from list of urls that specified in the file",
		Long: "Download and extract pages from list of urls that specified in the file.\n" +
			"The file is text file that contains a list of url. The extract result will\n" +
			"be saved in format of \"<line number>-<domain name>.html\". To specify custom\n" +
			"name, write it in the same line as url, separated with tab: e.g. \"<URL>[tab]<Name>\"",
		Args: cobra.ExactArgs(1),
		Run:  batchCmdHandler,
	}

	flags := cmd.Flags()
	flags.StringP("output", "o", ".", "output directory for the result (default current work dir)")
	flags.Int("parallel", 10, "number of concurrent download at a time (default 10)")
	flags.Int("delay", 0, "delay between each url download in seconds (default 0)")

	return cmd
}

func batchCmdHandler(cmd *cobra.Command, args []string) {
	// Parse arguments
	flags := cmd.Flags()
	delay, _ := flags.GetInt("delay")
	nThread, _ := flags.GetInt("parallel")
	outputDir, _ := flags.GetString("output")
	userAgent, _ := cmd.Flags().GetString("user-agent")

	// Parse input file
	urls, names, err := parseBatchFile(cmd, args[0])
	if err != nil {
		log.Fatal().Msgf("failed to parse input: %v", err)
	}

	if len(urls) == 0 {
		log.Fatal().Msgf("no valid url found")
	}

	// Make sure output dir exist
	os.MkdirAll(outputDir, os.ModePerm)

	// Download and process concurrently
	fnWrite := func(result *trafilatura.ExtractResult, url *nurl.URL, idx int) error {
		name := names[idx]
		dst, err := os.Create(fp.Join(outputDir, name))
		if err != nil {
			return err
		}
		defer dst.Close()

		return writeOutput(dst, result, cmd)
	}

	err = (&batchDownloader{
		userAgent:      userAgent,
		httpClient:     createHttpClient(cmd),
		extractOptions: createExtractorOptions(cmd),
		semaphore:      semaphore.NewWeighted(int64(nThread)),
		delay:          time.Duration(delay) * time.Second,
		cancelOnError:  false,
		writeFunc:      fnWrite,
	}).downloadURLs(context.Background(), urls)

	if err != nil {
		log.Fatal().Msgf("process failed: %v", err)
	}
}

func parseBatchFile(cmd *cobra.Command, path string) ([]*nurl.URL, []string, error) {
	// Open file
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	// Prepare result
	var urls []*nurl.URL
	var dstNames []string

	// Scan line by line
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		// Fetch the text
		line := scanner.Text()
		line = strings.TrimSpace(line)

		// Find URL and name
		var url, name string
		if strings.Contains(line, "\t") {
			parts := strings.SplitN(line, "\t", 2)
			url = strings.TrimSpace(parts[0])
			name = strings.TrimSpace(parts[1])
		} else {
			url = line
		}

		// Validate URL
		parsedURL, valid := validateURL(url)
		if !valid {
			continue
		}

		urls = append(urls, parsedURL)
		dstNames = append(dstNames, name)
	}

	// Generate name for urls without specified name
	// and set the file extension.
	nameExt := outputExt(cmd)
	nameIdx, nURLs := 0, len(urls)
	numberFormat := fmt.Sprintf("%%0%dd", len(strconv.Itoa(nURLs)))

	for i, url := range urls {
		dstName := dstNames[i]
		if dstName != "" {
			if fp.Ext(dstName) != nameExt {
				dstNames[i] += nameExt
			}
			continue
		}

		nameIdx++
		newName := nameFromURL(url)
		newName = fmt.Sprintf(numberFormat+"-%s%s", nameIdx, newName, nameExt)
		dstNames[i] = newName
	}

	return urls, dstNames, nil
}
