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
vm.runInNewContext(transpile(readFileSync(new URL('../src/utils/service-environment-config.ts', import.meta.url), 'utf8')), { module: utils, exports: utils.exports, require: () => yaml })
const file = new URL('../src/views/cicd/project-workspace/components/ServiceConfigurationPanel.vue', import.meta.url)
const { descriptor } = parse(readFileSync(file, 'utf8'))
assert.deepEqual(compileTemplate({ source: descriptor.template.content, id: 'config', filename: file.pathname }).errors, [])
const code = transpile(compileScript(descriptor, { id: 'config' }).content)
const flush = async () => { for (let i=0;i<6;i++) await new Promise(setImmediate) }
const deferred = () => { let resolve; const promise = new Promise(r => { resolve = r }); return { resolve, promise } }
const configuration = (id, port = '8080') => ({ environment_id: id, service_id: 17, bound: true, revision_id: 10, latest_revision_id: 11, format: 'k8s_yaml', replicas: null, image_ref: '', token: `${id}`.repeat(64), definition: '# AxiOps-Variables: [{"key":"port","value":"80"},{"key":"replicas","value":"1"}]\napiVersion: v1\nkind: Service\nmetadata:\n  name: {{.Service}}\nspec:\n  ports:\n  - port: {{.port}}', variables: [{ key:'port', value:port, origin:'override', secret:false }, { key:'replicas', value:'3', origin:'override', secret:false }] })

test('whole YAML, variables and resources use one environment-scoped preview and commit', async () => {
  const writes=[]
  const view=mount({saveServiceEnvironmentConfigApi:async(sid,body,preview)=>{
    writes.push({sid,body:JSON.parse(JSON.stringify(body)),preview})
    return {saved:!preview,validated:true,configs:[{...configuration(body.targets[0].environment_id),definition:body.definition,manifest:'kind: Deployment\nresources: 2Gi',config_digest:'d'.repeat(64)}]}
  }})
  await view.state.switchEnvironment(1)
  view.state.definitionDraft.value += '\n---\nkind: ConfigMap\n'
  view.state.draft.value.port='9090'
  view.state.resourcesEditor.value={dirty:true,getAssets:()=>({resources:{memory_limit:'2Gi'},files:[],secrets:[],routes:[],volumes:[]})}
  await view.state.previewCurrent(false)
  assert.equal(writes.length,2);assert.ok(writes.every(x=>x.preview))
  assert.equal(writes[1].body.definition,view.state.definitionDraft.value)
  assert.equal(writes[1].body.variables.port,'9090');assert.equal(writes[1].body.assets.resources.memory_limit,'2Gi')
  assert.deepEqual(writes[1].body.targets.map(t=>t.environment_id),[1])
  await view.state.commitCurrent()
  assert.equal(writes.length,3);assert.equal(writes[2].preview,undefined)
  assert.equal(view.events.some(e=>e[0]==='published'),false)
  view.unmount()
})

test('production configuration save requires review and does not submit a build',async()=>{
  let saves=0
  const view=mount({saveServiceEnvironmentConfigApi:async(sid,body,preview)=>{if(!preview)saves++;return {saved:!preview,validated:true,configs:[{...configuration(2),manifest:'kind: Service'}]}}})
  await flush()
  await view.state.switchEnvironment(2);await view.state.previewCurrent(false);await view.state.commitCurrent()
  assert.equal(saves,0)
  view.state.currentProductionConfirmed.value=true;await view.state.commitCurrent();assert.equal(saves,1)
  assert.equal(view.events.some(e=>e[0]==='published'),false);view.unmount()
})

test('configuration release uses the displayed deployment snapshot and saved digest without branch',async()=>{
  let submitted
  const conf={...configuration(1),current_deployment:{snapshot_id:27,namespace:'api-test',images:{app:'registry/api@sha256:'+'a'.repeat(64)}}}
  const view=mount({getServiceEnvironmentConfigApi:async()=>conf,getWorkflowApi:async()=>({id:9,target_environment_id:1,nodes:[{type:'build'},{type:'yaml_deploy'}]}),saveServiceEnvironmentConfigApi:async(sid,body,preview)=>({saved:!preview,validated:true,configs:[{...conf,manifest:'kind: Deployment',config_digest:'e'.repeat(64)}]}),runWorkflowApi:async(id,body)=>{submitted=body;return{id:123}}})
  view.props.workflows=[{id:9,target_environment_id:1,delivery_type:'k8s_yaml',status:'active'}]
  await view.state.switchEnvironment(1);await view.state.previewCurrent(true);await view.state.commitCurrent()
  assert.equal(submitted.execution_mode,'config_only');assert.equal(submitted.source_snapshot_id,27);assert.equal(submitted.expected_config_digest,'e'.repeat(64));assert.equal(submitted.branch,undefined);assert.equal(submitted.source_build_id,undefined);assert.equal(submitted.environment_id,1)
  assert.equal(view.events.some(e=>e[0]==='published'&&e[1].runId===123),true);view.unmount()
})
function mount(api = {}, confirm = async () => true) {
  const module = { exports: {} }, calls = [], events = [], warnings = []
  const defaults = {
    getServiceEnvironmentConfigApi: async id => configuration(id),
    saveServiceEnvironmentConfigApi: async (...args) => { calls.push(args); return { saved: !args[2], validated: true, configs: args[1].targets.map(t => ({ ...configuration(t.environment_id), ...(args[1].definition ? { definition: args[1].definition } : {}) })) } },
    ...api
  }
  vm.runInNewContext(code, { module, exports: module.exports, require(name) {
    if (name === 'vue') return vue
    if (name === '@/api/axiops') return defaults
    if (name === '@/utils/service-environment-config') return utils.exports
    if (name === 'element-plus') return { ElMessage: { success() {}, warning: m => warnings.push(m) }, ElMessageBox: { confirm } }
    if (name === '@/composables/useAxiOpsAccess') return { useAxiOpsAccess: () => ({ can: () => true }) }
    return {}
  } })
  const props = vue.reactive({ service: { id:17, project_id:19, current_revision_id:11, code:'api', delivery_type:'k8s_yaml' }, environments: [{ id:1, name:'Test', code:'test', namespace:'api-test' },{ id:2, name:'Prod', code:'prod', namespace:'api-prod', is_production:true }], clusters:[], canEdit:true, canViewBuild:true })
  let state
  const renderer = vue.createRenderer({ createComment: () => ({}), createText: () => ({}), createElement: () => ({}), insert() {}, remove() {}, setText() {}, setElementText() {}, patchProp() {}, parentNode: () => null, nextSibling: () => null })
  const app = renderer.createApp({ setup() { state = module.exports.default.setup(props, { expose() {}, emit: (...args) => events.push(args) }); return () => null } })
  app.mount({})
  return { state, props, calls, events, warnings, unmount: () => app.unmount() }
}

test('invalid companion configuration exits integrated preview without hanging or saving',async()=>{
  const view=mount();await flush()
  view.state.resourcesEditor.value={dirty:true,getAssets:()=>{throw Error('ALB annotation invalid')}}
  await view.state.previewCurrent(false)
  assert.equal(view.state.previewing.value,false);assert.equal(view.state.previewState.value,'error')
  assert.match(view.state.saveError.value,/ALB annotation invalid/);assert.equal(view.calls.length,0)
  view.unmount()
})

test('selected environment opens companion configuration directly without a base selector',async()=>{
  const view=mount();await flush();assert.equal(view.state.environmentId.value,1)
  view.state.openAssets();assert.equal(view.state.assetsVisible.value,true)
  assert.equal(view.state.assetsEnvironmentVisible.value,false)
  assert.equal(view.calls.length,0);view.unmount()
})

test('failed environment loading prevents companion drawer from opening',async()=>{
  const view=mount({getServiceEnvironmentConfigApi:async()=>{throw Error('offline')}})
  await flush();view.state.openAssets();assert.equal(view.state.assetsVisible.value,false)
  assert.equal(view.state.config.value,null);assert.equal(view.state.error.value,'offline');view.unmount()
})

test('blank manual service must save YAML before opening companion configuration',async()=>{
  const view=mount({getServiceEnvironmentConfigApi:async()=>({...configuration(1),bound:false,revision_id:0,definition:'',variables:[]})})
  await flush();view.state.openAssets();assert.equal(view.state.assetsVisible.value,false)
  assert.match(view.warnings.at(-1),/先保存/);view.unmount()
})

test('the visible resource drawer previews the current YAML draft and credential references together', async () => {
  const writes = []
  const view = mount({ saveServiceEnvironmentConfigApi: async (sid, body, preview) => {
    writes.push({ sid, body, preview })
    return { configs: [{ ...configuration(1), manifest: 'kind: Deployment' }] }
  } })
  await view.state.switchEnvironment(1)
  view.state.assetsVisible.value = true
  view.state.sharedEditor.value = { getEnvironmentDraft: () => ({ definition: 'kind: Deployment\n# current unsaved YAML', variables: { port: '9090' } }) }
  view.state.resourcesEditor.value = { dirty: true, getAssets: () => ({ secrets: [{ credential_id: 99 }] }) }
  await view.state.previewAssets()
  assert.equal(writes.length, 2)
  assert.ok(writes.every(write => write.preview === true && write.sid === 17))
  assert.match(writes[1].body.definition, /current unsaved YAML/)
  assert.equal(writes[1].body.variables.port, '9090')
  assert.equal(writes[1].body.assets.secrets[0].credential_id, 99)
  assert.equal(writes[1].body.targets[0].environment_id, 1)
  assert.equal(view.state.assetsVisible.value, false)
  assert.equal(view.state.currentPreviewVisible.value, true)
  view.unmount()
})

test('cancelled environment switching preserves YAML and resource drafts with one confirmation', async () => {
  let accept = false, prompts = 0, discarded = 0
  const view = mount({}, async () => { prompts++; if (!accept) throw Error('cancel') })
  await view.state.switchEnvironment(1)
  view.state.sharedEditor.value = { dirty: true, discardChanges: () => discarded++ }
  view.state.resourcesEditor.value = { dirty: true, discardChanges: () => discarded++ }
  await view.state.switchEnvironment(2)
  assert.equal(view.state.environmentId.value, 1)
  assert.equal(prompts, 1)
  assert.equal(discarded, 0)
  accept = true
  await view.state.switchEnvironment(2)
  assert.equal(view.state.environmentId.value, 2)
  assert.equal(prompts, 2)
  assert.equal(discarded, 2)
  view.unmount()
})

test('saving current environment prevents switching until completion', async () => {
  const view = mount()
  await flush()
  view.state.sharedEditor.value = { saving: true, dirty: false }
  await view.state.switchEnvironment(2)
  assert.equal(view.state.environmentId.value, 1)
  view.state.sharedEditor.value.saving = false
  await view.state.switchEnvironment(2)
  assert.equal(view.state.environmentId.value, 2)
  view.unmount()
})

test('only the selected port is synchronized after server validation; replicas stay unselected', async () => {
  const view = mount(); await view.state.switchEnvironment(1)
  view.state.openSync(); view.state.targetIds.value = [2]; view.state.syncRevision.value = false; view.state.syncKeys.value = ['port']
  await view.state.previewSync()
  assert.equal(view.calls.length, 1); assert.equal(view.calls[0][2], true)
  assert.equal(view.state.plan.value.length, 1)
  await view.state.applySync()
  assert.equal(view.calls.length, 1, 'production save requires the explicit inline acknowledgement')
  view.state.productionConfirmed.value = true
  await view.state.applySync()
  assert.equal(view.calls.length, 2)
  assert.equal(JSON.stringify(view.calls[1][1].variables), '{"port":"8080"}')
  assert.equal(view.calls[1][1].revision_id, 0)
  assert.equal(view.calls[1][1].source.environment_id, 1)
  assert.equal(view.events.some(e => e[0] === 'published'), false) // no build or deploy event
  assert.equal(view.events.some(e => e[0] === 'environment-saved'), true)
  view.unmount()
})
test('source switching protects unsaved edits and loads the selected environment', async () => {
  const view = mount({}, async () => { throw Error('cancel') })
  await view.state.switchEnvironment(1)
  view.state.draft.value.port = '9999'
  await view.state.switchEnvironment(2)
  assert.equal(view.state.environmentId.value, 1)
  assert.equal(view.state.draft.value.port, '9999')
  view.unmount()
})
test('stale environment response cannot replace a newly selected service', async () => {
  const pending = deferred(), view = mount({ getServiceEnvironmentConfigApi: () => pending.promise })
  const request = view.state.switchEnvironment(1); await flush()
  view.props.service = { ...view.props.service, id:18 }; await flush()
  pending.resolve(configuration(1)); await request
  assert.equal(view.state.config.value, null)
  assert.equal(view.state.environmentId.value, 1)
  view.unmount()
})
test('load failure provides no invented values and cannot save', async () => {
  const view = mount({ getServiceEnvironmentConfigApi: async () => { throw Error('offline') } })
  await view.state.switchEnvironment(1); await view.state.saveCurrent()
  assert.equal(view.state.error.value, 'offline')
  assert.equal(view.state.config.value, null)
  assert.equal(view.calls.length, 0)
  view.unmount()
})
test('secret overrides stay masked and cannot be copied over target credentials', async () => {
  const c = configuration(1); c.variables.push({ key:'password', value:'', origin:'override', secret:true })
  const view = mount({ getServiceEnvironmentConfigApi: async id => ({ ...c, environment_id: id }) })
  await view.state.switchEnvironment(1)
  assert.equal(view.state.rows.value.find(row => row.key === 'password').secret, true)
  assert.equal(Object.hasOwn(view.state.draft.value, 'password'), false)
  view.state.openSync(); view.state.targetIds.value = [2]; view.state.syncKeys.value = ['password']
  await view.state.previewSync()
  assert.equal(view.state.plan.value.length, 0); assert.match(view.state.syncError.value, /凭据/)
  view.unmount()
})
test('conflict after preview leaves the dialog open and invalidates the plan', async () => {
  const view = mount({ getServiceEnvironmentConfigApi: async id => ({ ...configuration(id), bound: false }), saveServiceEnvironmentConfigApi: async (_, body, preview) => { if (!preview) throw Error('目标已修改'); return { saved:false, validated:true, configs: [configuration(body.targets[0].environment_id)] } } })
  await flush(); view.state.openSync(); view.state.targetIds.value = [2]; await view.state.previewSync(); view.state.productionConfirmed.value = true; await view.state.applySync()
  assert.equal(view.state.syncVisible.value, true)
  assert.equal(view.state.plan.value.length, 0)
  assert.equal(view.state.syncError.value, '目标已修改')
  view.unmount()
})
test('restore inheritance creates a deletion patch, not a copied default', () => {
  const patch = utils.exports.variablePatch({ port:'9000', replicas:'3' }, { replicas:'3' })
  assert.equal(JSON.stringify(patch), '{"port":null}')
})

test('environment synchronization starts with no targets, variables or template selected', async () => {
  const view = mount(); await view.state.switchEnvironment(1); view.state.openSync()
  assert.equal(view.state.syncRevision.value, false)
  assert.equal(view.state.targetIds.value.length, 0)
  assert.equal(view.state.syncKeys.value.length, 0)
  assert.equal(view.state.syncYaml.value, false)
  assert.equal(view.state.transferEnvironments.value.some(e => e.id === 1), false, 'source is not a target option')
  view.unmount()
})

test('changing targets or fields invalidates preview and production acknowledgement immediately', async () => {
  const view = mount(); await view.state.switchEnvironment(1); view.state.openSync()
  view.state.targetIds.value = [2]; view.state.syncKeys.value = ['port']; await view.state.previewSync()
  view.state.productionConfirmed.value = true
  view.state.syncKeys.value.push('replicas')
  assert.equal(view.state.plan.value.length, 0)
  assert.equal(view.state.productionConfirmed.value, false)
  await view.state.applySync()
  assert.equal(view.calls.length, 1)
  await view.state.previewSync(); view.state.productionConfirmed.value = true
  view.state.targetIds.value = []
  await view.state.applySync()
  assert.equal(view.calls.length, 2)
  view.unmount()
})
test('Helm defaults stay inherited and environment values override scalar paths', () => {
  const values = utils.exports.configValues({ ...configuration(1), format:'helm_chart', definition:JSON.stringify({ files:[{ path:'values.yaml', content:'replicaCount: 1\nimage:\n  repository: team/api\n' }] }), variables:[{ key:'replicaCount', value:'3', origin:'override', secret:false }] })
  assert.equal(values['image.repository'].value, 'team/api')
  assert.equal(values['image.repository'].origin, 'Chart 默认')
  assert.equal(values.replicaCount.value, '3')
})
test('bounded loading preserves ordering and collects failures', async () => {
  let active = 0, peak = 0
  const result = await utils.exports.mapLimited([1,2,3,4,5,6], async item => { active++; peak=Math.max(peak,active); await new Promise(setImmediate); active--; if(item===3) throw Error('failed'); return item }, 2)
  assert.equal(peak, 2); assert.equal(result[2].status, 'rejected'); assert.equal(result[5].value, 6)
})

test('outer join uses base revision and target defaults, never source environment overrides', async () => {
  let joined = false
  const view = mount({ getServiceEnvironmentConfigApi: async id => ({ ...configuration(id), bound: id === 1 || joined }) })
  await view.state.switchEnvironment(1)
  view.state.sharedEditor.value = { revisionId: 10, dirty: false }
  view.state.openJoin()
  assert.equal(view.state.syncMode.value, 'join')
  assert.equal(view.state.targetIds.value.length, 0)
  view.state.targetId.value = 2
  await view.state.previewSync()
  assert.equal(view.calls[0][1].revision_id, 11)
  assert.equal(view.calls[0][1].source, undefined)
  assert.equal(JSON.stringify(view.calls[0][1].variables), '{}')
  assert.equal(view.calls[0][1].definition, undefined)
  assert.equal(view.calls[0][1].assets, undefined)
  await view.state.applySync(); assert.equal(view.calls.length, 1, 'production confirmation required')
  joined = true
  view.state.productionConfirmed.value = true
  await view.state.applySync()
  assert.equal(view.state.environmentId.value, 2, 'opens joined target for review')
  assert.equal(view.state.syncVisible.value, false)
  assert.equal(view.events.some(e => e[0] === 'published'), false)
  view.unmount()
})

test('joining an existing membership cannot upgrade its revision or rewrite overrides', async () => {
  const view = mount(); await flush(); view.state.openJoin(); view.state.targetId.value = 2
  await view.state.previewSync()
  assert.match(view.state.syncError.value, /已加入/)
  assert.equal(view.calls.length, 0)
  view.unmount()
})

test('join permits target-specific required values before membership exists', async () => {
  const view = mount({ getServiceEnvironmentConfigApi: async id => ({ ...configuration(id), bound:false, variables:[], definition:configuration(id).definition.replace('"value":"80"', '"value":""') }) })
  await flush(); view.state.openJoin(); view.state.targetId.value = 2; await view.state.loadTransferTarget()
  assert.equal(view.state.joinRows.value.find(row => row.key === 'port').value, '')
  view.state.joinOverrides.value.port = '9090'
  await view.state.previewSync()
  assert.equal(view.calls[0][1].variables.port, '9090')
  assert.equal(view.calls[0][1].source, undefined)
  view.state.targetId.value = 1
  assert.equal(JSON.stringify(view.state.joinOverrides.value), '{}', 'target edit cannot carry to another environment')
  assert.equal(view.state.transferTarget.value, null)
  view.unmount()
})

test('synchronizing an inherited source value copies it instead of deleting target override', async () => {
  const view = mount({ getServiceEnvironmentConfigApi: async id => ({ ...configuration(id), variables: id === 1 ? [] : configuration(id).variables }) })
  await view.state.switchEnvironment(1); view.state.openSync(); view.state.targetId.value = 2; view.state.syncKeys.value = ['port']
  await view.state.previewSync()
  assert.equal(view.calls[0][1].variables.port, '80')
  assert.equal(Object.hasOwn(view.calls[0][1].variables, 'replicas'), false)
  view.unmount()
})

test('full YAML synchronization is source-guarded, single target, reviewed and one atomic write', async () => {
  const view = mount(); await view.state.switchEnvironment(1); view.state.openSync()
  view.state.targetId.value = 2; view.state.syncYaml.value = true
  view.state.syncDefinition.value = configuration(2).definition + '\n# production-specific edit'
  view.state.syncKeys.value = ['port']
  await view.state.previewSync()
  assert.equal(view.calls.length, 1)
  assert.equal(view.calls[0][1].source.environment_id, 1)
  assert.equal(view.calls[0][1].definition, view.state.syncDefinition.value)
  assert.equal(view.calls[0][1].variables.port, '8080')
  assert.equal(view.calls[0][1].assets, undefined, 'target credentials/files/volumes untouched')
  view.state.productionConfirmed.value = true
  await view.state.applySync(); assert.equal(view.calls.length, 1, 'YAML review required too')
  view.state.yamlConfirmed.value = true
  await view.state.applySync(); assert.equal(view.calls.length, 2)
  assert.equal(view.state.environmentId.value, 1, 'source selection stays after synchronization')
  view.unmount()
})

test('incomplete preview cannot enable saving', async () => {
  const view = mount({ saveServiceEnvironmentConfigApi: async () => ({ saved:false, validated:true, configs:[] }) })
  await view.state.switchEnvironment(1); view.state.openSync(); view.state.targetId.value = 2; view.state.syncKeys.value = ['port']
  await view.state.previewSync()
  assert.equal(view.state.plan.value.length, 0); assert.match(view.state.syncError.value, /不完整/)
  view.unmount()
})

test('late preview cannot approve a different target selection', async () => {
  const pending = deferred()
  const view = mount({ saveServiceEnvironmentConfigApi: async () => pending.promise })
  await view.state.switchEnvironment(1); view.state.openSync(); view.state.targetId.value = 2; view.state.syncKeys.value = ['port']
  const checking = view.state.previewSync(); await flush()
  view.state.targetId.value = undefined
  pending.resolve({ saved:false, validated:true, configs:[configuration(2)] }); await checking
  assert.equal(view.state.plan.value.length, 0)
  assert.equal(view.state.plannedRequest.value, null)
  view.unmount()
})

test('synchronization rejects self targets, multiple targets and unjoined targets', async () => {
  const view = mount({ getServiceEnvironmentConfigApi: async id => ({ ...configuration(id), bound: id === 1 }) })
  await view.state.switchEnvironment(1); view.state.openSync(); view.state.syncKeys.value = ['port']
  for (const targets of [[1], [1,2], [2]]) {
    view.state.targetIds.value = targets; await view.state.previewSync()
    assert.equal(view.state.plan.value.length, 0)
    assert.equal(view.calls.length, 0)
    assert.ok(view.state.syncError.value)
  }
  view.unmount()
})


test('switching with shared edits uses parent confirmation without nonexistent child method', async () => {
  const view = mount(); await flush()
  let discarded = false
  view.state.sharedEditor.value = { dirty: true, discardChanges: () => { discarded = true } }
  await view.state.switchEnvironment(2)
  assert.equal(view.state.environmentId.value, 2)
  assert.equal(discarded, true)
  view.unmount()
})
