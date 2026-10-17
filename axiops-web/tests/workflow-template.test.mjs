import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import vm from 'node:vm'
import ts from 'typescript'
import * as vue from 'vue'
import * as yaml from 'js-yaml'
import { parse, compileScript, compileTemplate } from '@vue/compiler-sfc'

function load(path, imports = {}) {
  const module = { exports: {} }
  const source = readFileSync(new URL('../src/' + path, import.meta.url), 'utf8')
  const code = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 } }).outputText
  vm.runInNewContext(code, { module, exports: module.exports, require: name => imports[name] || {} })
  return module.exports
}
const workflowUtils = load('utils/workflow-template.ts')
const envUtils = load('utils/service-environment-config.ts', { 'js-yaml': yaml })
const comparison = load('utils/environment-comparison.ts', { './service-environment-config': envUtils }).environmentComparison
const service = { id: 17, code: 'api', current_revision_id: 11, repo_branch: 'test' }
const template = (deploy = true) => ({ id: 1, name: 'Template', kind: 'workflow', revision: 1, files: [{ path: 'workflow.json', content: JSON.stringify({ delivery_type: 'k8s_yaml', nodes: [{ key: 'build', name: 'Build', type: 'build', config: { service_id: 0 } }, ...(deploy ? [{ key: 'deploy', name: 'Deploy', type: 'yaml_deploy', config: { service_id: 0, service_revision_id: 0 }, depends_on: ['build'] }] : [])] }) }] })
test('workflow templates produce independent service-bound definitions and reject incompatible deployment', () => {
  const original = template(), before = JSON.stringify(original)
  const def = workflowUtils.instantiateWorkflowTemplate(original, 'k8s_yaml', service)
  assert.equal(def.nodes[0].config.service_id, 17)
  assert.equal(def.nodes[0].config.branch, 'test')
  assert.equal(def.nodes[1].config.service_revision_id, 11)
  assert.equal(JSON.stringify(original), before)
  assert.throws(() => workflowUtils.instantiateWorkflowTemplate(original, 'helm', service), /不匹配/)
  assert.equal(workflowUtils.instantiateWorkflowTemplate(template(false), 'host', service).delivery_type, 'host')
})
test('environment comparison masks sensitive values and displays independent references and domains', () => {
  const base = { format: 'k8s_yaml', definition: '', revision_id: 11, variables: [{ key: 'PASSWORD', value: 'must-not-appear', secret: true, origin: 'override' }], assets: { secrets: [{ name: 'db', credential_id: 4, credential_key: 'password', env_name: 'DB_PASSWORD' }], routes: [{ name: 'web', host: 'qa.example', path: '/', service: 'api', port: 8080 }] } }
  const target = structuredClone(base); target.assets.routes[0].host = 'prod.example'; target.assets.secrets[0].credential_id = 5
  const rows = comparison(base, target)
  assert.ok(!JSON.stringify(rows).includes('must-not-appear'))
  assert.ok(rows.find(row => row.key === '域名 / web').changed)
  assert.ok(rows.find(row => row.key === '凭据 / db').changed)
})

const flush = async () => { for (let i=0;i<6;i++) await new Promise(setImmediate) }
test('workflow creation requires an explicit deployment environment, but build-only stays unbound', async () => {
  const file = new URL('../src/views/cicd/project-workspace/components/WorkflowCreationDialog.vue', import.meta.url)
  const { descriptor } = parse(readFileSync(file, 'utf8'))
  assert.deepEqual(compileTemplate({ source: descriptor.template.content, id: 'create', filename: file.pathname }).errors, [])
  const code = ts.transpileModule(compileScript(descriptor, { id: 'create' }).content, { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 } }).outputText
  const module = { exports: {} }, calls = [], events = []
  const build = { ...template(false), id: 2 }
  vm.runInNewContext(code, { module, exports: module.exports, require(name) {
    if (name === 'vue') return vue
    if (name === '@/utils/workflow-template') return workflowUtils
    if (name === 'element-plus') return { ElMessage: { warning() {}, success() {} } }
    if (name === '@/api/axiops') return { listDeliveryTemplatesApi: async () => [template(), build], previewDeliveryTemplateApi: async () => ({}), createWorkflowApi: async (id, body) => { calls.push(body); return { id: 10 } } }
    return {}
  } })
  const props = vue.reactive({ visible: true, project: { id: 19, delivery_type: 'k8s_yaml' }, services: [service], environments: [{ id: 7, name: 'QA', status: 'active', binding_status: 'validated' }] })
  const scope = vue.effectScope(), state = scope.run(() => module.exports.default.setup(props, { expose() {}, emit: (...e) => events.push(e) }))
  await flush(); state.templateId.value = 1; state.name.value = 'API deploy'
  await state.create(); assert.equal(calls.length, 0)
  state.environmentId.value = 7; await state.create()
  assert.equal(calls[0].target_environment_id, 7)
  state.templateId.value = 2; await state.create()
  assert.equal(calls[1].target_environment_id, 0, 'stale selected environment must not leak into build-only workflow')
  assert.equal(calls[1].definition.nodes.length, 1)
  assert.ok(events.some(e => e[0] === 'created'))
  scope.stop()
})

test('notification templates need no service and multi-service bindings remain independent', () => {
 const notice={files:[{path:'workflow.json',content:JSON.stringify({delivery_type:'k8s_yaml',nodes:[{key:'notice',name:'Notice',type:'notify',config:{message:'ok'}}]})}]}
 assert.equal(workflowUtils.instantiateWorkflowTemplate(notice,'helm').nodes[0].config.message,'ok')
 const multi=template(); const second={...service,id:18,current_revision_id:21,code:'other'}
 const def=workflowUtils.instantiateWorkflowTemplate(multi,'k8s_yaml',undefined,{build:service,deploy:second})
 assert.equal(def.nodes[0].config.service_id,17)
 assert.equal(def.nodes[1].config.service_id,18)
 assert.equal(def.nodes[1].config.service_revision_id,21)
 assert.throws(()=>workflowUtils.instantiateWorkflowTemplate(multi,'k8s_yaml'),/请选择服务/)
})
