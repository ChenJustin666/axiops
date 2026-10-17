export const scopeOptions = [
  { label: '全部范围', value: '' },
  { label: '全局巡检', value: 'global' },
  { label: '单集群', value: 'cluster' }
]

export const scheduleOptions = [
  { label: '全部调度', value: '' },
  { label: '手动', value: 'manual' },
  { label: '定时', value: 'cron' }
]

export const enabledOptions = [
  { label: '全部状态', value: undefined },
  { label: '已启用', value: true },
  { label: '已禁用', value: false }
]

export const checkTypeOptions = [
  { label: '内置检查', value: 'builtin' },
  { label: 'PromQL', value: 'promql' },
  { label: 'SQL', value: 'sql' }
]

export const targetTypeOptions = [
  { label: '集群', value: 'cluster' },
  { label: '主机', value: 'host' },
  { label: 'Pod', value: 'pod' },
  { label: '工作负载', value: 'workload' },
  { label: '数据库', value: 'database' },
  { label: '中间件', value: 'middleware' },
  { label: 'Prometheus', value: 'prometheus' }
]

export const severityOptions = [
  { label: '严重', value: 'critical' },
  { label: '警告', value: 'warning' },
  { label: '提示', value: 'info' }
]

export const operatorOptions = [
  { label: '大于', value: '>' },
  { label: '大于等于', value: '>=' },
  { label: '小于', value: '<' },
  { label: '小于等于', value: '<=' },
  { label: '等于', value: '==' },
  { label: '不等于', value: '!=' }
]

export const recordStatusOptions = [
  { label: '全部状态', value: '' },
  { label: '执行中', value: 'running' },
  { label: '成功', value: 'success' },
  { label: '失败', value: 'failed' },
  { label: '部分成功', value: 'partial' }
]

export const healthLevelOptions = [
  { label: '全部健康度', value: '' },
  { label: '健康', value: 'healthy' },
  { label: '警告', value: 'warning' },
  { label: '严重', value: 'critical' },
  { label: '未知', value: 'unknown' }
]

export function scopeText(value?: string) {
  const map: Record<string, string> = {
    global: '全局巡检',
    cluster: '单集群'
  }
  return value ? map[value] || value : '-'
}

export function scheduleText(value?: string) {
  const map: Record<string, string> = {
    manual: '手动',
    cron: '定时'
  }
  return value ? map[value] || value : '-'
}

export function checkTypeText(value?: string) {
  const map: Record<string, string> = {
    builtin: '内置检查',
    promql: 'PromQL',
    sql: 'SQL'
  }
  return value ? map[value] || value : '-'
}

export function targetTypeText(value?: string) {
  const map: Record<string, string> = {
    cluster: '集群',
    host: '主机',
    node: '主机',
    pod: 'Pod',
    workload: '工作负载',
    database: '数据库',
    middleware: '中间件',
    etcd: 'etcd',
    prometheus: 'Prometheus'
  }
  return value ? map[value] || value : '-'
}

export function severityText(value?: string) {
  const map: Record<string, string> = {
    critical: '严重',
    warning: '警告',
    info: '提示'
  }
  return value ? map[value] || value : '-'
}

export function statusText(value?: string) {
  const map: Record<string, string> = {
    running: '执行中',
    success: '成功',
    failed: '失败',
    partial: '部分成功',
    warning: '警告',
    critical: '严重'
  }
  return value ? map[value] || value : '-'
}

export function healthText(value?: string) {
  const map: Record<string, string> = {
    healthy: '健康',
    warning: '警告',
    critical: '严重',
    unknown: '未知'
  }
  return value ? map[value] || value : '-'
}

export function severityTagType(value?: string) {
  if (value === 'critical') return 'danger'
  if (value === 'warning') return 'warning'
  if (value === 'info') return 'info'
  return 'info'
}

export function statusTagType(value?: string) {
  if (value === 'success') return 'success'
  if (value === 'running') return 'primary'
  if (value === 'failed' || value === 'critical') return 'danger'
  if (value === 'partial' || value === 'warning') return 'warning'
  return 'info'
}

export function healthTagType(value?: string) {
  if (value === 'healthy') return 'success'
  if (value === 'warning') return 'warning'
  if (value === 'critical') return 'danger'
  return 'info'
}

export function formatTime(value?: number) {
  if (!value) return '-'
  const timestamp = value < 1000000000000 ? value * 1000 : value
  return new Date(timestamp).toLocaleString('zh-CN', { hour12: false })
}

export function formatPercent(value: number, total: number) {
  if (!total) return '0%'
  return `${Math.round((value / total) * 100)}%`
}
