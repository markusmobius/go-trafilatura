// This file is part of go-trafilatura, Go package for extracting readable
// content, comments and metadata from a web page. Source available in
// <https://github.com/markusmobius/go-trafilatura>.
//
// Copyright (C) 2021 Markus Mobius
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/go-shiori/dom"
	"github.com/markusmobius/go-trafilatura"
	"github.com/spf13/cobra"
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
	doc := trafilatura.CreateReadableDocument(result)
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
