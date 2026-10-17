import * as yaml from 'js-yaml'

export type PendingRollback = {
  key: string
  snapshotId: number
  workflowId: number
  request: { origin_run_id: number; service_id: number; expected_token: string }
}

export function rollbackConfigChanges(before: string, after: string) {
  const changes: { path: string; before: string; after: string }[] = []
  const documents = (text: string): unknown => {
    try {
      const docs = yaml.loadAll(text).filter(Boolean) as any[]
      return Object.fromEntries(
        docs.map((doc, index) => [
          doc.kind && doc.metadata?.name ? `${doc.kind}/${doc.metadata.name}` : String(index),
          doc
        ])
      )
    } catch {
      return text
    }
  }
  const display = (value: unknown) =>
    value === undefined ? '（无）' : typeof value === 'string' ? value : JSON.stringify(value)
  const visit = (a: any, b: any, path: string) => {
    if (JSON.stringify(a) === JSON.stringify(b)) return
    if (a && b && typeof a === 'object' && typeof b === 'object') {
      for (const key of new Set([...Object.keys(a), ...Object.keys(b)]))
        visit(a[key], b[key], path ? `${path}.${key}` : key)
    } else changes.push({ path, before: display(a), after: display(b) })
  }
  visit(documents(before), documents(after), '')
  return changes
}

// Persist only operation identity, never YAML, credentials or form secrets.
export function loadPendingRollback(runId: number): PendingRollback | null {
  try {
    const value = JSON.parse(sessionStorage.getItem(`axiops:rollback:${runId}`) || 'null')
    return value?.request?.origin_run_id === runId &&
      value?.request?.service_id > 0 &&
      value?.request?.expected_token?.length === 64 &&
      value?.snapshotId > 0 &&
      typeof value?.key === 'string'
      ? value
      : null
  } catch {
    return null
  }
}

export function savePendingRollback(runId: number, value: PendingRollback | null) {
  try {
    if (value) sessionStorage.setItem(`axiops:rollback:${runId}`, JSON.stringify(value))
    else sessionStorage.removeItem(`axiops:rollback:${runId}`)
  } catch {
    /* Memory state still retains the key if storage is unavailable. */
  }
}

export function rollbackResultUncertain(error: {
  code?: unknown
  businessCode?: unknown
  requestUncertain?: boolean
}): boolean {
  if (error?.requestUncertain || error?.code === 408) return true
  const code = error?.code
  // HTTP 5xx, timeouts and network errors may arrive after the server committed.
  return !(
    typeof code === 'number' &&
    ((code >= 400 && code < 500) ||
      (code >= 10001 && code < 50001) ||
      code === 50004 ||
      code === 50005)
  )
}
