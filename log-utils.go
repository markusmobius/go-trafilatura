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

package trafilatura

import "github.com/sirupsen/logrus"

func logInfo(opts Options, format string, args ...interface{}) {
	if opts.EnableLog {
		logrus.Infof(format, args...)
	}
}

func logWarn(opts Options, format string, args ...interface{}) {
	if opts.EnableLog {
		logrus.Warnf(format, args...)
	}
}

func logDebug(opts Options, format string, args ...interface{}) {
	if opts.EnableLog {
		logrus.Debugf(format, args...)
	}
}
