# Go-Trafilatura Maintenance

These are instructions for LLM agents and maintainers updating README.md,
UPSTREAM.md, CHANGELOG.md, benchmark claims and releases. Read current files
and git status first; preserve unrelated work and explicit user constraints.

## Product and Scope

This is a supplied-HTML extraction library, not an application-worker project.
Use the existing Go README as the readability reference when aligning the Rust
port. Keep its useful scope, usage and API explanations; avoid broad rewrites.
Go-Trafilatura follows the pinned Python non-fallback core with documented
differences. Compatibility tests are evidence, not proof of complete parity.

FAST disables external fallback, not native recall or baseline recovery.
Non-FAST uses only internally generated bundled readability-lxml. Standalone
Mozilla Readability and DomDistiller remain independent extractors. Removing
their use as candidates does not remove their standalone APIs or consumer flags.

## Document Roles

- README.md: purpose, scope, usage, important choices, concise current quality
  and speed. Explain why a behavior changed, not the history of the investigation.
- UPSTREAM.md: exact source/dependency pins, compatibility boundaries, controlled
  experiments, validation commands, hashes, coverage and unresolved differences.
- CHANGELOG.md: dated, versioned user-visible changes and their reasons. Keep
  historical entries historical. No rustHTML references or deployment incidents.
- GitHub release notes: the same release-facing changes and benchmark definitions,
  not a separate set of results. Link detailed evidence instead of copying logs.
- AGENTS.md: durable instructions, not a status journal or current results.

## Candidate Removal Explanation

Explain that candidates extracted before Trafilatura's input cleanup can retain
long boilerplate that passes length checks and replaces article content. Do not
say DomDistiller performs no cleaning: the relevant difference is input
preparation. Use the historical same-version control, clearly labeled: Go 2.2.2
selected external fallback on 780/6,554 pages (11.90%) with generated candidates
and 2,408/6,554 (36.74%) with supplied candidates; DomDistiller selections were
4 versus 1,614. The complete 2.2.6 lxml-only policy measured 202/6,554 (3.08%).
That last reduction also changes the algorithm set and is not solely the effect
of candidate removal. Selection rates are not accuracy scores. Count final
returned sources, not calls or temporary selections; keep failures in the
denominator and internal recovery separate. See the controlled evidence in
UPSTREAM and the benchmark repository before adding or changing numbers.

## Comparable Benchmarks

The shared contract lives in
[the benchmark maintenance guide](https://github.com/markusmobius/content-extractor-benchmark/blob/master/AGENTS.md).
Apply it consistently to go-domdistiller, rust-domdistiller, go-readabilityV2,
rust-readability, go-trafilatura and rust-trafilatura.

1. Keep `## Current Quality and Speed` in every README. Use the same completed
	shared-suite report, measured versions and table in all six repositories.
2. Table columns are `Extractor`, `Go Version`, `Rust Version`, `Go ms/page`,
	`Rust ms/page`, `Go/Rust`. Rows are Readability, DomDistiller and Trafilatura
	FAST. Show milliseconds/page to three decimals and ratios to two decimals.
3. Derive numbers from structured reports; compute ratios before rounding.
	The current shared protocol uses all four measured passes after one warmup.
	Never mix dates, machines, modes, means/medians or selected/all-pass results.
4. Charge shared parsing once per language/page and report it separately.
	State corpus/counts, environment, options, warmups/passes and timing boundaries.
	Include any separate noscript parse; do not hide it in untimed setup.
5. Report non-FAST Trafilatura as a separately labeled comparison from its own
	report. A within-run Go/Rust ratio is not an old/new release speedup.
6. Keep corpus F1 scores separate, with errors retained and five-decimal
	percentages. Different scoring definitions cannot be averaged. Matching text
	scores do not establish identical HTML or metadata; retain known differences.
7. Link immutable benchmark commits/reports. Preserve old reports unchanged and
	label historical tables. Do not mix annotated quality and fallback corpora.
8. Documentation-only releases retain actual measured version labels. Do not
	rerun measurements for wording changes or relabel an old run as a new one.

## Workflow and Release Checks

1. Identify the change and its evidence before editing. Keep algorithm, worker,
	benchmark and documentation tasks separate. Do not change runtime behavior
	to justify wording or restart a completed benchmark campaign.
2. Update README, UPSTREAM and CHANGELOG in their roles. Coordinate the common
	benchmark section with all six repositories and the benchmark repository.
3. Validate every number, link, option name and version. Compare the six common
	sections and run `git diff --check`. Preserve unrelated dirty files.
4. Use the existing reviewed-difference gate in `scripts/check_tests.py` for
	relevant Go validation: ordinary tests intentionally report documented Python
	differences. Do not weaken assertions or claim unrun gates passed.
5. Obtain explicit authorization before commits, pushes, new versions/tags or
	releases. Never move a published tag. Go documentation-only edits do not
	automatically require a new module version or changes to consumers.
6. For an authorized release, finalize all documentation before tagging. Keep
	dependency/runtime changes separate from documentation-only releases. Verify
	normal module download with checksum verification enabled and verify the
	GitHub release page; a tag alone is not a release page.
7. Check exact committed and remote bytes before publishing hashes; Windows
	CRLF worktree bytes can differ from Git LF bytes. Do not expose credentials.
8. Crates.io READMEs cannot be changed in place. Coordinated Rust documentation
	updates need explicit approval for new patch versions; finish docs before
	packaging, inspect the archive and verify published source/doc bytes.
9. Report actual checks, publication state and remaining limits concisely.
	Put detailed receipts in UPSTREAM/reports, not the README or changelog.