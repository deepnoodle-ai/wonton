# Library Spec Analysis and Recommendations

Date: April 12, 2026

This document captures analysis and recommendations for the three new library
specs:

- [spec-pty.md](/Users/curtis/git/deepnoodle/wonton/docs/spec-pty.md)
- [spec-runewidth.md](/Users/curtis/git/deepnoodle/wonton/docs/spec-runewidth.md)
- [spec-valuediff.md](/Users/curtis/git/deepnoodle/wonton/docs/spec-valuediff.md)

It is intended to be a decision record for follow-on implementation work.

## Project Context

From [CLAUDE.md](/Users/curtis/git/deepnoodle/wonton/CLAUDE.md), Wonton is a
curated collection of Go packages for rapid application development, with a
strong emphasis on:

- cohesive packages that work well together
- minimal dependencies where practical
- good documentation and consistent APIs
- strong support for AI-assisted development

That context matters because "replace a dependency" is not enough on its own.
The replacement should either:

1. simplify Wonton materially, or
2. provide an API and implementation quality that is legitimately competitive
   with the best existing Go libraries in that category.

## External Ecosystem Baseline

As of April 12, 2026, the relevant ecosystem baseline checked during review was:

- `github.com/creack/pty` v1.1.24, with `pty/v2` also available
- `github.com/mattn/go-runewidth` v0.0.21
- `github.com/rivo/uniseg` v0.4.7
- `github.com/google/go-cmp/cmp` v0.7.0

Implication:

- `pty` only needs to exceed a relatively small, utilitarian API surface
- `runewidth` must be compared against both `go-runewidth` and `uniseg`
- `valuediff` should not assume "better than go-cmp" is a small target

## Current Local Usage

The actual current dependency footprint in this repo is:

- `creack/pty`: used in `termsession/session.go`
- `go-runewidth`: used broadly in `tui`, `terminal`, and `termtest`
- `go-cmp`: used in `assert`

Notable local observations:

- [termsession/session.go](/Users/curtis/git/deepnoodle/wonton/termsession/session.go:220)
  uses only `pty.Start(...)` and resize helpers via `Setsize`.
- [tui/input_view.go](/Users/curtis/git/deepnoodle/wonton/tui/input_view.go:335)
  calls `runewidth.StringWidth(string(r))` for a single rune; this should be
  `RuneWidth(r)`.
- [assert/assert.go](/Users/curtis/git/deepnoodle/wonton/assert/assert.go:131)
  performs a `reflect.DeepEqual` gate before calling `cmp.Diff`, which means
  the documented `go-cmp` semantics are not actually what determine equality on
  the happy path.

That last point is important for the `valuediff` review. The current behavior is
already semantically inconsistent.

## Package-by-Package Analysis

## `pty`

### Overall Assessment

This is the strongest spec of the three and the safest to implement first.

Reasons:

- small current usage surface
- clear dependency reduction
- low API complexity relative to the category
- meaningful ergonomic improvements over raw `*os.File`

The proposed `PTY` wrapper is directionally better than the current upstream
usage in Wonton:

- method-oriented API
- explicit `Size` type
- nil-safe, idempotent `Close()`
- a higher-level `Start(...)` entry point with initial sizing support

### What Is Good in the Spec

- The scope is grounded in actual usage rather than speculative completeness.
- A `PTY` type is a real improvement over returning a bare file.
- `Start(cmd, size)` matches how PTYs are commonly consumed.
- Supporting `GetSize()` and `Resize()` directly on the wrapper is correct.

### Gaps and Risks

The spec is good, but there are a few places where it should be tightened.

1. Platform ambition is broader than the likely test surface.

The spec names Darwin, Linux, FreeBSD, OpenBSD, NetBSD, DragonFly, Windows,
and generic fallback support. That is fine only if the package will actually be
validated on those platforms. Otherwise it becomes a maintenance liability.

Recommendation:

- Treat Linux and Darwin as the required tier.
- Treat the BSD variants as best-effort unless CI coverage exists.
- Be explicit in package docs about which platforms are first-class.

2. The API may accidentally regress advanced upstream use cases.

`creack/pty` exposes helpers such as `StartWithSize`, `StartWithAttrs`, and
size inheritance helpers. Wonton does not currently need the full surface, but
if the new package is meant to be ecosystem-grade, a few advanced hooks should
exist.

Recommendation:

- Add either `StartWithAttrs(...)` or a way to preserve caller-provided
  `SysProcAttr` safely.
- Add a size-inheritance helper, for example `InheritSize(dst, src)` or a
  method equivalent.
- Consider exposing `File()` or `SyscallConn()` in addition to `Fd()` for
  advanced callers.

3. `Start(...)` should specify overwrite semantics.

The spec currently implies that `Stdin`, `Stdout`, `Stderr`, and session
attributes are assigned unconditionally. That behavior should be documented
explicitly so callers know whether preconfigured fields are respected or
replaced.

### Recommendation

Proceed with this package first.

Suggested target:

- first-class Linux and Darwin support
- clean public wrapper API
- strong cleanup semantics
- enough escape hatches to avoid painting the package into a corner

If implemented carefully, this package can clearly exceed the practical value
of `creack/pty` for Wonton and for most downstream users.

## `runewidth`

### Overall Assessment

This package has the highest upside, but also the highest correctness risk.

The spec is strongest where it focuses on:

- ASCII fast path performance
- current Unicode data
- better handling of terminal-relevant emoji cases
- truncation that respects grapheme boundaries

The spec is weakest where it implies that a focused, width-oriented grapheme
implementation would exceed the current ecosystem. That is not true unless the
package is measured against `uniseg`, not just `go-runewidth`.

### What Is Good in the Spec

- The motivation is real. Wonton uses width calculations heavily.
- Eliminating `go-runewidth` and its transitive dependency is meaningful.
- The ASCII fast path is exactly the kind of optimization that matters for TUI
  and terminal workloads.
- Fixing the single-rune call site in
  [tui/input_view.go](/Users/curtis/git/deepnoodle/wonton/tui/input_view.go:335)
  is clearly worthwhile.

### Main Strategic Concern

The real bar is not `go-runewidth`.

`uniseg` already provides:

- grapheme segmentation
- width calculation
- step-based iteration
- a mature Unicode-text-segmentation model

If Wonton exports `Graphemes(...)` publicly and positions it for cursor
movement, text editing, and cluster-aware traversal, the implementation must be
fully credible as a grapheme engine. Handling only:

- combining marks
- ZWJ
- variation selectors
- regional indicators
- skin tone modifiers
- keycaps
- enclosing marks

is not enough to claim ecosystem-leading behavior.

### Specific Spec Issues

1. The proposed public iterator overpromises.

If `Graphemes(s)` is public, downstream users will treat it as a real grapheme
API, not merely a width helper. That increases the required correctness bar
substantially.

Recommendation:

- Keep grapheme scanning internal for v1.
- Export only `RuneWidth`, `StringWidth`, and `Truncate` initially.
- Add a public iterator only after conformance is strong enough to support it.

2. The width model is too restrictive.

The spec frames width as 0, 1, or 2. That misses some uncommon but real cases
handled by existing libraries, including characters with width 3 or 4 in
monospace-width calculations.

Recommendation:

- Do not hard-code the conceptual model to 0/1/2 only.
- Define the API in terms of "terminal cells" without promising a narrow range.

3. The Unicode data plan is underspecified.

`EastAsianWidth.txt` and `emoji-data.txt` are not enough to fully support:

- combining classification
- nonprint classification
- grapheme boundary behavior

Recommendation:

- Make the generator inputs explicit and pinned.
- Define the Unicode versioning and regeneration policy in the package docs.
- Add compatibility tests against representative Unicode corpora.

4. Truncation and cursor movement should be separated conceptually.

Terminal width calculation is narrower than full cursor-edit semantics. The spec
starts to blend them. That is attractive, but it increases the chance of
shipping a package whose API promises more than its implementation guarantees.

### Recommendation

Proceed, but narrow the initial public scope.

Recommended v1 target:

- `RuneWidth(r rune) int`
- `StringWidth(s string) int`
- `Truncate(s string, w int, tail string) string`
- internal grapheme-aware scanner
- exhaustive tests and benchmarks against both `go-runewidth` and `uniseg`

Recommended v2 target, only after correctness is proven:

- public grapheme iterator or low-level stepping API

If the goal is to exceed the Go ecosystem rather than just replace a dependency,
this package should aim to be:

- faster on ASCII-heavy terminal text
- as correct or more correct than existing width implementations
- explicit about what segmentation guarantees it does and does not provide

## `valuediff`

### Overall Assessment

This is the most useful package internally, but also the easiest to overscope.

The strongest reason to do this work is not "replace go-cmp everywhere." The
strongest reason is that Wonton's `assert` package wants:

- deterministic output
- colored diffs without post-processing hacks
- no surprise panics on unexported fields
- sane error comparison defaults

That is a good package target. A full `go-cmp` replacement is a much larger one.

### What Is Good in the Spec

- It correctly identifies that Wonton uses `go-cmp` in a narrow way.
- Native colored output is a real improvement for `assert`.
- Comparing unexported fields by default matches current Wonton expectations.
- Error comparison via `errors.Is` is the right default for assertion-oriented
  equality.

### Most Important Current-Code Observation

`assert.Equal` does not actually use `go-cmp` semantics for its equality check.

At [assert/assert.go](/Users/curtis/git/deepnoodle/wonton/assert/assert.go:131),
the implementation first checks `reflect.DeepEqual(got, want)`, and only then
uses `cmp.Diff(...)` to render a diff on failure.

That means:

- `cmpopts.EquateErrors()` does not influence pass/fail in `Equal`
- `Equal(T) bool` methods do not influence pass/fail in `Equal`
- the package documentation currently overstates the semantic role of `go-cmp`

This makes a local replacement more defensible, because behavior is already not
equivalent to a pure `go-cmp` model.

### Main Spec Risks

1. Public compatibility risk is understated.

The current `assert.EqualOpts` public API exposes `cmp.Option` semantics, and
[assert/README.md](/Users/curtis/git/deepnoodle/wonton/assert/README.md) shows
examples using `cmpopts.IgnoreFields(...)`.

Changing that directly to `valuediff.Option` is a breaking public API change.

Recommendation:

- Either accept and document the break explicitly, or
- stage the migration and preserve compatibility temporarily, or
- keep `valuediff` internal until the new option model is proven.

2. `IgnoreFields(names ...string)` is too broad.

Ignoring fields by unqualified name across all struct types is dangerous and can
produce false equality in unrelated parts of a value graph.

Recommendation:

- Make field ignoring type-scoped or path-scoped.
- Avoid a global field-name-only rule in the public API.

3. "Maps with NaN keys skipped" is not acceptable output behavior.

A deterministic diff package should not silently skip difficult cases.

Recommendation:

- Define a stable representation strategy for problematic map keys.
- If a value cannot be represented precisely, emit that fact explicitly.

4. The package should not be forced to match all of `go-cmp` in v1.

`go-cmp` has a deep and flexible option system. Wonton does not need all of it.
Trying to compete head-on immediately will slow the package down and muddy the
API.

### Recommended Package Positioning

Position `valuediff` as:

"Deterministic, assertion-friendly deep comparison and diffing for Go values."

Not as:

"A complete drop-in replacement for go-cmp."

That narrower claim is both more accurate and more achievable.

### Recommendation

Build this package first as the comparison backend for `assert`.

Recommended v1 target:

- `Equal(x, y any, opts ...Option) bool`
- `Diff(x, y any, opts ...Option) string`
- deterministic, readable output
- safe unexported-field handling
- `errors.Is` semantics for `error`
- color support

Recommended v1 non-goals:

- broad `go-cmp` option parity
- broad transformation system parity
- "works exactly like go-cmp"

If the internal migration succeeds and the API holds up, then promote it as a
standalone public package with confidence.

## Cross-Cutting Recommendations

### 1. Build Order

Recommended implementation order:

1. `pty`
2. `runewidth`
3. `valuediff`

Reasoning:

- `pty` is low-risk and immediately useful
- `runewidth` is high-value but correctness-sensitive
- `valuediff` benefits from a tighter internal-first rollout

There is also a reasonable alternate order:

1. `pty`
2. `valuediff`
3. `runewidth`

That order is appropriate if reducing risk and tightening `assert` semantics is
more important than landing the highest-upside Unicode work early.

### 2. Public API Discipline

All three specs are strongest when they expose only the part of the surface that
Wonton can maintain with confidence.

In particular:

- `pty` should avoid pretending to support every Unix nuance unless tested
- `runewidth` should avoid exporting a grapheme API before it is truly solid
- `valuediff` should avoid claiming general `go-cmp` parity too early

### 3. Test Strategy Should Match Ambition

If the goal is to exceed what is currently available in the Go ecosystem, tests
must do more than verify local usage.

Minimum bar by package:

- `pty`: cross-platform behavior tests on supported targets
- `runewidth`: conformance-oriented Unicode cases plus benchmark comparisons
- `valuediff`: golden output tests, fuzzing, edge-case coverage, and behavioral
  invariants

### 4. Performance Claims Need Benchmarks

Two specs make performance claims:

- `runewidth` on ASCII-heavy content
- `valuediff.Equal` via fast-path behavior

Those claims should be accepted only if benchmarked and documented.

## Final Recommendations

### `pty`

Decision: strong yes

Refinements:

- prioritize Linux and Darwin
- add a small number of escape hatches for advanced callers
- document overwrite semantics clearly

### `runewidth`

Decision: yes, but narrow v1 scope

Refinements:

- benchmark against `uniseg`, not only `go-runewidth`
- avoid exporting `Graphemes(...)` until correctness is stronger
- do not assume width is limited to 0, 1, or 2
- make Unicode data generation more explicit

### `valuediff`

Decision: yes, but position it as assert-oriented first

Refinements:

- treat it as an `assert` backend before positioning it as a general public
  replacement for `go-cmp`
- be explicit about the breaking-change implications for `assert.EqualOpts`
- redesign field-ignore semantics to be narrower and safer

## Suggested Next Steps

1. Tighten `spec-pty.md` with explicit supported-platform tiers and one or two
   advanced API escape hatches.
2. Tighten `spec-runewidth.md` to:
   - narrow the v1 public API
   - compare against `uniseg`
   - clarify Unicode data sources and guarantees
3. Tighten `spec-valuediff.md` to:
   - frame the package as assert-oriented
   - call out the `assert.EqualOpts` compatibility break explicitly
   - replace field-name-only ignore semantics with a safer model
4. Decide the implementation sequence before starting code, because these specs
   currently imply different levels of public ambition.

