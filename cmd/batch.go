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

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

func batchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "batch [flags] [file]",
		Short: "Download and extract page from list of urls that specified in the file",
		Long: "Download and extract page from list of urls that specified in the file.\n" +
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
	nThread, _ := flags.GetInt("parallel")
	intDelay, _ := flags.GetInt("delay")
	outputDir, _ := flags.GetString("output")
	delay := time.Duration(intDelay) * time.Second

	// Parse input file
	urls, names, err := parseBatchFile(cmd, args[0])
	if err != nil {
		logrus.Fatalf("failed to parse input: %v", err)
	}

	if len(urls) == 0 {
		logrus.Fatalf("no valid url found")
	}

	// Prepare extractor options and http client
	opts := createExtractorOptions(cmd)
	httpClient := createHttpClient(cmd)

	// Download and process concurrently
	g, ctx := errgroup.WithContext(context.Background())
	sem := semaphore.NewWeighted(int64(nThread))

	for i, url := range urls {
		url, name := url, names[i]

		g.Go(func() error {
			// Acquire semaphore to limit concurrent download
			err := sem.Acquire(ctx, 1)
			if err != nil {
				return nil
			}
			defer sem.Release(1)

			// Process URL
			result, err := processURL(httpClient, url, opts)
			if err != nil {
				return err
			}

			// Create destination file
			dst, err := os.Create(fp.Join(outputDir, name))
			if err != nil {
				return err
			}
			defer dst.Close()

			// Write output to file
			writeOutput(dst, result, cmd)

			// Add delay (to prevent too many request to target server)
			time.Sleep(delay)
			return nil
		})
	}

	err = g.Wait()
	if err != nil {
		logrus.Fatalf("process failed: %v", err)
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
		if !isValidURL(url) {
			continue
		}

		parsedURL, _ := nurl.ParseRequestURI(url)
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
		domainName := strings.ReplaceAll(url.Hostname(), ".", "-")
		newName := fmt.Sprintf(numberFormat+"-%s%s", nameIdx, domainName, nameExt)
		dstNames[i] = newName
	}

	return urls, dstNames, nil
}
