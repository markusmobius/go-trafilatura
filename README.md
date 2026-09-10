# Go-Trafilatura

Go-Trafilatura extracts main text, comments, and metadata from supplied HTML while preserving useful formatting and document structure. It is a Go port of [Trafilatura][0], the Python extractor created by [Adrien Barbaresi][1].

The port began as a close, largely line-by-line translation. We keep the extraction code recognizable so upstream fixes can be reviewed and ported systematically, while preserving Go APIs and HTML-oriented results.

The goal is faithful extraction behavior within the scope below, not identical output for every input or a Go implementation of every Python feature.

## Table of Contents

- [Status](#status)
- [Philosophy and Scope](#philosophy-and-scope)
- [Usage as a Go Package](#usage-as-a-go-package)
- [Usage as a CLI Application](#usage-as-a-cli-application)
- [Performance](#performance)
- [Comparison with Other Go Packages](#comparison-with-other-go-packages)
- [Comparison with Python Trafilatura](#comparison-with-python-trafilatura)
- [Acknowledgements](#acknowledgements)
- [License](#license)

## Status

The supplied-HTML extraction implementation tracks the applicable changes through upstream Trafilatura [v2.2.0][last-version], pinned to commit [c1bc9531a2a978326112ca9987e1382745116136][last-commit].

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

## Usage as a Go Package

Use Go 1.26.0 or newer. [go.mod](go.mod) selects Go 1.27.1 as the preferred development toolchain. To add the package, run:

```sh
go get github.com/markusmobius/go-trafilatura
```

Import the package in your application:

```go
import "github.com/markusmobius/go-trafilatura"
```

See the [examples](examples) for basic usage.

### Known Input Encoding

If your scraper already knows the encoding of the supplied HTML, set `InputEncoding` to skip statistical charset detection:

```go
result, err := trafilatura.Extract(reader, trafilatura.Options{
    InputEncoding: "utf-8",
})
```

Use the encoding of the bytes passed to `Extract`, not the page's original encoding if your scraper has already decoded it. For example, HTML converted to UTF-8 should use `"utf-8"`, even if its original charset declaration says otherwise. Other supported HTML charset labels, such as `"windows-1252"` and `"shift_jis"`, decode the input without detection. Unsupported labels return an error.

This option is strictly opt-in: omitting it or using `""` keeps the existing automatic-detection path unchanged. Both paths retain NFC Unicode normalization and soft-hyphen removal. An explicit label takes precedence over declarations in the input; use it only when the encoding is known. `ExtractDocument` ignores this option because its input is already parsed.

## Usage as a CLI Application

To install the CLI with Go 1.26.0 or newer:

```sh
go install github.com/markusmobius/go-trafilatura/cmd/go-trafilatura@latest
```

Use `--help` to see the available commands and options, including options for individual subcommands:

```sh
go-trafilatura --help
go-trafilatura batch --help
```

Common examples:

- Extract from an existing HTML file without downloading a page:

  ```sh
  go-trafilatura article.html
  ```

- Extract readable content from a URL:

  ```sh
  go-trafilatura https://example.org/some/path
  ```

  The result is written to standard output.

- Use `batch` to process URLs listed in a file. For example, `input.txt` might contain:

  ```text
  https://example.org/first-article
  https://example.org/second-article
  https://example.org/third-article
  ```

  Download these pages and save the results in the `extract` directory:

  ```sh
  go-trafilatura batch -o extract input.txt
  ```

- Use `sitemap` to find and process pages listed in a sitemap:

  ```sh
  go-trafilatura sitemap -o extract https://example.org/sitemap.xml
  ```

  Or supply the site URL and let Go-Trafilatura look for its sitemap:

  ```sh
  go-trafilatura sitemap -o extract https://example.org
  ```

- Use `feed` to find and process pages listed in an RSS or Atom feed:

  ```sh
  go-trafilatura feed -o extract https://example.org/feed.xml
  ```

  Or supply the site URL and let Go-Trafilatura look for its feed:

  ```sh
  go-trafilatura feed -o extract https://example.org
  ```

## Performance

Extraction time depends on document size and structure, character-encoding detection, metadata processing, and optional fallback extractors. Use `ExtractDocument` when you already have a parsed DOM, or supply a [known input encoding](#known-input-encoding) to avoid statistical charset detection.

The boilerplate text filter, `IsTextFilter`, uses Go code generated by [re2go]. It is one matcher containing multiple pattern alternatives, not a translation of every regex in the package. Other patterns retain their existing regex implementations. The generated source is checked in, so normal builds require neither re2go nor CGO; the generator is needed only when regenerating that source.

## Comparison with Other Go Packages

The comparisons in this section are historical, from May 2025. They are not measurements of the v2.2.0 update. Current verification on the same 960-document corpus is recorded in [UPSTREAM.md](UPSTREAM.md).

The local comparison includes [Go-Readability][readability], [Go-DomDistiller][dom-distiller], and Go-Trafilatura. Their algorithms and fallback choices can produce different results on individual pages. See the [comparison documentation](scripts/comparison/README.md) for commands and dataset sources, or the [separate benchmark project][benchmark] for additional comparisons.

The historical results below used 960 documents, one worker, and an AMD Ryzen 5 7535HS with 16 GB of RAM:

|            Package             | Precision | Recall | Accuracy | F-Score | Time (s) |
| :----------------------------: | :-------: | :----: | :------: | :-----: | :------: |
|        `go-readability`        |   0.871   | 0.891  |  0.880   |  0.881  |   2.87   |
|       `go-domdistiller`        |   0.873   | 0.872  |  0.873   |  0.872  |   2.66   |
|        `go-trafilatura`        |   0.912   | 0.897  |  0.906   |  0.904  |   4.25   |
| `go-trafilatura` with fallback |   0.909   | 0.921  |  0.914   |  0.915  |   8.39   |

## Comparison with Python Trafilatura

The table below compares this port with **Python Trafilatura [v2.2.0][last-version]** on the same 960 saved HTML documents. Both use balanced mode, with fallbacks disabled, comments excluded, and tables included. Failed extractions count as empty text rather than being omitted.

| Extractor | Precision | Recall | F1 |
| --- | ---: | ---: | ---: |
| Go-Trafilatura, updated through v2.2.0 | 0.9171 | 0.9113 | 0.9142 |
| Python Trafilatura v2.2.0 | 0.9155 | 0.9082 | 0.9118 |

These phrase-level annotation scores measure content retention and boilerplate removal, not metadata quality or exact output identity. After removing whitespace differences, 838 of 960 extracted bodies matched. All 22 representative structural comparisons matched after normalizing the two DOM vocabularies and whitespace.

The measurements were collected during the v2.2.0 update using Go 1.24.2 and Python 3.12. The newer supported Go toolchains have since passed the regression suite; the scores above are not a new benchmark on those toolchains. See [UPSTREAM.md](UPSTREAM.md#verification) for the pinned versions, methodology, and compatibility boundaries.

Comparable v2.2.0 timings are not reported here: interleaved timing runs varied even for unchanged control extractors. The older Python timings and fallback-mode rows have therefore not been carried into this table. Go and Python also use different fallback engines, so fallback results must be evaluated separately.

## Acknowledgements

This port builds on the work of Adrien Barbaresi, who created the original Python package as part of an effort to [build text databases for research][k-web] and improve corpus quality. For background and citation details:

```bibtex
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

Like the original, `go-trafilatura` is distributed under the [Apache License 2.0](LICENSE).

[0]: https://github.com/adbar/trafilatura
[1]: https://github.com/adbar
[last-version]: https://github.com/adbar/trafilatura/releases/tag/v2.2.0
[last-commit]: https://github.com/adbar/trafilatura/commit/c1bc9531a2a978326112ca9987e1382745116136
[paper-1]: https://aclanthology.org/2021.acl-demo.15/
[paper-2]: https://hal.archives-ouvertes.fr/hal-02447264/document
[paper-3]: https://hal.archives-ouvertes.fr/hal-01371704v2/document
[wac-x]: https://www.sigwac.org.uk/wiki/WAC-X
[k-web]: https://www.dwds.de/d/k-web
[re2go]: https://re2c.org/manual/manual_go.html
[dom-distiller]: https://github.com/markusmobius/go-domdistiller/
[readability]: https://github.com/go-shiori/go-readability
[benchmark]: https://github.com/markusmobius/content-extractor-benchmark
