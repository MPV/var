# ADR 0007 — Go `testing` integration

- **Status:** Proposed
- **Date:** 2026-07-09
- **Deciders:** TBD
- **Tags:** go, golang, testing, test-runner-adapter, cross-language

## Context

Go is a full language port ([ADR 0006](0006-go-port.md)). Like every other
port, its test-framework adapter must give **one independently
selectable/reportable test per Markdown example**, with failures rendered
anchored to the `.md` source span — the contract `var-vitest`, `var-pytest`,
`var-unittest`, `var-junit`, `var-kotest`, and the Ruby adapters all satisfy
([ADR 0003](0003-java-junit-integration.md) is the precedent for filling in the
mechanism per framework).

Go's testing story is unusual: there is one canonical runner, `go test`, driving
the stdlib `testing` package. It has **no file-collection hook** (unlike
pytest's `pytest_collect_file`) and **no discovery SPI** (unlike JUnit's
`TestEngine`). Tests are ordinary `TestXxx(t *testing.T)` functions the compiler
links in; sub-tests are created *at run time* by calling `t.Run(name, fn)`.
Crucially, `t.Run` sub-tests are **independently selectable and reportable**:
`go test -run 'TestSpecs/Converting_1'` runs exactly one, and each reports its
own pass/fail. This is precisely the per-example granularity the contract needs.

## Decision

### `vartesting` — one `t.Run` sub-test per example

The adapter exposes a single entry point the author calls from one ordinary Go
test in their package:

```go
func TestVar(t *testing.T) { vartesting.Run(t) }
```

At run time `vartesting.Run`:

- reads `var.config.json` via `varconfig`, obtains the step definitions from the
  author (see registration below), and finds + plans every matching `.md` spec
  through `varrunner`;
- calls `t.Run` **once per spec file** and, nested within, **once per planned
  example** (header-bound table rows are separate examples), reproducing the
  example's scope-stack headings as nested `t.Run` names so the sub-test path
  reads like the spec structure;
- names each sub-test from the example name (sanitised the way `go test` itself
  sanitises sub-test names — spaces to underscores) so `-run` can target it,
  and reports the `.md` source location in the failure message
  (`<spec>.md:<startLine>`), since `testing` ties a failure's file/line to the
  `t.Errorf` call site rather than exposing a settable source like JUnit's
  `TestSource`;
- runs the example through the core executor inside the sub-test; on a `var`
  diff/failure it calls `t.Error` with the shared span-anchored
  `varrunner.RenderFailure` text (a *failure*); a genuine unexpected panic in a
  handler is recovered and reported as a failing sub-test, never crashing the
  whole run.

Plan-stage diagnostics (`ambiguous-match`, `error-fence-without-step`, `drift`)
surface as failing marker sub-tests so they are reportable, not silent.

### Registration — injected Registrar (per ADR 0006)

The author supplies steps via an injected `Registrar`, not a global
accumulator. `vartesting.Run` accepts the author's `DefineSteps` function (or
reads it from a package-level value the author sets in the same file) and
replays it against a fresh registry sink each run — no global mutable state, so
repeated runs and parallel packages cannot interfere.

### Fixture bridging

Handlers keep the core signature `func(state S, captures...) (S, error)` under
the full-replacement state model. Trailing handler parameters beyond the matched
expression's **actual capture count** (read from the compiled cucumber
expression, not guessed) are classified as injected values by position, wrapping
the registry's handlers rather than changing the core contract. Per-example
fixture *lifecycle* (setup/teardown) is out of scope for v1, matching
`var-unittest`.

### Drift gate

`vartesting` reconciles each spec against `var.lock.json` through the runner's
filesystem `BaselineStore`, surfaces a `drift` diagnostic that fails the suite,
writes the baseline on a clean run, and honours an `--update`/acknowledgment
path ([ADR 0002](0002-drift-detection-and-acknowledgment.md) — never silently
accept drift). Acknowledgment is via an env var / flag the author threads into
`Run` (Go tests take no arbitrary CLI flags without registration), mirroring the
other adapters' `--update`.

## Consequences

### Positive

- Individual examples are first-class, independently selectable (`go test -run`)
  and reportable — parity with every other adapter, using only the stdlib.
- Zero test-framework dependency: the whole integration is "add the module and
  write one three-line `TestVar`."
- The adapter is thin — discovery, planning, drift, and failure rendering live
  in `varrunner`/`varcore`; a shared dogfood test asserts its outcomes against
  the `trace.json` goldens.

### Negative / risks

- **`.md` anchoring is the subtle part** (mirrors ADR 0003's `TestSource` risk
  and ADR 0005's RSpec-location risk). `testing` reports the `t.Errorf` call
  site as the failure location; the `.md:line` must therefore live in the
  failure *message* (and sub-test name), which IDEs surface but do not treat as
  a jump target. Acceptable for v1; a future `go test` JSON post-processor could
  improve it. Verify with a real sample project.
- Run-time (not compile-time) example discovery means a `go test` run lists
  examples only once `TestVar` executes — inherent to Go's no-file-collection
  model, and matches `var-unittest`/`var-kotest`.
- One adapter only for v1 (no second framework); Go's ecosystem has one
  canonical runner, so there is no `unittest`/`minitest`-style second target to
  prove the seam against.

## Alternatives considered

- **Bind to `godog`.** Rejected — godog is a full Gherkin/BDD runtime; its
  scenario-collection model does not map to one `.md` example per test, and it
  would impose a heavyweight dependency for no gain over `t.Run`.
- **Generate `.go` test files from specs (codegen).** Rejected — a build-time
  generator adds a toolchain step and drifts from the specs; run-time `t.Run`
  fan-out gives the same per-example granularity with no generated code.
- **A custom `testing.M`/`TestMain` runner.** Considered and folded in: authors
  may call `vartesting.Run` from `TestMain` if they prefer, but the default
  single-`TestVar` entry is simpler and needs no `os.Exit` bookkeeping.

## References

- [ADR 0003 — Java JUnit integration](0003-java-junit-integration.md) — the
  adapter-contract precedent this ADR fills in for Go.
- [ADR 0002 — drift detection & acknowledgment](0002-drift-detection-and-acknowledgment.md).
- Reference adapters: `python/packages/var-unittest` (closest — generate at run
  time, stdlib framework), `python/packages/var-pytest`.
- `doc/superpowers/specs/2026-07-09-go-runner-testing-design.md`.
