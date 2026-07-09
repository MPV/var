# go-testing

A standalone [Vár](https://var.oselvar.com) project that runs Markdown specs as
tests with Go's standard `testing` package — one `t.Run` sub-test per example.

```bash
go test ./...
```

Select a single example the usual way:

```bash
go test -run 'TestVar/roman-numerals.md/944_/_CMXLIV'
```

The `.md` files are the specs; the step definitions live in `steps/`. One
ordinary test wires them together:

```go
func TestVar(t *testing.T) { vartesting.Run(t, steps.Bundle()) }
```

`Bundle()` assembles every spec's step definitions into one registry via the
injected `Registrar` (no global state); state evolves by full replacement (a
stimulus returns the whole next state). Drift is reconciled against
`var.lock.json`; run with `VAR_UPDATE=1 go test ./...` to accept intentional
changes.

Specs covered: `hello-var`, `deep-thought`, `tables-and-docstrings`,
`roman-numerals`, and `yahtzee`.
