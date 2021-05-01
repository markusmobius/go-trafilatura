# Go-Trafilatura

Go-Trafilatura is a Go package and command-line tool which seamlessly downloads, parses, and scrapes web page data: it can extract metadata, main body text and comments while preserving parts of the text formatting and page structure.

As implied by its name, this package is based on [Trafilatura][0] which is a Python package that created by [Adrien Barbaresi][1]. We decided to port this package because according to ScrapingHub [benchmark][2], at the time this port is created Trafilatura is the most efficient open-source article extractor. This is especially impressive considering how robust its code, only around 4,000 lines of Python code that separated in 26 files. As comparison, [Dom Distiller][3] has 148 files with around 17,000 lines of code.

The structure of this package is arranged following the structure of original Python code. This way, any improvements from the original can be implemented easily here. Another advantage, hopefully all web page that can be parsed by the original Trafilatura can be parsed by this package as well with identical result.

## Status

This package is still in development and the port process is still not finished. There are 20 files with 3,542 lines of code that haven’t been ported, so there is still long way to go.

## Changelog

### 1 May 2021

- Port `LRUCache` in `lru.py`
- Port `textfilter` in `filters.py`
- Port `duplicate_test` in `filters.py`
- Port `extract_comments` in `core.py`. It's still not tested though since there are no specific unit test for this.

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