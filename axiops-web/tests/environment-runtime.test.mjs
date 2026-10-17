import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import vm from 'node:vm'
import ts from 'typescript'
import * as vue from 'vue'
import * as yaml from 'js-yaml'
import { parse, compileScript, compileTemplate } from '@vue/compiler-sfc'

const file = new URL('../src/views/cicd/project-workspace/components/EnvironmentRuntimePanel.vue', import.meta.url)
const { descriptor } = parse(readFileSync(file, 'utf8'))
assert.deepEqual(compileTemplate({ source: descriptor.template.content, filename: file.pathname, id: 'runtime' }).errors, [])
const code = ts.transpileModule(compileScript(descriptor, { id: 'runtime' }).content, { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 } }).outputText
const flush = async () => { for (let i=0; i<6; i++) await new Promise(setImmediate) }
const deferred = () => { let resolve; const promise = new Promise(r => { resolve=r }); return {resolve,promise} }
const environment = (id=1) => ({ id, name: id===1?'开发':'生产', cluster_id:id+10, namespace:id===1?'api-dev':'api-prod', environment_type:'k8s_namespace', binding_status:'validated' })
function mount(api={}) {
  const calls=[], events=[], module={exports:{}}, doc={hidden:false}; let interval, stopped=false
  const defaults=Object.fromEntries(['listEnvWorkloadsApi','listEnvPodsApi','listEnvServicesApi','listEnvIngressesApi','listEnvPVCsApi','listEnvConfigsApi'].map(name=>[name,async (...args)=>{calls.push([name,...args]);return []}]))
  const props=vue.reactive({environment:environment(),clusterName:'美国',deliveryType:'k8s_yaml'})
  vm.runInNewContext(code,{module,exports:module.exports,document:doc,setInterval(fn){interval=fn;return 1},clearInterval(){stopped=true},require(name){
    if(name==='vue')return vue
    if(name==='js-yaml')return yaml
    if(name==='@/api/axiops/env-resource')return {...defaults,...api}
    return {}
  }})
  let state
  const renderer=vue.createRenderer({createComment:()=>({}),createText:()=>({}),createElement:()=>({}),insert(){},remove(){},setText(){},setElementText(){},patchProp(){},parentNode:()=>null,nextSibling:()=>null})
  const app=renderer.createApp({setup(){state=module.exports.default.setup(props,{expose(){},emit:(...args)=>events.push(args)});return()=>null}})
  app.mount({})
  return {state,props,calls,events,doc,tick:()=>interval(),stopped:()=>stopped,unmount:()=>app.unmount()}
}

test('only selected environment/category loads and old environment response is ignored',async()=>{
  const old=deferred(),next=deferred(),calls=[]
  const view=mount({listEnvWorkloadsApi:(id,kind)=>{calls.push([id,kind]);return id===1?old.promise:next.promise}})
  assert.deepEqual(calls,[[1,'deployment']])
  view.props.environment=environment(2)
  next.resolve([{name:'prod-api',namespace:'api-prod',status:'Ready'}]);await flush()
  old.resolve([{name:'dev-api',namespace:'api-dev'}]);await flush()
  assert.equal(view.state.workloads.value[0].name,'prod-api')
  view.state.activeTab.value='storage';await flush()
  assert.equal(view.state.workloads.value.length,0)
  assert.equal(view.calls.at(-1)[0],'listEnvPVCsApi');assert.equal(view.calls.at(-1)[1],2)
  view.unmount()
})

test('query failure is shown as failure, never as an empty successful environment',async()=>{
  const view=mount({listEnvWorkloadsApi:async()=>{throw Error('无权限读取')}});await flush()
  assert.equal(view.state.error.value,'无权限读取');assert.equal(view.state.lastUpdated.value,0);assert.equal(view.state.loading.value,false)
  view.unmount()
})

test('refresh keeps current rows while loading, but switching environments clears them immediately',async()=>{
  const pending=deferred();let count=0
  const view=mount({listEnvWorkloadsApi:async()=>++count===1?[{name:'api',namespace:'api-dev'}]:pending.promise});await flush()
  const refresh=view.state.loadResources()
  assert.equal(view.state.workloads.value[0].name,'api');assert.equal(view.state.loading.value,true)
  view.props.environment=environment(2)
  assert.equal(view.state.workloads.value.length,0)
  pending.resolve([]);await refresh;await flush();view.unmount()
})

test('wrong namespace and malformed responses are rejected',async()=>{
  for(const result of [null,[{name:'other',namespace:'api-prod'}]]){
    const view=mount({listEnvWorkloadsApi:async()=>result});await flush()
    assert.match(view.state.error.value,/范围/);assert.equal(view.state.workloads.value.length,0)
    view.unmount()
  }
})

test('refresh is opt-in and pauses when hidden, busy or unmounted',async()=>{
  const view=mount();await flush();view.tick();assert.equal(view.calls.length,1)
  view.state.autoRefresh.value=true;view.tick();await flush();assert.equal(view.calls.length,2)
  view.doc.hidden=true;view.tick();assert.equal(view.calls.length,2)
  view.doc.hidden=false;view.state.loading.value=true;view.tick();assert.equal(view.calls.length,2)
  view.unmount();assert.equal(view.stopped(),true)
})

test('host environments do not query Kubernetes',async()=>{
  const view=mount();await flush();view.props.environment={...environment(3),environment_type:'host_group',host_group_ref:'servers'};await flush()
  assert.equal(view.calls.length,1);assert.equal(view.state.workloads.value.length,0)
  view.unmount()
})

test('changing environment closes detail and discards late details/logs',async()=>{
  const late=deferred()
  const view=mount({getEnvResourceDetailApi:()=>late.promise});await flush()
  const task=view.state.openDetail('pod','api','logs')
  view.props.environment=environment(2)
  late.resolve({kind:'pod',name:'api',namespace:'api-dev',yaml:'kind: Pod',events:[]});await task
  assert.equal(view.state.detailVisible.value,false);assert.equal(view.state.detail.value,null)
  view.unmount()
})

test('logs use the selected environment, pod, container and previous switch',async()=>{
  const calls=[]
  const view=mount({getEnvResourceDetailApi:async()=>({kind:'pod',name:'api',namespace:'api-dev',yaml:'spec:\n  containers:\n  - name: app\n  initContainers:\n  - name: init',events:[]}),getEnvPodLogsApi:async(...args)=>{calls.push(args);return {logs:'ready'}}})
  await view.state.openLogs({name:'api'});await flush()
  assert.deepEqual(calls[0],[1,'api','app',false])
  view.state.container.value='init';view.state.previousLogs.value=true;await flush()
  assert.deepEqual(calls.at(-1),[1,'api','init',true]);assert.equal(view.state.logs.value,'ready')
  view.unmount()
})

test('late logs from a previous environment never enter the new view',async()=>{
  const pending=deferred()
  const view=mount({getEnvResourceDetailApi:async()=>({kind:'pod',name:'api',namespace:'api-dev',yaml:'spec:\n  containers:\n  - name: app',events:[]}),getEnvPodLogsApi:()=>pending.promise})
  await view.state.openLogs({name:'api'});await flush()
  view.props.environment=environment(2);pending.resolve({logs:'old environment output'});await flush()
  assert.equal(view.state.logs.value,'');assert.equal(view.state.detailVisible.value,false)
  view.unmount()
})
