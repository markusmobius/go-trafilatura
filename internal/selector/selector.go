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

// Code in this file is ported from <https://github.com/adbar/trafilatura>
// which available under GNU GPL v3 license.

package selector

import "golang.org/x/net/html"

type Rule func(*html.Node) bool

var MetaTitleRules = []Rule{
	metaTitleRule1,
	metaTitleRule2,
	metaTitleRule3,
	metaTitleRule4,
}

var MetaAuthorRules = []Rule{
	metaAuthorRule1,
	metaAuthorRule2,
	metaAuthorRule3,
	metaAuthorRule4,
}

var MetaCategoriesRules = []Rule{
	metaCategoriesRule1,
	metaCategoriesRule2,
	metaCategoriesRule3,
	metaCategoriesRule4,
	metaCategoriesRule5,
	metaCategoriesRule6,
}

var MetaTagsRules = []Rule{
	metaTagsRule1,
	metaTagsRule2,
	metaTagsRule3,
	metaTagsRule4,
}

var CommentsRules = []Rule{
	commentsRule1,
	commentsRule2,
	commentsRule3,
	commentsRule4,
}

var DiscardedCommentsRules = []Rule{
	discardedCommentsRule1,
	discardedCommentsRule2,
	discardedCommentsRule3,
}

var ContentRules = []Rule{
	contentRule1,
	contentRule2,
	contentRule3,
	contentRule4,
	contentRule5,
	contentRule6,
	contentRule7,
}

var DiscardedContentRules = []Rule{
	discardedContentRule1,
	discardedContentRule2,
	discardedContentRule3,
	discardedContentRule4,
}
