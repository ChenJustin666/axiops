import type { ServiceEnvironmentConfig } from '@/api/axiops/cicd'
import * as yaml from 'js-yaml'

export const hasConfigKey = (value: object, key: PropertyKey) =>
  Object.prototype.hasOwnProperty.call(value, key)

export function configValues(config: ServiceEnvironmentConfig) {
  const values: Record<string, { value: string; origin: string; secret: boolean }> =
    Object.create(null)
  if (config.format === 'helm_chart') {
    const chart = JSON.parse(config.definition) as {
      files: Array<{ path: string; content: string }>
    }
    const file = chart.files.find((item) => item.path === 'values.yaml')
    const flatten = (value: unknown, path = '') => {
      if (value && typeof value === 'object' && !Array.isArray(value)) {
        for (const [key, child] of Object.entries(value))
          flatten(child, path ? `${path}.${key}` : key)
      } else if (path && ['string', 'number', 'boolean'].includes(typeof value)) {
        values[path] = { value: String(value), origin: 'Chart 默认', secret: false }
      }
    }
    if (file) flatten(yaml.load(file.content))
  }
  if (config.format !== 'helm_chart') {
    const match = config.definition.match(/^# AxiOps-Variables:\s*(.+)$/m)
    if (match) {
      const defaults = JSON.parse(match[1]) as Array<{ key: string; value: string }>
      for (const item of defaults)
        values[item.key] = { value: item.value, origin: '模板默认', secret: false }
    }
    for (const match of config.definition.matchAll(
      /\{\{\s*\.?([A-Za-z][A-Za-z0-9_]*)\s*\}\}|\$([A-Za-z][A-Za-z0-9_-]*)\$/g
    )) {
      const key = match[1] || match[2]
      if (
        !['Product', 'Service', 'Namespace', 'EnvName', 'ClusterName', 'Image'].includes(key) &&
        !key.endsWith('-image')
      )
        values[key] ||= { value: '', origin: '缺少值', secret: false }
    }
  }
  for (const item of config.variables || [])
    values[item.key] = {
      value: item.value,
      origin: item.origin === 'override' ? '当前环境覆盖' : '继承变量',
      secret: item.secret
    }
  if (config.replicas !== null)
    values.replicas = { value: String(config.replicas), origin: '当前环境覆盖', secret: false }
  return values
}

export function localOverrides(config: ServiceEnvironmentConfig): Record<string, string> {
  const result: Record<string, string> = Object.create(null)
  for (const item of config.variables || [])
    if (item.origin === 'override' && !item.secret) result[item.key] = item.value
  if (config.replicas !== null) result.replicas = String(config.replicas)
  return result
}

export function variablePatch(before: Record<string, string>, after: Record<string, string>) {
  const patch: Record<string, string | null> = Object.create(null)
  for (const key of new Set([...Object.keys(before), ...Object.keys(after)])) {
    if (before[key] !== after[key]) patch[key] = hasConfigKey(after, key) ? after[key] : null
  }
  return patch
}

// Bound concurrency keeps large projects from flooding the API/browser pool.
export async function mapLimited<T, R>(
  items: T[],
  fn: (item: T) => Promise<R>,
  limit = 4
): Promise<PromiseSettledResult<R>[]> {
  const result: PromiseSettledResult<R>[] = new Array(items.length)
  let next = 0
  await Promise.all(
    Array.from({ length: Math.min(limit, items.length) }, async () => {
      while (next < items.length) {
        const index = next++
        try {
          result[index] = { status: 'fulfilled', value: await fn(items[index]) }
        } catch (reason) {
          result[index] = { status: 'rejected', reason }
        }
      }
    })
  )
  return result
}
