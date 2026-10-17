import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import vm from 'node:vm'
import ts from 'typescript'

const source = readFileSync(new URL('../src/api/axiops/cicd.ts', import.meta.url), 'utf8')
const code = ts.transpileModule(source, {
  compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2020 }
}).outputText
const body = () => ({ targets: [{ environment_id: 16, token: 'a'.repeat(64) }], revision_id: 12, variables: { port: '8080', replicas: null } })
const response = (preview) => ({ saved: !preview, validated: true, configs: [{
  environment_id: 16, service_id: 17, bound: true, revision_id: 12,
  definition: 'kind: Deployment', variables: [], token: 'b'.repeat(64)
}] })
function load(result) {
  const calls = [], module = { exports: {} }
  const http = Object.fromEntries(['get', 'post', 'put', 'del'].map(method => [method, async input => {
    calls.push({ method, ...input })
    return result
  }]))
  vm.runInNewContext(code, { module, exports: module.exports, require(name) {
    if (name === '@/utils/http') return { default: http }
    if (name === '@/utils/service-name') return {}
    throw Error(`Unexpected import ${name}`)
  } })
  return { api: module.exports, calls }
}

test('environment API preserves preview/save semantics and inheritance deletion on the wire', async () => {
  for (const preview of [true, false]) {
    const result = response(preview), view = load(result), input = body()
    assert.equal(await view.api.saveServiceEnvironmentConfigApi(17, input, preview), result)
    assert.equal(view.calls[0].method, 'post')
    assert.equal(view.calls[0].url, `/cicd/v1/services/17/environment-config${preview ? '?preview=1' : ''}`)
    assert.equal(view.calls[0].data, input)
    assert.equal(view.calls[0].data.variables.replicas, null)
  }
})

test('incomplete or wrong-target preview cannot enable saving', async () => {
  const cases = [undefined, {}, { ...response(true), validated: false },
    { ...response(true), saved: true }, { ...response(true), configs: null },
    { ...response(true), configs: [] }]
  for (const patch of [{ service_id: 18 }, { environment_id: 20 }, { bound: false },
    { token: '' }, { variables: undefined }, { revision_id: 0 }]) {
    const result = response(true)
    Object.assign(result.configs[0], patch)
    cases.push(result)
  }
  for (const result of cases) {
    await assert.rejects(load(result).api.saveServiceEnvironmentConfigApi(17, body(), true), /无法确认差异/)
  }
})

test('duplicate target results cannot hide a missing environment in preview', async () => {
  const result = response(true)
  result.configs.push({ ...result.configs[0] })
  const input = body()
  input.targets.push({ environment_id: 20, token: 'c'.repeat(64) })
  await assert.rejects(load(result).api.saveServiceEnvironmentConfigApi(17, input, true), /无法确认差异/)
})

test('an unconfirmed save asks for reload rather than reporting success or automatically retrying', async () => {
  const view = load(response(true))
  await assert.rejects(view.api.saveServiceEnvironmentConfigApi(17, body()), /重新加载核对目标环境/)
  assert.equal(view.calls.length, 1)
})

test('environment resource create and update retain target and optimistic revision', async () => {
  const view = load({})
  const resource = { name: 'settings', kind: 'ConfigMap', manifest: 'kind: ConfigMap' }
  await view.api.saveEnvironmentResourceApi(16, resource)
  await view.api.saveEnvironmentResourceApi(20, { ...resource, id: 5, expected_revision: 3 })
  assert.equal(view.calls[0].method, 'post')
  assert.equal(view.calls[0].url, '/cicd/v1/environments/16/resources')
  assert.equal(view.calls[1].method, 'put')
  assert.equal(view.calls[1].url, '/cicd/v1/environments/20/resources')
  assert.equal(view.calls[1].data.expected_revision, 3)
})
