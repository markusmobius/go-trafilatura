// This file is part of go-trafilatura, Go package for extracting readable
// content, comments and metadata from a web page. Source available in
// <https://github.com/markusmobius/go-trafilatura>.
// Copyright (C) 2021 Markus Mobius
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by the
// Free Software Foundation, either version 3 of the License, or (at your
// option) any later version.
//
// This program is distributed in the hope that it will be useful, but
// WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY
// or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License
// for more details.
//
// You should have received a copy of the GNU General Public License along
// with this program. If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/go-shiori/dom"
	"github.com/markusmobius/go-trafilatura"
	"github.com/markusmobius/go-trafilatura/internal/etree"
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

func writeOutput(w io.Writer, result *trafilatura.ExtractResult, cmd *cobra.Command) error {
	outputFormat, _ := cmd.Flags().GetString("format")

	switch outputFormat {
	case "txt":
		return writeText(w, result)
	case "json":
		return writeJSON(w, result)
	default:
		return writeHTML(w, result)
	}
}

func writeText(w io.Writer, result *trafilatura.ExtractResult) error {
	buffer := bytes.NewBuffer(nil)
	buffer.WriteString(result.ContentText)

	if result.CommentsText != "" {
		if buffer.Len() > 0 {
			buffer.WriteString(" ")
		}
		buffer.WriteString(result.ContentText)
	}

	_, err := io.Copy(w, buffer)
	return err
}

func writeJSON(w io.Writer, result *trafilatura.ExtractResult) error {
	data := jsonExtractResult(*result)
	return json.NewEncoder(w).Encode(data)
}

func writeHTML(w io.Writer, result *trafilatura.ExtractResult) error {
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

	meta = etree.SubElement(head, "meta")
	dom.SetAttribute(meta, "name", "license")
	dom.SetAttribute(meta, "content", result.Metadata.License)

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
	_, err := fmt.Fprint(w, dom.OuterHTML(doc))
	return err
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
		"license":     r.Metadata.License,
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
