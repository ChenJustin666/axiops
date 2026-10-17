import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { createRequire } from 'node:module'
import { test } from 'node:test'
import vm from 'node:vm'
import ts from 'typescript'
import { parseAllDocuments } from 'yaml'

const require = createRequire(import.meta.url)
const module = { exports: {} }
const emptyAssets = () => ({ command: [], args: [], workload_kind: '', workload_name: '', container: '', files: [], secrets: [], routes: [], volumes: [] })
vm.runInNewContext(ts.transpileModule(readFileSync(new URL('../src/utils/template-configuration.ts', import.meta.url), 'utf8'), {
  compilerOptions: { module: ts.ModuleKind.CommonJS }
}).outputText, { TextEncoder, module, exports: module.exports, require: name => name === 'yaml' ? require(name) : { emptyServiceAssets: emptyAssets } })
const { configureTemplate, emptyTemplateOptions, supportsTemplateOptions, templateParameterSection, templateYAMLBody, updateTemplateYAMLBody, templateHasProbes } = module.exports
const { configureTemplateSelections } = module.exports
const {parseIngressMetadata, validateIngressRoute} = module.exports
const definition = `# AxiOps-Variables: [{"key":"port","value":"9099"}]
# AxiOps-File: api-deployment.yaml
apiVersion: apps/v1
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
            - containerPort: {{.port}}
          resources:
            requests:
              cpu: 100m
---
# AxiOps-File: api-service.yaml
apiVersion: v1
kind: Service
metadata:
  name: {{.Service}}-http
spec:
  ports:
    - port: 80
      targetPort: {{.port}}
---
# AxiOps-File: disruption.yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: {{.Service}}-budget
spec:
  minAvailable: 1
`
const render = body => parseAllDocuments(body.replace(/\{\{\.Service\}\}/g, 'api').replace(/\{\{\.Image\}\}/g, 'registry/api:v1').replace(/\{\{\.port\}\}/g, '9099')).map(doc => {
  assert.equal(doc.errors.length, 0)
  return doc.toJSON()
})

test('multiple resource selections preserve IDs, paths and references across add edit and removal', () => {
  const selection=(id,key,values)=>({id,key,options:{...emptyTemplateOptions(),...values}})
  const items=[selection(1,'ingress',{host:'api.example.com'}),selection(2,'ingress',{host:'admin.example.com'}),selection(3,'secret',{secretName:'auth',secretKey:'SK',secretEnv:'API_SK'}),selection(4,'secret',{secretName:'auth',secretKey:'tls.key',secretUsage:'file',secretMount:'/secrets/tls.key'}),selection(5,'configmap',{filename:'app.yaml',content:'port: 9099',configMount:'/config/app.yaml'}),selection(6,'configmap',{configName:'shared',filename:'flags.json',configMount:'/config/flags.json'}),selection(7,'pvc',{mountPath:'/data'}),selection(8,'pvc',{mountPath:'/cache',existingClaim:'cache'})]
  const result=configureTemplateSelections(definition,items,'api',{})
  assert.equal(result.assets.routes.length,2);assert.equal(result.assets.secrets.length,2);assert.equal(result.assets.files.length,2);assert.equal(result.assets.volumes.length,2)
  assert.equal(result.assets.secrets[1].mount_path,'/secrets/tls.key');assert.equal(result.assets.secrets[1].env_name,'')
  assert.equal(result.assets.files[1].content,'');assert.equal(result.assets.files[1].existing_config_map,'shared')
  assert.equal(result.assets.volumes[1].capacity,'')
  const changed=configureTemplateSelections(definition,items.filter(item=>item.id!==1),'api',{})
  assert.equal(changed.assets.routes[0].name,result.assets.routes[1].name)
  assert.throws(()=>configureTemplateSelections(definition,[...items,selection(9,'pvc',{mountPath:'/secrets'})],'api',{}),/冲突/)
  assert.throws(()=>configureTemplateSelections(definition,[...items,selection(9,'secret',{secretName:'other',secretKey:'token',secretEnv:'API_SK'})],'api',{}),/重复/)
  assert.throws(()=>configureTemplateSelections(definition,[...items,selection(9,'ingress',{host:'api.example.com'})],'api',{}),/重复/)
})

test('unselected options preserve exact handwritten YAML', () => {
  const result = configureTemplate(definition, emptyTemplateOptions(), 'api', {})
  assert.equal(result.definition, definition)
  assert.equal(result.assets.routes.length, 0)
})

test('optional resources keep folded JVM flags and omit unselected Nacos credentials', () => {
  const template = definition.replace('          ports:', `          env:
            - name: JAVA_OPTS
              value: >-
                -XX:+UseContainerSupport
                -XX:MaxRAMPercentage={{.javaMaxRamPercentage}}
                -XX:MaxMetaspaceSize={{.javaMaxMetaspaceSize}}
                -XX:+UseG1GC
            - name: SPRING_PROFILES_ACTIVE
              value: test
          ports:`)
  const configured = configureTemplate(
    template,
    { ...emptyTemplateOptions(), ingress: true, host: 'api.example.com' },
    'api',
    { port: '9099', javaMaxRamPercentage: '72.0', javaMaxMetaspaceSize: '256m' }
  )
  assert.match(configured.definition, /name: JAVA_OPTS\n\s+value: >-/)
  assert.doesNotMatch(configured.definition, /SPRING_CLOUD_NACOS_(USERNAME|PASSWORD)/)
  const rendered = configured.definition
    .replaceAll('{{.Service}}', 'api')
    .replaceAll('{{.Image}}', 'registry/api:v1')
    .replaceAll('{{.port}}', '9099')
    .replaceAll('{{.javaMaxRamPercentage}}', '72.0')
    .replaceAll('{{.javaMaxMetaspaceSize}}', '256m')
  const env = parseAllDocuments(rendered)[0].toJSON().spec.template.spec.containers[0].env
  assert.equal(env[0].value, '-XX:+UseContainerSupport -XX:MaxRAMPercentage=72.0 -XX:MaxMetaspaceSize=256m -XX:+UseG1GC')
})

test('editable YAML hides internal parameter metadata but preserves it across source edits',()=>{
  const body=templateYAMLBody(definition)
  assert.equal(body.includes('AxiOps-Variables'),false)
  assert.match(body,/# AxiOps-File: api-deployment.yaml/)
  assert.match(body,/\{\{\.Image\}\}/)
  const updated=updateTemplateYAMLBody(definition,body+'\n# source edit\n')
  assert.match(updated,/AxiOps-Variables:.*9099/)
  assert.equal((updated.match(/AxiOps-Variables/g)||[]).length,1)
  assert.match(updated,/# source edit/)
  assert.equal(templateYAMLBody('  # AxiOps-Variables: []\r\nkind: Service'), 'kind: Service')
})

test('native probes are detected without treating missing or invalid templates as probes',()=>{
  assert.equal(templateHasProbes(definition),false)
  const configured=configureTemplate(definition,{...emptyTemplateOptions(),probe:true},'api',{port:'9099'})
  assert.equal(templateHasProbes(configured.definition),true)
  assert.equal(templateHasProbes('spec: ['),false)
})

test('optional capabilities preserve custom resources, filenames and system placeholders', () => {
  const options = { ...emptyTemplateOptions(), ingress: true, pvc: true, secret: true, configmap: true, hpa: true, probe: true, monitor: true,
    host: 'api.example.com', secretName: 'api-password', content: 'server:\n  port: 9099', probePath: '' }
  const result = configureTemplate(definition, options, 'api', { port: '9099' })
  assert.equal(result.assets.routes[0].service, 'api-http')
  assert.equal(result.assets.routes[0].port, 80)
  assert.equal(result.assets.volumes[0].mount_path, '/data')
  assert.equal(result.assets.secrets[0].existing_secret, 'api-password')
  assert.equal(result.assets.files[0].content, options.content)
  assert.match(result.definition, /# AxiOps-File: disruption.yaml/)
  assert.match(result.definition, /# AxiOps-File: api-service.yaml/)
  assert.match(result.definition, /\{\{\.Image\}\}/)
  const docs = render(result.definition)
  assert.equal(docs[2].kind, 'PodDisruptionBudget')
  assert.equal(docs[3].kind, 'HorizontalPodAutoscaler')
  assert.equal(docs[3].spec.scaleTargetRef.name, 'api')
  const pod = docs[0].spec.template
  assert.equal(pod.metadata.annotations['prometheus.io/port'], '9099')
  for (const probe of ['startupProbe', 'readinessProbe']) {
    assert.equal(pod.spec.containers[0][probe].tcpSocket.port, 9099)
    assert.equal(pod.spec.containers[0][probe].httpGet, undefined)
  }
  assert.equal(pod.spec.containers[0].livenessProbe, undefined, 'liveness must be deliberately enabled')
})

test('HTTP probes require path; duplicate resources and unsafe HPA settings fail before saving', () => {
  assert.throws(() => configureTemplate(definition, { ...emptyTemplateOptions(), probe: true, probeType: 'http', probePath: '' }, 'api', { port: '9099' }), /HTTP/)
  const http = configureTemplate(definition, { ...emptyTemplateOptions(), probe: true, probeType: 'http', probePath: '/ready' }, 'api', { port: '9099' })
  assert.equal(render(http.definition)[0].spec.template.spec.containers[0].readinessProbe.httpGet.path, '/ready')
  assert.throws(() => configureTemplate(http.definition, { ...emptyTemplateOptions(), probe: true }, 'api', { port: '9099' }), /已有探针/)
  assert.throws(() => configureTemplate(definition, { ...emptyTemplateOptions(), hpa: true, minReplicas: 4, maxReplicas: 2 }, 'api', { port: '9099' }), /HPA/)
  assert.throws(() => configureTemplate(definition.replace('cpu: 100m', 'memory: 1Gi'), { ...emptyTemplateOptions(), hpa: true }, 'api', { port: '9099' }), /CPU requests/)
  assert.equal(supportsTemplateOptions(definition.replace('        - name: app', '        - name: sidecar\n          image: busybox\n        - name: app')), false)
})

test('optional probe purposes are independent and TCP needs no HTTP path', () => {
  for (const selected of ['Readiness', 'Startup', 'Liveness']) {
    const options = { ...emptyTemplateOptions(), probe: true, probePath: '', probeReadinessEnabled: false, probeStartupEnabled: false, probeLivenessEnabled: false, [`probe${selected}Enabled`]: true }
    const configured = configureTemplate(definition, options, 'api', { port: '9099' })
    const container = render(configured.definition)[0].spec.template.spec.containers[0]
    for (const purpose of ['readiness', 'startup', 'liveness']) assert.equal(!!container[`${purpose}Probe`], purpose === selected.toLowerCase())
  }
  assert.throws(() => configureTemplate(definition, { ...emptyTemplateOptions(), probe: true, probeReadinessEnabled: false, probeStartupEnabled: false, probeLivenessEnabled: false }, 'api', {port:'9099'}), /至少选择一种/)
})

test('legacy probe parameters stay out of Env and JVM even when grouped as advanced', () => {
  for(const key of ['healthCheckType','healthPath','healthPort','probePeriodSeconds','probeTimeoutSeconds','probeFailureThreshold','startupFailureThreshold','metricsPath'])
    assert.equal(templateParameterSection({key,group:'advanced'}),'health')
  for(const key of ['javaMaxRamPercentage','javaMaxMetaspaceSize'])
    assert.equal(templateParameterSection({key,group:'advanced'}),'env')
  assert.equal(templateParameterSection({key:'JAVA_OPTS',group:'env'}),'env')
  assert.equal(templateParameterSection({key:'extra',group:'advanced'}),'advanced')
  assert.equal(templateParameterSection({key:'replicas'}),'primary')
})

test('unfinished paths, storage and conflicting legacy probes cannot complete a capability', () => {
  assert.throws(()=>configureTemplate(definition,{...emptyTemplateOptions(),ingress:true,host:'api.example.com',path:''},'api',{port:'9099'}),/路径/)
  assert.throws(()=>configureTemplate(definition,{...emptyTemplateOptions(),pvc:true,capacity:''},'api',{port:'9099'}),/容量/)
  assert.throws(()=>configureTemplate(definition,{...emptyTemplateOptions(),probe:true},'api',{port:'9099',healthCheckType:'http'}),/模板健康检查已启用/)
})

test('existing PVC references omit creation-only fields even when default capacity was retained',()=>{
  const result=configureTemplate(definition,{...emptyTemplateOptions(),pvc:true,existingClaim:'shared-data',capacity:'1Gi',storageClass:'standard'},'api',{port:'9099'})
  assert.equal(result.assets.volumes[0].existing_claim,'shared-data')
  assert.equal(result.assets.volumes[0].capacity,'')
  assert.equal(result.assets.volumes[0].storage_class,'')
})

test('resource fields reject malformed names, paths, units and oversized UTF-8 files', () => {
  const invalid = [
    { ingress: true, host: 'https://api.example.com' },
    { ingress: true, host: '10.0.0.1' },
    { ingress: true, host: 'api.example.com', path: '/a/../b' },
    { ingress: true, host: 'api.example.com', ingressClass: 'Bad_Class' },
    { ingress: true, host: 'api.example.com', tlsSecret: 'tls/key' },
    { pvc: true, capacity: '2' }, { pvc: true, capacity: '2gi' },
    { pvc: true, capacity: '0Gi' }, { pvc: true, capacity: '1Ki' },
    { pvc: true, storageClass: 'bad_class' },
    { pvc: true, existingClaim: 'BAD', mountPath: '/data' },
    { pvc: true, mountPath: '/' },
    { secret: true, secretName: 'Bad' },
    { secret: true, secretName: 'auth', secretKey: '../key' },
    { secret: true, secretName: 'auth', secretEnv: 'BAD-ENV' },
    { secret: true, secretName: 'auth', secretUsage: 'file', secretMount: '/etc/../key' },
    { configmap: true, configName: 'Bad' },
    { configmap: true, filename: '../file' },
    { configmap: true, content: '中'.repeat(90000) },
    { probe: true, probePort: 'bad_port' },
    { monitor: true, metricsPath: 'actuator/prometheus' }
  ]
  for (const options of invalid) assert.throws(() => configureTemplate(definition, { ...emptyTemplateOptions(), ...options }, 'api', { port: '9099' }), JSON.stringify(options).slice(0,150))
  for (const capacity of ['1Mi', '2Gi', '1.5Gi', '1G', '1e9']) {
    assert.equal(configureTemplate(definition, { ...emptyTemplateOptions(), pvc: true, capacity }, 'api', {}).assets.volumes[0].capacity, capacity)
  }
  const content = '  value: unchanged\n\n'
  const config = configureTemplate(definition, { ...emptyTemplateOptions(), configmap: true, content }, 'api', {})
  assert.equal(config.assets.files[0].content, content)
  assert.equal(configureTemplate(definition, { ...emptyTemplateOptions(), ingress: true, host: '*.example.com' }, 'api', {}).assets.routes[0].host, '*.example.com')
})
