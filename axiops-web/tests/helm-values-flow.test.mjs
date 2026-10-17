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
const flush=async()=>{for(let i=0;i<5;i++)await new Promise(setImmediate)}
const service={id:17,project_id:19,name:'API',code:'api',current_revision_id:10,delivery_type:'helm'}
const helm=()=>({values:'{}\n',image:{repository_path:'image.repository',tag_path:'image.tag',ref_path:''}})
function mount(name,initial,api={}){
 const file=new URL(name==='ProjectWizard' ? '../src/views/cicd/project-wizard/index.vue' : `../src/views/cicd/project-workspace/components/${name}.vue`,import.meta.url)
 const {descriptor}=parse(readFileSync(file,'utf8'))
 assert.deepEqual(compileTemplate({source:descriptor.template.content,id:name,filename:file.pathname}).errors,[])
 const module={exports:{}}
 vm.runInNewContext(transpile(compileScript(descriptor,{id:name}).content),{module,exports:module.exports,require(name){if(name==='vue')return vue;if(name==='@/api/axiops')return api;if(name==='@/utils/helm-service-config')return utils.exports;if(name==='element-plus')return{ElMessage:{success(){},error(){},warning(){}}};if(name==='vue-router')return{useRoute:()=>({params:{projectId:'19'},query:{}}),useRouter:()=>({replace:async()=>{},push:async()=>{}})};if(name==='@/composables/useAxiOpsAccess')return{useAxiOpsAccess:()=>({can:()=>true,load:async()=>{}})};return{}}})
 const events=[],props=vue.reactive(initial),exposed={};let state
 const renderer=vue.createRenderer({createComment:()=>({}),createText:()=>({}),createElement:()=>({}),insert(){},remove(){},setText(){},setElementText(){},patchProp(){},parentNode:()=>null,nextSibling:()=>null})
 const app=renderer.createApp({setup(){state=module.exports.default.setup(props,{expose:v=>Object.assign(exposed,v),emit:(...args)=>{events.push(args);if(args[0]==='update:modelValue')props.modelValue=args[1]}});return()=>null}});app.mount({})
 return{props,state,events,exposed,unmount:()=>app.unmount()}
}
const sourceProps=()=>({modelValue:helm(),service,files:[{path:'values.yaml',content:'replicas: 1\n'},{path:'values-prod.yaml',content:'replicas: 3\n'}],environments:[],disabled:false})

test('Chart values import changes only a draft and clears unrelated Git provenance',async()=>{
 const v=mount('HelmValuesSources',sourceProps(),{listCodeSourcesApi:async()=>[]})
 await v.state.open();v.state.filePath.value='values-prod.yaml';v.state.chooseFile();v.state.apply()
 assert.equal(v.props.modelValue.values,'replicas: 3\n');assert.equal(v.props.modelValue.source,undefined)
 assert.equal(v.events.filter(e=>e[0]==='update:modelValue').length,1);v.unmount()
})

test('Git updates require explicit conflict choices and preserve local unrelated fields',async()=>{
 const props=sourceProps(),src={code_source_id:9,repo_path:'team/config',ref:'main',path:'values.yaml',commit_id:'old',base:'replicas: 1\nmemory: 2Gi\n'}
 props.modelValue={...helm(),values:'replicas: 2\nmemory: 4Gi\n',source:src}
 const remote={...src,commit_id:'new',base:'replicas: 3\nmemory: 2Gi\nregion: us\n'}
 const v=mount('HelmValuesSources',props,{listCodeSourcesApi:async()=>[],readHelmValuesApi:async()=>remote})
 await v.state.refresh();await flush();assert.equal(v.state.unresolved.value,true)
 v.state.apply();assert.equal(v.events.filter(e=>e[0]==='update:modelValue').length,0)
 v.state.choices['/replicas']='remote';v.state.apply()
 assert.deepEqual(yaml.load(v.props.modelValue.values),{replicas:3,memory:'4Gi',region:'us'})
 assert.equal(v.props.modelValue.source.commit_id,'new');assert.equal(v.props.modelValue.source.base,remote.base);v.unmount()
})

test('late Git reads and changed local drafts cannot apply an earlier import',async()=>{
 let resolve;const pending=new Promise(r=>resolve=r)
 const v=mount('HelmValuesSources',sourceProps(),{listCodeSourcesApi:async()=>[],readHelmValuesApi:()=>pending})
 await v.state.open();v.state.mode.value='git';await flush();const read=v.state.readGit();v.state.mode.value='chart';await flush()
 resolve({base:'replicas: 3',commit_id:'new'});await read
 assert.equal(v.state.candidate.value,undefined);assert.equal(v.state.working.value,false)
 v.state.filePath.value='values-prod.yaml';v.state.chooseFile();v.props.modelValue.values='replicas: 9\n';v.state.apply()
 assert.equal(v.events.filter(e=>e[0]==='update:modelValue').length,0);assert.match(v.state.error.value,/草稿已变化/);v.unmount()
})

test('a values import belonging to a previous environment is discarded on unmount',async()=>{
 const v=mount('HelmValuesSources',sourceProps(),{listCodeSourcesApi:async()=>[]})
 await v.state.open();v.state.filePath.value='values-prod.yaml';v.state.chooseFile();v.unmount();v.state.apply()
 assert.equal(v.events.filter(e=>e[0]==='update:modelValue').length,0)
 assert.equal(v.events.filter(e=>e[0]==='busy-change').at(-1)[1],false)
})

test('shared values preview reuses exact service tokens and production needs confirmation',async()=>{
 const calls=[];const original={values:'{}\n',revision:7,services:[{service_id:17,name:'API',token:'a'.repeat(64)}]}
 const v=mount('HelmSharedValuesEditor',{environment:{id:2,name:'生产',is_production:true},canEdit:true,disabled:false},{getHelmSharedValuesApi:async()=>original,saveHelmSharedValuesApi:async(eid,data,preview)=>{calls.push({eid,data:JSON.parse(JSON.stringify(data)),preview});return{...original,saved:!preview,validated:true,services:[]}}})
 await v.state.open();v.state.values.value='region: prod\n';await v.state.check();await v.state.save();assert.equal(calls.length,1)
 v.state.confirmed.value=true;await v.state.save();assert.equal(calls.length,2);assert.deepEqual(calls[0].data,calls[1].data);assert.equal(calls[1].eid,2);assert.equal(calls[1].data.tokens[17],original.services[0].token)
 assert.equal(v.events.some(e=>e[0]==='saved'),true);v.unmount()
})

test('new environment selects fixed Chart revisions without copying default values',async()=>{
 const chart=JSON.stringify({schema_version:1,files:utils.exports.starterChart()})
 const v=mount('HelmEnvironmentServices',{modelValue:[],services:[service],environments:[],disabled:false},{listServiceRevisionsApi:async()=>[{id:10,revision_number:1,definition:chart}]})
 await v.state.selectServices([17]);assert.equal(v.props.modelValue[0].revision_id,10);assert.equal(v.props.modelValue[0].helm.values,'{}\n')
 v.state.update(v.props.modelValue[0],{...v.props.modelValue[0].helm,values:'replicas: 3\n'})
 assert.equal(v.props.modelValue[0].helm.values,'replicas: 3\n');await v.state.selectServices([]);assert.equal(v.props.modelValue.length,0);v.unmount()
})

test('environment creation preview is read-only and cancellation creates nothing',async()=>{
 const calls=[],input={name:'生产',code:'prod',namespace:'api-prod',is_production:true,deploy_approval:false,helm_services:[{service_id:17,revision_id:10,helm:helm()}]}
 const v=mount('HelmEnvironmentReview',{projectId:19,services:[service]},{createEnvironmentApi:async(pid,data,preview)=>{calls.push({pid,data,preview});return{saved:false,configs:[{service_id:17,final_values:'replicas: 1',manifest:'kind: Deployment'}]}}})
 const pending=v.exposed.review(input);await flush();assert.equal(v.state.visible.value,true);assert.equal(calls[0].preview,true)
 input.namespace='changed';assert.equal(v.state.input.value.namespace,'api-prod')
 v.state.finish(false);assert.equal(await pending,false);assert.equal(calls.length,1);v.unmount()
})

function wizardApi(calls,bound=true){return{
 getProjectApi:async()=>({id:19,code:'app',delivery_type:'helm',initialization_status:'initializing',initialization_step:3}),
 listAllServicesApi:async()=>[{...service,id:1,name:'A',code:'a',current_revision_id:99},{...service,id:2,name:'B',code:'b',current_revision_id:100}],
 listEnvironmentsApi:async()=>[{id:3,name:'QA',code:'qa',namespace:'app-qa'}],listClustersApi:async()=>[],listCodeSourcesApi:async()=>[],listWorkflowsApi:async()=>[],
 getServiceEnvironmentConfigApi:async(eid,sid)=>({bound:bound && sid===2,revision_id:47}),createWorkflowApi:async(pid,data)=>{calls.push({pid,data});return{id:4}}
}}
test('initial workflow uses a service bound to the environment and its pinned Chart revision',async()=>{
 const calls=[],v=mount('ProjectWizard',{},wizardApi(calls));await flush();await v.state.generateWorkflows()
 assert.equal(calls.length,1);assert.equal(calls[0].data.target_environment_id,3)
 const deploy=calls[0].data.definition.nodes.find(n=>n.type==='helm_deploy')
 assert.equal(deploy.config.service_id,2);assert.equal(deploy.config.service_revision_id,47);assert.equal(deploy.config.release_name,'b')
 assert.equal(calls[0].data.definition.nodes.find(n=>n.type==='build').config.service_id,2);v.unmount()
})
test('initial workflow refuses to point an empty Helm environment at an unbound service',async()=>{
 const calls=[],v=mount('ProjectWizard',{},wizardApi(calls,false));await flush();await v.state.generateWorkflows()
 assert.equal(calls.length,0);assert.equal(v.state.workflowGenerating.value,false);v.unmount()
})
