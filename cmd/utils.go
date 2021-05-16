package main

import (
	"io"
	"net/http"
	"os"
)

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
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
