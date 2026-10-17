import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import vm from 'node:vm'
import ts from 'typescript'

const module = { exports: {} }
vm.runInNewContext(ts.transpileModule(readFileSync(new URL('../src/utils/workflow-actor.ts', import.meta.url), 'utf8'), {
  compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 }
}).outputText, { module, exports: module.exports })
const { workflowActorLabel } = module.exports

test('shows recorded execution account and preserves unknown identities', () => {
  assert.equal(workflowActorLabel({ trigger_user_id: 7, trigger_username: ' deployer ' }), 'deployer')
  assert.equal(workflowActorLabel({ trigger_user_id: 7 }), '账号 #7')
  assert.equal(workflowActorLabel({ trigger_user_id: 7, trigger_username: ' ' }), '账号 #7')
  assert.equal(workflowActorLabel({ trigger_user_id: 0 }), '系统')
  assert.equal(workflowActorLabel(null), '—')
})
