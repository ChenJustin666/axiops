import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import ts from 'typescript'
import * as yaml from 'js-yaml'

const source = readFileSync(new URL('../src/utils/yaml-editing.ts', import.meta.url), 'utf8')
const code = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.ES2022 } }).outputText
// A temporary module is unnecessary: resolve the two installed formatter imports first.
const moduleSource = code.replace("import('prettier/standalone')", `import('${import.meta.resolve('prettier/standalone')}')`)
  .replace("import('prettier/plugins/yaml')", `import('${import.meta.resolve('prettier/plugins/yaml')}')`)
const { indentYamlSelection, yamlNewline, formatYamlSource, yamlSyntaxSource } = await import(`data:text/javascript;base64,${Buffer.from(moduleSource).toString('base64')}`)
const apply = (text, edit) => text.slice(0, edit.start) + edit.text + text.slice(edit.end)

test('template syntax validation accepts a placeholder with a suffix and preserves error offsets', () => {
  const text = 'name: {{.Service}}-nacos-auth\nvalue: "optional:nacos:{{.Service}}"\n'
  const sanitized = yamlSyntaxSource(text)
  assert.equal(sanitized.length, text.length)
  assert.doesNotThrow(() => yaml.load(sanitized))
  assert.throws(() => yaml.load(yamlSyntaxSource(text + 'broken: [\n')))
})

test('multi-line Tab preserves content, Shift+Tab reverses it, and next unselected line stays unchanged', () => {
  const original = 'env:\n  - name: JAVA_OPTS\n    value: >-\n      -XX:+UseG1GC\nnext: value\n'
  const end = original.indexOf('next:')
  const edit = indentYamlSelection(original, 5, end)
  const changed = apply(original, edit)
  assert.match(changed, /\n    - name: JAVA_OPTS\n      value: >-\n        -XX:\+UseG1GC\nnext:/)
  assert.equal(apply(changed, indentYamlSelection(changed, edit.selectionStart, edit.selectionEnd, true)), original)
})

test('Tab aligns a cursor to two spaces; Shift+Tab removes only leading indentation', () => {
  assert.equal(apply(' x', indentYamlSelection(' x', 1, 1)), '  x')
  assert.equal(apply('  key: value', indentYamlSelection('  key: value', 6, 6, true)), 'key: value')
  assert.equal(apply('key: value', indentYamlSelection('key: value', 6, 6, true)), 'key: value')
})

test('Enter inherits indentation and indents a folded JAVA_OPTS block', () => {
  const source = '    value: >-'
  assert.equal(apply(source, yamlNewline(source, source.length, source.length)), source + '\n      ')
  const option = '      -Dspring.config.import=optional:nacos:gateway'
  assert.equal(apply(option, yamlNewline(option, option.length, option.length)), option + '\n      ')
})

test('formatting preserves comments, file markers, variables, documents and folded Java options', async () => {
  const source = '# AxiOps-File: deployment.yaml\napiVersion: apps/v1\nkind: Deployment\nmetadata: {name: {{.Service}}}\nspec:\n    replicas: {{.replicas}}\n    template:\n      spec:\n        containers:\n          - name: app\n            image: "{{.Image}}"\n            env:\n              - name: JAVA_OPTS\n                value: >-\n                  -XX:+UseG1GC\n                  -Dspring.config.import=optional:nacos:{{.Service}}\n---\n# AxiOps-File: service.yaml\n# keep this comment\nkind: Service\nmetadata: {name: {{.Service}}}\n'
  const result = await formatYamlSource(source)
  assert.match(result, /# AxiOps-File: deployment.yaml/)
  assert.match(result, /# AxiOps-File: service.yaml/)
  assert.match(result, /# keep this comment/)
  assert.match(result, /\n  replicas: \{\{\.replicas\}\}/)
  const render = (text) => text.replaceAll('{{.Service}}', 'gateway').replaceAll('{{.replicas}}', '2').replaceAll('{{.Image}}', 'registry/gateway:1')
  const parse = (text) => { const docs = []; yaml.loadAll(render(text), (doc) => docs.push(doc)); return docs }
  assert.deepEqual(parse(result), parse(source))
  assert.equal(await formatYamlSource(result), result)
})

test('invalid Java options indentation and complex Helm syntax fail without rewriting source', async () => {
  await assert.rejects(formatYamlSource('env:\n  - name: JAVA_OPTS\n    value: >-\n -XX:+UseG1GC\n'))
  await assert.rejects(formatYamlSource('spec:\n  replicas: {{ .Values.replicas }}\n'), /Helm/)
})
