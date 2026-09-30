# go-trafilatura Agent Instructions

These are instructions for LLM agents and maintainers updating README.md,
UPSTREAM.md, CHANGELOG.md, benchmark claims and releases. Read current files
and git status first; preserve unrelated work and explicit user constraints.

## Library Identity

This is a supplied-HTML extraction library, not an application-worker project.
Use the approved nine-section format below, with package-specific APIs and
limits. `go-trafilatura` follows the pinned `adbar/trafilatura` non-fallback core with documented
differences. Compatibility tests are evidence, not proof of complete parity.

FAST disables external fallback, not native recall or baseline recovery.
Non-FAST uses only internally generated bundled readability-lxml. Standalone
extractors are outside this package's API. Keep application-worker history and
other packages' integration policies out of its maintenance rules.

## Required Creator Acknowledgments

The README's License and Credits section must explicitly name:

- Adrien Barbaresi, creator of the original `adbar/trafilatura` Python package;
	link the upstream project and his 2021 ACL/IJCNLP paper.
- Radhi Fadlillah, author of the initial `go-trafilatura` port from the Python
	`adbar/trafilatura` package; explicitly distinguish initial port authorship
	from current maintenance.
- Markus Mobius, maintainer of `go-trafilatura`; do not attribute the original
	Python package or initial Go port to the current maintainer.
- Arc90 for the original algorithm, starrhorne and iterationlabs for the Ruby
	port, and gfxmonk for the Python port, as named in `adbar/trafilatura`'s
	bundled readability-lxml header. Retain its links to the
	`timbertson/python-readability` and `buriy/python-readability` contributors;
	do not turn the two Ruby authors into an invented repository name.

Preserve the Apache-2.0 LICENSE and original notices in adapted source files.
Check UPSTREAM.md and the pinned upstream source before changing attribution.
Name the creators in prose, not just repository links; do not imply endorsement.

## Candidate Removal Explanation

Explain that candidates extracted before `go-trafilatura` input cleanup can retain
long boilerplate that passes length checks and replaces article content. Do not
say `go-domdistiller` performs no cleaning: the relevant difference is input
preparation. Use the historical same-version control, clearly labeled: `go-trafilatura` 2.2.2
selected external fallback on 780/6,554 pages (11.90%) with generated candidates
and 2,408/6,554 (36.74%) with supplied candidates; `go-domdistiller` selections were
4 versus 1,614. The complete 2.2.6 lxml-only policy measured 202/6,554 (3.08%).
That last reduction also changes the algorithm set and is not solely the effect
of candidate removal. Selection rates are not accuracy scores. Count final
returned sources, not calls or temporary selections; keep failures in the
denominator and internal recovery separate. See the controlled evidence in
UPSTREAM and the benchmark repository before adding or changing numbers.

## Document Roles

- README.md: purpose, scope, usage and concise current quality/speed. Preserve
  useful API/examples and clearly label historical comparisons and limitations.
- UPSTREAM.md: source/dependency pins, deliberate deviations, controlled evidence,
  reproduction commands, hashes, coverage and unresolved compatibility issues.
- CHANGELOG.md: dated/versioned user-visible changes and reasons, not worker
  incidents or an audit transcript. Label documentation-only changes explicitly.
- Release notes: the same release-facing changes, benchmark definitions and
  limitations as the README/changelog, with links to detailed evidence.
- AGENTS.md: durable instructions, not current results or work-in-progress.

## Package Naming

Always identify the implementation by its full package name, including in
titles, headings, prose, tables, captions, changelogs and release notes:

- `go-domdistiller`
- `rust-domdistiller`
- `go-readabilityV2`
- `rust-readability-v2` (repository: `rust-readability`; Rust import: `rust_readability`)
- `go-trafilatura` (Go module: `github.com/markusmobius/go-trafilatura/v2`)
- `rust-trafilatura`

Never replace a package identifier with a bare algorithm name or a generic
language/algorithm label. Do not drop the language prefix or the versioned
package suffix. Give measured versions beside package names in benchmarks.
When discussing upstream projects, use their owner-qualified repository names,
not names that could be mistaken for one of these packages. Keep actual code identifiers
and import aliases unchanged; naming prose precisely is not an API rename.

## README Format

The rules below define the approved structure and required content for all six
library READMEs. Each repository's README demonstrates these rules; it is not a
substitute for this specification. Keep package-specific guidance local to its
own repository. The benchmark repository retains its methodology/results layout.

Consistency means the whole README, not just an identical benchmark table.
Use these exact level-two headings in this exact order. Every point in the
Required Content column is mandatory, not a suggestion:

| Order | Heading | Required Content |
| --- | --- | --- |
| 1 | `## Philosophy` | State all three principles in order: bring your own HTML; stay as close as possible to upstream; provide very fast native Go and Rust packages. Explain each using the required points below. |
| 2 | `## Overview` | Inputs, outputs, scope, current release and source/reference relationship. Identify which entry points fetch pages, if any. |
| 3 | `## Installation` | One command pinned to the current published version. Explain package/import names where they differ and relevant compiler requirements. Never recommend `@main`, `-u` or an unpinned branch as the default. |
| 4 | `## Usage` | One complete runnable example using local HTML, without network access or external fixtures. Show useful output. Follow with a short entry-point/result summary; link generated API docs instead of copying full type definitions. |
| 5 | `## Options` | A compact table of this package's important controls, their actual defaults and effects. Put explanations of its own behavior under level-three headings here. |
| 6 | `## Current Quality and Speed` | The shared six-package comparison with `### Extraction Speed` and `### Text Quality`. Use full package names, measured versions, common units, timing boundaries and evidence links. |
| 7 | `## Compatibility and Limitations` | Current behavioral target, important omissions, input/output safety and known differences. Link UPSTREAM for evidence rather than copying investigation logs. |
| 8 | `## Development` | Existing commands for relevant tests and checks, with prerequisites. Listing commands does not mean they were run successfully. |
| 9 | `## License and Credits` | Name the original upstream creators and the relevant port authors explicitly, using this repository's verified attribution. Link actual licenses/notices and upstream projects; links alone do not replace creator acknowledgments. Do not reduce multiple inherited licenses to a blanket MIT claim. |

### Required Philosophy Points

1. **Bring your own HTML.** Present supplied HTML as the primary workflow.
	The caller controls fetching, caching, rendering, retries and scheduling;
	extraction is a separate concern. Do not claim existing URL convenience
	helpers are absent: identify them in Overview without making them the default
	installation example or integration path.
2. **Stay close to upstream.** Preserve the declared upstream algorithms and
	behavior as closely as possible rather than inventing a separate extractor.
	Name the actual reference in Overview, including when a Rust package follows
	its Go counterpart. Document deliberate differences and compatibility limits
	in UPSTREAM; do not turn this principle into a claim of universal parity.
3. **Provide very fast Go and Rust packages.** State native execution and high
	throughput as design goals. Improve performance without silently changing the
	intended extraction behavior. Support concrete speed claims with reproducible
	benchmarks and report quality alongside speed; do not claim every package is
	always faster than its reference or that Rust is always faster than Go.

Before those sections, use `# <Full Package Name>` and one short purpose
paragraph. Keep section headings language-neutral; real API and language
differences belong in their contents. Do not force identical functionality
onto different packages or retain a competing legacy outline below the new one.

Keep historical results in dated technical records linked from the current
comparison. Do not retain old ranking essays, old timing tables, project origin
stories or full copied API structs as extra README sections. Preserve evidence
in UPSTREAM or immutable reports rather than deleting or relabeling it. Claims
such as "fastest", "best accuracy", "stable enough" or "minimal effects" require
specific current evidence and stated limits, not an old anecdote or citation.

Before finalizing README changes:

1. Check the entire level-two heading sequence against the table above, not
	just the benchmark section. Check every required content point, including
	all three philosophy principles in their specified order.
2. Verify installation, API names and option defaults against the actual
	release. Run the exact example against that version without network input.
3. Check benchmark arithmetic and labels against the saved structured report.
	Keep the measured versions when a later release changes documentation only.
4. Check every package reference and local link throughout the draft. Remove
	stale recommendations and unsupported rankings, not merely stale tables.
5. Preserve this approved structure and its named creator acknowledgments.
	Obtain approval before redesigning it; passing checks or previous release
	authorization is not approval of a new README design.

## Six-Repository Benchmark Contract

Follow the complete
[benchmark maintenance guide](https://github.com/markusmobius/content-extractor-benchmark/blob/master/AGENTS.md).
Coordinate go-domdistiller, rust-domdistiller, go-readabilityV2, rust-readability,
go-trafilatura and rust-trafilatura, not only this library's language pair.

1. Every README has `## Current Quality and Speed` with the same six-package
	comparison from one completed shared-suite report. Use measured version labels.
2. Speed columns: `Go Package (Measured Version)`, `Rust Package (Measured Version)`,
	`Go ms/page`, `Rust ms/page`, `Go/Rust`. Pair `go-readabilityV2` with
	`rust-readability-v2`, `go-domdistiller` with `rust-domdistiller`, and
	`go-trafilatura` with `rust-trafilatura`, in that order. Mark FAST on both
	packages in the last pair. Display milliseconds/page to three decimals and
	ratios to two decimals. Quality rows also name both packages explicitly.
3. Read structured JSON and compute ratios from unrounded means. The current
	shared protocol uses all four measured passes after one warmup. Never mix
	dates, environments, modes, means/medians or selected/all-pass aggregates.
4. Report parsing separately, charged once per language/page. Include decoding,
	normalization, DOM construction and any additional parse trees recorded in
	the shared benchmark's timing boundary.
	State corpus/counts, options, hardware, toolchains and included/excluded work.
	`go-domdistiller` and `rust-domdistiller` pagination must be identified as
	on/off and by algorithm.
5. Keep separate measurement runs and configurations separate. A paired
	language ratio is not an isolated version speedup or request latency.
6. Report named-corpus F1 percentages to five decimals, retaining errors in the
	denominators. Do not average different scoring definitions. Matching text
	scores do not prove byte-identical HTML or metadata; preserve known differences.
7. Link immutable reports/commits and preserve old artifacts unchanged. Clearly
	date old standalone timing/allocation tables; do not compare them directly
	to the current shared-input extraction boundary.
8. Documentation-only patches retain actual measured versions; do not relabel
	old measurements or rerun benchmarks just to change wording.

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