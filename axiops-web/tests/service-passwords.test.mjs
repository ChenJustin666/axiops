import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import vm from 'node:vm'
import { parse, compileScript, compileTemplate } from '@vue/compiler-sfc'
import ts from 'typescript'
import * as vue from 'vue'
import * as yaml from 'js-yaml'
import * as templateYaml from 'yaml'

const file = new URL('../src/views/cicd/project-workspace/components/ServiceResourcesEditor.vue', import.meta.url)
const { descriptor } = parse(readFileSync(file, 'utf8'))
assert.deepEqual(compileTemplate({ source: descriptor.template.content, id: 'passwords', filename: file.pathname }).errors, [])
const code = ts.transpileModule(compileScript(descriptor, { id: 'passwords' }).content, {
  compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 }
}).outputText
const empty = () => ({ command: [], args: [], workload_kind: '', workload_name: '', container: '', files: [], secrets: [], routes: [], volumes: [] })
const routeUtils = {exports:{}}, configUtils = {exports:{}}
vm.runInNewContext(ts.transpileModule(readFileSync(new URL('../src/utils/template-configuration.ts',import.meta.url),'utf8'),{compilerOptions:{module:ts.ModuleKind.CommonJS}}).outputText,
  {module:routeUtils,exports:routeUtils.exports,TextEncoder,require:name=>name==='yaml'?templateYaml:{emptyServiceAssets:empty}})
vm.runInNewContext(ts.transpileModule(readFileSync(new URL('../src/utils/service-environment-config.ts',import.meta.url),'utf8'),{compilerOptions:{module:ts.ModuleKind.CommonJS}}).outputText,
  {module:configUtils,exports:configUtils.exports,require:()=>yaml})
const secret = (name, id) => ({ name, credential_id: id, credential_key: 'password', existing_secret: '', key: 'password', env_name: name.toUpperCase() + '_PASSWORD', mount_path: '' })
const deferred = () => { let resolve; const promise = new Promise(r => { resolve = r }); return { resolve, promise } }
const flush = async () => { for (let i = 0; i < 4; i++) await new Promise(setImmediate) }

function mount(api = {}, confirm = async () => true) {
  const module = { exports: {} }, writes = [], events = []
  vm.runInNewContext(code, { module, exports: module.exports, Date, require(name) {
    if (name === 'vue') return vue
    if (name === 'js-yaml') return yaml
    if (name === '@/utils/template-configuration') return routeUtils.exports
    if (name === '@/utils/service-environment-config') return configUtils.exports
    if (name === 'element-plus') return { ElMessage: { success() {}, warning() {} }, ElMessageBox: { confirm } }
    if (name === '@/api/axiops') return {
      listCredentialsApi: async () => [],
      createCredentialApi: async body => { writes.push(JSON.parse(JSON.stringify(body))); return { id: 99 } },
      ...api
    }
    if (name === '@/api/axiops/service-assets') return { emptyServiceAssets: empty, ...api }
    return {}
  } })
  const props = vue.reactive({
    service: { id: 17, code: 'risk', name: 'Risk', project_id: 19 },
    environment: { id: 1, code: 'test', name: 'Test', namespace: 'risk-test' },
    config: { bound: true, token: 'test-token', definition: '', assets: { ...empty(), secrets: [secret('db', 42), secret('nacos', 43)] } },
    editable: true, variablesDirty: false, integrated: true
  })
  let state, exposed
  const renderer = vue.createRenderer({ createComment: () => ({}), createText: () => ({}), createElement: () => ({}), insert() {}, remove() {}, setText() {}, setElementText() {}, patchProp() {}, parentNode: () => null, nextSibling: () => null })
  const app = renderer.createApp({ setup() { state = module.exports.default.setup(props, { expose(value) { exposed = value }, emit: (...args) => events.push(args) }); return () => null } })
  app.mount({})
  return { state, exposed, props, writes, events, unmount: () => app.unmount() }
}

test('route editor resolves environment Service name and port instead of container port', async () => {
  const view=mount()
  view.props.config.definition='apiVersion: v1\nkind: Service\nmetadata:\n  name: "{{.Service}}-svc"\nspec:\n  ports:\n    - port: {{.port}}\n      targetPort: 9099\n'
  view.props.config.variables=[{key:'port',value:'80',origin:'override',secret:false}]
  await flush()
  view.state.addRoute()
  assert.equal(view.state.form.value.routes[0].service,'risk-svc')
  assert.equal(view.state.form.value.routes[0].port,80)
  assert.equal(view.state.form.value.routes[0].tls_enabled,false)
  view.unmount()
})

test('invalid route edit exits preview cleanly and never calls the save API', async () => {
  let calls=0
  const view=mount({saveServiceAssetsApi:async()=>{calls++;throw Error('should not be called')}})
  view.state.form.value.routes.push({name:'route',host:'invalid host',path:'/',service:'risk',port:80,class:'alb',tls_secret:''})
  await view.state.preview()
  assert.equal(calls,0);assert.equal(view.state.busy.value,false);assert.match(view.state.error.value,/域名/)
  view.unmount()
})

test('replacement creates a new encrypted credential and changes only the selected draft reference', async () => {
  const view = mount(), before = JSON.stringify(view.props.config.assets)
  view.state.openCredential(view.state.form.value.secrets[1])
  assert.equal(view.state.credentialDraft.value.password, '')
  view.state.credentialMode.value = 'value'
  view.state.credentialDraft.value.value = 'test-new-password'
  await view.state.createCredential()
  assert.equal(view.writes.length, 1)
  assert.equal(view.writes[0].project_id, 19)
  assert.equal(JSON.parse(view.writes[0].data).password, 'test-new-password')
  assert.equal(view.state.form.value.secrets[0].credential_id, 42)
  assert.equal(view.state.form.value.secrets[1].credential_id, 99)
  assert.equal(view.state.form.value.secrets[1].credential_key, 'password')
  assert.equal(JSON.stringify(view.props.config.assets), before, 'saved environment remains unchanged')
  assert.equal(JSON.stringify(view.exposed.getAssets()).includes('test-new-password'), false)
  assert.equal(view.state.credentialDraft.value.value, '')
  assert.equal(view.state.credentialTarget.value, null)
  assert.equal(view.events.some(event => event[0] === 'saved'), false)
  view.unmount()
})

test('new password adds its own reference without hijacking an unfinished existing reference', async () => {
  const view = mount()
  view.state.addSecret()
  const unfinished = view.state.form.value.secrets[2]
  view.state.openCredential()
  view.state.credentialMode.value = 'value'
  view.state.credentialDraft.value.value = 'new-only'
  await view.state.createCredential()
  assert.equal(unfinished.credential_id, 0)
  assert.equal(view.state.form.value.secrets.length, 4)
  assert.equal(view.state.form.value.secrets[3].credential_id, 99)
  assert.equal(view.state.form.value.secrets[3].credential_key, 'value')
  view.unmount()
})

test('failed creation leaves every binding intact and allows retry', async () => {
  const view = mount({ createCredentialApi: async () => { throw Error('权限不足') } })
  const before = JSON.stringify(view.state.form.value)
  view.state.openCredential(view.state.form.value.secrets[0])
  view.state.credentialMode.value = 'value'
  view.state.credentialDraft.value.value = 'new'
  await view.state.createCredential()
  assert.equal(JSON.stringify(view.state.form.value), before)
  assert.equal(view.state.credentialVisible.value, true)
  assert.equal(view.state.credentialBusy.value, false)
  assert.equal(view.state.credentialError.value, '权限不足')
  view.unmount()
})

test('service or environment change during creation cannot replace another target binding', async () => {
  const pending = deferred(), view = mount({ createCredentialApi: () => pending.promise })
  view.state.openCredential(view.state.form.value.secrets[0])
  view.state.credentialMode.value = 'value'
  view.state.credentialDraft.value.value = 'new'
  const saving = view.state.createCredential()
  assert.equal(await view.state.confirmLeave(), false)
  view.props.environment = { id: 2, code: 'prod', name: 'Prod', namespace: 'risk-prod', is_production: true }
  view.props.config = { ...view.props.config, token: 'prod-token', assets: { ...empty(), secrets: [secret('prod-db', 71)] } }
  await flush()
  pending.resolve({ id: 99 })
  await saving
  assert.equal(view.state.form.value.secrets.length, 1)
  assert.equal(view.state.form.value.secrets[0].credential_id, 71)
  assert.equal(view.state.credentials.value.length, 0)
  assert.equal(view.state.credentialDraft.value.value, '')
  view.unmount()
})

test('leaving an unsubmitted password dialog is guarded and discarding clears plaintext', async () => {
  let cancel = true
  const view = mount({}, async () => { if (cancel) throw Error('cancel') })
  view.state.openCredential()
  view.state.credentialDraft.value.value = 'unsubmitted'
  assert.equal(view.state.dirty.value, true)
  assert.equal(await view.state.confirmLeave(), false)
  cancel = false
  assert.equal(await view.state.confirmLeave(), true)
  assert.equal(view.state.credentialVisible.value, false)
  assert.equal(view.state.credentialDraft.value.value, '')
  assert.equal(view.writes.length, 0)
  view.unmount()
})

test('read-only or unbound service cannot create credentials', async () => {
  const view = mount()
  view.props.editable = false
  view.state.openCredential()
  assert.equal(view.state.credentialVisible.value, false)
  view.props.editable = true
  view.state.openCredential()
  view.state.credentialDraft.value.value = 'new'
  view.props.config.bound = false
  await view.state.createCredential()
  assert.equal(view.writes.length, 0)
  view.unmount()
})

test('tokens and multiline private keys use encrypted credential payloads, never service assets', async () => {
  for (const [mode, value] of [['value', 'test-sk-token'], ['file', '-----BEGIN PRIVATE KEY-----\ntest-only\n-----END PRIVATE KEY-----\n']]) {
    const view = mount()
    view.state.openCredential()
    view.state.credentialMode.value = mode
    view.state.credentialDraft.value.value = value
    await view.state.createCredential()
    assert.equal(view.writes.length, 1)
    assert.equal(JSON.parse(view.writes[0].data).value, value)
    assert.equal(JSON.stringify(view.exposed.getAssets()).includes(value), false)
    assert.equal(view.state.credentialDraft.value.value, '')
    view.unmount()
  }
})

const field = (view, key, value, usage, destination) => {
  view.state.addCredentialField()
  Object.assign(view.state.credentialFields.value.at(-1), { key, value, usage, destination })
}
const grouped = (name, key, id=42) => ({...secret(name,id),group:'auth',key,credential_key:key,env_name:key.toUpperCase()})

test('one Secret supports several arbitrary keys and multiline files without leaking plaintext into assets', async () => {
  const view = mount()
  view.state.openCredential()
  Object.assign(view.state.credentialFields.value[0], {key:'SK',value:'test-sensitive-sk',destination:'API_SK'})
  field(view,'access-key','test-sensitive-ak','env','ACCESS_KEY')
  field(view,'tls.key','private\nkey\n','file','/secrets/tls.key')
  await view.state.createCredential()
  assert.equal(view.writes.length,1)
  assert.deepEqual(JSON.parse(view.writes[0].data), {'SK':'test-sensitive-sk','access-key':'test-sensitive-ak','tls.key':'private\nkey\n'})
  const refs = view.exposed.getAssets().secrets.slice(2)
  assert.equal(refs.length,3)
  assert.equal(new Set(refs.map(item => item.group)).size,1)
  assert.equal(new Set(refs.map(item => item.name)).size,3)
  assert.ok(refs.every(item => item.credential_id === 99))
  assert.equal(refs[2].mount_path,'/secrets/tls.key')
  assert.equal(JSON.stringify(view.exposed.getAssets()).includes('test-sensitive-'),false)
  assert.equal(view.state.credentialFields.value.length,0)
  view.unmount()
})

test('group edit retains hidden keys, deletes selected keys and changes only that group', async () => {
  const derived=[]
  const view=mount({ getCredentialApi:async()=>({id:42,type:'secret',keys:['SK','token','remove','unused'],token:'a'.repeat(64)}),deriveSecretCredentialApi:async(id,body)=>{ derived.push({id,body:JSON.parse(JSON.stringify(body))});return {id:100} } })
  view.state.form.value.secrets=[grouped('sk','SK'),grouped('token','token'),grouped('remove','remove'),secret('other',43)]
  await view.state.openCredential(view.state.form.value.secrets[0])
  assert.ok(view.state.credentialFields.value.every(item=>item.value===''))
  view.state.credentialFields.value[1].value='new-test-token'
  view.state.credentialFields.value.splice(2,1)
  field(view,'access-key','new-test-ak','env','ACCESS_KEY')
  await view.state.createCredential()
  assert.deepEqual(derived,[{id:42,body:{name:derived[0].body.name,token:'a'.repeat(64),retain:['SK','unused'],values:{token:'new-test-token','access-key':'new-test-ak'}}}])
  const refs=view.exposed.getAssets().secrets
  assert.equal(refs.find(item=>item.name==='other').credential_id,43)
  assert.ok(refs.filter(item=>item.group==='auth').every(item=>item.credential_id===100))
  assert.equal(refs.some(item=>item.key==='remove'),false)
  assert.equal(JSON.stringify(refs).includes('new-test-token'),false)
  assert.equal(view.writes.length,0)
  assert.equal(view.props.config.assets.secrets[0].credential_id,42)
  view.unmount()
})

test('duplicate keys, environment names and overlapping mounts fail before credential creation', async () => {
  for (const mode of ['key','env','path','pvc']) {
    const view=mount()
    view.state.openCredential()
    Object.assign(view.state.credentialFields.value[0],{key:'token',value:'new',destination:'API_TOKEN'})
    if(mode==='key') field(view,'token','new2','env','OTHER')
    if(mode==='env') field(view,'SK','new2','env','API_TOKEN')
    if(mode==='path') { Object.assign(view.state.credentialFields.value[0],{usage:'file',destination:'/secrets/key'});field(view,'SK','new2','file','/secrets/key/child') }
    if(mode==='pvc') { view.state.form.value.volumes.push({name:'data',mount_path:'/secrets'}); Object.assign(view.state.credentialFields.value[0],{usage:'file',destination:'/secrets/key'}) }
    await view.state.createCredential()
    assert.equal(view.writes.length,0)
    assert.ok(view.state.credentialError.value)
    view.unmount()
  }
})

test('failed or late Secret derivation cannot change saved configuration or another environment', async () => {
  const pending=deferred(),view=mount({getCredentialApi:async()=>({id:42,type:'secret',keys:['SK'],token:'a'.repeat(64)}),deriveSecretCredentialApi:()=>pending.promise})
  view.state.form.value.secrets=[grouped('sk','SK')]
  await view.state.openCredential(view.state.form.value.secrets[0])
  const saving=view.state.createCredential()
  assert.equal(await view.state.confirmLeave(),false)
  view.props.environment={id:2,code:'prod',name:'Prod'}
  view.props.config={...view.props.config,token:'prod-token',assets:{...empty(),secrets:[secret('prod-db',71)]}}
  await flush()
  pending.resolve({id:100});await saving
  assert.equal(view.exposed.getAssets().secrets[0].credential_id,71)
  assert.equal(view.state.credentialSource.value,null)
  assert.equal(view.state.credentialFields.value.length,0)
  view.unmount()
})

test('resource deletion followed by addition never reuses an existing identifier', () => {
  const view=mount()
  for(const [add,key] of [['addFile','files'],['addRoute','routes'],['addVolume','volumes']]) {
    view.state[add]();view.state[add]();view.state[add]();view.state.form.value[key].splice(1,1);view.state[add]()
    assert.equal(new Set(view.state.form.value[key].map(item=>item.name)).size,3)
  }
  view.unmount()
})
