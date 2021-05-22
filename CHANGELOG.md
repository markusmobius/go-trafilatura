# Changelog

### 22 May 2021

- Fix `sanitizeTree` and real world test.
- Add additional selector rules.
- Restructure cmd.
- Update README.

### 21 May 2021

- Port the `comparison.py`. At this point all code have been ported.
- Strip text elements containing only spaces.
- Fix HTML language element filter.
- Fix `postCleaning`.
- Improve test coverage.
- Add support for details/summary tags.
- Refined metadata title selector.
- Include page license in metadata extraction.
- Fix: don't remove tail of discarded elements.
- Define generic function to remove nodes.
- Fix wrong constant in `collectLinkInfo`.

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
- Good news: we might not need to port Python's [`courlan`][1] package since Go's `net/url` is good enough.
- Bad news: we might need to port Python's [`htmldate`][2] which used to find publish date of a web page, which used in metadata extraction.

### 25 April 2021

- Porting process started

[1]: https://github.com/adbar/courlan
[2]: https://github.com/adbar/htmldate