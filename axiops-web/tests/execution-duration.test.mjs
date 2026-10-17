import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import vm from 'node:vm'
import ts from 'typescript'
const module = { exports: {} }
vm.runInNewContext(ts.transpileModule(readFileSync(new URL('../src/utils/execution-duration.ts', import.meta.url), 'utf8'), {
  compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 }
}).outputText, { module, exports: module.exports })
const { formatExecutionDuration } = module.exports

test('elapsed time switches units at minute/hour boundaries using real instants', () => {
  const start = '2026-10-15T17:00:00+08:00', now = Date.parse(start)
  for (const [seconds, expected] of [[0, '0 秒'], [59, '59 秒'], [60, '1 分 0 秒'], [125, '2 分 5 秒'], [3600, '1 小时 0 分 0 秒'], [3725, '1 小时 2 分 5 秒']]) {
    assert.equal(formatExecutionDuration(start, undefined, now + seconds * 1000), expected)
  }
  assert.equal(formatExecutionDuration(start, '2026-10-15T09:02:05Z', now + 9999999), '2 分 5 秒')
  assert.equal(formatExecutionDuration(), '-')
  assert.equal(formatExecutionDuration('invalid'), '-')
  assert.equal(formatExecutionDuration(start, 'invalid'), '-')
})
