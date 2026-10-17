import { ref } from 'vue'
import type { AccessPolicy } from '@/api/axiops/access'

// No API/store imports here: authentication can clear this state without an
// HTTP -> user store -> API circular dependency.
export function createAccessState(now = Date.now) {
  const policies = ref<AccessPolicy[]>([])
  const loaded = ref(false)
  let generation = 0
  let loadedAt = 0
  let pending: Promise<void> | undefined
  const reset = () => {
    generation++
    policies.value = []
    loaded.value = false
    loadedAt = 0
    pending = undefined
  }
  const load = (fetchPolicies: () => Promise<AccessPolicy[]>, force = false): Promise<void> => {
    if (!force && pending) return pending
    if (!force && loaded.value && now() - loadedAt < 60_000) return Promise.resolve()
    reset()
    const revision = generation
    pending = (async () => {
      try {
        const result = await Promise.resolve().then(fetchPolicies)
        if (revision !== generation) return
        policies.value = result || []
        loaded.value = true
        loadedAt = now()
      } catch (error) {
        if (revision !== generation) return
        // Failed refresh must never preserve an old administrator policy.
        policies.value = []
        loaded.value = false
        throw error
      } finally {
        if (revision === generation) pending = undefined
      }
    })()
    return pending
  }
  return { policies, loaded, reset, load }
}

export const axiopsAccessState = createAccessState()
export const resetAxiOpsAccess = () => axiopsAccessState.reset()
