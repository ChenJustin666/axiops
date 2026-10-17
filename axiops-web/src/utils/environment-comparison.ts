import type { ServiceEnvironmentConfig } from '@/api/axiops/cicd'
import { configValues } from './service-environment-config'

export function environmentComparison(
  left: ServiceEnvironmentConfig,
  right: ServiceEnvironmentConfig
) {
  const values = (config: ServiceEnvironmentConfig) => {
    const result: Record<string, { value: string; origin: string }> = {}
    for (const [key, item] of Object.entries(configValues(config))) {
      result[`变量 / ${key}`] = {
        value: item.secret ? '凭据引用（不显示正文）' : String(item.value ?? ''),
        origin: item.origin
      }
    }
    result['模板版本'] = {
      value: `#${config.revision_id}`,
      origin: config.bound ? '已加入' : '尚未加入'
    }
    const assets = config.assets
    result['环境 YAML'] = {
      value: config.environment_yaml_revision_id
        ? `独立 YAML #${config.environment_yaml_revision_id}`
        : `公共定义 #${config.revision_id}`,
      origin: config.environment_yaml_revision_id ? '当前环境独立定义' : '公共定义'
    }
    for (const [key, label] of Object.entries({
      cpu_request: 'CPU 申请',
      cpu_limit: 'CPU 上限',
      memory_request: '内存申请',
      memory_limit: '内存上限'
    })) {
      const value = assets?.resources?.[key as keyof NonNullable<typeof assets.resources>]
      result[`资源 / ${label}`] = {
        value: value || '继承 YAML',
        origin: value ? '当前环境覆盖' : '基础定义'
      }
    }
    for (const file of assets?.files || [])
      result[`文件 / ${file.name}`] = {
        value: `${file.mount_path}\n${file.existing_config_map ? `ConfigMap: ${file.existing_config_map} / ${file.filename}` : file.content}`,
        origin: '环境独立配置'
      }
    for (const secret of assets?.secrets || [])
      result[`凭据 / ${secret.name}`] = {
        value: `${secret.existing_secret ? `Secret: ${secret.existing_secret}` : `凭据 #${secret.credential_id} / ${secret.credential_key}`} → ${secret.env_name || secret.mount_path}`,
        origin: '仅比较引用'
      }
    for (const route of assets?.routes || [])
      result[`域名 / ${route.name}`] = {
        value: `${route.host}${route.path} → ${route.service}:${route.port}\nClass: ${route.class || '默认'} / TLS: ${route.tls_secret || '无'}`,
        origin: '环境独立配置'
      }
    for (const volume of assets?.volumes || [])
      result[`存储 / ${volume.name}`] = {
        value: `${volume.existing_claim || `${volume.capacity} / ${volume.storage_class} / ${volume.access_mode}`} → ${volume.mount_path}${volume.read_only ? '（只读）' : ''}`,
        origin: '环境独立配置'
      }
    return result
  }
  const a = values(left),
    b = values(right)
  return [...new Set([...Object.keys(a), ...Object.keys(b)])].sort().map((key) => ({
    key,
    before: a[key]?.value ?? '未配置',
    after: b[key]?.value ?? '未配置',
    beforeOrigin: a[key]?.origin || '',
    afterOrigin: b[key]?.origin || '',
    changed: a[key]?.value !== b[key]?.value
  }))
}
