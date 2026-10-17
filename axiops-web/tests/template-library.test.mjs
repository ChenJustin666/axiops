import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import vm from 'node:vm'
import { parse, compileScript, compileTemplate } from '@vue/compiler-sfc'
import ts from 'typescript'
import * as vue from 'vue'
import * as yaml from 'js-yaml'
import * as yamlTemplate from 'yaml'

const flush = async () => { for (let i = 0; i < 5; i++) await new Promise(setImmediate) }
const deferred = () => { let resolve; const promise = new Promise(r => { resolve = r }); return { promise, resolve } }
const template = (id = 1, kind = 'k8s_yaml') => ({ id, kind, name: 'service', revision: 1, description: '', builtin_key: 'base', files: [{ path: 'deployment.yaml', content: 'image: {{.Image}}' }], variables: [{ key: 'port', value: '8080', description: '' }] })
const namingModule = { exports: {} }
vm.runInNewContext(ts.transpileModule(readFileSync(new URL('../src/utils/service-name.ts', import.meta.url), 'utf8'), {
  compilerOptions: { module: ts.ModuleKind.CommonJS }
}).outputText, { module: namingModule, exports: namingModule.exports })

const configModule = { exports: {} }
vm.runInNewContext(ts.transpileModule(readFileSync(new URL('../src/utils/service-environment-config.ts', import.meta.url),'utf8'),{compilerOptions:{module:ts.ModuleKind.CommonJS}}).outputText, {module:configModule,exports:configModule.exports,require:()=>yaml})
const assets = () => ({command:[],args:[],workload_kind:'',workload_name:'',container:'',files:[],secrets:[],routes:[],volumes:[]})

test('enabling health checks from an older template offers independent checks with conservative defaults', async () => {
  const view = mount('views/cicd/project-workspace/components/TemplateConfigurationForm.vue', { definition: '', parameters: [{key:'healthCheckType',value:'none'}], values: {healthCheckType:'none'}, serviceCode:'api' }, {})
  await flush()
  view.state.setParameter('healthCheckType', 'tcp')
  const next = view.events.find(e=>e[0]==='update:values')[1]
  assert.equal(next.probeReadinessEnabled, 'true');assert.equal(next.probeStartupEnabled, 'true');assert.equal(next.probeLivenessEnabled, 'false')
  assert.equal(view.state.parameterSections.value.find(s=>s.key==='health').parameters.filter(p=>/Enabled$/.test(p.key)).length, 3)
  view.unmount()
})
const templateConfigModule = {exports:{}}
vm.runInNewContext(ts.transpileModule(readFileSync(new URL('../src/utils/template-configuration.ts',import.meta.url),'utf8'),{compilerOptions:{module:ts.ModuleKind.CommonJS}}).outputText,{
  TextEncoder,module:templateConfigModule,exports:templateConfigModule.exports,require:name=>name==='yaml'?yamlTemplate:{emptyServiceAssets:assets}
})
function mount(path, input, api, confirm = async () => true) {
  const filename = new URL('../src/' + path, import.meta.url).pathname
  const { descriptor } = parse(readFileSync(new URL('../src/' + path, import.meta.url), 'utf8'))
  assert.deepEqual(compileTemplate({ source: descriptor.template.content, id: 'templates', filename }).errors, [])
  const code = ts.transpileModule(compileScript(descriptor, { id: 'templates' }).content, {
    compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 }
  }).outputText
  const module = { exports: {} }, events = [], errors = []
  vm.runInNewContext(code, { module, exports: module.exports, require(name) {
    if (name === 'vue') return vue
    if (name === '@/api/axiops') return api
    if (name === '@/utils/service-name') return namingModule.exports
    if (name === '@/utils/service-environment-config') return configModule.exports
    if (name === '@/utils/template-configuration') return templateConfigModule.exports
    if (name === '@/api/axiops/service-assets') return {emptyServiceAssets:assets}
    if (name === 'element-plus') return { ElMessageBox: { confirm }, ElMessage: { success() {}, warning() {}, error: e => errors.push(e) } }
    if (name === 'vue-router') return { useRouter: () => ({ replace: async () => {} }), useRoute: () => ({ params: {projectId:'1'}, query: {} }), onBeforeRouteLeave() {} }
    if (name === '@/composables/useAxiOpsAccess') return { useAxiOpsAccess: () => ({ can: () => true, load: async () => {} }) }
    return {}
  } })
  const props = vue.reactive(input)
  let state, exposed
  const renderer = vue.createRenderer({ createComment: () => ({}), createText: () => ({}), createElement: () => ({}), insert() {}, remove() {}, setText() {}, setElementText() {}, patchProp() {}, parentNode: () => null, nextSibling: () => null })
  const app = renderer.createApp({ setup() {
    state = module.exports.default.setup(props, { expose(value) { exposed=value }, emit: (...e) => events.push(e) })
    return () => null
  } })
  app.mount({})
  return { state, props, events, errors, get exposed(){return exposed}, unmount: () => app.unmount() }
}

test('picker ignores stale categories and emits an independent copy only on initial load or selection', async () => {
  const first = deferred(), second = deferred(), original = template(2, 'helm')
  let calls = 0
  const view = mount('components/delivery-template-select/index.vue', { kind: 'k8s_yaml', autoSelect: true }, {
    listDeliveryTemplatesApi: () => ++calls === 1 ? first.promise : second.promise
  })
  view.props.kind = 'helm'
  await flush()
  second.resolve([original]); await flush()
  first.resolve([template()]); await flush()
  assert.equal(view.state.templates.value[0].id, 2)
  assert.equal(view.events.length, 1)
  view.events[0][1].files[0].content = 'local edit'
  assert.equal(original.files[0].content, 'image: {{.Image}}')
  await view.state.load()
  assert.equal(view.events.length, 1, 'refresh must not overwrite service edits')
  view.unmount()
})

test('library preserves dirty drafts when discard is canceled and on revision conflict', async () => {
  const original = template()
  const view = mount('views/cicd/template-library/index.vue', {}, {
    listDeliveryTemplatesApi: async () => [original],
    saveDeliveryTemplateApi: async () => { throw new Error('revision conflict') }
  }, async () => { throw new Error('cancel') })
  await flush()
  view.state.draft.value.files[0].content = 'unsaved'
  await view.state.changeKind('helm')
  assert.equal(view.state.kind.value, 'k8s_yaml')
  await view.state.save()
  assert.equal(view.state.draft.value.files[0].content, 'unsaved')
  assert.equal(view.state.dirty.value, true)
  assert.equal(original.files[0].content, 'image: {{.Image}}')
  view.unmount()
})

test('canceling template application restores the previously applied selection', async () => {
  const view = mount('components/delivery-template-select/index.vue', {
    kind: 'k8s_yaml', autoSelect: true, apply: async item => item.id === 1
  }, { listDeliveryTemplatesApi: async () => [template(1), template(2)] })
  await flush()
  view.state.selected.value = 2
  await view.state.choose(2)
  assert.equal(view.state.selected.value, 1)
  assert.equal(view.events.length, 1)
  view.unmount()
})

test('service editor ignores an old service response and blocks save while loading', async () => {
  const first = deferred(), second = deferred()
  const view = mount('views/cicd/project-workspace/components/ServiceCreationWizard.vue', {
    visible: false, projectId: 1, deliveryType: 'k8s_yaml', service: { id: 1 }, codeSources: [], environments: [], canCreateWorkflow: false, canUpdateBuild: false
  }, { listServiceRevisionsApi: id => id === 1 ? first.promise : second.promise })
  view.props.visible = true; await flush()
  assert.equal(await view.state.validateCurrentStep(), false)
  view.props.service = { id: 2 }; await flush()
  second.resolve([{ revision_number: 1, definition: '# AxiOps-Variables: [{"key":"port","value":"9000"}]\nimage: second' }]); await flush()
  first.resolve([{ revision_number: 1, definition: 'image: first' }]); await flush()
  assert.equal(view.state.fileContents['deployment.yaml'], 'image: second')
  assert.equal(view.state.serviceVariables.port, '9000')
  view.unmount()
})

test('service validates the selected template before creating any records', async () => {
  let created = 0
  const view = mount('views/cicd/project-workspace/components/ServiceCreationWizard.vue', {
    visible: true, projectId: 1, deliveryType: 'k8s_yaml', service: null, codeSources: [], environments: [], canCreateWorkflow: false, canUpdateBuild: false
  }, { previewDeliveryTemplateApi: async () => { throw new Error('invalid template') }, createServiceApi: async () => { created++; return { id: 1 } } })
  view.state.reset()
  view.state.form.name = 'service'
  view.state.form.code = 'service'
  view.state.formRef.value = { validate: async () => true }
  await view.state.useLibraryTemplate(template())
  await view.state.submit()
  assert.equal(created, 0)
  assert.equal(view.errors.at(-1), 'invalid template')
  view.unmount()
})

const helmCreationProps = (extra = {}) => ({
  visible: true, projectId: 1, deliveryType: 'helm', service: null,
  codeSources: [], environments: [], canCreateWorkflow: false, canUpdateBuild: false,
  createWorkflow: false, ...extra
})
const creationPath = 'views/cicd/project-workspace/components/ServiceCreationWizard.vue'

test('new Helm services expose only template and Git, normalize stale manual entry and have no generated Chart', async () => {
  let writes = 0
  const view = mount(creationPath, helmCreationProps({initialSource:'manual'}), {
    createServiceApi: async () => { writes++; return {id:1} }
  })
  view.state.reset()
  assert.deepEqual(Array.from(view.state.modeOptions.value, o => o.value), ['template','git'])
  assert.equal(view.state.form.source_type, 'template')
  assert.equal(view.state.serviceFiles.value.length, 0)
  view.state.formRef.value = {validate:async()=>true}
  await view.state.submit()
  assert.equal(writes, 0, 'cannot save a generated or unselected Chart')
  view.state.openSourceDialog('manual')
  assert.equal(view.state.form.source_type, 'template')
  view.state.createManualServiceDirectory()
  assert.equal(view.state.form.source_type, 'template')
  assert.equal(await view.state.useLibraryTemplate(template(4,'k8s_yaml')), false)
  view.unmount()
})

test('Helm template creation keeps exact source reference and customized copies remain independent', async () => {
  for (const customized of [false,true]) {
    const writes=[]
    const view=mount(creationPath,helmCreationProps({initialSource:'template'}),{
      previewDeliveryTemplateApi:async()=>({definition:'validated chart'}),
      createServiceApi:async(_id,payload)=>{writes.push(payload);return {id:7}},
      createServiceRevisionApi:async(_id,payload)=>{writes.push(payload);return {id:8}}
    })
    view.state.reset()
    view.state.form.name='gateway';view.state.form.code='gateway'
    view.state.formRef.value={validate:async()=>true}
    await view.state.useLibraryTemplate(template(4,'helm'))
    if(customized)view.state.fileContents['deployment.yaml']='changed chart'
    await view.state.submit()
    assert.equal(writes[0].source_type,'template')
    assert.equal(writes[1].template?.id,customized?undefined:4)
    assert.equal(writes[1].definition,'validated chart')
    assert.equal(view.errors.length,0)
    view.unmount()
  }
})

test('Helm repository import keeps repository and Chart paths separate and fixes the imported commit', async () => {
  const requests=[]
  const view=mount(creationPath,helmCreationProps({initialSource:'git'}),{
    createServiceApi:async()=>({id:7}),
    previewServiceImportApi:async(service,request)=>{requests.push({service,request});return {commit_id:'fixed-commit'}},
    importServiceRevisionApi:async(service,request)=>{requests.push({service,request});return {id:8}}
  })
  view.state.reset()
  Object.assign(view.state.form,{name:'gateway',code:'gateway',code_source_id:1,repo_url:'https://git.example/team/repo',repo_path:'team/repo',chart_path:'deploy/chart',paths:'deploy/chart/Chart.yaml\ndeploy/chart/values.yaml'})
  view.state.formRef.value={validate:async()=>true}
  await view.state.submit()
  assert.equal(requests.length,2)
  assert.equal(requests[0].service.delivery_type,'helm','API must select helm-import, not yaml-import')
  assert.equal(requests[0].request.repo_path,'team/repo')
  assert.equal(requests[0].request.chart_path,'deploy/chart')
  assert.equal(requests[1].request.expected_commit,'fixed-commit')
  assert.equal(view.errors.length,0)
  view.unmount()
})

test('YAML manual creation and existing custom Helm editing remain available', async () => {
  const yamlView=mount(creationPath,helmCreationProps({deliveryType:'k8s_yaml'}),{})
  yamlView.state.reset()
  assert.equal(yamlView.state.modeOptions.value.some(o=>o.value==='manual'),true)
  assert.equal(yamlView.state.form.source_type,'manual')
  yamlView.unmount()
  const view=mount(creationPath,helmCreationProps({service:{id:7,source_type:'manual'}}),{
    listServiceRevisionsApi:async()=>[{revision_number:1,definition:JSON.stringify({files:[{path:'Chart.yaml',content:'name: custom'}]})}]
  })
  view.state.reset()
  await view.state.loadExistingDefinition()
  await flush()
  assert.equal(view.state.form.source_type,'manual')
  assert.equal(view.state.fileContents['Chart.yaml'],'name: custom')
  view.state.formRef.value={validate:async()=>true}
  assert.equal(await view.state.validateCurrentStep(),true)
  view.unmount()
})

test('initialization redirects stale Helm manual entry to template and blocks directory creation', async () => {
  let writes=0
  const view=mount('views/cicd/project-wizard/index.vue',{}, {
    getProjectApi:async()=>({id:1,initialization_status:'completed'}),
    createServiceApi:async()=>{writes++;return {id:1}}
  })
  await flush()
  view.state.project.value={id:1,delivery_type:'helm'}
  view.state.beginManualDirectory()
  assert.equal(view.state.wizardSource.value,'template')
  assert.equal(view.state.manualDirectoryOpen.value,false)
  await view.state.createManualDirectory()
  assert.equal(writes,0)
  view.unmount()
})

test('service template menu browses only the project delivery type and discards stale results', async () => {
  const first = deferred(), second = deferred(), calls = []
  const view = mount('views/cicd/project-workspace/components/ServiceTemplatePicker.vue', {
    visible: true, projectId: 19, kind: 'k8s_yaml', existingCodes: []
  }, { listDeliveryTemplatesApi: kind => { calls.push(kind); return kind === 'helm' ? second.promise : first.promise } })
  view.props.kind = 'helm'; await flush()
  second.resolve([template(2, 'helm'), template(3, 'dockerfile')]); await flush()
  first.resolve([template(1)]); await flush()
  assert.deepEqual(calls, ['k8s_yaml', 'helm'])
  assert.equal(view.state.templates.value.length, 1)
  assert.equal(view.state.templates.value[0].kind, 'helm')
  assert.equal(view.state.selected.value, null, 'do not silently apply a template')
  view.unmount()
})


const pickerPath = 'views/cicd/project-workspace/components/ServiceTemplatePicker.vue'
const envs = [{id:1,name:'Test',namespace:'api-test'},{id:2,name:'Prod',namespace:'api-prod',is_production:true}]
const service = {id:17,name:'gateway',code:'gateway',project_id:19,delivery_type:'k8s_yaml',status:'active',current_revision_id:23}
const config = (eid=1) => ({environment_id:eid,service_id:17,bound:true,revision_id:23,token:String(eid).repeat(64),definition:'saved YAML',variables:[],assets:assets()})
const pickerProps = (extra={}) => ({visible:true,projectId:19,kind:'k8s_yaml',existingCodes:[],environments:envs,...extra})
const pickerAPI = (extra={}) => ({listDeliveryTemplatesApi:async()=>[template(7)],previewDeliveryTemplateApi:async()=>({definition:'saved snapshot'}),getServiceEnvironmentConfigApi:async eid=>config(eid),...extra})
const select = async view => {await flush();view.state.selected.value=view.state.templates.value[0];view.state.name.value= view.state.target.value?.name || 'risk_server';await flush()}

test('template creation validates and commits atomically only to the selected environment',async()=>{
  const writes=[]
  const view=mount(pickerPath,pickerProps(),pickerAPI({
    createConfiguredServiceApi:async(eid,payload,preview)=>{writes.push({eid,payload,preview});return {service_id:81,saved:!preview,validated:true,configs:[{...config(eid),service_id:81,manifest:'rendered'}]}},
    createServiceApi:async()=>{throw Error('legacy directory creation prohibited')},createServiceRevisionApi:async()=>{throw Error('shared revision edit prohibited')}
  }))
  await select(view);await view.state.create()
  assert.deepEqual(writes.map(w=>[w.eid,!!w.preview]),[[1,true],[1,false]])
  assert.equal(writes[1].payload.definition,'saved snapshot')
  assert.equal(writes[1].payload.variables.port,'8080')
  assert.equal(view.events.some(([event,id,,env])=>event==='saved'&&id===81&&env===1),true)
  view.unmount()
})

test('invalid template and duplicate service cannot write or close the picker',async()=>{
  let writes=0
  const view=mount(pickerPath,pickerProps({existingCodes:['risk-server']}),pickerAPI({createConfiguredServiceApi:async()=>writes++}))
  await select(view);await view.state.create();assert.match(view.state.error.value,/已存在/)
  view.state.name.value='other';view.state.preparedDefinition.value='';await view.state.create()
  assert.equal(writes,0);assert.equal(view.events.length,0)
  view.unmount()
  const bad=mount(pickerPath,pickerProps(),pickerAPI({previewDeliveryTemplateApi:async()=>{throw Error('invalid snapshot')},createConfiguredServiceApi:async()=>writes++}))
  await select(bad);assert.equal(bad.state.error.value,'invalid snapshot');await bad.state.create();assert.equal(writes,0);bad.unmount()
})

test('existing service template replacement saves target config with token and never rewrites shared revisions',async()=>{
  const writes=[]
  const view=mount(pickerPath,pickerProps({service}),pickerAPI({
    createServiceRevisionApi:async()=>{throw Error('must not rewrite shared template')},
    saveServiceEnvironmentConfigApi:async(id,payload,preview)=>{writes.push({id,payload,preview});return {saved:!preview,validated:true,configs:[{...config(),manifest:'rendered'}]}}
  }))
  await select(view);await view.state.create()
  assert.equal(writes.length,2)
  assert.equal(writes[1].id,17);assert.equal(writes[1].payload.revision_id,0)
  assert.equal(writes[1].payload.targets[0].token,'1'.repeat(64))
  assert.deepEqual(Array.from(writes[1].payload.targets,t=>t.environment_id),[1])
  assert.equal(view.events.some(([event,id,replaced])=>event==='saved'&&id===17&&replaced),true)
  view.unmount()
})

test('preview failure and canceled confirmation keep editable configuration and do not commit',async()=>{
  for(const scenario of ['failed','cancel']) {
    let commits=0
    const view=mount(pickerPath,pickerProps(),pickerAPI({createConfiguredServiceApi:async(eid,payload,preview)=>{
      if(!preview)commits++
      if(scenario==='failed')throw Error('target invalid')
      return{service_id:81,validated:true,configs:[{manifest:'rendered'}]}
    }}),async()=>{throw 'cancel'})
    await select(view);await view.state.create()
    assert.equal(commits,0);assert.equal(view.events.length,0)
    assert.equal(view.state.preparedDefinition.value,'saved snapshot')
    assert.equal(view.state.error.value,scenario==='failed'?'target invalid':'')
    view.unmount()
  }
})

test('referencing an environment keeps handwritten resources and guards source and production target',async()=>{
  const writes=[]
  const source={...config(1),definition:'# AxiOps-Variables: [{"key":"port","value":"80"}]\nkind: Deployment\n---\nkind: PodDisruptionBudget',variables:[{key:'port',value:'9090',secret:false}],assets:{...assets(),secrets:[{name:'test-secret',credential_id:42}]}}
  const target={...config(2),assets:{...assets(),secrets:[{name:'prod-secret',credential_id:71}]}}
  const view=mount(pickerPath,pickerProps({service,preferredEnvironmentId:2}),pickerAPI({
    getServiceEnvironmentConfigApi:async eid=>eid===1?source:target,
    saveServiceEnvironmentConfigApi:async(id,payload,preview)=>{writes.push({id,payload,preview});return{validated:true,configs:[{...target,manifest:'rendered'}]}}
  }))
  await flush();view.state.referenceId.value=1;await flush()
  assert.match(view.state.preparedDefinition.value,/PodDisruptionBudget/)
  assert.equal(view.state.parameterValues.value.port,'9090')
  await view.state.create();assert.equal(writes.length,0)
  view.state.targetReviewed.value=true;await view.state.create()
  assert.equal(writes.length,2);assert.equal(writes[1].payload.source.token,source.token)
  assert.equal(writes[1].payload.assets,undefined,'target assets retained and source secret never copied')
  assert.equal(writes[1].payload.targets[0].environment_id,2)
  view.unmount()
})

test('closing or switching services invalidates pending template preparation',async()=>{
  const pending=deferred()
  const view=mount(pickerPath,pickerProps({service}),pickerAPI({previewDeliveryTemplateApi:()=>pending.promise}))
  await flush();view.state.selected.value=view.state.templates.value[0];await flush()
  view.props.service={...service,id:18};await flush()
  pending.resolve({definition:'old service YAML'});await flush()
  assert.equal(view.state.preparedDefinition.value,'')
  assert.equal(view.events.length,0);view.unmount()
})

test('deleted target, mismatched API response and conflict are blocked without closing',async()=>{
  for(const scenario of ['deleted','mismatched','conflict']) {
    let writes=0
    const view=mount(pickerPath,pickerProps({service:{...service,status:scenario==='deleted'?'deleted':'active'}}),pickerAPI({
      getServiceEnvironmentConfigApi:async()=>({...config(),service_id:scenario==='mismatched'?18:17}),
      saveServiceEnvironmentConfigApi:async()=>{writes++;throw Error('配置已更新，请刷新')}
    }))
    await select(view);await view.state.create()
    assert.ok(view.state.error.value);assert.equal(view.events.length,0)
    assert.equal(writes,scenario==='conflict'?1:0);view.unmount()
  }
})

test('template workspace keeps draft when returning to the same template and protects replacement and exit',async()=>{
  let allowDiscard=false, confirmations=0
  const view=mount(pickerPath,pickerProps(),pickerAPI({listDeliveryTemplatesApi:async()=>[template(7),template(8)]}),async()=>{
    confirmations++
    if(!allowDiscard) throw 'cancel'
  })
  await flush()
  assert.equal(view.state.stage.value,'select')
  await view.state.chooseTemplate(view.state.templates.value[0]);await flush()
  assert.equal(view.state.stage.value,'select')
  view.state.nextStep()
  assert.equal(view.state.stage.value,'configure')
  view.state.name.value='gateway'
  view.state.preparedDefinition.value='my edited YAML'
  view.state.stage.value='select'
  await view.state.chooseTemplate(view.state.templates.value[0]);await flush()
  assert.equal(view.state.stage.value,'select')
  view.state.nextStep()
  assert.equal(view.state.stage.value,'configure')
  assert.equal(view.state.preparedDefinition.value,'my edited YAML')
  assert.equal(confirmations,0)
  await view.state.chooseTemplate(view.state.templates.value[1]);await flush()
  assert.equal(view.state.selected.value.id,7)
  assert.equal(view.state.preparedDefinition.value,'my edited YAML')
  await view.state.setVisible(false)
  assert.equal(view.events.length,0)
  allowDiscard=true
  await view.state.chooseTemplate(view.state.templates.value[1]);await flush()
  assert.equal(view.state.selected.value.id,8)
  assert.equal(view.state.preparedDefinition.value,'saved snapshot')
  assert.equal(view.events.length,0,'template switching never saves')
  view.unmount()
})

test('template next step waits for preparation and rejects an invalid target without saving',async()=>{
  const pending=deferred()
  let writes=0
  const view=mount(pickerPath,pickerProps(),pickerAPI({previewDeliveryTemplateApi:()=>pending.promise,createConfiguredServiceApi:async()=>writes++}))
  await flush()
  view.state.nextStep();assert.equal(view.state.stage.value,'select')
  await view.state.chooseTemplate(template(7));await flush()
  view.state.nextStep();assert.equal(view.state.stage.value,'select')
  pending.resolve({definition:'prepared YAML'});await flush()
  view.state.environmentId.value=999;await flush()
  view.state.nextStep();assert.equal(view.state.stage.value,'select')
  view.state.environmentId.value=1;await flush()
  view.state.error.value='Previous save failed; draft retained'
  view.state.nextStep();assert.equal(view.state.stage.value,'configure')
  assert.equal(writes,0);assert.equal(view.events.length,0)
  view.unmount()
})

test('target environment change retains draft and resets production confirmation',async()=>{
  const view=mount(pickerPath,pickerProps(),pickerAPI())
  await view.state.chooseTemplate(template(7));await flush()
  view.state.parameterValues.value.port='9099'
  view.state.preparedDefinition.value='custom resources'
  view.state.targetReviewed.value=true
  view.state.environmentId.value=2;await flush()
  assert.equal(view.state.parameterValues.value.port,'9099')
  assert.equal(view.state.preparedDefinition.value,'custom resources')
  assert.equal(view.state.targetReviewed.value,false)
  assert.equal(view.events.length,0)
  view.unmount()
})

const capabilityDefinition='apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: {{.Service}}\nspec:\n  template:\n    spec:\n      containers:\n        - name: app\n          image: {{.Image}}\n          ports:\n            - containerPort: 9099\n---\napiVersion: v1\nkind: Service\nmetadata:\n  name: {{.Service}}\nspec:\n  ports:\n    - port: 9099\n'
test('health parameters follow the selected probe type without hiding custom health fields',async()=>{
  const parameters=['healthCheckType','healthPath','healthPort','probeTimeoutSeconds','javaMaxRamPercentage','metricsPath'].map(key=>({key,value:'',group:'advanced'}))
  const view=mount('views/cicd/project-workspace/components/TemplateConfigurationForm.vue',{definition:capabilityDefinition,parameters,values:{healthCheckType:'none',healthPath:'/ready'},serviceCode:'api'}, {})
  const visible=key=>view.state.showParameter(parameters.find(parameter=>parameter.key===key))
  assert.equal(visible('healthCheckType'),true)
  assert.equal(visible('healthPort'),false)
  assert.equal(visible('probeTimeoutSeconds'),false)
  assert.equal(visible('metricsPath'),true,'monitoring is independent of probes')
  view.props.values.healthCheckType='tcp';await flush()
  assert.equal(visible('healthPort'),true)
  assert.equal(visible('probeTimeoutSeconds'),true)
  assert.equal(visible('healthPath'),false)
  view.props.values.healthCheckType='http';await flush()
  assert.equal(visible('healthPath'),true)
  view.props.values.healthCheckType='none';await flush()
  assert.equal(view.props.values.healthPath,'/ready','closing probes preserves the previous path')
  view.props.parameters=parameters.filter(parameter=>parameter.key!=='healthCheckType');await flush()
  assert.equal(visible('healthPath'),true,'a custom field without the standard type selector remains editable')
  view.unmount()
})

test('optional configuration completes one editor at a time, preserves previous items and cancels edits',async()=>{
  const view=mount('views/cicd/project-workspace/components/TemplateConfigurationForm.vue',{definition:capabilityDefinition,parameters:[],values:{},serviceCode:'api'}, {})
  view.state.beginOption('ingress');await flush()
  view.state.beginOption('probe');assert.equal(view.state.activeOption.value,'ingress')
  assert.throws(()=>view.exposed.getConfigured(),/先完成或取消/)
  view.state.completeOption();assert.match(view.state.optionError.value,/域名/)
  assert.equal(view.state.options.ingress,false)
  view.state.optionDraft.host='api.example.com';view.state.completeOption();await flush()
  assert.equal(view.state.activeOption.value,'');assert.equal(view.state.options.ingress,true)
  view.state.beginOption('probe');view.state.completeOption();await flush()
  const result=view.exposed.getConfigured()
  assert.equal(result.assets.routes[0].host,'api.example.com');assert.match(result.definition,/tcpSocket/)
  view.state.beginOption('ingress');view.state.optionDraft.host='discard.example.com';view.state.cancelOption()
  assert.equal(view.exposed.getConfigured().assets.routes[0].host,'api.example.com')
  view.state.removeOption('ingress');assert.equal(view.exposed.getConfigured().assets.routes.length,0)
  assert.match(view.exposed.getConfigured().definition,/tcpSocket/)
  view.unmount()
})

test('standard health parameters or native probes suppress the duplicate optional probe editor',async()=>{
  const path='views/cicd/project-workspace/components/TemplateConfigurationForm.vue'
  const view=mount(path,{definition:capabilityDefinition,parameters:[{key:'healthCheckType',value:'none'}],values:{healthCheckType:'none'},serviceCode:'api'}, {})
  assert.equal(view.state.visibleCapabilities.value.some(item=>item.key==='probe'),false)
  view.state.beginOption('probe');assert.equal(view.state.activeOption.value,'')
  assert.deepEqual(Array.from(view.state.healthNumberBounds.probeTimeoutSeconds),[1,60])
  view.props.parameters=[];await flush()
  assert.equal(view.state.visibleCapabilities.value.some(item=>item.key==='probe'),true)
  view.props.definition=templateConfigModule.exports.configureTemplate(capabilityDefinition,{...templateConfigModule.exports.emptyTemplateOptions(),probe:true},'api',{}).definition
  await flush();assert.equal(view.state.visibleCapabilities.value.some(item=>item.key==='probe'),false)
  view.unmount()
})

test('template resource editors add several entries and edit or remove only the selected entry',async()=>{
  const view=mount('views/cicd/project-workspace/components/TemplateConfigurationForm.vue',{definition:capabilityDefinition,parameters:[],values:{},serviceCode:'api'}, {})
  view.state.beginOption('ingress',0);view.state.optionDraft.host='first.example.com';view.state.completeOption()
  view.state.beginOption('ingress',0);view.state.optionDraft.host='second.example.com';view.state.completeOption()
  assert.equal(view.state.completedCapabilities.value.length,2)
  const second=view.state.selections.value[1].id
  view.state.beginOption('ingress',second);view.state.optionDraft.host='edited.example.com';view.state.completeOption()
  assert.deepEqual(Array.from(view.exposed.getConfigured().assets.routes,item=>item.host),['first.example.com','edited.example.com'])
  view.state.beginOption('secret',0);view.state.optionDraft.secretName='auth';view.state.optionDraft.secretKey='SK';view.state.optionDraft.secretEnv='API_SK';view.state.completeOption()
  view.state.beginOption('secret',0);view.state.optionDraft.secretName='auth';view.state.optionDraft.secretKey='tls.key';view.state.optionDraft.secretUsage='file';view.state.optionDraft.secretMount='/secrets/tls.key';view.state.completeOption()
  assert.equal(view.exposed.getConfigured().assets.secrets.length,2)
  view.state.removeOption('ingress',second)
  assert.equal(view.exposed.getConfigured().assets.routes.length,1)
  assert.equal(view.exposed.getConfigured().assets.secrets[1].mount_path,'/secrets/tls.key')
  view.unmount()
})

test('completed resources automatically request a final preview and hide internal metadata',async()=>{
  const requests=[]
  const view=mount(pickerPath,pickerProps(),pickerAPI({createConfiguredServiceApi:async(eid,payload,preview)=>{
    requests.push({eid,payload,preview});return{validated:true,configs:[{manifest:'# AxiOps-Variables: []\nkind: Ingress\n'}]}
  }}))
  await select(view);view.state.nextStep()
  view.state.preparedDefinition.value='# AxiOps-Variables: [{"key":"port","value":"8080"}]\nkind: Deployment'
  assert.equal(view.state.editableTemplateBody.value,'kind: Deployment')
  view.state.editableTemplateBody.value='kind: Service'
  assert.match(view.state.preparedDefinition.value,/AxiOps-Variables:.*8080/)
  await view.state.previewCompletedOptions()
  assert.equal(requests.length,1);assert.equal(requests[0].preview,true)
  assert.equal(view.state.previewManifest.value,'kind: Ingress\n')
  assert.equal(view.events.length,0)
  view.unmount()
})

test('complete preview uses the composed configuration without a save and invalidates on edits',async()=>{
  const requests=[]
  const view=mount(pickerPath,pickerProps(),pickerAPI({createConfiguredServiceApi:async(eid,payload,preview)=>{
    requests.push({eid,payload,preview});return{validated:true,configs:[{...config(eid),manifest:'rendered Deployment + Ingress'}]}
  }}))
  await select(view)
  view.state.configurationForm.value={getDraftOptions:()=>({ingress:true}),getConfigured:()=>({definition:'composed Deployment',assets:{...assets(),routes:[{host:'api.example.com'}]}})}
  await flush();await view.state.create(true)
  assert.equal(requests.length,1);assert.equal(requests[0].preview,true)
  assert.equal(requests[0].payload.assets.routes[0].host,'api.example.com')
  assert.equal(view.state.previewManifest.value,'rendered Deployment + Ingress')
  assert.equal(view.events.length,0)
  view.state.parameterValues.value.port='9000';await flush()
  assert.equal(view.state.previewManifest.value,'')
  view.unmount()
})

test('production preview needs no deployment confirmation and late preview cannot apply to a new target',async()=>{
  const pending=deferred(), requests=[]
  const view=mount(pickerPath,pickerProps({preferredEnvironmentId:2}),pickerAPI({createConfiguredServiceApi:async(eid,payload,preview)=>{requests.push({eid,preview});return pending.promise}}))
  await select(view)
  const run=view.state.create(true);await flush()
  assert.equal(requests.length,1);assert.equal(requests[0].eid,2);assert.equal(requests[0].preview,true)
  view.state.environmentId.value=1;await flush()
  pending.resolve({validated:true,configs:[{...config(2),manifest:'old production preview'}]});await run
  assert.equal(view.state.previewManifest.value,'');assert.equal(view.events.length,0)
  view.unmount()
})
