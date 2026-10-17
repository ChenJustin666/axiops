import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import vm from 'node:vm'
import ts from 'typescript'
import { parse, compileScript, compileTemplate } from '@vue/compiler-sfc'
import * as vue from 'vue'
import * as yaml from 'js-yaml'

const transpile = source => ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 } }).outputText
const source = readFileSync(new URL('../src/views/cicd/project-workspace/components/WorkflowRollbackDialog.vue', import.meta.url), 'utf8')
const { descriptor } = parse(source)
const script = compileScript(descriptor, { id: 'rollback-test' })
assert.deepEqual(compileTemplate({ source: descriptor.template.content, filename: 'WorkflowRollbackDialog.vue', id: 'rollback-test', compilerOptions: { bindingMetadata: script.bindings } }).errors, [])
const code = transpile(script.content)
const flush = async () => { for (let i = 0; i < 8; i++) await new Promise(setImmediate) }
const version = { snapshot_id: 30, run_id: 20, run_number: 2, workflow_id: 11, revision_id: 8, images: ['registry/app@sha256:abc'], created_at: '2026-10-15T10:00:00Z', status: 'applied' }
const preview = { token: 'a'.repeat(64), origin_run_id: 40, workflow_id: 11, service_id: 1, service_name: 'server', environment_id: 22, cluster_id: 2, namespace: 'qa', current: { ...version, snapshot_id: 31 }, target: version, current_manifest: 'kind: Deployment\nmetadata: {name: app}\nspec: {replicas: 2}', target_manifest: 'kind: Deployment\nmetadata: {name: app}\nspec: {replicas: 1}', blocked_reason: '', warnings: [] }

function setup(overrides = {}, propsOverrides = {}, storage = new Map()) {
  const pendingStorage = { getItem: key => storage.get(key), setItem: (key, value) => storage.set(key, value), removeItem: key => storage.delete(key) }
  const utility = { exports: {} }
  vm.runInNewContext(transpile(readFileSync(new URL('../src/utils/workflow-rollback.ts', import.meta.url), 'utf8')), { module: utility, exports: utility.exports, require: () => yaml, sessionStorage: pendingStorage })
  const calls = [], events = []
  const api = {
    getRollbackOptionsApi: async () => [1, 2].map(id => ({ service_id: id, name: id === 1 ? 'server' : 'gateway', versions: [version] })),
    previewRollbackApi: async () => structuredClone(preview),
    submitRollbackApi: async (...args) => { calls.push(args); return { workflow_run_id: 99 } },
    ...overrides
  }
  const module = { exports: {} }
  vm.runInNewContext(code, { module, exports: module.exports, require: name => name === 'vue' ? vue : name === '@/api/axiops' ? api : name === '@/utils/workflow-rollback' ? utility.exports : {}, crypto: { randomUUID: () => 'request-id-for-test' } })
  const props = vue.reactive({ visible: true, runId: 40, ...propsOverrides })
  const scope = vue.effectScope()
  const state = scope.run(() => module.exports.default.setup(props, { emit: (...args) => events.push(args), expose() {} }))
  return { state, props, calls, events, utility: utility.exports, storage, stop: () => scope.stop() }
}

test('requires explicit service/version selection and confirmation; sends only selected service', async () => {
  const view = setup(); await flush()
  assert.equal(view.state.selectedServiceId.value, undefined)
  assert.equal(view.state.canSubmit.value, false)
  view.state.selectedServiceId.value = 1; view.state.selectService()
  view.state.snapshotId.value = 30; await view.state.loadPreview()
  await view.state.submit(); assert.equal(view.calls.length, 0)
  view.state.confirmed.value = true; await view.state.submit()
  assert.equal(view.calls.length, 1)
  assert.equal(view.calls[0][0], 30)
  assert.equal(JSON.stringify(view.calls[0][1]), JSON.stringify({ origin_run_id: 40, service_id: 1, expected_token: 'a'.repeat(64) }))
  assert.deepEqual(view.events.at(-2), ['submitted', 99, 11])
  assert.equal(view.storage.size, 0)
  view.stop()
})

test('service detail preselects one service, ignores delayed preview of another service', async () => {
  let resolveOld
  const view = setup({ previewRollbackApi: async (_, sid) => sid === 1 ? new Promise(resolve => { resolveOld = resolve }) : { ...structuredClone(preview), service_id: 2 } }, { serviceId: 1 })
  await flush(); assert.equal(view.state.selectedServiceId.value, 1)
  view.state.snapshotId.value = 30; const first = view.state.loadPreview()
  view.state.selectedServiceId.value = 2; view.state.selectService(); view.state.snapshotId.value = 30
  await view.state.loadPreview(); resolveOld(preview); await first
  assert.equal(view.state.preview.value.service_id, 2)
  assert.equal(view.state.confirmed.value, false)
  view.stop()
})

test('blocks duplicate clicks and safely resumes an uncertain request after reopening', async () => {
  const storage = new Map(), sent = []
  let release
  const view = setup({ submitRollbackApi: (...args) => { sent.push(args); return new Promise((_, reject) => { release = () => reject({ code: 504, message: 'gateway timeout' }) }) } }, { serviceId: 1 }, storage)
  await flush(); view.state.snapshotId.value = 30; await view.state.loadPreview(); view.state.confirmed.value = true
  const first = view.state.submit(); await view.state.submit(); assert.equal(sent.length, 1)
  release(); await first; assert.equal(view.state.uncertain.value, true)
  const stored = [...storage.values()][0]; assert.ok(stored); assert.ok(!stored.includes('kind: Deployment'))
  view.stop()
  const second = setup({}, {}, storage); await flush()
  assert.equal(second.state.uncertain.value, true)
  assert.equal(second.state.selectedServiceId.value, 1)
  await second.state.submit()
  assert.equal(JSON.stringify(second.calls[0]), JSON.stringify(sent[0]))
  assert.equal(storage.size, 0); second.stop()
})

test('stale server validation clears confirmation and requires a new preview', async () => {
  const view = setup({ submitRollbackApi: async () => { throw { code: 400, message: '版本已变化' } } }, { serviceId: 1 })
  await flush(); view.state.snapshotId.value = 30; await view.state.loadPreview(); view.state.confirmed.value = true
  await view.state.submit()
  assert.equal(view.state.error.value, '版本已变化')
  assert.equal(view.state.preview.value, null)
  assert.equal(view.state.canSubmit.value, false)
  assert.equal(view.state.uncertain.value, false)
  assert.equal(view.storage.size, 0); view.stop()
})

test('shows semantic configuration differences and treats network/server errors as uncertain', async () => {
  const view = setup(); await flush()
  const changes = view.utility.rollbackConfigChanges(preview.current_manifest, preview.target_manifest)
  assert.equal(changes.length, 1)
  assert.equal(changes[0].path, 'Deployment/app.spec.replicas')
  assert.equal(changes[0].before, '2'); assert.equal(changes[0].after, '1')
  for (const code of [500, 502, 503, 504, 50002, 50001, 'ERR_NETWORK', undefined]) assert.equal(view.utility.rollbackResultUncertain({ code }), true)
  for (const code of [400, 401, 403, 404, 50004, 10003]) assert.equal(view.utility.rollbackResultUncertain({ code }), false)
  assert.equal(view.utility.rollbackResultUncertain({ code: 400, requestUncertain: true }), true)
  assert.equal(view.utility.rollbackResultUncertain({ code: 408 }), true)
  view.stop()
})
