import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import vm from 'node:vm'
import { parse, compileScript } from '@vue/compiler-sfc'
import ts from 'typescript'
import * as vue from 'vue'

// Run the actual SFC setup and lifecycle with Vue's in-memory renderer.
// No browser, network, real timers, or additional test dependencies are needed.
const filename = new URL('../src/views/cicd/workflow-run/index.vue', import.meta.url)
const { descriptor } = parse(readFileSync(filename, 'utf8'))
const script = compileScript(descriptor, { id: 'workflow-run-test' }).content
const code = ts.transpileModule(script, {
  compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 }
}).outputText
const durationModule = { exports: {} }
const actorModule = { exports: {} }
vm.runInNewContext(ts.transpileModule(readFileSync(new URL('../src/utils/workflow-actor.ts', import.meta.url), 'utf8'), {
  compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 }
}).outputText, { module: actorModule, exports: actorModule.exports })
vm.runInNewContext(ts.transpileModule(readFileSync(new URL('../src/utils/execution-duration.ts', import.meta.url), 'utf8'), {
  compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 }
}).outputText, { module: durationModule, exports: durationModule.exports })
const flush = async () => {
  for (let i = 0; i < 8; i++) await new Promise(setImmediate)
}
const task = (id, status = 'running') => ({
  id,
  node_type: 'build',
  node_key: `build-${id}`,
  status,
  provider_run_id: `build:${id}`
})

function mount(overrides = {}) {
  let latest = { tasks: [task(1), task(2)], status: 'running' }
  const calls = []
  const api = {
    getProjectApi: async () => ({ id: 1, name: 'Project' }),
    getWorkflowRunApi: async () => latest,
    getWorkflowApi: async () => ({ name: 'Workflow' }),
    listEnvironmentsApi: async () => [],
    getBuildRunApi: async (id) => ({ id, status: 'running', error_message: 'Insufficient cpu' }),
    getBuildLogsApi: async (id) => {
      calls.push(id)
      return [{ log_text: `build ${id} waiting for cpu` }]
    },
    ...overrides
  }
  let nextTimer = 0
  const timers = new Map()
  const intervals = new Map()
  const module = { exports: {} }
  vm.runInNewContext(code, {
    module,
    exports: module.exports,
    require: (name) => {
      if (name === 'vue') return vue
      if (name === 'vue-router')
        return {
          useRoute: () => ({ params: { projectId: 1, workflowId: 1, runId: 1 } }),
          useRouter: () => ({ back() {} })
        }
      if (name === '@/api/axiops') return api
      if (name === '@/utils/execution-duration') return durationModule.exports
      if (name === '@/utils/workflow-actor') return actorModule.exports
      return {}
    },
    Error,
    setTimeout: (callback) => {
      timers.set(++nextTimer, callback)
      return nextTimer
    },
    clearTimeout: (id) => timers.delete(id),
    setInterval: (callback) => { intervals.set(++nextTimer, callback); return nextTimer },
    clearInterval: (id) => intervals.delete(id)
  })
  const renderer = vue.createRenderer({
    createComment: () => ({}),
    createText: () => ({}),
    createElement: () => ({}),
    insert() {},
    remove() {},
    setText() {},
    setElementText() {},
    patchProp() {},
    parentNode: () => null,
    nextSibling: () => null
  })
  let state
  const app = renderer.createApp({
    setup() {
      state = module.exports.default.setup({}, { expose() {} })
      return () => null
    }
  })
  app.mount({})
  return {
    state,
    calls,
    unmount: () => app.unmount(),
    timers,
    intervals,
    setRun: (value) => {
      latest = value
    },
    tick: async () => {
      const callbacks = [...timers.values()]
      timers.clear()
      callbacks.forEach((callback) => callback())
      await flush()
    }
  }
}

test('polls selected logs and task status, then stops after unmount', async () => {
  const view = mount()
  await flush()
  view.state.toggleTask(task(1))
  await flush()
  assert.equal(view.state.logs.value[0], 'build 1 waiting for cpu')
  assert.equal(view.state.buildRun.value.error_message, 'Insufficient cpu')
  view.setRun({ tasks: [task(1, 'failed')], status: 'failed' })
  await view.tick()
  assert.equal(view.state.activeTask.value.status, 'failed')
  assert.equal(view.calls.length, 2)
  view.unmount()
  assert.equal(view.timers.size, 0)
  assert.equal(view.intervals.size, 0)
})

test('each service stage has its own elapsed time and completed stages stop counting', async () => {
  const view = mount(); await flush()
  const start = '2026-10-15T09:00:00Z'
  const build = { started_at: start, finished_at: '2026-10-15T09:02:05Z' }
  const deploy = { started_at: '2026-10-15T09:02:05Z' }
  view.state.clockNow.value = Date.parse('2026-10-15T09:03:10Z')
  assert.equal(view.state.taskDuration(build), '2 分 5 秒')
  assert.equal(view.state.taskDuration(deploy), '1 分 5 秒')
  view.state.clockNow.value += 60000
  assert.equal(view.state.taskDuration(build), '2 分 5 秒')
  assert.equal(view.state.taskDuration(deploy), '2 分 5 秒')
  assert.equal(view.state.taskDuration({}), '-')
  view.unmount(); assert.equal(view.intervals.size, 0)
})

test('deployment metadata reads its matching build snapshot, never invents main', async () => {
  const deployment = { id: 86, node_type: 'yaml_deploy', node_snapshot: { config: { service_id: 17 } } }
  const view = mount({ getWorkflowRunApi: async () => ({
    status: 'failed', context_snapshot: { branch: 'test' },
    tasks: [deployment, { id: 85, node_type: 'build', node_snapshot: { config: { service_id: 17, branch: 'test' } } }]
  }) })
  await flush()
  view.state.toggleTask(deployment)
  await flush()
  assert.equal(view.state.taskMeta.value.branch, 'test')
  view.state.run.value = { tasks: [] }
  assert.equal(view.state.taskMeta.value.branch, '-')
  view.unmount()
})

test('shows log request errors and recovers on the next poll', async () => {
  let fail = true
  const view = mount({
    getBuildLogsApi: async () => {
      if (fail) throw new Error('403 forbidden')
      return [{ log_text: 'recovered' }]
    }
  })
  await flush()
  view.state.toggleTask(task(1))
  await flush()
  assert.match(view.state.logError.value, /403 forbidden/)
  assert.match(view.state.emptyLogText.value, /读取失败/)
  fail = false
  await view.tick()
  assert.equal(view.state.logError.value, '')
  assert.equal(view.state.logs.value[0], 'recovered')
  view.unmount()
})

test('deployment logs persist through running, failed and successful states without build log requests', async () => {
  const view = mount()
  const deploy = { id: 3, node_key: 'deploy', node_type: 'yaml_deploy', status: 'running', log_text: '[apply] submitted\n[rollout] ready=false' }
  view.setRun({ tasks: [deploy], status: 'running' })
  await flush()
  view.state.toggleTask(deploy)
  await flush()
  assert.match(view.state.taskLogTitle.value, /部署检查/)
  assert.match(view.state.taskLogText.value, /ready=false/)
  assert.equal(view.calls.length, 0)
  view.setRun({ tasks: [{ ...deploy, status: 'failed', log_text: '[failed] ImagePullBackOff' }], status: 'failed' })
  await view.tick()
  assert.match(view.state.taskLogText.value, /ImagePullBackOff/)
  view.setRun({ tasks: [{ ...deploy, status: 'succeeded', log_text: '[succeeded] rollout passed' }], status: 'succeeded' })
  await view.tick()
  assert.match(view.state.taskLogText.value, /rollout passed/)
  view.setRun({ tasks: [{ ...deploy, status: 'succeeded', log_text: '' }], status: 'succeeded' })
  await view.tick()
  assert.match(view.state.emptyLogText.value, /未保存/)
  assert.doesNotMatch(view.state.emptyLogText.value, /等待 Kubernetes/)
  view.unmount()
})

test('ignores a delayed response after switching tasks without overlapping polls', async () => {
  let release
  const calls = []
  const view = mount({
    getBuildLogsApi: async (id) => {
      calls.push(id)
      if (id === 1)
        return await new Promise((resolve) => {
          release = resolve
        })
      return [{ log_text: 'second task' }]
    }
  })
  await flush()
  view.state.toggleTask(task(1))
  await flush()
  view.state.toggleTask(task(2))
  await flush()
  assert.deepEqual(calls, [1])
  release([{ log_text: 'stale first task' }])
  await flush()
  assert.equal(view.state.logs.value.length, 0)
  await view.tick()
  assert.equal(view.state.logs.value[0], 'second task')
  assert.equal(view.state.activeTask.value.id, 2)
  view.unmount()
})

test('isolated test jobs fetch container logs and preserve workflow diagnostics', async () => {
 const mounted=mount();await flush()
 const item={...task(8),node_type:'test',log_text:'[timeout] stopping test'}
 mounted.setRun({tasks:[item],status:'running'})
 mounted.state.toggleTask(item);await flush()
 assert.ok(mounted.calls.includes(8))
 assert.match(mounted.state.taskLogText.value,/build 8/)
 assert.match(mounted.state.taskLogText.value,/stopping test/)
 mounted.unmount()
})

test('retry backoff clears the previous attempt container output', async () => {
 const mounted=mount();await flush();const item=task(8)
 mounted.setRun({tasks:[item],status:'running'});mounted.state.toggleTask(item);await flush()
 assert.match(mounted.state.taskLogText.value,/build 8/)
 mounted.setRun({tasks:[{...item,status:'pending',provider_run_id:'',log_text:'[retry] waiting'}],status:'running'})
 await mounted.tick()
 assert.doesNotMatch(mounted.state.taskLogText.value,/build 8/)
 assert.match(mounted.state.taskLogText.value,/waiting/)
 mounted.unmount()
})
