import type { LogQueryMode } from '@/api/manager/log'

type LogSourceType = 'container' | 'system'

type QueryCapability = {
  platform: boolean
  loki: boolean
  es: boolean
}

export interface LogSearchParams {
  sourceType: LogSourceType
  clusterUuid: string
  namespaceName?: string
  projectUuid?: string
  podName?: string
  containerName?: string
  host?: string
  podIp?: string
  level?: string
  sourceTypeHint?: string
  message?: string
  interval?: string
  from: number
  to: number
  maxLines: number
  queryMode?: LogQueryMode
}

type LabelFilter = {
  key: string
  operator: string
  value: string
}

const lokiSelectorKeys = new Set([
  'app',
  'job',
  'level',
  'log_type',
  'namespace_name',
  'node_name',
  'source',
  'source_type'
])

export function normalizeQueryModeByCapability(
  mode: LogQueryMode,
  capability: QueryCapability
): LogQueryMode {
  if (mode === 'loki' && !capability.loki) return 'platform'
  if (mode === 'es' && !capability.es) return 'platform'
  return mode
}

export function buildLogCodeText(
  params: LogSearchParams,
  capability: QueryCapability,
  labels: LabelFilter[] = []
) {
  const queryMode = params.queryMode
    ? normalizeQueryModeByCapability(params.queryMode, capability)
    : 'platform'

  const labelParts: string[] = []
  if (params.sourceType) labelParts.push(`log_type="${params.sourceType}"`)
  if (params.namespaceName) labelParts.push(`namespace_name="${params.namespaceName}"`)
  if (params.host) labelParts.push(`node_name="${params.host}"`)
  if (params.level) labelParts.push(`level="${params.level}"`)
  const lokiJSONParts: string[] = []
  if (params.clusterUuid)
    lokiJSONParts.push(formatLokiFilter('cluster_uuid', '=', params.clusterUuid))
  if (params.sourceType === 'container' && params.projectUuid) {
    lokiJSONParts.push(formatLokiFilter('project_uuid', '=', params.projectUuid))
  }
  if (params.podName) lokiJSONParts.push(formatLokiFilter('pod_name', '=', params.podName))
  if (params.containerName)
    lokiJSONParts.push(formatLokiFilter('container_name', '=', params.containerName))
  if (params.podIp) lokiJSONParts.push(formatLokiFilter('pod_ip', '=', params.podIp))
  if (labels.length > 0) {
    for (const item of labels) {
      const key = item.key?.trim()
      const value = item.value?.trim()
      if (!key || !value) continue
      const op = normalizeLabelOperator(item.operator)
      const target = formatLokiFilter(key, op, value)
      if (lokiSelectorKeys.has(key)) {
        labelParts.push(target)
      } else {
        lokiJSONParts.push(target)
      }
    }
  }

  const selector = labelParts.length > 0 ? `{${labelParts.join(', ')}}` : '{}'
  const lokiJSONExpr = lokiJSONParts.length > 0 ? ` | json | ${lokiJSONParts.join(' | ')}` : ''
  const message = params.message ? ` |= "${escapeLokiString(params.message)}"` : ''

  if (queryMode === 'loki') {
    return `${selector}${lokiJSONExpr}${message}`.trim()
  }

  const terms: string[] = []
  if (params.clusterUuid) terms.push(`cluster_uuid:${formatESTerm(params.clusterUuid)}`)
  if (params.sourceType) terms.push(`log_type:${formatESTerm(params.sourceType)}`)
  if (params.sourceType === 'container' && params.projectUuid)
    terms.push(`project_uuid:${formatESTerm(params.projectUuid)}`)
  if (params.message) terms.push(`message:${formatESTerm(params.message)}`)
  if (params.namespaceName) terms.push(`namespace_name:${formatESTerm(params.namespaceName)}`)
  if (params.podName) terms.push(`pod_name:${formatESTerm(params.podName)}`)
  if (params.containerName) terms.push(`container_name:${formatESTerm(params.containerName)}`)
  if (params.podIp) terms.push(`pod_ip:${formatESTerm(params.podIp)}`)
  if (params.host) terms.push(`node_name:${formatESTerm(params.host)}`)
  if (params.level) terms.push(`level:${formatESTerm(params.level)}`)

  if (labels.length > 0) {
    for (const item of labels) {
      const key = item.key?.trim()
      const value = item.value?.trim()
      if (!key || !value) continue

      const quoted = `${key}:${formatESTerm(value)}`
      const regex = `${key}:/${escapeESRegex(value)}/`
      switch (normalizeLabelOperator(item.operator)) {
        case '=':
          terms.push(quoted)
          break
        case '!=':
          terms.push(`NOT ${quoted}`)
          break
        case '=~':
          terms.push(regex)
          break
        case '!~':
          terms.push(`NOT ${regex}`)
          break
      }
    }
  }

  return terms.join(' AND ')
}

export function buildLogQueryRequest(params: LogSearchParams, capability: QueryCapability) {
  const queryMode = params.queryMode
    ? normalizeQueryModeByCapability(params.queryMode, capability)
    : undefined

  const request: Record<string, any> = {
    sourceType: params.sourceType,
    clusterUuid: params.clusterUuid,
    start: params.from,
    end: params.to,
    limit: params.maxLines
  }

  if (params.projectUuid) request.projectUuid = params.projectUuid
  if (params.namespaceName) request.namespace = params.namespaceName
  if (params.podName) request.podName = params.podName
  if (params.containerName) request.containerName = params.containerName
  if (params.podIp) request.podIp = params.podIp
  if (params.host) request.hosts = [params.host]
  if (params.level) request.level = params.level
  if (params.message) request.message = params.message
  if (queryMode) request.queryMode = queryMode

  return request
}

function normalizeLabelOperator(operator: string) {
  switch (operator) {
    case '=':
    case '!=':
    case '=~':
    case '!~':
      return operator
    default:
      return '='
  }
}

function escapeLokiString(value: string) {
  return value.replace(/\\/g, '\\\\').replace(/"/g, '\\"')
}

function formatLokiFilter(key: string, operator: string, value: string) {
  const escaped = escapeLokiString(value)
  if (value.includes('*')) {
    const regexOperator = operator === '!=' || operator === '!~' ? '!~' : '=~'
    return `${key}${regexOperator}"${escaped.replace(/\*/g, '.*')}"`
  }
  return `${key}${operator}"${escaped}"`
}

function quoteESTerm(value: string) {
  return `"${value.replace(/\\/g, '\\\\').replace(/"/g, '\\"')}"`
}

function formatESTerm(value: string) {
  if (value.includes('*') || value.includes('?')) {
    return value.replace(/\\/g, '\\\\').replace(/"/g, '\\"').replace(/ /g, '\\ ')
  }
  return quoteESTerm(value)
}

function escapeESRegex(value: string) {
  return value.replace(/\\/g, '\\\\').replace(/\//g, '\\/')
}
