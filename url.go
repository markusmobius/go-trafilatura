package trafilatura

import (
	nurl "net/url"
	"path"
	"strings"
)

// isAbsoluteURL checks if URL is valid and absolute.
func isAbsoluteURL(s string) (bool, *nurl.URL) {
	// Make sure URL is absolute
	url, err := nurl.ParseRequestURI(s)
	if err != nil {
		return false, nil
	}

	// We only want HTTP(s)
	if url.Scheme != "http" && url.Scheme != "https" {
		return false, nil
	}

	return true, url
}

// CreateAbsoluteURL convert url to absolute path based on base.
// However, if url is prefixed with hash (#), the url won't be changed.
func createAbsoluteURL(url string, base *nurl.URL) string {
	if url == "" || base == nil {
		return url
	}

	// If it is hash tag, return as it is
	if strings.HasPrefix(url, "#") {
		return url
	}

	// If it is data URI, return as it is
	if strings.HasPrefix(url, "data:") {
		return url
	}

	// If it is javascript URI, return as it is
	if strings.HasPrefix(url, "javascript:") {
		return url
	}

	// If it is already an absolute URL, return as it is
	tmp, err := nurl.ParseRequestURI(url)
	if err == nil && tmp.Scheme != "" && tmp.Hostname() != "" {
		return url
	}

	// Otherwise, resolve against base URI.
	// Normalize URL first.
	if !strings.HasPrefix(url, "/") {
		url = path.Join(base.Path, url)
	}

	tmp, err = nurl.Parse(url)
	if err != nil {
		return url
	}

	return base.ResolveReference(tmp).String()
}

func extractDomainURL(url string) string {
	isAbsolute, parsedURL := isAbsoluteURL(url)
	if !isAbsolute {
		return ""
	}

	return parsedURL.Host
}
