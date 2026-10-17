import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import vm from 'node:vm'
import { parse, compileScript, compileTemplate } from '@vue/compiler-sfc'
import ts from 'typescript'
import * as vue from 'vue'
import * as yaml from 'js-yaml'
const transpile = source => ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 } }).outputText
const utils = { exports: {} }
vm.runInNewContext(transpile(readFileSync(new URL('../src/utils/helm-service-config.ts', import.meta.url), 'utf8')), { module: utils, exports: utils.exports, require: () => yaml })
const filename = new URL('../src/views/cicd/project-workspace/components/HelmRepositorySync.vue', import.meta.url)
const { descriptor } = parse(readFileSync(filename, 'utf8'))
assert.deepEqual(compileTemplate({ source: descriptor.template.content, id: 'git-chart', filename: filename.pathname }).errors, [])
const code = transpile(compileScript(descriptor, { id: 'git-chart' }).content)
const before = [{ path: 'Chart.yaml', content: 'version: 1.0.0' }, { path: 'templates/old.yaml', content: 'old' }]
const after = [{ path: 'Chart.yaml', content: 'version: 1.1.0' }, { path: 'templates/new.yaml', content: '{{ .Release.Name }}' }]
const flush = async () => { for (let i = 0; i < 6; i++) await new Promise(setImmediate) }
function mount(api = {}) {
 const events = [], calls = [], module = { exports: {} }
 const preview = { commit_id: 'new-commit', current_revision_id: 12, local_changes: true, files: after, file_count: 2, metadata: { name: 'native', version: '1.1.0' } }
 const defaults = { previewServiceImportApi: async (_, request) => { calls.push(['preview', request]); return preview }, importServiceRevisionApi: async (_, request) => { calls.push(['import', request]); return { id: 13, definition: JSON.stringify({ schema_version: 1, files: after }) } }, ...api }
 vm.runInNewContext(code, { module, exports: module.exports, require(name) { if (name === 'vue') return vue; if (name === '@/api/axiops') return defaults; if (name === '@/utils/helm-service-config') return utils.exports; if (name === 'element-plus') return { ElMessage: { success() {} } }; return {} } })
 const props = vue.reactive({ service: { id: 7, code_source_id: 9, repo_url: 'https://git.example/team/native.git', repo_branch: 'main', source_ref: { helm_repository: { repo_path: 'team/native', ref: 'main', chart_path: 'deploy/chart', commit_id: 'old-commit', revision_id: 10 } } }, latestRevision: { id: 12, source_commit: 'manual-chart-edit', definition: JSON.stringify({ schema_version: 1, files: before }) }, canEdit: true, disabled: false })
 let state
 const renderer = vue.createRenderer({ createComment: () => ({}), createText: () => ({}), createElement: () => ({}), insert() {}, remove() {}, setText() {}, setElementText() {}, patchProp() {}, parentNode: () => null, nextSibling: () => null })
 const app = renderer.createApp({ setup() { state = module.exports.default.setup(props, { expose() {}, emit: (...args) => events.push(args) }); return () => null } }); app.mount({})
 return { state, props, calls, events, preview, unmount: () => app.unmount() }
}
test('native Git diff includes additions and deletions; importing preserves environment configuration', async () => {
 const v = mount(); v.state.open(); await flush(); await v.state.check()
 assert.equal(v.state.form.chart_path, 'deploy/chart'); assert.equal(v.state.changes.value, 3)
 await v.state.save(); assert.equal(v.calls.length, 1, 'custom Chart requires acknowledgment')
 v.state.acceptLocal.value = true; await v.state.save()
 assert.equal(v.calls.length, 2); const request = v.calls[1][1]
 assert.equal(request.expected_revision_id, 12); assert.equal(request.expected_commit, 'new-commit'); assert.equal(request.accept_local_changes, true)
 assert.equal(request.paths.length, 0); assert.equal(Object.hasOwn(request, 'environment_id'), false)
 assert.equal(v.events.some(e => e[0] === 'imported'), true); v.unmount()
})
test('changing branch invalidates preview and acknowledgment before import', async () => {
 const v = mount(); v.state.open(); await flush(); await v.state.check(); v.state.acceptLocal.value = true
 v.state.form.ref = 'prod'; await flush(); await v.state.save()
 assert.equal(v.calls.length, 1); assert.equal(v.state.preview.value, undefined); assert.equal(v.state.acceptLocal.value, false); v.unmount()
})
test('concurrent platform revision change blocks an outdated comparison', async () => {
 const v = mount({ previewServiceImportApi: async () => ({ commit_id: 'new', current_revision_id: 99, files: after }) }); v.state.open(); await flush(); await v.state.check()
 assert.equal(v.state.preview.value, undefined); assert.match(v.state.error.value, /已更新/); v.unmount()
})
test('old service response cannot populate a different service', async () => {
 let resolve; const waiting = new Promise(r => { resolve = r })
 const v = mount({ previewServiceImportApi: () => waiting }); v.state.open(); await flush(); const pending = v.state.check()
 v.props.service = { ...v.props.service, id: 8 }; await flush(); resolve(v.preview); await pending
 assert.equal(v.state.preview.value, undefined); assert.equal(v.state.visible.value, false); v.unmount()
})
test('readonly access and rejected import never announce success', async () => {
 const v = mount({ importServiceRevisionApi: async () => { throw Error('revision changed') } }); v.props.canEdit = false; v.state.open(); await v.state.check(); assert.equal(v.calls.length, 0)
 v.props.canEdit = true; v.state.open(); await flush(); await v.state.check(); v.state.acceptLocal.value = true; await v.state.save()
 assert.match(v.state.error.value, /revision changed/); assert.equal(v.events.some(e => e[0] === 'imported'), false); assert.equal(v.state.preview.value, undefined); v.unmount()
})
