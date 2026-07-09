import type { Node } from 'web-tree-sitter'
import type { HandlerParam, HandlerParams } from '../step-defs.ts'
import { decodeSimpleOrHexEscape } from './escape-decode.ts'
import { type LanguageSpec, toRange } from './types.ts'

// Verified against tree-sitter-go 0.25.0 and all 15 conformance bundles
// (2026-07-09). A step def is a method call on the injected Registrar —
// `r.Stimulus("expr", func(...) ...)` / `r.Sensor(...)` — i.e. a call_expression
// whose function is a selector_expression with field "Stimulus"/"Sensor", first
// argument the expression string, and a func_literal handler. Go's exported
// methods are capitalized; the scanner lowercases the captured name to the
// 'stimulus'|'sensor' StepKind.
const STEP_DEFINITION_QUERY = `
(call_expression
  function: (selector_expression
    field: (field_identifier) @function-name)
  arguments: (argument_list
    .
    [(interpreted_string_literal) (raw_string_literal)] @expression
    .
    (func_literal) @handler)
  (#match? @function-name "^(Stimulus|Sensor)$")
) @root
`

// A custom parameter type is `r.DefineParameterType("name", "regexp", …)`: name
// the first string, regexp the second. Trailing parse/format args are not
// anchored, so they're ignored.
const PARAMETER_TYPE_QUERY = `
(call_expression
  function: (selector_expression
    field: (field_identifier) @function-name)
  arguments: (argument_list
    .
    [(interpreted_string_literal) (raw_string_literal)] @name
    .
    [(interpreted_string_literal) (raw_string_literal)] @regexp-value)
  (#eq? @function-name "DefineParameterType")
) @root
`

// Go interpreted-string escapes (\a \b \f \n \r \t \v \\ \" plus \x, \u, \U,
// and octal handled below). Raw strings (backtick) have no escapes at all.
const SIMPLE_ESCAPES: Readonly<Record<string, string>> = {
  '\\': '\\',
  '"': '"',
  "'": "'",
  a: '\x07',
  b: '\b',
  f: '\f',
  n: '\n',
  r: '\r',
  t: '\t',
  v: '\v',
}

function decodeEscape(text: string): string {
  const body = text.slice(1) // drop the leading backslash
  const simple = decodeSimpleOrHexEscape(body, SIMPLE_ESCAPES)
  if (simple !== undefined) return simple
  if (body.startsWith('u') && body.length === 5) {
    return String.fromCodePoint(Number.parseInt(body.slice(1), 16))
  }
  if (body.startsWith('U') && body.length === 9) {
    return String.fromCodePoint(Number.parseInt(body.slice(1), 16))
  }
  if (/^[0-7]{3}$/.test(body)) {
    return String.fromCodePoint(Number.parseInt(body, 8))
  }
  // Unknown escape: keep the character, drop the backslash.
  return body
}

// interpreted_string_literal children are interpreted_string_literal_content and
// escape_sequence siblings; raw_string_literal wraps raw_string_literal_content
// verbatim (so a pattern like `£\d+\.\d{2}` survives untouched).
function decodeString(node: Node): string {
  if (node.type === 'raw_string_literal') {
    return node.namedChildren
      .filter((c): c is Node => c?.type === 'raw_string_literal_content')
      .map((c) => c.text)
      .join('')
  }
  let out = ''
  for (const child of node.children) {
    if (child?.type === 'interpreted_string_literal_content') {
      out += child.text
    } else if (child?.type === 'escape_sequence') {
      out += decodeEscape(child.text)
    }
  }
  return out
}

/* jscpd:ignore-start — shared param-extraction shape; per-dialect on purpose */
function extractHandlerParams(handlerNode: Node): HandlerParams | undefined {
  const parameters = handlerNode.childForFieldName('parameters')
  const decls = parameters?.namedChildren.filter((p): p is Node => p?.type === 'parameter_declaration') ?? []
  const structured: HandlerParam[] = []
  let first: Node | undefined
  let last: Node | undefined
  for (const decl of decls) {
    const typeNode = decl.childForFieldName('type')
    const typeText = typeNode?.text ?? ''
    // A declaration may bind several names to one type: `func(a, b int)`.
    const names = decl.namedChildren.filter((c): c is Node => c?.type === 'identifier')
    for (const name of names) {
      structured.push({ name: name.text, typeText })
      first ??= name
      last = name
    }
  }
  if (structured.length === 0 || !first || !last) return undefined
  return { range: toRange(first, last), params: structured }
}
/* jscpd:ignore-end */

export const goSpec: LanguageSpec = {
  stepDefQuery: STEP_DEFINITION_QUERY,
  parameterTypeQuery: PARAMETER_TYPE_QUERY,
  decodeString,
  extractHandlerParams: (handlerNode) =>
    handlerNode.type === 'func_literal' ? extractHandlerParams(handlerNode) : undefined,
  resolveRegexp: (node) =>
    node.type === 'interpreted_string_literal' || node.type === 'raw_string_literal'
      ? decodeString(node)
      : node.text,
}
