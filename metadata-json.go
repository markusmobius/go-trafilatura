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

		// Decode JSON+LD text
		persons, organizations, articles, err := decodeJsonLd(jsonLdText)
		if err != nil {
			logWarn(opts, "error in JSON metadata extraction: %v", err)
			continue
		}

		// Extract metadata from each article
		for _, article := range articles {
			// Grab "author" property from schema with @type "Person"
			if metadata.Author == "" {
				var validAuthors []string
				for _, author := range getSchemaNames(article["author"], "Person") {
					author = validateMetadataName(author)
					author = normalizeAuthors("", author)
					if author != "" {
						validAuthors = append(validAuthors, author)
					}
				}

				if len(validAuthors) > 0 {
					metadata.Author = strings.Join(validAuthors, "; ")
				}
			}

			// Grab sitename
			if metadata.Sitename == "" {
				if sitenames := getSchemaNames(article["publisher"]); len(sitenames) > 0 {
					metadata.Sitename = sitenames[0]
				}
			}

			// Grab category
			category := trim(getValue[string](article, "articleSection"))
			if category != "" {
				metadata.Categories = append(metadata.Categories, category)
			}

			// Grab tags
			tags := getSchemaNames(article["keywords"])
			if len(tags) > 0 {
				metadata.Tags = append(metadata.Tags, tags...)
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
				for _, name := range getSchemaNames(person) {
					name = validateMetadataName(name)
					name = normalizeAuthors("", name)
					if name != "" {
						names = append(names, name)
					}
				}
			}

			if len(names) > 0 {
				metadata.Author = strings.Join(names, "; ")
			}
		}

		// If sitename not found, look in organizations
		if metadata.Sitename == "" {
			names := []string{}
			for _, org := range organizations {
				for _, name := range getSchemaNames(org) {
					name = validateMetadataName(name)
					if name != "" {
						names = append(names, name)
					}
				}
			}

			if len(names) > 0 {
				metadata.Sitename = strings.Join(names, "; ")
			}
		}

		// Stop if all metadata found
		if metadata.Author != "" && metadata.Sitename != "" &&
			len(metadata.Categories) != 0 && metadata.Title != "" {
			break
		}
	}

	// Uniquify tags and categories
	metadata.Tags = uniquifyLists(metadata.Tags...)
	metadata.Categories = uniquifyLists(metadata.Categories...)

	// If available, override author, categories and tags in original metadata
	originalMetadata.Author = strOr(metadata.Author, originalMetadata.Author)

	if len(metadata.Categories) > 0 {
		originalMetadata.Categories = metadata.Categories
	}

	if len(metadata.Tags) > 0 {
		originalMetadata.Tags = metadata.Tags
	}

	// If the new sitename exist and longer, override the original
	if utf8.RuneCountInString(metadata.Sitename) > utf8.RuneCountInString(originalMetadata.Sitename) {
		originalMetadata.Sitename = metadata.Sitename
	}

	// The new title is only used if original metadata doesn't have any title
	if originalMetadata.Title == "" {
		originalMetadata.Title = metadata.Title
	}

	return originalMetadata
}

func decodeJsonLd(rawJsonLd string) (persons, organizations, articles []map[string]any, err error) {
	// Prepare function to find articles and persons inside JSON+LD recursively
	var findImportantObjects func(obj map[string]any)
	findImportantObjects = func(obj map[string]any) {
		// Check if this object is either Article or Person
		objType := getValue[string](obj, "@type")
		switch {
		case objType == "Person":
			persons = append(persons, obj)

		case objType == "WebSite",
			strings.Contains(objType, "Organization"):
			organizations = append(organizations, obj)

		case strings.Contains(objType, "Article"), strings.Contains(objType, "Posting"), objType == "Report":
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

	// First decode JSON text assuming it is an array
	var dataList []map[string]any
	err = json.Unmarshal([]byte(rawJsonLd), &dataList)
	if err != nil {
		// If not succeed, try it as an object
		var data map[string]any
		err = json.Unmarshal([]byte(rawJsonLd), &data)
		if err == nil {
			dataList = []map[string]any{data}
		} else {
			return
		}
	}

	// Look and return
	for _, data := range dataList {
		findImportantObjects(data)
	}

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

func getSchemaNames(v any, schemaTypes ...string) []string {
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
		// If there are schema types specified, make sure this schema is one of those types.
		// If not, we just return empty handed.
		schemaType := getValue[string](value, "@type")
		if len(schemaTypes) > 0 && (schemaType == "" || !strIn(schemaType, schemaTypes...)) {
			return nil
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

		// If name is found, we can return it
		if name != "" {
			return []string{name}
		}

		// At this found, since name still not found, there is a possibility that the JSON+LD use
		// name with uncommon format, so here we try to treat it as schema or array.
		switch childValue := value["name"].(type) {
		case map[string]any, []any:
			return getSchemaNames(childValue, schemaTypes...)
		}

		// If nothing else, return nil
		return nil
	}

	// Finally, check if its array
	if values, isArray := v.([]any); isArray {
		var names []string
		for _, value := range values {
			if subNames := getSchemaNames(value, schemaTypes...); len(subNames) > 0 {
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
