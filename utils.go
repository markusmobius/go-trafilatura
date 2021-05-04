package trafilatura

import (
	"regexp"
	"strings"

	"github.com/go-shiori/dom"
	"golang.org/x/net/html"
)

var (
	rxImageExtension = regexp.MustCompile(`(?i)([^\s]+(\.(jpe?g|png|gif|bmp)))`)
)

func strNormalize(s string) string {
	s = strings.TrimSpace(s)
	return strings.Join(strings.Fields(s), " ")
}

func strWordCount(s string) int {
	return len(strings.Fields(s))
}

func strOr(args ...string) string {
	for i := 0; i < len(args); i++ {
		if args[i] != "" {
			return args[i]
		}
	}
	return ""
}

func strIn(s string, args ...string) bool {
	for i := 0; i < len(args); i++ {
		if args[i] == s {
			return true
		}
	}
	return false
}

func getRune(s string, idx int) rune {
	for i, r := range s {
		if i == idx {
			return r
		}
	}

	return -1
}

func isImageFile(imageSrc string) bool {
	return imageSrc != "" && rxImageExtension.MatchString(imageSrc)
}

// iterTags is similar with dom.GetElementsByTagName, except here we also
// include node itself if it has a suitable tags.
func iterTags(node *html.Node, tags ...string) []*html.Node {
	var subElements []*html.Node
	if len(tags) == 0 {
		subElements = dom.GetElementsByTagName(node, "*")
	} else {
		for _, tag := range tags {
			matches := dom.GetElementsByTagName(node, tag)
			subElements = append(subElements, matches...)
		}
	}

	if len(tags) == 0 || strIn(dom.TagName(node), tags...) {
		subElements = append([]*html.Node{node}, subElements...)
	}

	return subElements
}

// extendNode is similar with dom.AppendChild, except here we only append
// the child while excluding html.ElementNode inside the child. It's created
// to emulate the behavior of `etree.extend` in Python.
func extendNode(node, child *html.Node) {
	// Clone the child node first
	clone := dom.Clone(child, false)

	// Put the non-element nodes from child to clone
	for _, gc := range dom.ChildNodes(child) {
		if gc.Type != html.ElementNode {
			dom.AppendChild(clone, gc)
		}
	}

	// Append the clone to the target node
	dom.AppendChild(node, clone)
}

func htmlFromString(str string) *html.Node {
	doc, err := html.Parse(strings.NewReader(str))
	if err != nil {
		return nil
	}

	return doc
}

func nodeFromString(str string) *html.Node {
	doc, err := html.Parse(strings.NewReader(str))
	if err != nil {
		return nil
	}

	return dom.QuerySelector(doc, "body>*")
}
