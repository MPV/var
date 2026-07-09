package varcore

// parse.go — top-level parse entry: scan blocks then group them into Examples.
// Port of parse.py / parse.ts.

// Parse parses source into a VarDoc: scan blocks, then structure them into
// Examples. plugins participate in block recognition before the built-in rules
// (the core conformance corpus is reproduced with none).
func Parse(path, source string, plugins []ScannerPlugin) VarDoc {
	return Structure(path, source, Scan(source, plugins))
}
