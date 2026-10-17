import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import vm from 'node:vm'
import { parse, compileScript, compileTemplate } from '@vue/compiler-sfc'
import ts from 'typescript'
import * as vue from 'vue'

const filename = new URL('../src/views/cicd/build-settings/index.vue', import.meta.url)
const { descriptor } = parse(readFileSync(filename, 'utf8'))
assert.deepEqual(
  compileTemplate({
    source: descriptor.template.content,
    id: 'build-settings',
    filename: filename.pathname
  }).errors,
  []
)
const code = ts.transpileModule(compileScript(descriptor, { id: 'build-settings' }).content, {
  compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2020 }
}).outputText

function setup(overrides = {}) {
  const calls = [],
    messages = []
  const module = { exports: {} }
  vm.runInNewContext(code, {
    module,
    exports: module.exports,
    require(name) {
      if (name === 'vue') return { ...vue, onMounted() {} }
      if (name === 'vue-router') return { useRouter: () => ({ push() {} }) }
      if (name === 'element-plus')
        return {
          ElMessage: Object.fromEntries(
            ['warning', 'success', 'error'].map((type) => [
              type,
              (message) => messages.push({ type, message })
            ])
          )
        }
      if (name === '@/api/axiops')
        return {
          getPlatformBuildSettingsApi: async () => ({
            enabled: true,
            cluster_id: 2,
            namespace: 'ci',
            builder_image: 'builder:v1',
            build_command: 'build',
            resource_cpu: '',
            resource_memory: '',
            execution_policy: {
              node_selector: { 'axiops.io/ci-pool': 'ci' },
              max_builds_per_node: 2,
              log_retention_days: 30
            }
          }),
          getConcurrencyPolicyApi: async () => ({ max_running: 2, queue_timeout_seconds: 3600 }),
          listClustersApi: async () => [{ id: 2, name: 'CI', status: 'active' }],
          savePlatformBuildSettingsApi: async (body) => calls.push(['settings', body]),
          saveConcurrencyPolicyApi: async (body) => calls.push(['concurrency', body]),
          ...overrides
        }
      return {}
    }
  })
  return { state: module.exports.default.setup({}, { expose() {} }), calls, messages }
}

test('node policy and retention round-trip while optional CPU and memory remain empty', async () => {
  const { state, calls } = setup()
  await state.load()
  assert.equal(calls.length, 0)
  assert.equal(state.maxBuildsPerNode.value, 2)
  await state.save()
  assert.equal(calls.length, 2)
  const body = calls[0][1]
  assert.equal(body.resource_cpu, '')
  assert.equal(body.resource_memory, '')
  assert.equal(body.execution_policy.node_selector['axiops.io/ci-pool'], 'ci')
  assert.equal(body.execution_policy.log_retention_days, 30)
  assert.equal('updated_at' in body, false)
})

test('failed reads block writes until explicit retry succeeds', async () => {
  let unavailable = true
  const { state, calls } = setup({
    getConcurrencyPolicyApi: async () => {
      if (unavailable) throw new Error('offline')
      return { max_running: 2 }
    }
  })
  await state.load()
  assert.ok(state.loadError.value)
  await state.save()
  assert.equal(calls.length, 0)
  unavailable = false
  await state.load()
  await state.save()
  assert.equal(calls.length, 2)
})

test('duplicate node labels cannot silently overwrite a selection', async () => {
  const { state, calls, messages } = setup()
  await state.load()
  state.nodeSelectorText.value = 'pool=ci\npool=production'
  await state.save()
  assert.equal(calls.length, 0)
  assert.equal(messages.at(-1).type, 'warning')
})

test('partial settings save reports concurrency failure without a success toast', async () => {
  const { state, calls, messages } = setup({
    saveConcurrencyPolicyApi: async () => {
      throw new Error('offline')
    }
  })
  await state.load()
  await state.save()
  assert.equal(calls.length, 1)
  assert.equal(messages.at(-1).type, 'error')
  assert.equal(
    messages.some((item) => item.type === 'success'),
    false
  )
})
