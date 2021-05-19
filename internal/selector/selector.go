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
