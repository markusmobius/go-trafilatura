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
