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
	categoryTracker := map[string]struct{}{}

	for _, script := range scriptNodes {
		// Get the json text inside the script
		jsonLdText := dom.TextContent(script)
		jsonLdText = strings.TrimSpace(jsonLdText)
		if jsonLdText == "" {
			continue
		}

		// Decode JSON+LD text
		persons, articles, err := decodeJsonLd(jsonLdText)
		if err != nil {
			logWarn(opts, "error in JSON metadata extraction: %v", err)
			continue
		}

		// Extract metadata from each article
		for _, article := range articles {
			// Grab "author" property from schema with @type "Person"
			if metadata.Author == "" {
				metadata.Author = getSchemaName(article["author"], "Person")
				metadata.Author = validateMetadataAuthor(metadata.Author)
			}

			// Grab sitename
			if metadata.Sitename == "" {
				metadata.Sitename = getSchemaName(article["publisher"])
			}

			// Grab category
			category := trim(getValue[string](article, "articleSection"))
			if _, tracked := categoryTracker[category]; category != "" && !tracked {
				categoryTracker[category] = struct{}{}
				metadata.Categories = append(metadata.Categories, category)
			}

			// Grab title
			if metadata.Title == "" {
				metadata.Title = trim(getValue[string](article, "name"))
			}

			// If title is empty or only consist of one word, try to look in headline
			if metadata.Title == "" || strWordCount(metadata.Title) == 1 {
				for attr := range article {
					if !strings.Contains(strings.ToLower(attr), "headline") {
						continue
					}

					title := trim(getValue[string](article, attr))
					if title != "" && !strings.Contains(title, "...") {
						metadata.Title = title
						break
					}
				}
			}
		}

		// If author not found, look in persons
		if metadata.Author == "" {
			names := []string{}
			for _, person := range persons {
				name := getSchemaName(person)
				name = validateMetadataAuthor(name)
				if name != "" {
					names = append(names, name)
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

func decodeJsonLd(rawJsonLd string) (persons, articles []map[string]any, err error) {
	// Decode JSON text, assuming it is an object
	data := map[string]any{}
	err = json.Unmarshal([]byte(rawJsonLd), &data)
	if err != nil {
		return
	}

	// Find articles and persons inside JSON+LD recursively
	var findImportantObjects func(obj map[string]any)
	findImportantObjects = func(obj map[string]any) {
		// Check if this object is either Article or Person
		objType := getValue[string](obj, "@type")
		switch {
		case objType == "Person":
			persons = append(persons, obj)

		case strings.Contains(objType, "Article"),
			strings.Contains(objType, "Posting"),
			objType == "Report":
			articles = append(articles, obj)
		}

		// Continue to look in its sub values
		for _, value := range obj {
			switch v := value.(type) {
			case map[string]any:
				findImportantObjects(v)

			case []any:
				for _, item := range v {
					if subObj, isObj := item.(map[string]any); isObj {
						findImportantObjects(subObj)
					}
				}
			}
		}
	}

	// Look and return
	findImportantObjects(data)
	return
}

func getValue[T comparable](obj map[string]any, key string) T {
	// If value is T type, return it
	value := obj[key]
	if v, isT := value.(T); isT {
		return v
	}

	// If not, return its zero value
	var zero T
	return zero
}

func getSchemaName(v any, schemaTypes ...string) string {
	// First, check if its string
	if value, isString := v.(string); isString {
		// There are some case where the name string contains an unescaped JSON,
		// so try to handle it here.
		parts := rxNameJson.FindStringSubmatch(value)
		if rxJsonSymbol.MatchString(value) && len(parts) > 0 {
			value = parts[1]
		}

		// Return cleaned up string
		return trim(value)
	}

	// Second, check if its schema
	if value, isObject := v.(map[string]any); isObject {
		// If there are schema types specified, make sure this schema is one of those types.
		// If not, we just return empty handed.
		schemaType := getValue[string](value, "@type")
		if len(schemaTypes) > 0 && (schemaType == "" || !strIn(schemaType, schemaTypes...)) {
			return ""
		}

		// If this schema has "name" string property, try it
		name := trim(getValue[string](value, "name"))

		// If name is empty and its @type is Person, try name combination
		if name == "" && schemaType == "Person" {
			givenName := getValue[string](value, "givenName")
			additionalName := getValue[string](value, "additionalName")
			familyName := getValue[string](value, "familyName")
			name = trim(givenName + " " + additionalName + " " + familyName)
		}

		// If name still empty, try alternate name
		if name == "" {
			name = trim(getValue[string](value, "alternateName"))
		}

		return name
	}

	// Finally, check if its array
	if values, isArray := v.([]any); isArray {
		var names []string
		for _, value := range values {
			if name := getSchemaName(value, schemaTypes...); name != "" {
				names = append(names, name)
			}
		}

		if len(names) > 0 {
			return strings.Join(names, "; ")
		}
	}

	// If nothing found, just return empty
	return ""
}
