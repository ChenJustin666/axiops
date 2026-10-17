import { computed } from 'vue'
import { getCurrentAccessApi } from '@/api/axiops/access'
import { canAccess, type AccessScope } from '@/utils/access-policy'
import { axiopsAccessState } from './axiops-access-state'

export function useAxiOpsAccess() {
  const { policies, loaded } = axiopsAccessState
  const load = (force = false) => axiopsAccessState.load(getCurrentAccessApi, force)
  const can = (resource: string, action: string, scope: AccessScope = {}) =>
    loaded.value && canAccess(policies.value, resource, action, scope)

  return {
    policies: computed(() => policies.value),
    loaded: computed(() => loaded.value),
    load,
    can
  }
}
