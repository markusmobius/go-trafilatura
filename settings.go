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

// Code in this file is ported from <https://github.com/adbar/trafilatura>
// which available under Apache 2.0 license.

package trafilatura

var tagsToClean = sliceToMap(
	// important
	"aside", "embed", "footer", "form", "head", "iframe", "menu", "object", "script",
	// other content
	"applet", "audio", "canvas", "figure", "map", "picture", "svg", "video",
	// secondary
	"area", "blink", "button", "datalist", "dialog", "frame", "frameset", "fieldset",
	"link", "input", "ins", "label", "legend", "marquee", "math", "menuitem", "nav",
	"noscript", "optgroup", "option", "output", "param", "progress", "rp", "rt", "rtc",
	"select", "source", "style", "track", "textarea", "time", "use",
	// "meta", "hr", "img", "data", "details", "summary"
)

var tagsToStrip = sliceToMap(
	"abbr", "acronym", "address", "bdi", "bdo", "big", "cite", "data", "dfn", "font",
	"hgroup", "img", "ins", "mark", "meta", "ruby", "small", "template",
	"tbody", "tfoot", "thead",
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
	"details", "summary",
)

var formatTagCatalog = sliceToMap(
	"em", "i", "b", "strong", "u", "kbd",
	"samp", "tt", "var", "sub", "sup",
)

var validTagCatalog = sliceToMap(
	"a", "abbr", "address", "area", "b", "base", "bdo", "blockquote", "body", "br", "button",
	"caption", "cite", "code", "col", "colgroup", "dd", "del", "dfn", "div", "dl", "dt", "em",
	"fieldset", "form", "h1", "h2", "h3", "h4", "h5", "h6", "head", "hr", "html", "i", "iframe",
	"img", "input", "ins", "kbd", "label", "legend", "li", "link", "map", "menu", "meta",
	"noscript", "object", "ol", "optgroup", "option", "p", "param", "pre", "q", "s", "samp",
	"script", "select", "small", "span", "strong", "style", "sub", "sup", "table", "tbody",
	"td", "textarea", "tfoot", "th", "thead", "title", "tr", "u", "ul", "var", "article",
	"aside", "audio", "canvas", "command", "datalist", "details", "embed", "figcaption",
	"figure", "footer", "header", "mark", "meter", "nav", "output", "progress", "rp", "rt",
	"ruby", "section", "source", "summary", "time", "track", "video", "wbr")

var elementWithSizeAttr = sliceToMap("table", "th", "td", "hr", "pre")

// List of allowed attributes are taken from go-domdistiller
var allowedAttributes = sliceToMap(
	"abbr", "accept-charset", "accept", "accesskey", "action", "align", "alink",
	"allow", "allowfullscreen", "allowpaymentrequest", "alt", "archive", "as",
	"async", "autocapitalize", "autocomplete", "autocorrect", "autofocus",
	"autoplay", "autopictureinpicture", "axis", "background", "behavior",
	"bgcolor", "border", "bordercolor", "capture", "cellpadding", "cellspacing",
	"char", "challenge", "charoff", "charset", "checked", "cite", "class",
	"classid", "clear", "code", "codebase", "codetype", "color", "cols",
	"colspan", "compact", "content", "contenteditable", "controls",
	"controlslist", "conversiondestination", "coords", "crossorigin",
	"csp", "data", "datetime", "declare", "decoding", "default", "defer",
	"dir", "direction", "dirname", "disabled", "disablepictureinpicture",
	"disableremoteplayback", "disallowdocumentaccess", "download", "draggable",
	"elementtiming", "enctype", "end", "enterkeyhint", "event", "exportparts",
	"face", "for", "form", "formaction", "formenctype", "formmethod",
	"formnovalidate", "formtarget", "frame", "frameborder", "headers",
	"height", "hidden", "high", "href", "hreflang", "hreftranslate", "hspace",
	"http-equiv", "id", "imagesizes", "imagesrcset", "importance",
	"impressiondata", "impressionexpiry", "incremental", "inert", "inputmode",
	"integrity", "is", "ismap", "keytype", "kind", "invisible", "label", "lang",
	"language", "latencyhint", "leftmargin", "link", "list", "loading", "longdesc",
	"loop", "low", "lowsrc", "manifest", "marginheight", "marginwidth", "max",
	"maxlength", "mayscript", "media", "method", "min", "minlength", "multiple",
	"muted", "name", "nohref", "nomodule", "nonce", "noresize", "noshade",
	"novalidate", "nowrap", "object", "open", "optimum", "part", "pattern",
	"placeholder", "playsinline", "ping", "policy", "poster", "preload", "pseudo",
	"readonly", "referrerpolicy", "rel", "reportingorigin", "required", "resources",
	"rev", "reversed", "role", "rows", "rowspan", "rules", "sandbox", "scheme",
	"scope", "scrollamount", "scrolldelay", "scrolling", "select", "selected",
	"shadowroot", "shadowrootdelegatesfocus", "shape", "size", "sizes", "slot",
	"span", "spellcheck", "src", "srcset", "srcdoc", "srclang", "standby", "start",
	"step", "style", "summary", "tabindex", "target", "text", "title", "topmargin",
	"translate", "truespeed", "trusttoken", "type", "usemap", "valign", "value",
	"valuetype", "version", "vlink", "vspace", "virtualkeyboardpolicy",
	"webkitdirectory", "width", "wrap")

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

func inMap(key string, maps map[string]struct{}) bool {
	_, exist := maps[key]
	return exist
}
