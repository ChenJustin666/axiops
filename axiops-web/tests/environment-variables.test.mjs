import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import vm from 'node:vm'
import ts from 'typescript'
import * as vue from 'vue'
import * as yaml from 'js-yaml'
import { parse, compileScript, compileTemplate } from '@vue/compiler-sfc'
const file=new URL('../src/views/cicd/project-workspace/components/EnvironmentConfigDialog.vue',import.meta.url)
const {descriptor}=parse(readFileSync(file,'utf8'))
assert.deepEqual(compileTemplate({source:descriptor.template.content,filename:file.pathname,id:'variables'}).errors,[])
const code=ts.transpileModule(compileScript(descriptor,{id:'variables'}).content,{compilerOptions:{module:ts.ModuleKind.CommonJS,target:ts.ScriptTarget.ES2022}}).outputText
const flush=async()=>{for(let i=0;i<6;i++)await new Promise(setImmediate)}
const deferred=()=>{let resolve;const promise=new Promise(r=>resolve=r);return {resolve,promise}}
const set=(id)=>({id:id+10,environment_id:id,service_id:0,revision:1,items:[{key:'port',value:'8080',credential_id:0,is_secret:false}]})
function mount(api={},confirm=async()=>true){
 const module={exports:{}},calls=[],success=[],events=[]
 const defaults={getEnvironmentVariableSetApi:async id=>set(id),listCredentialsApi:async()=>[],listEnvironmentResourcesApi:async()=>[],updateEnvironmentVariableSetApi:async(id,body)=>{calls.push({id,body});return {...set(id-10),revision:2,items:body.items}},...api}
 vm.runInNewContext(code,{module,exports:module.exports,require(name){if(name==='vue')return vue;if(name==='js-yaml')return yaml;if(name==='@/api/axiops')return defaults;if(name==='@/utils/http/error')return {HttpError:Error};if(name==='element-plus')return {ElMessage:{warning(){},error(){},success:m=>success.push(m)},ElMessageBox:{confirm}};return {}}})
 const props=vue.reactive({visible:false,environment:{id:1,name:'开发',project_id:19},services:[],editable:true,initialTab:'variables'})
 let state
 const renderer=vue.createRenderer({createComment:()=>({}),createText:()=>({}),createElement:()=>({}),insert(){},remove(){},setText(){},setElementText(){},patchProp(){},parentNode:()=>null,nextSibling:()=>null})
 const app=renderer.createApp({setup(){state=module.exports.default.setup(props,{expose(){},emit:(...args)=>events.push(args)});return()=>null}});app.mount({});props.visible=true
 return {state,props,calls,success,events,unmount:()=>app.unmount()}
}
test('global variable diff, restore and save use the target set revision',async()=>{
 const view=mount();await flush();assert.equal(view.state.variableChanges.value.length,0)
 view.state.variableItems.value[0].value='9000';assert.equal(view.state.variableChanges.value[0].before,'8080');assert.equal(view.state.variableChanges.value[0].after,'9000')
 view.state.restoreVariables();assert.equal(view.state.variableChanges.value.length,0)
 view.state.variableItems.value[0].value='9001';await view.state.saveVariables()
 assert.equal(view.calls[0].id,11);assert.equal(view.calls[0].body.revision,1);assert.match(view.success[0],/重新提交部署/);view.unmount()
})
test('slow old environment cannot replace or save the new environment variable set',async()=>{
 const old=deferred();const view=mount({getEnvironmentVariableSetApi:id=>id===1?old.promise:Promise.resolve(set(id))});await flush()
 view.props.environment={id:2,name:'QA',project_id:19};await flush();old.resolve(set(1));await flush()
 assert.equal(view.state.variableSet.value.environment_id,2);view.state.variableItems.value[0].value='9000';await view.state.saveVariables();assert.equal(view.calls[0].id,12);view.unmount()
})
test('load errors block saving rather than retaining another environment values',async()=>{
 const view=mount({getEnvironmentVariableSetApi:async()=>{throw Error('offline')}});await flush();await view.state.saveVariables()
 assert.equal(view.state.variableError.value,'offline');assert.equal(view.state.variableSet.value,null);assert.equal(view.calls.length,0);view.unmount()
})
test('production cancellation and dirty close preserve edits without saving',async()=>{
 const view=mount({},async()=>{throw Error('cancel')});await flush();view.props.environment.is_production=true;view.state.variableItems.value[0].value='9000'
 await view.state.saveVariables();assert.equal(view.calls.length,0);await view.state.requestClose();assert.equal(view.events.length,0);assert.equal(view.state.variableItems.value[0].value,'9000');view.unmount()
})
test('mismatched save response never reports success',async()=>{
 const view=mount({updateEnvironmentVariableSetApi:async()=>({...set(2),revision:2})});await flush();view.state.variableItems.value[0].value='9000';await view.state.saveVariables()
 assert.equal(view.success.length,0);assert.match(view.state.variableError.value,/无法确认/);view.unmount()
})
