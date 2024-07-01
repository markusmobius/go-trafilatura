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

import "fmt"

func logInfo(opts Options, format string, args ...interface{}) {
	if opts.EnableLog {
		log.Info().Msg(ellipsis(format, args...))
	}
}

func logWarn(opts Options, format string, args ...interface{}) {
	if opts.EnableLog {
		log.Warn().Msg(ellipsis(format, args...))
	}
}

func logDebug(opts Options, format string, args ...interface{}) {
	if opts.EnableLog {
		log.Debug().Msg(ellipsis(format, args...))
	}
}

func ellipsis(format string, args ...any) string {
	for i, arg := range args {
		if str, isString := arg.(string); isString {
			str = trim(str)
			if limit := 120; len(str) > limit {
				str = str[:limit] + "..."
			}
			args[i] = trim(str)
		}
	}

	return fmt.Sprintf(format, args...)
}
