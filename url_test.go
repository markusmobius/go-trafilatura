package trafilatura

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_isAbsoluteURL(t *testing.T) {
	assertURL := func(url string, expected bool) {
		isAbs, _ := isAbsoluteURL(url)
		assert.Equal(t, expected, isAbs)
	}

	assertURL("http://www.test.org:7ERT/test", false)
	assertURL("ntp://www.test.org/test", false)
	assertURL("ftps://www.test.org/test", false)
	assertURL("http://t.g/test", true)
	assertURL("http://test.org/test", true)
}
