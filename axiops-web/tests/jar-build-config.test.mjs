import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import vm from 'node:vm'
import { parse, compileScript, compileTemplate } from '@vue/compiler-sfc'
import ts from 'typescript'
import * as vue from 'vue'

const filename = new URL(
  '../src/views/cicd/project-workspace/components/BuildConfigDialog.vue',
  import.meta.url
)
const { descriptor } = parse(readFileSync(filename, 'utf8'))
const script = compileScript(descriptor, { id: 'jar-build-config-test' }).content
assert.equal(
  compileTemplate({
    source: descriptor.template.content,
    id: 'jar-build-config-test',
    filename: filename.pathname
  }).errors.length,
  0
)
const code = ts.transpileModule(script, {
  compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 }
}).outputText

const utilityModule = { exports: {} }
const buildSourceModule = { exports: {} }
const serviceOrderModule = { exports: {} }
vm.runInNewContext(
  ts.transpileModule(
    readFileSync(new URL('../src/utils/workflow-service-order.ts', import.meta.url), 'utf8'),
    { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 } }
  ).outputText,
  { module: serviceOrderModule, exports: serviceOrderModule.exports }
)
vm.runInNewContext(
  ts.transpileModule(
    readFileSync(new URL('../src/utils/build-source.ts', import.meta.url), 'utf8'),
    { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 } }
  ).outputText,
  { module: buildSourceModule, exports: buildSourceModule.exports }
)
vm.runInNewContext(
  ts.transpileModule(
    readFileSync(new URL('../src/utils/cicd-workspace.ts', import.meta.url), 'utf8'),
    {
      compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 }
    }
  ).outputText,
  { module: utilityModule, exports: utilityModule.exports, require: name => {
    if (name === './workflow-service-order') return serviceOrderModule.exports
    throw new Error(`Unexpected utility dependency: ${name}`)
  } }
)

function setup({ service = {}, registries = [], config = {}, api = {} } = {}) {
  const module = { exports: {} }
  const saves = [],
    warnings = [],
    updates = []
  vm.runInNewContext(code, {
    Blob,
    module,
    exports: module.exports,
    require: (name) => {
      if (name === 'vue') return vue
      if (name === '@/utils/cicd-workspace') return utilityModule.exports
      if (name === '@/utils/build-source') return buildSourceModule.exports
      if (name === 'element-plus')
        return { ElMessage: { warning: (value) => warnings.push(value), success() {} } }
      if (name === '@/api/axiops')
        return {
          listCodeSourcesApi: async () => [],
          getBuildConfigApi: async () => ({
            build_mode: 'dockerfile',
            image_repo: '',
            cache_dir: '',
            ...config
          }),
          updateServiceApi: async (id, body) => updates.push({ id, body }),
          saveBuildConfigApi: async (id, body) => saves.push({ id, body }),
          listRepositoriesApi: async () => [],
          listBranchesApi: async () => [],
          ...api
        }
      if (name === '@/api/axiops/build-registry')
        return { listBuildRegistriesApi: async () => registries }
      return {}
    }
  })
  const scope = vue.effectScope()
  const props = vue.reactive({
    visible: false,
    editable: true,
    service: { id: 17, language: 'java', code: 'ad-gateway', ...service }
  })
  const state = scope.run(() => module.exports.default.setup(props, { expose() {}, emit() {} }))
  return { state, props, saves, warnings, updates, stop: () => scope.stop() }
}

test('default registry and namespace generate the service image, switching namespace saves the same target', async () => {
  const { state, saves, updates, warnings, stop } = setup({
    service: { code: 'risk-server', name: 'risk_server' },
    registries: [
      {
        id: 1,
        name: '阿里云-美国',
        url: 'https://registry.example/sinozo',
        namespaces: ['abc', 'dbc'],
        status: 'active'
      }
    ]
  })
  await state.load()
  assert.equal(state.form.registry_id, 1)
  assert.equal(state.imageNamespace.value, 'sinozo')
  assert.equal(state.form.image_repo, 'registry.example/sinozo/risk_server')
  state.imageNamespace.value = 'abc'
  state.updateGeneratedImage()
  assert.equal(state.form.image_repo, 'registry.example/abc/risk_server')
  // Optional overrides must not make ordinary users re-enter the service name.
  assert.equal(state.imageAdvancedOpen.value.length, 0)
  assert.equal(state.imageNameOverride.value, '')
  state.imageNameOverride.value = 'risk-alias'
  assert.equal(state.form.image_repo, 'registry.example/abc/risk-alias')
  state.imageNameOverride.value = ''
  assert.equal(state.form.image_repo, 'registry.example/abc/risk_server')
  Object.assign(state.form, {
    code_source_id: 2,
    repo_path: 'server/risk_server',
    repo_branch: 'main'
  })
  state.repositories.value = [
    { full_name: 'server/risk_server', clone_url: 'https://git.example/server/risk_server.git' }
  ]
  await state.save()
  assert.equal(warnings.length, 0)
  assert.equal(saves[0].body.image_repo, 'registry.example/abc/risk_server')
  assert.equal(updates.length, 0, 'service and build config must be saved in one request')
  assert.equal(saves[0].body.service_binding.registry_id, 1)
  assert.equal(saves[0].body.service_binding.repo_url, 'https://git.example/server/risk_server.git')
  stop()
})

test('saved alternative registry, namespace and custom image name survive reopening', async () => {
  const { state, stop } = setup({
    service: { registry_id: 2 },
    registries: [
      { id: 1, name: '阿里云-美国', url: 'https://registry.example/sinozo', status: 'active' },
      {
        id: 2,
        name: 'Harbor',
        url: 'https://harbor.example',
        namespaces: ['abc', 'dbc'],
        status: 'active'
      }
    ],
    config: { image_repo: 'harbor.example/dbc/custom_name' }
  })
  await state.load()
  assert.equal(state.form.registry_id, 2)
  assert.equal(state.imageNamespace.value, 'dbc')
  assert.equal(state.imageName.value, 'custom_name')
  assert.equal(state.form.image_repo, 'harbor.example/dbc/custom_name')
  state.imageNamespace.value = 'abc'
  state.updateGeneratedImage()
  assert.equal(state.form.image_repo, 'harbor.example/abc/custom_name')
  state.customImagePath.value = true
  state.form.image_repo = 'harbor.example/dbc/another_name'
  state.restoreServiceImageName()
  assert.equal(state.imageNamespace.value, 'dbc')
  assert.equal(state.form.image_repo, 'harbor.example/dbc/ad-gateway')
  stop()
})

test('custom paths are preserved for review and cannot bypass configured namespaces', async () => {
  const { state, saves, warnings, stop } = setup({
    service: { registry_id: 1 },
    registries: [
      {
        id: 1,
        name: 'Harbor',
        url: 'https://harbor.example/sinozo',
        namespaces: ['abc'],
        status: 'active'
      }
    ],
    config: { image_repo: 'harbor.example/private/custom' }
  })
  await state.load()
  assert.equal(state.form.image_repo, 'harbor.example/private/custom')
  assert.equal(state.customImagePath.value, true)
  await state.save()
  assert.equal(saves.length, 0)
  assert.match(warnings[0], /命名空间/)
  state.toggleImagePathMode()
  assert.equal(state.form.image_repo, 'harbor.example/sinozo/ad-gateway')
  stop()
})

test('selecting JAR fills Maven/JDK17 and preserves edits across mode switches', () => {
  const { state, stop } = setup()
  state.form.build_mode = 'jar_image'
  state.onBuildModeChange('jar_image')
  assert.equal(state.form.template_type, 'maven')
  assert.equal(state.form.build_image, 'maven:3.9-eclipse-temurin-17')
  assert.match(state.form.build_command, /clean package/)
  assert.doesNotMatch(state.form.build_command, /skipTests|docker build/)
  state.form.build_command = 'mvn -pl ad-gateway -am clean package'
  state.form.build_mode = 'dockerfile'
  state.onBuildModeChange('dockerfile')
  state.form.build_mode = 'jar_image'
  state.onBuildModeChange('jar_image')
  assert.equal(state.form.build_command, 'mvn -pl ad-gateway -am clean package')
  stop()
})

test('Maven reactor command requires an explicit selection and is never injected into Dockerfile mode', () => {
  const { state, stop } = setup()
  state.form.build_mode = 'jar_image'
  state.onBuildModeChange('jar_image')
  state.form.build_command = 'sh ci/custom-package.sh'
  state.form.dockerfile_path = 'card-scout-gateway/Dockerfile'
  assert.equal(state.form.build_command, 'sh ci/custom-package.sh')
  state.applyMavenModule('card-scout-gateway')
  assert.match(state.form.build_command, /-pl 'card-scout-gateway' -am clean package$/)
  state.form.build_mode = 'dockerfile'
  state.form.build_command = ''
  state.applyMavenModule('card-scout-server')
  assert.equal(state.form.build_command, '')
  stop()
})

test('Gradle template changes compilation image/command and saved custom commands survive loading', () => {
  const { state, stop } = setup()
  state.form.build_mode = 'jar_image'
  state.onBuildModeChange('jar_image')
  state.applyTemplate('gradle')
  assert.equal(state.form.build_image, 'gradle:8-jdk17')
  assert.match(state.form.build_command, /gradle.*clean build/)
  state.applyConfig({
    build_mode: 'jar_image',
    template_type: 'custom',
    build_image: 'private/java:17',
    build_command: 'sh ci/package.sh'
  })
  assert.equal(state.form.build_command, 'sh ci/package.sh')
  assert.equal(state.form.build_image, 'private/java:17')
  stop()
})

test('JAR selection saves compile configuration and requires source binding', async () => {
  const { state, saves, warnings, stop } = setup()
  state.form.build_mode = 'jar_image'
  state.onBuildModeChange('jar_image')
  state.form.registry_id = 1
  state.form.image_repo = 'registry.example/team/ad-gateway'
  state.imageRegistries.value = [{ id: 1, url: 'registry.example/team' }]
  await state.save()
  assert.equal(saves.length, 0)
  assert.match(warnings[0], /代码源/)
  state.form.code_source_id = 2
  state.form.repo_path = 'team/ad-gateway'
  state.form.repo_branch = 'master'
  state.repositories.value = [
    { full_name: 'team/ad-gateway', clone_url: 'https://git.example/team/ad-gateway.git' }
  ]
  await state.save()
  assert.equal(saves.length, 1)
  assert.equal(saves[0].body.build_mode, 'jar_image')
  assert.equal(saves[0].body.build_image, 'maven:3.9-eclipse-temurin-17')
  assert.match(saves[0].body.build_command, /clean package/)
  assert.equal(saves[0].body.dockerfile_path, 'Dockerfile')
  stop()
})

test('only an explicit missing-config error permits defaults; failures block saving and retry recovers', async () => {
  let failure = { code: 400, businessCode: 30001 }
  const { state, saves, stop } = setup({
    api: {
      getBuildConfigApi: async () => {
        throw failure
      }
    }
  })
  await state.load()
  assert.equal(state.loadError.value, '')
  for (const error of [{ code: 403 }, { code: 500 }, { code: 404 }, new Error('offline')]) {
    failure = error
    await state.load()
    assert.match(state.loadError.value, /读取失败/)
    await state.save()
  }
  assert.equal(saves.length, 0)
  failure = { businessCode: 30001 }
  await state.load()
  assert.equal(state.loadError.value, '')
  stop()
})

test('late repository and branch responses cannot replace a newer source selection', async () => {
  const repos = {},
    branches = {}
  const { state, stop } = setup({
    api: {
      listRepositoriesApi: (source) =>
        new Promise((resolve) => {
          repos[source] = resolve
        }),
      listBranchesApi: (source) =>
        new Promise((resolve) => {
          branches[source] = resolve
        })
    }
  })
  state.form.code_source_id = 1
  state.form.repo_path = 'old/repo'
  const oldRepo = state.loadRepositories(),
    oldBranch = state.loadBranches()
  state.form.code_source_id = 2
  state.form.repo_path = 'new/repo'
  const newRepo = state.loadRepositories(),
    newBranch = state.loadBranches()
  repos[2]([{ full_name: 'new/repo' }])
  branches[2]([{ name: 'new' }])
  await Promise.all([newRepo, newBranch])
  repos[1]([{ full_name: 'old/repo' }])
  branches[1]([{ name: 'old' }])
  await Promise.all([oldRepo, oldBranch])
  assert.equal(state.repositories.value[0].full_name, 'new/repo')
  assert.equal(state.branches.value[0].name, 'new')
  stop()
})

test('late config load from another service cannot overwrite the active dialog', async () => {
  const pending = {}
  const { state, props, stop } = setup({
    api: {
      getBuildConfigApi: (id) =>
        new Promise((resolve) => {
          pending[id] = resolve
        })
    }
  })
  const old = state.load()
  props.service = { ...props.service, id: 18 }
  await vue.nextTick()
  const current = state.load()
  pending[18]({ build_mode: 'artifact', build_command: 'current' })
  await current
  pending[17]({ build_mode: 'artifact', build_command: 'old' })
  await old
  assert.equal(state.form.build_command, 'current')
  stop()
})

test('repository without Dockerfile saves platform template text and reopens unchanged', async () => {
  const { state, saves, warnings, stop } = setup()
  state.applyJarDockerfile()
  assert.equal(state.form.build_mode, 'jar_image')
  assert.equal(state.form.dockerfile_source, 'inline')
  assert.match(state.form.build_command, /mvn.*clean package/)
  state.applyLibraryDockerfile({ files: [{ path: 'Dockerfile', content: 'FROM eclipse-temurin:17-jre\nCOPY target/app.jar app.jar\nENTRYPOINT ["java","-jar","app.jar"]' }] })
  Object.assign(state.form, { registry_id: 1, image_repo: 'registry.example/team/ad-gateway', code_source_id: 2, repo_path: 'team/ad-gateway', repo_branch: 'main' })
  state.imageRegistries.value = [{ id: 1, url: 'registry.example/team' }]
  state.repositories.value = [{ full_name: 'team/ad-gateway', clone_url: 'https://git.example/team/ad-gateway.git' }]
  state.form.build_context = './'
  await state.save()
  assert.equal(warnings.length, 0)
  assert.equal(saves[0].body.dockerfile_source, 'inline')
  assert.equal(saves[0].body.build_context, '.')
  assert.equal(saves[0].body.dockerfile_content, state.form.dockerfile_content)
  const next = setup({ config: saves[0].body })
  await next.state.load()
  assert.equal(next.state.form.dockerfile_source, 'inline')
  assert.equal(next.state.form.dockerfile_content, state.form.dockerfile_content)
  state.form.dockerfile_content = '# missing FROM'
  await state.save()
  assert.equal(saves.length, 1)
  assert.match(warnings.at(-1), /必须包含 FROM/)
  stop(); next.stop()
})

test('save failure leaves the dialog retryable and a repeated click sends one request', async () => {
  let rejectSave,
    calls = 0
  const { state, stop } = setup({
    api: {
      saveBuildConfigApi: () => {
        calls++
        return new Promise((_, reject) => {
          rejectSave = reject
        })
      }
    }
  })
  state.form.build_mode = 'artifact'
  state.form.includes = 'main, release/*'
  const saving = state.save()
  await state.save()
  assert.equal(calls, 1)
  rejectSave(new Error('save failed'))
  await assert.rejects(saving, /save failed/)
  assert.equal(state.saving.value, false)
  assert.equal(state.form.includes, 'main, release/*')
  stop()
})
