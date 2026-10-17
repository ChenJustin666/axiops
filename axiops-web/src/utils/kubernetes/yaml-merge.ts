/**
 * YAML 合并工具
 * 用于保护 Kubernetes 资源的元数据完整性
 * 在编辑资源时，只更新用户可编辑的字段，保留系统管理的字段
 */

import * as yaml from 'js-yaml'
import { cloneDeep, set, get, unset } from 'lodash-es'

/** 原始资源包装对象 */
export interface RawResourceWrapper<T = any> {
  /** 原始 YAML 字符串 */
  rawYaml: string
  /** 解析后的原始对象（冻结，不可修改） */
  rawObject: T
  /** 用户可编辑的字段路径列表 */
  editableFields: string[]
}

/** 系统保留字段，表单不应覆盖这些字段 */
const DEFAULT_PRESERVE_FIELDS = [
  'metadata.uid',
  'metadata.resourceVersion',
  'metadata.generation',
  'metadata.creationTimestamp',
  'metadata.managedFields',
  'metadata.selfLink',
  'metadata.finalizers',
  'status'
]

/**
 * 解析 YAML 字符串并创建资源包装对象
 * @param yamlStr - YAML 字符串
 * @param editableFields - 用户可编辑的字段路径列表
 * @returns 资源包装对象
 */
export function parseAndWrap<T = any>(
  yamlStr: string,
  editableFields: string[]
): RawResourceWrapper<T> {
  const parsed = yaml.load(yamlStr) as T
  return {
    rawYaml: yamlStr,
    rawObject: Object.freeze(cloneDeep(parsed)),
    editableFields
  }
}

/**
 * 将表单变更合并到原始 YAML
 * 保留系统管理的字段，只更新用户可编辑的字段
 * @param wrapper - 原始资源包装对象
 * @param formChanges - 表单变更（路径 -> 值 的映射）
 * @param preserveFields - 需要保留的字段路径列表
 * @returns 合并后的 YAML 字符串
 */
export function mergeFormToYaml<T = any>(
  wrapper: RawResourceWrapper<T>,
  formChanges: Record<string, any>,
  preserveFields: string[] = DEFAULT_PRESERVE_FIELDS
): string {
  const merged = cloneDeep(wrapper.rawObject) as any

  for (const [path, value] of Object.entries(formChanges)) {
    // 检查是否为保留字段
    const isPreserved = preserveFields.some((pf) => path === pf || path.startsWith(pf + '.'))

    if (isPreserved) {
      continue
    }

    // 处理值
    if (value === undefined || value === null) {
      unset(merged, path)
    } else {
      set(merged, path, value)
    }
  }

  return yaml.dump(merged, {
    indent: 2,
    lineWidth: -1,
    noRefs: true,
    sortKeys: false
  })
}

/**
 * 从原始对象提取表单数据
 * @param wrapper - 原始资源包装对象
 * @param fieldMapping - 字段映射（表单字段 -> 资源路径）
 * @returns 表单数据对象
 */
export function extractFormData<T = any>(
  wrapper: RawResourceWrapper<T>,
  fieldMapping: Record<string, string>
): Record<string, any> {
  const result: Record<string, any> = {}

  for (const [formField, resourcePath] of Object.entries(fieldMapping)) {
    result[formField] = get(wrapper.rawObject, resourcePath)
  }

  return result
}

/**
 * 构建表单变更对象
 * 用于将表单数据转换为 mergeFormToYaml 所需的格式
 * @param formData - 表单数据
 * @param fieldMapping - 字段映射（表单字段 -> 资源路径）
 * @returns 变更对象
 */
export function buildFormChanges(
  formData: Record<string, any>,
  fieldMapping: Record<string, string>
): Record<string, any> {
  const changes: Record<string, any> = {}

  for (const [formField, resourcePath] of Object.entries(fieldMapping)) {
    const value = formData[formField]
    changes[resourcePath] = value
  }

  return changes
}

/**
 * 检查是否为系统保留字段
 * @param path - 字段路径
 * @param preserveFields - 保留字段列表
 * @returns 是否为保留字段
 */
export function isPreservedField(
  path: string,
  preserveFields: string[] = DEFAULT_PRESERVE_FIELDS
): boolean {
  return preserveFields.some((pf) => path === pf || path.startsWith(pf + '.'))
}

/**
 * 获取默认保留字段列表
 * @returns 默认保留字段列表
 */
export function getDefaultPreserveFields(): string[] {
  return [...DEFAULT_PRESERVE_FIELDS]
}

/**
 * 深度合并对象
 * 用于合并嵌套的配置对象
 * @param target - 目标对象
 * @param source - 源对象
 * @param preserveFields - 需要保留的字段
 * @returns 合并后的对象
 */
export function deepMergeWithPreserve<T extends Record<string, any>>(
  target: T,
  source: Partial<T>,
  preserveFields: string[] = DEFAULT_PRESERVE_FIELDS
): T {
  const result = cloneDeep(target)

  function merge(obj: any, src: any, currentPath: string = ''): any {
    if (!src || typeof src !== 'object') {
      return src
    }

    for (const key of Object.keys(src)) {
      const fullPath = currentPath ? `${currentPath}.${key}` : key

      // 检查是否为保留字段
      if (isPreservedField(fullPath, preserveFields)) {
        continue
      }

      const targetValue = obj[key]
      const sourceValue = src[key]

      if (
        sourceValue !== null &&
        typeof sourceValue === 'object' &&
        !Array.isArray(sourceValue) &&
        targetValue !== null &&
        typeof targetValue === 'object' &&
        !Array.isArray(targetValue)
      ) {
        obj[key] = merge(targetValue, sourceValue, fullPath)
      } else {
        obj[key] = sourceValue
      }
    }

    return obj
  }

  return merge(result, source)
}
