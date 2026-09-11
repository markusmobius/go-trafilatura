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

The latest recorded runs (September 11, 2026), on Go 1.26.0 and Go 1.27.1 with Readeck v2.1.2, have 968 passing checks, 14 failing checks, and 41 skips. The failures comprise **eight fallback-related checks and six language-detection checks**. Another 56 native coverage mappings are reported separately, not as passes or skips. Ordinary `go test` remains red; CI rejects changes outside the reviewed compatibility differences. These counts are not a document-extraction error rate. See the [test breakdown](UPSTREAM.md#current-results), [Readability migration](UPSTREAM.md#readeck-v2-migration), and [skip accounting](UPSTREAM.md#skipped-checks).

## Philosophy and Scope

**Bring your own HTML.** The primary use case is processing HTML already obtained by your application or scraper. `Extract` accepts an `io.Reader`; `ExtractDocument` accepts a parsed HTML DOM and leaves the caller's document unchanged. `OriginalURL` supplies context for metadata and relative URLs; it is not a request to download the page.

We follow upstream improvements to extraction, cleaning, metadata (including JSON-LD), tables, images, links, comments, and recovery. The compatibility target deliberately stops short of reproducing the whole Python package:

- **Acquisition:** crawler, downloader, feed, and sitemap parity is out of scope because the primary caller already supplies HTML. Existing CLI acquisition conveniences remain available, but are not expanded by this extraction update.
- **Fallback extractors:** we use [Readeck Go-Readability v2.1.2][readability] (`codeberg.org/readeck/go-readability/v2`) and `go-domdistiller`, not Python's Readability fork and jusText. Different selected content is an accepted tradeoff, not something to force into Python parity. Library fallbacks are off by default (`EnableFallback`); the CLI enables them unless `--no-fallback` is set. Use Python's `fast=True` for extractor-only comparisons.
- **Language detection:** `whatlanggo` is the sole detector, rather than Python's optional `py3langid`. It is a best-effort labeler and can misclassify short or repetitive text. With `TargetLanguage` set, wrong or empty labels can reject otherwise valid content. Lingua was evaluated and removed because its runtime and memory cost were too high for this workload.
- **Output and APIs:** results contain an HTML DOM, plain text, and metadata; the CLI supports HTML, text, and JSON. Python's Markdown, XML/TEI, CSV, YAML headers, configuration-file loading, deprecated wrappers, and process-global cache APIs are not reproduced. Standalone Simhash/fingerprint/token APIs are also excluded; per-extraction paragraph deduplication remains supported.

HTML parsing and date extraction also use Go dependencies, so exact output identity is not guaranteed. These implementation boundaries do not exclude fixes to supported extraction or metadata behavior. The [detailed compatibility document](UPSTREAM.md#scope) explains each omission, the fallback and language choices, and the remaining test-coverage gaps.

Balanced extraction can retry in recall mode when a short result covers little of the page. Recovery also understands embedded JSON content, and schema-identified discussion-forum posts are treated as main content; the selected fallback engine can still affect the final result.

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

### Language Detection

Text-language identification uses the [RadhiFadlillah/whatlanggo](https://github.com/RadhiFadlillah/whatlanggo) fork exclusively, not Python's optional `py3langid`. It classifies the longer of extracted body and comment text and uses the resulting ISO code for `Metadata.Language`. Short or repetitive text can be mislabeled, and some predictions have no ISO 639-1 code.

When `TargetLanguage` is set, a mismatching or empty detected code rejects the extraction; HTML language tags are also checked separately. With `TargetLanguage` unset, an incorrect prediction does not discard the document, but its language metadata may be wrong or absent. Known French, English, and Italian failures remain covered by enabled tests. See the [language-detection limitations](UPSTREAM.md#language-detection-limitations); the small test sample is not a production accuracy estimate.

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

## Development

Run the complete suite with `go test -mod=readonly ./... -count=1 -timeout 5m`, or `make test`. The Makefile accepts `GO`, `TEST_TIMEOUT`, and `TEST_ARGS` overrides. Tests do not regenerate source; `make generate` is a separate operation requiring re2go and a POSIX shell.

For the same reviewed-difference check used by CI, run `python scripts/check_tests.py` with Python 3.12 or newer. This uses only the Python standard library, runs every Go test, and checks exact failing leaves and assertion counts against [test-files/known-differences.json](test-files/known-differences.json). Unexpected failures, fixed/skipped/missing known differences, crashes, and incomplete native coverage fail the check; reference assertions remain enabled. `--log /path/to/results.jsonl` retains the unmodified Go test events.

[CI](.github/workflows/ci.yml) checks Linux and Windows on both supported Go versions, including formatting, module tidiness, builds, and `go vet`. It does not silently upgrade the minimum-version job to the preferred toolchain. See the [cleanup and CI notes](UPSTREAM.md#repository-cleanup-and-ci) for the reporting boundaries.

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

The measurements were collected during the v2.2.0 update using Go 1.24.2 and Python 3.12. The newer supported Go toolchains and updated date dependencies passed the then-existing regression suite; the subsequently synchronized Python tests expose additional differences. The scores above are not a new benchmark on those versions or on the corrected upstream phrase annotation. See [UPSTREAM.md](UPSTREAM.md#verification) for the pinned versions, methodology, and compatibility boundaries, including the [date-dependency comparison](UPSTREAM.md#date-dependency-update).

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
[readability]: https://codeberg.org/readeck/go-readability
[benchmark]: https://github.com/markusmobius/content-extractor-benchmark
