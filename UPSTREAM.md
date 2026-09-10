# Upstream Compatibility

This record covers all 53 commits in `v2.0.0..v2.2.0` of [adbar/trafilatura](https://github.com/adbar/trafilatura).

- Baseline: v2.0.0, `c6e834030779f0fb59aa3888c2f3222101bbdd0f` (December 3, 2024).
- Intermediate release: v2.1.0, `2f4702d2117b0f95fabdd4ea35c9c2a4f3f39d04` (June 7, 2026).
- Target: v2.2.0, `c1bc9531a2a978326112ca9987e1382745116136` (July 31, 2026).
- Post-v2.2.0 commits are not included. The Go public API and dependency versions are retained.

## Scope

The primary contract is extraction from supplied HTML or a parsed DOM, not page acquisition. The [README](README.md#philosophy-and-scope) explains the philosophy and intentional differences. Existing CLI network conveniences are retained, but downloader, crawler, feed, and sitemap changes are excluded from this update. No pages were fetched for regression testing; all extraction comparisons used saved HTML.

The port preserves its structured JSON-LD parser, HTML-oriented output, Go date/language dependencies, and Go fallback extractors. Python serializer fixes are applied only where they also change the extracted structure. They do not introduce Markdown, XML/TEI, CSV, or YAML output. Standalone Simhash/token APIs and Python configuration/deprecation machinery are not implemented here.

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

Tests were extended in the existing files rather than introducing a parallel test framework:

- [html-processing_test.go](html-processing_test.go): pruning tails, cleaning, DOCTYPE parsing, code detection, linked images, fenced frames, link density, and selector boundaries.
- [trafilatura_test.go](trafilatura_test.go): image URLs, nested formatting, table alignment and nesting, Unicode span/deduplication bounds, recovery, fallback guards, comment/forum routing, recall escalation, and caller-DOM preservation.
- [baseline_test.go](baseline_test.go): JSON content shapes, Discourse preload data, embedded HTML versus comparison operators, strategy isolation, multiple articles, consent pruning, and block-aware text measurement.
- [metadata_test.go](metadata_test.go) and [metadata-json_test.go](metadata-json_test.go): titles, image metadata, full author names, nested license text, safe JSON values, and publisher replacement.
- [realworld_test.go](realworld_test.go): retained the existing saved-page suite and updated one obsolete expectation that deliberately allowed boilerplate now removed by both implementations.

## Verification

Verification used Windows, Go 1.24.2, and a separate Python 3.12 environment with Trafilatura 2.2.0 installed from the pinned source. Python dependencies included lxml 6.1.3 and htmldate 1.10.0; Go dependencies remain those in [go.mod](go.mod). Neither the reference checkout nor temporary comparison tooling is part of this repository.

- All Go package and saved-page tests pass with `go test ./...`.
- 22 representative comparisons matched after normalizing the two DOM vocabularies and whitespace: body/comments, link targets, image sources, headings, and table cells. They cover cleaning, inline nesting, code, images, tables, recovery, comments/forums, JSON rescue, and recall escalation, with fallbacks disabled.
- On the existing 960-page corpus, 838 bodies matched Python after removing whitespace differences. The remaining 122 are not claimed to have identical output.
- The isolated corpus comparison below uses balanced mode, no fallbacks, comments excluded, tables included, and the same saved HTML/annotations. Failed extraction is counted as empty text, not omitted. The updated Go run and Python both reject one page under these options.

| Extractor | Precision | Recall | F1 |
| --- | ---: | ---: | ---: |
| Untouched Go baseline | 0.9123 | 0.8966 | 0.9044 |
| Updated Go port | 0.9171 | 0.9113 | 0.9142 |
| Pinned Python 2.2.0 | 0.9155 | 0.9082 | 0.9118 |

These are phrase-level annotation scores, not proof of universal superiority or exact parity. The Python comparison normalizes inter-element whitespace; the Go text column uses `ContentText`. The repository's existing multi-extractor benchmark also showed higher F1 in all four Go modes, but its shared input/execution order yields slightly different aggregate counts, so the isolated results above are the compatibility record.

Timing was measured with interleaved baseline/updated runs after a warm-up. Unchanged Readability and Dom Distiller runs also slowed materially during the later measurements, so no reliable speedup or slowdown is claimed. Historical README timing tables are not relabeled as v2.2.0 results.

The race detector could not run: this environment has CGO disabled and no GCC/Clang toolchain. The `make test` generator prerequisite could not run because re2go is unavailable; no generated regex source was changed. These gates should be rerun in a development environment providing those tools:

```sh
go test -race ./...
make test
```

The existing local corpus benchmark remains available without temporary tooling:

```sh
go run ./scripts/comparison content -j 1
```

## Compatibility Boundaries

The Go fallback sequence intentionally does not copy jusText's sanitized-output trigger and replacement ratio. Testing that substitution with Dom Distiller reintroduced gallery boilerplate in an existing real-world fixture. The established Go candidate ordering/stopping rule is retained, while applicable empty/raw-JSON guards, recall preference, URL conversion, and a bounded Dom Distiller escalation candidate are included.

JSON-LD metadata traversal retains Go's article-priority rules and strict parser behavior, including support for bare publisher strings. Its results can differ from Python's regex recovery, graph ordering, and live-blog metadata choices. Baseline content recovery supports the new schema properties and Discourse preloads, but does not accept malformed JSON solely to mimic permissive Python parsing.

The returned HTML DOM is not an untrusted-HTML security sanitizer. As before, applications rendering extracted content must apply their own security policy.