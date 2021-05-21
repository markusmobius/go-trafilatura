# Go-Trafilatura

Go-Trafilatura is a Go package and command-line tool which seamlessly downloads, parses, and scrapes web page data: it can extract metadata, main body text and comments while preserving parts of the text formatting and page structure.

As implied by its name, this package is based on [Trafilatura][0] which is a Python package that created by [Adrien Barbaresi][1]. We decided to port this package because according to ScrapingHub [benchmark][2], at the time this port is created Trafilatura is the most efficient open-source article extractor. This is especially impressive considering how robust its code, only around 4,000 lines of Python code that separated in 26 files. As comparison, [Dom Distiller][3] has 148 files with around 17,000 lines of code.

The structure of this package is arranged following the structure of original Python code. This way, any improvements from the original can be implemented easily here. Another advantage, hopefully all web page that can be parsed by the original Trafilatura can be parsed by this package as well with identical result.

## Status

This package is still in development and the port process is finished. We only need to catch up to the latest commit in the original Trafilatura.

As it is right now, the content extractor already functional and can be used within Go code, as can be seen in this [example](examples/from-url.go) :

```go
package main

import (
	"fmt"
	"net/http"
	nurl "net/url"
	"time"

	"github.com/go-shiori/dom"
	"github.com/markusmobius/go-trafilatura"
	"github.com/sirupsen/logrus"
)

var (
	httpClient = &http.Client{Timeout: 30 * time.Second}
)

func main() {
	// Prepare URL
	url := "https://arstechnica.com/science/2021/05/rare-flesh-eating-black-fungus-rides-covids-coattails-in-india/"
	parsedURL, err := nurl.ParseRequestURI(url)
	if err != nil {
		logrus.Fatalf("failed to parse url: %v", err)
	}

	// Fetch article
	resp, err := httpClient.Get(url)
	if err != nil {
		logrus.Fatalf("failed to fetch the page: %v", err)
	}
	defer resp.Body.Close()

	// Extract content
	opts := trafilatura.Options{
		IncludeImages: true,
		OriginalURL:   parsedURL,
	}

	result, err := trafilatura.Extract(resp.Body, opts)
	if err != nil {
		logrus.Fatalf("failed to extract: %v", err)
	}

	// Print result
	fmt.Println(dom.OuterHTML(result.ContentNode))
}
```

There are some difference between this port and the original Trafilatura:

- In the original, metadata from JSON+LD is extracted using regular expressions while in this port it's done using a JSON parser. This way, our metadata extraction should be more correct, however the code is a bit more complex.
- In the original, they use `python-readability` and `justext` as fallback extractors. In this port we use `go-readability` and `go-domdistiller` instead. Thanks to this, while rare, there will be some difference in extraction result between our port and the original. This is why there are some differences in the real world tests between the original and our port.
- The main output of the original Trafilatura is XML, while in our port the main output is HTML. Thanks to this, there are some difference in handling formatting tags (e.g. `<b>`, `<i>`) and paragraphs.

What's not done yet:

- Port `htmldate`. It's a medium library which consists of around 900 lines of Python code (excluding unit tests), whose sole purpose is to extract publishing date of a web page. 

	I do wonder whether it's worth it to port the entire library, since it's only used for metadata extraction, only called twice in the whole code, and doesn't affect the extracted content in any way.

- The original Trafilatura uses `cld3` to detect language of the web page. In this port, we use `whatlanggo` as alternative, and from the test so far it works nicely and quite accurate. The only issue that have been occured so far is it's struggle a bit when the sample text is too short (less than 50 characters), yet works properly after the sample text is increased a bit. Need more investigation.

	Personally, I believe it's good enough especially since language detection is only used to discard web page that doesn't match with our target language. And, just like `htmldate`, this language detection also doesn't affect the extracted content.

- Since Trafilatura is still actively developed, we might need to catch up to the latest Trafilatura version.

## Changelog

### 21 May 2021

- Port the `comparison.py`. At this point all code have been ported.

### 20 May 2021

- Add license header in each file
- Improve charset encoding to make sure parsing HTML document always done in UTF-8.

### 19 May 2021

- In CLI, add flags to fetch only the urls from sitemap.
- In CLI, implement feed finder and downloader.
- In CLI, add flags for custom user agent.
- Move `etree` and `selector` package to internal dir so it can't be reached by user.
- Remove finished python codes.

### 18 May 2021

- In CLI, implement sitemap finder and downloader.

### 17 May 2021

- In CLI, add support for several type of output.
- In CLI, add subcommand for batch download from file that contains list of url.

### 16 May 2021

- Make the log less verbose.
- Implement initial CLI.

### 12 May 2021

- Modify paragraphs handling since our output is in HTML, not XML like the original Trafilatura.
- Put whitespace in place of void element when writing text using `etree.IterText`.
- Dont strip image elements when sanitizing extraction result.
- Add initial example.

### 11 May 2021

- Implement real world test from `tests/realworld_test.py`. In the original Trafilatura, in this test the extraction is done while enabling fallback extractors. However, since the fallback extractors in the original Trafilatura is different with the one that used in this port, obviously the result is different as well which make the test can't be ported as is.

	To solve this, I've changed the test in the original Trafilatura to disable the fallback extractors. This way the test is more focused on the capability of Trafilatura alone, which make the test is compatible and can be ported.

### 10 May 2021

- Since our port use `go-readability` as one of its fallback, here we updated it to more recent version of Readability.js.
- Fix external `dom` package to not appending child to void elements (elements that can't have any children, eg `<br/>`).

### 9 May 2021

- Now `Extract` also returns metadata along the extracted content.
- Add advanced config in extraction `Options`.
- Minor change  in `etree.ToString` to make it more readable.
- Implement unit tests.

### 8 May 2021

- Finished implementing `Extract` function. At this point the port is kind of finished, but it's still not tested, so there is still a long way to go.
- Restructure test files.

### 7 May 2021

- Fix implementation of `IterText` in `etree` package.
- Implement fallback extraction using `go-readability` and `go-domdistiller`.

### 6 May 2021

- Restructure selector files.
- Implement comments extraction.
- Implement content extraction.

### 5 May 2021

- Port some of LXML functionality to `etree` package.
- Fix major issue when appending or replacing node in external `dom` package. Apparently this issue goes unnoticed in both `go-readability` and `go-domdistiller`.
- Restart porting process from zero 😢.
- Reimplement `cache`.
- Reimplement metadata extractor.

### 4 May 2021

- No code today. Looks like I've made a wrong assumptions about LXML library that used by the original Trafilatura. In functionality it's really similar with `dom` package, however there are several difference in how it works. Might need to port some codes.

### 3 May 2021

- Port `link_density_test` and `link_density_test_tables` from `htmlprocessing.py`

### 2 May 2021

- Port `DISCARD_XPATH` in `xpaths.py`

### 1 May 2021

- Port `LRUCache` in `lru.py`
- Port `textfilter` in `filters.py`
- Port `duplicate_test` in `filters.py`
- Port `extract_comments` in `core.py`. It's still not tested though since there are no specific unit test for this.
- Port `CONTENT_XPATH` in `xpaths.py`

### 29 April 2021

- Port `check_html_lang` function in `filters.py`
- Port metadata extraction in `metadata.py`. There is a minor modification in metadata extraction from JSON+LD data. In the original Trafilatura, this step is done using regular expressions which is not exactly ideal for handling JSON data. Instead, here we use a proper JSON parser with fallback to the original regular expressions. This way, the extraction should be more accurate yet still give the same result when tested.
- Port `tree_cleaning` and `prune_html` in `htmlprocessing.py`
- Good news: we might not need to port Python's [`courlan`][4] package since Go's `net/url` is good enough.
- Bad news: we might need to port Python's [`htmldate`][5] which used to find publish date of a web page, which used in metadata extraction.

### 25 April 2021

- Porting process started

[0]: https://github.com/adbar/trafilatura
[1]: https://github.com/adbar
[2]: https://github.com/scrapinghub/article-extraction-benchmark
[3]: https://chromium.googlesource.com/chromium/dom-distiller
[4]: https://github.com/adbar/courlan
[5]: https://github.com/adbar/htmldate