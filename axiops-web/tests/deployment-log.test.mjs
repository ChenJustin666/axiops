import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import vm from 'node:vm'
import ts from 'typescript'

const module = { exports: {} }
vm.runInNewContext(ts.transpileModule(readFileSync(new URL('../src/utils/deployment-log.ts', import.meta.url), 'utf8'), {
  compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 }
}).outputText, { module, exports: module.exports })
const { summarizeDeploymentLog: summary } = module.exports
const snapshot = (restart, second) => `[2026-10-02T02:00:${second}Z]\n[check] cluster=2 namespace=risk-qa
[rollout] Deployment/api ready=false generation=7 observed=7 desired=1 replicas=1 updated=1 readyReplicas=0 available=0 scheduled=0
[condition] Deployment/api Available=False reason=MinimumReplicasUnavailable not available
[condition] Deployment/api Progressing=True reason=ReplicaSetUpdated progressing
[pod] api-1 phase=Running container=api ready=false restarts=${restart} waiting=CrashLoopBackOff back-off 5m0s previous=Error exitCode=1
[pod] api-1 Ready=False reason=ContainersNotReady not ready
[event] Pod/api-1 2026-10-02T02:00:${second}Z Pulled: very-long-image@sha256:abcdef
[event] Pod/api-1 2026-10-02T02:00:${second}Z Unhealthy: Readiness probe failed: Get "http://10.0.0.2:8001/health": connect: connection refused
[resource] Service/api submitted; no rollout check for this kind
[waiting] waiting for rollout; transient pod errors will be retried until the deployment deadline`

test('collapses repeated snapshots while preserving actionable failures and authoritative restart counts', () => {
  const log = `[2026-10-02T02:00:00Z]\n[prepare] YAML rendered and validated; deployment snapshot saved\n[apply] cluster=2 namespace=risk-qa; manifest validated\n[apply] Deployment/api submitted\n[apply] Service/api submitted\n` +
    Array.from({ length: 30 }, (_, i) => snapshot(i, String(i).padStart(2, '0'))).join('\n') +
    '\n[failed] Deployment/api has timed out progressing.'
  const result = summary(log, 'failed')
  assert.ok(result.split('\n').length < 12)
  for (const evidence of ['risk-qa', 'Deployment/api', 'Service/api', 'CrashLoopBackOff', '退出码 1', '重启 29 次', ':8001/health', 'connection refused', 'timed out progressing']) assert.ok(result.includes(evidence), evidence)
  assert.equal(result.match(/CrashLoopBackOff/g).length, 1)
  assert.equal(result.match(/connection refused/g).length, 1)
  assert.doesNotMatch(result, /MinimumReplicasUnavailable|very-long-image|scheduled=|<nil>|\[等待\]/)
  assert.match(result.split('\n').at(-1), /失败.*timed out/)
  assert.match(log, /MinimumReplicasUnavailable/)
})

test('does not report running or zero replica workloads as business success', () => {
  assert.match(summary(snapshot(1, '01'), 'running'), /等待.*继续检查/)
  assert.doesNotMatch(summary(snapshot(1, '01'), 'running'), /\[成功\]/)
  const zero = '[rollout] Deployment/api ready=true desired=0 available=0 updated=0\n[succeeded] Kubernetes rollout checks passed. Application health follows configured readiness probes; external traffic and business checks are not included.'
  assert.match(summary(zero, 'succeeded'), /已缩容为 0，不代表对外可用/)
  assert.match(summary(zero, 'succeeded'), /未执行外部访问或业务检查/)
  assert.match(summary('[succeeded] resources submitted; manifest has no supported rollout target; application health not verified', 'succeeded'), /应用健康未验证/)
})

test('keeps controller lag, scheduling, unknown errors, and diagnostic gaps', () => {
  const result = summary('[rollout] Deployment/api ready=false generation=8 observed=7 desired=1 available=1 updated=1\n[pod] api-1 PodScheduled=False reason=Unschedulable no matching node\n[event] Pod/api-1 2026-10-02T02:00:00Z FailedScheduling: Insufficient cpu\n[diagnostics] unavailable: forbidden\n[system] earlier checks truncated\nunexpected backend error', 'running')
  for (const token of ['等待控制器确认新版本', 'Unschedulable', 'Insufficient cpu', 'forbidden', '保存上限', 'unexpected backend error']) assert.ok(result.includes(token), token)
})

test('handles multiple containers, recovery, Job and DaemonSet counters', () => {
  const result = summary('[pod] api-1 phase=Running container=api ready=false restarts=2 terminated=Error exitCode=1; container=sidecar ready=false restarts=1 waiting=ImagePullBackOff\n[pod] api-1 phase=Running container=api ready=true restarts=2 previous=Error exitCode=1; container=sidecar ready=true restarts=1\n[rollout] DaemonSet/agent ready=false desired=3 availableScheduled=2 updatedScheduled=3\n[rollout] Job/migrate ready=false succeeded=0 failed=1', 'running')
  for (const token of ['容器 api', '容器 sidecar', 'ImagePullBackOff', '退出码 1', '已就绪', '就绪 2/3', '失败 1']) assert.ok(result.includes(token), token)
})

test('legacy errors, cancelled runs and empty records stay truthful', () => {
  assert.equal(summary('', 'succeeded'), '')
  assert.match(summary('yaml contains undefined variables: port', 'failed'), /undefined variables: port/)
  assert.match(summary('[apply] Deployment/api submitted', 'cancelled'), /已取消/)
  assert.doesNotMatch(summary('[pod] api-1 phase=Pending container=api ready=false waiting=ContainerCreating <nil>', 'running'), /<nil>/)
  assert.match(summary('', 'failed', 'access denied'), /access denied/)
})

test('does not duplicate a truncated task error copy or parse a partial timestamp as an event', () => {
  const log = '[2026-10-02T02:00:00Z]\n[failed] Deployment/api rollout timed out\n[pod] api-1 phase=Running container=api ready=false waiting=CrashLoopBackOff previous=Error exitCode=1'
  const error = 'Deployment/api rollout timed out\n[event] Pod/api-1 2026-10-02T09:43: [truncated]'
  const result = summary(log, 'failed', error)
  assert.equal(result.match(/rollout timed out/g).length, 1)
  assert.doesNotMatch(result, /2026-10-02T09/)
  assert.match(summary('[event] Pod/api-1 2026-10-02T09:43: [truncated]', 'failed'), /部分事件信息已截断/)
})
