import type { AxiOpsBuildConfig, AxiOpsService, WorkflowNode } from '@/api/axiops/cicd'
import { workflowDeploymentOrder } from './workflow-service-order'

export function isBuildConfigMissing(error: unknown): boolean {
  const value = error as { code?: number; businessCode?: number } | null
  return value?.businessCode === 30001 || value?.code === 30001
}

export function errorMessage(error: unknown, fallback: string): string {
  return error instanceof Error
    ? error.message || fallback
    : (error as { message?: string } | null)?.message || fallback
}

// A complete form is not a guarantee of network, credential or cluster availability.
export function buildConfigurationIssues(
  service: AxiOpsService,
  config: AxiOpsBuildConfig
): string[] {
  const issues: string[] = []
  if (!config?.id) return ['尚未保存构建配置']
  if (config.build_mode !== 'artifact') {
    if (!service.code_source_id || !service.repo_url?.trim() || !service.repo_branch?.trim())
      issues.push('缺少代码源、仓库或分支')
    if (!service.registry_id || !config.image_repo?.trim()) issues.push('缺少镜像仓库或产物路径')
  }
  if (
    config.build_mode === 'jar_image' &&
    (!config.build_image?.trim() || !config.build_command?.trim())
  )
    issues.push('缺少 JAR 编译镜像或命令')
  if (config.resource_memory?.trim() && /^\d+(\.\d+)?$/.test(config.resource_memory.trim()))
    issues.push('构建内存缺少单位，如 2Gi')
  if (config.cache_enabled && config.cache_strategy === 'pvc' && !config.cache_pvc_name?.trim())
    issues.push('缺少缓存 PVC 名称')
  return issues
}

export function workflowServiceSelection(nodes: WorkflowNode[]): number[] {
  return workflowDeploymentOrder(nodes)
}

export function supportsServiceOverride(nodes: WorkflowNode[]): boolean {
  const deploys = nodes.filter((node) =>
    ['yaml_deploy', 'helm_deploy', 'host_deploy'].includes(node.type)
  )
  const serviceIds = new Set(
    nodes
      .filter((node) =>
        ['build', 'test', 'yaml_deploy', 'helm_deploy', 'host_deploy'].includes(node.type)
      )
      .map((node) => Number(node.config?.service_id || 0))
      .filter(Boolean)
  )
  return (
    (deploys.length === 1 && serviceIds.size <= 1) || workflowServiceSelection(nodes).length > 1
  )
}
