// Stored Dockerfile paths remain relative to build_context for compatibility
// with the current BuildKit executor. The picker shows repository paths.
export function repositoryPath(value: string): string | null {
  const path = value.trim()
  if (/^[\\/]|^[a-z]:|\\/i.test(path) || Array.from(path).some((char) => char.charCodeAt(0) < 32))
    return null
  const parts = path.split('/').filter((part) => part && part !== '.')
  if (parts.includes('..')) return null
  return parts.join('/') || '.'
}

export function buildSourceError(context: string, dockerfile: string): string {
  if (repositoryPath(context) === null)
    return '构建上下文必须是仓库内的相对目录，不能包含 .. 或绝对路径'
  const file = repositoryPath(dockerfile)
  if (file === null || file === '.') return '请选择 Dockerfile，或填写构建上下文内的相对文件路径'
  return ''
}

export function dockerfileInRepository(context: string, dockerfile: string): string {
  if (buildSourceError(context, dockerfile)) return ''
  return [repositoryPath(context), repositoryPath(dockerfile)]
    .filter((part) => part !== '.')
    .join('/')
}

export function parentDirectory(path: string): string {
  const slash = path.lastIndexOf('/')
  return slash < 0 ? '.' : path.slice(0, slash)
}

export function mavenModuleCommand(module: string): string {
  const path = repositoryPath(module)
  if (!path || path === '.' || path.startsWith('-')) return ''
  const quoted = `'${path.replace(/'/g, "'\\''")}'`
  return `mvn -B -ntp -Dmaven.repo.local="\${CACHE_DIR:-/workspace/.cache/maven}" -pl ${quoted} -am clean package`
}
