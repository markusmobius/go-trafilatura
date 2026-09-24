# Go-Trafilatura

Go-Trafilatura extracts main text, comments, and metadata from supplied HTML while preserving useful formatting and document structure. It is a Go port of [Trafilatura][0], the Python extractor created by [Adrien Barbaresi][1].

The current source follows the non-fallback extractor in Python Trafilatura [v2.2.0][last-version], with native Go fallback engines and deliberate differences described below. It does not aim to reproduce every Python feature or guarantee identical output for every input.

This README describes the library as it works now, its design choices, and how to use it. [CHANGELOG.md](CHANGELOG.md) records changes by release. [UPSTREAM.md](UPSTREAM.md) is the technical reference for Python compatibility, intentional differences, and verification evidence.

The current released version is **v2.2.2**. This documentation refresh changes no Go source, dependency versions or release tags.

## Table of Contents

- [Philosophy and Scope](#philosophy-and-scope)
- [Extraction Choices](#extraction-choices)
- [Language Detection](#language-detection)
- [Usage as a Go Package](#usage-as-a-go-package)
- [Usage as a CLI Application](#usage-as-a-cli-application)
- [Development](#development)
- [Current Quality and Speed](#current-quality-and-speed)
- [Performance](#performance)
- [Acknowledgements](#acknowledgements)
- [License](#license)

## Philosophy and Scope

**Bring your own HTML.** The primary use case is processing HTML already obtained by your application or scraper. `Extract` accepts an `io.Reader`; `ExtractDocument` accepts a parsed HTML DOM and leaves the caller's document unchanged. `OriginalURL` supplies context for metadata and relative URLs; it is not a request to download the page.

The library extracts main content, comments, and metadata, including JSON-LD. It supports tables, images, links, structural formatting, and recovery from embedded content. Results contain an HTML DOM, plain text, and metadata. The CLI supports HTML, text, and JSON output.

The returned HTML is not a security sanitizer. Applications rendering untrusted content must apply their own sanitization policy.

Extraction runs natively in Go without Python or runtime model downloads. Callers control acquisition and concurrency. The existing CLI can fetch URLs, feeds, and sitemaps, but reproducing Python's crawler is not a goal.

Python's Markdown, XML/TEI, CSV, YAML headers, configuration-file interfaces, and process-global cache APIs are outside the Go API. Standalone Simhash and fingerprint APIs are also excluded; per-extraction deduplication remains supported. See the [detailed scope](UPSTREAM.md#scope) for the full boundary.

## Extraction Choices

- **Python-aligned core.** We follow the supplied-HTML, non-fallback behavior of the pinned Python 2.2.0 source. Compare with Python's `fast=True`; Python's default fallback path is a different algorithm combination.
- **Native Go fallbacks.** Optional [Go-ReadabilityV2 v0.6.0][readability] and [Go-DomDistiller][dom-distiller] retain the Go candidate ordering, acceptance rules, cleanup, and recall rescue. They are not substitutes intended to match Python's Readability fork and jusText. Custom `FallbackCandidates` are supported. Library fallbacks are off by default; the CLI enables them unless `--no-fallback` is set.
- **Complete cleanup and accurate recovery decisions.** Cleaning removes all matching unwanted elements, including nested ones. Recovery decisions use text from the final cleaned body, so removed duplicates cannot make an incomplete article appear long enough. These are deliberate corrections to Python behavior, not website-specific exceptions.
- **Whitespace-tolerant author selection.** Author-selection and author-discard rules trim ID whitespace and normalize class whitespace, so values such as `class=" username "` remain usable. This deliberate difference from Python does not broaden content selectors or change case matching. Single-word meta-tag authors are also retained. See the [metadata boundaries](UPSTREAM.md#metadata-and-language).
- **Upstream candidate selection.** We retain Python's narrower `article ` class-prefix rule. The broader old Go `article` prefix is not restored; other Python-alignment improvements remain in place.
- **Go parsing and date handling.** HTML and date processing use native Go dependencies. Parser repairs, rendering, and date interpretation can therefore differ from Python even when the extraction rules agree.

Balanced extraction can retry in recall mode when a short result covers little of the page. Recovery also understands embedded JSON content, and schema-identified discussion-forum posts are treated as main content. Different fallback engines can select different article regions. The [compatibility reference](UPSTREAM.md#deliberate-core-deviations) describes the exact deviations and their consequences; corpus agreement is not proof of equivalence on arbitrary input.

## Language Detection

[Go-py3langid v0.4.0](https://github.com/markusmobius/go-py3langid) provides automatic `Metadata.Language` using the same model as Python's py3langid. Its pure-Go implementation avoids a Python, NumPy, or CGO runtime dependency. One private identifier is initialized lazily and reused across extractions; model startup and memory are one-time process costs, while classification adds work per document.

The classifier uses the longer of the extracted body and comments by Unicode code-point count; equal lengths select comments. Setting `TargetLanguage` rejects a mismatching or empty label, with HTML language tags checked separately. Without a target, a wrong prediction affects metadata but does not discard the article. This automatic annotation is an intentional difference from Python's target-only classification.

Short, noisy, or multilingual text can be mislabeled, and longer comments can determine the label. Raw model labels are retained without a confidence threshold or remapping. See the [language comparison](UPSTREAM.md#language-detection) for details.

## Usage as a Go Package

Use Go 1.26.0 or newer. [go.mod](go.mod) selects Go 1.27.1 as the preferred development toolchain. To add the package, run:

```sh
go get github.com/markusmobius/go-trafilatura/v2
```

Import the package in your application:

```go
import "github.com/markusmobius/go-trafilatura/v2"
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
go install github.com/markusmobius/go-trafilatura/v2/cmd/go-trafilatura@latest
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

Ordinary `go test` reports the known fallback differences as failures; a passing reviewed-difference check means that only those reviewed differences remain. Dated test totals and skipped coverage are recorded in [UPSTREAM.md](UPSTREAM.md#current-results), not treated as an extraction accuracy score.

[CI](.github/workflows/ci.yml) checks Linux and Windows on both supported Go versions, including formatting, module tidiness, builds, and `go vet`. It does not silently upgrade the minimum-version job to the preferred toolchain. See the [verification scope](UPSTREAM.md#verification) for the reporting boundaries.

## Current Quality and Speed

The [2026-09-23 benchmark JSON](https://github.com/markusmobius/content-extractor-benchmark/blob/d433ab637f0a56c0926aa3698f470794a553472f/go_rust_shared_performance_2026_09_23.json)
is the source for these tables. All six released engines use the same 2,659
development pages: 983 LegoNews, 181 ScrapingHub and 1,495 WCXB. The three F1
scores use different scoring rules and must not be averaged. Errors are shown
in that corpus order and remain in the denominators.

| Implementation | LegoNews F1 | ScrapingHub F1 | WCXB F1 | Errors |
| --- | ---: | ---: | ---: | --- |
| go-readabilityV2-0.6.0 | 87.82711% | 95.20557% | 78.47603% | 7 / 0 / 28 |
| rust-readability-0.6.3 | 87.82711% | 95.20557% | 78.47603% | 7 / 0 / 28 |
| go-domdistiller-1.0.0 | 86.74080% | 92.74280% | 74.39696% | 0 / 0 / 0 |
| rust-domdistiller-1.0.1 | 86.74080% | 92.74280% | 74.39696% | 0 / 0 / 0 |
| go-trafilatura-2.2.2 | 90.88412% | 96.15663% | 78.49352% | 3 / 0 / 10 |
| rust-trafilatura-2.2.4 | 90.88412% | 96.15663% | 78.49352% | 3 / 0 / 10 |

| Implementation | Shared Parse ms/page | Extraction ms/page | Extraction ms/page, All Four Passes |
| --- | ---: | ---: | ---: |
| go-readabilityV2-0.6.0 | 5.638 | 2.669 | 2.705 |
| rust-readability-0.6.3 | 2.525 | 2.441 | 2.451 |
| go-domdistiller-1.0.0 | 5.638 | 3.618 | 3.628 |
| rust-domdistiller-1.0.1 | 2.525 | 1.965 | 1.973 |
| go-trafilatura-2.2.2 | 5.638 | 6.815 | 6.839 |
| rust-trafilatura-2.2.4 | 2.525 | 4.000 | 4.026 |

One full warmup precedes four measured passes. The first two timing columns
use the common best two complete passes (1 and 3), an optimistic estimate;
the final column retains the all-four mean. Go/Rust extraction ratios from
unrounded means are **1.09x Readability, 1.84x DomDistiller and 1.70x Trafilatura**.
Parsing is charged once per language/page, not once per engine. All-four parse
means are Go 5.667 and Rust 2.531 ms/page.

These Windows 11 / Ryzen AI 7 PRO 350 measurements use Go 1.27.1 and Rust
1.98.1 GNU, the released Trafilatura dependency graphs, and Rust ThinLTO/mimalloc.
Parsing includes eager decoding, normalization and DOM construction after the
file read; extraction includes private working copies, native metadata and text
rendering. Rust temporary trees are destroyed inside the timer; Go uses normal
GC, which can cross stage boundaries. File I/O, startup, IPC and scoring are
excluded. Fallbacks, comments and pagination are off; tables are on.

Every repeated scored output was stable. Go/Rust Readability and DomDistiller
match all scored outputs; Trafilatura retains two metadata-only differences.
Separate metadata scores, exact source pins and protocol limits are in
[UPSTREAM.md](UPSTREAM.md#released-suite-benchmark). Go-Trafilatura remains
2.2.2. These are shared-input suite timings, not standalone end-to-end latency
or measurements of the fallback-enabled CLI. Historical measurements use
different protocols and are not pooled with this run.

## Performance

Extraction time depends on document size and structure, character-encoding detection, metadata processing, and optional fallback extractors. Use `ExtractDocument` when you already have a parsed DOM, or supply a [known input encoding](#known-input-encoding) to avoid statistical charset detection.

The boilerplate text filter matches whole lines with Python-compatible whitespace and case handling. Other matchers use checked-in Go code generated by [re2go]; this is not a translation of every regex in the package. Normal builds require neither re2go nor CGO; the generator is needed only when regenerating that source.

For measurements, use the [comparison tooling](scripts/comparison/README.md) or the [separate benchmark project][benchmark]. Evaluate representative supplied HTML with matching options and dependencies. Test counts and historical measurements do not establish current throughput or exact Python parity.

## Acknowledgements

The upgrade from Go-Trafilatura v2.0.0 to v2.2.1 was performed with assistance from GPT-6 Astra. Coding LLMs continue to assist maintenance and upstream synchronization.

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
[paper-1]: https://aclanthology.org/2021.acl-demo.15/
[paper-2]: https://hal.archives-ouvertes.fr/hal-02447264/document
[paper-3]: https://hal.archives-ouvertes.fr/hal-01371704v2/document
[wac-x]: https://www.sigwac.org.uk/wiki/WAC-X
[k-web]: https://www.dwds.de/d/k-web
[re2go]: https://re2c.org/manual/manual_go.html
[dom-distiller]: https://github.com/markusmobius/go-domdistiller/
[readability]: https://github.com/markusmobius/go-readabilityV2
[benchmark]: https://github.com/markusmobius/content-extractor-benchmark
