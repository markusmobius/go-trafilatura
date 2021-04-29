package trafilatura

var metaTitleSelectors = []string{
	".post-title, .entry-title",

	`h1[id*="title"], h1[class*="title"]`,
	`h1[id*="headline"], h1[class*="headline"]`,
	`h1[itemprop*="headline"]`,

	`h2[id*="title"], h2[class*="title"]`,
	`h2[id*="headline"], h2[class*="headline"]`,
	`h2[itemprop*="headline"]`,

	"header > h1",
}

var metaAuthorSelectors = []string{
	`a.author, a[rel="author"], a[rel="me"]`,
	`address.author, address[rel="author"], address[rel="me"]`,
	`link.author, link[rel="author"], link[rel="me"]`,
	`p.author, p[rel="author"], p[rel="me"]`,
	`span.author, span[rel="author"], span[rel="me"]`,
	`author`,

	`a[class*="author"], a[class*="posted-by"], a[itemprop*="author"]`,
	`span[class*="author"], span[class*="posted-by"], span[itemprop*="author"]`,

	`a[class*="byline"]`,
	`p[class*="byline"]`,
	`div[class*="byline"]`,
	`span[class*="byline"]`,

	`*[class*="author"], *[class*="screenname"]`,
}

var metaCategoriesSelectors = []string{
	`div[class^="postinfo"] a`,
	`div[class^="post-info"] a`,
	`div[class^="postmeta"] a`,
	`div[class^="post-meta"] a`,
	`div[class^="meta"] a`,
	`div[class^="entry-meta"] a`,
	`div[class^="entry-info"] a`,
	`div[class^="entry-utility"] a`,
	`div[class^="postpath"] a`,

	`p[class^="postmeta"] a`,
	`p[class^="entry-categories"] a`,
	`p.postinfo a`,
	`p#filedunder a`,

	`footer[class^="entry-meta"] a`,
	`footer[class^="entry-footer"] a`,

	`li.postcategory a`,
	`li.post-category a`,
	`li.entry-category a`,
	`span.postcategory a`,
	`span.post-category a`,
	`span.entry-category a`,

	`header.entry-header a`,

	`div.row a`,
	`div.tags a`,

	// `div[class*="byline"]`,
	// `p[class*="byline"]`,
	// `span.cat-links`,
}

var metaTagsSelectors = []string{
	`div.tags a`,

	`p[class^="entry-tags"] a`,

	`div.row a`,
	`div.jp-relatedposts a`,
	`div.entry-utility a`,
	`div[class^="tag"] a`,
	`div[class^="postmeta"] a`,
	`div[class^="meta"] a`,

	`.entry-meta a`,
	`*[class*="topics"]`,
}

// TODO: add span class tag-links
// "related-topics"
// https://github.com/grangier/python-goose/blob/develop/goose/extractors/tags.py
