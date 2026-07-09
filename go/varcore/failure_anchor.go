package varcore

// failure_anchor.go — where a failure points in the .md source. Port of
// failure_anchor.py / failure-anchor.ts. A mismatch anchors at its first failing
// span (the cell, the doc-string fence body); anything else at the fallback (the
// step's match start). Pinned in the conformance trace as failure.anchor, so
// every port reproduces it byte-for-byte.
func failureAnchor(err error, fallback Span) Span {
	if c, ok := isCellMismatchError(err); ok {
		for _, cell := range c.Cells {
			if !cell.OK {
				return cell.Span
			}
		}
		return fallback
	}
	if d, ok := isDocStringMismatchError(err); ok {
		return d.Diff.Span
	}
	return fallback
}
