package trafilatura

const (
	cacheSize = 4096

	minExtractedSize        = 200
	minExtractedCommentSize = 10
	minOutputSize           = 10
	minOutputCommentSize    = 10

	minDuplicateCheckSize = 100
	maxDuplicateCount     = 2
)

var tagsToClean = sliceToMap(
	// important
	"aside", "embed", "footer", "form", "head", "iframe", "menu", "object", "script",
	// other content
	"applet", "audio", "canvas", "figure", "map", "picture", "svg", "video",
	// secondary
	"area", "blink", "button", "datalist", "details", "dialog",
	"frame", "frameset", "fieldset", "link", "input", "ins", "label", "legend",
	"marquee", "math", "menuitem", "nav", "noscript", "optgroup", "option",
	"output", "param", "progress", "rp", "rt", "rtc", "select", "source",
	"style", "summary", "track", "template", "textarea", "time", "use",
	// "meta", "hr", "img", "data"
)

var tagsToStrip = sliceToMap(
	"abbr", "acronym", "address", "bdi", "bdo", "big", "cite", "data", "dfn", "font",
	"hgroup", "img", "ins", "mark", "meta", "ruby", "small", "tbody", "tfoot", "thead",
	// "center", "rb", "wbr",
)

var emptyTagsToRemove = sliceToMap(
	"article", "b", "blockquote", "dd", "div", "dt", "em",
	"h1", "h2", "h3", "h4", "h5", "h6", "i", "li", "main",
	"p", "pre", "q", "section", "span", "strong",
	// "meta", "td", "a", "caption", "dl", "header", "colgroup", "col",
)

var tagCatalog = sliceToMap(
	"blockquote", "code",
	"del", "s", "strike",
	"h1", "h2", "h3", "h4", "h5", "h6",
	"em", "i", "b", "strong", "u", "kbd", "samp", "tt", "var", "sub", "sup",
	"br", "hr",
	"ul", "ol", "dl",
	"p", "pre", "q",
)

var formatTagCatalog = sliceToMap(
	"em", "i", "b", "strong", "u", "kbd",
	"samp", "tt", "var", "sub", "sup",
)

var tagsToSanitize = sliceToMap(
	"aside", "audio", "button", "fieldset", "figure", "footer", "iframe",
	"img", "image", "input", "label", "link", "nav", "noindex", "noscript",
	"object", "option", "select", "source", "svg", "time",
)

func sliceToMap(strings ...string) map[string]struct{} {
	result := make(map[string]struct{})
	for _, s := range strings {
		result[s] = struct{}{}
	}
	return result
}

func duplicateMap(original map[string]struct{}) map[string]struct{} {
	duplicate := make(map[string]struct{})
	for key, val := range original {
		duplicate[key] = val
	}
	return duplicate
}
