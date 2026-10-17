import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import vm from 'node:vm'
import ts from 'typescript'
import { parse, compileScript, compileTemplate } from '@vue/compiler-sfc'
import * as vue from 'vue'

const paths = [
  '../src/components/build-log-viewer/LogConsole.vue',
  '../src/components/build-log-viewer/index.vue',
  '../src/views/cicd/workflow-run/index.vue',
  '../src/views/cicd/project-workspace/components/WorkflowRunDrawer.vue'
]
let code
for (const path of paths) {
  const filename = new URL(path, import.meta.url)
  const { descriptor } = parse(readFileSync(filename, 'utf8'))
  const script = compileScript(descriptor, { id: path })
  assert.deepEqual(
    compileTemplate({
      source: descriptor.template.content,
      filename: filename.pathname,
      id: path,
      compilerOptions: { bindingMetadata: script.bindings }
    }).errors,
    []
  )
  if (path.endsWith('LogConsole.vue'))
    code = ts.transpileModule(script.content, {
      compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 }
    }).outputText
}
const flush = async () => {
  for (let i = 0; i < 5; i++) await vue.nextTick()
}
function mount(initialState = {}, text = 'first\nsecond\nthird\n', viewportSize = {}) {
  const module = { exports: {} },
    snapshots = [],
    observers = []
  vm.runInNewContext(code, {
    module,
    exports: module.exports,
    require: (name) => (name === 'vue' ? vue : {}),
    ResizeObserver: class {
      constructor(callback) {
        this.callback = callback
        this.disconnected = false
        observers.push(this)
      }
      observe() {}
      disconnect() {
        this.disconnected = true
      }
    }
  })
  const props = vue.reactive({
    text,
    title: 'Build log',
    emptyText: 'waiting',
    loading: false,
    expanded: false,
    initialState: { following: true, wrap: true, top: 0, unread: false, ...initialState }
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
      state = module.exports.default.setup(props, {
        expose() {},
        emit: (event, value) => {
          if (event === 'state') snapshots.push(value)
        }
      })
      return () => null
    }
  })
  app.mount({})
  let top = 0
  state.viewport.value = {
    scrollHeight: 2000,
    clientHeight: 400,
    scrollLeft: 0,
    ...viewportSize,
    get scrollTop() {
      return top
    },
    set scrollTop(value) {
      top = Math.min(Math.max(value, 0), this.scrollHeight - this.clientHeight)
    }
  }
  return {
    state,
    props,
    snapshots,
    observers,
    element: state.viewport.value,
    unmount: () => app.unmount()
  }
}

test('opens at the end and follows new output without needing to drag the scrollbar', async () => {
  const view = mount()
  await flush()
  assert.equal(view.element.scrollTop, 1600)
  assert.equal(view.state.lineCount.value, 3)
  view.element.scrollHeight = 2400
  view.props.text += 'new output\n'
  await flush()
  assert.equal(view.element.scrollTop, 2000)
  assert.equal(view.state.following.value, true)
  view.unmount()
})

test('scrolling upward pauses follow and preserves position while new output arrives', async () => {
  const view = mount()
  await flush()
  view.element.scrollTop = 350
  view.state.onScroll()
  assert.equal(view.state.following.value, false)
  view.element.scrollHeight = 2800
  view.props.text += 'more output\n'
  await flush()
  assert.equal(view.element.scrollTop, 350)
  assert.equal(view.state.unread.value, true)
  view.state.jumpEnd()
  assert.equal(view.element.scrollTop, 2400)
  assert.equal(view.state.following.value, true)
  assert.equal(view.state.unread.value, false)
  view.unmount()
})

test('top pauses follow; manual scrolling to the end resumes; horizontal scroll cannot override a manual pause', async () => {
  const view = mount()
  await flush()
  view.state.jumpTop()
  assert.equal(view.element.scrollTop, 0)
  assert.equal(view.state.following.value, false)
  view.element.scrollTop = 1600
  view.state.onScroll()
  assert.equal(view.state.following.value, true)
  view.state.setFollowing(false)
  view.element.scrollLeft = 80
  view.state.onScroll()
  assert.equal(view.state.following.value, false)
  view.unmount()
})

test('fullscreen remount restores paused position and wrap preference; resizing follows only when enabled', async () => {
  const first = mount()
  await flush()
  first.element.scrollTop = 610
  first.state.onScroll()
  first.state.wrap.value = false
  await first.state.layoutChanged()
  first.unmount()
  assert.equal(first.observers[0].disconnected, true)
  const second = mount(first.snapshots.at(-1))
  await flush()
  assert.equal(second.element.scrollTop, 610)
  assert.equal(second.state.wrap.value, false)
  second.element.clientHeight = 600
  second.observers[0].callback()
  assert.equal(second.element.scrollTop, 610)
  second.state.jumpEnd()
  second.element.clientHeight = 800
  second.observers[0].callback()
  assert.equal(second.element.scrollTop, 1200)
  second.unmount()
})

test('large loaded logs remain intact and jump to the end; identical polls do not create unread output', async () => {
  const text = Array.from({ length: 50000 }, (_, i) => `line ${i + 1}`).join('\n')
  const view = mount({}, text)
  view.element.scrollHeight = 1100000
  await flush()
  assert.equal(view.state.lineCount.value, 50000)
  assert.equal(view.props.text, text)
  assert.equal(view.element.scrollTop, 1099600)
  view.state.jumpTop()
  view.props.text = text
  await flush()
  assert.equal(view.state.unread.value, false)
  view.unmount()
})

test('a fullscreen dialog becoming visible restores the saved position after its first measurable layout', async () => {
  const view = mount({ following: false, top: 700 }, 'saved output', { clientHeight: 0 })
  await flush()
  view.element.clientHeight = 500
  view.observers[0].callback()
  assert.equal(view.element.scrollTop, 700)
  assert.equal(view.state.following.value, false)
  view.unmount()
})
