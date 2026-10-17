import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import vm from 'node:vm'
import ts from 'typescript'

const module = { exports: {} }
vm.runInNewContext(
  ts.transpileModule(
    readFileSync(new URL('../src/utils/workflow-service-order.ts', import.meta.url), 'utf8'),
    { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 } }
  ).outputText,
  { module, exports: module.exports }
)
const { workflowDeploymentOrder, withWorkflowDeploymentOrder, hasWorkflowDeploymentSequence, hasLegacyWorkflowDeploymentSequence } = module.exports

const nodes = [
  { key: 'build-gateway', type: 'build', config: { service_id: 69 } },
  { key: 'deploy-gateway', type: 'yaml_deploy', config: { service_id: 69 }, depends_on: ['build-gateway'] },
  { key: 'build-server', type: 'build', config: { service_id: 47 } },
  { key: 'deploy-server', type: 'yaml_deploy', config: { service_id: 47 }, depends_on: ['build-server'] }
]

test('server can be moved before gateway without changing saved source nodes', () => {
  assert.deepEqual(Array.from(workflowDeploymentOrder(nodes)), [69, 47])
  assert.equal(hasWorkflowDeploymentSequence(nodes), false)
  assert.equal(hasLegacyWorkflowDeploymentSequence(nodes), false)
  const ordered = withWorkflowDeploymentOrder(nodes, [47, 69])
  assert.equal(hasWorkflowDeploymentSequence(ordered), true)
  assert.equal(hasLegacyWorkflowDeploymentSequence(ordered), false)
  assert.deepEqual(Array.from(workflowDeploymentOrder(ordered)), [47, 69])
  assert.deepEqual(Array.from(ordered[1].depends_on), ['build-gateway'])
  assert.deepEqual(Array.from(ordered[3].depends_on), ['build-server'])
  assert.equal(ordered[1].config.start_order, 2)
  assert.equal(ordered[3].config.start_order, 1)
  assert.deepEqual(nodes[1].depends_on, ['build-gateway'])
  const reversed = withWorkflowDeploymentOrder(ordered, [69, 47])
  assert.deepEqual(Array.from(reversed[1].depends_on), ['build-gateway'])
  assert.deepEqual(Array.from(reversed[3].depends_on), ['build-server'])
  assert.equal(reversed[1].config.start_order, 1)
  assert.equal(reversed[3].config.start_order, 2)
})

test('legacy success edge is identified and removed on conversion', () => {
  const legacy = nodes.map((node) => ({ ...node, depends_on: [...(node.depends_on || [])] }))
  legacy[1].depends_on.push('deploy-server')
  assert.equal(hasLegacyWorkflowDeploymentSequence(legacy), true)
  const converted = withWorkflowDeploymentOrder(legacy, [47, 69])
  assert.equal(hasWorkflowDeploymentSequence(converted), true)
  assert.deepEqual(Array.from(converted[1].depends_on), ['build-gateway'])
})

test('unsafe cross-service build dependency and missing deploy cannot be reordered', () => {
  const crossBuild = nodes.map((node) => ({ ...node, depends_on: [...(node.depends_on || [])] }))
  crossBuild[2].depends_on = ['deploy-gateway']
  assert.deepEqual(Array.from(workflowDeploymentOrder(crossBuild)), [])
  assert.deepEqual(Array.from(workflowDeploymentOrder(nodes.slice(0, 3))), [])
  assert.throws(() => withWorkflowDeploymentOrder(crossBuild, [47, 69]))
})
