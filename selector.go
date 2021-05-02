package trafilatura

import "golang.org/x/net/html"

type selectorRule func(*html.Node) bool

var commentSelectorRules = []selectorRule{
	commentSelectorRule1,
	commentSelectorRule2,
	commentSelectorRule3,
	commentSelectorRule4,
}

var discardedCommentSelectorRules = []selectorRule{
	discardedCommentSelectorRule1,
	discardedCommentSelectorRule2,
	discardedCommentSelectorRule3,
}

var contentSelectorRules = []selectorRule{
	contentSelectorRule1,
	contentSelectorRule2,
	contentSelectorRule3,
	contentSelectorRule4,
	contentSelectorRule5,
	contentSelectorRule6,
	contentSelectorRule7,
}

var discardedContentSelectorRules = []selectorRule{
	discardedContentSelectorRule1,
	discardedContentSelectorRule2,
	discardedContentSelectorRule3,
	discardedContentSelectorRule4,
}
