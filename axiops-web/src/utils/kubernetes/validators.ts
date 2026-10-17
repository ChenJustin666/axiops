/**
 * Kubernetes 资源命名验证工具
 * 符合 Kubernetes 命名规范
 */

/** 验证结果 */
export interface ValidationResult {
  valid: boolean
  message?: string
}

/** DNS-1123 标签正则表达式 */
const DNS_1123_LABEL_REGEX = /^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/

/** DNS-1123 子域名正则表达式 */
const DNS_1123_SUBDOMAIN_REGEX = /^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$/

/** DNS-1035 标签正则表达式（必须字母开头） */
const DNS_1035_LABEL_REGEX = /^[a-z]([-a-z0-9]*[a-z0-9])?$/

/** 标签键名正则表达式（含可选前缀） */
const LABEL_KEY_REGEX =
  /^([a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*\/)?[a-zA-Z0-9]([-a-zA-Z0-9_.]*[a-zA-Z0-9])?$/

/** 标签值正则表达式 */
const LABEL_VALUE_REGEX = /^([a-zA-Z0-9]([-a-zA-Z0-9_.]*[a-zA-Z0-9])?)?$/

/** 长度限制 */
const DNS_1123_LABEL_MAX_LENGTH = 63
const DNS_1123_SUBDOMAIN_MAX_LENGTH = 253
const LABEL_KEY_NAME_MAX_LENGTH = 63
const LABEL_VALUE_MAX_LENGTH = 63

/**
 * 验证 DNS-1123 标签格式
 * 用于 Pod、Namespace 等资源名称（最大 63 字符）
 * @param name - 待验证的名称
 * @returns 验证结果
 */
export function validateDNS1123Label(name: string): ValidationResult {
  if (!name) {
    return { valid: false, message: '名称不能为空' }
  }

  if (name.length > DNS_1123_LABEL_MAX_LENGTH) {
    return {
      valid: false,
      message: `名称长度不能超过 ${DNS_1123_LABEL_MAX_LENGTH} 个字符`
    }
  }

  if (!DNS_1123_LABEL_REGEX.test(name)) {
    return {
      valid: false,
      message: '名称只能包含小写字母、数字和连字符，必须以字母或数字开头和结尾'
    }
  }

  return { valid: true }
}

/**
 * 验证 DNS-1123 子域名格式
 * 用于 StorageClass、ClusterRole 等资源名称（最大 253 字符）
 * @param name - 待验证的名称
 * @returns 验证结果
 */
export function validateDNS1123Subdomain(name: string): ValidationResult {
  if (!name) {
    return { valid: false, message: '名称不能为空' }
  }

  if (name.length > DNS_1123_SUBDOMAIN_MAX_LENGTH) {
    return {
      valid: false,
      message: `名称长度不能超过 ${DNS_1123_SUBDOMAIN_MAX_LENGTH} 个字符`
    }
  }

  if (!DNS_1123_SUBDOMAIN_REGEX.test(name)) {
    return {
      valid: false,
      message: '名称只能包含小写字母、数字、连字符和点，必须以字母或数字开头和结尾'
    }
  }

  return { valid: true }
}

/**
 * 验证 DNS-1035 标签格式
 * 用于 Service 等资源名称（必须字母开头，最大 63 字符）
 * @param name - 待验证的名称
 * @returns 验证结果
 */
export function validateDNS1035Label(name: string): ValidationResult {
  if (!name) {
    return { valid: false, message: '名称不能为空' }
  }

  if (name.length > DNS_1123_LABEL_MAX_LENGTH) {
    return {
      valid: false,
      message: `名称长度不能超过 ${DNS_1123_LABEL_MAX_LENGTH} 个字符`
    }
  }

  if (!DNS_1035_LABEL_REGEX.test(name)) {
    return {
      valid: false,
      message: '名称只能包含小写字母、数字和连字符，必须以字母开头，以字母或数字结尾'
    }
  }

  return { valid: true }
}

/**
 * 验证标签键名格式
 * 支持可选的前缀（prefix/name 格式）
 * @param key - 待验证的标签键
 * @returns 验证结果
 */
export function validateLabelKey(key: string): ValidationResult {
  if (!key) {
    return { valid: false, message: '标签键不能为空' }
  }

  // 检查是否包含前缀
  const parts = key.split('/')

  if (parts.length > 2) {
    return { valid: false, message: '标签键格式不正确，最多只能包含一个斜杠' }
  }

  if (parts.length === 2) {
    const [prefix, name] = parts

    // 验证前缀（DNS 子域名格式）
    if (prefix.length > DNS_1123_SUBDOMAIN_MAX_LENGTH) {
      return {
        valid: false,
        message: `标签键前缀长度不能超过 ${DNS_1123_SUBDOMAIN_MAX_LENGTH} 个字符`
      }
    }

    if (!DNS_1123_SUBDOMAIN_REGEX.test(prefix)) {
      return { valid: false, message: '标签键前缀格式不正确' }
    }

    // 验证名称部分
    if (name.length > LABEL_KEY_NAME_MAX_LENGTH) {
      return {
        valid: false,
        message: `标签键名称部分长度不能超过 ${LABEL_KEY_NAME_MAX_LENGTH} 个字符`
      }
    }

    if (!/^[a-zA-Z0-9]([-a-zA-Z0-9_.]*[a-zA-Z0-9])?$/.test(name)) {
      return { valid: false, message: '标签键名称部分格式不正确' }
    }
  } else {
    // 无前缀的键名
    if (key.length > LABEL_KEY_NAME_MAX_LENGTH) {
      return {
        valid: false,
        message: `标签键长度不能超过 ${LABEL_KEY_NAME_MAX_LENGTH} 个字符`
      }
    }

    if (!/^[a-zA-Z0-9]([-a-zA-Z0-9_.]*[a-zA-Z0-9])?$/.test(key)) {
      return {
        valid: false,
        message: '标签键只能包含字母、数字、连字符、下划线和点，必须以字母或数字开头和结尾'
      }
    }
  }

  return { valid: true }
}

/**
 * 验证标签值格式
 * @param value - 待验证的标签值
 * @returns 验证结果
 */
export function validateLabelValue(value: string): ValidationResult {
  // 空值是允许的
  if (!value || value === '') {
    return { valid: true }
  }

  if (value.length > LABEL_VALUE_MAX_LENGTH) {
    return {
      valid: false,
      message: `标签值长度不能超过 ${LABEL_VALUE_MAX_LENGTH} 个字符`
    }
  }

  if (!LABEL_VALUE_REGEX.test(value)) {
    return {
      valid: false,
      message: '标签值只能包含字母、数字、连字符、下划线和点，必须以字母或数字开头和结尾'
    }
  }

  return { valid: true }
}

/**
 * 验证注解键名格式
 * 注解键名规则与标签键名相同
 * @param key - 待验证的注解键
 * @returns 验证结果
 */
export function validateAnnotationKey(key: string): ValidationResult {
  return validateLabelKey(key)
}

/**
 * 验证资源名称（通用）
 * 根据资源类型选择合适的验证规则
 * @param name - 待验证的名称
 * @param resourceType - 资源类型
 * @returns 验证结果
 */
export function validateResourceName(
  name: string,
  resourceType:
    | 'pod'
    | 'namespace'
    | 'service'
    | 'storageclass'
    | 'clusterrole'
    | 'ingressclass'
    | 'default' = 'default'
): ValidationResult {
  switch (resourceType) {
    case 'service':
      return validateDNS1035Label(name)
    case 'pod':
    case 'namespace':
      return validateDNS1123Label(name)
    case 'storageclass':
    case 'clusterrole':
    case 'ingressclass':
    default:
      return validateDNS1123Subdomain(name)
  }
}

/**
 * 快速验证名称是否有效（不返回详细信息）
 * @param name - 待验证的名称
 * @param pattern - 验证类型
 * @returns 是否有效
 */
export function isValidName(
  name: string,
  pattern: 'dns1123label' | 'dns1123subdomain' | 'dns1035label' = 'dns1123subdomain'
): boolean {
  switch (pattern) {
    case 'dns1123label':
      return validateDNS1123Label(name).valid
    case 'dns1035label':
      return validateDNS1035Label(name).valid
    case 'dns1123subdomain':
    default:
      return validateDNS1123Subdomain(name).valid
  }
}
