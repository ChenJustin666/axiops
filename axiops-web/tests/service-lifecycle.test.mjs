import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import vm from 'node:vm'
import { parse, compileScript, compileTemplate } from '@vue/compiler-sfc'
import ts from 'typescript'
import * as vue from 'vue'

const filename = new URL('../src/views/cicd/project-workspace/components/ServiceLifecycleDialog.vue', import.meta.url)
const { descriptor } = parse(readFileSync(filename, 'utf8'))
assert.deepEqual(compileTemplate({ source: descriptor.template.content, id: 'lifecycle', filename: filename.pathname }).errors, [])
const code = ts.transpileModule(compileScript(descriptor, { id: 'lifecycle' }).content, {
  compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 }
}).outputText
const flush = async () => { for (let i = 0; i < 5; i++) await new Promise(setImmediate) }
const deferred = () => { let resolve; const promise = new Promise(r => { resolve = r }); return { resolve, promise } }
const service = { id: 17, project_id: 19, name: 'risk_server', code: 'risk-server' }
function mount(api, mode = 'delete', confirm = async () => true, prompt = async () => ({ value: service.code })) {
  const module = { exports: {} }, events = []
  vm.runInNewContext(code, { module, exports: module.exports, require(name) {
    if (name === 'vue') return vue
    if (name === '@/api/axiops') return api
    if (name === 'element-plus') return { ElMessage: { success() {} }, ElMessageBox: { confirm, prompt } }
    if (name === '@/composables/useAxiOpsAccess') return { useAxiOpsAccess: () => ({ can: () => true }) }
    return {}
  } })
  const props = vue.reactive({ visible: true, projectId: 19, mode, service })
  let state
  const renderer = vue.createRenderer({ createComment: () => ({}), createText: () => ({}), createElement: () => ({}), insert() {}, remove() {}, setText() {}, setElementText() {}, patchProp() {}, parentNode: () => null, nextSibling: () => null })
  const app = renderer.createApp({ setup() { state = module.exports.default.setup(props, { expose() {}, emit: (...args) => events.push(args) }); return () => null } })
  app.mount({})
  return { state, props, events, unmount: () => app.unmount() }
}

test('deletion check fails closed and stale response cannot authorize a different service', async () => {
  const first = deferred(), second = deferred()
  let calls = 0
  const view = mount({ checkServiceDeletionApi: id => id === 17 ? first.promise : second.promise, deleteServiceApi: async () => { calls++ } })
  await view.state.remove()
  assert.equal(calls, 0)
  view.props.service = { ...service, id: 18 }; await flush()
  second.resolve({ can_delete: false, blockers: [{ kind: 'workflow', id: 1, name: 'workflow' }] }); await flush()
  first.resolve({ can_delete: true, blockers: [] }); await flush()
  await view.state.remove()
  assert.equal(calls, 0)
  assert.equal(view.state.check.value.can_delete, false)
  view.unmount()
})

test('delete rechecks on a server conflict and leaves dialog open with current references', async () => {
  let loads = 0
  const view = mount({
    checkServiceDeletionApi: async () => ++loads === 1 ? { can_delete: true, blockers: [] } : { can_delete: false, blockers: [{ kind: 'build', id: 1, name: 'Build #1' }] },
    deleteServiceApi: async () => { throw new Error('service in use') }
  })
  await flush(); await view.state.remove()
  assert.equal(loads, 2)
  assert.equal(view.state.check.value.can_delete, false)
  assert.equal(view.events.length, 0)
  assert.equal(view.state.error.value, 'service in use')
  view.unmount()
})

test('rename sends only the display name and does not mutate stable service fields', async () => {
  const calls = []
  const view = mount({ renameServiceApi: async (...args) => calls.push(args) }, 'rename')
  await flush()
  view.state.name.value = ' 风控网关 '
  await view.state.rename()
  assert.deepEqual(calls, [[17, '风控网关']])
  assert.equal(service.code, 'risk-server')
  assert.equal(view.events.some(([name]) => name === 'changed'), true)
  view.unmount()
})

test('trash lists deleted services explicitly and canceled restore does not mutate anything', async () => {
  let restored = 0
  const view = mount({
    listAllServicesApi: async (projectId, params) => { assert.equal(projectId, 19); assert.equal(params.status, 'deleted'); return [service] },
    restoreServiceApi: async () => { restored++ }
  }, 'trash', async () => { throw 'cancel' })
  await flush(); await view.state.restore(service)
  assert.equal(restored, 0)
  assert.equal(view.events.length, 0)
  view.unmount()
})

test('permanent delete requires a fresh trash check and matching code', async () => {
  for (const delivery_type of ['k8s_yaml', 'helm']) {
    const calls = []
    const row = { ...service, delivery_type }
    const view = mount({
      listAllServicesApi: async () => [row],
      checkServiceDeletionApi: async id => { calls.push(['check', id]); return { can_purge: true, blockers: [] } },
      purgeServiceApi: async (...args) => calls.push(['purge', ...args])
    }, 'trash', undefined, async (_, __, options) => {
      assert.equal(options.inputValidator('wrong'), '请输入完整服务编码')
      assert.equal(options.inputValidator(service.code), true)
      return { value: service.code }
    })
    await flush(); await view.state.purge(row)
    assert.deepEqual(calls, [['check', 17], ['purge', 17, 'risk-server']])
    assert.equal(view.events.some(([name]) => name === 'changed'), true)
    view.unmount()
  }
})

test('canceled, incorrect or blocked purge never sends a destructive request', async () => {
  for (const scenario of ['blocked', 'cancel', 'wrong']) {
    let purged = 0
    const view = mount({
      listAllServicesApi: async () => [service],
      checkServiceDeletionApi: async () => ({ can_purge: scenario !== 'blocked', blockers: [] }),
      purgeServiceApi: async () => { purged++ }
    }, 'trash', undefined, async () => { if (scenario === 'cancel') throw 'cancel'; return { value: 'wrong' } })
    await flush(); await view.state.purge(service)
    assert.equal(purged, 0)
    assert.equal(view.events.length, 0)
    assert.equal(view.state.saving.value, false)
    view.unmount()
  }
})
