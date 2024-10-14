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

package trafilatura

import (
	"mime"
	"path/filepath"
	"strings"
)

// trim removes unnecessary spaces within a text string.
func trim(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	return strings.TrimSpace(s)
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

func isImageFile(imageSrc string) bool {
	if imageSrc == "" {
		return false
	}

	ext := filepath.Ext(imageSrc)
	mimeType := mime.TypeByExtension(ext)
	return strings.HasPrefix(mimeType, "image")
}

func uniquifyLists(currents ...string) []string {
	var finalTags []string
	tracker := map[string]struct{}{}

	for _, current := range currents {
		separator := ","
		if strings.Count(current, ";") > strings.Count(current, ",") {
			separator = ";"
		}

		for _, entry := range strings.Split(current, separator) {
			entry = trim(entry)
			entry = strings.ReplaceAll(entry, `"`, "")
			entry = strings.ReplaceAll(entry, `'`, "")

			if _, tracked := tracker[entry]; entry != "" && !tracked {
				finalTags = append(finalTags, entry)
				tracker[entry] = struct{}{}
			}
		}
	}

	return finalTags
}
