package trafilatura

var (
	listXmlListTags    = []string{"ul", "ol", "dl"}
	listXmlQuoteTags   = []string{"blockquote", "pre", "q"}
	listXmlHeadTags    = []string{"h1", "h2", "h3", "h4", "h5", "h6", "summary"}
	listXmlLbTags      = []string{"br", "hr", "lb"}
	listXmlHiTags      = []string{"em", "i", "b", "strong", "u", "kbd", "samp", "tt", "var", "sub", "sup"}
	listXmlRefTags     = []string{"a"}
	listXmlGraphicTags = []string{"img"}
	listXmlItemTags    = []string{"dd", "dt", "li"}
	listXmlCellTags    = []string{"th", "td"}
)

var (
	mapXmlListTags    = sliceToMap(listXmlListTags...)
	mapXmlQuoteTags   = sliceToMap(listXmlQuoteTags...)
	mapXmlHeadTags    = sliceToMap(listXmlHeadTags...)
	mapXmlLbTags      = sliceToMap(listXmlLbTags...)
	mapXmlHiTags      = sliceToMap(listXmlHiTags...)
	mapXmlRefTags     = sliceToMap(listXmlRefTags...)
	mapXmlGraphicTags = sliceToMap(listXmlGraphicTags...)
	mapXmlItemTags    = sliceToMap(listXmlItemTags...)
	mapXmlCellTags    = sliceToMap(listXmlCellTags...)
)
