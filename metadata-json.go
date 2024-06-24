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
	Types      []string
	Data       map[string]any
	Importance float64
	Parent     *SchemaData
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
			var authorNames string
			for _, author := range getSchemaNames(article.Data["author"], "person") {
				author = validateMetadataName(author)
				authorNames = normalizeAuthors(authorNames, author)
			}

			if authorNames != "" {
				metadata.Author = authorNames
			}
		}

		// Grab sitename
		if metadata.Sitename == "" {
			if sitenames := getSchemaNames(article.Data["publisher"]); len(sitenames) > 0 {
				metadata.Sitename = sitenames[0]
			}
		}

		// Grab category
		categories := getStringValues(article.Data, "articleSection")
		if len(categories) != 0 {
			metadata.Categories = append(metadata.Categories, categories...)
		}

		// Grab tags
		tags := getSchemaNames(article.Data["keywords"])
		if len(tags) > 0 {
			metadata.Tags = append(metadata.Tags, tags...)
		}

		// Grab title
		if metadata.Title == "" {
			metadata.Title = getSingleStringValue(article.Data, "name")
		}

		// If title is empty or only consist of one word, try to look in headline
		if metadata.Title == "" || strWordCount(metadata.Title) == 1 {
			for attr := range article.Data {
				if !strings.Contains(strings.ToLower(attr), "headline") {
					continue
				}

				title := getSingleStringValue(article.Data, attr)
				if title != "" && !strings.Contains(title, "...") {
					metadata.Title = title
					break
				}
			}
		}

		// If title found, use article type as page type
		if metadata.PageType == "" && metadata.Title != "" && len(article.Types) > 0 {
			metadata.PageType = article.Types[0]
		}
	}

	// If author not found, look in persons
	if metadata.Author == "" {
		var authorNames string
		for _, person := range persons {
			for _, name := range getSchemaNames(person.Data) {
				name = validateMetadataName(name)
				authorNames = normalizeAuthors(authorNames, name)
			}
		}

		if authorNames != "" {
			metadata.Author = authorNames
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
	if metadata.PageType == "" && len(articles) > 0 && len(articles[0].Types) > 0 {
		metadata.PageType = articles[0].Types[0]
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
	var findImportantObjects func(obj map[string]any, parent *SchemaData)
	findImportantObjects = func(obj map[string]any, parent *SchemaData) {
		// Schema type could be either string or slices, so extract it properly
		schemaTypes := getSchemaTypes(obj, false)

		// Check if the schemas is usable for our purpose
		var isPerson bool
		var isWebsite, isOrganization bool
		var isArticle, isPosting, isReport, isBlog, isPage, isListing bool

		for _, st := range schemaTypes {
			st = strings.ToLower(st)
			isPerson = isPerson || st == "person"
			isWebsite = isWebsite || st == "website"
			isOrganization = isOrganization || strings.Contains(st, "organization")
			isArticle = isArticle || strings.Contains(st, "article")
			isPosting = isPosting || strings.Contains(st, "posting")
			isReport = isReport || st == "report"
			isBlog = isBlog || st == "blog"
			isPage = isPage || strings.Contains(st, "page")
			isListing = isListing || strings.Contains(st, "listing")
		}

		// Create initial schema data
		schemaData := SchemaData{
			Types:  schemaTypes,
			Data:   obj,
			Parent: parent,
		}

		// Depending on its type, save the schema to respective slice
		if isPerson {
			persons = append(persons, schemaData)
		}

		if isWebsite || isOrganization {
			// Organization is more important than website.
			switch {
			case isOrganization:
				schemaData.Importance = 2
			default:
				schemaData.Importance = 1
			}

			organizations = append(organizations, schemaData)
		}

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
		}

		// Continue to look in its sub values
		for _, value := range obj {
			switch v := value.(type) {
			case map[string]any:
				findImportantObjects(v, &schemaData)

			case []any:
				for _, item := range v {
					if subObj, isObj := item.(map[string]any); isObj {
						findImportantObjects(subObj, &schemaData)
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
			findImportantObjects(data, nil)
		}
	}

	// Sort schemas based on importance
	sort.SliceStable(organizations, func(i, j int) bool {
		return organizations[i].Importance > organizations[j].Importance
	})

	sort.SliceStable(articles, func(i, j int) bool {
		return articles[i].Importance > articles[j].Importance
	})

	// When possible, only use persons from articles
	var articlePersons []SchemaData
	for _, person := range persons {
		if schemaInArticle(person, "person") {
			articlePersons = append(articlePersons, person)
		}
	}

	if len(articlePersons) > 0 {
		persons = articlePersons
	}

	// Do the same for organizations
	var articleOrganizations []SchemaData
	for _, org := range organizations {
		if schemaInArticle(org, "organization") {
			articleOrganizations = append(articleOrganizations, org)
		}
	}

	if len(articleOrganizations) > 0 {
		organizations = articleOrganizations
	}

	return
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
		names := getStringValues(value, "name")

		// If name is empty and its type is Person, try name combination
		if len(names) == 0 && strIn("person", schemaTypes...) {
			givenName := getSingleStringValue(value, "givenName")
			additionalName := getSingleStringValue(value, "additionalName")
			familyName := getSingleStringValue(value, "familyName")
			fullName := trim(givenName + " " + additionalName + " " + familyName)
			names = []string{fullName}
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

func schemaInArticle(data SchemaData, wantedType string) bool {
	// If it doesn't have any parent, it's important
	if data.Parent == nil {
		return true
	}

	// Check if parent is person or organization
	var parentIsPerson bool
	var parentIsOrganization bool

	for _, st := range data.Parent.Types {
		st = strings.ToLower(st)
		parentIsPerson = parentIsPerson || st == "person"
		parentIsOrganization = parentIsOrganization || st == "website" || strings.Contains(st, "organization")
	}

	// If necessary, check grandparent types
	parentTypesToCheck := data.Parent.Types
	if (wantedType == "person" && parentIsPerson) || (wantedType == "organization" && parentIsOrganization) {
		if data.Parent.Parent == nil {
			return true
		} else {
			parentTypesToCheck = data.Parent.Parent.Types
		}
	}

	// Now, check if this schema inside article
	for _, st := range parentTypesToCheck {
		st = strings.ToLower(st)
		isArticle := strings.Contains(st, "article")
		isPosting := strings.Contains(st, "posting")
		isReport := st == "report"
		isBlog := st == "blog"
		isPage := strings.Contains(st, "page")
		isListing := strings.Contains(st, "listing")
		if isArticle || isPosting || isReport || isBlog || isPage || isListing {
			return true
		}
	}

	return false
}
