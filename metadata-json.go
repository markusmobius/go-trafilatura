package trafilatura

import (
	"encoding/json"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/go-shiori/dom"
	"golang.org/x/net/html"
)

type SchemaData struct {
	Type       string
	Data       map[string]any
	Importance float64
}

// extractJsonLd search metadata from JSON+LD data following the Schema.org guidelines
// (https://schema.org). Here we don't really care about error here, so if parse failed
// we just return the original metadata.
func extractJsonLd(opts Options, doc *html.Node, originalMetadata Metadata) Metadata {
	var metadata Metadata

	// Decode all script nodes that contain JSON+Ld schema
	persons, organizations, articles := decodeJsonLd(doc, opts)

	// Extract metadata from each article
	for _, article := range articles {
		// Grab "author" property from schema with @type "Person"
		if metadata.Author == "" {
			var validAuthors []string
			for _, author := range getSchemaNames(article.Data["author"], "person") {
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
			if sitenames := getSchemaNames(article.Data["publisher"]); len(sitenames) > 0 {
				metadata.Sitename = sitenames[0]
			}
		}

		// Grab category
		category := trim(getValue[string](article.Data, "articleSection"))
		if category != "" {
			metadata.Categories = append(metadata.Categories, category)
		}

		// Grab tags
		tags := getSchemaNames(article.Data["keywords"])
		if len(tags) > 0 {
			metadata.Tags = append(metadata.Tags, tags...)
		}

		// Grab title
		if metadata.Title == "" {
			metadata.Title = trim(getValue[string](article.Data, "name"))
		}

		// If title is empty or only consist of one word, try to look in headline
		if metadata.Title == "" || strWordCount(metadata.Title) == 1 {
			for attr := range article.Data {
				if !strings.Contains(strings.ToLower(attr), "headline") {
					continue
				}

				title := trim(getValue[string](article.Data, attr))
				if title != "" && !strings.Contains(title, "...") {
					metadata.Title = title
					break
				}
			}
		}

		// If title found, use article type as page type
		if metadata.PageType == "" && metadata.Title != "" {
			metadata.PageType = article.Type
		}
	}

	// If author not found, look in persons
	if metadata.Author == "" {
		names := []string{}
		for _, person := range persons {
			for _, name := range getSchemaNames(person.Data) {
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
			for _, name := range getSchemaNames(org.Data) {
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

	// If type not found, use the first article type
	if metadata.PageType == "" && len(articles) > 0 {
		metadata.PageType = articles[0].Type
	}

	// Uniquify tags and categories
	metadata.Tags = uniquifyLists(metadata.Tags...)
	metadata.Categories = uniquifyLists(metadata.Categories...)

	// If available, override type, title, author, categories and tags in original metadata
	originalMetadata.Title = strOr(originalMetadata.Title, metadata.Title)
	originalMetadata.PageType = strOr(originalMetadata.PageType, metadata.PageType)
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

	return originalMetadata
}

func decodeJsonLd(doc *html.Node, opts Options) (persons, organizations, articles []SchemaData) {
	// Prepare function to find articles and persons inside JSON+LD recursively
	var findImportantObjects func(obj map[string]any)
	findImportantObjects = func(obj map[string]any) {
		// Schema type could be either string or slices, so extract it properly
		schemaTypes := getSchemaTypes(obj, false)

		for _, schemaType := range schemaTypes {
			schemaData := SchemaData{Type: schemaType, Data: obj}
			schemaType = strings.ToLower(schemaType)

			// Check if it's person
			if schemaType == "person" {
				persons = append(persons, schemaData)
				break
			}

			// Check if it's organization or website.
			isWebsite := schemaType == "website"
			isOrganization := strings.Contains(schemaType, "organization")

			if isWebsite || isOrganization {
				// Organization is more important than website.
				switch {
				case isOrganization:
					schemaData.Importance = 2
				default:
					schemaData.Importance = 1
				}

				organizations = append(organizations, schemaData)
				break
			}

			// Check if it's article, blog or page.
			isArticle := strings.Contains(schemaType, "article")
			isPosting := strings.Contains(schemaType, "posting")
			isReport := schemaType == "report"
			isBlog := schemaType == "blog"
			isPage := strings.Contains(schemaType, "page")
			isListing := strings.Contains(schemaType, "listing")

			if isArticle || isPosting || isReport || isBlog || isPage || isListing {
				// Adjust its importance level
				switch {
				case isArticle, isPosting, isReport:
					schemaData.Importance = 3
				case isBlog:
					schemaData.Importance = 2
				case isPage, isListing:
					schemaData.Importance = 1
				}

				articles = append(articles, schemaData)
				break
			}
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

	// Find all script nodes that contain JSON+Ld schema
	scriptNodes1 := dom.QuerySelectorAll(doc, `script[type="application/ld+json"]`)
	scriptNodes2 := dom.QuerySelectorAll(doc, `script[type="application/settings+json"]`)
	scriptNodes := append(scriptNodes1, scriptNodes2...)

	for _, script := range scriptNodes {
		// Get the json text inside the script
		jsonLdText := dom.TextContent(script)
		jsonLdText = strings.TrimSpace(jsonLdText)
		jsonLdText = html.UnescapeString(jsonLdText)
		if jsonLdText == "" {
			continue
		}

		// Decode JSON text assuming it is an array
		var dataList []map[string]any
		jsonLdByte := []byte(jsonLdText)
		err := json.Unmarshal(jsonLdByte, &dataList)
		if err != nil {
			// If not succeed, try it as an object
			var data map[string]any
			err = json.Unmarshal(jsonLdByte, &data)
			if err == nil {
				dataList = []map[string]any{data}
			} else {
				logWarn(opts, "error in JSON metadata extraction: %v", err)
				continue
			}
		}

		// Extract each data
		for _, data := range dataList {
			findImportantObjects(data)
		}
	}

	// Sort schemas based on importance
	sort.SliceStable(organizations, func(i, j int) bool {
		return organizations[i].Importance > organizations[j].Importance
	})

	sort.SliceStable(articles, func(i, j int) bool {
		return articles[i].Importance > articles[j].Importance
	})

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
		if len(expectedTypes) > 0 {
			if len(schemaTypes) == 0 {
				return nil
			}

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
		name := trim(getValue[string](value, "name"))

		// If name is empty and its type is Person, try name combination
		if name == "" && strIn("person", schemaTypes...) {
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
	schemaRawType, exist := schema["@type"]
	if !exist {
		return nil
	}

	var schemaTypes []string
	switch tp := schemaRawType.(type) {
	case string:
		if toLower {
			tp = strings.ToLower(tp)
		}
		schemaTypes = []string{tp}

	case []any:
		for _, entry := range tp {
			if strType, isString := entry.(string); isString {
				if toLower {
					strType = strings.ToLower(strType)
				}
				schemaTypes = append(schemaTypes, strType)
			}
		}
	}

	return schemaTypes
}
