package main

import (
	"fmt"
	"net/http"
	nurl "net/url"
	"os"
	fp "path/filepath"
	"strings"
	"time"

	"github.com/go-shiori/dom"
	"github.com/markusmobius/go-trafilatura"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "go-trafilatura [flags] [source]",
		Run:   rootCmdHandler,
		Short: "extract readable content from a HTML file or url",
		Long: "Extract readable content from a HTML file or url.\n" +
			"It also has supports for batch download url either\n" +
			"from a file, RSS feeds and sitemap.",
		Args: cobra.ExactArgs(1),
	}

	rootCmd.Flags().StringP("language", "l", "", "target language (ISO 639-1 codes)")
	rootCmd.Flags().Bool("no-fallback", false, "disable fallback extraction using readability and dom-distiller")
	rootCmd.Flags().Bool("no-comments", false, "exclude comments  extraction result")
	rootCmd.Flags().Bool("no-tables", false, "include tables in extraction result")
	rootCmd.Flags().Bool("images", false, "include images in extraction result (experimental)")
	rootCmd.Flags().Bool("links", false, "keep links in extraction result (experimental)")
	rootCmd.Flags().Bool("deduplicate", false, "filter out duplicate segments and sections")
	rootCmd.Flags().Bool("has-metadata", false, "only output documents with title, URL and date")
	rootCmd.Flags().BoolP("verbose", "v", false, "enable log message")
	rootCmd.Flags().IntP("timeout", "t", 30, "timeout for downloading web page in seconds")

	err := rootCmd.Execute()
	if err != nil {
		logrus.Fatalln(err)
	}
}

func rootCmdHandler(cmd *cobra.Command, args []string) {
	// Fetch arguments
	source := args[0]
	language, _ := cmd.Flags().GetString("language")
	noFallback, _ := cmd.Flags().GetBool("no-fallback")
	noComments, _ := cmd.Flags().GetBool("no-comments")
	noTables, _ := cmd.Flags().GetBool("no-tables")
	includeImages, _ := cmd.Flags().GetBool("images")
	includeLinks, _ := cmd.Flags().GetBool("links")
	deduplicate, _ := cmd.Flags().GetBool("deduplicate")
	hasMetadata, _ := cmd.Flags().GetBool("has-metadata")
	verbose, _ := cmd.Flags().GetBool("verbose")
	timeout, _ := cmd.Flags().GetInt("timeout")

	// Create extraction options
	opts := trafilatura.Options{
		TargetLanguage:       language,
		NoFallback:           noFallback,
		ExcludeComments:      noComments,
		ExcludeTables:        noTables,
		IncludeImages:        includeImages,
		IncludeLinks:         includeLinks,
		Deduplicate:          deduplicate,
		HasEssentialMetadata: hasMetadata,
		EnableLog:            verbose,
	}

	// Process source
	var err error
	var result *trafilatura.ExtractResult

	if fileExists(source) {
		result, err = processFile(source, opts)
	} else {
		validURL, errURL := nurl.ParseRequestURI(source)
		if errURL != nil {
			logrus.Fatalf("URL is not valid: %v", err)
		}
		result, err = processURL(validURL, timeout, opts)
	}

	if err != nil {
		logrus.Fatalf("failed to extract %s: %v", source, err)
	}

	if result == nil {
		logrus.Fatalf("failed to extract %s: no readable content", source)
	}

	// Print result
	fmt.Println(dom.OuterHTML(result.ContentNode))
	fmt.Println(dom.OuterHTML(result.CommentsNode))
}

func processFile(path string, opts trafilatura.Options) (*trafilatura.ExtractResult, error) {
	// Open file
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Make sure it's html
	if fp.Ext(path) != ".html" {
		contentType, err := getFileContentType(f)
		if err != nil {
			return nil, err
		}

		if !strings.Contains(contentType, "text/html") {
			return nil, fmt.Errorf("%s is not html file: %s", path, contentType)
		}
	}

	// Extract
	result, err := trafilatura.Extract(f, opts)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func processURL(url *nurl.URL, timeout int, opts trafilatura.Options) (*trafilatura.ExtractResult, error) {
	// Prepare HTTP client
	client := http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	// Download URL
	strURL := url.String()
	logrus.Println("downloading", strURL)

	resp, err := client.Get(strURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Make sure it's html
	contentType, err := getFileContentType(resp.Body)
	if err != nil {
		return nil, err
	}

	if !strings.Contains(contentType, "text/html") {
		return nil, fmt.Errorf("%s is not html: %s", strURL, contentType)
	}

	// Extract
	opts.OriginalURL = url
	result, err := trafilatura.Extract(resp.Body, opts)
	if err != nil {
		return nil, err
	}

	return result, nil
}
