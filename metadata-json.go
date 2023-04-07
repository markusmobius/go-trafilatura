package trafilatura

import (
	"encoding/json"
	"strings"
	"unicode/utf8"

	"github.com/go-shiori/dom"
	"golang.org/x/net/html"
)

// extractJsonLd search metadata from JSON+LD data following the Schema.org guidelines
// (https://schema.org). Here we don't really care about error here, so if parse failed
// we just return the original metadata.
func extractJsonLd(opts Options, doc *html.Node, originalMetadata Metadata) Metadata {
	// Find all script nodes that contain JSON+Ld schema
	scriptNodes1 := dom.QuerySelectorAll(doc, `script[type="application/ld+json"]`)
	scriptNodes2 := dom.QuerySelectorAll(doc, `script[type="application/settings+json"]`)
	scriptNodes := append(scriptNodes1, scriptNodes2...)

	// Process each script node
	var metadata Metadata
	for _, script := range scriptNodes {
		// Get the json text inside the script
		jsonLdText := dom.TextContent(script)
		jsonLdText = strings.TrimSpace(jsonLdText)
		if jsonLdText == "" {
			continue
		}

		// Decode JSON text, assuming it is an object
		data := map[string]interface{}{}
		err := json.Unmarshal([]byte(jsonLdText), &data)
		if err != nil {
			logWarn(opts, "error in JSON metadata extraction: %v", err)
			continue
		}

		// Find articles and persons inside JSON+LD recursively
		persons := make([]map[string]interface{}, 0)
		articles := make([]map[string]interface{}, 0)

		var findImportantObjects func(obj map[string]interface{})
		findImportantObjects = func(obj map[string]interface{}) {
			// First check if this object type matches with our need.
			if objType, hasType := obj["@type"]; hasType {
				if strObjType, isString := objType.(string); isString {
					isPerson := strObjType == "Person"
					isArticle := strings.Contains(strObjType, "Article") ||
						strings.Contains(strObjType, "Posting") ||
						strObjType == "Report"

					switch {
					case isArticle:
						articles = append(articles, obj)
						return

					case isPerson:
						persons = append(persons, obj)
						return
					}
				}
			}

			// If not, look in its children
			for _, value := range obj {
				switch v := value.(type) {
				case map[string]interface{}:
					findImportantObjects(v)

				case []interface{}:
					for _, item := range v {
						itemObject, isObject := item.(map[string]interface{})
						if isObject {
							findImportantObjects(itemObject)
						}
					}
				}
			}
		}

		findImportantObjects(data)

		// Extract metadata from each article
		for _, article := range articles {
			if metadata.Author == "" {
				// For author, if taken from schema, we only want it from schema with type "Person"
				metadata.Author = extractJsonArticleThingName(article, "author", "Person")
				metadata.Author = validateMetadataAuthor(metadata.Author)
			}

			if metadata.Sitename == "" {
				metadata.Sitename = extractJsonArticleThingName(article, "publisher")
			}

			if len(metadata.Categories) == 0 {
				if section, exist := article["articleSection"]; exist {
					category := extractJsonString(section)
					metadata.Categories = append(metadata.Categories, category)
				}
			}

			if metadata.Title == "" {
				if name, exist := article["name"]; exist {
					metadata.Title = extractJsonString(name)
				}
			}

			// If title is empty or only consist of one word, try to look in headline
			if metadata.Title == "" || strWordCount(metadata.Title) == 1 {
				for key, value := range article {
					if !strings.Contains(strings.ToLower(key), "headline") {
						continue
					}

					title := extractJsonString(value)
					if title == "" || strings.Contains(title, "...") {
						continue
					}

					metadata.Title = title
					break
				}
			}
		}

		// If author not found, look in persons
		if metadata.Author == "" {
			names := []string{}
			for _, person := range persons {
				personName := extractJsonThingName(person)
				personName = validateMetadataAuthor(personName)
				if personName != "" {
					names = append(names, personName)
				}
			}

			if len(names) > 0 {
				metadata.Author = strings.Join(names, "; ")
			}
		}

		// Stop if all metadata found
		if metadata.Author != "" && metadata.Sitename != "" &&
			len(metadata.Categories) != 0 && metadata.Title != "" {
			break
		}
	}

	// If available, override author and categories in original metadata
	originalMetadata.Author = strOr(metadata.Author, originalMetadata.Author)
	if len(metadata.Categories) > 0 {
		originalMetadata.Categories = metadata.Categories
	}

	// If the new sitename exist and longer, override the original
	if utf8.RuneCountInString(metadata.Sitename) > utf8.RuneCountInString(originalMetadata.Sitename) {
		originalMetadata.Sitename = metadata.Sitename
	}

	// The new title is only used if original metadata doesn't have any title
	if originalMetadata.Title == "" {
		originalMetadata.Title = metadata.Title
	}

	// Clean up authors
	originalMetadata.Author = normalizeAuthors("", originalMetadata.Author)

	return originalMetadata
}

func extractJsonArticleThingName(article map[string]interface{}, key string, allowedTypes ...string) string {
	// Fetch value from the key
	value, exist := article[key]
	if !exist {
		return ""
	}

	return extractJsonThingName(value, allowedTypes...)
}

func extractJsonThingName(iface interface{}, allowedTypes ...string) string {
	// Decode the value of interface
	switch val := iface.(type) {
	case string:
		// There are some case where the string contains an unescaped
		// JSON, so try to handle it here
		if rxJsonSymbol.MatchString(val) {
			matches := rxNameJson.FindStringSubmatch(val)
			if len(matches) == 0 {
				return ""
			}
			val = matches[1]
		}

		// Clean up the string
		return trim(val)

	case map[string]interface{}:
		// If it's object, make sure its type allowed
		if len(allowedTypes) > 0 {
			if objType, hasType := val["@type"]; hasType {
				if strObjType, isString := objType.(string); isString {
					if !strIn(strObjType, allowedTypes...) {
						return ""
					}
				}
			}
		}

		// Return its name
		if iName, exist := val["name"]; exist {
			return extractJsonString(iName)
		}

	case []interface{}:
		// If it's array, merge names into one
		names := []string{}
		for _, entry := range val {
			switch entryVal := entry.(type) {
			case string:
				entryVal = trim(entryVal)
				names = append(names, entryVal)

			case map[string]interface{}:
				if iName, exist := entryVal["name"]; exist {
					if name := extractJsonString(iName); name != "" {
						names = append(names, name)
					}
				}
			}
		}

		if len(names) > 0 {
			return strings.Join(names, "; ")
		}
	}

	return ""
}

func extractJsonString(iface interface{}) string {
	if s, isString := iface.(string); isString {
		return trim(s)
	}

	return ""
}
