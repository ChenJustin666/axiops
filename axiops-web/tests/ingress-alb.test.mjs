import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import vm from 'node:vm'
import ts from 'typescript'
import * as vue from 'vue'
import * as yaml from 'yaml'
import { parse, compileScript, compileTemplate } from '@vue/compiler-sfc'

const module = { exports: {} }
const emptyServiceAssets = () => ({ command: [], args: [], files: [], secrets: [], routes: [], volumes: [] })
vm.runInNewContext(ts.transpileModule(readFileSync(new URL('../src/utils/template-configuration.ts', import.meta.url), 'utf8'), { compilerOptions: { module: ts.ModuleKind.CommonJS } }).outputText,
  { module, exports: module.exports, TextEncoder, require: name => name === 'yaml' ? yaml : { emptyServiceAssets } })
const { configureTemplate, emptyTemplateOptions, parseIngressMetadata, validateIngressRoute } = module.exports
const base = () => ({ name: 'route', host: 'api.example.com', path: '/', service: 'api-svc', port: 80, class: 'alb', tls_secret: '' })
const definition = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.Service}}
spec:
  template:
    spec:
      containers:
        - name: app
          image: {{.Image}}
          ports:
            - containerPort: 9099
---
apiVersion: v1
kind: Service
metadata:
  name: {{.Service}}-svc
spec:
  ports:
    - port: {{.servicePort}}
      targetPort: 9099
`
test('ALB options keep capture references, cloud TLS and actual Service backend through JSON persistence', () => {
  const annotations = parseIngressMetadata('alb.ingress.kubernetes.io/use-regex: "true"\nalb.ingress.kubernetes.io/rewrite-target: /${2}\nalb.ingress.kubernetes.io/order: "1"\nalb.ingress.kubernetes.io/listen-ports: \'[{"HTTP":80},{"HTTPS":443}]\'')
  for (const pathType of ['Prefix', 'Exact', 'ImplementationSpecific']) {
    const options = { ...emptyTemplateOptions(), ingress: true, ingressClass: 'alb', host: 'api.example.com', pathType, tlsEnabled: true, tlsHosts: ['api.example.com'], ingressAnnotations: annotations, ingressLabels: { 'ingress-controller': 'alb' } }
    if (pathType === 'ImplementationSpecific') options.path = '/api(/|$)(.*)'
    const route = JSON.parse(JSON.stringify(configureTemplate(definition, options, 'api', {servicePort:'80'}).assets.routes[0]))
    assert.equal(route.service, 'api-svc'); assert.equal(route.port, 80)
    assert.equal(route.path_type, pathType); assert.equal(route.tls_enabled, true)
    assert.deepEqual(route.tls_hosts, ['api.example.com']); assert.equal(route.tls_secret, '')
    assert.equal(route.annotations['alb.ingress.kubernetes.io/rewrite-target'], '/${2}')
    assert.equal(route.labels['ingress-controller'], 'alb')
  }
  const plain = configureTemplate(definition, {...emptyTemplateOptions(), ingress:true, host:'api.example.com'}, 'api', {servicePort:'80'}).assets.routes[0]
  assert.equal(plain.tls_enabled, false); assert.equal(plain.tls_hosts.length, 0)
})
test('metadata edits reject duplicate/nested values, invalid listeners and TLS domain omissions', () => {
  for (const text of ['a: x\na: y', 'a: [x]', 'a:\n  nested: x', '- x']) assert.throws(() => parseIngressMetadata(text))
  for (const listener of ['[{"HTTP":"80"}]','[{"HTTP":80.5}]','[{"HTTP":80},{"HTTPS":80}]', '[]'])
    assert.throws(() => validateIngressRoute({...base(),annotations:{'alb.ingress.kubernetes.io/listen-ports':listener}}))
  assert.throws(() => validateIngressRoute({...base(),tls_enabled:true,tls_hosts:[]}))
  assert.throws(() => validateIngressRoute({...base(),labels:{'axiops.io/service-id':'999'}}))
  assert.throws(() => validateIngressRoute({...base(),annotations:{'':'invalid'}}))
})
test('native Ingress YAML remains unchanged and optional route cannot silently duplicate it', () => {
  const native = definition + '\n---\napiVersion: networking.k8s.io/v1\nkind: Ingress\nmetadata:\n  name: custom\n  annotations:\n    alb.ingress.kubernetes.io/rewrite-target: /${2}\nspec:\n  ingressClassName: alb\n'
  assert.equal(configureTemplate(native, emptyTemplateOptions(), 'api', {}).definition, native)
  assert.throws(() => configureTemplate(native, {...emptyTemplateOptions(), ingress:true,host:'api.example.com'}, 'api', {servicePort:'80'}))
})
test('TLS toggles and invalid metadata drafts never retain stale valid configuration', async () => {
  const filename = '../src/views/cicd/project-workspace/components/IngressRouteFields.vue'
  const { descriptor } = parse(readFileSync(new URL(filename, import.meta.url),'utf8'))
  assert.deepEqual(compileTemplate({source:descriptor.template.content,filename,id:'ingress'}).errors,[])
  const component = {exports:{}}
  vm.runInNewContext(ts.transpileModule(compileScript(descriptor,{id:'ingress'}).content,{compilerOptions:{module:ts.ModuleKind.CommonJS}}).outputText,
    {module:component,exports:component.exports,require:name=>name==='vue'?vue:name==='yaml'?yaml:module.exports})
  const props = vue.reactive({modelValue:base()});let state
  const renderer = vue.createRenderer({createComment:()=>({}),createText:()=>({}),createElement:()=>({}),insert(){},remove(){},setText(){},setElementText(){},patchProp(){},parentNode:()=>null,nextSibling:()=>null})
  const app = renderer.createApp({setup(){state=component.exports.default.setup(props,{expose(){},emit:(_,value)=>{props.modelValue=value}});return()=>null}})
  app.mount({})
  state.toggleTLS(true); await vue.nextTick()
  assert.deepEqual(Array.from(props.modelValue.tls_hosts),['api.example.com'])
  state.patch({tls_secret:'cert'});state.toggleTLS(false);await vue.nextTick()
  assert.equal(props.modelValue.tls_secret,'');assert.equal(props.modelValue.tls_hosts.length,0)
  state.editMetadata('annotations','alb.ingress.kubernetes.io/order: "1"');await vue.nextTick()
  state.editMetadata('annotations','invalid: [');await vue.nextTick()
  assert.ok(state.errors.annotations);assert.throws(()=>validateIngressRoute(props.modelValue))
  assert.equal(state.texts.annotations,'invalid: [')
  state.editMetadata('annotations','');await vue.nextTick()
  assert.equal(Object.keys(props.modelValue.annotations).length,0);validateIngressRoute(props.modelValue)
  app.unmount()
})
