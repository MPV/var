# Go runner + `testing` adapter — design

- **Date:** 2026-07-09
- **Status:** Proposed
- **ADRs:** [0006 — Go port](../../adr/0006-go-port.md),
  [0007 — Go `testing` integration](../../adr/0007-go-testing-integration.md).
- **Mirrors:** `python/packages/var-runner` + `python/packages/var-unittest`
  (closest: run-time generation, stdlib framework).

## Goal

The imperative shell for Go: spec/step discovery, planning, failure rendering,
the filesystem drift baseline (`varrunner`), and one stdlib `testing` adapter
(`vartesting`) giving one `t.Run` sub-test per Markdown example. Built **only
after `varcore` is conformance-green** — the shell is validated *through* the
proven core; building it first debugs two unknowns at once.

## `varrunner` (mirror `var_runner`)

| Go file | Ports | Owns |
|---|---|---|
| `discovery.go` | `discovery.py` | Resolve `docs`/`steps` globs from `var.config.json` → spec + step file lists. |
| `steps.go` | `steps.py` | `LoadSteps` — obtain the author's registrar-populated registry. |
| `run.go` | `run.py` | `RunSpec` / `PlanSpec` orchestration over `varcore`. |
| `render.go` | `render.py` | `RenderFailure` — human-readable text anchored to the `.md`, reusing core diff payloads (never re-derived). |
| `baseline_store.go` | `baseline_store.py` | Filesystem `BaselineStore` reading/writing `var.lock.json` (its own serializer, not `CanonicalStringify`). |

No `var` CLI for v1 (`hasCli:false` in `languages.json`) — TS-only so far.

## `vartesting` (stdlib adapter — ADR 0007)

Entry point the author calls from one ordinary test:

```go
func TestVar(t *testing.T) { vartesting.Run(t, steps.Define) }
```

`Run`:

1. reads `var.config.json` (`varconfig`), replays the author's `DefineSteps`
   against a fresh registrar (injected registration, no globals), finds + plans
   every matched spec (`varrunner`);
2. `t.Run` per spec file → nested `t.Run` per example (header-bound table rows
   are separate examples), scope-stack headings as nested names, sanitised for
   `-run` targeting;
3. executes each example through the core executor; a `var` diff/failure →
   `t.Error` with `varrunner.RenderFailure` text carrying `<spec>.md:<line>`; a
   handler panic is `recover`ed into a failing sub-test;
4. plan diagnostics (`ambiguous-match`, `error-fence-without-step`, `drift`) →
   failing marker sub-tests;
5. **drift gate:** reconcile each spec vs `var.lock.json` via the runner's
   `BaselineStore`; a drifted example fails; write the baseline on a clean run;
   acknowledgment via env var / flag threaded into `Run` (ADR 0002).

Fixture bridging: trailing handler params beyond the expression's actual capture
count are injected by position (count from the compiled expression). Per-example
fixture lifecycle out of scope for v1.

## Testing

- **Dogfood bundles test:** run every `conformance/bundles/*` through
  `vartesting` and assert outcomes match `trace.json` (mirrors py
  `test_dogfood_bundles.py`).
- **Adapter drift test:** a `var.lock.json` fixture → assert a `drift` diagnostic
  fails the suite; `--update` path writes/accepts (mirrors py `test_drift.py`).
- **Failure rendering test:** intentionally break a cell → assert
  `CellMismatchError` renders at the correct `.md` span.
- **End-to-end:** `examples/go-testing` runs its `.md` specs under `go test`,
  one sub-test per example.
