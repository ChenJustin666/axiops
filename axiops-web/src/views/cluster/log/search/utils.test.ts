import assert from 'node:assert/strict'
import { buildLogCodeText, buildLogQueryRequest, normalizeQueryModeByCapability } from './utils'

const capability = { platform: true, loki: false, es: true }

assert.equal(normalizeQueryModeByCapability('platform', capability), 'platform')
assert.equal(normalizeQueryModeByCapability('loki', capability), 'platform')
assert.equal(normalizeQueryModeByCapability('es', capability), 'es')

const containerRequest = buildLogQueryRequest(
  {
    sourceType: 'container',
    clusterUuid: 'cluster-1',
    namespaceName: 'team-a',
    projectUuid: 'project-1',
    podName: 'web-7c8f9d',
    containerName: 'app',
    podIp: '10.0.0.2',
    host: 'node-a',
    level: 'error',
    message: 'error',
    interval: '5m',
    from: 1710000000,
    to: 1710000300,
    maxLines: 200,
    queryMode: 'platform'
  },
  capability
)

assert.deepEqual(containerRequest, {
  sourceType: 'container',
  clusterUuid: 'cluster-1',
  projectUuid: 'project-1',
  namespace: 'team-a',
  podName: 'web-7c8f9d',
  containerName: 'app',
  podIp: '10.0.0.2',
  hosts: ['node-a'],
  level: 'error',
  message: 'error',
  start: 1710000000,
  end: 1710000300,
  limit: 200,
  queryMode: 'platform'
})

const lokiCapability = { platform: true, loki: true, es: true }

const codeText = buildLogCodeText(
  {
    sourceType: 'container',
    clusterUuid: 'cluster-1',
    namespaceName: 'team-a',
    podName: 'web-7c8f9d',
    containerName: 'app',
    podIp: '10.0.0.2',
    host: 'node-a',
    level: 'error',
    message: 'error',
    from: 1710000000,
    to: 1710000300,
    maxLines: 200,
    queryMode: 'loki'
  },
  lokiCapability
)

assert.equal(
  codeText,
  '{log_type="container", namespace_name="team-a", node_name="node-a", level="error"} | json | cluster_uuid="cluster-1" | pod_name="web-7c8f9d" | container_name="app" | pod_ip="10.0.0.2" |= "error"'
)

const codeTextWithLabels = buildLogCodeText(
  {
    sourceType: 'container',
    clusterUuid: 'cluster-1',
    namespaceName: 'team-a',
    podName: 'web-7c8f9d',
    containerName: 'app',
    podIp: '10.0.0.2',
    host: 'node-a',
    level: 'error',
    message: 'error',
    from: 1710000000,
    to: 1710000300,
    maxLines: 200,
    queryMode: 'loki'
  },
  lokiCapability,
  [{ key: 'app', operator: '=', value: 'web' }]
)

assert.equal(
  codeTextWithLabels,
  '{log_type="container", namespace_name="team-a", node_name="node-a", level="error", app="web"} | json | cluster_uuid="cluster-1" | pod_name="web-7c8f9d" | container_name="app" | pod_ip="10.0.0.2" |= "error"'
)

const systemRequest = buildLogQueryRequest(
  {
    sourceType: 'system',
    clusterUuid: 'cluster-2',
    host: 'node-b',
    level: 'warning',
    message: 'panic',
    interval: '1h',
    from: 1710000000,
    to: 1710003600,
    maxLines: 50,
    queryMode: 'es'
  },
  capability
)

assert.deepEqual(systemRequest, {
  sourceType: 'system',
  clusterUuid: 'cluster-2',
  hosts: ['node-b'],
  level: 'warning',
  message: 'panic',
  start: 1710000000,
  end: 1710003600,
  limit: 50,
  queryMode: 'es'
})

const esCodeText = buildLogCodeText(
  {
    sourceType: 'container',
    clusterUuid: 'cluster-1',
    namespaceName: 'pay-uat',
    projectUuid: 'project-1',
    podName: 'dep-web-test*',
    from: 1710000000,
    to: 1710003600,
    maxLines: 50,
    queryMode: 'es'
  },
  capability
)

assert.equal(
  esCodeText,
  'cluster_uuid:"cluster-1" AND log_type:"container" AND project_uuid:"project-1" AND namespace_name:"pay-uat" AND pod_name:dep-web-test*'
)

console.log('utils.test.ts passed')
