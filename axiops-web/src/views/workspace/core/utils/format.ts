/**
 * 格式化 Kubernetes 风格的时长字符串
 * 将 "4903h26m15.28875s" 格式转换为可读格式
 * 显示规则：
 * - 超过24小时：显示天数+小时（如 30d5h）
 * - 小于24小时：显示小时+分钟（如 5h30m）
 * - 小于1小时：显示分钟+秒（如 30m15s）
 * - 小于1分钟：显示秒（如 30s）
 *
 * @param age - Kubernetes 风格的时长字符串
 * @returns 格式化后的时长字符串
 *
 * @example
 * formatK8sAge("4903h26m15s") // "204d7h"
 * formatK8sAge("72h30m15s") // "3d"
 * formatK8sAge("26m15s") // "26m"
 * formatK8sAge("15s") // "15s"
 */
export function formatK8sAge(age: string | undefined | null): string {
  if (!age) return '-'

  // 移除小数部分，只保留整数
  const cleanAge = age.replace(/\.\d+s$/, 's')

  // 解析各时间单位
  const dayMatch = cleanAge.match(/(\d+)d/)
  const hourMatch = cleanAge.match(/(\d+)h/)
  const minuteMatch = cleanAge.match(/(\d+)m/)
  const secondMatch = cleanAge.match(/(\d+)s/)

  const days = dayMatch ? parseInt(dayMatch[1]) : 0
  const hours = hourMatch ? parseInt(hourMatch[1]) : 0
  const minutes = minuteMatch ? parseInt(minuteMatch[1]) : 0
  const seconds = secondMatch ? parseInt(secondMatch[1]) : 0

  const parts: string[] = []

  // 计算总小时数，转换为天数
  const totalHours = days * 24 + hours
  const displayDays = Math.floor(totalHours / 24)
  const displayHours = totalHours % 24

  // 根据时长大小决定显示精度
  if (displayDays > 0) {
    // 有天数，显示天数和小时
    parts.push(`${displayDays}d`)
    if (displayHours > 0) {
      parts.push(`${displayHours}h`)
    }
  } else if (displayHours > 0) {
    // 没有天数但有小数，显示小时和分钟
    parts.push(`${displayHours}h`)
    if (minutes > 0) {
      parts.push(`${minutes}m`)
    }
  } else if (minutes > 0) {
    // 没有小时但有分钟，显示分钟和秒
    parts.push(`${minutes}m`)
    if (seconds > 0) {
      parts.push(`${seconds}s`)
    }
  } else if (seconds > 0) {
    // 只有秒
    parts.push(`${seconds}s`)
  }

  // 如果没有任何部分，返回 '-'
  if (parts.length === 0) {
    return '-'
  }

  return parts.join('')
}

/**
 * 格式化相对时间（基于创建时间戳）
 * 显示为 Kubernetes 风格的时长
 *
 * @param creationTimestamp - 创建时间戳（毫秒或秒）
 * @returns 格式化后的时长字符串
 */
export function formatRelativeAge(creationTimestamp: number | undefined | null): string {
  if (!creationTimestamp) return '-'

  // 判断是秒还是毫秒
  const timestamp = creationTimestamp > 9999999999 ? creationTimestamp : creationTimestamp * 1000
  const now = Date.now()
  const diff = now - timestamp

  const seconds = Math.floor(diff / 1000)
  const minutes = Math.floor(seconds / 60)
  const hours = Math.floor(minutes / 60)
  const days = Math.floor(hours / 24)

  const parts: string[] = []

  if (days > 0) {
    parts.push(`${days}d`)
    const remainingHours = hours % 24
    if (remainingHours > 0) {
      parts.push(`${remainingHours}h`)
    }
  } else if (hours > 0) {
    parts.push(`${hours}h`)
    const remainingMinutes = minutes % 60
    if (remainingMinutes > 0) {
      parts.push(`${remainingMinutes}m`)
    }
  } else if (minutes > 0) {
    parts.push(`${minutes}m`)
    const remainingSeconds = seconds % 60
    if (remainingSeconds > 0) {
      parts.push(`${remainingSeconds}s`)
    }
  } else {
    parts.push(`${seconds}s`)
  }

  return parts.join('')
}
