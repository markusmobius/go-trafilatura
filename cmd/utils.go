package main

import (
	"io"
	"net/http"
	nurl "net/url"
	"os"
	"path"
	"regexp"
	"strings"
)

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func isValidURL(url string) bool {
	parsedURL, err := nurl.ParseRequestURI(url)
	if err != nil {
		return false
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return false
	}

	return true
}

func getFileContentType(r io.Reader) (string, error) {
	// Only the first 512 bytes are used to sniff the content type.
	buffer := make([]byte, 512)
	_, err := r.Read(buffer)
	if err != nil {
		return "", err
	}

	contentType := http.DetectContentType(buffer)
	return contentType, nil
}

func sliceToMap(strings ...string) map[string]struct{} {
	result := make(map[string]struct{})
	for _, s := range strings {
		result[s] = struct{}{}
	}
	return result
}

func rxFromString(str string) (*regexp.Regexp, error) {
	if str == "" {
		return nil, nil
	}

	return regexp.Compile(str)
}

func nameFromURL(url *nurl.URL) string {
	urlPath := strings.Trim(url.Path, "/")
	domain := strings.TrimPrefix(url.Hostname(), "www.")

	newName := strings.ReplaceAll(domain, ".", "-")
	if urlPath != "" {
		urlPath = path.Base(urlPath)
		urlPath = strings.ReplaceAll(urlPath, "/", "-")
		urlPath = strings.ReplaceAll(urlPath, ".", "-")
		newName += "-" + urlPath
	}

	return newName
}
