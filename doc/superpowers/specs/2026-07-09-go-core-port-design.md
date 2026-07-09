# Go core port — design

- **Date:** 2026-07-09
- **Status:** Proposed
- **ADR:** [0006 — Go as a supported language](../../adr/0006-go-port.md)
- **Mirrors:** `python/packages/var-core` (closest full-pipeline precedent),
  reference `typescript/packages/var-core/src`.

## Goal

Port the pure functional core of `var` to Go as `github.com/oselvar/var/go`'s
`varcore` package, proven by reproducing every `conformance/bundles/*/golden/*`
artifact byte-for-byte, plus the config corpus and (unit-gated) drift. This
spec covers the **pure core + author facade**; the runner + `testing` adapter
are in the companion spec `2026-07-09-go-runner-testing-design.md`.

A port **translates the cited algorithm; it does not redesign it.** Each Go
file below names its TS/Python source. The behavioural spec for each module is
that module's TS source file *and* its `*.test.ts`.

## Package layout

Single Go module, `internal`-enforceable boundaries:

```
go/
  go.mod                      module github.com/oselvar/var/go  (go 1.24)
  varcore/                    pure pipeline — no facade/runner/adapter imports
  var/                        author facade: DefineState, Registrar
  varconfig/                  var.config.json reader
  varrunner/                  imperative shell (companion spec)
  vartesting/                 stdlib testing adapter (companion spec)
```

**Boundary gate:** CI greps that `varcore/` imports nothing from
`.../var`, `.../varrunner`, `.../vartesting` (mirrors the Python split's
`grep -rn "from var\b"` gate).

## Naming conventions

TS camelCase / Python snake_case → Go idiom: exported `PascalCase` funcs/types,
file names snake_case kept **parallel** to the Python module for reviewability
(`scanner.go`, `cell_diff.go`, …). Immutable data types are structs with
unexported fields + constructors, or exported readonly-by-convention structs
returned by value; slices/maps are never mutated after construction (the ADR
0001 immutability principle).

## Module map (port these `varcore` files)

Mirror `python/packages/var_core/src/var_core/*.py` (which mirrors the TS core):

| Go file | Ports (py / ts) | Notes |
|---|---|---|
| `span.go` | `span.py` | **UTF-16 layer.** Done — offsets/cols in UTF-16 code units over UTF-8 source. |
| `hash.go` | `hash.py` | Done — FNV-1a over UTF-16 code units, `fnv1a:` prefix. |
| `canonical_json.go` | `canonical_json.py` | Done — `encoding/json` encoder, `SetEscapeHTML(false)`, 2-space, trailing `\n`, map-key sort. |
| `ast.go` | `ast.py` | Node structs: `Heading, Paragraph, ListItem, Blockquote, Table/Row, Fence, ThematicBreak, Example, VarDoc`, `SegmentOffset`. |
| `inline.go` | `inline.ts` (folded in py) | Inline segment mapping (segmentMap). Confirm scope vs py — py folds this into scanner/structurer. |
| `table_cells.go` | `table_cells.py` | Split `| a | b |` row → trimmed cells + per-cell spans. |
| `sentences.go` | `sentences.py` | Sentence segmentation (example name = first sentence). |
| `scanner.go` | `scanner.py` (~461 LOC) | Largest. Line-based markdown scanner + gherkin plugins (tables, doc-strings). |
| `structurer.go` | `structurer.py` | Paragraphs/list-items/blockquotes → candidate examples; headings → scope. |
| `parse.go` | `parse.py` | `Parse(path, source, plugins)` = `structure(path, source, scan(...))`. |
| `step_role.go` | `step_role.py` | Role enum: stimulus / sensor. |
| `registry.go` | `registry.py` | Immutable step registry (expression + compiled cucumber expr + role + source file). |
| `matcher.go` | `matcher.py` (~245) | Hit resolution, ambiguity, capture-offset shifting **around** cucumber-expressions. |
| `plan.go` | `plan.py` (~518) | 2nd largest. `ExecutionPlan`: examples, scope stacks, matched steps, args, tables, doc strings, expected outcome, diagnostics. |
| `diagnostics.go` | `diagnostics.py` | `DiagnosticCode`, `Severity`, diagnostic types. |
| `execute.go` | `execute.py` (~374) | Executor; **full-replacement** state merge (fork from py's partial-merge). |
| `deep_freeze.go` | `deep_freeze.py` | N/A in Go's value semantics; likely a no-op or dropped — confirm during trace stage. |
| `cell_diff.go` | `cell_diff.py` (~183) | Per-column row comparison; `CellMismatchError`, `ReturnShapeError`. |
| `doc_string_diff.go` | `doc_string_diff.py` | Doc-string body diff. |
| `param_diff.go` | `param_diff.py` | Render one side of a parameter diff via the parameter type's format. |
| `failure.go` | `failure.py` | Recover 1-based failing line (language-agnostic source position). |
| `failure_anchor.go` | `failure_anchor.py` | Where a failure points in the `.md`. |
| `result.go` | `result.py` | Mismatch-as-source-offset-range value type. |
| `conformance.go` | `conformance.py` (~397) | `ToVarDocArtifact`, `ToRegistryArtifact`, `ToPlanArtifact`, `toFailureArtifact`, `fileStem`, `RunConformance` (trace inline). |
| `drift.go` | `drift.py` (~250) | Jaccard 0.5 re-identification; `drift` diagnostic. Unit-gated. |
| `baseline_store.go` | `ports.py` (BaselineStore) | Port interface only; fs impl lives in `varrunner`. |

Deliberate py divergences to reproduce: `canonical_json` is its own module (TS
folds it into `conformance.ts`); py has no `deep_equal`/`run_diagnostics`/`ports`
separate modules. For Go, decide `deep_freeze` (Go returns by value — likely
unneeded) and `expression_segments` (only needed if the cucumber-expressions AST
route fails) during the relevant stage; do not port speculatively.

## Author facade (`var` package)

- `DefineState[S]` / `Registrar[S]`: the injected-registration API (ADR 0006).
  `stimulus` returns `(S, error)` (full next state); `sensor` returns
  `(any, error)` compared against the Markdown by the core's slot contract.
- The sensor slot contract is unchanged from the shared spec (zero/one/two+
  slots → return-shape rules); only the **state merge** differs
  (full-replacement, not shallow-merge).

## Conformance staging (each an independently gated milestone)

Gate order — never jump ahead:

1. **var-doc.json** — parse only. Build the Go conformance harness (a `go test`
   that reads each `bundles/*/example.md`, projects the stage artifact,
   `CanonicalStringify`s it, byte-compares the golden). **Prove the UTF-16
   layer against bundles `11-emoji-offsets` + `12-combining-marks`.** No
   `steps.go` fixtures needed yet.
2. **registry.json** — needs a `*.steps.go` fixture in every bundle (6th
   language), registering the same expressions/handlers as `*.steps.ts`.
   `parameterTypeNames` from the compiled expression AST.
3. **plan.json** — matcher + plan + diagnostics.
4. **trace.json** — executor (full-replacement merge), diffs, failure
   artifacts. Built inline in `RunConformance` (no `toTraceArtifact`).
5. **drift** — unit-gated (no golden): translate `hash.test.ts`/`drift.test.ts`.
   `var.lock.json` uses its own serializer (2-space, spec paths sorted,
   insertion-order keys otherwise) — NOT `CanonicalStringify`.

## Canonical JSON — resolved

`encoding/json`'s `Encoder` with `SetEscapeHTML(false)` + `SetIndent("", "  ")`
plus a trailing newline (Encoder.Encode appends one) reproduces the format when
artifacts are built from `map[string]any` (the encoder sorts map keys). Proven
byte-for-byte against **all 65 goldens** by a round-trip test
(`canonical_json_test.go::TestCanonicalStringifyRoundTripsGoldens`). Known edge:
Go always escapes U+2028/U+2029; no golden contains them (if one ever does, the
writer switches to a hand-rolled recursive serializer).

## Verification

- Per stage: the Go conformance harness is green for that artifact across all 15
  bundles.
- UTF-16: bundles 11 + 12 pass.
- Boundary gate green; `go vet` + `gofmt` clean; `go test ./...` green.
