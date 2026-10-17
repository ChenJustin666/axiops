import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import vm from 'node:vm'
import ts from 'typescript'
import * as vue from 'vue'
import { parse, compileScript, compileTemplate } from '@vue/compiler-sfc'

const file = new URL('../src/views/cicd/environment-detail/components/WorkloadRestartDialog.vue', import.meta.url)
const { descriptor } = parse(readFileSync(file, 'utf8'))
assert.deepEqual(compileTemplate({ source: descriptor.template.content, filename: file.pathname, id: 'restart' }).errors, [])
const code = ts.transpileModule(compileScript(descriptor, { id: 'restart' }).content, { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 } }).outputText
const flush = async () => { for (let i = 0; i < 6; i++) await new Promise(setImmediate) }
const deferred = () => { let resolve; const promise = new Promise(r => { resolve = r }); return { resolve, promise } }
const target = (id = 1) => ({ id, name: `env-${id}`, cluster_id: id + 10, namespace: `ns-${id}` })
const preview = (id = 1) => ({ environment_id: id, cluster_id: id + 10, namespace: `ns-${id}`, kind: 'Deployment', name: 'app', token: 'token', images: [], is_production: false })
function mount(api = {}) {
  const props = vue.reactive({ modelValue: true, environment: target(), workload: { kind: 'Deployment', name: 'app' } })
  const events = [], requests = [], module = { exports: {} }
  const defaults = { previewEnvRestartApi: async id => preview(id), submitEnvRestartApi: async (...args) => { requests.push(args); return { id: 90 } } }
  vm.runInNewContext(code, { module, exports: module.exports, crypto: { randomUUID: () => 'unique-key' }, require(name) {
    if (name === 'vue') return vue
    if (name === '@/api/axiops/env-resource') return { ...defaults, ...api }
    return {}
  } })
  let state
  const renderer = vue.createRenderer({ createComment: () => ({}), createText: () => ({}), createElement: () => ({}), insert() {}, remove() {}, setText() {}, setElementText() {}, patchProp() {}, parentNode: () => null, nextSibling: () => null })
  const app = renderer.createApp({ setup() { state = module.exports.default.setup(props, { expose() {}, emit: (...args) => events.push(args) }); return () => null } })
  app.mount({})
  return { state, props, events, requests, unmount: () => app.unmount() }
}

test('restart preview rejects mismatched cluster and never submits it', async () => {
  const view = mount({ previewEnvRestartApi: async () => ({ ...preview(), cluster_id: 999 }) })
  await flush()
  assert.match(view.state.error.value, /不一致/)
  assert.equal(view.state.preview.value, null)
  await view.state.submit()
  assert.equal(view.requests.length, 0)
  view.unmount()
})

test('changing environment discards a delayed preview from the old environment', async () => {
  const old = deferred()
  const view = mount({ previewEnvRestartApi: id => id === 1 ? old.promise : Promise.resolve(preview(id)) })
  view.props.environment = target(2)
  await flush()
  old.resolve(preview(1)); await flush()
  assert.equal(view.state.preview.value.environment_id, 2)
  await view.state.submit()
  assert.equal(view.requests[0][0], 2)
  view.unmount()
})

test('production requires confirmation and uncertain submission reuses its original request key', async () => {
  const requests = []
  const view = mount({ previewEnvRestartApi: async () => ({ ...preview(), is_production: true }), submitEnvRestartApi: async (...args) => { requests.push(args); if (requests.length === 1) throw Error('response lost'); return { id: 91 } } })
  await flush(); await view.state.submit()
  assert.equal(requests.length, 0)
  view.state.productionConfirmed.value = true
  await view.state.submit()
  assert.equal(view.state.submittedUncertain.value, true)
  await view.state.submit()
  assert.equal(requests.length, 2)
  assert.deepEqual(requests[0], requests[1])
  assert.ok(view.events.some(([name, id]) => name === 'submitted' && id === 91))
  view.unmount()
})

test('submit cannot duplicate in flight and an old response never opens a run in another environment', async () => {
  const pending = deferred(), requests = []
  const view = mount({ submitEnvRestartApi: (...args) => { requests.push(args); return pending.promise } })
  await flush()
  const first = view.state.submit()
  await view.state.submit()
  assert.equal(requests.length, 1)
  view.props.environment = target(2); await flush()
  pending.resolve({ id: 92 }); await first; await flush()
  assert.equal(view.events.some(([name]) => name === 'submitted'), false)
  assert.equal(view.state.preview.value.environment_id, 2)
  view.unmount()
})
