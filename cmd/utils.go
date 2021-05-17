package main

import (
	"io"
	"net/http"
	nurl "net/url"
	"os"
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
