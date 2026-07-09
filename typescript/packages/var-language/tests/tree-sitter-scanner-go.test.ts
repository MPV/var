import { describe, expect, test } from 'vitest'
import { createTreeSitterScanner } from '../src/tree-sitter-scanner.ts'
import { bundleFixture } from './bundle-fixtures.ts'
import { createTestGrammarLoader } from './test-grammar-loader.ts'

// (kind, expression) and parameter-type extraction are proven across every
// bundle and language by extraction-conformance.test.ts. This file covers the
// Go-specific pieces that test doesn't: typed handler-param extraction
// (including multi-name declarations) and interpreted vs raw string decoding.
async function goScanner() {
  return createTreeSitterScanner(createTestGrammarLoader(), ['go'])
}

describe('go dialect', () => {
  test('extracts typed handler params from bundle fixtures', async () => {
    const scanner = await goScanner()

    // 01: func(_ state, n int) — plain, one name per declaration.
    const simple = bundleFixture('01-roman-numerals', '.go')
    const simpleDefs = scanner.discoverStepDefs(simple.name, simple.source)
    expect(simpleDefs.map((d) => d.kind)).toEqual(['stimulus', 'sensor'])
    expect(simpleDefs[0]?.handlerParams?.params).toEqual([
      { name: '_', typeText: 'state' },
      { name: 'n', typeText: 'int' },
    ])

    // 03: func(s state, a, b int) — two names sharing one type declaration.
    const multi = bundleFixture('03-expected-failure', '.go')
    const multiDefs = scanner.discoverStepDefs(multi.name, multi.source)
    expect(multiDefs[0]?.handlerParams?.params).toEqual([
      { name: 's', typeText: 'state' },
      { name: 'a', typeText: 'int' },
      { name: 'b', typeText: 'int' },
    ])
  })

  test('raw strings keep backslashes; interpreted strings decode escapes', async () => {
    const scanner = await goScanner()
    const raw = scanner.discoverStepDefs(
      'a.steps.go',
      'package a\nfunc B(r *R) {\n\tr.Sensor(`a \\d+ and \\.`, func(_ S) any { return nil })\n}\n',
    )
    expect(raw[0]?.expression).toBe('a \\d+ and \\.')
    const interpreted = scanner.discoverStepDefs(
      'a.steps.go',
      'package a\nfunc B(r *R) {\n\tr.Sensor("said \\"hi\\"\\n\\ttab é", func(_ S) any { return nil })\n}\n',
    )
    expect(interpreted[0]?.expression).toBe('said "hi"\n\ttab é')
  })

  test('extracts custom parameter types with a raw-string regexp', async () => {
    const scanner = await goScanner()
    const fixture = bundleFixture('13-custom-parameter-type', '.go')
    const types = scanner.discoverParameterTypes(fixture.name, fixture.source)
    expect(types.map((t) => [t.name, t.regexp])).toEqual([['airport', '[A-Z]{3}']])
  })
})
