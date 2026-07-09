# ADR 0006 — Go as a supported language

- **Status:** Proposed
- **Date:** 2026-07-09
- **Deciders:** TBD
- **Tags:** strategy, language-support, go, golang, cross-language

## Context

`var` supports TypeScript, Python, Java, Kotlin, and Ruby. Adding a language is
now a well-trodden path: a language-neutral conformance corpus, a pure
functional core mirrored per language, and a thin runner + test-framework
adapter shell (the [`adding-a-language-port`](../../.claude/skills/adding-a-language-port/SKILL.md)
skill). [ADR 0001](0001-second-language-python.md) established the seam table
that every port inherits unchanged.

Go is attractive for the same reason each new port is: the marginal cost is
low and bounded, and Go reaches an audience the existing ports do not —
cloud-native / infrastructure teams who test in `go test` and would not pull in
a JVM or a Node toolchain to author executable specifications. Go's incumbents
(`godog` and the `gocuke`/`testify`-BDD lineage) are Gherkin-shaped; none offers
`var`'s Markdown-native, keyword-free, return-based model.

Go has **no runtime interop** with any existing port (unlike Kotlin, which sits
on the Java engine because both are JVM bytecode). So — like Python and Ruby —
Go is a **full pipeline port**: its own pure core mirroring `@oselvar/var-core`,
proven by the shared conformance corpus, not a facade over another engine
(the mechanical rule from the skill: *does the target language share a runtime
with an already-ported one?* Go does not).

## Decision

**We will support Go as a full language port**, following the Python precedent
module-for-module. Concretely:

- A `go/` module at the repo root (module path `github.com/oselvar/var/go`)
  with internal packages mirroring the Python layout: `varcore` (pure
  pipeline), `var` (author facade), `varconfig`, `varrunner`, and one
  test-framework adapter, `vartesting` (stdlib `testing`). A single Go module
  with `internal`-enforceable package boundaries is used rather than five
  separate modules — it keeps the hexagonal boundary enforceable (a grep gate
  proves `varcore` imports nothing from the facade/runner/adapter) without the
  overhead of five `go.mod` files.
- The pure core reproduces every bundle's four conformance artifacts, the
  config corpus, and (unit-gated) the drift feature, byte-for-byte.
- Go depends on the official **`github.com/cucumber/cucumber-expressions/go/v20`**
  module — the same `20.0.0` line every other port pins — used for regex
  compilation and matching. The `matcher` module ports only `var`'s own
  hit-resolution and offset-shifting *around* the library. **Caveat
  (investigated 2026-07-09):** unlike the JS/Python/Java packages, the Go module
  keeps its expression AST unexported, so parameter-type names cannot be read
  from it; the port instead vendors cucumber-expressions' own upstream
  tokenizer+parser (`ast.go`) into `varcore` to recover parameter nodes in
  source order — the same algorithm, not a redesign (see the core design spec).

### Author-API forks (decided explicitly)

Two facade decisions legitimately fork on the target language's idioms; Go, a
statically-typed language, follows the JVM ports rather than the
dynamically-typed ones:

- **State evolution: full-replacement immutable value.** A `stimulus` returns
  the whole next state value, not a shallow partial merged over the running
  state (the TS/Python model). This fits Go's static typing and matches
  Java/Kotlin; it sets the executor's merge step and the sensor slot contract.
- **Registration: injected Registrar.** The runner replays
  `DefineSteps(registrar)` against a fresh sink each run — no global mutable
  accumulator (the TS/Python import-for-side-effect model). This pairs
  naturally with full-replacement state and matches the JVM ports.

## Consequences

### Positive

- Reach into the Go/cloud-native community with a tool that has no equivalent
  there, using the toolchain those teams already run (`go test`).
- A statically-typed full port (after the JVM facade) that does *not* share a
  runtime further validates the core abstractions are not JVM- or
  TypeScript-shaped.
- Cheap and objective: the conformance corpus + completed Python port make
  "done" a mechanical, byte-for-byte target.

### Negative / risks

- **UTF-16 offset conversion (the single riskiest part).** Every span offset in
  the goldens is a UTF-16 code-unit offset. Go strings are **UTF-8 / byte
  indexed** — further from UTF-16 than Python's code-point indexing — so Go
  needs an explicit conversion layer (`span`, the matcher's capture-group
  offsets, and the drift `hash`). Gated by the multibyte bundles
  `11-emoji-offsets` and `12-combining-marks`.
- **Canonical JSON.** Go's `encoding/json` does not sort object keys for structs
  and HTML-escapes by default; the canonical writer builds artifacts from
  `map[string]any` (whose keys the encoder sorts) with `SetEscapeHTML(false)`
  and a trailing newline. Proven byte-for-byte against the whole golden corpus
  by a round-trip test before the pipeline exists.
- LSP / editor / snippet-generation stay TypeScript-only (per ADR 0001's seam
  table); the only per-language editor deliverable is the **tree-sitter-go
  dialect** for extraction (enforced by `language-coverage.test.ts`).

## Alternatives considered

- **A facade over an existing engine (as Kotlin does over Java).** Rejected:
  Go shares no runtime with any ported language. It is a full port.
- **Binding to `godog` instead of a custom stdlib adapter.** Rejected — see
  [ADR 0007](0007-go-testing-integration.md); godog pulls in a full
  Gherkin/BDD runtime `var` does not need and whose collection model does not
  map to one `.md` example per test.
- **Partial-merge state / global registration (the TS/Python model).**
  Rejected for Go — both are awkward in a statically-typed language; the JVM
  full-replacement + injected-Registrar model is the better fit.
- **Not adding Go.** Rejected — the marginal cost is low and Go reaches an
  audience no current port does.

## References

- [ADR 0001 — Python as the second language](0001-second-language-python.md) —
  the seam table and conformance strategy this port inherits unchanged.
- [ADR 0007 — Go `testing` integration](0007-go-testing-integration.md).
- [`adding-a-language-port` skill](../../.claude/skills/adding-a-language-port/SKILL.md).
- Reference port to mirror: `python/packages/*`.
- `doc/superpowers/specs/2026-07-09-go-core-port-design.md`,
  `doc/superpowers/specs/2026-07-09-go-runner-testing-design.md`.
