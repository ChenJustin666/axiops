/**
 * YAML 合并工具
 * 用于在编辑模式下保留原始 YAML 中表单未显示的字段（如 status、managedFields 等）
 */

import * as yaml from 'js-yaml'

/**
 * 深度合并两个对象
 * @param target 目标对象（原始数据）
 * @param source 源对象（表单数据）
 * @returns 合并后的对象
 */
export function deepMerge<T extends Record<string, any>>(target: T, source: Partial<T>): T {
  const result = { ...target }

  for (const key in source) {
    if (Object.prototype.hasOwnProperty.call(source, key)) {
      const sourceValue = source[key]
      const targetValue = target[key]

      if (sourceValue === undefined) {
        continue
      }

      if (sourceValue === null) {
        // null 值表示删除该字段
        delete (result as any)[key]
      } else if (
        typeof sourceValue === 'object' &&
        !Array.isArray(sourceValue) &&
        typeof targetValue === 'object' &&
        !Array.isArray(targetValue) &&
        targetValue !== null
      ) {
        // 递归合并对象
        ;(result as any)[key] = deepMerge(targetValue, sourceValue)
      } else {
        // 直接覆盖（包括数组）
        ;(result as any)[key] = sourceValue
      }
    }
  }

  return result
}

/**
 * 解析 YAML 字符串为对象
 * @param yamlStr YAML 字符串
 * @returns 解析后的对象，解析失败返回 null
 */
export function parseYaml<T = any>(yamlStr: string): T | null {
  try {
    return yaml.load(yamlStr) as T
  } catch (error) {
    console.error('YAML 解析失败:', error)
    return null
  }
}

/**
 * 将对象序列化为 YAML 字符串
 * @param obj 要序列化的对象
 * @param options 序列化选项
 * @returns YAML 字符串
 */
export function stringifyYaml(
  obj: any,
  options: yaml.DumpOptions = {
    indent: 2,
    lineWidth: -1,
    noRefs: true,
    sortKeys: false,
    // 使用 flow level:-1 确保所有对象都用块样式格式化
    // 这样可以避免单键对象被简化
    flowLevel: -1
  }
): string {
  const yamlStr = yaml.dump(obj, options)

  // 修复 NetworkPolicy 的 podSelector 和 namespaceSelector 格式
  // 确保 matchLabels 始终正确显示，不被 js-yaml 简化
  return fixSelectorFormat(yamlStr)
}

/**
 * 修复 YAML 中的选择器格式
 * 确保 podSelector 和 namespaceSelector 正确显示 matchLabels
 */
function fixSelectorFormat(yamlStr: string): string {
  // 匹配 podSelector 或 namespaceSelector 后面跟着行内格式的键值对
  // 例如: podSelector: app: nginx
  // 修复为: podSelector:\n    matchLabels:\n      app: nginx

  // 匹配各种选择器后面可能的行内格式
  const patterns = [
    // podSelector: key: value 或 namespaceSelector: key: value
    /(\s*)(podSelector|namespaceSelector):\s*([a-zA-Z0-9_.-]+):\s*(.+)$/gm
    // 处理多标签的行内格式 (可能已经被格式化为对象)
  ]

  let result = yamlStr

  // 首先处理已存在的对象格式但缺少 matchLabels 的情况
  // 例如: podSelector:\n      app: nginx\n      version: v1
  result = result.replace(
    /(\s+)(podSelector|namespaceSelector):\n((?:\s{2,}[a-z0-9_.-]+:\s*.+\n)+)/gi,
    (match, indent, selectorName, content) => {
      // 检查内容是否已经包含 matchLabels
      if (content.trim().startsWith('matchLabels:')) {
        return match
      }
      // 重新格式化，添加 matchLabels
      const labels = content
        .trim()
        .split('\n')
        .map((line: string) => {
          const [key, ...valueParts] = line.trim().split(':')
          const value = valueParts.join(':').trim()
          return `      ${key}: ${value}`
        })
        .join('\n')
      return `${indent}${selectorName}:\n${indent}  matchLabels:\n${labels}`
    }
  )

  return result
}

/**
 * 合并表单数据到原始 YAML 对象
 * 保留原始对象中表单未涉及的字段（如 status、managedFields 等）
 * @param originalYaml 原始 YAML 字符串
 * @param formObject 从表单构建的对象
 * @param preserveFields 需要从原始对象保留的字段路径
 * @returns 合并后的 YAML 字符串
 */
export function mergeFormToYaml(
  originalYaml: string,
  formObject: Record<string, any>,
  preserveFields: string[] = [
    'metadata.uid',
    'metadata.resourceVersion',
    'metadata.generation',
    'metadata.creationTimestamp',
    'metadata.managedFields',
    'status'
  ]
): string {
  const original = parseYaml<Record<string, any>>(originalYaml)
  if (!original) {
    return stringifyYaml(formObject)
  }

  // 首先深度合并
  const merged = deepMerge(original, formObject)

  // 确保保留字段不被覆盖
  for (const fieldPath of preserveFields) {
    const originalValue = getNestedValue(original, fieldPath)
    if (originalValue !== undefined) {
      setNestedValue(merged, fieldPath, originalValue)
    }
  }

  return stringifyYaml(merged)
}

/**
 * 从原始 YAML 提取表单需要的字段
 * @param yamlStr YAML 字符串
 * @param extractFields 需要提取的字段映射
 * @returns 提取的字段值对象
 */
export function extractFieldsFromYaml(
  yamlStr: string,
  extractFields: Record<string, string>
): Record<string, any> {
  const obj = parseYaml<Record<string, any>>(yamlStr)
  if (!obj) {
    return {}
  }

  const result: Record<string, any> = {}
  for (const [formKey, yamlPath] of Object.entries(extractFields)) {
    result[formKey] = getNestedValue(obj, yamlPath)
  }

  return result
}

/**
 * 获取嵌套对象的值
 * @param obj 目标对象
 * @param path 属性路径（如 'metadata.labels'）
 * @returns 属性值
 */
export function getNestedValue(obj: any, path: string): any {
  const keys = path.split('.')
  let current = obj

  for (const key of keys) {
    if (current === null || current === undefined) {
      return undefined
    }
    current = current[key]
  }

  return current
}

/**
 * 设置嵌套对象的值
 * @param obj 目标对象
 * @param path 属性路径（如 'metadata.labels'）
 * @param value 要设置的值
 */
export function setNestedValue(obj: any, path: string, value: any): void {
  const keys = path.split('.')
  let current = obj

  for (let i = 0; i < keys.length - 1; i++) {
    const key = keys[i]
    if (current[key] === undefined || current[key] === null) {
      current[key] = {}
    }
    current = current[key]
  }

  current[keys[keys.length - 1]] = value
}

/**
 * 删除嵌套对象的值
 * @param obj 目标对象
 * @param path 属性路径（如 'metadata.labels'）
 */
export function deleteNestedValue(obj: any, path: string): void {
  const keys = path.split('.')
  let current = obj

  for (let i = 0; i < keys.length - 1; i++) {
    const key = keys[i]
    if (current[key] === undefined || current[key] === null) {
      return
    }
    current = current[key]
  }

  delete current[keys[keys.length - 1]]
}

/**
 * 清理对象中的空值（空对象、空数组）
 * @param obj 要清理的对象
 * @returns 清理后的对象
 */
export function cleanEmptyValues(obj: any): any {
  if (Array.isArray(obj)) {
    const cleaned = obj
      .map((item) => cleanEmptyValues(item))
      .filter(
        (item) =>
          item !== null &&
          item !== undefined &&
          (typeof item !== 'object' || Object.keys(item).length > 0)
      )
    return cleaned.length > 0 ? cleaned : undefined
  }

  if (typeof obj === 'object' && obj !== null) {
    const cleaned: Record<string, any> = {}
    for (const [key, value] of Object.entries(obj)) {
      const cleanedValue = cleanEmptyValues(value)
      if (
        cleanedValue !== null &&
        cleanedValue !== undefined &&
        (typeof cleanedValue !== 'object' ||
          (Array.isArray(cleanedValue) && cleanedValue.length > 0) ||
          Object.keys(cleanedValue).length > 0)
      ) {
        cleaned[key] = cleanedValue
      }
    }
    return Object.keys(cleaned).length > 0 ? cleaned : undefined
  }

  return obj
}

/**
 * 对象转键值对数组
 * @param obj 对象
 * @returns 键值对数组
 */
export function objectToKeyValueArray(
  obj?: Record<string, string>
): Array<{ key: string; value: string }> {
  if (!obj) return []
  return Object.entries(obj)
    .filter(([key]) => key && key.trim())
    .map(([key, value]) => ({ key, value }))
}

/**
 * 键值对数组转对象
 * @param arr 键值对数组
 * @returns 对象
 */
export function keyValueArrayToObject(
  arr: Array<{ key: string; value: string }>
): Record<string, string> {
  const obj: Record<string, string> = {}
  arr.forEach((item) => {
    if (item.key && item.key.trim()) {
      obj[item.key.trim()] = item.value ?? ''
    }
  })
  return obj
}

/**
 * 保留未知字段合并 YAML
 * 用于编辑模式下保留原始 YAML 中表单未显示的字段
 * @param originalYaml 原始 YAML 字符串
 * @param formObject 从表单构建的对象
 * @returns 合并后的 YAML 字符串
 */
export function mergeYamlPreserveUnknown(
  originalYaml: string,
  formObject: Record<string, any>
): string {
  return mergeFormToYaml(originalYaml, formObject)
}
