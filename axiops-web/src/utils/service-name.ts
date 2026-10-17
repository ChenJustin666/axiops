export const SERVICE_NAME_HINT =
  '服务名称须为 1–128 位小写字母、数字及合法的 .、_、- 分隔符，以字母或数字开头和结尾，例如 risk_server'

export function serviceNameError(value: string): string {
  const name = value.trim()
  return name.length <= 128 && /^[a-z0-9]+(?:(?:[._]|__|-+)[a-z0-9]+)*$/.test(name)
    ? ''
    : SERVICE_NAME_HINT
}

export function serviceCodeFromName(value: string): string {
  let code = value
    .toLowerCase()
    .replace(/[^a-z0-9-]+/g, '-')
    .replace(/^-+|-+$/g, '')
  if (!/^[a-z]/.test(code)) code = `svc-${code}`
  return code.slice(0, 64).replace(/-+$/, '')
}
