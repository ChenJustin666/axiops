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
const chartFiles=[{path:'Chart.yaml',content:'apiVersion: v2\nname: application\nversion: 1.0.0\n'},{path:'values.yaml',content:'replicaCount: 1\n'}]
const template={id:31,revision:2,kind:'helm',name:'通用应用',files:[...chartFiles.map(f=>({...f})),{path:'templates/ingress.yaml',content:'route'}],variables:[]}
const config=()=>({service_id:17,environment_id:1,revision_id:10,latest_revision_id:10,token:'a'.repeat(64),bound:true,definition:JSON.stringify({schema_version:1,files:chartFiles}),variables:[],helm:{values:'replicaCount: 3\nhost: qa.example.com\n',image:{repository_path:'image.repository',tag_path:'image.tag',ref_path:''}}})
const flush=async()=>{for(let i=0;i<5;i++)await new Promise(setImmediate)}
const copy=value=>JSON.parse(JSON.stringify(value))
function mount(path,initial,overrides={}){
 const calls=[],events=[],module={exports:{}}
 const defaults={
  getHelmTemplateStatusApi:async()=>({mode:'template',template:{id:31,revision:1},name:'通用应用',latest_revision:2,update_available:true}),
  listDeliveryTemplatesApi:async()=>[template],
  listDeliveryTemplateVersionsApi:async()=>[{revision:2,name:'通用应用'},{revision:1,name:'通用应用'}],
  getDeliveryTemplateVersionApi:async(id,revision)=>({template:{...copy(template),id,revision}}),
  saveServiceEnvironmentConfigApi:async(sid,request,preview)=>{calls.push({sid,request:copy(request),preview});return{saved:!preview,validated:true,configs:[{service_id:sid,environment_id:request.targets[0].environment_id,manifest:'apiVersion: v1\nkind: Service\nmetadata:\n  name: gateway\n'}]}},
  listDeliveryTemplateUsagesApi:async()=>[{service_name:'gateway',environment_name:'QA',revision:1}],
  listCredentialsApi:async()=>[{id:100,name:'App auth',keys:['password','SK'],type:'secret'}],
  ...overrides
 }
 const file=new URL(`../src/views/cicd/${path}`,import.meta.url),{descriptor}=parse(readFileSync(file,'utf8'))
 assert.deepEqual(compileTemplate({source:descriptor.template.content,id:path,filename:file.pathname}).errors,[])
 vm.runInNewContext(transpile(compileScript(descriptor,{id:path}).content),{module,exports:module.exports,require(name){if(name==='vue')return vue;if(name==='@/api/axiops')return defaults;if(name==='@/utils/helm-service-config')return utils.exports;if(name==='element-plus')return{ElMessage:{success(){}}};return{}}})
 const props=vue.reactive(initial);let state
 const renderer=vue.createRenderer({createComment:()=>({}),createText:()=>({}),createElement:()=>({}),insert(){},remove(){},setText(){},setElementText(){},patchProp(){},parentNode:()=>null,nextSibling:()=>null})
 const app=renderer.createApp({setup(){state=module.exports.default.setup(props,{expose(){},emit:(...args)=>events.push(args)});return()=>null}});app.mount({})
 return{state,props,calls,events,unmount:()=>app.unmount()}
}
const upgrade=(api={},production=false,canEdit=true)=>mount('project-workspace/components/HelmTemplateUpgrade.vue',{service:{id:17,code:'gateway'},environment:{id:1,name:production?'生产':'测试',namespace:production?'app-prod':'app-qa',is_production:production},config:config(),settings:config().helm,revisionId:10,canEdit,disabled:false},api)

test('template update previews exact version with target values and saves atomically without a separate revision call',async()=>{
 const v=upgrade();await flush();await v.state.open()
 assert.equal(v.state.selected.value.revision,2);assert.equal(v.state.differences.value[0].path,'templates/ingress.yaml')
 await v.state.preview();assert.equal(v.calls.length,1);const request=v.calls[0].request
 assert.deepEqual(request.template,{id:31,revision:2});assert.equal(request.revision_id,0);assert.deepEqual(request.targets,[{environment_id:1,token:'a'.repeat(64)}]);assert.equal(request.helm.values,config().helm.values)
 await v.state.save();assert.equal(v.calls.length,2);assert.deepEqual(v.calls[1].request,request);assert.ok(v.events.some(e=>e[0]==='saved'));v.unmount()
})
test('custom Chart replacement requires explicit confirmation',async()=>{
 const v=upgrade({getHelmTemplateStatusApi:async()=>({mode:'custom',template:{id:31,revision:1}})});await flush();await v.state.open();await v.state.preview();assert.equal(v.calls.length,0)
 v.state.replaceConfirmed.value=true;await v.state.preview();assert.equal(v.calls.length,1);v.unmount()
})
test('production Chart update requires confirmation after actual YAML preview',async()=>{
 const v=upgrade({},true);await flush();await v.state.open();await v.state.preview();await v.state.save();assert.equal(v.calls.length,1)
 v.state.productionConfirmed.value=true;await v.state.save();assert.equal(v.calls.length,2);v.unmount()
})
test('configuration changes invalidate a template preview and never save it',async()=>{
 const v=upgrade();await flush();await v.state.open();await v.state.preview();v.props.settings.values+='port: 9099\n';await v.state.save();assert.equal(v.calls.length,1);assert.match(v.state.error.value,/重新预览/);v.unmount()
})
test('late template preview cannot cross environment selection',async()=>{
 let resolve;const v=upgrade({saveServiceEnvironmentConfigApi:()=>new Promise(r=>resolve=r)});await flush();await v.state.open();const promise=v.state.preview()
 v.props.config={...config(),environment_id:2,token:'b'.repeat(64)};await flush();resolve({saved:false,validated:true,configs:[{service_id:17,environment_id:1,manifest:'stale'}]});await promise
 assert.equal(v.state.visible.value,false);assert.equal(v.state.pending.value,undefined);assert.equal(v.state.rendered.value,'');v.unmount()
})
test('out-of-order version responses preserve the newest selection',async()=>{
 let resolve;const v=upgrade({getDeliveryTemplateVersionApi:async(id,revision)=>revision===1?new Promise(r=>resolve=r):({template:{...copy(template),revision}})});await flush();await v.state.open()
 const old=v.state.selectVersion(1);await v.state.selectVersion(2);resolve({template:{...copy(template),revision:1}});await old;assert.equal(v.state.selected.value.revision,2);v.unmount()
})
test('readonly cannot open, preview or save template upgrades',async()=>{
 const v=upgrade({},false,false);await flush();await v.state.open();v.state.selected.value=copy(template);v.state.templateId.value=31;v.state.version.value=2;await v.state.preview();assert.equal(v.calls.length,0);assert.equal(v.state.visible.value,false);v.unmount()
})
test('template history restores old content using current optimistic revision and loads consumers lazily',async()=>{
 let usages=0;const v=mount('template-library/TemplateHistory.vue',{modelValue:true,template:{...copy(template),revision:5},canEdit:true},{listDeliveryTemplateUsagesApi:async()=>{usages++;return[]}});await flush()
 assert.equal(usages,0);v.state.restore();const restore=v.events.find(e=>e[0]==='restore');assert.equal(restore[1].revision,5);assert.equal(restore[1].id,31)
 v.state.tab.value='usages';await v.state.loadUsages();await v.state.loadUsages();assert.equal(usages,1);v.unmount()
})
test('credential editor loads metadata only and sends separate password and SK references',async()=>{
 let loaded=0;const v=mount('project-workspace/components/HelmCredentialBindings.vue',{modelValue:{},config:config(),projectId:19,disabled:false},{listCredentialsApi:async()=>{loaded++;return[{id:100,name:'Auth',keys:['password','SK']}]}})
 assert.equal(loaded,0);await v.state.loadCredentials(['credentials']);assert.equal(loaded,1)
 v.state.add();Object.assign(v.state.rows.value[0],{path:'secrets.auth.stringData.password',credential_id:100,key:'password'});v.state.add();Object.assign(v.state.rows.value[1],{path:'secrets.auth.stringData.SK',credential_id:100,key:'SK'});v.state.update()
 const delta=v.events.filter(e=>e[0]==='update:modelValue').at(-1)[1];assert.equal(delta['secrets.auth.stringData.password'].key,'password');assert.equal(delta['secrets.auth.stringData.SK'].key,'SK');assert.ok(!JSON.stringify(delta).includes('plaintext'))
 v.state.rows.value[1].path=v.state.rows.value[0].path;v.state.update();assert.match(v.state.error.value,/不能重复/)
 v.props.config={...config(),token:'b'.repeat(64)};await flush();assert.equal(v.state.rows.value.length,0);v.unmount()
})
