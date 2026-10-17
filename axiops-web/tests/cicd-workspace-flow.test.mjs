import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import vm from 'node:vm'
import { parse, compileScript, compileTemplate } from '@vue/compiler-sfc'
import ts from 'typescript'
import * as vue from 'vue'
import * as yaml from 'js-yaml'

const transpile = (source) =>
  ts.transpileModule(source, {
    compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 }
  }).outputText
const utils = { exports: {} }
const actorModule = { exports: {} }
vm.runInNewContext(
  transpile(readFileSync(new URL('../src/utils/workflow-actor.ts', import.meta.url), 'utf8')),
  { module: actorModule, exports: actorModule.exports }
)
const orderUtils = { exports: {} }
vm.runInNewContext(
  transpile(
    readFileSync(new URL('../src/utils/workflow-service-order.ts', import.meta.url), 'utf8')
  ),
  { module: orderUtils, exports: orderUtils.exports }
)
const environmentUtils = { exports: {} }
vm.runInNewContext(
  transpile(
    readFileSync(new URL('../src/utils/service-environment-config.ts', import.meta.url), 'utf8')
  ),
  { module: environmentUtils, exports: environmentUtils.exports, require: () => yaml }
)
vm.runInNewContext(
  transpile(readFileSync(new URL('../src/utils/cicd-workspace.ts', import.meta.url), 'utf8')),
  {
    module: utils,
    exports: utils.exports,
    require: (name) => (name === './workflow-service-order' ? orderUtils.exports : {})
  }
)
const path = '../src/views/cicd/project-workspace/index.vue'
const { descriptor } = parse(readFileSync(new URL(path, import.meta.url), 'utf8'), {
  templateParseOptions: { expressionPlugins: ['typescript'] }
})
const code = transpile(compileScript(descriptor, { id: 'workspace-flow' }).content)
const workflow = {
  id: 4,
  target_environment_id: 3,
  code: 'risk-qa',
  name: 'QA',
  nodes: [
    { key: 'build', type: 'build', config: { service_id: 1, branch: 'main' } },
    { key: 'deploy', type: 'yaml_deploy', config: { service_id: 1 } }
  ]
}
const environment = {
  id: 3,
  project_id: 19,
  name: 'QA',
  code: 'qa',
  status: 'active',
  binding_status: 'validated',
  environment_type: 'k8s_namespace',
  cluster_id: 2,
  namespace: 'risk-qa'
}
const services = [1, 2].map((id) => ({
  id,
  name: `api-${id}`,
  code: `api-${id}`,
  current_revision_id: id + 10,
  repo_branch: 'main',
  code_source_id: 1,
  repo_url: 'https://git.example/team/api.git',
  registry_id: 1
}))

function setup(api = {}, query = {}) {
  let leaveGuard
  let state
  const module = { exports: {} },
    calls = [],
    warnings = [],
    routes = []
  const route = vue.reactive({ params: { projectId: '19' }, query })
  const defaults = {
    getProjectApi: async () => ({
      id: 19,
      code: 'risk',
      delivery_type: 'k8s_yaml',
      initialization_status: 'completed'
    }),
    listAllServicesApi: async () => services,
    listEnvironmentsApi: async () => [environment],
    listClustersApi: async () => [{ id: 2, name: '美国集群' }],
    listWorkflowsApi: async () => [workflow],
    listBuildRunsApi: async () => ({ list: [] }),
    listCodeSourcesApi: async () => [],
    listWorkflowRunsApi: async () => ({ list: [] }),
    getWorkflowApi: async () => workflow,
    getEnvironmentApi: async (id) => state.environments.value.find((e) => e.id === id),
    getServiceEnvironmentConfigApi: async (eid, sid) => ({
      environment_id: eid,
      service_id: sid,
      revision_id: sid + 10
    }),
    getBuildConfigApi: async () => ({
      id: 7,
      build_mode: 'dockerfile',
      image_repo: 'registry.example/team/api'
    }),
    createEnvironmentApi: async () => {
      throw Error('loading a page must not create an environment')
    },
    runWorkflowApi: async (id, body) => {
      calls.push({ id, body })
      return { id: calls.length + 100 }
    },
    updateProjectApi: async (id, body) => {
      calls.push({ id, body })
    },
    ...api
  }
  const getProject = defaults.getProjectApi
  defaults.getProjectApi = async (...args) => ({ ...(await getProject(...args)) })
  vm.runInNewContext(code, {
    module,
    exports: module.exports,
    URL,
    require: (name) => {
      if (name === 'vue') return { ...vue, onMounted() {}, onBeforeUnmount() {} }
      if (name === 'vue-router')
        return {
          onBeforeRouteLeave(guard) {
            leaveGuard = guard
          },
          useRoute: () => route,
          useRouter: () => ({
            push: async (to) => routes.push(to),
            replace: async (to) => routes.push(to)
          })
        }
      if (name === 'element-plus')
        return {
          ElMessage: {
            warning: (value) => warnings.push(value),
            success() {},
            error: (value) => warnings.push(value)
          }
        }
      if (name === '@/api/axiops') return defaults
      if (name === '@/utils/workflow-actor') return actorModule.exports
      if (name === '@/utils/cicd-workspace') return utils.exports
      if (name === '@/utils/workflow-service-order') return orderUtils.exports
      if (name === '@/utils/workflow-template')
        return {
          isDeploymentNode: (type) => ['yaml_deploy', 'helm_deploy', 'host_deploy'].includes(type)
        }
      if (name === '@/utils/service-environment-config') return environmentUtils.exports
      if (name === '@/composables/useAxiOpsAccess')
        return { useAxiOpsAccess: () => ({ can: () => true, load: async () => {} }) }
      return {}
    }
  })
  const scope = vue.effectScope()
  state = scope.run(() => module.exports.default.setup({}, { expose() {} }))
  state.project.value = { id: 19, code: 'risk', delivery_type: 'k8s_yaml' }
  state.services.value = services
  state.environments.value = [environment]
  state.clusters.value = [{ id: 2, name: '美国集群' }]
  return {
    state,
    calls,
    warnings,
    routes,
    route,
    leave: () => leaveGuard(),
    stop: () => scope.stop()
  }
}

test('Helm workspace offers Chart repository import and redirects legacy manual action to templates', async () => {
  const { state, stop } = setup()
  state.project.value.delivery_type = 'helm'
  state.openImport()
  assert.equal(state.wizardSource.value, 'git')
  assert.equal(state.serviceWizardVisible.value, true)
  assert.equal(state.bulkImportVisible.value, false, 'must not open the YAML-only importer')
  state.beginManualDirectory()
  assert.equal(state.templatePickerVisible.value, true)
  assert.equal(state.manualDirectoryOpen.value, false)
  await state.createManualDirectory()
  stop()
})

test('YAML file selection belongs to its service and is restored independently', async () => {
  const { state, stop } = setup()
  state.selectService(services[0])
  await state.selectFile('service.yaml', services[0])
  assert.equal(state.activeFile.value, 'service.yaml')
  state.selectService(services[1])
  assert.equal(state.activeFile.value, 'deployment.yaml')
  await state.selectFile('service.yaml', services[1])
  await state.selectFile('deployment.yaml', services[0])
  assert.equal(state.selectedService.value.id, services[0].id)
  assert.equal(state.activeFile.value, 'deployment.yaml')
  state.selectService(services[1])
  assert.equal(state.activeFile.value, 'service.yaml')
  state.selectService(services[0])
  assert.equal(state.activeFile.value, 'deployment.yaml')
  stop()
})

test('existing Helm repository action updates that service instead of opening creation', async () => {
  const { state, stop } = setup()
  const service = { ...services[1], delivery_type: 'helm', source_type: 'git' }
  const opened = []
  state.serviceConfiguration.value = {
    confirmLeave: async () => false,
    openRepositorySync: async () => opened.push(state.selectedService.value.id)
  }
  state.serviceContextMenu.value = { service }
  await state.openServiceRepositoryFromContext()
  assert.equal(opened.length, 0, 'unsaved changes must be respected')
  state.serviceConfiguration.value.confirmLeave = async () => true
  state.serviceContextMenu.value = { service }
  await state.openServiceRepositoryFromContext()
  assert.deepEqual(opened, [service.id])
  assert.equal(state.serviceWizardVisible.value, false)
  assert.equal(state.bulkImportVisible.value, false)
  stop()
})

test('initialization configuration entry opens the selected service and environment without changing their configuration', async () => {
  const { state, calls, stop } = setup({}, { init: '1', tab: 'environments' })
  const switches = []
  state.project.value.initialization_status = 'initializing'
  state.selectedEnvironmentId.value = environment.id
  state.serviceConfiguration.value = {
    switchEnvironment: async (id) => switches.push(id),
    confirmLeave: async () => true
  }
  await state.openEnvironmentServiceConfiguration(services[1].id)
  assert.equal(state.activeTab.value, 'services')
  assert.equal(state.selectedService.value.id, services[1].id)
  assert.deepEqual(switches, [environment.id])
  assert.ok(calls.length > 0)
  assert.ok(
    calls.every(
      (call) =>
        call.body.initialization_status === 'in_progress' && call.body.initialization_step === 2
    )
  )
  stop()
})

test('clicking another service file respects unsaved and in-progress configuration guards', async () => {
  const { state, stop } = setup()
  state.selectService(services[0])
  await state.selectFile('service.yaml', services[0])
  state.serviceConfiguration.value = { confirmLeave: async () => false }
  await state.selectFile('deployment.yaml', services[1])
  assert.equal(state.selectedService.value.id, services[0].id)
  assert.equal(state.activeFile.value, 'service.yaml')
  state.serviceConfiguration.value = { confirmLeave: async () => true }
  await state.selectFile('deployment.yaml', services[1])
  assert.equal(state.selectedService.value.id, services[1].id)
  assert.equal(state.activeFile.value, 'deployment.yaml')
  state.selectService(services[0])
  assert.equal(state.activeFile.value, 'service.yaml')
  stop()
})

test('workflow execution uses saved nodes and never assumes the first environment for an unmatched workflow', async () => {
  const unbound = { ...workflow, code: 'custom', target_environment_id: 0 }
  const { state, calls, stop } = setup({ getWorkflowApi: async () => unbound })
  assert.equal(state.workflowEnvironment(unbound), undefined)
  await state.openWorkflowExecute(unbound)
  assert.equal(state.executeEnvironmentId.value, 0)
  await state.confirmWorkflowExecute()
  assert.equal(calls.length, 0)
  state.executeEnvironmentId.value = environment.id
  await state.confirmWorkflowExecute()
  assert.equal(calls.length, 0, 'manually selecting an environment cannot bypass missing binding')
  assert.match(state.executeTargetDescription.value, /美国集群.*risk-qa/)
  stop()
})

test('execution defaults to one workflow-bound service and the service configured branch', async () => {
  const saved = {
    ...workflow,
    nodes: workflow.nodes.map((node) => ({ ...node, config: { ...node.config, branch: 'qa' } }))
  }
  const { state, calls, stop } = setup({ getWorkflowApi: async () => saved })
  state.selectedService.value = services[1]
  await state.openWorkflowExecute(workflow)
  assert.equal(state.executeScope.value, 'selected')
  assert.deepEqual(Array.from(state.executeServiceIds.value), ['1'])
  assert.equal(state.executeBranchValues.value[1], 'main')
  await state.confirmWorkflowExecute()
  assert.equal(calls.length, 1)
  assert.equal(calls[0].body.service_id, 1)
  assert.equal(calls[0].body.branch, 'main')
  assert.equal(calls[0].body.environment_id, environment.id)
  assert.equal(calls[0].body.use_environment_config, true)
  stop()
})

test('reopening execution resets batch selection and all-services mode to the workflow service', async () => {
  const { state, stop } = setup()
  await state.openWorkflowExecute(workflow)
  state.executeServiceIds.value = ['1', '2']
  state.executeScope.value = 'all'
  await state.openWorkflowExecute(workflow)
  assert.equal(state.executeScope.value, 'selected')
  assert.deepEqual(Array.from(state.executeServiceIds.value), ['1'])
  stop()
})

test('missing workflow service cannot be replaced, while the service branch wins over stale node branch', async () => {
  const missing = {
    ...workflow,
    nodes: workflow.nodes.map((node) => ({ ...node, config: { service_id: 999 } }))
  }
  const first = setup({ getWorkflowApi: async () => missing })
  first.state.selectedService.value = services[1]
  await first.state.openWorkflowExecute(missing)
  assert.equal(first.state.executeServiceIds.value.length, 0)
  await first.state.confirmWorkflowExecute()
  assert.equal(first.calls.length, 0)
  first.stop()
  const conflict = {
    ...workflow,
    nodes: [...workflow.nodes, { type: 'test', config: { service_id: 1, branch: 'other' } }]
  }
  const second = setup({ getWorkflowApi: async () => conflict })
  await second.state.openWorkflowExecute(conflict)
  assert.equal(second.state.executeBranchValues.value[1], 'main')
  await second.state.confirmWorkflowExecute()
  assert.equal(second.calls.length, 1)
  assert.equal(second.calls[0].body.service_id, 1)
  assert.equal(second.calls[0].body.branch, 'main')
  second.stop()
})

test('legacy execution directs missing binding to configuration without writing or running', async () => {
  const unbound = { ...workflow, target_environment_id: 0 }
  const { state, calls, routes, stop } = setup({ getWorkflowApi: async () => unbound })
  await state.openWorkflowExecute(unbound)
  assert.equal(state.executeEnvironment.value, undefined)
  await state.confirmWorkflowExecute()
  assert.equal(calls.length, 0)
  state.openWorkflowBinding()
  assert.equal(state.executeDialogVisible.value, false)
  assert.equal(routes.at(-1), '/cicd/projects/19/workflows/4?configure=1')
  assert.equal(state.bindExecuteEnvironment, undefined)
  stop()
})

test('initialization rejects unbound deployment list entries without expanded nodes', async () => {
  const { state, calls, warnings, stop } = setup({
    listWorkflowsApi: async () => [
      { ...workflow, nodes: undefined, requires_environment: true, target_environment_id: 0 }
    ]
  })
  await state.loadWorkspace()
  await state.completeInitialization()
  assert.equal(calls.length, 0)
  assert.equal(state.unboundDeploymentWorkflows.value.length, 1)
  assert.ok(warnings.length > 0)
  stop()
})
test('config read failure before submission remains retryable and never reports an uncertain deployment', async () => {
  let fail = true
  const { state, calls, stop } = setup({
    getServiceEnvironmentConfigApi: async (eid, sid) => {
      if (fail) throw Error('network timeout')
      return { environment_id: eid, service_id: sid, revision_id: 11 }
    }
  })
  await state.openWorkflowExecute(workflow)
  await state.confirmWorkflowExecute()
  assert.equal(calls.length, 0)
  assert.equal(state.executeUncertain.value, false)
  fail = false
  await state.confirmWorkflowExecute()
  assert.equal(calls.length, 1)
  stop()
})

test('independent multi-service workflows execute only selected service nodes', async () => {
  const multiple = {
    ...workflow,
    nodes: [
      ...workflow.nodes,
      { key: 'other-build', type: 'build', config: { service_id: 2, branch: 'main' } },
      {
        key: 'other-deploy',
        type: 'yaml_deploy',
        config: { service_id: 2 },
        depends_on: ['other-build']
      }
    ]
  }
  const { state, calls, stop } = setup({ getWorkflowApi: async () => multiple })
  await state.openWorkflowExecute(multiple)
  assert.equal(state.executeCanOverride.value, true)
  assert.equal(state.executeScope.value, 'selected')
  assert.deepEqual(Array.from(state.executeSelectableServices.value.map((s) => s.id)), [1, 2])
  state.executeServiceIds.value = ['2']
  await state.confirmWorkflowExecute()
  assert.equal(calls.length, 1)
  assert.equal(calls[0].body.service_id, 2)
  assert.equal(calls[0].body.select_workflow_service, true)
  stop()
})

test('cross-service dependencies cannot be split into one service', async () => {
  const multiple = {
    ...workflow,
    nodes: [
      ...workflow.nodes,
      { key: 'other', type: 'yaml_deploy', config: { service_id: 2 }, depends_on: ['build'] }
    ]
  }
  const { state, calls, stop } = setup({ getWorkflowApi: async () => multiple })
  await state.openWorkflowExecute(multiple)
  assert.equal(state.executeCanOverride.value, false)
  assert.equal(state.executeScope.value, 'all')
  state.executeScope.value = 'selected'
  state.executeServiceIds.value = ['1']
  await state.confirmWorkflowExecute()
  assert.equal(calls.length, 0)
  stop()
})

test('ordered deployments can be selected and moved before submission', async () => {
  const ordered = {
    ...workflow,
    nodes: [
      { ...workflow.nodes[0], key: 'build-gateway' },
      {
        ...workflow.nodes[1],
        key: 'deploy-gateway',
        config: { service_id: 1, start_order: 2 },
        depends_on: ['build-gateway']
      },
      { key: 'build-server', type: 'build', config: { service_id: 2, branch: 'main' } },
      {
        key: 'deploy-server',
        type: 'yaml_deploy',
        config: { service_id: 2, start_order: 1 },
        depends_on: ['build-server']
      }
    ]
  }
  const { state, calls, stop } = setup({ getWorkflowApi: async () => ordered })
  await state.openWorkflowExecute(ordered)
  assert.equal(state.executeCanOverride.value, true)
  assert.equal(state.executeScope.value, 'selected')
  assert.deepEqual(Array.from(state.executeWorkflowServiceIds.value), [2, 1])
  assert.deepEqual(Array.from(state.executeOrderedServiceNames.value), ['api-2', 'api-1'])
  assert.deepEqual(Array.from(state.executeSelectableServices.value.map((s) => s.id)), [2, 1])
  assert.deepEqual(Array.from(state.executeServiceIds.value), ['2'])
  state.selectExecuteService(1, true)
  assert.deepEqual(Array.from(state.executeServiceIds.value), ['2', '1'])
  state.moveExecuteService(1, -1)
  assert.deepEqual(Array.from(state.executeServiceIds.value), ['1', '2'])
  assert.deepEqual(Array.from(state.executeSelectableServices.value.map((s) => s.id)), [1, 2])
  await state.confirmWorkflowExecute()
  assert.equal(calls.length, 1)
  assert.deepEqual(Array.from(calls[0].body.services.map((s) => s.service_id)), [1, 2])
  assert.equal(calls[0].body.services[0].branch, 'main')
  assert.equal(calls[0].body.use_environment_config, true)
  assert.equal(calls[0].body.service_id, undefined)
  stop()
})

test('build-only workflows submit with environment zero; bound workflows reject a different target', async () => {
  const build = { ...workflow, target_environment_id: 0, nodes: [workflow.nodes[0]] }
  const first = setup({ getWorkflowApi: async () => build })
  await first.state.openWorkflowExecute(build)
  await first.state.confirmWorkflowExecute()
  assert.equal(first.calls[0].body.environment_id, 0)
  first.stop()
  const second = setup()
  second.state.environments.value.push({ ...environment, id: 8, code: 'prod', is_production: true })
  await second.state.openWorkflowExecute(workflow)
  second.state.executeEnvironmentId.value = 8
  await second.state.confirmWorkflowExecute()
  assert.equal(second.calls.length, 0)
  second.stop()
})

test('batch preflight failure creates no partial run and retry submits one run', async () => {
  const multiple = {
    ...workflow,
    nodes: [
      ...workflow.nodes,
      { key: 'build-b', type: 'build', config: { service_id: 2 } },
      { key: 'deploy-b', type: 'yaml_deploy', config: { service_id: 2 }, depends_on: ['build-b'] }
    ]
  }
  let fail = true
  const { state, calls, stop } = setup({
    getWorkflowApi: async () => multiple,
    getServiceEnvironmentConfigApi: async (eid, sid) => {
      if (sid === 2 && fail) throw Error('target invalid')
      return { environment_id: eid, service_id: sid, revision_id: sid + 10 }
    }
  })
  await state.openWorkflowExecute(multiple)
  state.executeScope.value = 'selected'
  state.executeServiceIds.value = ['1', '2']
  await state.confirmWorkflowExecute()
  assert.equal(calls.length, 0)
  assert.equal(state.executeSubmitted.value.length, 0)
  assert.equal(state.executeDialogVisible.value, true)
  assert.equal(state.executeUncertain.value, false)
  fail = false
  await state.confirmWorkflowExecute()
  assert.equal(calls.length, 1)
  assert.deepEqual(Array.from(calls[0].body.services.map((s) => s.service_id)), [1, 2])
  stop()
})

test('repeated clicks during submission cannot create duplicate runs', async () => {
  let finish,
    count = 0
  const { state, stop } = setup({
    runWorkflowApi: () => {
      count++
      return new Promise((resolve) => {
        finish = resolve
      })
    }
  })
  await state.openWorkflowExecute(workflow)
  const first = state.confirmWorkflowExecute()
  await state.confirmWorkflowExecute()
  await Promise.resolve()
  assert.equal(count, 1)
  finish({ id: 100 })
  await first
  stop()
})

test('an ambiguous network failure pauses resubmission until the user checks run records', async () => {
  let count = 0
  const { state, stop } = setup({
    runWorkflowApi: async () => {
      count++
      throw Error('timeout')
    }
  })
  await state.openWorkflowExecute(workflow)
  await state.confirmWorkflowExecute()
  assert.equal(state.executeUncertain.value, true)
  await state.confirmWorkflowExecute()
  assert.equal(count, 1)
  assert.match(state.executeSubmitMessage.value, /查看运行记录/)
  stop()
})

test('failed workspace reads show an error and never trigger environment creation', async () => {
  const { state, stop } = setup({
    listEnvironmentsApi: async () => {
      throw Error('permission denied')
    }
  })
  await state.loadWorkspace()
  assert.match(state.loadError.value, /环境列表读取失败.*permission denied/)
  stop()
})

test('readiness requires actual inputs and missing configs differ from failed reads', async () => {
  const { state, stop } = setup({
    getBuildConfigApi: async (id) => {
      if (id === 2) throw { businessCode: 30001 }
      return { id: 7, build_mode: 'dockerfile', image_repo: '' }
    }
  })
  await state.loadBuildStates()
  assert.equal(state.allBuildsReady.value, false)
  assert.match(state.buildIssues.value[1].join(), /产物路径/)
  assert.match(state.buildIssues.value[2].join(), /尚未配置/)
  stop()
})

test('completing initialization updates the project without deploying any environment', async () => {
  const { state, calls, stop } = setup()
  await state.loadWorkspace()
  await state.completeInitialization()
  assert.equal(calls.length, 1)
  assert.equal(calls[0].body.initialization_status, 'completed')
  assert.equal(calls[0].body.environment_id, undefined)
  stop()
})

test('service configuration opens without fetching cluster resources or per-service summaries', async () => {
  let buildCalls = 0,
    historyCalls = 0,
    runtimeCalls = 0
  const { state, stop } = setup(
    {
      getBuildConfigApi: async () => {
        buildCalls++
        return {}
      },
      listWorkflowRunsApi: async () => {
        historyCalls++
        return { list: [] }
      },
      listEnvWorkloadsApi: async () => {
        runtimeCalls++
        return []
      }
    },
    { tab: 'services' }
  )
  await state.loadWorkspace()
  await vue.nextTick()
  assert.equal(state.selectedService.value.id, 1)
  assert.equal(state.loading.value, false)
  assert.equal(buildCalls, 0)
  assert.equal(historyCalls, 0)
  assert.equal(runtimeCalls, 0)
  stop()
})

test('all changed workflow surfaces compile their templates', () => {
  for (const name of [
    path,
    '../src/views/cicd/projects/index.vue',
    '../src/views/cicd/workflow-job/index.vue',
    '../src/views/cicd/project-workspace/components/WorkflowDesignerDialog.vue',
    '../src/views/cicd/workflow-run/index.vue',
    '../src/views/cicd/project-workspace/components/ServiceResourcesEditor.vue',
    '../src/views/cicd/project-workspace/components/BuildConfigDialog.vue',
    '../src/views/cicd/project-workspace/components/ReleaseDraftDrawer.vue'
  ]) {
    const { descriptor, errors } = parse(readFileSync(new URL(name, import.meta.url), 'utf8'), {
      templateParseOptions: { expressionPlugins: ['typescript'] }
    })
    assert.equal(errors.length, 0)
    const script = compileScript(descriptor, { id: name })
    assert.equal(
      compileTemplate({
        source: descriptor.template.content,
        filename: name,
        id: name,
        compilerOptions: { bindingMetadata: script.bindings, expressionPlugins: ['typescript'] }
      }).errors.length,
      0
    )
  }
})

const pendingProject = {
  id: 19,
  code: 'risk',
  name: 'risk',
  delivery_type: 'k8s_yaml',
  initialization_status: 'pending',
  initialization_step: 0
}

const successfulImage = {
  id: 42,
  project_id: 19,
  service_id: 1,
  status: 'success',
  build_number: 1,
  image_url: 'registry.example/team/api:test-1',
  image_digest: `sha256:${'a'.repeat(64)}`
}

test('existing-image execution selects a successful digest and sends only the bound target config', async () => {
  const reads = []
  const { state, calls, stop } = setup({
    listBuildRunsApi: async (...args) => {
      reads.push(args)
      return { list: [successfulImage], total: 1 }
    }
  })
  await state.openWorkflowExecute(workflow)
  state.executeMode.value = 'existing_image'
  await state.loadPromotionBuilds(1)
  state.executeSourceBuildIds.value[1] = 42
  state.executeBranchValues.value[1] = 'must-not-build'
  await state.confirmWorkflowExecute()
  assert.equal(calls.length, 1)
  assert.equal(calls[0].body.execution_mode, 'existing_image')
  assert.equal(calls[0].body.source_build_id, 42)
  assert.equal(calls[0].body.environment_id, environment.id)
  assert.equal(calls[0].body.use_environment_config, true)
  assert.equal('branch' in calls[0].body, false)
  assert.equal(reads[0][1].service_id, 1)
  assert.equal(reads[0][1].status, 'success')
  stop()
})

test('missing, failed, wrong-service and unverified images never silently trigger a rebuild', async () => {
  const { state, calls, warnings, stop } = setup({
    listBuildRunsApi: async () => ({
      list: [
        { ...successfulImage, status: 'failed' },
        { ...successfulImage, service_id: 2 },
        { ...successfulImage, image_digest: '' }
      ],
      total: 3
    })
  })
  await state.openWorkflowExecute(workflow)
  state.executeMode.value = 'existing_image'
  await state.loadPromotionBuilds(1)
  state.executeSourceBuildIds.value[1] = 42
  await state.confirmWorkflowExecute()
  assert.equal(calls.length, 0)
  assert.equal(state.promotionBuilds[1].length, 0)
  assert.match(warnings.at(-1), /选择已有成功镜像/)
  assert.equal(state.executeMode.value, 'existing_image')
  stop()
})

test('production defaults to existing-image release rather than immediately building', async () => {
  const approved = {
    ...workflow,
    nodes: [{ key: 'approval', type: 'approval' }, ...workflow.nodes]
  }
  const { state, calls, stop } = setup({ getWorkflowApi: async () => approved })
  state.environments.value = [{ ...environment, is_production: true }]
  await state.openWorkflowExecute(approved)
  assert.equal(state.productionVisible.value, true)
  assert.equal(state.productionWorkflow.value.id, approved.id)
  assert.equal(state.executeDialogVisible.value, false)
  assert.equal(Object.keys(state.executeSourceBuildIds.value).length, 0)
  await state.confirmWorkflowExecute()
  assert.equal(calls.length, 0)
  stop()
})

test('explicit production build keeps the same environment and waits for the execute button', async () => {
  const production = { ...environment, is_production: true }
  const approved = {
    ...workflow,
    nodes: [{ key: 'approval', type: 'approval' }, ...workflow.nodes]
  }
  const v = setup({
    getWorkflowApi: async () => approved,
    getEnvironmentApi: async () => production
  })
  v.state.environments.value = [production]
  v.state.productionWorkflow.value = approved
  await v.state.openProductionBuild([1])
  assert.equal(v.state.executeDialogVisible.value, true)
  assert.equal(v.state.executeProductionBuild.value, true)
  assert.equal(v.state.executeEnvironment.value.id, production.id)
  assert.equal(v.calls.length, 0)
  await v.state.confirmWorkflowExecute()
  assert.equal(v.calls.length, 1)
  assert.equal(v.calls[0].body.environment_id, production.id)
  assert.equal(v.calls[0].body.execution_mode, 'build')
  assert.equal(v.calls[0].body.use_environment_config, true)
  v.stop()
})

test('production build can switch back to artifacts without submitting and preserves selected service', async () => {
  const production = { ...environment, is_production: true }
  const v = setup({ getEnvironmentApi: async () => production })
  v.state.environments.value = [production]
  await v.state.openWorkflowExecute(workflow, true)
  v.state.changeExecuteServices('2')
  v.state.executeMode.value = 'existing_image'
  v.state.changeExecuteMode()
  assert.equal(v.state.executeDialogVisible.value, false)
  assert.equal(v.state.productionVisible.value, true)
  assert.equal(v.state.productionWorkflow.value.target_environment_id, production.id)
  assert.equal(JSON.stringify(v.state.productionInitialServiceIds.value), '[2]')
  assert.equal(v.calls.length, 0)
  v.stop()
})

test('test execution defaults to build and collapsed advanced; service picker rejects unknown services', async () => {
  const v = setup()
  await v.state.openWorkflowExecute(workflow)
  assert.equal(v.state.executeMode.value, 'build')
  assert.equal(v.state.executeAdvanced.value, false)
  v.state.changeExecuteServices('2')
  assert.equal(JSON.stringify(v.state.executeServiceIds.value), '["2"]')
  v.state.changeExecuteServices('999')
  assert.equal(v.state.executeServiceIds.value.length, 0)
  v.state.executeAdvanced.value = true
  await v.state.openWorkflowExecute(workflow)
  assert.equal(v.state.executeAdvanced.value, false)
  v.stop()
})

test('explicit build fails closed when the workflow target changes before the form loads', async () => {
  const current = { ...workflow, target_environment_id: 8 }
  const v = setup({
    getWorkflowApi: async () => current,
    getEnvironmentApi: async () => ({ ...environment, id: 8 })
  })
  await v.state.openWorkflowExecute(workflow, true)
  assert.match(v.state.executeLoadError.value, /目标环境已变化/)
  await v.state.confirmWorkflowExecute()
  assert.equal(v.calls.length, 0)
  v.stop()
})

test('latest workflow binding redirects a stale test card to production before submission', async () => {
  const production = { ...environment, id: 8, is_production: true }
  const current = { ...workflow, target_environment_id: 8 }
  const v = setup({ getWorkflowApi: async () => current })
  v.state.environments.value = [environment, production]
  await v.state.openWorkflowExecute(workflow)
  assert.equal(v.state.productionVisible.value, true)
  assert.equal(v.state.executeDialogVisible.value, false)
  assert.equal(v.state.executeWorkflowTarget.value, null)
  assert.equal(v.state.productionWorkflow.value.target_environment_id, 8)
  await v.state.confirmWorkflowExecute()
  assert.equal(v.calls.length, 0)
  v.stop()
})

test('opening execution refreshes the production flag even when environment id is unchanged', async () => {
  const production = { ...environment, name: '生产环境', code: 'prod', is_production: true }
  const v = setup({ getEnvironmentApi: async () => production })
  v.state.environments.value = [{ ...production, is_production: false }]
  await v.state.openWorkflowExecute(workflow)
  assert.equal(v.state.productionVisible.value, true)
  assert.equal(v.state.executeDialogVisible.value, false)
  assert.equal(v.state.environments.value[0].is_production, true)
  await v.state.confirmWorkflowExecute()
  assert.equal(v.calls.length, 0)
  v.stop()
})

test('unreadable or mismatched fresh environment blocks ordinary execution', async () => {
  for (const getEnvironmentApi of [
    async () => {
      throw Error('environment access denied')
    },
    async () => ({ ...environment, project_id: 99 })
  ]) {
    const v = setup({ getEnvironmentApi })
    await v.state.openWorkflowExecute(workflow)
    assert.ok(v.state.executeLoadError.value)
    await v.state.confirmWorkflowExecute()
    assert.equal(v.calls.length, 0)
    v.stop()
  }
})

test('test execution always resets to building after a prior image-only run', async () => {
  const v = setup()
  v.state.executeMode.value = 'existing_image'
  await v.state.openWorkflowExecute(workflow)
  assert.equal(v.state.executeMode.value, 'build')
  await v.state.confirmWorkflowExecute()
  assert.equal(v.calls.length, 1)
  assert.equal(v.calls[0].body.source_build_id, undefined)
  v.stop()
})

test('source build shortcut preserves exact service order but waits for explicit execution', async () => {
  const multi = {
    ...workflow,
    nodes: [1, 2].flatMap((id) => [
      { key: `build-${id}`, type: 'build', config: { service_id: id } },
      {
        key: `deploy-${id}`,
        type: 'yaml_deploy',
        depends_on: [`build-${id}`],
        config: { service_id: id }
      }
    ])
  }
  const v = setup(
    { getWorkflowApi: async () => multi },
    { execute: '4', build_services: '2,1', build_environment: '3' }
  )
  v.state.workflows.value = [multi]
  await v.state.openRequestedExecution()
  assert.equal(v.state.executeMode.value, 'build')
  assert.equal(JSON.stringify(v.state.executeServiceIds.value), '["2","1"]')
  assert.equal(v.calls.length, 0)
  assert.match(v.state.executeSubmitMessage.value, /先在来源环境/)
  assert.equal(v.routes.at(-1).query.build_services, undefined)
  v.stop()
})

test('source build shortcut rejects missing services, duplicates and changed environment binding', async () => {
  for (const query of [
    { build_services: '999', build_environment: '3' },
    { build_services: '1,1', build_environment: '3' },
    { build_services: '1', build_environment: '999' }
  ]) {
    const v = setup({}, { execute: '4', ...query })
    v.state.workflows.value = [workflow]
    await v.state.openRequestedExecution()
    assert.ok(v.state.executeLoadError.value)
    assert.equal(v.state.executeServiceIds.value.length, 0)
    await v.state.confirmWorkflowExecute()
    assert.equal(v.calls.length, 0)
    v.stop()
  }
})

test('promotion pagination is explicit and switching workflows resets selected source', async () => {
  const pages = []
  const { state, stop } = setup({
    listBuildRunsApi: async (_, options) => {
      pages.push(options.page)
      return { list: [{ ...successfulImage, id: options.page }], total: 21 }
    }
  })
  await state.openWorkflowExecute(workflow)
  state.executeMode.value = 'existing_image'
  await state.loadPromotionBuilds(1)
  assert.equal(state.promotionHasMore[1], true)
  await state.loadPromotionBuilds(1, true)
  assert.equal(state.promotionBuilds[1].length, 2)
  assert.equal(state.promotionHasMore[1], false)
  assert.deepEqual(pages, [1, 2])
  state.executeSourceBuildIds.value[1] = 1
  await state.openWorkflowExecute(workflow)
  assert.equal(state.executeMode.value, 'build')
  assert.equal(Object.keys(state.executeSourceBuildIds.value).length, 0)
  stop()
})

test('workflows with test or non-YAML execution do not offer unsupported image reuse', async () => {
  const saved = { ...workflow, nodes: [...workflow.nodes, { key: 'test', type: 'test' }] }
  const { state, calls, stop } = setup({ getWorkflowApi: async () => saved })
  await state.openWorkflowExecute(saved)
  assert.equal(state.executeCanPromote.value, false)
  state.executeMode.value = 'existing_image'
  await state.confirmWorkflowExecute()
  assert.equal(calls.length, 0)
  stop()
})

test('reopening an unfinished project resumes the first incomplete step without creating anything', async () => {
  const cases = [
    { serviceList: [], envs: [], expected: 'environments' },
    {
      serviceList: [{ ...services[0], current_revision_id: 0 }],
      envs: [],
      expected: 'environments'
    },
    { serviceList: services, envs: [], expected: 'environments' },
    { serviceList: services, envs: [environment], missingBuild: true, expected: 'build' },
    { serviceList: services, envs: [environment], expected: 'workflows' },
    {
      serviceList: services,
      envs: [{ ...environment, binding_status: 'pending' }],
      expected: 'environments'
    }
  ]
  for (const item of cases) {
    const { state, calls, routes, stop } = setup({
      getProjectApi: async () => ({ ...pendingProject, initialization_step: 4 }),
      listAllServicesApi: async () => item.serviceList,
      listEnvironmentsApi: async () => item.envs,
      listWorkflowsApi: async () => [],
      ...(item.missingBuild
        ? {
            getBuildConfigApi: async () => {
              throw { businessCode: 30001 }
            }
          }
        : {})
    })
    await state.loadWorkspace()
    assert.equal(state.activeTab.value, item.expected)
    assert.equal(state.initializationMode.value, true)
    assert.equal(state.project.value.initialization_status, 'pending')
    assert.equal(routes.at(-1).query.tab, item.expected)
    assert.equal(calls.length, 0)
    stop()
  }
})

test('explicit initialization tabs cannot skip missing setup; saved service editing is retained', async () => {
  const first = setup(
    {
      getProjectApi: async () => pendingProject,
      listEnvironmentsApi: async () => [],
      listWorkflowsApi: async () => []
    },
    { tab: 'workflows' }
  )
  await first.state.loadWorkspace()
  assert.equal(first.state.activeTab.value, 'environments')
  await first.state.selectTab('workflows')
  assert.equal(first.calls.length, 0)
  first.stop()
  const second = setup(
    { getProjectApi: async () => pendingProject },
    { init: '1', tab: 'services' }
  )
  await second.state.loadWorkspace()
  assert.equal(second.state.activeTab.value, 'services')
  await second.state.handleServiceSaved()
  assert.equal(second.state.activeTab.value, 'services')
  assert.equal(second.calls.at(-1).body.initialization_status, 'in_progress')
  assert.equal(second.calls.at(-1).body.initialization_step, 2)
  second.stop()
})

for (const selection of ['default', 'qa', 'dev', 'prod', 'custom']) {
  test(`environment selection ${selection} creates exactly one environment and one matching workflow`, async () => {
    const envs = [],
      flows = [],
      created = []
    const { state, calls, stop } = setup({
      getProjectApi: async () => pendingProject,
      listEnvironmentsApi: async () => [...envs],
      listWorkflowsApi: async () => [...flows],
      createEnvironmentApi: async (_, body) => {
        created.push({ ...body })
        envs.push({ ...environment, ...body, id: 10 })
        return { id: 10 }
      },
      createWorkflowApi: async (_, body) => {
        assert.equal(body.target_environment_id, 10)
        flows.push({ ...body, id: 20, requires_environment: true, nodes: body.definition.nodes })
        return { id: 20 }
      }
    })
    await state.loadWorkspace()
    await state.nextInitStep()
    assert.equal(state.environmentCreatorVisible.value, true)
    assert.equal(state.environmentForm.code, 'qa')
    assert.equal(created.length, 0, 'opening the dialog must not create a default environment')
    if (selection !== 'default') state.environmentPresetCode.value = selection
    if (selection === 'custom') {
      state.environmentForm.code = 'uat'
      state.environmentForm.name = '验收环境'
      state.syncNamespace()
    }
    const expected = selection === 'default' ? 'qa' : selection === 'custom' ? 'uat' : selection
    await state.createEnvironment()
    assert.equal(created.length, 1)
    assert.equal(created[0].code, expected)
    assert.equal(created[0].namespace, `risk-${expected}`)
    assert.equal(created[0].is_production, selection === 'prod')
    await state.loadBuildStates()
    await state.generateWorkflows()
    await state.generateWorkflows()
    assert.equal(flows.length, 1)
    assert.equal(flows[0].code, `risk-${expected}`)
    assert.equal(envs.length, 1)
    assert.ok(calls.every((call) => call.body.initialization_status !== 'completed'))
    stop()
  })
}

for (const code of ['prod', 'production']) {
  test(`typing reserved ${code} in a custom environment cannot create a non-production target`, async () => {
    const created = []
    const v = setup({
      createEnvironmentApi: async (_, input) => {
        created.push(input)
        return { id: 10 }
      }
    })
    v.state.openEnvironmentCreator()
    v.state.environmentPresetCode.value = 'custom'
    v.state.environmentForm.name = 'Production'
    v.state.environmentForm.code = code
    v.state.syncNamespace()
    assert.equal(v.state.environmentForm.is_production, true)
    assert.equal(v.state.productionEnvironmentCode.value, true)
    v.state.environmentForm.is_production = false
    await v.state.createEnvironment()
    assert.equal(created[0].is_production, true)
    v.stop()
  })
}

test('empty selection defaults to test, while repeated create clicks and duplicate codes cannot add environments', async () => {
  let finish,
    count = 0
  const envs = []
  const { state, stop } = setup({
    getProjectApi: async () => pendingProject,
    listEnvironmentsApi: async () => envs,
    createEnvironmentApi: async (_, body) => {
      count++
      await new Promise((resolve) => {
        finish = resolve
      })
      envs.push({ ...environment, ...body })
      return { id: environment.id }
    }
  })
  await state.loadWorkspace()
  state.openEnvironmentCreator()
  state.environmentPresetCode.value = 'prod'
  state.environmentPresetCode.value = ''
  assert.equal(state.environmentForm.code, 'qa')
  const saving = state.createEnvironment()
  await state.createEnvironment()
  assert.equal(count, 1)
  finish()
  await saving
  await state.createEnvironment()
  assert.equal(count, 1)
  stop()
})

test('workflow retries read persisted successes and never duplicate workflows after a partial failure', async () => {
  const prod = { ...environment, id: 4, code: 'prod', name: '生产环境', is_production: true }
  const flows = [],
    attempted = []
  let fail = true
  const { state, stop } = setup({
    listEnvironmentsApi: async () => [environment, prod],
    listWorkflowsApi: async () => [...flows],
    createWorkflowApi: async (_, body) => {
      attempted.push(body.target_environment_id)
      if (body.target_environment_id === 4 && fail) throw Error('network error')
      flows.push({ ...body, id: flows.length + 1, requires_environment: true })
      return { id: flows.length }
    }
  })
  await state.loadWorkspace()
  await state.loadBuildStates()
  assert.equal(await state.generateWorkflows(), undefined)
  assert.equal(flows.length, 1)
  fail = false
  assert.equal(await state.generateWorkflows(), true)
  assert.equal(flows.length, 2)
  assert.deepEqual(attempted, [3, 4, 4])
  stop()
})

test('failed progress save keeps the current step; retry persists in-progress without marking complete', async () => {
  let fail = true
  const writes = []
  const { state, warnings, stop } = setup(
    {
      getProjectApi: async () => pendingProject,
      updateProjectApi: async (_, body) => {
        if (fail) throw Error('network timeout')
        writes.push(body)
      }
    },
    { tab: 'services' }
  )
  await state.loadWorkspace()
  await state.selectTab('environments')
  assert.equal(state.activeTab.value, 'services')
  assert.match(warnings.join(), /network timeout/)
  fail = false
  await state.selectTab('environments')
  assert.equal(state.activeTab.value, 'environments')
  assert.equal(writes[0].initialization_status, 'in_progress')
  assert.equal(writes[0].initialization_step, 1)
  stop()
})

test('all saved settings still require explicit completion, and a completed project ignores stale init links', async () => {
  const first = setup({
    getProjectApi: async () => ({ ...pendingProject, initialization_step: 4 })
  })
  await first.state.loadWorkspace()
  assert.equal(first.state.activeTab.value, 'workflows')
  assert.equal(first.state.initializationMode.value, true)
  assert.equal(first.calls.length, 0)
  await first.state.completeInitialization()
  assert.equal(first.calls.length, 1)
  assert.equal(first.calls[0].body.initialization_status, 'completed')
  assert.equal(first.state.project.value.initialization_status, 'completed')
  first.stop()
  const second = setup({}, { init: '1' })
  await second.state.loadWorkspace()
  assert.equal(second.state.initializationMode.value, false)
  assert.equal(second.routes.at(-1).query.init, undefined)
  assert.equal(second.calls.length, 0)
  second.stop()
})

test('missing service definitions and invalid environment bindings prevent completion', async () => {
  const first = setup({
    listAllServicesApi: async () => [{ ...services[0], current_revision_id: 0 }]
  })
  await first.state.loadWorkspace()
  await first.state.completeInitialization()
  assert.equal(first.calls.length, 0)
  first.stop()
  const second = setup({
    listEnvironmentsApi: async () => [{ ...environment, status: 'disabled' }]
  })
  await second.state.loadWorkspace()
  await second.state.completeInitialization()
  assert.equal(second.calls.length, 0)
  second.stop()
})

test('leaving service setup respects the existing unsaved YAML confirmation', async () => {
  const { state, leave, stop } = setup()
  state.activeTab.value = 'services'
  state.serviceConfiguration.value = { confirmLeave: async () => false }
  assert.equal(await leave(), false)
  state.serviceConfiguration.value = { confirmLeave: async () => true }
  assert.equal(await leave(), true)
  state.activeTab.value = 'environments'
  assert.equal(await leave(), true)
  stop()
})
