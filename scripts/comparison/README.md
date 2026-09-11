# Comparison

Compare content extraction by Go-Trafilatura, Go-Readability, and Go-DomDistiller using the saved HTML corpus and phrase annotations in this repository.

## Usage

Run these commands from the repository root. To compare content extraction with one worker:

```sh
go run ./scripts/comparison content -j 1
```

To use up to `GOMAXPROCS` concurrent workers:

```sh
go run ./scripts/comparison content -j -1
```

To check author extraction:

```sh
go run ./scripts/comparison author
```

## Interpreting Results

The content command reports precision, recall, accuracy, F1, and elapsed time for each extractor configuration. The corpus contains 960 documents. File loading and HTML parsing happen before the timed extraction runs; these timings are not measurements of the complete reader-to-result pipeline.

The current Readability row uses `codeberg.org/readeck/go-readability/v2` v2.1.2 and its `RenderText` API. Historical tables in the main README used the deprecated Go-Shiori backend; they are not measurements of this replacement.

The Go runner excludes failed extractions from its score totals rather than scoring them as empty text. Take this into account when comparing scores or runs with different numbers of failures.

This tool does not run Python. The [Python Trafilatura 2.2.0 comparison](../../README.md#comparison-with-python-trafilatura) was measured separately, counting failed extractions as empty text. See the [verification record](../../UPSTREAM.md#verification) for versions, options, and limitations. Running the commands above with newer code or dependencies will not reproduce the historical timing table exactly.

## Python Test Fixtures

[import-python-tests.py](import-python-tests.py) is a separate test-source importer, not a corpus benchmark. It records original assertions from Python Trafilatura 2.2.0 and generates the JSON fixtures consumed by the Go tests. It also imports the original gzip resource and writes a source-level coverage inventory. Normal `go test` runs do not require Python.

Use an isolated Python 3.12 environment with the pinned upstream checkout installed, plus pytest and lxml. The verified environment also includes py3langid for the reference language tests; full reference versions and current failures are recorded in [UPSTREAM.md](../../UPSTREAM.md#python-test-synchronization). The checkout must be at commit `c1bc9531a2a978326112ca9987e1382745116136` with unchanged tracked test resources.

From this repository's root, regenerate or verify without writing files:

```sh
python scripts/comparison/import-python-tests.py --upstream /path/to/trafilatura
python scripts/comparison/import-python-tests.py --upstream /path/to/trafilatura --check
```

Do not replace expectations with Go output. Source-line exclusions and native DOM/API translations are recorded in the generated [coverage inventory](../../test-files/python-2.2.0-coverage.json); a translated test is not claimed to be an identical Python serializer or private API test. Both commands read saved local resources and do not acquire pages.

The fixtures currently contain 465 operations and 575 original assertion cases, including four JSON-normalization checks. Native mappings include explicit `go_tests` targets and are logged separately from genuine exclusions, without adding pass/skip results. The [CI checker](../check_tests.py) verifies their target groups execute; it uses only the standard library and does not require the importer environment. Known failing leaves remain listed in the reviewed [difference manifest](../../test-files/known-differences.json).

## Sources

Annotated HTML documents:

- BBAW collection (multilingual): Adrien Barbaresi, Lukas Kozmus.
- Polish news: [tsolewski](https://github.com/tsolewski/Text_extraction_comparison_PL).

HTML documents:

- Additional German news sites: diskursmonitor.de, courtesy of Jan Oliver Rüdiger.
