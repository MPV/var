package varcore

// expression_params.go — recover a cucumber expression's parameter-type names in
// source order.
//
// Every other port reads these from the compiled expression's AST. The Go
// cucumber-expressions module keeps that AST unexported, so this scans the
// expression directly with the same escape rules the upstream tokenizer uses:
// a backslash escapes the following character (so `\{` and `\}` are literal
// text, not a parameter), and an unescaped `{...}` is a parameter whose inner
// text is the type name. This yields byte-identical parameterTypeNames to the
// AST walk for the cucumber-expression grammar (parameters cannot nest).
func parameterTypeNames(expression string) []string {
	names := make([]string, 0)
	runes := []rune(expression)
	i := 0
	for i < len(runes) {
		c := runes[i]
		if c == '\\' {
			i += 2 // skip the escaped character
			continue
		}
		if c == '{' {
			j := i + 1
			var name []rune
			for j < len(runes) {
				if runes[j] == '\\' {
					if j+1 < len(runes) {
						name = append(name, runes[j+1])
					}
					j += 2
					continue
				}
				if runes[j] == '}' {
					break
				}
				name = append(name, runes[j])
				j++
			}
			names = append(names, string(name))
			i = j + 1
			continue
		}
		i++
	}
	return names
}
