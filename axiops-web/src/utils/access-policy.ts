import type { AccessPolicy } from '@/api/axiops/access'

// Mirrors internal/shared/authz.Request. Callers supply server-resolved resource
// identities; a missing target never means "any resource".
export interface AccessScope {
  type?: string
  id?: number
  project_id?: number
  service_id?: number
  build_id?: number
  workflow_id?: number
  cluster_id?: number
  environment_id?: number
  environment?: string
}

export function canAccess(
  policies: readonly AccessPolicy[],
  resource: string,
  action: string,
  scope: AccessScope = {}
): boolean {
  if (!resource || !action) return false
  const ids: Record<string, number | undefined> = {
    project: scope.project_id,
    service: scope.service_id,
    build: scope.build_id,
    workflow: scope.workflow_id,
    cluster: scope.cluster_id,
    environment: scope.environment_id
  }
  if (scope.type && Object.prototype.hasOwnProperty.call(ids, scope.type) && scope.id !== undefined)
    ids[scope.type] = scope.id
  let allowed = false
  for (const policy of policies) {
    if (policy.resource !== '*' && policy.resource !== resource) continue
    if (
      policy.action !== '*' &&
      policy.action !== action &&
      !(action === 'view' && policy.action === 'read') &&
      !(['create', 'update', 'delete'].includes(action) && policy.action === 'write')
    )
      continue
    if (
      policy.environment &&
      policy.environment !== '*' &&
      policy.environment !== scope.environment
    )
      continue
    if (policy.scope_type && policy.scope_type !== 'platform') {
      if (!Object.prototype.hasOwnProperty.call(ids, policy.scope_type)) continue
      const id = ids[policy.scope_type]
      if (!id || policy.scope_id <= 0 || policy.scope_id !== id) continue
    }
    const effect = policy.effect.toLowerCase()
    if (effect === 'deny') return false
    if (effect === 'allow') allowed = true
  }
  return allowed
}
