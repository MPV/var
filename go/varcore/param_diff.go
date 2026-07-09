package varcore

import "reflect"

// param_diff.go — compare a sensor's returned inline actuals against captured
// document values. Port of param_diff.py / param-diff.ts.

// renderParamValue renders one side of a parameter diff as (text, viaFormat).
// The parameter type's format wins when present (document notation — the only
// rendering conformance can pin), else the shared string/primitive/fallback
// chain.
func renderParamValue(value any, format ParameterFormat) (string, bool) {
	if format != nil {
		return format(value), true
	}
	return renderCellValue(value), false
}

// compareParams compares returned actuals against expected values captured from
// the document. expected, paramSpans, and sourceTexts align 1:1 with returned.
// Structural equality (reflect.DeepEqual) is used so values compare across
// references. Port of compare_params.
func compareParams(returned, expected []any, paramSpans []Span, sourceTexts []string, formats []ParameterFormat) []CellDiff {
	diffs := make([]CellDiff, 0, len(expected))
	for i := range expected {
		ok := reflect.DeepEqual(returned[i], expected[i])
		var format ParameterFormat
		if i < len(formats) {
			format = formats[i]
		}
		actualText, viaFormat := renderParamValue(returned[i], format)
		expectedText := ""
		if i < len(sourceTexts) {
			expectedText = sourceTexts[i]
		} else {
			expectedText, _ = renderParamValue(expected[i], format)
		}
		var span Span
		if i < len(paramSpans) {
			span = paramSpans[i]
		}
		diffs = append(diffs, CellDiff{
			Column:        "arg " + itoa(i+1),
			Span:          span,
			Expected:      expectedText,
			Actual:        actualText,
			OK:            ok,
			ExpectedValue: expected[i],
			ActualValue:   returned[i],
			Formatted:     viaFormat,
		})
	}
	return diffs
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
