# go-trafilatura

`go-trafilatura` extracts main text, comments and metadata from supplied HTML,
preserving useful formatting and document structure. It is a native Go port of
[adbar/trafilatura](https://github.com/adbar/trafilatura), created by Adrien Barbaresi.

## Philosophy

Our extractor packages share three principles:

1. **Bring your own HTML.** Keep page acquisition separate from extraction.
   The primary workflow uses HTML supplied by the caller, who controls fetching,
   caching, rendering, retries and scheduling.
2. **Stay close to upstream.** Preserve the algorithms and behavior of each
   package's declared upstream reference as closely as possible. Document
   deliberate differences and compatibility limits in [UPSTREAM.md](UPSTREAM.md)
   rather than claiming exact equivalence on every page.
3. **Provide very fast Go and Rust packages.** Run extraction natively, without
   a Python or Java runtime. Improve throughput and allocation efficiency while
   preserving intended behavior, and substantiate performance with reproducible
   benchmarks that report quality alongside speed.

## Overview

The current `go-trafilatura` release is **v2.2.6**. It accepts an `io.Reader`
or an existing HTML tree and returns content/comment trees, plain text and
metadata, including JSON-LD, dates and language. Tables, images, links and
per-extraction deduplication are configurable. Extraction preserves input trees.

The reference is `adbar/trafilatura` 2.2.0's non-fallback core, with documented
Go differences and an optional bundled readability-lxml implementation. The
corresponding [rust-trafilatura](https://github.com/markusmobius/rust-trafilatura)
package follows `go-trafilatura`. Library entry points never fetch the original
URL; the separate CLI can download URLs, feeds and sitemaps.

## Installation

```sh
go get github.com/markusmobius/go-trafilatura/v2@v2.2.6
```

Use Go 1.26.0 or newer. [go.mod](go.mod) selects Go 1.27.1 for development.
The `/v2` module suffix is required. See [CHANGELOG.md](CHANGELOG.md) for releases.

## Usage

Extract text from HTML already held in memory:

```go
package main

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/markusmobius/go-trafilatura/v2"
)

func main() {
	pageURL, err := url.Parse("https://example.org/research")
	if err != nil {
		panic(err)
	}
	source := `<html><head><title>Research results</title></head><body><article>
<h1>Research results</h1>
<p>The research team compared several methods for extracting articles from saved
web pages. Every method received the same original HTML, and the evaluation
kept the reference text separate from the input supplied to each extractor.</p>
<p>The report records the complete experiment, including errors and repeated
measurements. Its results describe this collection of pages and do not promise
the same quality or execution time for every website.</p>
</article></body></html>`
	result, err := trafilatura.Extract(strings.NewReader(source), trafilatura.Options{
		OriginalURL:     pageURL,
		InputEncoding:   "utf-8",
		ExcludeComments: true,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(result.ContentText)
}
```

| Entry Point | Input |
| --- | --- |
| `Extract` | HTML from an `io.Reader`, with decoding and parsing |
| `ExtractDocument` | An existing `*html.Node`, preserved during extraction |

Use `ContentText` for text, `ContentNode` for the extracted tree and `Metadata`
for title, author, date and language. Comment output is separate. See the
[go-trafilatura API reference](https://pkg.go.dev/github.com/markusmobius/go-trafilatura/v2)
and [examples](examples) for complete types and additional uses.

### Command Line

The optional CLI accepts saved HTML and can also fetch pages:

```sh
go install github.com/markusmobius/go-trafilatura/v2/cmd/go-trafilatura@v2.2.6
go-trafilatura article.html
go-trafilatura --help
```

The `batch`, `feed` and `sitemap` subcommands have their own `--help`. The CLI
enables bundled fallback unless `--no-fallback` is used; library defaults differ.

## Options

Pass `Options{}` for library defaults:

| Option | Default | Effect |
| --- | --- | --- |
| `OriginalURL` | Unset | URL context for metadata and relative links; no download. |
| `InputEncoding` | `""` | Detect encoding; a known label such as `utf-8` skips statistical detection. |
| `EnableFallback` | `false` | FAST mode; `true` permits the bundled readability-lxml candidate. |
| `Focus` | `Balanced` | Balance precision and recall, or select `FavorPrecision` / `FavorRecall`. |
| `ExcludeComments` / `ExcludeTables` | `false` | Keep comments and tables unless explicitly excluded. |
| `IncludeImages` / `IncludeLinks` | `false` | Opt into images and links in extracted HTML. |
| `TargetLanguage` | Unset | Reject a mismatching or empty language label when a target is supplied. |
| `HtmlDateMode` | Default | Select date-search behavior explicitly or follow the fallback setting. |

### Input and Language

`InputEncoding` describes the bytes actually supplied, not their pre-decoding
source. It retains normalization and soft-hyphen removal; unsupported labels
fail. Parsed-document extraction ignores this option. To match reader parsing,
construct input with `html.ParseWithOptions(reader, html.ParseOptionEnableScripting(false))`;
noscript markup then becomes child nodes. Extraction does not silently reparse a DOM.

`go-py3langid` supplies automatic language metadata from an embedded model.
It classifies the longer of body/comments; comments win ties. Without a target,
a wrong prediction affects metadata rather than discarding the article. Short,
noisy or multilingual input can be mislabeled. Model initialization is a one-time
process cost, while classification adds per-document work.

### Optional Fallback

FAST disables external fallback, not native recall or baseline recovery.
Enabling fallback uses only the internally prepared bundled readability-lxml
implementation. Legacy `ReadabilityFallback` and `FallbackCandidates` fields
are ignored; they cannot select another extractor.

Caller-supplied candidates were removed because they could bypass input cleanup
and retain long boilerplate that passed length checks. In the controlled
`go-trafilatura` 2.2.2 comparison, supplied candidates raised final external
fallback from **780/6,554 (11.90%)** to **2,408/6,554 (36.74%)**;
`go-domdistiller` selections rose from **4 to 1,614**. This is an input-preparation
issue, not a claim that another extractor performs no cleanup. See the
[controlled evidence](UPSTREAM.md#why-supplied-candidates-were-removed).

A [separate non-FAST run](https://github.com/markusmobius/content-extractor-benchmark/blob/49c426d6135df81b7d492bea7e6aec8e6d77d80c/go_rust_lxml_performance_2026_09_29.json)
measured `go-trafilatura` 2.2.6 at **24.745 ms/page** and `rust-trafilatura` 2.2.6
at **10.910 ms/page** (**2.27x** Go/Rust), with parsing at **11.518 / 6.545 ms/page**.
Both scored **91.13924% / 95.98168% / 79.56922% F1**, with **4 / 0 / 9 errors**
in LegoNews / ScrapingHub / WCXB order. Do not pool these timings with FAST.
The simpler policy is not universally more accurate; its WCXB score is below
the older **81.17181%** configuration that also allowed `go-domdistiller` rescue.

On a separate 6,554-page unannotated corpus, final external fallback supplied
**0/6,554 (0%)** results in FAST and **202/6,554 (3.082%)** in non-FAST. These are
selection rates, not accuracy scores; failures remain in the denominator.
Calls and temporary selections are not final-source counts. The current policy
changes both candidate preparation and the algorithm set, not just reuse.

## Current Quality and Speed

The [2026-09-29 shared benchmark](https://github.com/markusmobius/content-extractor-benchmark/blob/ec719092d12f4d2a438dd29d9f4405aab6e0a321/README.md#results-2026-09-29)
compares the six packages below on **2,659 saved pages**: 983 LegoNews,
181 ScrapingHub and 1,495 WCXB.

### Extraction Speed

| Go Package (Measured Version) | Rust Package (Measured Version) | Go ms/page | Rust ms/page | Go/Rust |
| --- | --- | ---: | ---: | ---: |
| `go-readabilityV2` 0.6.0 | `rust-readability-v2` 0.6.5 | 4.755 | 3.945 | 1.21x |
| `go-domdistiller` 1.0.0 | `rust-domdistiller` 1.0.1 | 6.159 | 3.400 | 1.81x |
| `go-trafilatura` 2.2.6 (FAST) | `rust-trafilatura` 2.2.6 (FAST) | 11.329 | 6.570 | 1.72x |

Times are means of **all four measured passes after one warmup**. Go/Rust is
the named Go package's time divided by the named Rust package's time, not an
old/new release speedup. Later documentation-only releases do not change the
versions actually measured.

The run used Windows 11, Ryzen AI 7 PRO 350, Go 1.27.1 and Rust 1.98.1 GNU
with ThinLTO/mimalloc. Extraction includes required working copies, metadata
and text rendering. File I/O, startup, IPC, response serialization and scoring
are excluded. Comments and pagination are off; tables are on.
`go-trafilatura` and `rust-trafilatura` use FAST with external fallback disabled.
Power and sleep checks passed.

Parsing is separate: **Go 11.283 / Rust 6.386 ms/page**, charged once per
language/page for the shared suite. It includes decoding, DOM construction and
the separate `go-trafilatura` / `rust-trafilatura` noscript tree when needed.
These are extraction-stage comparisons, not complete request latencies.

### Text Quality

Each named pair has equal text scores. Errors are listed in LegoNews /
ScrapingHub / WCXB order and remain in the scoring denominators.

| Go Package | Rust Package | LegoNews F1 | ScrapingHub F1 | WCXB F1 | Errors |
| --- | --- | ---: | ---: | ---: | --- |
| `go-readabilityV2` | `rust-readability-v2` | 87.82711% | 95.20557% | 78.47603% | 7 / 0 / 28 |
| `go-domdistiller` | `rust-domdistiller` | 86.74080% | 92.74280% | 74.39696% | 0 / 0 / 0 |
| `go-trafilatura` (FAST) | `rust-trafilatura` (FAST) | 90.91534% | 96.15663% | 78.51703% | 4 / 0 / 10 |

The corpora use different scoring rules; their F1 scores must not be averaged.
Equal text scores do not imply identical metadata: `go-trafilatura` and
`rust-trafilatura` differ on one title and one author field. The
[full report](https://github.com/markusmobius/content-extractor-benchmark/blob/49c426d6135df81b7d492bea7e6aec8e6d77d80c/go_rust_shared_performance_2026_09_29.json)
contains metadata scores, differences, every pass and source/build identities.

## Compatibility and Limitations

- **Declared reference.** Follow `adbar/trafilatura` 2.2.0's supplied-HTML core,
  compared with its `fast=True` mode. Its default fallback combination differs.
- **Deliberate differences.** Complete nested cleanup, cleaned-body recovery
  decisions, normalized author selectors and automatic language annotation
  are documented in [UPSTREAM.md](UPSTREAM.md).
- **Native dependencies.** HTML repairs, rendering and date interpretation can
  differ from Python. Finite corpus agreement is not universal output parity.
- **Limited scope.** No browser rendering or JavaScript execution is provided.
  The Python crawler, format suite, global caches and `miso-belica/jusText`
  fallback are not implemented; per-extraction deduplication remains supported.
- **Not a sanitizer.** Sanitize extracted HTML before displaying untrusted input.

## Development

Use Go and Python 3.12 or newer for the reviewed-difference gate:

```sh
python scripts/check_tests.py
go vet -mod=readonly ./...
go mod tidy -diff
```

The gate runs every Go test and checks exact reviewed differences against
[test-files/known-differences.json](test-files/known-differences.json). Ordinary
`go test -mod=readonly ./... -count=1 -timeout 5m` reports those differences as
failures; unexpected, missing or skipped cases fail the gate. Do not weaken
reference assertions. Regeneration via `make generate` additionally requires
re2go and a POSIX shell; ordinary builds use checked-in generated code.
See [AGENTS.md](AGENTS.md) for documentation and release requirements.

## License and Credits

`go-trafilatura` is distributed under [Apache-2.0](LICENSE). Adrien Barbaresi
created [adbar/trafilatura](https://github.com/adbar/trafilatura), the original
Python package on which this port builds. Markus Mobius maintains `go-trafilatura`.

The bundled readability-lxml ancestry credits Arc90 for the original algorithm,
starrhorne and iterationlabs for the Ruby port, and gfxmonk for the Python port.
The [pinned upstream notice](https://github.com/adbar/trafilatura/blob/c1bc9531a2a978326112ca9987e1382745116136/trafilatura/readability_lxml.py)
also links the `timbertson/python-readability` and `buriy/python-readability`
contributors. Inherited notices and dependency licenses remain in force.

For research citation, see Adrien Barbaresi's
[2021 ACL/IJCNLP paper](https://aclanthology.org/2021.acl-demo.15/),
[2019 KONVENS paper](https://hal.archives-ouvertes.fr/hal-02447264/document) and
[2016 WAC-X paper](https://hal.archives-ouvertes.fr/hal-01371704v2/document).
