package trafilatura

import (
	"strings"
	"testing"

	"github.com/go-shiori/dom"
	"github.com/stretchr/testify/assert"
)

func Test_Core_images(t *testing.T) {
	// Basic test
	assert.NotNil(t, handleImage(nodeFromString(`<img src="test.jpg"/>`)))
	assert.NotNil(t, handleImage(nodeFromString(`<img data-src="test.jpg" alt="text" title="a title"/>`)))
	assert.Nil(t, handleImage(nodeFromString(`<img other="test.jpg"/>`)))

	// Check utility
	assert.True(t, isImageFile("test.jpg"))
	assert.False(t, isImageFile("test.txt"))

	// Check handle text element
	assert.Nil(t, handleTextElem(dom.CreateElement("img"), nil, nil, false))

	// CNN example
	img := nodeFromString(`<img class="media__image media__image--responsive" alt="Harry and Meghan last March, in their final royal engagement."
		data-src-mini="//cdn.cnn.com/cnnnext/dam/assets/210307091919-harry-meghan-commonwealth-day-small-169.jpg"
		data-src-xsmall="//cdn.cnn.com/cnnnext/dam/assets/210307091919-harry-meghan-commonwealth-day-medium-plus-169.jpg"
		data-src-small="//cdn.cnn.com/cnnnext/dam/assets/210307091919-harry-meghan-commonwealth-day-large-169.jpg"
		data-src-medium="//cdn.cnn.com/cnnnext/dam/assets/210307091919-harry-meghan-commonwealth-day-exlarge-169.jpg"
		data-src-large="//cdn.cnn.com/cnnnext/dam/assets/210307091919-harry-meghan-commonwealth-day-super-169.jpg"
		data-src-full16x9="//cdn.cnn.com/cnnnext/dam/assets/210307091919-harry-meghan-commonwealth-day-full-169.jpg"
		data-src-mini1x1="//cdn.cnn.com/cnnnext/dam/assets/210307091919-harry-meghan-commonwealth-day-small-11.jpg"
		data-demand-load="loaded" data-eq-pts="mini: 0, xsmall: 221, small: 308, medium: 461, large: 781"
		src="//cdn.cnn.com/cnnnext/dam/assets/210307091919-harry-meghan-commonwealth-day-exlarge-169.jpg"
		data-eq-state="mini xsmall small medium"
		data-src="//cdn.cnn.com/cnnnext/dam/assets/210307091919-harry-meghan-commonwealth-day-exlarge-169.jpg">`)
	processedImg := handleImage(img)
	assert.NotNil(t, processedImg)
	assert.True(t, dom.HasAttribute(processedImg, "alt"))
	assert.True(t, dom.HasAttribute(processedImg, "src"))

	// Modified CNN example
	img = nodeFromString(`<img class="media__image media__image--responsive" alt="Harry and Meghan last March, in their final royal engagement."
		data-src-mini="//cdn.cnn.com/cnnnext/dam/assets/210307091919-harry-meghan-commonwealth-day-small-169.jpg"
		data-src-xsmall="//cdn.cnn.com/cnnnext/dam/assets/210307091919-harry-meghan-commonwealth-day-medium-plus-169.jpg"
		data-src-small="//cdn.cnn.com/cnnnext/dam/assets/210307091919-harry-meghan-commonwealth-day-large-169.jpg"
		data-src-medium="//cdn.cnn.com/cnnnext/dam/assets/210307091919-harry-meghan-commonwealth-day-exlarge-169.jpg"
		data-src-large="//cdn.cnn.com/cnnnext/dam/assets/210307091919-harry-meghan-commonwealth-day-super-169.jpg"
		data-src-full16x9="//cdn.cnn.com/cnnnext/dam/assets/210307091919-harry-meghan-commonwealth-day-full-169.jpg"
		data-src-mini1x1="//cdn.cnn.com/cnnnext/dam/assets/210307091919-harry-meghan-commonwealth-day-small-11.jpg"
		data-demand-load="loaded" data-eq-pts="mini: 0, xsmall: 221, small: 308, medium: 461, large: 781">`)
	processedImg = handleImage(img)
	assert.NotNil(t, processedImg)
	assert.True(t, dom.HasAttribute(processedImg, "alt"))
	assert.True(t, dom.HasAttribute(processedImg, "src"))
	assert.True(t, strings.HasPrefix(dom.GetAttribute(processedImg, "src"), "http"))
}

func Test_Core_links(t *testing.T) {
	assert.Nil(t, handleTextElem(dom.CreateElement("a"), nil, nil, false))

	a := nodeFromString(`<a href="testlink.html">Test link text.</a>`)
	assert.NotNil(t, handleFormatting(a))
}

func Test_Core_exoticTags(t *testing.T) {
	// <p> within <p>
	first := dom.CreateElement("p")
	dom.SetTextContent(first, "1st part.")

	second := dom.CreateElement("p")
	dom.SetTextContent(second, "2nd part.")
	dom.AppendChild(first, second)

	element := dom.Clone(first, true)
	converted := handleParagraphs(element, map[string]struct{}{"p": {}}, nil, false)
	assert.Equal(t, "<p>1st part. 2nd part.</p>", dom.OuterHTML(converted))

	// Delete last line break
	third := dom.CreateElement("br")
	dom.AppendChild(first, third)

	element = dom.Clone(first, true)
	converted = handleParagraphs(element, map[string]struct{}{"p": {}}, nil, false)
	assert.Equal(t, "<p>1st part. 2nd part.</p>", dom.OuterHTML(converted))

	// Malformed lists
	// element = nodeFromString(`
	// 	<ul>Description of the list:
	// 		<li>List item 1</li>
	// 		<li>List item 2</li>
	// 		<li>List item 3</li>
	// 	</ul>`)

	// result := handleLists(element, nil, false)
	// assert.NotNil(t, result)
}
