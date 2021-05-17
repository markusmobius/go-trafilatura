package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/go-shiori/dom"
	"github.com/markusmobius/go-trafilatura"
	"github.com/markusmobius/go-trafilatura/etree"
	"github.com/spf13/cobra"
	"golang.org/x/net/html"
)

func outputExt(cmd *cobra.Command) string {
	outputFormat, _ := cmd.Flags().GetString("format")

	switch outputFormat {
	case "txt":
		return ".txt"
	case "json":
		return ".json"
	default:
		return ".html"
	}
}

func writeOutput(w io.Writer, result *trafilatura.ExtractResult, cmd *cobra.Command) {
	outputFormat, _ := cmd.Flags().GetString("format")

	switch outputFormat {
	case "txt":
		writeText(w, result)
	case "json":
		writeJSON(w, result)
	default:
		writeHTML(w, result)
	}
}

func writeText(w io.Writer, result *trafilatura.ExtractResult) {
	buffer := bytes.NewBuffer(nil)
	buffer.WriteString(result.ContentText)

	if result.CommentsText != "" {
		if buffer.Len() > 0 {
			buffer.WriteString(" ")
		}
		buffer.WriteString(result.ContentText)
	}

	io.Copy(w, buffer)
}

func writeJSON(w io.Writer, result *trafilatura.ExtractResult) {
	data := jsonExtractResult(*result)
	json.NewEncoder(w).Encode(data)
}

func writeHTML(w io.Writer, result *trafilatura.ExtractResult) {
	// Create base document
	doc, _ := html.Parse(bytes.NewBuffer(nil))
	head := dom.QuerySelector(doc, "head")
	body := dom.QuerySelector(doc, "body")

	// Put metadata
	meta := etree.SubElement(head, "meta")
	dom.SetAttribute(meta, "name", "title")
	dom.SetAttribute(meta, "content", result.Metadata.Title)

	meta = etree.SubElement(head, "meta")
	dom.SetAttribute(meta, "name", "author")
	dom.SetAttribute(meta, "content", result.Metadata.Author)

	meta = etree.SubElement(head, "meta")
	dom.SetAttribute(meta, "name", "url")
	dom.SetAttribute(meta, "content", result.Metadata.URL)

	meta = etree.SubElement(head, "meta")
	dom.SetAttribute(meta, "name", "hostname")
	dom.SetAttribute(meta, "content", result.Metadata.Hostname)

	meta = etree.SubElement(head, "meta")
	dom.SetAttribute(meta, "name", "description")
	dom.SetAttribute(meta, "content", result.Metadata.Description)

	meta = etree.SubElement(head, "meta")
	dom.SetAttribute(meta, "name", "sitename")
	dom.SetAttribute(meta, "content", result.Metadata.Sitename)

	meta = etree.SubElement(head, "meta")
	dom.SetAttribute(meta, "name", "date")
	dom.SetAttribute(meta, "content", result.Metadata.Date)

	meta = etree.SubElement(head, "meta")
	dom.SetAttribute(meta, "name", "categories")
	dom.SetAttribute(meta, "content", strings.Join(result.Metadata.Categories, ", "))

	meta = etree.SubElement(head, "meta")
	dom.SetAttribute(meta, "name", "tags")
	dom.SetAttribute(meta, "content", strings.Join(result.Metadata.Tags, "; "))

	// Put content
	content := result.ContentNode
	if content != nil {
		content.Data = "div"
		dom.SetAttribute(content, "id", "content-body")
		dom.AppendChild(body, content)
	}

	// Put comments
	comments := result.CommentsNode
	if comments != nil {
		comments.Data = "div"
		dom.SetAttribute(comments, "id", "comments-body")
		dom.AppendChild(body, comments)
	}

	// Print as HTML
	fmt.Fprint(w, dom.OuterHTML(doc))
}

type jsonExtractResult trafilatura.ExtractResult

func (r jsonExtractResult) MarshalJSON() ([]byte, error) {
	// Convert metadata to map first
	metadata := map[string]interface{}{
		"title":       r.Metadata.Title,
		"author":      r.Metadata.Author,
		"url":         r.Metadata.URL,
		"hostname":    r.Metadata.Hostname,
		"description": r.Metadata.Description,
		"sitename":    r.Metadata.Sitename,
		"date":        r.Metadata.Date,
		"categories":  r.Metadata.Categories,
		"tags":        r.Metadata.Tags,
	}

	// Convert result to map
	result := map[string]interface{}{
		"contentHTML": dom.OuterHTML(r.ContentNode),
		"contentText": r.ContentText,
		"metadata":    metadata,
	}

	if r.CommentsNode != nil {
		result["commentsText"] = r.CommentsText
		result["commentsHTML"] = dom.OuterHTML(r.CommentsNode)
	}

	return json.Marshal(&result)
}
