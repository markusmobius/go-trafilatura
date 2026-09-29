# Go-Trafilatura

Go-Trafilatura extracts main text, comments, and metadata from supplied HTML while preserving useful formatting and document structure. It is a Go port of [Trafilatura][0], the Python extractor created by [Adrien Barbaresi][1].

The current source follows the non-fallback extractor in Python Trafilatura [v2.2.0][last-version], with a native readability-lxml fallback and deliberate differences described below. It does not aim to reproduce every Python feature or guarantee identical output for every input.

This README describes the library as it works now, its design choices, and how to use it. [CHANGELOG.md](CHANGELOG.md) records changes by release. [UPSTREAM.md](UPSTREAM.md) is the technical reference for Python compatibility, intentional differences, and verification evidence.

The current released version is **v2.2.6**. Non-FAST extraction uses only bundled readability-lxml; FAST disables that external fallback. Reader parsing now treats noscript contents as HTML children.

## Table of Contents

- [Philosophy and Scope](#philosophy-and-scope)
- [Extraction Choices](#extraction-choices)
- [Language Detection](#language-detection)
- [Usage as a Go Package](#usage-as-a-go-package)
- [Native Readability-Lxml](#native-readability-lxml)
- [Usage as a CLI Application](#usage-as-a-cli-application)
- [Development](#development)
- [Current Quality and Speed](#current-quality-and-speed)
- [Non-FAST Trafilatura](#non-fast-trafilatura)
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
- **One internal fallback implementation.** `EnableFallback: true` permits only the bundled native readability-lxml candidate. Mozilla/Go-ReadabilityV2, DomDistiller and supplied/custom candidates are never used by Trafilatura. Library fallback is off by default; the CLI enables lxml unless `--no-fallback` is set. Native recall and baseline recovery remain available in FAST mode.
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

## Native Readability-Lxml

We removed caller-supplied fallback candidates because they can bypass
Trafilatura's input cleanup. A standalone result may retain long boilerplate,
such as a legal footer, that passes the fallback length checks and replaces
the article.

In a controlled comparison using Go 2.2.2 on 6,554 pages, supplied candidates
raised final external fallback from **780 pages (11.90%)** to **2,408 (36.74%)**.
DomDistiller accounted for most of the increase: **4 to 1,614 selections**.
Trafilatura now prepares its own candidates and uses only bundled readability-lxml.
Standalone Mozilla Readability and DomDistiller remain separate extractors.
See [UPSTREAM.md](UPSTREAM.md#why-supplied-candidates-were-removed) for the controls
and the distinction between candidate reuse and fallback algorithm choice.

Fallback is off by default. To enable bundled readability-lxml:

```go
options := trafilatura.Options{
  EnableFallback: true,
}
```

`ReadabilityFallback` and `FallbackCandidates` are deprecated and ignored;
legacy enum values remain source-compatible but cannot select another engine.
Standalone Mozilla/Go-ReadabilityV2 is a separate extractor and is unchanged.
The Apache-2.0 port follows Python Trafilatura
2.2.0's bundled readability-lxml, with Arc90, starrhorne/iterationlabs and
gfxmonk/python-readability ancestry. See [CHANGELOG.md](CHANGELOG.md) for validation
and remaining Python differences.

`Extract` uses scripting-disabled HTML parsing, so noscript markup becomes
child nodes instead of leaking into extracted text. For `ExtractDocument`,
parse its input with `html.ParseWithOptions(reader,
html.ParseOptionEnableScripting(false))`. Document extraction preserves the
supplied tree and does not silently reparse it. Keep a separate default parser
input for standalone Mozilla Readability, whose noscript image recovery expects
raw markup.

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

The [2026-09-29 shared benchmark](https://github.com/markusmobius/content-extractor-benchmark/blob/ec719092d12f4d2a438dd29d9f4405aab6e0a321/README.md#results-2026-09-29)
compares all six implementations on **2,659 saved pages**: 983 LegoNews,
181 ScrapingHub and 1,495 WCXB. All six READMEs use this same comparison.

### Extraction Speed

| Extractor | Go Version | Rust Version | Go ms/page | Rust ms/page | Go/Rust |
| --- | --- | --- | ---: | ---: | ---: |
| Readability | 0.6.0 | 0.6.5 | 4.755 | 3.945 | 1.21x |
| DomDistiller | 1.0.0 | 1.0.1 | 6.159 | 3.400 | 1.81x |
| Trafilatura FAST | 2.2.6 | 2.2.6 | 11.329 | 6.570 | 1.72x |

Times are means of **all four measured passes after one warmup**. Go/Rust is
Go time divided by Rust time, not an old/new release speedup. Measured versions
are shown explicitly; later documentation-only releases are not new measurements.

The run used Windows 11, Ryzen AI 7 PRO 350, Go 1.27.1 and Rust 1.98.1 GNU
with ThinLTO/mimalloc. Extraction includes required working copies, metadata
and text rendering. File I/O, startup, IPC, response serialization and scoring
are excluded. Comments, pagination and Trafilatura external fallback are off;
tables are on. Power and sleep checks passed.

Parsing is separate: **Go 11.283 / Rust 6.386 ms/page**, charged once per
language/page for the shared suite. It includes decoding, DOM construction and
the separate Trafilatura noscript tree when needed. These are extraction-stage
comparisons, not complete request latencies.

### Text Quality

Go and Rust have the same text scores for each engine. Errors are listed in
LegoNews / ScrapingHub / WCXB order and remain in the scoring denominators.

| Extractor | LegoNews F1 | ScrapingHub F1 | WCXB F1 | Errors |
| --- | ---: | ---: | ---: | --- |
| Readability | 87.82711% | 95.20557% | 78.47603% | 7 / 0 / 28 |
| DomDistiller | 86.74080% | 92.74280% | 74.39696% | 0 / 0 / 0 |
| Trafilatura FAST | 90.91534% | 96.15663% | 78.51703% | 4 / 0 / 10 |

The corpora use different scoring rules; their F1 scores must not be averaged.
Equal text scores do not imply identical metadata: Trafilatura differs on one
title and one author field. The [full report](https://github.com/markusmobius/content-extractor-benchmark/blob/49c426d6135df81b7d492bea7e6aec8e6d77d80c/go_rust_shared_performance_2026_09_29.json)
contains metadata scores, differences, every pass and source/build identities.

## Non-FAST Trafilatura

A [separate run](https://github.com/markusmobius/content-extractor-benchmark/blob/49c426d6135df81b7d492bea7e6aec8e6d77d80c/go_rust_lxml_performance_2026_09_29.json)
measured both 2.2.6 ports with bundled readability-lxml enabled, using the same
corpora and four-pass protocol. Its measurements are not pooled with FAST.

| Mode | Go ms/page | Rust ms/page | Go/Rust |
| --- | ---: | ---: | ---: |
| Non-FAST lxml | 24.745 | 10.910 | 2.27x |

Parsing was Go 11.518 / Rust 6.545 ms/page. Both ports scored **91.13924% /
95.98168% / 79.56922% F1**, with **4 / 0 / 9 errors** in corpus order.
WCXB F1 is below the older 81.17181% configuration that also allowed DomDistiller
rescue; FAST has one extra LegoNews rejection. The simpler fallback policy is
not a claim of universally better quality. Python jusText is not implemented.

### Fallback Selection Rates

On a separate **6,554-page unannotated corpus**, the final returned sources
were as follows in both ports. These are selection rates, not accuracy scores;
failures stay in the denominator. Standalone extractors are outside this count.

| Final Content Source | FAST | Non-FAST |
| --- | ---: | ---: |
| Native core | 5,726 | 5,604 |
| Native recall | 81 | 58 |
| Bundled readability-lxml | 0 | 202 |
| Mozilla / DomDistiller / custom | 0 | 0 |
| Internal baseline | 726 | 669 |
| No result | 21 | 21 |
| **External fallback total** | **0 / 6,554 (0%)** | **202 / 6,554 (3.082%)** |

In non-FAST, lxml ran on all inputs but supplied final content on only 202;
calls and temporary selections are not the final fallback rate. Native recall
and baseline are internal recovery, not external fallback. See
[UPSTREAM.md](UPSTREAM.md) for the evidence and limits.

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
