package trafilatura

import (
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/go-shiori/dom"
	"golang.org/x/net/html"
)

type SchemaData struct {
	Types      []string
	Data       map[string]any
	Importance float64
	Parent     *SchemaData
}

// extractJsonLd search metadata from JSON+LD data following the Schema.org guidelines
// (https://schema.org). Here we don't really care about error here, so if parse failed
// we just return the original metadata.
func extractJsonLd(opts Options, doc *html.Node, metadata Metadata) Metadata {
	for _, script := range dom.QuerySelectorAll(doc, `script[type="application/ld+json"], script[type="application/settings+json"]`) {
		input := normalizeJSONText(dom.TextContent(script))
		if input == "" {
			continue
		}
		var value any
		if err := decodeJSON(input, &value); err != nil {
			logWarn(opts, "error in JSON metadata extraction: %v", err)
			metadata = recoverJSONMetadata(input, metadata)
		} else {
			metadata = extractJSONMetadata(value, metadata)
		}
	}
	return metadata
}

var jsonArticleTypes = sliceToMap("article", "backgroundnewsarticle", "blogposting", "medicalscholarlyarticle", "newsarticle", "opinionnewsarticle", "reportagenewsarticle", "scholarlyarticle", "socialmediaposting", "liveblogposting")
var jsonPageTypes = sliceToMap("aboutpage", "checkoutpage", "collectionpage", "contactpage", "faqpage", "itempage", "medicalwebpage", "profilepage", "qapage", "realestatelisting", "searchresultspage", "webpage", "website", "article", "advertisercontentarticle", "newsarticle", "analysisnewsarticle", "askpublicnewsarticle", "backgroundnewsarticle", "opinionnewsarticle", "reportagenewsarticle", "reviewnewsarticle", "report", "satiricalarticle", "scholarlyarticle", "medicalscholarlyarticle", "socialmediaposting", "blogposting", "liveblogposting", "discussionforumposting", "techarticle", "blog", "jobposting")
var rxJSONContext = regexp.MustCompile(`(?i)^https?://schema\.org`)
var rxJSONUnicode = regexp.MustCompile(`\\u[0-9a-fA-F]{4}`)
var rxJSONAuthor = regexp.MustCompile(`(?s)"author"\s*:[^}\[]+?"name?\\?"\s*:\s*\\?"([^"\\]+)|"author"[^}\[]+?"names?".+?"([^"]+)`)
var rxJSONPerson = regexp.MustCompile(`(?s)"[Pp]erson"[^}]+?"names?".+?"([^"]+)`)
var rxJSONAuthorRemove = regexp.MustCompile(`,?(?:"\w+"\s*:?[:|,\[])?\{?"@type"\s*:\s*"(?:[Ii]mageObject|[Oo]rganization|[Ww]eb[Pp]age)",[^}\[]+}[\]|}]?`)
var rxJSONPublisher = regexp.MustCompile(`(?s)"publisher"\s*:[^}]+?"name?\\?"\s*:\s*\\?"([^"\\]+)`)
var rxJSONType = regexp.MustCompile(`(?s)"@type"\s*:\s*"([^"]*)"`)
var rxJSONCategory = regexp.MustCompile(`(?s)"articleSection"\s*:\s*"([^"\\]+)`)
var rxJSONArticleName = regexp.MustCompile(`(?s)"@type"\s*:\s*"[Aa]rticle",\s*"name"\s*:\s*"([^"\\]+)`)
var rxJSONHeadline = regexp.MustCompile(`(?s)"headline"\s*:\s*"([^"\\]+)`)

func normalizeJSONText(input string) string {
	if strings.Contains(input, `\`) {
		input = strings.NewReplacer(`\n`, "", `\r`, "", `\t`, "").Replace(input)
		input = rxJSONUnicode.ReplaceAllStringFunc(input, func(escape string) string {
			value, _ := strconv.ParseUint(escape[2:], 16, 16)
			if value >= 0xd800 && value <= 0xdfff {
				return ""
			}
			return string(rune(value))
		})
		input = html.UnescapeString(input)
	}
	return trim(rxHtmlStripTag.ReplaceAllString(input, ""))
}

func plausibleJSONSitename(current, candidate, contentType string) bool {
	return candidate != "" && (current == "" || (utf8.RuneCountInString(candidate) > utf8.RuneCountInString(current) && contentType != "webpage") || (strings.HasPrefix(current, "http") && !strings.HasPrefix(candidate, "http")))
}

func processJSONMetadata(parents any, metadata Metadata) Metadata {
	for _, item := range jsonItems(parents) {
		content, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if publisher, ok := content["publisher"].(map[string]any); ok {
			candidate, _ := publisher["name"].(string)
			if plausibleJSONSitename(metadata.Sitename, candidate, "") {
				metadata.Sitename = candidate
			}
		}
		types := getSchemaTypes(content, true)
		if len(types) == 0 {
			continue
		}
		contentType := types[0]
		if metadata.PageType == "" && inMap(contentType, jsonPageTypes) {
			metadata.PageType = normalizeJSONText(contentType)
		}
		switch {
		case strIn(contentType, "newsmediaorganization", "organization", "webpage", "website"):
			candidate := strOr(getSingleStringValue(content, "name"), getSingleStringValue(content, "legalName"), getSingleStringValue(content, "alternateName"))
			if plausibleJSONSitename(metadata.Sitename, candidate, contentType) {
				metadata.Sitename = candidate
			}
		case contentType == "person":
			if name, ok := content["name"].(string); ok && !strings.HasPrefix(name, "http") {
				metadata.Author = normalizeAuthors(metadata.Author, name)
			}
		case inMap(contentType, jsonArticleTypes):
			authors := content["author"]
			if text, ok := authors.(string); ok {
				var decoded any
				if decodeJSON(text, &decoded) == nil {
					authors = decoded
				}
			}
			for _, author := range jsonItems(authors) {
				if name, ok := author.(string); ok {
					metadata.Author = normalizeAuthors(metadata.Author, name)
					continue
				}
				object, ok := author.(map[string]any)
				if !ok {
					continue
				}
				if authorType, exists := object["@type"]; exists && authorType != "Person" {
					continue
				}
				name := strings.Join(getSchemaNames(object), "; ")
				if name == "" && object["givenName"] != nil && object["familyName"] != nil {
					name = trim(getSingleStringValue(object, "givenName") + " " + getSingleStringValue(object, "additionalName") + " " + getSingleStringValue(object, "familyName"))
				}
				metadata.Author = normalizeAuthors(metadata.Author, name)
			}
			if len(metadata.Categories) == 0 {
				metadata.Categories = getStringValues(content, "articleSection")
			}
			if metadata.Title == "" {
				if contentType == "article" {
					metadata.Title = getSingleStringValue(content, "name")
				}
				if metadata.Title == "" {
					metadata.Title = getSingleStringValue(content, "headline")
				}
			}
		}
	}
	return metadata
}

func extractJSONMetadata(value any, metadata Metadata) Metadata {
	var parents []any
	for _, item := range jsonItems(value) {
		parent, ok := item.(map[string]any)
		if !ok {
			continue
		}
		context, _ := parent["@context"].(string)
		if !rxJSONContext.MatchString(context) {
			continue
		}
		if graph, exists := parent["@graph"]; exists {
			parents = append(parents, jsonItems(graph)...)
		} else if contentType, ok := parent["@type"].(string); ok && strings.Contains(strings.ToLower(contentType), "liveblogposting") && parent["liveBlogUpdate"] != nil {
			parents = append(parents, jsonItems(parent["liveBlogUpdate"])...)
		} else {
			parents = append(parents, parent)
		}
	}
	return processJSONMetadata(parents, metadata)
}

func extractJSONAuthors(input string, pattern *regexp.Regexp) string {
	var authors string
	for {
		match := pattern.FindStringSubmatchIndex(input)
		if match == nil {
			break
		}
		var name string
		for index := 2; index < len(match); index += 2 {
			if match[index] >= 0 {
				name = input[match[index]:match[index+1]]
				break
			}
		}
		if !strings.Contains(name, " ") {
			break
		}
		authors = normalizeAuthors(authors, name)
		input = input[:match[0]] + input[match[1]:]
	}
	return authors
}

func recoverJSONMetadata(input string, metadata Metadata) Metadata {
	authorInput := rxJSONAuthorRemove.ReplaceAllString(input, "")
	if authors := strOr(extractJSONAuthors(authorInput, rxJSONAuthor), extractJSONAuthors(authorInput, rxJSONPerson)); authors != "" {
		metadata.Author = authors
	}
	if match := rxJSONType.FindStringSubmatch(input); len(match) > 1 {
		if candidate := normalizeJSONText(strings.ToLower(match[1])); inMap(candidate, jsonPageTypes) {
			metadata.PageType = candidate
		}
	}
	if match := rxJSONPublisher.FindStringSubmatch(input); len(match) > 1 && !strings.Contains(match[1], ",") {
		if candidate := normalizeJSONText(match[1]); plausibleJSONSitename(metadata.Sitename, candidate, "") {
			metadata.Sitename = candidate
		}
	}
	if match := rxJSONCategory.FindStringSubmatch(input); len(match) > 1 {
		metadata.Categories = []string{normalizeJSONText(match[1])}
	}
	for _, pattern := range []*regexp.Regexp{rxJSONArticleName, rxJSONHeadline} {
		if match := pattern.FindStringSubmatch(input); metadata.Title == "" && len(match) > 1 {
			metadata.Title = normalizeJSONText(match[1])
		}
	}
	return metadata
}

func getSchemaNames(v any, expectedTypes ...string) []string {
	// First, check if its string
	if value, isString := v.(string); isString {
		// There are some case where the name string contains an unescaped JSON,
		// so try to handle it here.
		parts := rxNameJson.FindStringSubmatch(value)
		if rxJsonSymbol.MatchString(value) && len(parts) > 0 {
			value = parts[1]
		}

		// Return cleaned up string
		if value = trim(value); value != "" {
			return []string{value}
		} else {
			return nil
		}
	}

	// Second, check if its schema
	if value, isObject := v.(map[string]any); isObject {
		// If there are expected types specified, make sure this schema is one of those types.
		// If not, we just return empty handed.
		schemaTypes := getSchemaTypes(value, true)
		if len(expectedTypes) > 0 && len(schemaTypes) > 0 {
			var schemaAllowed bool
			for _, schemaType := range schemaTypes {
				if strIn(schemaType, expectedTypes...) {
					schemaAllowed = true
					break
				}
			}

			if !schemaAllowed {
				return nil
			}
		}

		// If this schema has "name" string property, try it
		names := getStringValues(value, "name")

		// If name is empty and its type is Person, try name combination
		if len(names) == 0 && strIn("person", schemaTypes...) {
			givenName := getSingleStringValue(value, "givenName")
			additionalName := getSingleStringValue(value, "additionalName")
			familyName := getSingleStringValue(value, "familyName")
			fullName := trim(givenName + " " + additionalName + " " + familyName)
			if fullName != "" {
				names = []string{fullName}
			}
		}

		// If name still empty, try its legal name
		if len(names) == 0 {
			names = getStringValues(value, "legalName")
		}

		// If name still empty, next try its alternate name
		if len(names) == 0 {
			names = getStringValues(value, "alternateName")
		}

		// If name is found, we can return it
		if len(names) != 0 {
			return names
		}

		// At this point name is still not found, so there is a possibility that the
		// JSON+LD use name with uncommon format. Here we try to treat it as schema or array.
		switch childValue := value["name"].(type) {
		case map[string]any, []any:
			return getSchemaNames(childValue, expectedTypes...)
		}

		// If nothing else, return nil
		return nil
	}

	// Finally, check if its array
	if values, isArray := v.([]any); isArray {
		var names []string
		for _, value := range values {
			if subNames := getSchemaNames(value, expectedTypes...); len(subNames) > 0 {
				names = append(names, subNames...)
			}
		}

		if len(names) > 0 {
			return names
		} else {
			return nil
		}
	}

	// If nothing found, just return empty
	return nil
}

func getSchemaTypes(schema map[string]any, toLower bool) []string {
	schemaTypes := getStringValues(schema, "@type")
	if toLower {
		for i, tp := range schemaTypes {
			schemaTypes[i] = strings.ToLower(tp)
		}
	}

	return schemaTypes
}

func getStringValues(obj map[string]any, key string) []string {
	var result []string

	switch value := obj[key].(type) {
	case string:
		if cleanStr := trim(value); cleanStr != "" {
			result = []string{cleanStr}
		}

	case []any:
		result = []string{}
		for _, item := range value {
			str, ok := item.(string)
			if !ok {
				continue
			}

			if cleanStr := trim(str); cleanStr != "" {
				result = append(result, cleanStr)
			}
		}
	}

	return result
}

func getSingleStringValue(obj map[string]any, key string) string {
	values := getStringValues(obj, key)
	if len(values) > 0 {
		return values[0]
	}
	return ""
}
