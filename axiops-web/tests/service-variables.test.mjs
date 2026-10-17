import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import vm from 'node:vm'
import { parse, compileScript, compileTemplate } from '@vue/compiler-sfc'
import ts from 'typescript'
import * as vue from 'vue'
import * as yaml from 'js-yaml'

const filename = new URL('../src/views/cicd/project-workspace/components/ServiceWorkbenchPanel.vue', import.meta.url)
const { descriptor } = parse(readFileSync(filename, 'utf8'))
assert.deepEqual(compileTemplate({ source: descriptor.template.content, id: 'variables', filename: filename.pathname }).errors, [])
const code = ts.transpileModule(compileScript(descriptor, { id: 'variables' }).content, {
  compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 }
}).outputText
const flush = async () => { for (let i = 0; i < 5; i++) await new Promise(setImmediate) }
const definition = 'apiVersion: v1\nkind: Service\nmetadata:\n  name: {{.Service}}\nspec:\n  ports:\n  - port: {{.port}}\n'

test('probe editor supports none TCP and independent purposes, follows application port and preserves native YAML', async () => {
  const view = mount(); await flush()
  view.state.editorContent.value = 'apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: api\nspec:\n  template:\n    spec:\n      containers:\n      - name: api\n        image: registry/api:v1\n'
  view.state.setVariableValue('port', '28080')
  view.state.setVariableValue('healthPort', '0')
  view.state.setVariableValue('healthCheckType', 'none')
  view.state.setHealthCheckMode('tcp')
  assert.equal(view.state.findVariableValue('probeLivenessEnabled'), 'false')
  view.state.previewMode.value = true
  let container = yaml.load(view.state.editorDisplayContent.value).spec.template.spec.containers[0]
  assert.equal(container.readinessProbe.tcpSocket.port, 28080)
  assert.ok(container.startupProbe); assert.equal(container.livenessProbe, undefined)
  assert.equal(view.state.visibleGroupItems({ key: 'probe', items: [{key:'healthPath'},{key:'probeTimeoutSeconds'}] }).some(v=>v.key==='healthPath'), false)
  view.state.setVariableValue('probeStartupEnabled', 'false')
  container = yaml.load(view.state.editorDisplayContent.value).spec.template.spec.containers[0]
  assert.equal(container.startupProbe, undefined)
  view.state.setHealthCheckMode('none')
  container = yaml.load(view.state.editorDisplayContent.value).spec.template.spec.containers[0]
  assert.equal(container.readinessProbe, undefined)
  view.state.editorContent.value += '        readinessProbe:\n          httpGet: {path: /native, port: 9000}\n'
  assert.equal(view.state.hasNativeProbes.value, true)
  assert.equal(yaml.load(view.state.editorDisplayContent.value).spec.template.spec.containers[0].readinessProbe.httpGet.port, 9000)
  view.unmount()
})

function mount(api = {}) {
  const warnings = [], saved = [], events = [], module = { exports: {} }
  vm.runInNewContext(code, {
    module, exports: module.exports,
    require: (name) => {
      if (name === 'vue') return vue
      if (name === 'js-yaml') return yaml
      if (name === 'element-plus') return { ElMessage: { warning: (m) => warnings.push(m), success() {}, error: (m) => warnings.push(m) } }
      if (name === '@/api/axiops') return {
        listServiceRevisionsApi: async (id) => [{ id, revision_number: 1, definition: id === 1
          ? '# AxiOps-Variables: [{"key":"port","value":"9000"}]\n' + definition : definition }],
        createServiceRevisionApi: async (id, payload) => saved.push({ id, ...payload }),
        ...api
      }
      return {}
    },
    document: { body: { style: { removeProperty() {} } } },
    window: { removeEventListener() {} }
  })
  const props = vue.reactive({ service: { id: 1, code: 'api', image_name: 'api' }, canEdit: true, environments: [], clusters: [], projectName: 'project' })
  let state
  const renderer = vue.createRenderer({
    createComment: () => ({}), createText: () => ({}), createElement: () => ({}),
    insert() {}, remove() {}, setText() {}, setElementText() {}, patchProp() {}, parentNode: () => null, nextSibling: () => null
  })
  const app = renderer.createApp({ setup() {
    state = module.exports.default.setup(props, { expose() {}, emit: (...args) => events.push(args) })
    return () => null
  } })
  app.mount({})
  return { state, props, warnings, saved, events, unmount: () => app.unmount() }
}

test('saving service A leaves service B and the imported template independent', async () => {
  const template = '# AxiOps-Variables: [{"key":"port","value":"9000"}]\n' + definition
  const revisions = new Map([1, 2].map(id => [id, [{ id: id * 10, revision_number: 1, definition: template }]]))
  const writes = []
  const view = mount({
    listServiceRevisionsApi: async id => revisions.get(id),
    createServiceRevisionApi: async (id, payload) => {
      writes.push({ id, ...payload })
      revisions.get(id).unshift({ id: 101, revision_number: 2, ...payload })
    }
  })
  await flush()
  view.state.customVariables.value.find(v => v.key === 'port').value = '9100'
  view.state.editorContent.value += '\n# A only\n'
  await view.state.saveRevision()
  view.props.service = { id: 2, code: 'service-b' }
  await flush()
  assert.equal(view.state.customVariables.value.find(v => v.key === 'port').value, '9000')
  assert.equal(view.state.editorContent.value.includes('# A only'), false)
  assert.equal(revisions.get(2).length, 1)
  assert.equal(revisions.get(2)[0].definition, template)
  view.props.service = { id: 1, code: 'service-a' }
  await flush()
  assert.equal(view.state.customVariables.value.find(v => v.key === 'port').value, '9100')
  assert.equal(view.state.editorContent.value.includes('# A only'), true)
  assert.equal(writes.length, 1)
  assert.equal(writes[0].id, 1)
  assert.equal(writes[0].expected_revision_id, 10)
  assert.deepEqual(Array.from(view.events, event => event[0]), ['saved'])
  view.unmount()
})

test('a late save response cannot reload or emit saved for another service', async () => {
  let resolve
  const pending = new Promise(r => { resolve = r })
  const loads = [], writes = []
  const view = mount({
    listServiceRevisionsApi: async id => { loads.push(id); return [{ id, revision_number: 1, definition }] },
    createServiceRevisionApi: async (id, payload) => { writes.push({ id, ...payload }); await pending }
  })
  await flush()
  view.state.unlockEditor()
  view.state.editorContent.value += '\n# A only\n'
  const saving = view.state.saveRevision()
  assert.equal(view.state.saving.value, true)
  assert.equal(view.state.editorReadonly.value, true)
  view.props.service = { id: 2, code: 'service-b' }
  await flush()
  const current = view.state.editorContent.value
  resolve()
  await saving
  assert.deepEqual(loads, [1, 2])
  assert.equal(writes[0].id, 1)
  assert.equal(view.state.editorContent.value, current)
  assert.equal(view.events.length, 0)
  assert.equal(view.state.saving.value, false)
  view.unmount()
})

test('loading an old revision never invents a port or retains another service value', async () => {
  const view = mount()
  await flush()
  assert.equal(view.state.customVariables.value.find((v) => v.key === 'port').value, '9000')
  view.props.service = { id: 17, code: 'risk-server', image_name: 'risk_server' }
  await flush()
  assert.equal(view.state.customVariables.value.find((v) => v.key === 'port').value, '')
  assert.deepEqual(Array.from(view.state.unsetVariables.value), ['port'])
  assert.equal(view.state.customVariables.value.some((v) => v.key === 'Service'), false)
  assert.equal(view.state.validateVariables(), false)
  assert.match(view.warnings.at(-1), /缺少变量值/)
  // Environment-provided values may remain unset in the portable template.
  await view.state.saveRevision()
  assert.equal(view.saved.length, 1)
  assert.match(view.saved[0].definition, /\{\{\.port\}\}/)
  view.unmount()
})

test('saving explicit defaults preserves the placeholders and metadata', async () => {
  const view = mount()
  await flush()
  view.state.customVariables.value.find((v) => v.key === 'port').value = '8001'
  await view.state.saveRevision()
  assert.equal(view.saved.length, 1)
  assert.match(view.saved[0].definition, /"key":"port","value":"8001"/)
  assert.match(view.saved[0].definition, /port: \{\{\.port\}\}/)
  view.unmount()
})

test('manual services never invent defaults and retain explicitly saved metadata', async () => {
  const view = mount(); await flush()
  view.props.service = { id: 17, code: 'api', image_name: 'api', source_type: 'manual' }
  await flush()
  assert.equal(view.state.customVariables.value.find(v => v.key === 'port').value, '')
  view.props.service = { id: 1, code: 'api', image_name: 'api', source_type: 'manual' }
  await flush()
  assert.equal(view.state.customVariables.value.find(v => v.key === 'port').value, '9000')
  view.unmount()
})

test('manual native YAML stays blank initially and system placeholders need no template parameters', async () => {
  const view = mount({ listServiceRevisionsApi: async () => [] })
  view.props.service = { id: 18, code: 'api', source_type: 'manual', delivery_type: 'k8s_yaml' }
  view.props.environmentId = 1
  view.props.environmentConfig = { environment_id: 1, service_id: 18, bound: false, format: 'k8s_yaml', definition: '', variables: [] }
  await flush()
  assert.equal(view.state.editorContent.value, '')
  assert.equal(view.state.customVariables.value.length, 0)
  const body = 'apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: {{.Service}}\nspec:\n  template:\n    spec:\n      containers:\n      - name: api\n        image: {{.Image}}\n'
  view.state.editorContent.value = body
  await flush()
  assert.equal(view.state.usesTemplateParameters.value, false)
  assert.equal(view.state.customVariables.value.length, 0)
  assert.equal(view.state.validateVariables(), true)
  assert.equal(view.state.editorContent.value, body)
  view.state.editorContent.value = body.replace('{{.Service}}', 'api').replace('{{.Image}}', 'registry/api:v1')
  await flush()
  assert.equal(view.state.customVariables.value.length, 0)
  assert.equal(view.state.usesTemplateParameters.value, false)
  view.unmount()
})

test('discarding shared changes restores saved defaults and keeps save/join actions available', async () => {
  const view = mount(); await flush()
  view.state.unlockEditor()
  view.state.customVariables.value.find(v => v.key === 'port').value = '9999'
  assert.equal(view.state.dirty.value, true)
  view.state.toggleEditing()
  assert.equal(view.state.customVariables.value.find(v => v.key === 'port').value, '9000')
  assert.equal(view.state.dirty.value, false)
  assert.equal(view.state.editing.value, true)
  assert.equal(view.saved.length, 0)
  view.unmount()
})


test('new YAML file preserves existing template and saves all files in one revision', async () => {
  const view = mount(); await flush()
  view.state.unlockEditor()
  const before = view.state.editorContent.value
  view.state.openNewFile()
  view.state.newFileName.value = 'configmap.yaml'
  view.state.newFileContent.value = 'apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: app-config\ndata:\n  app.yaml: |\n    port: 8080'
  view.state.createManualFile()
  assert.equal(view.state.activeFileName.value, 'configmap.yaml')
  assert.match(view.state.editorContent.value, /kind: Service/)
  assert.match(view.state.editorContent.value, /kind: ConfigMap/)
  assert.equal(view.state.dirty.value, true)
  assert.equal(view.saved.length, 0, 'adding a file must not save or deploy')
  await view.state.saveRevision()
  assert.equal(view.saved.length, 1)
  assert.match(view.saved[0].definition, /# AxiOps-File: configmap.yaml/)
  assert.ok(view.saved[0].definition.includes(before.trim()))
  view.unmount()
})

test('duplicate or path-like YAML filename cannot overwrite existing files', async () => {
  const view = mount(); await flush()
  view.state.unlockEditor()
  const before = view.state.editorContent.value
  view.state.newFileContent.value = 'apiVersion: v1'
  for (const name of ['deployment.yaml', '../config.yaml', 'file.txt', '__proto__']) {
    view.state.newFileName.value = name
    view.state.createManualFile()
    assert.equal(view.state.editorContent.value, before)
  }
  assert.equal(view.saved.length, 0)
  view.unmount()
})

test('plain Secret YAML cannot enter ordinary template revision from editor', async () => {
  const view = mount(); await flush()
  view.state.unlockEditor()
  view.state.newFileName.value = 'secret.yaml'
  view.state.newFileContent.value = 'apiVersion: v1\nkind: Secret\nmetadata:\n  name: db\nstringData:\n  password: test-only'
  view.state.createManualFile()
  await view.state.saveRevision()
  assert.equal(view.saved.length, 0)
  assert.ok(view.warnings.some(message => message.includes('Secret')))
  view.unmount()
})

test('saved and read-only service definitions can be validated without saving or unlocking', async () => {
  const view = mount(); await flush()
  assert.equal(view.state.editorReadonly.value, true)
  const before = view.state.editorContent.value
  assert.equal(view.state.validateVariables(), true)
  view.props.canEdit = false
  assert.equal(view.state.validateVariables(), true)
  assert.equal(view.state.editorContent.value, before)
  assert.equal(view.saved.length, 0)
  assert.equal(view.state.editing.value, false)
  view.unmount()
})

test('structural checks allow environment-supplied defaults but never save malformed documents', async () => {
  const view = mount(); await flush()
  view.state.customVariables.value.find(v => v.key === 'port').value = ''
  assert.equal(view.state.validateVariables(false, true), true)
  assert.match(view.warnings.at(-1), /待目标环境提供变量.*port/)
  assert.equal(view.state.validateVariables(), false)
  const original = view.state.editorContent.value
  for (const broken of [original + '\n---\ninvalid scalar', original + '\ninvalid: [', original + '\napiVersion: v1']) {
    view.state.editorContent.value = broken
    await view.state.saveRevision()
  }
  assert.equal(view.saved.length, 0)
  assert.equal(view.state.validateVariables(false, true), false)
  view.unmount()
})
