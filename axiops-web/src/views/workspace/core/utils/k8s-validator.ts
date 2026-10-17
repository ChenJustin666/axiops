/**
 * Kubernetes 资源名称校验工具
 * 提供 DNS-1123 标准的名称校验功能
 */

/**
 * DNS-1123 标签（label）格式正则
 * 规则：小写字母、数字、中划线，首尾必须是字母或数字，最大 63 个字符
 */
export const DNS_1123_LABEL_PATTERN = /^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$/

/**
 * DNS-1123 子域名（subdomain）格式正则
 * 规则：可包含点号分隔的多个标签，最大 253 个字符
 */
export const DNS_1123_SUBDOMAIN_PATTERN = /^[a-z0-9]([a-z0-9.-]{0,251}[a-z0-9])?$/

/**
 * Kubernetes 标签键名格式正则
 * 规则：可选前缀（DNS子域名/）+ 名称部分
 */
export const K8S_LABEL_KEY_PATTERN =
  /^([a-z0-9]([a-z0-9.-]{0,251}[a-z0-9])?\/)?[a-z0-9A-Z]([a-z0-9A-Z._-]{0,61}[a-z0-9A-Z])?$/

/**
 * 校验资源名称是否符合 DNS-1123 标签格式
 * @param name 资源名称
 * @returns 校验结果对象
 */
export function validateDns1123Label(name: string): {
  valid: boolean
  message: string
} {
  if (!name) {
    return { valid: false, message: '名称不能为空' }
  }

  if (name.length > 63) {
    return { valid: false, message: '名称长度不能超过 63 个字符' }
  }

  if (!DNS_1123_LABEL_PATTERN.test(name)) {
    return {
      valid: false,
      message: '名称必须符合 DNS-1123 标准：小写字母、数字、中划线，首尾必须是字母或数字'
    }
  }

  return { valid: true, message: '' }
}

/**
 * 校验资源名称是否符合 DNS-1123 子域名格式
 * @param name 资源名称
 * @returns 校验结果对象
 */
export function validateDns1123Subdomain(name: string): {
  valid: boolean
  message: string
} {
  if (!name) {
    return { valid: false, message: '名称不能为空' }
  }

  if (name.length > 253) {
    return { valid: false, message: '名称长度不能超过 253 个字符' }
  }

  if (!DNS_1123_SUBDOMAIN_PATTERN.test(name)) {
    return {
      valid: false,
      message:
        '名称必须符合 DNS-1123 子域名标准：小写字母、数字、中划线、点号，首尾必须是字母或数字'
    }
  }

  return { valid: true, message: '' }
}

/**
 * 创建 DNS-1123 标签校验规则（用于 Element Plus 表单）
 * @param required 是否必填
 * @param fieldLabel 字段标签名（用于错误提示）
 * @returns 校验规则数组
 */
export function createDns1123LabelRules(required = true, fieldLabel = '名称'): any[] {
  const rules: any[] = []

  if (required) {
    rules.push({
      required: true,
      message: `请输入${fieldLabel}`,
      trigger: 'blur'
    })
  }

  rules.push({
    validator: (_rule: any, value: string, callback: (error?: Error) => void) => {
      if (!value && !required) {
        callback()
        return
      }

      const result = validateDns1123Label(value)
      if (!result.valid) {
        callback(new Error(result.message))
      } else {
        callback()
      }
    },
    trigger: 'blur'
  })

  return rules
}

/**
 * 创建 DNS-1123 子域名校验规则（用于 Element Plus 表单）
 * @param required 是否必填
 * @param fieldLabel 字段标签名（用于错误提示）
 * @returns 校验规则数组
 */
export function createDns1123SubdomainRules(required = true, fieldLabel = '名称'): any[] {
  const rules: any[] = []

  if (required) {
    rules.push({
      required: true,
      message: `请输入${fieldLabel}`,
      trigger: 'blur'
    })
  }

  rules.push({
    validator: (_rule: any, value: string, callback: (error?: Error) => void) => {
      if (!value && !required) {
        callback()
        return
      }

      const result = validateDns1123Subdomain(value)
      if (!result.valid) {
        callback(new Error(result.message))
      } else {
        callback()
      }
    },
    trigger: 'blur'
  })

  return rules
}

/**
 * 校验 Kubernetes 标签键名格式
 * @param key 标签键名
 * @returns 校验结果对象
 */
export function validateLabelKey(key: string): {
  valid: boolean
  message: string
} {
  if (!key) {
    return { valid: false, message: '标签键不能为空' }
  }

  if (key.length > 317) {
    // 253 (prefix) + 1 (/) + 63 (name)
    return { valid: false, message: '标签键长度不能超过 317 个字符' }
  }

  if (!K8S_LABEL_KEY_PATTERN.test(key)) {
    return {
      valid: false,
      message: '标签键格式不正确，支持格式：[prefix/]name'
    }
  }

  return { valid: true, message: '' }
}

/**
 * 校验 Kubernetes 标签值格式
 * @param value 标签值
 * @returns 校验结果对象
 */
export function validateLabelValue(value: string): {
  valid: boolean
  message: string
} {
  // 标签值可以为空
  if (!value) {
    return { valid: true, message: '' }
  }

  if (value.length > 63) {
    return { valid: false, message: '标签值长度不能超过 63 个字符' }
  }

  // 标签值格式：字母数字开头结尾，中间可包含 - _ .
  const labelValuePattern = /^[a-zA-Z0-9]([a-zA-Z0-9._-]{0,61}[a-zA-Z0-9])?$/
  if (!labelValuePattern.test(value)) {
    return {
      valid: false,
      message: '标签值格式不正确，必须以字母或数字开头和结尾'
    }
  }

  return { valid: true, message: '' }
}
