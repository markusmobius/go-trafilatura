# Changelog

Changes by Go release. For current behavior and usage, see [README.md](README.md); for detailed Python compatibility decisions and evidence, see [UPSTREAM.md](UPSTREAM.md).

### Documentation - 2026-09-23

- Refresh README quality and six-engine speed comparisons from the published
	[benchmark JSON](https://github.com/markusmobius/content-extractor-benchmark/blob/d433ab637f0a56c0926aa3698f470794a553472f/go_rust_shared_performance_2026_09_23.json),
	with separate metadata scores and exact provenance in [UPSTREAM.md](UPSTREAM.md).
- Trafilatura text F1 is 90.88412% / 96.15663% / 78.49352% on LegoNews /
	ScrapingHub / WCXB. Selected Go/Rust extraction is 6.815 / 4.000 ms/page
	(1.70x); all-four means are 6.839 / 4.026 ms/page. Shared parsing is separate.
- Retain the two observed Go/Rust metadata differences. Go-Trafilatura remains
	2.2.2; no source, module, dependency, fixture or tag changes are introduced.

### v2.2.2

This section compares released **v2.2.2** with **v2.2.1**. Both target [Python Trafilatura 2.2.0, commit c1bc9531](https://github.com/adbar/trafilatura/commit/c1bc9531a2a978326112ca9987e1382745116136): these are corrections to the Go port, not a change of Python target version. Python is authoritative for non-fallback extraction, subject to the deliberate Go exceptions below. Exact end-to-end parity is not claimed.

#### Python-Alignment Checkpoint

The intermediate [Go commit ed2b4c86](https://github.com/markusmobius/go-trafilatura/commit/ed2b4c86a5727110178172cb18080efe98fdcdb2), `Align non-fallback extraction with Python 2.2.0`, records the upstream-aligned stage before complete cleaning, final-body measurement, automatic language metadata, and author-selector whitespace normalization were restored. It is the reference for reviewing those retained Go choices, not the final v2.2.2 source.

This was the "pure Python alignment" stage of the **Go core**, not literal Python code, a runtime bridge, or replacement of native Go fallbacks. Go API, parser, and output boundaries still applied.

#### Expanded Quality Benchmark

Use the expanded [Content Extractor Benchmark methodology](https://github.com/markusmobius/content-extractor-benchmark#change-evaluation-methodology), which separates four evaluations:

| Evaluation | Inputs | Score |
| --- | --- | --- |
| LegoNews | The original 983 saved pages | Case-sensitive wanted/unwanted snippet micro F1 |
| ScrapingHub | 181 article pages | Four-word shingle F1 computed from mean page precision and recall |
| WCXB | 1,495 distinct development pages; 372 test pages after known-overlap removal | Mean page word-multiset F1, also grouped by page type |
| Metadata | Nonempty supplied author, title, and date annotations on the selected inputs | Exact author-set match and author-unit micro F1; exact title/date match |

Inputs, annotations, source revisions, adapters, options, dependency locks, and build identities are recorded. Each experimental variant changes one behavior in an isolated copy of the same frozen worktree. Core-only, comments-enabled, and native-fallback profiles are separate; failures remain in the scores. Compare each corpus independently, inspect page-level losses and domain/template concentration, and report uncertainty rather than merging the three text F1s.

The [September 19 recommendation study](https://github.com/markusmobius/content-extractor-benchmark/blob/master/RECOMMENDATION_EVALUATION.md) contains 27 independently changed variants and 34 complete evaluations, including an unchanged repeat and one predeclared held-out comparison. Its paired page-bootstrap intervals are exploratory, not adjusted for multiple comparisons or domain clustering. The runner's test profile repeats LegoNews and ScrapingHub as controls; only its 372 WCXB pages are held out. That subset has now been used for the author-normalization comparison and must not be relabeled unseen during further tuning.

The expanded methodology and study are published in that repository. The study evaluates quality against annotations, not Go-versus-Python extraction equality, a complete v2.2.1-versus-v2.2.2 comparison, or execution speed. Its baseline includes the recorded development worktree, not just the intermediate commit. The bundled historical v1 adapter is not the current v2 extractor.

#### v2.2.1 Behaviors Retained

These are existing Go behaviors being kept, **not newly introduced v2.2.2 features**.

| Old Go behavior retained | Why it remains |
| --- | --- |
| Remove every matched unwanted cleaning node, using collected matches rather than live deletion | Python's live-removal walk can skip later nodes after entering a detached subtree. Retaining complete removal avoids leftover boilerplate. Independent removal of this safeguard worsens LegoNews and WCXB without page-F1 gains. Python's cleaning tag list/order is still adopted. |
| Measure the final body after duplicate removal, `done` removal, and `div` unwrapping | Deleted text must not inflate recovery thresholds or block embedded-content recovery. Independent removal worsens LegoNews and WCXB without page-F1 gains. Threshold constants are unchanged. |
| Populate `Metadata.Language` without requiring `TargetLanguage` | Preserve Go's automatic metadata feature. Only an explicit target enables language-based rejection. This is an API choice, not a language-accuracy gain established by the text benchmark. |
| Normalize ID/class whitespace for author selection and author-node discarding | Restore v2.2.1's trimming/normalization instead of Python's raw-attribute comparisons. The measured variant adds 17 exact development author matches (678 to 695 of 1,290), loses none, and leaves body text, titles, dates, and language unchanged. This does not broaden content selectors, case matching, JSON-LD, or meta-tag author parsing. |
| Retain single-word authors obtained from meta tags | Do not reject a valid one-word name or pseudonym solely for lacking spaces. Existing blacklists and JSON-LD overrides still apply. Python clears these meta-tag authors before later sources; this is not a difference for every single-word author. The new study did not test this policy. |
| Keep native Go fallback selection and recovery | Preserve Go candidate order, acceptance, sanitization, lazy stopping, custom candidates, and bounded recall rescue. Go-ReadabilityV2 and DomDistiller remain the engines; Python's bundled Readability/jusText behavior is not ported. Library fallbacks remain opt-in, while the CLI enables them by default. |
| Scope deduplication state to one extraction | Preserve independent Go calls rather than introducing Python's process-global deduplication cache. This remains a contract difference, not a benchmark-driven change. |
| Keep the native Go API and always-returned metadata | Preserve typed options, `Metadata` and date values, supplied-HTML entry points, and caller-owned DOM/candidate preservation. Python wrapper switches, missing-value representation, general XPath, and acquisition/output APIs are not substituted for the existing Go contract. |
| Keep Go HTML parsing, result trees, and serialization | Continue using the native parser and HTML vocabulary instead of lxml and Python's internal/XML vocabulary. Adapt text/tail handling explicitly, without claiming identical malformed-HTML repair or serializer bytes. These boundaries are not permission to ignore core extraction defects. |

The full current contract, unsupported Python interfaces, and known differences remain in [UPSTREAM.md](UPSTREAM.md). Native fallbacks, API choices, and parser boundaries are distinct from the two measured cleaning/recovery safeguards.

The implemented author-normalization patch matches the study's isolated M3 variant. Its 17 author gains and unchanged body text also hold with native fallbacks enabled. All 13 changed WCXB test authors lack labels, so held-out author improvement and false-author rates remain unproven. The production integration adds regression coverage but does not relabel or rerun the frozen benchmark results.

#### v2.2.1 Behaviors Replaced by Python Rules

Where no deliberate exception is retained, use Python's non-fallback semantics. This is a compatibility decision, **not a claim that every individual change increases F1**.

| Area | Old Go behavior replaced | Current Python-aligned behavior and reason |
| --- | --- | --- |
| Content-selector whitespace | Trim IDs and collapse class whitespace | Inspect raw attributes. Restoring normalization produces only small, mixed development gains, insufficient to adopt another exception now. |
| Content-selector case | Apply broad case-insensitive matching | Match Python's specified variants and limited character translations. The old rule's measured benefit is too small and narrow to justify restoring it now. |
| Article class prefix | Match any class beginning with `article` | Require `article ` in this predicate. Broad matching helps LegoNews/ScrapingHub but causes larger WCXB losses by selecting unrelated regions; other article selectors remain active. |
| Core discard predicates | Use broader normalized or concatenated attribute checks | Match raw attributes, exact case rules, and source-first attribute unions where Python uses them. Keep the old fallback-specific predicates separate so core alignment does not change native fallback sanitation. |
| Comment selection, removal, and discarding | Normalize/concatenate attributes, accept broader case variants, and use the incorrect `dsq_comments` prefix | Follow raw/source-first attribute rules, the specified `comment`/`Comment` variants, and `dsq-comments`. Map Go quote-like tags to Python's internal quote rule. Comment-output quality is not established without comment labels. |
| Image-caption and teaser selectors | Normalize attributes and fully case-fold teaser checks | Read raw attributes and translate only Python's specified teaser character. The literal `caption` substring check is behavior-equivalent under the old whitespace normalization; it is not an independent quality fix. |
| Forum routing and salvage | Recursively parse/normalize forum schema types and remeasure whenever captured posts exist | Use Python's exact direct-script-text type pattern, including its malformed-JSON behavior, and update the decision snapshot only when missing posts are actually appended. No development-score benefit was measured from restoring structured forum detection. |
| Language-length ties | Choose body text when body and comments have equal character counts | Choose comments, matching Python. Automatic language metadata remains enabled as the separate retained Go choice above. |
| Cleaning rules | Omit `noindex` from this cleaning path and use Go's tag ordering | Use Python's tag order and `noindex`, followed by extra Go tags, while retaining complete collected removal. |
| Empty-node pruning | Also delete parents made empty by earlier deletions | Remove nodes that were empty when selected, matching Python's XPath snapshot. Restoring cascading removal remains under review below. |
| Link preparation | Protect links under `div`, `ul`, `ol`, `dl`, and `p` | Use Python's `div`, `li`, and `p` ancestor query, with conditional table handling unchanged. |
| Structural/code conversion | Retain formatting/heading attributes, omit core details/summary conversion, and apply broader code heuristics | Clear those attributes, convert details/summary into the internal container/heading representation, limit code heuristics to `pre`, and require the `hljs` class prefix. |
| Text and tail slots | Treat absent text and explicitly empty text as the same string | Preserve the distinction throughout preparation and body/comment handlers. Represent a text-bearing `wbr` as a Go-compatible span rather than losing its text. |
| Extraction traversal and unwrapping | Walk pre-collected node slices and clone children while stripping wrappers | Observe live mutations and preserve node identity when unwrapping. Snapshot traversal slightly worsens both new corpora; complete unwanted-node cleaning deliberately remains collected rather than live. |
| Heading/code copies and nested handlers | Lose clone-tail operations or reconstruct nested/list/quote content using string-only slots and stale walks | Retain heading/code tails, deep-copy headings before processing their original children, and use slot-aware live nested/list/quote handling. Small mixed corpus effects do not justify restoring text-loss behavior. |
| Paragraph, list, table, and image details | Over-rescue rejected inline wrappers, over-apply child carrying, trim some list/cell text early, and collapse absent/empty tails | Follow Python's handler decisions, preserve leading list and complex-cell whitespace, and retain break/image tail-slot distinctions. These are structural corrections, not a new table-layout algorithm. |
| Decision, caption, and comment text | Collapse internal whitespace through depth-based text rendering | Join DOM text slots with spaces and trim only the outside, matching Python's extraction text. Public Go serializers are not replaced wholesale. |
| Paragraph-length ownership | Count paragraphs in the original document even when pruning returns a detached backup | Measure from the selected/pruned tree's owner. The measurement is Python-aligned, but its downstream extraction decisions have mixed quality results; see the unresolved item below. |
| Boilerplate text filter | Reject substring matches inside otherwise legitimate text | Match whole boilerplate lines with Python-compatible separators and case behavior. Legitimate paragraphs ending in `.pdf`, for example, survive. Keeping this correction improves LegoNews and WCXB. |

#### Open Quality Questions

Author-selector whitespace normalization is implemented as the retained behavior above. The following two experimental changes remain unapplied.

- **Investigate old cascading empty-parent pruning.** It improves one ScrapingHub page and five WCXB pages without measured page-F1 losses, but the aggregate gain is small and concentrated. Keep the current Python snapshot rule while reviewing this separately from complete unwanted-node removal.
- **Investigate the decision around paragraph-length ownership.** Returning to the original-document measurement helps WCXB by 0.364 F1 percentage points but hurts LegoNews and ScrapingHub. Much of the WCXB effect comes from eight Steam product pages. Keep the current measurement temporarily; neither an unconditional quality-win claim nor a blanket rollback is justified. Any correction must be general, not a site-specific exception.

#### Dependencies, Implementation, and Verification

- Replace Readeck Readability v2.1.2 with Go-ReadabilityV2 v0.6.0. This is a native Go backend update, separate from Python core alignment; the fallback pipeline is retained, not asserted byte-identical to the previous backend.
- Upgrade Go-HtmlDate 1.10.0 to 1.10.1 and Go-DateParser 1.4.3 to 1.4.7; include Go-Dateutil v2.9.1. These include date-extraction/parsing corrections, not just dependency bookkeeping. The rule experiments hold this graph fixed and do not measure the dependency upgrade's independent effect or prove date equality with Python's different pinned graph.
- Separate core cleaning/conversion from fallback preparation. Avoid cloning fallback input unless precision pruning or fenced frames require it; preserve caller input and existing fallback behavior. This is not a newly measured performance claim.
- Add independent Python-reference checks and general cleaning, mixed-content, tail, and recovery regressions. The 27 complete-cleaning, nine final-text, and six duplicate-recovery cases exercise the retained safeguards without rewriting Python expectations to match Go.
- Add 14 author-normalization regressions and explicit Go expectations for six whitespace-padded username cases, leaving the independent Python fixture unchanged.
- The current development snapshot passes the Windows Go 1.27.1 reviewed-difference gate with 7,363 passing leaves, eight reviewed fallback failures, 42 skips, and 56 native mappings. Build and `go vet` pass. Ordinary `go test` still fails those eight reviewed checks; the gate validates their identities. The expanded benchmark's 22 metric/preparation/provenance tests also pass. These are scoped checks, not proof of all-input parity, a fresh hosted-CI result, or universal quality/speed superiority.

### v2.2.1

- Publish the library and CLI with the Go module path `github.com/markusmobius/go-trafilatura/v2`, using the same path in examples, tools, and tests.
- Simplify installation and migration instructions. Existing v1 tags and commit-based pins remain unchanged.
- Retain the v2.2.0 extraction logic, dependency versions, and exported function signatures.

### v2.2.0

- Raise the minimum Go version to 1.26.0 and the preferred development toolchain to Go 1.27.1.
- Update `go-htmldate` to v1.10.0 and the indirect `go-dateparser` dependency to v1.4.3, along with the shared dependency versions required by those releases.
- Replace deprecated Go-Shiori Readability with `codeberg.org/readeck/go-readability/v2` v2.1.2 in the fallback, chained example, and comparison tool. Retain public APIs and fallback-selection rules.
- Add opt-in `Options.InputEncoding` to bypass charset detection for supplied HTML with a known encoding, retaining normalization and leaving automatic detection unchanged by default.
- Replace whatlanggo with `github.com/markusmobius/go-py3langid` v0.4.0, aligning language identification with upstream Python's py3langid classifier and model. Reuse one private lazy identifier, retain public APIs and filtering rules, and adopt Python's raw edge-case labels.
- Track supplied-HTML extraction changes through upstream Trafilatura v2.2.0.
- Preserve pruning tails, nested inline formatting, relative and linked images, table captions, empty cells, and bounded row/column spans.
- Improve metadata image, title, author, license, and JSON-LD publisher handling.
- Match Python 2.2.0 cleanup of invalid character references and non-printing characters in scalar meta-tag values, without changing tag lists.
- Exclude JSON-LD publisher staff from author fallback while preserving explicit nested author names and existing fallback behavior elsewhere.
- Update recovery deduplication, embedded-content baseline recovery, targeted boilerplate filtering, comment exclusion, forum-post routing, and bounded recall escalation.
- Keep caller-owned DOMs and supplied fallback candidates unchanged during extraction.
- Document the supplied-HTML philosophy and intentional differences. Existing CLI download, feed, and sitemap features are unchanged; crawler and Python-only API/output parity remain outside scope.
- Reconcile three stale legacy assertions with pinned Python behavior and report each of the 85 saved pages as a named subtest without changing its assertions.
- Import the four original JSON-normalization assertions and report 56 native coverage mappings separately from genuine skipped tests.
- Separate `make test` from source generation, with configurable Go command, timeout, and test flags.
- Remove the unused private `schemaInArticle` helper while preserving exported `SchemaData`.
- Add Linux/Windows CI on Go 1.26.0 and Go 1.27.1 with a reviewed known-difference manifest; raw Go test failures remain visible and new differences fail CI.

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