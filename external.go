package trafilatura

import (
	"fmt"
	nurl "net/url"
	"strings"

	"github.com/go-shiori/dom"
	"github.com/go-shiori/go-readability"
	distiller "github.com/markusmobius/go-domdistiller"
	"github.com/markusmobius/go-trafilatura/internal/etree"
	"golang.org/x/net/html"
)

func tryReadability(originalExtract, doc *html.Node, url string, opts Options) (*html.Node, error) {
	// Extract using go-readability
	docHtml := dom.OuterHTML(doc)
	r := strings.NewReader(docHtml)
	article, err := readability.FromReader(r, url)
	if err != nil {
		return nil, err
	}

	return article.Node, nil
}

func tryDomDistiller(originalExtract, doc *html.Node, url string, opts Options) (*html.Node, error) {
	// Parse URL
	parsedURL, err := nurl.ParseRequestURI(url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %v", err)
	}

	// Extract using go-domdistiller
	docHtml := dom.OuterHTML(doc)
	r := strings.NewReader(docHtml)

	res, err := distiller.ApplyForReader(r, &distiller.Options{
		OriginalURL:    parsedURL,
		PaginationAlgo: distiller.PageNumber})
	if err != nil {
		return nil, err
	}

	// Since dom-distiller return extract in HTML string, parse it back into node
	distillerExtract := dom.CreateElement("div")
	dom.SetInnerHTML(distillerExtract, res.HTML)
	if len(dom.Children(distillerExtract)) == 0 {
		return nil, fmt.Errorf("can't find any content")
	}

	return distillerExtract, nil
}

// sanitizeTree converts and sanitize the output from the generic
// fallback algorithm (post-processing).
func sanitizeTree(tree *html.Node, opts Options) {
	// Get list of tags to sanitize
	sanitizeList := duplicateMap(tagsToSanitize)
	if opts.IncludeImages {
		delete(sanitizeList, "img")
		delete(sanitizeList, "image")
	}

	// Delete unnecessary elements
	for _, elem := range dom.GetElementsByTagName(tree, "*") {
		elemTag := dom.TagName(elem)
		if _, exist := sanitizeList[elemTag]; exist {
			etree.Remove(elem)
		}
	}

	// Handle links
	strippingList := duplicateMap(tagsToStrip)
	strippingList["span"] = struct{}{}

	if !opts.IncludeLinks {
		strippingList["a"] = struct{}{}
	}

	if opts.IncludeImages {
		delete(strippingList, "img")
	}

	for tagName := range strippingList {
		etree.StripTags(tree, tagName)
	}

	pruneHTML(tree)

	// Sanitize
	var sanitizationList []string
	uniqueTags := make(map[string]struct{})
	for _, node := range dom.GetElementsByTagName(tree, "*") {
		tagName := dom.TagName(node)
		if _, exist := uniqueTags[tagName]; exist {
			continue
		}

		uniqueTags[tagName] = struct{}{}
		if _, exist := validTagCatalog[tagName]; !exist {
			sanitizationList = append(sanitizationList, tagName)
		}
	}

	if len(sanitizationList) > 0 {
		etree.StripTags(tree, sanitizationList...)
	}
}
