import assert from 'node:assert/strict'
import {readFileSync} from 'node:fs'
import {test} from 'node:test'
import vm from 'node:vm'
import {parse,compileScript,compileTemplate} from '@vue/compiler-sfc'
import ts from 'typescript'
import * as vue from 'vue'
import * as yaml from 'js-yaml'

const transpile=source=>ts.transpileModule(source,{compilerOptions:{module:ts.ModuleKind.CommonJS,target:ts.ScriptTarget.ES2022}}).outputText
const utils={exports:{}}
vm.runInNewContext(transpile(readFileSync(new URL('../src/utils/helm-service-config.ts',import.meta.url),'utf8')),{module:utils,exports:utils.exports,require:()=>yaml})
const chart=JSON.stringify({schema_version:1,chart_path:'.',files:utils.exports.starterChart('api')})
const configuration=(id=1)=>({service_id:17,environment_id:id,format:'helm_chart',bound:true,revision_id:10,latest_revision_id:11,definition:chart,token:String(id).repeat(64),variables:[],helm:{values:`replicaCount: ${id}\nhost: ${id===1?'qa':'prod'}.example.com\nresources:\n  limits:\n    memory: ${id===1?'1Gi':'4Gi'}\n`,image:{repository_path:'image.repository',tag_path:'image.tag',ref_path:''}}})
const file=new URL('../src/views/cicd/project-workspace/components/HelmServicePanel.vue',import.meta.url)
const {descriptor}=parse(readFileSync(file,'utf8'))
assert.deepEqual(compileTemplate({source:descriptor.template.content,id:'helm',filename:file.pathname}).errors,[])
const code=transpile(compileScript(descriptor,{id:'helm'}).content)
const flush=async()=>{for(let i=0;i<6;i++)await new Promise(setImmediate)}
function mount(api={},canEdit=true){
 const calls=[],events=[],module={exports:{}}
 const defaults={
  listServiceRevisionsApi:async()=>[{id:10,revision_number:1,definition:chart},{id:11,revision_number:2,definition:chart}],
  getServiceEnvironmentConfigApi:async eid=>configuration(eid),
  saveServiceEnvironmentConfigApi:async(sid,body,preview)=>{calls.push({sid,body:JSON.parse(JSON.stringify(body)),preview});return{saved:!preview,validated:true,configs:[{...configuration(body.targets[0].environment_id),manifest:'apiVersion: v1\nkind: Service\nmetadata:\n  name: api\n'}]}},
  validateHelmChartApi:async()=>({definition:chart}),
  createServiceRevisionApi:async()=>({id:12,revision_number:3,definition:chart}),...api
 }
 vm.runInNewContext(code,{module,exports:module.exports,require(name){if(name==='vue')return vue;if(name==='@/api/axiops')return defaults;if(name==='@/utils/helm-service-config')return utils.exports;if(name==='element-plus')return{ElMessage:{success(){},warning(){}},ElMessageBox:{confirm:async()=>true}};return{}}})
 const props=vue.reactive({service:{id:17,project_id:19,code:'api',name:'API',current_revision_id:11},environment:{id:1,name:'测试环境',namespace:'app-qa'},environments:[{id:1,name:'测试环境',namespace:'app-qa'},{id:2,name:'生产环境',namespace:'app-prod',is_production:true}],config:configuration(),canEdit,canEditChart:canEdit})
 let state
 const renderer=vue.createRenderer({createComment:()=>({}),createText:()=>({}),createElement:()=>({}),insert(){},remove(){},setText(){},setElementText(){},patchProp(){},parentNode:()=>null,nextSibling:()=>null})
 const app=renderer.createApp({setup(){state=module.exports.default.setup(props,{expose(){},emit:(...args)=>events.push(args)});return()=>null}});app.mount({})
 return{state,props,calls,events,unmount:()=>app.unmount()}
}

test('native values keep types; sync copies only selected fields and preserves target domain',()=>{
 const result=yaml.load(utils.exports.syncHelmValues(configuration(1).helm.values,configuration(2).helm.values,['resources']))
 assert.equal(result.host,'prod.example.com');assert.equal(result.replicaCount,2);assert.equal(result.resources.limits.memory,'1Gi')
 assert.throws(()=>utils.exports.valuesObject('a: 1\na: 2'));assert.throws(()=>utils.exports.valuesObject('---\na: 1\n---\nb: 2'))
 assert.throws(()=>utils.exports.validateHelmSettings({values:'{}',image:{repository_path:'image',tag_path:'image.tag'}}))
 assert.ok(utils.exports.starterChart().some(f=>f.path==='templates/service.yaml'))
})
test('empty overrides keep inheriting chart defaults in both new and saved environments',()=>{
 const c={...configuration(),helm:{...configuration().helm,values:'{}\n'}}
 assert.equal(utils.exports.initialHelmValues(c),'{}\n')
 assert.equal(utils.exports.initialHelmValues({...c,bound:false}),'{}\n')
})

test('deep sync preserves target-only siblings and treats arrays as whole values',()=>{
 const source='resources:\n  limits:\n    memory: 1Gi\n    cpu: 1\nenv:\n- name: A\n  value: qa\n'
 const target='resources:\n  limits:\n    memory: 8Gi\n    cpu: 4\nhpa:\n  enabled: true\nenv:\n- name: B\n  value: prod\n'
 const paths=Array.from(utils.exports.helmValuesPaths(source,['resources.limits.cpu']))
 assert.deepEqual(paths,['/resources/limits/memory','/env'])
 const result=yaml.load(utils.exports.syncHelmValues(source,target,['/resources/limits/memory']))
 assert.equal(result.resources.limits.cpu,4);assert.equal(result.resources.limits.memory,'1Gi');assert.equal(result.hpa.enabled,true);assert.equal(result.env[0].name,'B')
 assert.throws(()=>utils.exports.syncHelmValues('constructor: 1',target,['constructor']))
})

test('Git three-way merge preserves independent edits, deletion and null without array merging',()=>{
 const merge=utils.exports.mergeHelmValuesSource
 const base='resources:\n  limits:\n    cpu: 1\n    memory: 2Gi\nremoved: true\nnullable: true\nenv: [a]\n'
 const local='resources:\n  limits:\n    cpu: 1\n    memory: 4Gi\nremoved: true\nnullable: true\nenv: [local]\n'
 const remote='resources:\n  limits:\n    cpu: 2\n    memory: 2Gi\nnullable: null\nenv: [remote]\n'
 const result=merge(base,local,remote)
 const obj=yaml.load(result.values)
 assert.equal(obj.resources.limits.cpu,2);assert.equal(obj.resources.limits.memory,'4Gi');assert.equal('removed' in obj,false);assert.equal(obj.nullable,null)
 assert.deepEqual(Array.from(result.conflicts,c=>c.path),['/env'])
 assert.deepEqual(yaml.load(merge(base,local,remote,{'/env':'remote'}).values).env,['remote'])
 const deleted=merge('item: {a: 1}','{}','item: {a: 2}')
 assert.equal(deleted.conflicts[0].path,'/item');assert.equal('item' in yaml.load(deleted.values),false)
})
test('environment values preview then save the exact target without deploying',async()=>{
 const v=mount();await flush();v.state.settings.value.values+='port: 9090\n';await v.state.preview()
 assert.equal(v.calls.length,1);assert.equal(v.calls[0].preview,true);assert.equal(v.calls[0].body.targets[0].environment_id,1)
 await v.state.commit();assert.equal(v.calls.length,2);assert.deepEqual(v.calls[1].body,v.calls[0].body);assert.equal(v.events.some(e=>e[0]==='saved'),true);v.unmount()
})
test('sync preserves production image mapping, domain and Chart until explicitly selected',async()=>{
 const v=mount();await flush();v.state.openSync();v.state.targetId.value=2;v.state.syncKeys.value=['resources'];await v.state.previewSync()
 assert.equal(v.calls.length,1);const req=v.calls[0].body;assert.equal(req.targets[0].environment_id,2);assert.equal(req.source.environment_id,1);assert.equal(req.revision_id,10)
 assert.equal(yaml.load(req.helm.values).host,'prod.example.com');assert.equal(yaml.load(req.helm.values).resources.limits.memory,'1Gi')
 await v.state.commit();assert.equal(v.calls.length,1,'production confirmation required');v.state.productionConfirmed.value=true;await v.state.commit();assert.equal(v.calls.length,2);v.unmount()
})
test('changed values cannot commit an earlier preview',async()=>{
 const v=mount();await flush();await v.state.preview();v.state.settings.value.values+='port: 9090\n';await v.state.commit();assert.equal(v.calls.length,1);assert.match(v.state.error.value,/配置已变化/);v.unmount()
})
test('Chart save creates a revision and does not bind environments; file selection stays on Chart',async()=>{
 const v=mount();await flush();v.state.selectFile('values.yaml');assert.equal(v.state.tab.value,'chart');v.state.chartEditing.value=true;v.state.files.value[0].content+='# changed\n';await v.state.saveChart()
 assert.equal(v.state.revisionId.value,12);assert.equal(v.calls.length,0);assert.equal(v.events.some(e=>e[0]==='saved'),false);assert.equal(v.events.some(e=>e[0]==='file-change'&&e[1]==='values.yaml'),true);assert.equal(v.state.dirty.value,true);v.unmount()
})
test('readonly actions cannot save native values or Chart',async()=>{
 const v=mount({},false);await flush();await v.state.preview();v.state.chartEditing.value=true;await v.state.saveChart();v.state.targetId.value=2;v.state.syncKeys.value=['resources'];await v.state.previewSync();assert.equal(v.calls.length,0);v.unmount()
})

test('repository context entry loads current revisions but preserves unsaved edits',async()=>{
 const v=mount();await flush();v.props.service.code_source_id=9;let opened=0;v.state.repositorySync.value={open:()=>opened++}
 await v.state.openRepositorySync();assert.equal(opened,1);assert.equal(v.state.latestRevision.value.id,11)
 v.state.settings.value.values+='custom: true\n';await v.state.openRepositorySync();assert.equal(opened,1)
 assert.match(v.state.settings.value.values,/custom: true/);v.unmount()
})
test('late response for a previous environment cannot open a preview',async()=>{
 let resolve;const waiting=new Promise(r=>resolve=r)
 const v=mount({saveServiceEnvironmentConfigApi:()=>waiting});await flush();const pending=v.state.preview();v.props.config=configuration(2);v.props.environment=v.props.environments[1];await flush();resolve({configs:[{manifest:'old manifest'}]});await pending;assert.equal(v.state.previewVisible.value,false);assert.equal(v.state.pending.value,null);assert.equal(v.state.settings.value.values,configuration(2).helm.values);v.unmount()
})
test('malformed Chart reports a load error without an unhandled exception',async()=>{
 const v=mount();await flush();v.props.config={...configuration(),token:'z'.repeat(64),definition:'invalid json'};await flush();assert.ok(v.state.error.value);assert.equal(v.state.loading.value,false);v.unmount()
})

test('repository Chart exposes every real file and selected file opens the Chart editor',async()=>{
 const v=mount();await flush()
 const extra=JSON.stringify({schema_version:1,files:[...utils.exports.chartFiles(chart),{path:'templates/_helpers.tpl',content:'{{ define "app" }}app{{ end }}'},{path:'templates/ingress.yaml',content:'kind: Ingress'}]})
 v.props.activeFile='templates/_helpers.tpl';v.props.config={...configuration(),definition:extra,token:'new'};await flush()
 assert.equal(v.state.tab.value,'chart');assert.equal(v.state.activePath.value,'templates/_helpers.tpl')
 const emitted=v.events.filter(e=>e[0]==='files-change').at(-1)[1]
 assert.equal(emitted.includes('templates/ingress.yaml'),true)
 const tree=utils.exports.chartFileTree(utils.exports.chartFiles(extra))
 assert.equal(tree.find(n=>n.path==='templates').children.some(n=>n.path==='templates/_helpers.tpl'),true)
 v.unmount()
})
