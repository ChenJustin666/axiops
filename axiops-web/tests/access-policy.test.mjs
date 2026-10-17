import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import vm from 'node:vm'
import ts from 'typescript'
import * as vue from 'vue'

function readModule(path, require = () => ({})) {
  const source = readFileSync(new URL(path, import.meta.url), 'utf8')
  const code = ts.transpileModule(source, {
    compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 }
  }).outputText
  const module = { exports: {} }
  vm.runInNewContext(code, { module, exports: module.exports, require })
  return module.exports
}
const { canAccess } = readModule('../src/utils/access-policy.ts')
const { createAccessState } = readModule('../src/composables/axiops-access-state.ts', (name) =>
  name === 'vue' ? vue : {}
)
const cases = JSON.parse(
  readFileSync(
    new URL('../../internal/shared/authz/testdata/policy_cases.json', import.meta.url),
    'utf8'
  )
)
for (const c of cases) {
  test(`Go/TypeScript policy parity: ${c.name}`, () => {
    const r = c.request
    assert.equal(
      canAccess(c.policies, r.Resource, r.Action, {
        project_id: r.ProjectID,
        service_id: r.ServiceID,
        environment_id: r.EnvironmentID,
        cluster_id: r.ClusterID,
        workflow_id: r.WorkflowID,
        build_id: r.BuildID,
        environment: r.Environment
      }),
      c.want
    )
  })
}
const global = [
  {
    resource: '*',
    action: '*',
    effect: 'allow',
    scope_type: 'platform',
    scope_id: 0,
    environment: '*'
  }
]
test('row identity and project context match backend requests; unknown scope types cannot grant', () => {
  const p = [{ ...global[0], scope_type: 'cluster', scope_id: 50 }]
  assert.equal(canAccess(p, 'cluster', 'update', { type: 'cluster', id: 50 }), true)
  assert.equal(canAccess(p, 'cluster', 'update', { type: 'cluster', id: 51 }), false)
  assert.equal(
    canAccess([{ ...p[0], scope_type: 'unknown' }], 'cluster', 'update', {
      type: 'unknown',
      id: 50
    }),
    false
  )
  assert.equal(
    canAccess([{ ...p[0], scope_type: 'project' }], 'service', 'update', {
      type: 'project',
      project_id: 50
    }),
    true
  )
})
test('concurrent policy reads share a request, refresh expires after one minute and failures clear old rights', async () => {
  let now = 1,
    calls = 0
  const state = createAccessState(() => now)
  const read = async () => {
    calls++
    return global
  }
  await Promise.all([state.load(read), state.load(read)])
  assert.equal(calls, 1)
  assert.equal(canAccess(state.policies.value, 'cluster', 'delete'), true)
  now = 60_000
  await state.load(read)
  assert.equal(calls, 1)
  now = 60_001
  await state.load(read)
  assert.equal(calls, 2)
  await assert.rejects(
    state.load(async () => {
      throw Error('permissions unavailable')
    }, true)
  )
  assert.equal(state.loaded.value, false)
  assert.equal(canAccess(state.policies.value, 'cluster', 'delete'), false)
  await state.load(read)
  assert.equal(state.loaded.value, true)
})
test('logout or account switch discards delayed administrator policies without replacing the new user', async () => {
  const state = createAccessState()
  let resolveOld
  const old = state.load(
    () =>
      new Promise((resolve) => {
        resolveOld = resolve
      })
  )
  await Promise.resolve()
  state.reset()
  assert.equal(state.loaded.value, false)
  await state.load(async () => [])
  resolveOld(global)
  await old
  assert.equal(state.loaded.value, true)
  assert.equal(state.policies.value.length, 0)
})
test('forced refresh supersedes a stale read and synchronous loader errors remain retryable', async () => {
  const state = createAccessState()
  let resolveOld
  const old = state.load(
    () =>
      new Promise((resolve) => {
        resolveOld = resolve
      })
  )
  await Promise.resolve()
  await state.load(async () => [], true)
  resolveOld(global)
  await old
  assert.equal(state.policies.value.length, 0)
  await assert.rejects(
    state.load(() => {
      throw Error('sync failure')
    }, true)
  )
  await state.load(async () => global)
  assert.equal(state.loaded.value, true)
})
