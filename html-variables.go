package trafilatura

var tagsToKill = map[string]struct{}{
	// important
	"aside":  {},
	"embed":  {},
	"footer": {},
	"form":   {},
	"head":   {},
	"iframe": {},
	"menu":   {},
	"object": {},
	"script": {},

	// other content
	"applet":  {},
	"audio":   {},
	"canvas":  {},
	"figure":  {},
	"map":     {},
	"picture": {},
	"svg":     {},
	"video":   {},

	// secondary
	"area":     {},
	"blink":    {},
	"button":   {},
	"datalist": {},
	"details":  {},
	"dialog":   {},
	"frame":    {},
	"frameset": {},
	"fieldset": {},
	"link":     {},
	"input":    {},
	"ins":      {},
	"label":    {},
	"legend":   {},
	"marquee":  {},
	"math":     {},
	"menuitem": {},
	"nav":      {},
	"noscript": {},
	"optgroup": {},
	"option":   {},
	"output":   {},
	"param":    {},
	"progress": {},
	"rp":       {},
	"rt":       {},
	"rtc":      {},
	"select":   {},
	"source":   {},
	"style":    {},
	"summary":  {},
	"track":    {},
	"template": {},
	"textarea": {},
	"time":     {},
	"use":      {},

	// "meta": {},
	// "hr":   {},
	// "img":  {},
	// "data": {},
}

var tagsToRemove = map[string]struct{}{
	"abbr":    {},
	"acronym": {},
	"address": {},
	"bdi":     {},
	"bdo":     {},
	"big":     {},
	"cite":    {},
	"data":    {},
	"dfn":     {},
	"font":    {},
	"hgroup":  {},
	"img":     {},
	"ins":     {},
	"mark":    {},
	"meta":    {},
	"ruby":    {},
	"small":   {},
	"tbody":   {},
	"tfoot":   {},
	"thead":   {},

	// "center": {},
	// "rb":     {},
	// "wbr":    {},
}

var emptyTagsToRemove = map[string]struct{}{
	"article":    {},
	"b":          {},
	"blockquote": {},
	"dd":         {},
	"div":        {},
	"dt":         {},
	"em":         {},

	"h1":   {},
	"h2":   {},
	"h3":   {},
	"h4":   {},
	"h5":   {},
	"h6":   {},
	"i":    {},
	"li":   {},
	"main": {},

	"p":       {},
	"pre":     {},
	"q":       {},
	"section": {},
	"span":    {},
	"strong":  {},

	// "meta":     {},
	// "td":       {},
	// "a":        {},
	// "caption":  {},
	// "dl":       {},
	// "header":   {},
	// "colgroup": {},
	// "col":      {},
}

var tagCatalog = map[string]struct{}{
	"blockquote": {},
	"code":       {},
	"del":        {},
	"fw":         {},
	"head":       {},
	"hi":         {},
	"lb":         {},
	"list":       {},
	"p":          {},
	"pre":        {},
	"quote":      {},
}

func duplicateMap(original map[string]struct{}) map[string]struct{} {
	duplicate := make(map[string]struct{})
	for key, val := range original {
		duplicate[key] = val
	}
	return duplicate
}
