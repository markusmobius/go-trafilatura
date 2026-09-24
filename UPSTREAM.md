# Upstream Compatibility

This document compares the current Go implementation with **Python Trafilatura 2.2.0**, pinned to [c1bc9531a2a978326112ca9987e1382745116136](https://github.com/adbar/trafilatura/commit/c1bc9531a2a978326112ca9987e1382745116136). It distinguishes deliberate deviations, unsupported interfaces, and behavior whose exact equivalence is not established. Later Python commits are not included.

For the current library overview and usage, see [README.md](README.md). For changes between Go releases, see [CHANGELOG.md](CHANGELOG.md).

## Released Suite Benchmark

The [2026-09-23 JSON](https://github.com/markusmobius/content-extractor-benchmark/blob/d433ab637f0a56c0926aa3698f470794a553472f/go_rust_shared_performance_2026_09_23.json)
is authoritative for the current [README tables](README.md#current-quality-and-speed).
Its published-file SHA-256 (LF line endings) is `7d7be9839f1652606cb91850af5134b188f2508be25623df889372dab4a06cc6`.
Read text scores at `quality[worker][engine].evaluations[corpus].overall.f1`,
selected timings at `overall`, and all-four timings at `all_passes`.

Go pins are Readability 0.6.0 (`db6ab179f951f80ad176850001aaf486e2fbd367`),
DomDistiller 1.0.0 (`25b8d046ffb4053bf68345d6fa59bc9ae1961ad8`) and Trafilatura
2.2.2 (`f4684e100869274311107325e3b72e47cc78db20`). Rust uses Readability 0.6.3,
DomDistiller 1.0.1 and Trafilatura 2.2.4; exact commits, dependency graphs and
binary hashes are in the build receipts. Rust-Trafilatura remains private;
reproducing that suite requires authorized access. No Go version or tag changes.

| Implementation | Author Sets Exact / 1,290 | Author-Unit F1 | Titles Exact / 2,364 | Dates Exact / 1,530 |
| --- | ---: | ---: | ---: | ---: |
| go-readabilityV2-0.6.0 | 640 | 56.38767% | 1,247 | 763 |
| rust-readability-0.6.3 | 640 | 56.38767% | 1,247 | 763 |
| go-domdistiller-1.0.0 | 0 | 0.00000% | 1,106 | 0 |
| rust-domdistiller-1.0.1 | 0 | 0.00000% | 1,106 | 0 |
| go-trafilatura-2.2.2 | 695 | 58.80923% | 1,228 | 1,227 |
| rust-trafilatura-2.2.4 | 696 | 58.86640% | 1,227 | 1,227 |

Metadata uses only nonempty supplied annotations; unannotated is not negative,
and missing output is not filled by another engine. All six scored-output
digests match the preceding September 22 report. Trafilatura differs between
Go/Rust only on the title of `legonews/klaenge-des-verschweigens.de.geschichte.html`
and the author of `legonews/golf.de-augusta.html`; extracted text matches.

The full 2,659-page development run used seed 20260922, one warmup and four
measured passes. Passes 1 and 3 were selected by combined extraction time for
every row (5,318 observations each); all-four means retain 10,636 observations.
Worker order is balanced per page; three-engine order is a partial six-pass
block. Go uses `GOMAXPROCS=1`, `GOGC=100`, without forced collection. Native
timers exclude file reads and IPC; parsing and extraction stay separate.
The 26,590-response audit passed with no recorded sleep and AC power throughout.
This is a coordinated released-suite comparison with native fallbacks disabled,
not an isolated language change, Python-parity result or unseen holdout test.
The source-test and older request-latency records below remain historical.

## Comparison Setup

Python is the behavioral reference for **non-fallback extraction**, subject to the explicit deviations below. Compare the same supplied HTML, URL context, and options; do not compare the two libraries' defaults without accounting for their different fallback settings.

| Behavior | Go | Python Reference |
| --- | --- | --- |
| Core extraction only | `EnableFallback: false` | `fast=True` |
| Balanced extraction | `Focus: Balanced` | Neither `favor_precision` nor `favor_recall` |
| Precision or recall preference | `FavorPrecision` or `FavorRecall` | Corresponding `favor_precision` or `favor_recall` flag |
| Comments | Inverse of `ExcludeComments` | `include_comments` |
| Tables | Inverse of `ExcludeTables` | `include_tables` |
| Images and links | `IncludeImages`, `IncludeLinks` | `include_images`, `include_links` |
| Deduplication | `Deduplicate`, scoped to one extraction | `deduplicate`; process-global state is not reproduced |
| Language filtering | `TargetLanguage` | `target_language` |
| URL context | `OriginalURL` | `url` |
| Metadata comparison | Always part of the result | Request metadata explicitly, for example `with_metadata=True` |

Date-search settings and extraction-size limits must also match for a meaningful comparison. The reference fixture records Python 3.12.13, lxml 6.1.3, py3langid 0.4.0, and the other pinned dependencies. Go dependency versions are in [go.mod](go.mod).

## Scope

The comparison covers main text, comments, pruning, structural formatting, tables, images, links, metadata and JSON-LD, embedded-content recovery, and recall decisions. Acquisition and Python-only output formats are not part of this contract.

Outside the listed deviations, the core follows Python's selector predicates, whole-line boilerplate filters, text/tail handling, paragraph/list/quote/table processing, forum routing, and recovery rules. These are supported by independent reference tests, not a claim that every possible input produces identical output.

The article-class predicate is `startsWith(class, "article ")`, **the same as Python**. A broader `"article"` prefix is not used. Ordered JSON-LD traversal, targeted malformed-JSON recovery, and cleanup of invalid scalar metadata characters are also supported; JSON-LD is not an omitted feature.

## Deliberate Core Deviations

| Area | Python 2.2.0 | Go Behavior and Consequence |
| --- | --- | --- |
| Removing unwanted elements | Live removal can enter a detached nested subtree and skip later matching elements. | Collects matching nodes before removal. All matches for that cleaning tag are removed, with tails and tag order preserved. This can remove boilerplate that Python leaves and change the selected article. |
| Text used for recovery decisions | Can retain a text snapshot taken before final duplicate removal and output cleanup. | Measures the final body after duplicate removal, `done` removal, and `div` unwrapping. Removed text cannot prevent baseline recovery or inflate recall thresholds. The thresholds themselves are unchanged. |
| Automatic language metadata | Runs statistical classification when a target language is requested; otherwise leaves the language field unset. | Classifies accepted extractions even without a target and fills `Metadata.Language`. A prediction only rejects extraction when `TargetLanguage` is nonempty. |

Complete collected removal applies to core cleaning, not every mutation-aware extraction handler. The recovery difference can affect content, output size, and acceptance decisions; it is not merely a text-formatting difference. The language-annotation difference does not control article selection.

These behaviors are implemented in [html-processing.go](html-processing.go), [main-extractor.go](main-extractor.go), and [core.go](core.go). Synthetic tests cover nested removal, preserved tails, idempotence, final-body text, duplicate-independent recovery, and no-target language metadata. There are no website-specific corrections.

## Fallback Extractors

The fallback pipeline is an intentional algorithm difference, not a Python-parity target.

| Area | Go | Python Reference |
| --- | --- | --- |
| Library default | Fallbacks disabled | Fallbacks enabled unless `fast=True` |
| Engines | Go-ReadabilityV2 0.6.0 and Go-DomDistiller | Bundled Readability and jusText |
| Selection | Go candidate ordering, acceptance/stopping rules, sanitization, and bounded DomDistiller recall rescue | Python's Readability comparison and jusText-specific triggers and replacement rules |
| Caller-supplied candidates | `FallbackCandidates` | No equivalent Go candidate contract |

The Go CLI enables fallbacks unless `--no-fallback` is supplied. DomDistiller does not emulate jusText, and Python's jusText replacement ratios are not applied to it. Different engines can select different regions before common cleanup.

The current suite retains eight failing leaf checks covering these output differences:

| Scenario | Checks | Go Output Compared with Python Expectations |
| --- | ---: | --- |
| Forum introduction and replies | 3 | Selects replies without the introduction expected by Python's fallback tests. |
| Blog article with outside comments | 1 | Includes comments outside the article that Python excludes. |
| Saved love-hina page | 2 | Retains unwanted visitor-counter/comment-link content; covered by both imported and native tests. |
| Saved RNZ page | 2 | Misses required article text during recovery; covered by both imported and native tests. |

[test-files/known-differences.json](test-files/known-differences.json) specifies exact test names and assertion counts. Eight failing checks do not mean eight independent defects, and this list is not an exhaustive catalog of differences between fallback engines.

## Metadata and Language

Metadata extraction follows the Python reference for supported fields, including title, author, URL, site name, description, date, categories, tags, license, image, and page type. [metadata_test.go](metadata_test.go) and [metadata-json_test.go](metadata-json_test.go) check source-derived expectations. **Exact equality of all metadata values on arbitrary documents is not established.**

| Boundary | Go | Python Reference |
| --- | --- | --- |
| Author-selector whitespace | Trims IDs and normalizes class whitespace for author selection and author-node discarding | Reads the raw attribute values; padded values such as `class=" username "` need not match |
| Single-word author from meta tags | Retained, subject to author blacklists and JSON-LD overrides | Cleared before JSON-LD and DOM fallback; this can produce a different author or no author. |
| Result representation | A fixed `Metadata` struct, including string zero values and a `time.Time` date | A Python document representation with nullable fields and serialized date strings |
| Metadata availability | Always extracted and returned | Metadata/output switches can control availability |
| Language without a target | Predicted automatically | Unset |
| Date implementation | Go-HtmlDate 1.10.1, Go-DateParser 1.4.7, Go-Dateutil 2.9.1 | Pinned reference uses htmldate 1.10.0 and its Python dependencies |

Author whitespace normalization is limited to the `id` and `class` reads in [author selection](internal/selector/meta-author.go) and [author-node discarding](internal/selector/meta-author-discard.go). It does not rewrite input attributes, add case folding, or normalize `rel`, `itemprop`, or data attributes. Content/comment selectors, JSON-LD, and meta-tag parsing are unchanged by this choice.

The single-word rule applies specifically to the meta-tag stage in [metadata.go](metadata.go) and [Python's metadata extractor](https://github.com/adbar/trafilatura/blob/c1bc9531a2a978326112ca9987e1382745116136/trafilatura/metadata.py#L492). Python can still obtain a single-word author from later JSON-LD or DOM stages. Go intentionally avoids rejecting a meta-tag author merely for having one word.

Different date dependencies can produce different date interpretations. A shared extractor version alone does not establish date parity. Missing-value representations and serialized metadata must be compared explicitly rather than assumed to be byte-identical. Article-text quality scores say nothing about metadata accuracy.

### Language Detection

Go-py3langid 0.4.0 uses the same classifier model as the reference's py3langid 0.4.0. Both classify the longer of extracted body and comments by Unicode code-point count; **equal lengths select comments**. Go's private classifier is initialized lazily and reused, with no Python runtime or model download.

Go retains raw labels without confidence thresholds or remapping: empty/whitespace classifier input yields `af`, and the tested nonlinguistic input `12345 !?` yields `zxx`, matching the Python model. Short, noisy, or multilingual text can be mislabeled; longer comments can determine the label. These are model limitations, not demonstrated Go/Python differences.

A nonempty `TargetLanguage` rejects a mismatching or empty detected label. Without a target, an incorrect label affects metadata only. HTML language-tag checks remain separate; Go initialization or classification errors yield an empty label.

## Parser and Output Boundaries

| Area | Go | Python Reference and Consequence |
| --- | --- | --- |
| HTML parser | `golang.org/x/net/html` | lxml/libxml2; malformed-markup repair and namespaces can produce different input trees and hence different extraction results. |
| Supplied DOM | `ExtractDocument` preserves the caller's DOM and uses Go HTML nodes | lxml nodes and text/tail slots require an adapter for same-tree comparisons; reparsing serialized HTML is not a same-tree guarantee. |
| Result tree | HTML vocabulary and attributes | Python's internal/XML vocabulary differs, such as `ref`, `graphic`, `row`, `cell`, and `hi`. Structural tests map these explicitly. |
| Plain text and serialization | Native Go text rendering and HTML/JSON output | Python serializers have their own whitespace and output-format rules. Normalized text matches do not establish byte-for-byte output equality. |

Parser and serializer boundaries are separate from deliberate extraction changes. They are not a blanket reason to ignore supported behavior that fails comparison. The extracted HTML is also not a security sanitizer; applications rendering untrusted content need their own sanitization policy.

## Unsupported Interfaces

| Python Surface | Go Contract or Omission |
| --- | --- |
| Crawler, downloader, feed/sitemap discovery, acquisition CLI | Supplied HTML is the library input. Existing Go CLI acquisition conveniences are not Python API equivalents. |
| Markdown, XML/TEI, CSV, YAML metadata headers and validators | HTML DOM, plain text, and metadata; CLI HTML, text, and JSON. Python serializer options are not reproduced. |
| `with_metadata=False`, Python `Document`/dictionary wrappers, formatting-off output switches | Metadata is always part of the Go result. Python wrapper and serialization contracts are not provided. |
| Standalone Simhash, fingerprint, and token-sampling APIs | Not exposed as equivalent Go APIs. Per-extraction paragraph/body deduplication is supported. |
| Process-global deduplication caches and reset functions | Deduplication state is extraction-local; cross-document Python cache behavior is not reproduced. |
| General XPath and comment-node pruning | CSS element pruning through `PruneSelector`; normal comment removal is supported. |
| URL-blacklist extraction option | No equivalent option; callers can filter URLs themselves. |
| Python configuration files, deprecated argument aliases, dynamic wrappers, and monkeypatch hooks | Typed Go `Options` and `Config`; no Python-compatible configuration or interpreter API. |

An omitted interface is different from a tested output deviation or an unverified behavior. Unsupported Python output/API tests do not count as proof of extraction parity.

## Verification

The independent [Python reference fixture](test-files/python-2.2.0-reference.json) records the pinned source, runtime, and dependency versions. Its SHA-256 is `5c66c000f1e58dea7a870c3f4d02d41212bca71fff5fea088240ec5c12fab559`. It contains core, selector, metadata-attribute, pruning, and language cases. Approved Go deviations are asserted explicitly in Go tests; the Python fixture is not rewritten to match Go results.

All 60 metadata-attribute cases retain their Python reference values. Six padded-`username` cases additionally assert the intentional Go author result. Fourteen focused author regressions cover ID/class whitespace, unchanged case and other-attribute rules, author-node discarding, and input preservation.

The [source-test coverage inventory](test-files/python-2.2.0-coverage.json) maps imported assertions, native equivalents, and omissions. This is not a literal execution of the entire Python suite. Normal Go tests use frozen expectations and require neither Python nor a reference checkout.

Important comparison adaptations are explicit:

- Empty Go scalar metadata maps to Python `None`, and dates to ISO strings, in the test adapter. List `nil` versus `[]`, casing, keyword splitting, and text whitespace are not generally normalized away.
- Structural tests map the two DOM vocabularies and equivalent formatting tags. This does not validate Python serializer bytes.
- Some inputs are serialized from lxml for the Go adapter; internal-tree cases use tree adapters to isolate parser behavior.
- Native coverage mappings and skipped features are reported separately from passing assertions. Counts are not counts of independent documents or an extraction error rate.

### Current Results

The latest local check of this development snapshot, on Windows with Go 1.27.1 on September 19, 2026, reports **7,363 passing leaves, eight reviewed fallback failures, and 42 skips**, plus 56 native coverage mappings. Build and `go vet` also pass. This does not claim a hosted-CI result for the snapshot.

Run the reviewed-difference gate with:

```sh
python scripts/check_tests.py
```

[scripts/check_tests.py](scripts/check_tests.py) checks the exact failure names and assertion counts in the manifest, and rejects unexpected failures, missing/fixed/skipped known cases, crashes, and incomplete native coverage. Ordinary `go test` still fails on the eight reviewed differences. A passing gate means only that the reviewed expectations hold.

### Historical Measured Speed

The September 19, 2026 Go 2.2.2 development snapshot showed lower mean times on the three measured development corpora against published Go 2.2.1. That historical local comparison used Windows, Go 1.27.1, CGO disabled, the same adapter, and balanced core-only extraction with comments off, tables on, automatic language metadata, and `GOMAXPROCS=1`. It is separate from the released six-engine comparison above.

| Corpus | Pages | Go 2.2.1 Mean ms/page | Go 2.2.2 Mean ms/page |
| --- | ---: | ---: | ---: |
| LegoNews | 983 | 10.699 | 10.230 |
| ScrapingHub | 181 | 9.930 | 9.604 |
| WCXB development | 1,495 | 17.395 | 16.468 |
| All selected pages, page-weighted | 2,659 | 14.411 | 13.695 |

The current code used **4.97% less time overall** (1.0523x throughput) in this experiment. Both processes remained alive through one discarded full-data warmup and four timed passes. Each page ran serially on both versions before advancing; seeded ordering placed each version first exactly twice per page. There were 10,636 timed page observations per version, with stable text and scored metadata across its own repeats. Errors remained in the workloads and were identical in count: 3 LegoNews, 0 ScrapingHub, and 10 WCXB.

These are arithmetic means of request-to-response wall time, including IPC, HTML file reads, parsing, extraction, metadata, and rendering; startup, warmup, controller validation, and scoring are excluded. This is not DOM-only timing or a memory measurement. Each version uses its own locked dependencies, so the result does not isolate the effect of core alignment. Quality outputs differ between releases; faster processing does not imply higher accuracy. Other option profiles, platforms, inputs, and cold-start behavior remain unmeasured.

The measured identities are Go 2.2.1 commit `6e0d351d5f8e8b6809fc5505ed0b7d25e956c575` and Go 2.2.2 development commit `ed2b4c86a5727110178172cb18080efe98fdcdb2` plus worktree snapshot SHA-256 `bdb5762213428d0908feedf3ef54245e9a61604d0d7211077449de79020a2f0b`. The [benchmark repository](https://github.com/markusmobius/content-extractor-benchmark) holds the reproducible JSONL adapter and paired-page runner. Local evidence is retained under `results/go-v2.2.1-vs-current-page-speed-2026-09-19/` and build receipts under `.cache/go-v2.2.1-vs-current-speed-2026-09-19/` in that repository; these ignored artifacts and local runner updates must accompany any published performance claim. Run parameters were `--mode speed --timing page --runs 4 --warmups 1`.

### What Is Not Established

- Exact Python parity over arbitrary input, including malformed HTML and all option combinations.
- Byte-identical plain text, serialized HTML, or metadata for the current snapshot across an entire corpus.
- Equal dates under different native/Python dependency graphs.
- Python fallback equivalence, which is deliberately outside the target.
- Universal speed or quality superiority; the warm-process result above is limited to its measured configuration and workloads.

Known differences above remain differences even when tests explicitly accept them. Passing selected tests is evidence for those cases, not permission to claim end-to-end equivalence.