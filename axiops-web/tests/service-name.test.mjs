import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import vm from 'node:vm'
import ts from 'typescript'
import { parse, compileScript, compileTemplate } from '@vue/compiler-sfc'

function loadTS(path, require = () => ({})) {
  const code = ts.transpileModule(readFileSync(new URL(path, import.meta.url), 'utf8'), {
    compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 }
  }).outputText
  const module = { exports: {} }
  vm.runInNewContext(code, { module, exports: module.exports, require })
  return module.exports
}

const naming = loadTS('../src/utils/service-name.ts')

test('service creation rejects names that cannot become default image names before sending requests', async () => {
  const calls = []
  const api = loadTS('../src/api/axiops/cicd.ts', (name) =>
    name === '@/utils/service-name'
      ? naming
      : {
          default: {
            post: async (request) => {
              calls.push(request)
              return { id: 1 }
            }
          }
        }
  )
  for (const name of [
    'Risk_Server',
    '服务',
    'api/name',
    'api:latest',
    '-api',
    'api.',
    'a'.repeat(129)
  ]) {
    await assert.rejects(api.createServiceApi(19, { name, code: 'test' }), /risk_server/)
  }
  assert.equal(calls.length, 0)
  for (const name of ['risk_server', 'api.v2', 'api--v2', 'api__v2', '1worker']) {
    assert.equal(naming.serviceNameError(name), '')
    const code = naming.serviceCodeFromName(name)
    assert.match(code, /^[a-z][a-z0-9-]{0,63}$/)
    await api.createServiceApi(19, { name, code })
  }
  assert.equal(calls.length, 5)
  assert.equal(calls[0].data.name, 'risk_server')
  assert.equal(calls[0].data.code, 'risk-server')
})

test('all service creation surfaces and the namespace form compile', () => {
  for (const path of [
    '../src/views/cicd/project-workspace/index.vue',
    '../src/views/cicd/project-wizard/index.vue',
    '../src/views/cicd/project-workspace/components/ServiceCreationWizard.vue',
    '../src/views/cicd/project-workspace/components/BulkServiceImportDialog.vue',
    '../src/views/cicd/resource-center/BuildRegistries.vue'
  ]) {
    const { descriptor, errors } = parse(readFileSync(new URL(path, import.meta.url), 'utf8'), {
      filename: path,
      templateParseOptions: { expressionPlugins: ['typescript'] }
    })
    assert.equal(
      errors.length,
      0,
      `${path}: ${errors.map((error) => `${error.message} ${JSON.stringify(error.loc)}`).join('; ')}`
    )
    const script = compileScript(descriptor, { id: path })
    const result = compileTemplate({
      source: descriptor.template.content,
      filename: path,
      id: path,
      compilerOptions: { bindingMetadata: script.bindings, expressionPlugins: ['typescript'] }
    })
    assert.equal(
      result.errors.length,
      0,
      `${path}: ${result.errors.map((error) => error.message).join('; ')}`
    )
  }
})
