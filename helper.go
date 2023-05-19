package trafilatura

import (
	"strings"

	"github.com/go-shiori/dom"
	"github.com/markusmobius/go-trafilatura/internal/etree"
	"golang.org/x/net/html"
)

// CreateReadableComponent is helper function to convert the extract result to a single HTML document
// complete with its metadata and comment (if it exists).
func CreateReadableDocument(extract *ExtractResult) *html.Node {
	// Create base document
	doc := dom.CreateElement("html")
	head := etree.SubElement(doc, "head")
	body := etree.SubElement(doc, "body")

	// Put metadata
	meta := etree.SubElement(head, "meta")
	dom.SetAttribute(meta, "name", "title")
	dom.SetAttribute(meta, "content", extract.Metadata.Title)

	meta = etree.SubElement(head, "meta")
	dom.SetAttribute(meta, "name", "author")
	dom.SetAttribute(meta, "content", extract.Metadata.Author)

	meta = etree.SubElement(head, "meta")
	dom.SetAttribute(meta, "name", "url")
	dom.SetAttribute(meta, "content", extract.Metadata.URL)

	meta = etree.SubElement(head, "meta")
	dom.SetAttribute(meta, "name", "hostname")
	dom.SetAttribute(meta, "content", extract.Metadata.Hostname)

	meta = etree.SubElement(head, "meta")
	dom.SetAttribute(meta, "name", "description")
	dom.SetAttribute(meta, "content", extract.Metadata.Description)

	meta = etree.SubElement(head, "meta")
	dom.SetAttribute(meta, "name", "sitename")
	dom.SetAttribute(meta, "content", extract.Metadata.Sitename)

	meta = etree.SubElement(head, "meta")
	dom.SetAttribute(meta, "name", "date")
	dom.SetAttribute(meta, "content", extract.Metadata.Date.Format("2006-01-02"))

	meta = etree.SubElement(head, "meta")
	dom.SetAttribute(meta, "name", "categories")
	dom.SetAttribute(meta, "content", strings.Join(extract.Metadata.Categories, ", "))

	meta = etree.SubElement(head, "meta")
	dom.SetAttribute(meta, "name", "tags")
	dom.SetAttribute(meta, "content", strings.Join(extract.Metadata.Tags, "; "))

	meta = etree.SubElement(head, "meta")
	dom.SetAttribute(meta, "name", "license")
	dom.SetAttribute(meta, "content", extract.Metadata.License)

	// Put content
	content := extract.ContentNode
	if content != nil {
		content.Data = "div"
		dom.SetAttribute(content, "id", "content-body")
		dom.AppendChild(body, content)
	}

	// Put comments
	comments := extract.CommentsNode
	if comments != nil {
		comments.Data = "div"
		dom.SetAttribute(comments, "id", "comments-body")
		dom.AppendChild(body, comments)
	}

	return doc
}
