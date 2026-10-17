export function formatExecutionDuration(start?: string, finish?: string, now = Date.now()): string {
  if (!start) return '-'
  const started = new Date(start).getTime()
  const ended = finish ? new Date(finish).getTime() : now
  if (!Number.isFinite(started) || !Number.isFinite(ended)) return '-'
  const seconds = Math.max(0, Math.floor((ended - started) / 1000))
  if (seconds < 60) return `${seconds} 秒`
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes} 分 ${seconds % 60} 秒`
  return `${Math.floor(minutes / 60)} 小时 ${minutes % 60} 分 ${seconds % 60} 秒`
}
