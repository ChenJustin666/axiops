import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import vm from 'node:vm'
import ts from 'typescript'
import { parse, compileScript, compileTemplate } from '@vue/compiler-sfc'
import * as vue from 'vue'

const transpile = (source) =>
  ts.transpileModule(source, {
    compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 }
  }).outputText
function utility(path) {
  const module = { exports: {} }
  vm.runInNewContext(transpile(readFileSync(new URL(path, import.meta.url), 'utf8')), {
    module,
    exports: module.exports,
    require: name => {
      if (name === './workflow-service-order') return utility('../src/utils/workflow-service-order.ts')
      throw new Error(`Unexpected utility dependency: ${name}`)
    }
  })
  return module.exports
}
const paths = utility('../src/utils/build-source.ts')
const workspace = utility('../src/utils/cicd-workspace.ts')
const filename = new URL(
  '../src/views/cicd/project-workspace/components/BuildSourcePicker.vue',
  import.meta.url
)
const { descriptor } = parse(readFileSync(filename, 'utf8'))
assert.deepEqual(
  compileTemplate({
    source: descriptor.template.content,
    id: 'source-picker',
    filename: filename.pathname
  }).errors,
  []
)
const code = transpile(compileScript(descriptor, { id: 'source-picker' }).content)
const file = (path) => ({ name: path.split('/').at(-1), path, type: 'file', size: 100 })
const directory = (path) => ({ name: path, path, type: 'dir', size: 0 })
function setup(api = async () => [], overrides = {}) {
  const module = { exports: {} },
    events = []
  vm.runInNewContext(code, {
    module,
    exports: module.exports,
    require: (name) => {
      if (name === 'vue') return vue
      if (name === '@/utils/build-source') return paths
      if (name === '@/utils/cicd-workspace') return workspace
      if (name === '@/api/axiops') return { listRepositoryFilesApi: api }
      return {}
    }
  })
  const props = vue.reactive({
    codeSourceId: 5,
    repoPath: 'server/CardScout',
    branch: 'test',
    dockerfilePath: 'Dockerfile',
    buildContext: '.',
    disabled: false,
    active: true,
    offerMavenModule: true,
    jarImage: false,
    ...overrides
  })
  const scope = vue.effectScope()
  const state = scope.run(() =>
    module.exports.default.setup(props, {
      expose() {},
      emit: (name, value) => {
        events.push([name, value])
        if (name.startsWith('update:')) props[name.slice(7)] = value
      }
    })
  )
  return { props, state, events, stop: () => scope.stop() }
}

test('single module root Dockerfile selects directly without changing the build context', async () => {
  const { state, props, stop } = setup(async () => [
    file('Dockerfile'),
    file('pom.xml'),
    file('Jenkinsfile')
  ])
  await state.loadDirectory('.')
  assert.deepEqual(
    Array.from(state.visibleEntries.value, (entry) => entry.path),
    ['Dockerfile']
  )
  state.selectFile(state.visibleEntries.value[0])
  assert.equal(props.dockerfilePath, 'Dockerfile')
  assert.equal(props.buildContext, '.')
  assert.equal(state.mavenModule.value, '')
  stop()
})

test('reactor module selection keeps parent and siblings; switching context preserves the selected file', async () => {
  const requests = []
  const { state, props, stop } = setup(async (id, params) => {
    requests.push(params)
    return params.path
      ? [file('card-scout-gateway/Dockerfile'), file('card-scout-gateway/pom.xml')]
      : [directory('card-scout-gateway'), directory('card-scout-common'), file('pom.xml')]
  })
  await state.loadDirectory('.')
  await state.loadDirectory('card-scout-gateway')
  state.selectFile(state.visibleEntries.value[0])
  assert.equal(props.dockerfilePath, 'card-scout-gateway/Dockerfile')
  assert.equal(props.buildContext, '.')
  assert.equal(state.mavenModule.value, 'card-scout-gateway')
  assert.equal(requests[1].ref, 'test')
  state.changeContext('module')
  assert.equal(props.buildContext, 'card-scout-gateway')
  assert.equal(props.dockerfilePath, 'Dockerfile')
  assert.equal(state.selectedFile.value, 'card-scout-gateway/Dockerfile')
  state.changeContext('root')
  assert.equal(props.dockerfilePath, 'card-scout-gateway/Dockerfile')
  assert.equal(props.buildContext, '.')
  props.dockerfilePath = 'deploy/Dockerfile'
  assert.equal(
    state.mavenModule.value,
    '',
    'manual edits must not reuse POM detection for another folder'
  )
  stop()
})

test('existing custom context survives loading and browsing', async () => {
  const { state, props, events, stop } = setup(async () => [], {
    buildContext: 'services/api',
    dockerfilePath: 'docker/Dockerfile'
  })
  assert.equal(state.selectedFile.value, 'services/api/docker/Dockerfile')
  assert.equal(state.contextMode.value, 'custom')
  await state.loadDirectory('.')
  assert.equal(events.length, 0)
  assert.equal(props.buildContext, 'services/api')
  stop()
})

test('branch changes discard in-flight directory responses; failures remain visible and retryable', async () => {
  let resolveOld,
    calls = 0
  const { state, props, stop } = setup(async () => {
    calls++
    if (calls === 1)
      return new Promise((resolve) => {
        resolveOld = resolve
      })
    if (calls === 2) throw Error('network unavailable')
    return [file('Dockerfile')]
  })
  const old = state.loadDirectory('.')
  props.branch = 'main'
  await vue.nextTick()
  resolveOld([file('old/Dockerfile')])
  await old
  assert.equal(state.entries.value.length, 0)
  await state.loadDirectory('.')
  assert.match(state.browseError.value, /network unavailable/)
  await state.loadDirectory('.')
  assert.equal(state.browseError.value, '')
  assert.equal(state.entries.value[0].path, 'Dockerfile')
  stop()
})

test('path validation and Maven reactor commands keep paths inside the checkout and quote shell arguments', () => {
  assert.equal(
    paths.dockerfileInRepository('.', 'card-scout-server/Dockerfile'),
    'card-scout-server/Dockerfile'
  )
  for (const path of ['../secret', '/etc', 'C:\\workspace', 'a/../../b'])
    assert.ok(paths.buildSourceError(path, 'Dockerfile'))
  assert.ok(paths.buildSourceError('.', '..'))
  assert.match(
    paths.mavenModuleCommand('card-scout-gateway'),
    /-pl 'card-scout-gateway' -am clean package$/
  )
  assert.doesNotMatch(paths.mavenModuleCommand('card-scout-server'), /skipTests/)
  assert.match(paths.mavenModuleCommand("module's name"), /'module'\\''s name'/)
  assert.equal(paths.mavenModuleCommand('-x'), '')
})
