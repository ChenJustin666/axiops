import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import vm from 'node:vm'
import ts from 'typescript'
import { parse, compileScript, compileTemplate } from '@vue/compiler-sfc'
import * as vue from 'vue'

const file = '../src/views/cicd/project-workspace/components/ProductionReleaseDialog.vue'
const { descriptor } = parse(readFileSync(new URL(file, import.meta.url), 'utf8'))
const script = compileScript(descriptor, { id: 'production-test' })
assert.deepEqual(compileTemplate({ source: descriptor.template.content, filename: file, id: 'production-test', compilerOptions: { bindingMetadata: script.bindings } }).errors, [])
const code = ts.transpileModule(script.content, { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 } }).outputText
const flush = async () => { for (let i = 0; i < 8; i++) await new Promise(setImmediate) }
const candidate = (id, sid, deployed = true, eid = 1) => ({ id: 0, build_id: id, service_id: sid, environment_id: eid, build_number: id, commit: 'abcdef123', image_ref: `registry/app:test-${id}@sha256:${id}`, method: 'build', deployment_status: deployed ? 'succeeded' : '', built_at: '2026-10-16T02:00:00Z', branch: 'test' })
const options = () => ({ environment_id: 2, environment_name: '生产环境', policy: { environment_id: 2, source_environment_id: 0, revision: 0 }, image_sources: [{ id: 1, name: '测试环境', code: 'qa' }, { id: 2, name: '生产环境', code: 'prod' }], services: [1, 2].map(sid => ({ service_id: sid, name: `service-${sid}`, candidates: [], images: [candidate(sid * 10 + 1, sid, false), candidate(sid * 10, sid)] })) })
function setup(overrides = {}, extra = {}, storage = new Map()) {
  const calls = [], events = [], routes = []
  const api = {
    getProductionOptionsApi: async () => options(),
    getProductionImagePreviewApi: async (_, id) => ({ token: 'a'.repeat(64), candidate: candidate(id, Math.floor(id / 10)), blocked_reason: '', requires_approval: false }),
    submitProductionApi: async (...args) => { calls.push(args); return { workflow_run_id: 99 } }, ...overrides
  }
  const module = { exports: {} }
  vm.runInNewContext(code, { module, exports: module.exports, require: name => name === 'vue' ? vue : name === 'vue-router' ? { useRouter: () => ({ push(route) { routes.push(route) } }) } : name === '@/api/axiops' ? api : { rollbackConfigChanges: () => [], rollbackResultUncertain: e => e.code === 504 }, sessionStorage: { getItem: k => storage.get(k), setItem: (k, v) => storage.set(k, v), removeItem: k => storage.delete(k) }, crypto: { randomUUID: () => 'test-request' } })
  const props = vue.reactive({ visible: true, workflowId: 3, projectId: 4, ...extra })
  const scope = vue.effectScope()
  const state = scope.run(() => module.exports.default.setup(props, { emit: (...e) => events.push(e), expose() {} }))
  return { state, props, calls, events, routes, storage, stop: () => scope.stop() }
}

test('defaults to test and one service; recommends deployed image ahead of newer build, direct publish needs no manual acceptance', async () => {
  const v = setup(); await flush()
  assert.equal(v.state.sourceId.value, 1)
  assert.equal(v.state.advanced.value, false)
  assert.equal(v.state.mode.value, 'image')
  assert.equal(JSON.stringify(v.state.selectedIds.value), '[1]')
  assert.equal(v.state.chosen[1], 10); assert.equal(v.state.canSubmit.value, true)
  await v.state.submit()
  assert.equal(v.calls[0][1][0].build_id, 10); assert.equal(v.calls[0][1][0].verification_id, undefined)
  assert.equal(v.storage.size, 0); v.stop()
})

test('collapsing advanced keeps the chosen source and image; refresh preserves the advanced state', async () => {
  const v = setup(); await flush()
  v.state.advanced.value = true
  v.state.sourceId.value = 0; await v.state.sourceChanged()
  v.state.chosen[1] = 11; await v.state.loadPreview(1)
  v.state.advanced.value = false
  await v.state.refreshVersions()
  assert.equal(v.state.advanced.value, false)
  assert.equal(v.state.sourceId.value, 0)
  assert.equal(v.state.chosen[1], 11)
  v.state.advanced.value = true; await v.state.refreshVersions()
  assert.equal(v.state.advanced.value, true)
  await v.state.load()
  assert.equal(v.state.advanced.value, false); v.stop()
})

test('switching to build immediately opens branch selection with the same service order', async () => {
  const v = setup(); await flush()
  v.state.selectedIds.value = [2, 1]
  v.state.mode.value = 'build'; v.state.modeChanged()
  assert.equal(JSON.stringify(v.events.find(e => e[0] === 'build')[1]), '[2,1]')
  assert.equal(v.calls.length, 0); v.stop()
})

test('returning from build preserves services and rejects removed services instead of substituting', async () => {
  const v = setup({}, { initialServiceIds: [2, 1] }); await flush()
  assert.equal(JSON.stringify(v.state.selectedIds.value), '[2,1]')
  assert.equal(v.state.mode.value, 'image'); v.stop()
  const missing = setup({}, { initialServiceIds: [2, 99] }); await flush()
  assert.match(missing.state.error.value, /所选服务已变化/)
  assert.equal(missing.state.canSubmit.value, false); missing.stop()
})
test('without deployment history newest successful build is selectable but never labeled test passed', async () => {
  const v = setup({ getProductionOptionsApi: async () => { const o = options(); o.services.forEach(s => s.images.forEach(c => c.deployment_status = '')); return o } }); await flush()
  assert.equal(v.state.chosen[1], 11); assert.equal(v.state.canSubmit.value, true)
  assert.equal(v.state.candidate(1).method, 'build'); v.stop()
})
test('source is an editable filter; production own artifacts also selectable', async () => {
  const v = setup({ getProductionOptionsApi: async () => { const o = options(); o.services[0].images.push(candidate(12, 1, true, 2)); return o } }); await flush()
  v.state.sourceId.value = 2; await v.state.sourceChanged()
  assert.equal(v.state.chosen[1], 12); assert.equal(v.state.previews[1].candidate.build_id, 12)
  assert.equal(v.state.canSubmit.value, true); v.stop()
})
test('one environment, no policy and no images supports explicit build without source requirement', async () => {
  const v = setup({ getProductionOptionsApi: async () => { const o = options(); o.image_sources = [o.image_sources[1]]; o.services.forEach(s => s.images = []); return o } }); await flush()
  assert.equal(v.state.canSubmit.value, false)
  v.state.mode.value = 'build'; v.state.openBuild()
  assert.equal(JSON.stringify(v.events.find(e => e[0] === 'build')[1]), '[1]')
  assert.equal(v.calls.length, 0); v.stop()
})
test('optional refresh preserves version, source and service order', async () => {
  const v = setup(); await flush(); v.state.selectedIds.value = [2, 1]; v.state.chosen[1] = 11
  await v.state.selectServices(); await v.state.loadPreview(1); await v.state.refreshVersions()
  assert.equal(JSON.stringify(v.state.selectedIds.value), '[2,1]')
  assert.equal(v.state.chosen[1], 11); assert.equal(v.state.previews[1].candidate.build_id, 11); v.stop()
})
test('missing selected artifact after refresh blocks publish without replacing it', async () => {
  let removed = false
  const v = setup({ getProductionOptionsApi: async () => { const o = options(); if (removed) o.services[0].images = o.services[0].images.filter(c => c.build_id !== 11); return o } }); await flush()
  v.state.chosen[1] = 11; await v.state.loadPreview(1); removed = true; await v.state.refreshVersions()
  assert.equal(v.state.chosen[1], undefined); assert.equal(v.state.previews[1], undefined)
  assert.match(v.state.previewErrors[1], /原选版本已失效/); await v.state.submit(); assert.equal(v.calls.length, 0); v.stop()
})
test('delayed previews cannot replace newly selected version or source', async () => {
  let resolve
  const v = setup({ getProductionImagePreviewApi: async (_, id) => id === 10 ? new Promise(r => { resolve = r }) : { token: 'b'.repeat(64), candidate: candidate(id, 1) } }); await flush()
  v.state.chosen[1] = 11; await v.state.loadPreview(1)
  resolve({ token: 'a'.repeat(64), candidate: candidate(10, 1) }); await flush()
  assert.equal(v.state.previews[1].candidate.build_id, 11); v.stop()
})
test('source switch discards stale preview even when new source is empty', async () => {
  let resolve
  const v = setup({ getProductionImagePreviewApi: async () => new Promise(r => { resolve = r }) }); await flush()
  v.state.sourceId.value = 2; await v.state.sourceChanged()
  resolve({ token: 'a'.repeat(64), candidate: candidate(10, 1) }); await flush()
  assert.equal(v.state.previews[1], undefined); assert.equal(v.state.canSubmit.value, false); v.stop()
})
test('multiple services retain explicit order and one build per service', async () => {
  const v = setup(); await flush(); v.state.selectedIds.value = [1, 2]; await v.state.selectServices()
  v.state.move(2, -1); await v.state.submit()
  assert.equal(JSON.stringify(v.calls[0][1].map(p => p.build_id)), '[20,10]'); v.stop()
})
test('duplicate clicks and uncertain response recover original request after reload; no new build allowed', async () => {
  const storage = new Map(), sent = []; let reject
  const v = setup({ submitProductionApi: (...args) => { sent.push(args); return new Promise((_, r) => { reject = r }) } }, {}, storage); await flush()
  const first = v.state.submit(); await v.state.submit(); assert.equal(sent.length, 1)
  reject({ code: 504 }); await first; assert.equal(v.state.uncertain.value, true)
  v.state.openBuild(); assert.equal(v.events.some(e => e[0] === 'build'), false); v.stop()
  const second = setup({}, {}, storage); await flush(); assert.equal(second.state.uncertain.value, true)
  await second.state.submit(); assert.equal(JSON.stringify(second.calls[0]), JSON.stringify(sent[0])); assert.equal(storage.size, 0); second.stop()
})
test('server rejection clears stale preview; retry requires a fresh configuration check', async () => {
  const v = setup({ submitProductionApi: async () => { throw { code: 400, message: '配置已变化' } } }); await flush()
  await v.state.submit(); assert.equal(v.state.previews[1], undefined); assert.equal(v.state.canSubmit.value, false)
  await v.state.loadPreview(1); assert.ok(v.state.previews[1]); v.stop()
})
test('invalid direct verification shortcut fails closed instead of choosing different image', async () => {
  const v = setup({}, { serviceId: 1, verificationId: 999 }); await flush()
  assert.ok(v.state.error.value); await v.state.submit(); assert.equal(v.calls.length, 0); v.stop()
})
test('all source filter deduplicates same image deployed to several environments', async () => {
  const v = setup({ getProductionOptionsApi: async () => { const o = options(); o.services[0].images.push(candidate(10, 1, true, 2)); return o } }); await flush()
  v.state.sourceId.value = 0; await v.state.sourceChanged()
  assert.equal(v.state.images(1).length, 2); v.stop()
})

test('configuration shortcut pins the selected service and actual target environment', async () => {
  const v=setup(); await flush()
  v.state.configure(1)
  assert.equal(JSON.stringify(v.events.find(e=>e[0]==='configure')), '["configure",1,2]')
  assert.equal(v.calls.length,0);v.stop()
})
