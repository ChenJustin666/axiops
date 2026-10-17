import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import vm from 'node:vm'
import { compileScript, parse } from '@vue/compiler-sfc'
import ts from 'typescript'
import * as vue from 'vue'

// Compile the real page and header without the auto-import plugin. A missing
// header registration must not silently discard its named action slot.
function compile(file, mocks = {}) {
  const { descriptor } = parse(readFileSync(new URL(file, import.meta.url), 'utf8'))
  const script = compileScript(descriptor, { id: file, inlineTemplate: true }).content
  const code = ts.transpileModule(script, {
    compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 }
  }).outputText
  const module = { exports: {} }
  vm.runInNewContext(code, {
    module, exports: module.exports,
    require: name => name === 'vue' ? vue : mocks[name] || {}
  })
  return module.exports.default
}

const header = compile('../src/components/core/base/art-page-header/index.vue')
const renderer = vue.createRenderer({
  createElement: tag => ({ tag, children: [], props: {} }),
  createText: text => ({ text }),
  createComment: text => ({ comment: text }),
  setText: (node, text) => { node.text = text },
  setElementText: (node, text) => { node.text = text; node.children = [] },
  patchProp: (node, key, previous, next) => { node.props[key] = next },
  insert: (node, parent, anchor) => {
    if (node.parent) rendererRemove(node)
    node.parent = parent
    const index = anchor ? parent.children.indexOf(anchor) : -1
    parent.children.splice(index < 0 ? parent.children.length : index, 0, node)
  },
  remove: rendererRemove,
  parentNode: node => node.parent,
  nextSibling: node => node.parent?.children[node.parent.children.indexOf(node) + 1]
})
function rendererRemove(node) {
  if (node.parent) node.parent.children.splice(node.parent.children.indexOf(node), 1)
  node.parent = null
}
function nodes(root) { return [root, ...(root.children || []).flatMap(nodes)] }
function text(root) { return nodes(root).map(node => node.text || '').join('') }

async function mount(allowed) {
  const permissions = vue.ref(false)
  const errors = []
  const page = compile('../src/views/cicd/projects/index.vue', {
    '@/components/core/base/art-page-header/index.vue': { default: header, __esModule: true },
    '@/composables/useAxiOpsAccess': {
      useAxiOpsAccess: () => ({
        can: () => permissions.value,
        load: async () => { await Promise.resolve(); permissions.value = allowed }
      })
    },
    '@/api/axiops': { listProjectsApi: async () => ({ list: [], total: 0 }) },
    'vue-router': { useRouter: () => ({ push() {} }) },
    'element-plus': { ElMessage: { success() {} } },
    'lucide-vue-next': new Proxy({}, { get: () => vue.defineComponent({ render: () => null }) })
  })
  const app = renderer.createApp(page)
  app.config.warnHandler = message => errors.push(message)
  app.config.errorHandler = error => errors.push(error.message)
  for (const name of new Set(descriptorComponents())) {
    // Do not register ArtPageHeader globally: the page must import it itself.
    if (name === 'ArtPageHeader') continue
    app.component(name, vue.defineComponent({
      inheritAttrs: false,
      setup: (_, { attrs, slots }) => () => {
        if (name === 'ElDialog' && !attrs.modelValue) return null
        return vue.h(name === 'ElButton' ? 'button' : 'div', attrs, slots.default?.())
      }
    }))
  }
  app.directive('loading', {})
  const root = { children: [] }
  app.mount(root)
  await new Promise(resolve => setImmediate(resolve))
  await vue.nextTick()
  return { root, errors, app }
}
function descriptorComponents() {
  const source = readFileSync(new URL('../src/views/cicd/projects/index.vue', import.meta.url), 'utf8')
  return [...source.matchAll(/<(Art\w+|El\w+)\b/g)].map(match => match[1])
}

test('project header renders the permitted create action and opens its form', async () => {
  const { root, errors, app } = await mount(true)
  try {
    assert.ok(nodes(root).some(node => node.tag === 'header'))
    const button = nodes(root).find(node => node.tag === 'button' && text(node).trim() === '新建项目')
    assert.ok(button, 'the real header must render the actions slot')
    assert.ok(!errors.some(message => /ArtPageHeader|Failed to resolve component/.test(message)), errors.join('\n'))
    button.props.onClick()
    await vue.nextTick()
    assert.match(text(root), /开始新建项目/)
  } finally { app.unmount() }
})

test('project header retains the permission restriction for project creation', async () => {
  const { root, app } = await mount(false)
  try {
    assert.ok(!nodes(root).some(node => node.tag === 'button' && text(node).trim() === '新建项目'))
  } finally { app.unmount() }
})
