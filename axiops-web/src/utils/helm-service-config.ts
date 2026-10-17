import yaml from 'js-yaml'
import type { HelmEnvironmentSettings, ServiceEnvironmentConfig } from '@/api/axiops'

export type ChartFile = { path: string; content: string }

export function chartDifferences(before: ChartFile[], after: ChartFile[]) {
  const left = new Map(before.map((f) => [f.path, f.content]))
  const right = new Map(after.map((f) => [f.path, f.content]))
  return [...new Set([...left.keys(), ...right.keys()])].sort().map((path) => ({
    path,
    before: left.get(path),
    after: right.get(path),
    change: !left.has(path)
      ? '新增'
      : !right.has(path)
        ? '删除'
        : left.get(path) === right.get(path)
          ? '未变'
          : '修改'
  }))
}

export function chartFileTree(files: ChartFile[]) {
  type Node = { path: string; label: string; file?: boolean; children?: Node[] }
  const roots: Node[] = []
  for (const file of [...files].sort((a, b) => a.path.localeCompare(b.path))) {
    let nodes = roots
    const parts = file.path.split('/')
    parts.forEach((part, index) => {
      const path = parts.slice(0, index + 1).join('/')
      let node = nodes.find((n) => n.path === path)
      if (!node) {
        node = {
          path,
          label: part,
          ...(index === parts.length - 1 ? { file: true } : { children: [] })
        }
        nodes.push(node)
      }
      nodes = node.children || []
    })
  }
  return roots
}

export function starterChart(name = 'application'): ChartFile[] {
  return [
    {
      path: 'Chart.yaml',
      content: `apiVersion: v2\nname: ${name || 'application'}\nversion: 0.1.0\ntype: application\n`
    },
    {
      path: 'values.yaml',
      content:
        'replicaCount: 1\nimage:\n  repository: example.invalid/team/service\n  tag: latest\nservice:\n  port: 8080\nresources:\n  requests:\n    cpu: 100m\n    memory: 128Mi\n  limits:\n    memory: 512Mi\nenv: []\nenvFrom: []\ncommand: []\nargs: []\n'
    },
    {
      path: 'templates/deployment.yaml',
      content: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Release.Name }}
spec:
  replicas: {{ .Values.replicaCount }}
  selector:
    matchLabels:
      app: {{ .Release.Name }}
  template:
    metadata:
      labels:
        app: {{ .Release.Name }}
    spec:
      containers:
        - name: application
          image: "{{ .Values.image.repository }}:{{ .Values.image.tag }}"
          ports:
            - name: http
              containerPort: {{ .Values.service.port }}
          resources:
            {{- toYaml .Values.resources | nindent 12 }}
          {{- with .Values.env }}
          env:
            {{- toYaml . | nindent 12 }}
          {{- end }}
          {{- with .Values.envFrom }}
          envFrom:
            {{- toYaml . | nindent 12 }}
          {{- end }}
          {{- with .Values.command }}
          command:
            {{- toYaml . | nindent 12 }}
          {{- end }}
          {{- with .Values.args }}
          args:
            {{- toYaml . | nindent 12 }}
          {{- end }}
`
    },
    {
      path: 'templates/service.yaml',
      content: `apiVersion: v1
kind: Service
metadata:
  name: {{ .Release.Name }}
spec:
  type: ClusterIP
  selector:
    app: {{ .Release.Name }}
  ports:
    - name: http
      port: {{ .Values.service.port }}
      targetPort: http
`
    }
  ]
}

export function chartFiles(definition: string): ChartFile[] {
  if (!definition.trim()) return []
  const parsed = JSON.parse(definition)
  if (parsed.schema_version !== 1 || !Array.isArray(parsed.files)) throw Error('Chart 文件结构无效')
  return parsed.files.map((file: ChartFile) => {
    if (typeof file.path !== 'string' || typeof file.content !== 'string')
      throw Error('Chart 文件无效')
    return { path: file.path, content: file.content }
  })
}

export function valuesObject(text: string): Record<string, unknown> {
  const value = yaml.load(text)
  if (value == null) return {}
  if (typeof value !== 'object' || Array.isArray(value)) throw Error('values 必须是 YAML 对象')
  try {
    JSON.stringify(value)
  } catch {
    throw Error('values 不支持循环 YAML 引用')
  }
  return value as Record<string, unknown>
}

export function environmentHelmSettings(config: ServiceEnvironmentConfig): HelmEnvironmentSettings {
  const defaults = { repository_path: 'image.repository', tag_path: 'image.tag', ref_path: '' }
  return {
    values: config.helm?.values || '{}\n',
    image: { ...(config.helm?.image || defaults) },
    ...(config.helm?.source ? { source: { ...config.helm.source } } : {}),
    ...(config.helm?.credential_keys ? { credential_keys: { ...config.helm.credential_keys } } : {})
  }
}

export function initialHelmValues(config: ServiceEnvironmentConfig): string {
  const text = environmentHelmSettings(config).values
  // Chart defaults remain inherited, instead of becoming fixed overrides.
  return text
}

export function syncHelmValues(source: string, target: string, keys: string[]): string {
  const from = valuesObject(source),
    to = valuesObject(target)
  for (const key of keys) {
    const parts = key.startsWith('/') ? key.slice(1).split('/').map(unescapePath) : [key]
    if (parts.some(unsafeKey)) throw Error('不支持的 values 路径')
    let value: unknown = from
    for (const part of parts) {
      if (!isObject(value) || !Object.prototype.hasOwnProperty.call(value, part))
        throw Error('同步字段已变化，请重新选择')
      value = value[part]
    }
    let cursor = to
    for (const part of parts.slice(0, -1)) {
      if (!isObject(cursor[part])) cursor[part] = {}
      cursor = cursor[part] as Record<string, unknown>
    }
    cursor[parts[parts.length - 1]] = value
  }
  return yaml.dump(to, { noRefs: true, lineWidth: -1 })
}

const isObject = (v: unknown): v is Record<string, unknown> =>
  !!v && typeof v === 'object' && !Array.isArray(v)
const unsafeKey = (key: string) => ['__proto__', 'prototype', 'constructor'].includes(key)
const escapePath = (key: string) => key.replace(/~/g, '~0').replace(/\//g, '~1')
const unescapePath = (key: string) => key.replace(/~1/g, '/').replace(/~0/g, '~')
export const valuesPathLabel = (path: string) =>
  path
    ? path
        .slice(1)
        .split('/')
        .map(unescapePath)
        .map((key, index) =>
          /^[A-Za-z_][A-Za-z0-9_-]*$/.test(key)
            ? (index ? '.' : '') + key
            : `[${JSON.stringify(key)}]`
        )
        .join('')
    : ''

export function helmValuesPaths(raw: string, protectedPaths: string[] = []): string[] {
  const result: string[] = []
  const protectedPointers = protectedPaths
    .filter(Boolean)
    .map((p) => '/' + p.split('.').map(escapePath).join('/'))
  function walk(value: unknown, path: string) {
    if (protectedPointers.some((p) => path === p || path.startsWith(p + '/'))) return
    if (isObject(value) && Object.keys(value).length) {
      for (const key of Object.keys(value)) {
        if (!unsafeKey(key)) walk(value[key], path + '/' + escapePath(key))
      }
    } else if (path && !protectedPointers.some((p) => p.startsWith(path + '/'))) result.push(path)
  }
  walk(valuesObject(raw), '')
  return result
}

export type ValuesConflict = { path: string; base: string; local: string; remote: string }
// Missing is distinct from YAML null. Arrays follow Helm's whole-array semantics.
export function mergeHelmValuesSource(
  baseRaw: string,
  localRaw: string,
  remoteRaw: string,
  choices: Record<string, 'local' | 'remote'> = {}
) {
  const missing = Symbol('missing'),
    conflicts: ValuesConflict[] = []
  const stable = (v: any): string =>
    v === missing
      ? '<未设置>'
      : isObject(v)
        ? JSON.stringify(
            Object.keys(v)
              .sort()
              .map((k) => [k, stable(v[k])])
          )
        : JSON.stringify(v)
  const display = (v: any) =>
    v === missing ? '（删除覆盖，恢复继承）' : yaml.dump(v, { noRefs: true, lineWidth: -1 })
  function merge(base: any, local: any, remote: any, path: string): any {
    if (stable(local) === stable(remote) || stable(remote) === stable(base)) return local
    if (stable(local) === stable(base)) return remote
    if (isObject(local) && isObject(remote) && (base === missing || isObject(base))) {
      const result: Record<string, unknown> = {}
      for (const key of new Set([
        ...Object.keys(base === missing ? {} : base),
        ...Object.keys(local),
        ...Object.keys(remote)
      ])) {
        if (unsafeKey(key)) throw Error('不支持的 values 键')
        const own = (obj: any) =>
          obj !== missing && Object.prototype.hasOwnProperty.call(obj, key) ? obj[key] : missing
        const value = merge(own(base), own(local), own(remote), path + '/' + escapePath(key))
        if (value !== missing) result[key] = value
      }
      return result
    }
    conflicts.push({ path, base: display(base), local: display(local), remote: display(remote) })
    return choices[path] === 'remote' ? remote : local
  }
  const values = merge(valuesObject(baseRaw), valuesObject(localRaw), valuesObject(remoteRaw), '')
  return { values: yaml.dump(values, { noRefs: true, lineWidth: -1 }), conflicts }
}

export function validateHelmSettings(settings: HelmEnvironmentSettings): void {
  valuesObject(settings.values)
  const { ref_path: ref, repository_path: repo, tag_path: tag } = settings.image
  const path = /^[A-Za-z0-9_-]+(?:\.[A-Za-z0-9_-]+)*$/
  if (ref) {
    if (!path.test(ref) || repo || tag) throw Error('完整镜像与 repository/tag 映射只能选择一种')
  } else if (
    !path.test(repo) ||
    !path.test(tag) ||
    repo === tag ||
    repo.startsWith(tag + '.') ||
    tag.startsWith(repo + '.')
  ) {
    throw Error('请填写不同的镜像 repository/tag values 路径')
  }
}
