import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import vm from 'node:vm'
import { parse, compileScript } from '@vue/compiler-sfc'
import ts from 'typescript'
import * as vue from 'vue'

const { descriptor } = parse(readFileSync(new URL('../src/views/cicd/projects/index.vue', import.meta.url), 'utf8'))
const code = ts.transpileModule(compileScript(descriptor, { id: 'project-list-test' }).content, { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 } }).outputText
function setup(api) {
  const module = { exports: {} }, routes = []
  let mounted, activated
  vm.runInNewContext(code, { module, exports: module.exports, require: name => {
    if (name === 'vue') return { ...vue, onMounted: f => mounted = f, onActivated: f => activated = f }
    if (name === 'vue-router') return { useRouter: () => ({ push: async path => routes.push(path) }) }
    if (name === 'element-plus') return { ElMessage: { success() {} } }
    if (name === '@/composables/useAxiOpsAccess') return { useAxiOpsAccess: () => ({ load: async () => {}, can: () => true }) }
    if (name === '@/api/axiops') return api
    return {}
  } })
  const state = module.exports.default.setup({}, { expose() {} })
  return { state, routes, mount: () => mounted(), activate: () => activated() }
}
test('created project is refreshed before navigation and cached return refreshes again', async () => {
  const list = []; let reads = 0
  const { state, routes, mount, activate } = setup({ listProjectsApi: async () => { reads++; return { list: [...list], total: list.length } }, createProjectApi: async () => { list.push({ id: 20, name: 'New', delivery_type: 'k8s_yaml' }); return { id: 20 } } })
  await mount()
  state.formRef.value = { validate: async () => true }
  state.pagination.page = 4; state.filters.keyword = 'old'; state.projectCategory.value = 'host'
  await state.submitCreate()
  assert.equal(state.projects.value[0].id, 20)
  assert.equal(state.pagination.page, 1)
  assert.equal(state.filters.keyword, '')
  assert.equal(state.projectCategory.value, 'all')
  assert.match(routes[0], /projects\/20\?init=1/)
  list.push({ id: 21 }); activate(); await new Promise(resolve => setImmediate(resolve))
  assert.equal(state.projects.value.length, 2)
  assert.equal(reads, 3)
})
test('late list response cannot hide a newer list', async () => {
  const pending = []
  const { state } = setup({ listProjectsApi: () => new Promise(resolve => pending.push(resolve)) })
  const old = state.fetchProjects(), current = state.fetchProjects()
  pending[1]({ list: [{ id: 20 }], total: 1 }); await current
  pending[0]({ list: [], total: 0 }); await old
  assert.equal(state.projects.value[0].id, 20)
})
test('failed initial read can recover on cached return; successful creation is not repeated for a read failure', async () => {
  let fail = true, creates = 0
  const { state, routes, mount, activate } = setup({ listProjectsApi: async () => { if (fail) throw Error('read failed'); return { list: [{ id: 20 }], total: 1 } }, createProjectApi: async () => { creates++; return { id: 20 } } })
  await assert.rejects(mount(), /read failed/)
  fail = false; activate(); await new Promise(resolve => setImmediate(resolve))
  assert.equal(state.projects.value.length, 1)
  fail = true; state.formRef.value = { validate: async () => true }
  await state.submitCreate()
  assert.equal(creates, 1)
  assert.equal(routes.length, 1)
})

test('deletion defaults to permanent with exact name protection and refreshes the list', async () => {
  const project = { id: 20, name: 'Test', code: 'test', status: 'active' }
  const calls = []; let list = [project]
  const { state, mount } = setup({
    listProjectsApi: async () => ({ list: [...list], total: list.length }),
    checkProjectDeletionApi: async () => ({ project_id: 20, project_name: 'Test', can_delete: true, items: [], blockers: [] }),
    deleteProjectApi: async (...args) => { calls.push(args); list = [] }
  })
  await mount(); state.openDelete(project); await new Promise(resolve => setImmediate(resolve))
  assert.equal(state.preserveHistory.value, false)
  state.deleteName.value = ' Test'; await state.submitDelete(); assert.equal(calls.length, 0)
  state.deleteName.value = 'Test'; await state.submitDelete()
  assert.deepEqual(calls, [[20, 'Test', true]])
  assert.equal(state.projects.value.length, 0)
  assert.equal(state.pagination.total, 0)
  assert.equal(state.deleteVisible.value, false)
})

test('history retention is explicit and archived projects can be purged', async () => {
  const calls = []
  const project = { id: 20, name: 'Test', status: 'active' }
  const { state } = setup({
    listProjectsApi: async () => ({ list: [], total: 0 }),
    checkProjectDeletionApi: async () => ({ project_id: 20, project_name: 'Test', can_delete: true, items: [], blockers: [] }),
    deleteProjectApi: async (...args) => calls.push(args)
  })
  state.openDelete(project); await new Promise(resolve => setImmediate(resolve))
  state.preserveHistory.value = true; state.deleteName.value = 'Test'; await state.submitDelete()
  assert.deepEqual(calls[0], [20, 'Test', false])
  state.openDelete({ ...project, status: 'deleted' }); await new Promise(resolve => setImmediate(resolve))
  assert.equal(state.preserveHistory.value, false)
  state.deleteName.value = 'Test'; await state.submitDelete()
  assert.deepEqual(calls[1], [20, 'Test', true])
})

test('blocked or failed deletion cannot remove the project or keep an old confirmation', async () => {
  const project = { id: 20, name: 'Test', status: 'active' }
  let blocked = true; let deletes = 0
  const { state, mount } = setup({
    listProjectsApi: async () => ({ list: [project], total: 1 }),
    checkProjectDeletionApi: async () => ({ project_id: 20, project_name: 'Test', can_delete: !blocked, items: [], blockers: [] }),
    deleteProjectApi: async () => { deletes++; throw Error('changed after preview') }
  })
  await mount(); state.openDelete(project); await new Promise(resolve => setImmediate(resolve))
  state.deleteName.value = 'Test'; await state.submitDelete(); assert.equal(deletes, 0)
  blocked = false; await state.refreshDeletionCheck(); state.deleteName.value = 'Test'; await state.submitDelete()
  assert.equal(deletes, 1)
  assert.equal(state.deleteName.value, '')
  assert.equal(state.projects.value.length, 1)
  assert.equal(state.deleteVisible.value, true)
  assert.equal(state.deleting.value, false)
})
