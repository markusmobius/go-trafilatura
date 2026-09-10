# Changelog

### Unreleased

- Raise the minimum Go version to 1.26.0 and the preferred development toolchain to Go 1.27.1, without changing dependency versions.
- Add opt-in `Options.InputEncoding` to bypass charset detection for supplied HTML with a known encoding, retaining normalization and leaving automatic detection unchanged by default.
- Track supplied-HTML extraction changes through upstream Trafilatura v2.2.0. See [UPSTREAM.md](UPSTREAM.md) for the complete 53-commit audit from v2.0.0.
- Preserve pruning tails, nested inline formatting, relative and linked images, table captions, empty cells, and bounded row/column spans.
- Improve metadata image, title, author, license, and JSON-LD publisher handling.
- Update recovery deduplication, embedded-content baseline recovery, targeted boilerplate filtering, comment exclusion, forum-post routing, and bounded recall escalation.
- Keep caller-owned DOMs and supplied fallback candidates unchanged during extraction.
- Document the supplied-HTML philosophy and intentional differences. Existing CLI download, feed, and sitemap features are unchanged; crawler and Python-only API/output parity remain outside scope.

### 22 May 2021

- Fix `sanitizeTree` and the real-world test.
- Add additional selector rules.
- Restructure the CLI.
- Update the README.

### 21 May 2021

- Port the Python comparison script, completing the initial port.
- Strip text elements containing only spaces.
- Fix the HTML language element filter.
- Fix `postCleaning`.
- Improve test coverage.
- Add support for details/summary tags.
- Refine the metadata title selector.
- Include page licenses in metadata extraction.
- Preserve the tail text of discarded elements.
- Add a generic node-removal function.
- Fix an incorrect constant in `collectLinkInfo`.

### 20 May 2021

- Add a license header to each file.
- Decode HTML as UTF-8 before parsing.

### 19 May 2021

- Add CLI flags to fetch only the URLs from a sitemap.
- Implement feed discovery and downloading in the CLI.
- Add CLI flags for custom user agents.
- Move the `etree` and `selector` packages into `internal` to keep them private.
- Remove Python code whose port is complete.

### 18 May 2021

- Implement sitemap discovery and downloading in the CLI.

### 17 May 2021

- Add support for multiple output formats in the CLI.
- Add a CLI subcommand for batch downloads from a file containing URLs.

### 16 May 2021

- Make logging less verbose.
- Implement initial CLI.

### 12 May 2021

- Modify paragraph handling for HTML output rather than the original Trafilatura's XML output.
- Insert whitespace for void elements when writing text with `etree.IterText`.
- Preserve image elements when sanitizing extraction results.
- Add the initial example.

### 11 May 2021

- Port the real-world tests from `tests/realworld_test.py`. The original tests enable fallback extractors, but the Go port uses different fallback engines and can produce different results. Disable fallbacks in the reference tests to compare Trafilatura's extraction alone.

### 10 May 2021

- Update the `go-readability` fallback to reflect a more recent version of Readability.js.
- Fix the external `dom` package to avoid appending children to void elements, such as `<br/>`.

### 9 May 2021

- Return metadata alongside the extracted content from `Extract`.
- Add advanced configuration to extraction `Options`.
- Improve readability in `etree.ToString`.
- Implement unit tests.

### 8 May 2021

- Finish implementing `Extract`. The initial implementation is complete but has not yet been tested.
- Restructure test files.

### 7 May 2021

- Fix `IterText` in the `etree` package.
- Implement fallback extraction using `go-readability` and `go-domdistiller`.

### 6 May 2021

- Restructure selector files.
- Implement comment extraction.
- Implement content extraction.

### 5 May 2021

- Port some lxml functionality to the `etree` package.
- Fix a major issue when appending or replacing nodes in the external `dom` package. The issue appears to affect both `go-readability` and `go-domdistiller`.
- Restart the porting process from scratch.
- Reimplement `cache`.
- Reimplement the metadata extractor.

### 4 May 2021

- Reassess assumptions about lxml. It resembles the `dom` package, but behavioral differences may require additional porting.

### 3 May 2021

- Port `link_density_test` and `link_density_test_tables` from `htmlprocessing.py`.

### 2 May 2021

- Port `DISCARD_XPATH` from `xpaths.py`.

### 1 May 2021

- Port `LRUCache` from `lru.py`.
- Port `textfilter` from `filters.py`.
- Port `duplicate_test` from `filters.py`.
- Port `extract_comments` from `core.py`; dedicated unit tests are not yet available.
- Port `CONTENT_XPATH` from `xpaths.py`.

### 29 April 2021

- Port `check_html_lang` from `filters.py`.
- Port metadata extraction from `metadata.py`. Use a JSON parser for JSON-LD data, with a fallback to the original regular expressions, to preserve the reference test results.
- Port `tree_cleaning` and `prune_html` from `htmlprocessing.py`.
- Consider using Go's `net/url` instead of porting Python's [`courlan`][1] package.
- Consider porting Python's [`htmldate`][2] package to extract publication dates for metadata.

### 25 April 2021

- Start the porting process.

[1]: https://github.com/adbar/courlan
[2]: https://github.com/adbar/htmldate