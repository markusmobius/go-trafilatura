# Upstream Compatibility

This is the detailed companion to the [README](README.md#philosophy-and-scope): what is included, what is intentionally omitted, and where the implementations differ. It also records all 53 commits in `v2.0.0..v2.2.0` of [adbar/trafilatura](https://github.com/adbar/trafilatura).

- Baseline: v2.0.0, `c6e834030779f0fb59aa3888c2f3222101bbdd0f` (December 3, 2024).
- Intermediate release: v2.1.0, `2f4702d2117b0f95fabdd4ea35c9c2a4f3f39d04` (June 7, 2026).
- Target: v2.2.0, `c1bc9531a2a978326112ca9987e1382745116136` (July 31, 2026).
- Post-v2.2.0 commits are not included. Existing Go entry points are retained; subsequent dependency maintenance is recorded below.

For decisions and limitations, see [scope and omissions](#scope), [fallback extractors](#fallback-extractors), [language detection](#language-detection-limitations), and [test/skip accounting](#current-results). The [commit ledger](#commit-ledger) and [historical verification](#verification) retain the implementation audit.

## Scope

The primary contract is extraction from supplied HTML or a parsed DOM, not page acquisition. Main text, comments, cleaning, structural formatting, tables, images, links, metadata/JSON-LD, baseline recovery, and recall escalation are in scope. `OriginalURL` supplies context; the library does not fetch that URL. Regression comparisons use saved or synthetic HTML, not live-site contents.

### Intentional Omissions

| Area | Retained Go Behavior | Omission And Reason |
| --- | --- | --- |
| Page acquisition | Supplied HTML/DOM input; existing CLI URL, batch, feed, and sitemap conveniences. | No upstream crawler, downloader, feed-discovery, sitemap-discovery, or acquisition-CLI parity. The main application already obtains HTML; this update is not a scraper-stack replacement. |
| Serialization | HTML DOM and plain text with metadata; CLI HTML, text, and JSON. | No Python Markdown, XML/TEI, CSV, or YAML metadata headers, schemas, and validators. Structure-preservation fixes are ported without adding those output formats. |
| Metadata/output switches | Metadata is always returned; essential-metadata filtering is supported. | No `with_metadata=False` mode, Python `Document`/dictionary wrapper compatibility, or Markdown formatting-off switch. These do not match the established Go result API. |
| Hashing and deduplication | Optional per-extraction paragraph/body deduplication. | No standalone Simhash, fingerprint, or token-sampling APIs, nor Python's process-global cross-document deduplication cache. Those are separate corpus-processing contracts. |
| Configuration and state | Typed `Options`/`Config`, with extraction-local state. | No Python configuration-file loader, deprecated argument aliases, mutable module-global settings, or cache-reset API. Go callers configure their own calls rather than emulate Python module state. |
| Pruning and URL filtering | CSS `PruneSelector`, normal comment removal, and URL context. | No general XPath/comment-node selector API or URL-blacklist option. CSS element pruning is the supported selector contract; callers can filter URLs before or after extraction. |
| Python-only infrastructure | Go build, tests, and comparison tools. | Python packaging, documentation integrations, CI configuration, monkeypatch hooks, and interpreter-specific repair utilities are not exported as Go APIs. Their supported runtime effects are tested where applicable. |

JSON-LD is not an omission. The current port uses structured decoding, ordered schema/graph handling, and targeted malformed-JSON recovery. Older descriptions of retained article-priority traversal or strict-only decoding no longer describe the implementation. Go HTML and date dependencies can still differ from Python on malformed input, parser repair, or date interpretation; that is not a blanket exemption from fixing supported behavior.

### Interpretation

An omitted API, a deliberate dependency substitution, a detector limitation, and a missing test are different things. The skip ledger below distinguishes them. Accepted fallback differences do not establish that every fallback result is ideal; they mean exact Python fallback output is not the compatibility requirement.

## Fallback Extractors

The choice of Go Readability and `go-domdistiller` is deliberate and predates this update. The current Readability backend is [Readeck Go-Readability v2.1.2](https://codeberg.org/readeck/go-readability), module `codeberg.org/readeck/go-readability/v2`, replacing the deprecated `github.com/go-shiori/go-readability`. Python uses its bundled Readability fork and jusText. We retain the Go candidate cascade rather than tune outputs solely to satisfy Python-reference assertions. Custom `FallbackCandidates` remain supported.

| Setting | Go | Python Reference |
| --- | --- | --- |
| Library default | Fallbacks disabled; opt in with `EnableFallback: true`. | Fallbacks enabled; `fast=True` disables them. |
| Go CLI default | Fallbacks enabled; `--no-fallback` disables them. | CLI behavior is not a parity target. |
| Secondary algorithms | Readeck Go-Readability v2 and Dom Distiller. | Bundled Readability and jusText. |
| Candidate selection | Existing Go ordering, acceptance, and stopping rules, with a Dom Distiller recall-rescue candidate. | Readability comparison plus jusText-specific triggers and replacement rules. |

### Pre-Migration Investigation

Different engines can select different regions before common cleaning runs. The September 10 investigation of the old Go-Shiori backend found a default retry threshold of 500 text characters, with each attempt starting from a fresh DOM copy and progressively relaxing filters. The pinned Python fork is configured with a 250-character serialized-output retry threshold and retries its already-pruned tree. Ancestor scoring and candidate selection also differ. The thresholds measure different representations; simply changing 500 to 250 would not reproduce the Python algorithm. These implementation details describe the backend before the Readeck migration.

That investigation traced seven failing checks to four fallback scenarios before the dependency was replaced:

| Checks | Scenario | Observed Difference |
| ---: | --- | --- |
| 3 | Short forum introduction followed by replies. | Go's main extractor retains the 381-character introduction; Go Readability replaces it with 1,735 characters of replies only. Python retains the introduction, then jusText recovers all eight replies alongside it. |
| 1 | Blog with comments outside an article wrapper. | Go Readability replaces the clean introduction with a region including eight comments. The no-fallback path excludes them. Python's fallback sequence also excludes them. |
| 2 | Visitor counter on the same saved page. | Go's clean main extraction is replaced by a broader Readability region containing the counter. The legacy and imported tests both check this page; Python selects a region without the counter. |
| 1 | Table-exclusion fixture. | Python Readability selects a `tbody`; excluding table content leaves empty output. Go selects a broader region and retains surrounding non-table prose. This is not simply failure to remove tables. |

These were accepted consequences of the backend choice, not seven independently established defects in the port's main extractor. No fixture-specific pruning, candidate stitching, or threshold adjustment was made to close those gaps. For extractor-only comparisons, disable Go fallbacks and use Python's `fast=True`.

### Readeck v2 Migration

The library fallback, chained example, and comparison tool now use Readeck v2.1.2. `FromDocument` still clones the input DOM and returns an `Article` with a public `Node`; the comparison tool uses `RenderText` instead of the removed `TextContent` field and reports rendering errors through its existing error path. Public extraction APIs, fallback ordering and acceptance rules, custom candidates, Dom Distiller, and whatlanggo are unchanged by this migration.

At the dependency-only migration stage, the full suite changed from 879 passing / 15 failing / 98 skipped leaves to **878 / 16 / 98**. No expectations were changed during that migration; the subsequent cleanup is recorded under [current results](#current-results):

- The imported table-exclusion check (`unit_tests.py/test_external/line_841`) began passing: output is empty, matching Python. The legacy `Test_External` expected the opposite and failed at that stage; its stale expectation was subsequently reconciled with the reference.
- The saved RNZ article (`realworld_tests.py/test_extract/line_314`) now fails. Readeck returns no candidate; the main extractor and Dom Distiller also yield no article, so baseline recovery returns 4,030 characters of navigation-heavy text without the required article phrases. With fallbacks disabled, the same baseline result is returned. The old backend recovered the article; this is lost fallback recovery, not replacement of a clean main result.
- The existing love-hina page failure remained, with an additional unwanted comment link. RNZ added failed assertions to the same already-failing `Test_Extract` group, so the increase in failed assertions was larger than the one-leaf increase. The cleanup now reports each saved page as its own subtest.
- The three forum checks and one blog-comment check still fail. Language and legacy spacing results were unchanged by the dependency migration.

Every package compiles with v2.1.2, focused input/DOM and imported external-extraction checks pass, and the comparison command completed on all 960 saved documents using `RenderText`. Historical benchmark tables below were not rerun with identical controls and are not Readeck v2 measurements.

## Commit Ledger

**Ported** means the applicable behavior and focused regressions were incorporated. **Equivalent** means the Go implementation already provides the relevant behavior. **Excluded** means the change has no supported Go runtime counterpart. **Mixed** identifies commits containing more than one category. Earlier table/recovery implementations are represented by their final v2.2.0 behavior, not by retaining intermediate bugs.

| Commit | Disposition | Go Treatment |
| --- | --- | --- |
| [76200b7](https://github.com/adbar/trafilatura/commit/76200b741a485a1700348fb2b73fd292eb3dff70) | Ported | Parent-safe pruning and tail preservation; `Test_pruneUnwantedNodes`. |
| [7067937](https://github.com/adbar/trafilatura/commit/7067937893d25815f9079417cdfd51072307eb8b) | Mixed | Resolve image sources against the supplied URL and preserve tails; table runtime changes included in the final handler. XML serialization excluded. |
| [ad30d66](https://github.com/adbar/trafilatura/commit/ad30d6667d0262ed65a782f1841dc21a52e9844b) | Equivalent | Go HTML parsing does not need the lxml DOCTYPE repair regex; full-document DOCTYPE cases tested. |
| [b010779](https://github.com/adbar/trafilatura/commit/b0107793454f7c6334d87f1228e0ff24d19c02b7) | Equivalent | `Extract` and `ExtractDocument` already return metadata with content. No extra wrapper API. |
| [91c567c](https://github.com/adbar/trafilatura/commit/91c567c3efc4bd8e637c2820abeb72c71edc73d3) | Mixed | Table headers, spans, and structural handling incorporated in the final table port; XML helpers excluded. |
| [42ada5a](https://github.com/adbar/trafilatura/commit/42ada5a515132b3fa7f0b2fbbb8ea5b1e05f4e50) | Excluded | Python documentation, integrations, and documentation dependencies. |
| [051bf5f](https://github.com/adbar/trafilatura/commit/051bf5fa93a2d52a4679c9ab875ff0d691ec40cf) | Excluded | Python README dependents link. |
| [fbdffe3](https://github.com/adbar/trafilatura/commit/fbdffe35b5d8d9725adcdbef5b575e5f8a59d8c3) | Mixed | Code detection in `pre` elements ported and tested; code serializer changes excluded. |
| [139dfd6](https://github.com/adbar/trafilatura/commit/139dfd6e34d50ec082f24207d45112a0953e5de9) | Excluded | Unicode punctuation sampling for standalone Simhash, not paragraph deduplication. |
| [729b737](https://github.com/adbar/trafilatura/commit/729b7370253bf036fedc67b9f7c5dfa9d2c800f5) | Mixed | HTML code/inline structure and spacing covered by existing and extended tests; XML/TEI and Markdown serialization logic excluded. |
| [89581f3](https://github.com/adbar/trafilatura/commit/89581f37ebd5f13372afabe15a7756fd74e16a1f) | Equivalent | Go strings cannot be `None`; final table tests cover empty cells and rows. |
| [3d7e786](https://github.com/adbar/trafilatura/commit/3d7e786a58167eb1d0959f1d6872b48d908b77a9) | Excluded | Punctuation-table initialization for the unsupported token-sampling API. |
| [bbfd5f2](https://github.com/adbar/trafilatura/commit/bbfd5f2cc608818077eaddccebec469f091c6693) | Mixed | Protect image containers from link-density pruning; Python CI changes excluded. |
| [7fd8664](https://github.com/adbar/trafilatura/commit/7fd8664b0463a8b01cd92d05113072c1d28c9ea6) | Excluded | `None` handling in Python's code-block serializer; no nullable Go text equivalent. |
| [badd594](https://github.com/adbar/trafilatura/commit/badd5944a71f3c1826d924394ebad2c4d653b6d3) | Equivalent | One Go `EnableFallback` option is passed through extraction; no deprecated `no_fallback` alias. |
| [5ff6f5c](https://github.com/adbar/trafilatura/commit/5ff6f5cbce378b79570eacab2f0303150eb1d956) | Ported | Pruning retains text in its original parent/sibling position. |
| [ee1865b](https://github.com/adbar/trafilatura/commit/ee1865b22f03e8c52922b3274df621d2f56fe79d) | Ported | Support `meta name="image"` and related name-based image metadata without losing OpenGraph precedence. |
| [20d0429](https://github.com/adbar/trafilatura/commit/20d0429a286424e1e062a472700fb48d433573a6) | Mixed | Typed/default Go options and validated URLs already cover runtime guards. Control-character cleaning retained where needed for recovered text; lxml repair, acquisition, and Python setup excluded. |
| [4e33b47](https://github.com/adbar/trafilatura/commit/4e33b475d84a0cad4613d0b8c3995ba6e7fc0fbe) | Mixed | Keep native Go predicates instead of XSLT regex compilation. Audit predicate semantics and port author-selector priority (`authorname`); final discard changes covered below. |
| [ca1b128](https://github.com/adbar/trafilatura/commit/ca1b1288b1dfd348280745e8c5754272298f86c3) | Ported | Preserve linked images in document order, including multiple images and textual link tails. |
| [4ac91c8](https://github.com/adbar/trafilatura/commit/4ac91c8a458be61d8ab2889d00582128b850220c) | Ported | Discard `fencedframe` in cleaning, baseline, and fallback paths; do not modify the external Go Readability package. |
| [9c43df5](https://github.com/adbar/trafilatura/commit/9c43df5371efbba037be30d71d4064ba84668fd1) | Mixed | Keep fuller author names and safely handle mixed JSON author/publisher shapes. Existing metadata return API retained; feed, Simhash, and Python-only changes excluded. |
| [837766e](https://github.com/adbar/trafilatura/commit/837766e229996cc0d858c1e6eaef2cf41ba71249) | Excluded | jusText language stoplists; Go uses Dom Distiller and its existing language detector. |
| [7f2ba33](https://github.com/adbar/trafilatura/commit/7f2ba33c74cf308cf9d15c43c102609a0342f4c5) | Equivalent | Go result DOMs and extraction state are created per call, not shared mutable lxml defaults. |
| [1a8e59f](https://github.com/adbar/trafilatura/commit/1a8e59f7224a3f0c6d3ee812b47f2b5b355d7999) | Ported | Preserve table-cell links and targets, including links wrapping formatting. |
| [3d8872e](https://github.com/adbar/trafilatura/commit/3d8872eeaa68c87306f9d592ee5b082ecc67b68e) | Mixed | Nested license text and author regressions ported; parsed JSON shapes and copied date options cover equivalent cases. |
| [5715efc](https://github.com/adbar/trafilatura/commit/5715efcd2f491c1d19916955012ee2cb6bbd130a) | Equivalent | Test consistency and utility cleanup; executable function comparison found no additional extraction change. |
| [d6084ba](https://github.com/adbar/trafilatura/commit/d6084bae481cd089fa921d8ae80facfffa579b92) | Excluded | Python typing/modernization, warning stack levels, and cached Simhash vectors. No applicable new extraction behavior. |
| [9b37328](https://github.com/adbar/trafilatura/commit/9b373282ad7d5ee06a2343408ac5ac6d57840aa7) | Excluded | Ruff formatting/docstrings; AST comparison cleared executable changes in extraction modules. |
| [2f4702d](https://github.com/adbar/trafilatura/commit/2f4702d2117b0f95fabdd4ea35c9c2a4f3f39d04) | Equivalent | v2.1.0 release bookkeeping, recorded as the intermediate target. |
| [d698667](https://github.com/adbar/trafilatura/commit/d698667aab32c38b3c255ad90b1a466bbf56053a) | Ported | Better title fallback and first nonempty `h1` selection. |
| [25f8bdb](https://github.com/adbar/trafilatura/commit/25f8bdba1ffc5f01fab6a9814ad0a3c191974574) | Mixed | Author regex corrections, safe table nodes/spans, recovery tag copies, and typed JSON guards. Go defaults retained; Python config-file merging, acquisition, and serializers excluded. |
| [faee866](https://github.com/adbar/trafilatura/commit/faee8668599357888da67327b46ee59de1c6b4ab) | Excluded | Markdown list serialization spacing; HTML inline tails are covered separately. |
| [424363e](https://github.com/adbar/trafilatura/commit/424363ee046dce50c39cffe886dd7ff1e6cc90fe) | Ported | Protect link-only paragraphs inside list items and table cells. |
| [32b4050](https://github.com/adbar/trafilatura/commit/32b40509458e2116b13e97b1a2de1b33c193299a) | Mixed | Carry meaningful inline attributes/children and retain table alignment. Markdown formatting defaults, Python CLI, and serializers excluded. |
| [12f54a9](https://github.com/adbar/trafilatura/commit/12f54a9787795451554832587cc326ca1da69ff0) | Ported | Unwrap `nobr` without discarding its text. |
| [6b87332](https://github.com/adbar/trafilatura/commit/6b873324096825dc26a61f6305cc48d55f112542) | Ported | Convert Yoast FAQ question markers into headings. |
| [9068a97](https://github.com/adbar/trafilatura/commit/9068a9781276134d10f64b5426da4e5cb649e094) | Mixed | Preserve nested inline content, use image-aware extraction stopping, and remove substantial adjacent duplicates; serializer helpers excluded. |
| [1c61445](https://github.com/adbar/trafilatura/commit/1c614458e5bb3637e8e628c2c86ab2ceaee33b58) | Mixed | Decimal-only, bounded span parsing; output-format-specific rendering excluded. |
| [529d0b6](https://github.com/adbar/trafilatura/commit/529d0b6992cb4b4cdd7654a502f3103a2795d1ae) | Mixed | Carry nested links, formatting, deletions, code, and quote paragraphs; distinguish inline code from code blocks. Markdown/math rendering excluded. |
| [2949142](https://github.com/adbar/trafilatura/commit/2949142718f094268ce6c0d0c5ad7dd05e037ed0) | Excluded | Python documentation tests and CI. |
| [598483d](https://github.com/adbar/trafilatura/commit/598483d95b4ab984d695908152136ac77909567a) | Mixed | Row-aware tables, captions, empty cells, combined spans, nested tables, comments, and ARIA layout-table conversion. HTML headers retained instead of XML role encoding. |
| [9f2d49b](https://github.com/adbar/trafilatura/commit/9f2d49b6e61a87c67e90bcf2c27752f28a149213) | Excluded | YAML metadata header quoting for Markdown output. |
| [db7be91](https://github.com/adbar/trafilatura/commit/db7be91b1f2fc48f350ec88239f8b9dea37ab0e9) | Mixed | Pass URL context through fallback conversion and match inline-cell text handling. Go's focus enum avoids contradictory flags; Python CLI/deprecation warnings excluded. |
| [4de7eb7](https://github.com/adbar/trafilatura/commit/4de7eb7f475002373351b5694850451465948a61) | Excluded | Markdown link-destination escaping; HTML serialization remains unchanged. |
| [18a7b42](https://github.com/adbar/trafilatura/commit/18a7b42c1bf363eb336091b6fff59c897048a843) | Mixed | Recovery from a clean backup, bounded deduplication, embedded JSON baseline cascade, publisher guard, comment handling, forum routing, and recall escalation. Go fallbacks retain their own cascade. |
| [9397c17](https://github.com/adbar/trafilatura/commit/9397c171bbd57c9256756c666092e1958ec2e738) | Excluded | Python example for altering global element lists; no new mutable Go configuration API. |
| [3d89493](https://github.com/adbar/trafilatura/commit/3d89493a52cc3bce735090cce9eb08182cf6e99e) | Mixed | Targeted consent cleaning, both-attribute discard tokens, whole-token precision links, `xg1` retention, link-farm tests, and bounded recall decisions. jusText-specific override ratios are not imposed on Dom Distiller. |
| [078f9ac](https://github.com/adbar/trafilatura/commit/078f9ac1db9c4f43862aec994965389d52defa15) | Excluded | Python Ruff rule configuration. |
| [467fdb3](https://github.com/adbar/trafilatura/commit/467fdb3829a869c018834d7b02f790980dda263c) | Ported | Case-insensitive image extensions, including query/fragment suffixes, independent of the machine MIME database. |
| [09a3136](https://github.com/adbar/trafilatura/commit/09a313685d5cfeea29057c182849a27dec081960) | Mixed | Remove empty `sup`/`sub` markers while keeping their tails. Nonempty HTML tags remain; Markdown boundary encoding excluded. |
| [f9ae913](https://github.com/adbar/trafilatura/commit/f9ae91322c80d06580ff69733a175672a556dd3e) | Excluded | Bash selection for Python CI on Windows. |
| [c1bc953](https://github.com/adbar/trafilatura/commit/c1bc9531a2a978326112ca9987e1382745116136) | Equivalent | v2.2.0 release bookkeeping; this is the pinned compatibility target. |

## Regression Coverage

The initial update extended the existing Go tests:

- [html-processing_test.go](html-processing_test.go): pruning tails, cleaning, DOCTYPE parsing, code detection, linked images, fenced frames, link density, and selector boundaries.
- [trafilatura_test.go](trafilatura_test.go): image URLs, nested formatting, table alignment and nesting, Unicode span/deduplication bounds, recovery, fallback guards, comment/forum routing, recall escalation, and caller-DOM preservation.
- [baseline_test.go](baseline_test.go): JSON content shapes, Discourse preload data, embedded HTML versus comparison operators, strategy isolation, multiple articles, consent pruning, and block-aware text measurement.
- [metadata_test.go](metadata_test.go) and [metadata-json_test.go](metadata-json_test.go): titles, image metadata, full author names, nested license text, safe JSON values, and publisher replacement.
- [realworld_test.go](realworld_test.go): retained the existing saved-page suite and updated one obsolete expectation that deliberately allowed boilerplate now removed by both implementations.

## Python Test Synchronization

The subsequent test-only synchronization uses the exact v2.2.0 source commit above. It adds original source assertions alongside the existing tests; it does not change extraction algorithms or relax expected values to make Go pass. This is **not literal identity with the entire Python test suite**: native API/DOM translations and unsupported Python features remain explicitly distinguished.

[scripts/comparison/import-python-tests.py](scripts/comparison/import-python-tests.py) reads the pinned test source with `git show`, records extraction operations and original assertions, and emits four JSON fixtures. The Go runner evaluates those assertions against Go results. Normal Go tests need neither Python nor a reference checkout. The original gzip fixture is also imported byte-for-byte. Existing saved pages are reused; the test synchronization does not download pages.

The current fixtures contain 465 recorded operations and 575 original assertion cases: metadata 208/241, real-world 141/227, extraction 25/27, and structures 91/80. These include the four JSON-normalization assertions added during cleanup; the initial synchronization snapshot below retains its historical counts.

[test-files/python-2.2.0-coverage.json](test-files/python-2.2.0-coverage.json) inventories all 181 test functions in the baseline, unit, metadata, JSON metadata, filters, deduplication, and real-world modules. It records original assertion lines, imported cases, native test locations, and exclusions. A native translation is not evidence of identical Python input representation, API semantics, serializer output, or assertion count. The inventory accounts for scope; it is not a claim that every upstream assertion runs verbatim.

Important adaptations and exclusions:

- Directly recorded assertions retain their expected strings, lists, and comparisons. Inputs that upstream first parses as an lxml tree are serialized for the Go adapter; internal-tree helper cases use an XML-to-Go-node adapter to avoid HTML table repair. Unordered `all`/`any` predicate collections are sorted for reproducibility without changing their checks.
- Go zero-valued scalar metadata becomes Python `None`, and dates become ISO date strings. List `nil` versus `[]`, page-type casing, keyword splitting, and text whitespace are not normalized away. Python `Document` assertions are limited to fields represented by Go metadata.
- XML names such as `ref`, `graphic`, `row`, `cell`, and `hi` map to HTML nodes. Equivalent bold/italic HTML tags are canonicalized in structural comparisons. Markdown table/list/image/link assertions use native DOM translations rather than adding a Markdown renderer; metadata wrappers, serializer syntax, and formatting-off behavior are excluded.
- Python `fast`, extraction sizes, focus, links/images/comments, deduplication, and date options are mapped explicitly. XPath element pruning uses equivalent CSS selectors. Comment-node XPath, URL-blacklist options, dynamic input types, config-file loading, and unsupported metadata/output toggles are documented exclusions.
- Python's private JSON/author entry points use the existing Go metadata/parser helpers. Differences in their accepted schema/context are visible but are not automatically public-API regressions. The Go fallback engines remain Readability and Dom Distiller rather than Python's bundled readability and jusText.
- Acquisition/CLI and XML/TEI test modules, standalone hash/fingerprint/token APIs, process-global cache resets, and Python monkeypatch-only constant changes are excluded. Existing native tests still cover supported per-extraction deduplication and bounded recovery.

Run from the repository root:

```sh
go test -mod=readonly ./... -count=1 -timeout 5m
go test -mod=readonly . -run '^Test_Python220_' -count=1 -timeout 5m
```

### Current Results

The current reviewed baseline uses go-py3langid v0.4.0 and Readeck v2.1.2. Full Windows runs on September 11, 2026, using Go 1.26.0 and Go 1.27.1, compile every package and report **eight failures out of 982 executed leaf checks (0.81%)**. There are 974 passing checks and another 41 skipped checks, excluded from the failure-rate denominator. Parent test groups are not counted again. The 56 native coverage mappings are separate records, not additional pass/skip results. The reviewed-difference checker passes; ordinary `go test` still fails on the eight accepted fallback differences.

| Failing Area | Leaf Checks | Observed Differences |
| --- | ---: | --- |
| Fallback-related checks | 8 | Three forum checks, one blog-comment check, and imported plus legacy checks for each of love-hina and RNZ; see the [fallback breakdown](#fallback-extractors). |

Before the [classifier migration](#py3langid-migration), commit `5ecedf5` with whatlanggo reported 968 passing / 14 failing / 41 skipped checks. Replacing the detector fixes all six language failures. Three Go-only empty/nonlinguistic diagnostic expectations now reflect independently checked Python py3langid outputs; no imported Python assertion or fallback expectation was changed.

Across `Test_Python220_*`, there are 741 passing and six failing leaves: **6 of 747 executed checks fail (0.80%)**, with 41 skipped. All 15 additional classifier cases pass. The remaining Go checks pass 218 and fail two out of 220 (0.91%). These are Go check counts, not counts of unique documents, independent defects, or directly imported Python assertions. Several checks exercise the same input. The initial synchronization snapshot below predates the later fixes and does not describe the current suite.

### Repository Cleanup and CI

Three stale legacy expectations in [trafilatura_test.go](trafilatura_test.go) were corrected from independent reference evidence, without changing extraction: `"1\n3"` for paywall text, the `"1.\n2.\n3."` substring for div/line-break text, and empty output after table exclusion. The complete div output remains `"1.\n2.\n3.\n2.\n3."` in both implementations; even its repeated lines are not a Go/Python difference.

The saved-page suite now has 85 named subtests, preserving its existing assertions: 83 pass and two fail. Replacing its one old leaf with 85 raises the denominator by 84; importing the four original normalization assertions adds another four. Thus the change from 894 to 982 executed checks is reporting granularity and added coverage, not an extraction-quality improvement. Removing 56 bookkeeping skips and one stale normalization exclusion leaves 41 genuine skips. The unused private `schemaInArticle` helper was removed; exported `SchemaData` remains available.

[scripts/check_tests.py](scripts/check_tests.py) runs the complete Go suite and checks [test-files/known-differences.json](test-files/known-differences.json), which lists exact failing leaf names, expected assertion-failure counts, and reasons. New failures, changed counts, unexpectedly passing/skipped/missing known differences, build errors, crashes, and incomplete runs fail the check. It also verifies that every imported native mapping is reported and its target Go tests actually execute. A passing check means only the reviewed differences remain, not that every assertion passes or that all behavior within a failing assertion is identical. The nine checker unit tests cover these failure modes.

[CI](.github/workflows/ci.yml) runs this check on Linux and Windows with Go 1.26.0 and Go 1.27.1, using `GOTOOLCHAIN=local`. It also checks formatting, module tidiness, builds, and `go vet`, and preserves raw Go test events as artifacts. The checker needs only Python's standard library, not Trafilatura or the reference environment. Ordinary `go test` and `make test` still exit unsuccessfully for the eight known differences. `make test` no longer generates source; `make generate` remains explicit, and `GO`, `TEST_TIMEOUT`, and `TEST_ARGS` are configurable.

### Skipped Checks

All 41 skips are explicit entries in the Go test port, not runtime skips caused by missing dependencies:

| Category | Skips | Meaning |
| --- | ---: | --- |
| Excluded serialization/output variants | 24 | Python-specific output syntax, metadata headers, or serializer helpers outside the supported result contract. |
| API, scope, or Python-instrumentation differences | 17 | Unsupported API/state contracts and implementation-specific tests. |
| **Total** | **41** | Excluded from the 982 executed-check denominator. |

The runner logs 56 native coverage mappings separately in [metadata_test.go](metadata_test.go), without creating synthetic passing or skipped tests. Their entries break down as follows:

| Native Coverage | Markers |
| --- | ---: |
| HTML/DOM helpers | 24 |
| Image, link, URL, and license behavior | 16 |
| etree node manipulation | 4 |
| Fallback sanitization | 4 |
| Exotic tags, formatting, and empty links | 7 |
| Saved-page link/emphasis formatting | 1 |

The nine referenced groups are `Test_Python220_HTML`, `Test_Python220_Internals`, `Test_Python220_ImagesAndLinks`, `Test_HtmlProcessing`, `Test_External`, `Test_ExoticTags`, `Test_Formatting`, `Test_Links`, and `Test_Python220_RealWorldFormatting`. All ran and passed in the recorded full suite. Their execution does not claim one-to-one Python assertion identity.

The 24 serialization/output skips consist of 12 Markdown-formatting assertions, three formatted-text saved-page variants, three XML-to-text helper cases, two TEI checks, two XML/Markdown saved-page parameterizations, one YAML-metadata case, and one formatting-off case. Supported links, images, lists, tables, and emphasis are tested as Go DOM behavior rather than serializer syntax.

The remaining 17 entries are:

| Reason | Skips | Why / Coverage Boundary |
| --- | ---: | --- |
| Standalone hashing/token APIs | 4 | Outside the extraction API; paragraph deduplication remains supported. |
| Metadata-off behavior | 3 | Go always returns metadata. |
| Global cache/state APIs | 3 | Go uses extraction-local deduplication and has no Python-style reset or process-global string-cache API. |
| Monkeypatch/instrumentation branches | 3 | Two tests change compile-time deduplication limits; one counts calls to Python's `process_parent`. Default-limit behavior and metadata results are tested separately. |
| JSON minifier helper | 1 | Python's regex minifier has no direct Go counterpart; its exclusion remains explicit. |
| Configuration-file loading | 1 | Go uses `Options`/`Config`; the supported tree-size limit is tested through those options. |
| URL-blacklist option | 1 | No corresponding Go option. |
| Comment-node XPath pruning | 1 | CSS `PruneSelector` selects elements, not comment nodes; ordinary comment removal is tested. |

**Normalization gap closed:** the importer now records all four original `test_normalize_json` assertions and runs them against [normalizeJSONText](metadata-json.go#L53); all pass. The minifier-specific test retains an explicit implementation distinction. General JSON-LD coverage alone is not proof that its exact regression is covered end to end.

Entire acquisition/CLI and XML/TEI modules are excluded by the agreed scope and are not counted among these 41 registered skips. The [coverage inventory](test-files/python-2.2.0-coverage.json) lists those modules and accounts for the 181 source test functions in the seven synchronized modules. It is an inventory of translations and exclusions, not a claim that the whole Python suite runs unchanged in Go.

### Language Detection Limitations

The v2.2.0 update uses [go-py3langid v0.4.0](https://github.com/markusmobius/go-py3langid), aligning the classifier and model with the Python reference's optional py3langid 0.4.0. The previous detector was `github.com/RadhiFadlillah/whatlanggo`, pinned to `v0.0.0-20240916001553-aac1f0f737fc`. whatlanggo and Lingua are absent from the current module graph. The [migration](#py3langid-migration) records current behavior; the earlier detector limitations below are historical comparisons.

Go classifies the longer of the extracted body and comments by Unicode code-point count; equal lengths choose the body. Python's helper chooses comments on a tie. Go also assigns language metadata when no target is requested. Classification happens after extraction and does not control DOM selection, paragraph scoring, or fallback choice.

Before the replacement, whatlanggo produced these results in cases still covered by [trafilatura_test.go](trafilatura_test.go):

- The short French phrase was classified as Afrikaans (`af`) instead of French (`fr`). Its original test does not filter by language, so extraction succeeded with a wrong label.
- "In sleep a king, but waking no such matter." produced an empty ISO language code. The English-targeted extraction test therefore rejected a valid English document.
- The added Italian sentence was labeled Portuguese (`pt`) instead of Italian (`it`). Another Italian fixture was labeled Estonian (`et`). The existing negative English-filter checks passed for those inputs, despite the incorrect labels.

An isolated audit of the whatlanggo baseline's tests recorded 13 decisions at the statistical language-filter check: three correct acceptances, nine correct rejections, and one false rejection. The one-in-13 failure rate describes those small, repeated test inputs only. Most saved-page extraction assertions do not verify the predicted language and cannot establish detector accuracy.

No confidence threshold, language whitelist, or phrase-specific exception is added. The existing policy is preserved: a nonempty `TargetLanguage` rejects a mismatching or empty detected label. Without a target, an incorrect prediction affects language metadata rather than discarding the content. HTML language-tag checks remain separate. Passing a negative English-filter test does not demonstrate an accurate label: Italian mislabeled as Portuguese still correctly fails an English-only filter.

Lingua v1.4.0 was evaluated and removed for resource cost. It fixed some short-text labels but also misclassified a repeated Italian fixture as English. It was not an unqualified accuracy improvement on these cases.

| Warmed End-To-End Sample | whatlanggo | Lingua | Added Time |
| --- | ---: | ---: | ---: |
| 16 saved pages, default options | 12.13 ms/document | 36.83 ms/document | +204% |
| Same pages, fallbacks enabled | 17.39 ms/document | 46.66 ms/document | +168% |

These are averages of two interleaved warmed runs on 16 evenly spaced saved pages, on Windows with Go 1.27.1. Body/comment hashes matched between builds. File reads and model initialization are excluded; this is not a universal workload or accuracy estimate. A separate first French classification took about 2.94 seconds and retained an additional 732 MiB of Go heap after garbage collection. That is model-cache growth, not per-document memory. Lingua and its trial-only dependencies are absent from the production module graph; independently required shared dependencies are retained.

### Py3langid Migration

The v2.2.0 update replaces whatlanggo with [go-py3langid v0.4.0](https://github.com/markusmobius/go-py3langid), release commit `d3e0c0861455d7d84daedb994392d2e71a0f6270`. It embeds the py3langid 0.4.0 model used by the Python reference, aligning with [upstream adbar/trafilatura's optional classifier](https://github.com/adbar/trafilatura/blob/c1bc9531a2a978326112ca9987e1382745116136/pyproject.toml).

The private `languageClassifier` wrapper loads one independent identifier through `sync.OnceValues` and reuses it concurrently. It uses raw scores, without a confidence threshold, language restriction, or phrase-specific exception. A private instance avoids interference from another package calling `py3langid.SetLanguages`. Initialization/classification errors return an empty label through the existing string-only interface. Public extraction APIs, body/comment selection, HTML-language checks, fallback behavior, and target-language filtering are unchanged. No Python, NumPy, CGO, or runtime model download is needed by the detector.

All six whatlanggo-related failures now pass, including the English false rejection, French labels, and Italian sentence. The migration also adopts three raw model labels in place of the old Go-only empty-label expectations:

| Input | Previous Go Expectation | Current Go py3langid | Python py3langid 0.4.0 |
| --- | --- | --- | --- |
| Empty string | Empty label | `af` | `af` |
| Whitespace only | Empty label | `af` | `af` |
| `12345 !?` | Empty label | `zxx` | `zxx` |

These outputs were checked directly against the installed Python reference. `zxx` is the model's non-linguistic label, not an abstention threshold. The replacement retains the raw Python labels without a guard or remapping. The three Go-only diagnostics now assert these independently established results; no imported Python assertion was changed. The six resolved language entries were removed from the known-difference manifest, while all eight fallback entries and their assertion counts remain unchanged.

The current configuration passes the reviewed-difference checker on both Go versions: **974 passing / 8 failing / 41 skipped leaves**, plus 56 native mappings. All 15 classifier diagnostics pass. The imported Python subset has 741 passing / 6 failing / 41 skipped leaves; those six failures are fallback differences. These curated checks are not a representative language-accuracy estimate.

The September 11 performance comparison rebuilt the committed whatlanggo wrapper and matching module files against the same current extraction code, with only the detector swapped. It is not a whole-release v2.0.0-versus-v2.2.0 benchmark. Measurements used Windows amd64, AMD Ryzen AI 7 PRO 350, Go 1.27.1, and one worker. One warm-up round was discarded; the table gives medians of three further alternating A/B rounds at 750 ms per benchmark. The 16-page sample selects evenly spaced files from the existing 960-page corpus; it is not a random production sample.

| Warm Sample | whatlanggo | Go py3langid | Time Change |
| --- | ---: | ---: | ---: |
| 16 saved pages, default options | 10.60 ms/document | 9.02 ms/document | -14.9% |
| Same pages, fallbacks enabled | 13.29 ms/document | 12.01 ms/document | -9.6% |
| Single saved article, default options | 8.95 ms/document | 7.63 ms/document | -14.8% |
| Same article, fallbacks enabled | 11.14 ms/document | 9.03 ms/document | -18.9% |
| Classifier only, short French phrase | 121.59 microseconds | 1.33 microseconds | -98.9% |
| Classifier only, extracted saved article | 1.146 ms | 0.358 ms | -68.8% |

End-to-end runs include HTML parsing, automatic encoding detection, extraction, metadata, and language classification; file loading and model initialization are outside the timed loop. All 16 documents were accepted, and body/comment hashes matched between detectors in every measured round for each option set. Startup was measured separately in fresh processes: median first classification 185.7 ms, about 92.8 MiB retained heap after garbage collection, and 186.2 MiB total allocation during initialization. Retained model memory is a one-time process cost, not per-document allocation; concurrent working buffers and other application memory are additional. These sample results do not establish a universal throughput improvement.

All packages compile, `go vet` and module tidiness pass, and all 15 parallel classifier cases pass under the race detector without a race. Temporary A/B binaries, overlays, and measurement helpers remain outside the repository. The current compatibility baseline contains only the eight accepted fallback differences.

### Initial Synchronization Results

At the initial test-only synchronization on September 10, 2026, the full Go 1.27.1 run compiled every package and passed every then-existing test, but returned a failure for the new compatibility suite. The 571 directly imported assertion cases produced:

| Imported Fixture | Passed | Failed |
| --- | ---: | ---: |
| Metadata | 188 | 49 |
| Extraction | 27 | 0 |
| Mixed structural/unit cases | 74 | 6 |
| Saved-page content and metadata | 222 | 5 |
| **Total** | **511** | **60** |

Native translations add 13 failing leaf test cases. Across all `Test_Python220_*` groups, there are 670 passing, 73 failing, and 98 skipped leaves. Those totals include operation-only smoke checks and source-line markers pointing to separately executed native tests; they are not counts of distinct upstream assertions or independent defects.

| Failing Area | Leaf Cases | Observed Differences |
| --- | ---: | --- |
| Imported metadata | 49 | Page-type casing, schema/graph precedence, malformed JSON and author normalization, canonical URLs, image values, and keyword/list representation. |
| Imported mixed unit cases | 6 | Five plain-text spacing/line-break expectations and one fallback case that retains content Python discards. |
| Imported saved pages | 5 | Two content assertions and three tag-list assertions, including empty-list representation and comma-separated keywords. |
| Native baseline | 3 | Raw control characters in JSON recovery and plain-text/Atom `html2txt` behavior. |
| Native internals | 2 | Header-cell normalization and flattening text nested in inline elements. |
| Native inputs/options | 3 | A nil reader panics, gzip is not decompressed automatically, and short French text is classified differently. |
| Native filters | 1 | The original short Shakespeare sentence is rejected as non-English. |
| Native recovery | 4 | Forum introductions are dropped in three cases; a wrapper-less blog retains replies in another. |

The remaining imported extraction cases and native cleaning/selectors, tables, image/link, structure, and deduplication groups pass. Some failed checks intentionally expose existing API or dependency differences; they should not all be labeled regressions from this update. No production fixes were made as part of this test synchronization.

The Python control ran the seven original modules with saved local fixtures: **239 passed, 1 skipped**. The skip was the optional PyYAML Markdown-metadata test. Language detection was enabled. The reference used Python 3.12.13, Trafilatura 2.2.0, lxml 6.1.3, htmldate 1.10.0, courlan 1.4.0, jusText 3.0.2, py3langid 0.4.0, NumPy 2.5.2, and pytest 9.1.1. Python test-function counts are not directly comparable with Go assertion/subtest counts.

`go vet -mod=readonly ./...`, Go formatting, editor diagnostics, and the importer's read-only `--check` pass. The new suite has not been rerun with the race detector or Go 1.26.8; the historical runs below predate these test additions. Regeneration instructions are in the [comparison tooling README](scripts/comparison/README.md#python-test-fixtures).

The 960-page evaluation inventory is unchanged between upstream 2.0.0 and 2.2.0. One Go phrase annotation was corrected to include the upstream trailing comma. The historical scores below were not recomputed after that correction.

## Verification

This section records historical verification **before** the source-synchronized tests above were added. Its passing runs do not describe the current red compatibility suite.

Initial parity verification used Windows, Go 1.24.2, and a separate Python 3.12 environment with Trafilatura 2.2.0 installed from the pinned source. Python dependencies included lxml 6.1.3 and htmldate 1.10.0; the Go date dependencies were `go-htmldate` v1.9.3 and `go-dateparser` v1.2.4. Neither the reference checkout nor temporary comparison tooling is part of this repository.

Subsequent toolchain maintenance raised the minimum Go version to 1.26.0 and selected Go 1.27.1 for development. The then-existing package suite passed on Go 1.26.8 and Go 1.27.1, and the race-enabled suite passed on Go 1.27.1. That toolchain-only update retained dependency versions; the later date-module update is documented below. The corpus scores and timing observations below remain from the initial Go 1.24.2 measurements.

- All then-existing Go package and saved-page tests passed with `go test ./...`.
- 22 representative comparisons matched after normalizing the two DOM vocabularies and whitespace: body/comments, link targets, image sources, headings, and table cells. They cover cleaning, inline nesting, code, images, tables, recovery, comments/forums, JSON rescue, and recall escalation, with fallbacks disabled.
- On the existing 960-page corpus, 838 bodies matched Python after removing whitespace differences. The remaining 122 are not claimed to have identical output.
- The isolated corpus comparison below uses balanced mode, no fallbacks, comments excluded, tables included, and the same saved HTML/annotations. Failed extraction is counted as empty text, not omitted. The updated Go run and Python both reject one page under these options.

| Extractor | Precision | Recall | F1 |
| --- | ---: | ---: | ---: |
| Untouched Go baseline | 0.9123 | 0.8966 | 0.9044 |
| Updated Go port | 0.9171 | 0.9113 | 0.9142 |
| Pinned Python 2.2.0 | 0.9155 | 0.9082 | 0.9118 |

These are phrase-level annotation scores, not proof of universal superiority or exact parity. The Python comparison normalizes inter-element whitespace; Go is scored using `ContentText`. The repository's Go-only comparison tool uses pre-parsed DOMs and excludes failed extractions from its score totals. Its procedure differs from the isolated comparison above, which scores failures as empty text; the results are not interchangeable.

Timing was measured with interleaved baseline/updated runs after a warm-up. Unchanged Readability and Dom Distiller runs also slowed materially during the later measurements, so no reliable speedup or slowdown is claimed. Historical README timing tables are not relabeled as v2.2.0 results.

Regex regeneration and the race-enabled suite now pass using MSYS2 re2go 4.4 and UCRT64 GCC 16.2.0 with `CGO_ENABLED=1`. The `.re` rules are unchanged; regenerated Go files contain updated generator headers, a constant declaration, and equivalent end-of-input capture bookkeeping. The generator reports `-Wmatch-empty-string` on the intentional `$` end-of-input return rule.

The first `make test` run exceeded its 30-second timeout while opening a saved-page fixture. A full uncached rerun with a longer timeout passed, followed by a successful rerun of the unchanged `make test` target. The full race-enabled suite also passed without reported races:

```sh
make test
go test -race -timeout 5m ./... -count=1
```

The existing local corpus benchmark remains available without temporary tooling:

```sh
go run ./scripts/comparison content -j 1
```

### Date Dependency Update

The current module graph uses `go-htmldate` v1.10.0 and its indirect dependency `go-dateparser` v1.4.3. Their module requirements also raise shared versions, including Cascadia, `golang.org/x/net`, `golang.org/x/text`, the regex runtime, and CLI/test utilities. No extraction API changes were needed, and the minimum Go version remains 1.26.0.

Before the test synchronization, the updated graph passed the full package suite on Go 1.26.8 and Go 1.27.1, including the existing metadata/date tests, and the race-enabled suite on Go 1.27.1. A separate comparison used identical source and Go 1.27.1 with the pre-update and post-update dependency graphs. All 960 saved pages were processed in both fast and extensive date-search modes, without extraction fallbacks and with comments excluded.

Across the initial 1,920 dependency-comparison results, body text, HTML, extracted dates, and errors were unchanged. Each mode retained one empty-result error. Two non-date metadata differences were observed in both modes: an author list changed order on the Haufe fixture, and the WordPress description's `&#8` numeric entity decoded to U+0008 (backspace) instead of remaining literal text. Repeated extraction subsequently reproduced both author orders with both dependency graphs, identifying an existing nondeterministic fallback rather than a dependency regression. The entity change came from the updated HTML decoder; Python removes the invalid reference entirely. The following fixes address these parity defects.

### Metadata Parity Fixes

Scalar meta-tag values now remove invalid references and non-printing characters during entity cleanup, matching Python 2.2.0 without changing list-valued tags. JSON-LD publisher, employee, and founder subtrees no longer supply fallback authors. Explicit nested author names and the existing fallback behavior elsewhere are retained. The [description tests](metadata_test.go) and [author tests](metadata-json_test.go) include expectations checked against pinned Python 2.2.0 for both reported fixtures and neighboring cases.

A final before/after comparison used the same updated dependencies and all 960 saved pages in both fast and extensive date modes. In each mode, 956 results were completely unchanged. Four pages changed, covering one author, one title, and three descriptions; every changed field exactly matched Python 2.2.0. Body text, HTML, comments, dates, and errors were unchanged across all 1,920 results. This verifies the changed fields, not universal metadata parity or a new timing result.

At that stage, the implementation passed `make test`, the race-enabled suite and `go vet ./...` on Go 1.27.1, and the full package suite on Go 1.26.8. The later source-synchronized tests expose additional differences, as recorded above.

## Compatibility Boundaries

The Go fallback sequence intentionally does not copy jusText's sanitized-output trigger and replacement ratio. Testing that substitution with Dom Distiller reintroduced gallery boilerplate in an existing real-world fixture. The established Go candidate ordering/stopping rule is retained, while applicable empty/raw-JSON guards, recall preference, URL conversion, and a bounded Dom Distiller escalation candidate are included. The [fallback breakdown](#fallback-extractors) records the accepted output differences.

JSON-LD metadata traversal now follows source-guided ordered schema/graph handling and includes targeted regex recovery for malformed metadata. The shared decoder also tolerates raw control characters inside quoted JSON strings before retrying structural decoding. Baseline recovery supports the new schema properties and Discourse preloads. This is not a general promise to repair arbitrary invalid JSON, but strict-only parsing and retained article-priority traversal are no longer intentional differences.

The returned HTML DOM is not an untrusted-HTML security sanitizer. As before, applications rendering extracted content must apply their own security policy.