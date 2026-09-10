# Go-Trafilatura

Go-Trafilatura extracts main text, comments, and metadata from supplied HTML while preserving useful formatting and document structure. It is a Go port of [Trafilatura][0], the Python extractor created by [Adrien Barbaresi][1].

The port began as a close, largely line-by-line translation. We keep the extraction code recognizable so upstream fixes can be reviewed and ported systematically, while preserving Go APIs and HTML-oriented results.

The goal is faithful extraction behavior within the scope below, not identical output for every input or a Go implementation of every Python feature.

## Table of Contents

- [Status](#status)
- [Philosophy and Scope](#philosophy-and-scope)
- [Usage as Go package](#usage-as-go-package)
- [Usage as CLI Application](#usage-as-cli-application)
- [Performance](#performance)
- [Comparison with Other Go Packages](#comparison-with-other-go-packages)
- [Comparison with Original Trafilatura](#comparison-with-original-trafilatura)
- [Acknowledgements](#acknowledgements)
- [License](#license)

## Status

The supplied-HTML extraction implementation tracks the applicable changes through original Trafilatura [v2.2.0][last-version], pinned to commit [c1bc9531a2a978326112ca9987e1382745116136][last-commit].

[UPSTREAM.md](UPSTREAM.md) accounts for all 53 commits since v2.0.0, including ported behavior, existing Go equivalents, intentional exclusions, and verification results. This is an upstream compatibility target, not a new Go module version.

## Philosophy and Scope

**Bring your own HTML.** The primary use case is processing HTML already obtained by your application or scraper. `Extract` accepts an `io.Reader`; `ExtractDocument` accepts a parsed HTML DOM and leaves the caller's document unchanged. `OriginalURL` supplies context for metadata and relative URLs; it is not a request to download the page.

We follow upstream improvements to extraction, cleaning, metadata, tables, images, links, comments, and recovery. We do **not** aim for scraper, crawler, downloader, feed-discovery, or sitemap-discovery parity. The existing CLI download, batch, feed, and sitemap conveniences remain available for compatibility, but those subsystems are not expanded as part of extraction updates.

Intentional differences remain:

- JSON-LD is parsed structurally. Malformed JSON may be rejected instead of recovered through upstream's permissive decoding or regular-expression fallbacks. The Go parser also retains its article and publisher precedence rules.
- Optional fallbacks use `go-readability` and `go-domdistiller`, not Python's readability fork and jusText. Custom `FallbackCandidates` remain supported. Library fallbacks are off by default; use Python's `fast=True` for comparable extractor-only tests.
- Results contain an HTML DOM and plain text, with metadata always returned. The CLI supports HTML, text, and JSON; Python's Markdown, XML/TEI, CSV, and YAML-header serializers are not part of this port's compatibility target.
- HTML parsing, whitespace, language detection, and date extraction use Go implementations and dependencies. Their behavior can differ from the corresponding Python libraries.
- Standalone Simhash, fingerprinting, token-sampling, Python configuration files, and Python-specific APIs or tooling are outside this port's API surface. Existing paragraph deduplication is supported.

Balanced extraction can now retry in recall mode when a short result covers little of the page. Recovery also understands additional embedded JSON content, and schema-identified discussion-forum posts are treated as main content. These changes can intentionally change results on existing inputs; the [compatibility record](UPSTREAM.md) describes the tested boundaries.

## Usage as Go package

Use Go 1.24.1 or newer, as specified in [go.mod](go.mod), then run:

```
go get -u -v github.com/markusmobius/go-trafilatura
```

Next, include it in your application :

```go
import "github.com/markusmobius/go-trafilatura"
```

Now you can use Trafilatura to extract content of a web page. For basic usage you can check the [examples](examples).

## Usage as CLI Application

To install the CLI with Go 1.24.1 or newer:

```
go install github.com/markusmobius/go-trafilatura/cmd/go-trafilatura@latest
```

Once installed, you can use it from your terminal:

```
$ go-trafilatura -h
Extract readable content from a specified source which can be either a HTML file or url.
It also has supports for batch download url either from a file which contains list of url,
RSS feeds and sitemap.

Usage:
  go-trafilatura [flags] [source]
  go-trafilatura [command]

Available Commands:
  batch       Download and extract pages from list of urls that specified in the file
  feed        Download and extract pages from a feed
  help        Help about any command
  sitemap     Download and extract pages from a sitemap

Flags:
      --deduplicate         filter out duplicate segments and sections
  -f, --format string       output format for the extract result, either 'html' (default), 'txt' or 'json'
      --has-metadata        only output documents with title, URL and date
  -h, --help                help for go-trafilatura
      --images              include images in extraction result (experimental)
  -l, --language string     target language (ISO 639-1 codes)
      --links               keep links in extraction result (experimental)
      --no-comments         exclude comments  extraction result
      --no-fallback         disable fallback extraction using readability and dom-distiller
      --no-tables           include tables in extraction result
      --skip-tls            skip X.509 (TLS) certificate verification
  -t, --timeout int         timeout for downloading web page in seconds (default 30)
  -u, --user-agent string   set custom user agent (default "Mozilla/5.0 (X11; Linux x86_64; rv:88.0) Gecko/20100101 Firefox/88.0")
  -v, --verbose             enable log message

Use "go-trafilatura [command] --help" for more information about a command
```

Here are some example of common usage

- Extract from an existing HTML file without downloading a page:

  ```sh
  go-trafilatura article.html
  ```

- Fetch readable content from a specified URL

  ```
  go-trafilatura http://www.domain.com/some/path
  ```

  The output will be printed in stdout.

- Use `batch` command to fetch readable content from file which contains list of urls. So, say we have file
  named `input.txt` with following content:

  ```
  http://www.domain1.com/some/path
  http://www.domain2.com/some/path
  http://www.domain3.com/some/path
  ```

  We want to fetch them and save the result in directory `extract`. To do so, we can run:

  ```
  go-trafilatura batch -o extract input.txt
  ```

- Use `sitemap` to crawl sitemap then fetch all web pages that listed under the sitemap. We can explicitly
  specify the sitemap:

  ```
  go-trafilatura sitemap -o extract http://www.domain.com/sitemap.xml
  ```

  Or you can just put the domain and let Trafitula to look for the sitemap:

  ```
  go-trafilatura sitemap -o extract http://www.domain.com
  ```

- Use `feed` to crawl RSS or Atom feed, then fetch all web pages that listed under it. We can explicitly
  specify the feed url:

  ```
  go-trafilatura feed -o extract http://www.domain.com/feed-rss.php
  ```

  Or you can just put the domain and let Trafitula to look for the feed url:

  ```
  go-trafilatura feed -o extract http://www.domain.com
  ```

## Performance

This package and its dependencies heavily use regular expression for various purposes. Unfortunately, as commonly known, Go's regular expression is pretty [slow][go-regex-slow]. This is because:

- The regex engine in other language usually implemented in C, while in Go it's implemented from scratch in Go language. As expected, C implementation is still faster than Go's.
- Since Go is usually used for web service, its regex is designed to finish in time linear to the length of the input, which useful for protecting server from ReDoS attack. However, this comes with performance cost.

To solve this issue, we compile several important regexes into Go code using [re2go]. Thanks to this we are able to achieve greater speed without using cgo or external regex packages.

## Comparison with Other Go Packages

The comparisons in this section are historical, from May 2025. They are not measurements of the v2.2.0 update. Current verification on the same 960-document corpus is recorded in [UPSTREAM.md](UPSTREAM.md).

As far as we know, currently there are three content extractors built for Go:

- [Go-DomDistiller][dom-distiller]
- [Go-Readability][readability]
- Go-Trafilatura

Since every extractors use its own algorithms, their results are a bit different. In general they give satisfactory results, however we found out that there are some cases where DOM Distiller is better and vice versa. Here is the short summary of pros and cons for each extractor:

Dom Distiller:

- Very fast.
- Good at extracting images from article.
- Able to find next page in sites that separated its article to several partial pages.
- Since the original library was embedded in Chromium browser, its tests are pretty thorough.
- CON: has a huge codebase, mostly because it mimics the original Java code.
- CON: the original library is not maintained anymore and has been archived.

Readability:

- Fast, although not as fast as Dom Distiller.
- Better than DOM Distiller at extracting wiki and documentation pages.
- The original library in Readability.js is still actively used and maintained by Firefox.
- The codebase is pretty small.
- CON: the unit tests are not as thorough as the other extractors.

Trafilatura:

- Has the best accuracy compared to other extractors.
- Better at extracting web page's metadata, including its language and publish date.
- Its unit tests are thorough and focused on removing noise while making sure the real contents are still captured.
- Designed to be used in academic domain e.g. natural language processing.
- Actively maintained with new release almost every month.
- CON: slower than the other extractors, mostly because it also looks for language and publish date.
- CON: not very good at extracting images.

The benchmark that compares these extractors is available in [this repository][benchmark]. It uses each extractor to process 983 web pages in single thread. Here is its benchmark result when tested on my PC (Intel i7-8550U @ 4.000GHz, RAM 16 GB):

Here we compare the extraction result between `go-trafilatura`, `go-readability` and `go-domdistiller`. To reproduce this test, clone this repository then run:

```
go run scripts/comparison/*.go content
```

For the test, we use 960 documents taken from various sources (2025-05-01). Here is the result when tested in my PC (AMD Ryzen 5 7535HS @ 4.6GHz, RAM 16 GB):

|            Package             | Precision | Recall | Accuracy | F-Score | Time (s) |
| :----------------------------: | :-------: | :----: | :------: | :-----: | :------: |
|        `go-readability`        |   0.871   | 0.891  |  0.880   |  0.881  |   2.87   |
|       `go-domdistiller`        |   0.873   | 0.872  |  0.873   |  0.872  |   2.66   |
|        `go-trafilatura`        |   0.912   | 0.897  |  0.906   |  0.904  |   4.25   |
| `go-trafilatura` with fallback |   0.909   | 0.921  |  0.914   |  0.915  |   8.39   |

## Comparison with Original Trafilatura

The following historical results compare the earlier Go port with original Trafilatura v1.12.2, not v2.2.0:

|                 Package                 | Precision | Recall | Accuracy | F-Score | Time (s) |
| :-------------------------------------: | :-------: | :----: | :------: | :-----: | :------: |
|              `trafilatura`              |   0.918   | 0.898  |  0.909   |  0.908  |  10.38   |
|        `trafilatura` + fallback         |   0.919   | 0.915  |  0.917   |  0.917  |  14.53   |
|  `trafilatura` + fallback + precision   |   0.932   | 0.889  |  0.912   |  0.910  |  19.34   |
|    `trafilatura` + fallback + recall    |   0.907   | 0.919  |  0.913   |  0.913  |  11.63   |
|            `go-trafilatura`             |   0.912   | 0.897  |  0.906   |  0.904  |   4.25   |
|       `go-trafilatura` + fallback       |   0.909   | 0.921  |  0.914   |  0.915  |   8.39   |
| `go-trafilatura` + fallback + precision |   0.921   | 0.900  |  0.912   |  0.910  |   7.68   |
|  `go-trafilatura` + fallback + recall   |   0.893   | 0.927  |  0.908   |  0.910  |   6.43   |

These scores measure annotated content retention and boilerplate removal on that corpus. Similar aggregate scores do not imply identical extracted text. Parser, formatting, metadata, and fallback differences can all affect individual documents.

The historical timings are specific to those versions, options, and hardware. The port uses re2go-generated code for several critical regular expressions, but no general speedup is claimed for the v2.2.0 update.

The comparison tool can also run concurrently. These historical timings used all available threads on the same PC:

```
go run scripts/comparison/*.go content -j -1
```

|                 Package                 | Time (s) |
| :-------------------------------------: | :------: |
|            `go-trafilatura`             |  0.931   |
|       `go-trafilatura` + fallback       |  1.976   |
| `go-trafilatura` + fallback + precision |  1.856   |
|  `go-trafilatura` + fallback + recall   |  1.599   |

## Acknowledgements

This package won't be exist without effort by Adrien Barbaresi, the author of the original Python package. He created `trafilatura` as part of effort to [build text databases for research][k-web], to facilitate a better text data collection which lead to a better corpus quality. For more information:

```
@inproceedings{barbaresi-2021-trafilatura,
  title = {{Trafilatura: A Web Scraping Library and Command-Line Tool for Text Discovery and Extraction}},
  author = "Barbaresi, Adrien",
  booktitle = "Proceedings of the Joint Conference of the 59th Annual Meeting of the Association for Computational Linguistics and the 11th International Joint Conference on Natural Language Processing: System Demonstrations",
  pages = "122--131",
  publisher = "Association for Computational Linguistics",
  url = "https://aclanthology.org/2021.acl-demo.15",
  year = 2021,
}
```

- Barbaresi, A. [Trafilatura: A Web Scraping Library and Command-Line Tool for Text Discovery and Extraction][paper-1], Proceedings of ACL/IJCNLP 2021: System Demonstrations, 2021, p. 122-131.
- Barbaresi, A. ["Generic Web Content Extraction with Open-Source Software"][paper-2], Proceedings of KONVENS 2019, Kaleidoscope Abstracts, 2019.
- Barbaresi, A. ["Efficient construction of metadata-enhanced web corpora"][paper-3], Proceedings of the [10th Web as Corpus Workshop (WAC-X)][wac-x], 2016.

## License

Like the original, `go-trafilatura` is distributed under the [Apache v2.0](LICENSE) license.

[0]: https://github.com/adbar/trafilatura
[1]: https://github.com/adbar
[2]: https://github.com/scrapinghub/article-extraction-benchmark
[3]: https://chromium.googlesource.com/chromium/dom-distiller
[last-version]: https://github.com/adbar/trafilatura/releases/tag/v2.2.0
[last-commit]: https://github.com/adbar/trafilatura/commit/c1bc9531a2a978326112ca9987e1382745116136
[paper-1]: https://aclanthology.org/2021.acl-demo.15/
[paper-2]: https://hal.archives-ouvertes.fr/hal-02447264/document
[paper-3]: https://hal.archives-ouvertes.fr/hal-01371704v2/document
[wac-x]: https://www.sigwac.org.uk/wiki/WAC-X
[k-web]: https://www.dwds.de/d/k-web
[go-regex-slow]: https://github.com/golang/go/issues/26623
[re2go]: https://re2c.org/manual/manual_go.html
[dom-distiller]: https://github.com/markusmobius/go-domdistiller/
[readability]: https://github.com/go-shiori/go-readability
[benchmark]: https://github.com/markusmobius/content-extractor-benchmark
