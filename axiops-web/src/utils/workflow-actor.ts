type WorkflowActor = { trigger_user_id?: number; trigger_username?: string }

export function workflowActorLabel(run?: WorkflowActor | null): string {
  if (!run) return '—'
  if (!run.trigger_user_id) return '系统'
  return run.trigger_username?.trim() || `账号 #${run.trigger_user_id}`
}
