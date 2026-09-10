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

The Go runner excludes failed extractions from its score totals rather than scoring them as empty text. Take this into account when comparing scores or runs with different numbers of failures.

This tool does not run Python. The [Python Trafilatura 2.2.0 comparison](../../README.md#comparison-with-python-trafilatura) was measured separately, counting failed extractions as empty text. See the [verification record](../../UPSTREAM.md#verification) for versions, options, and limitations. Running the commands above with newer code or dependencies will not reproduce the historical timing table exactly.

## Sources

Annotated HTML documents:

- BBAW collection (multilingual): Adrien Barbaresi, Lukas Kozmus.
- Polish news: [tsolewski](https://github.com/tsolewski/Text_extraction_comparison_PL).

HTML documents:

- Additional German news sites: diskursmonitor.de, courtesy of Jan Oliver Rüdiger.
